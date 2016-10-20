package dnssd

import (
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/miekg/dns"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

type netserver struct {
	ipv4pconn *ipv4.PacketConn
	ipv6pconn *ipv6.PacketConn

	response, query *dns.Msg

	closed    bool
	msgCh     chan *incomingMsg
	closeLock sync.Mutex
}

type incomingMsg struct {
	msg     *dns.Msg
	ifIndex int
	from    net.Addr
}

var (
	// Multicast groups used by mDNS
	mdnsGroupIPv4 = net.IPv4(224, 0, 0, 251)
	mdnsGroupIPv6 = net.ParseIP("ff02::fb")

	// mDNS wildcard addresses
	mdnsWildcardAddrIPv4 = &net.UDPAddr{
		IP:   net.ParseIP("224.0.0.0"),
		Port: 5353,
	}
	mdnsWildcardAddrIPv6 = &net.UDPAddr{
		IP:   net.ParseIP("ff02::"),
		Port: 5353,
	}

	// mDNS endpoint addresses
	ipv4Addr = &net.UDPAddr{
		IP:   mdnsGroupIPv4,
		Port: 5353,
	}
	ipv6Addr = &net.UDPAddr{
		IP:   mdnsGroupIPv6,
		Port: 5353,
	}
)

func (im *incomingMsg) String() string {
	return fmt.Sprintf("IM{%d,%s,%s}", im.ifIndex, im.from, im.msg)
}

func makeNetserver() (*netserver, error) {
	var iface *net.Interface = nil

	// Create wildcard connections (because :5353 can be already taken by other apps)
	ipv4conn, err := net.ListenUDP("udp4", mdnsWildcardAddrIPv4)
	if err != nil {
		netlog("[ERR] dnssd: Failed to bind to udp4 port: ", err)
	}
	ipv6conn, err := net.ListenUDP("udp6", mdnsWildcardAddrIPv6)
	if err != nil {
		netlog("[ERR] dnssd: Failed to bind to udp6 port: ", err)
	}
	if ipv4conn == nil && ipv6conn == nil {
		return nil, fmt.Errorf("[ERR] dnssd: Failed to bind to any udp port!")
	}

	// Join multicast groups to receive announcements
	p1 := ipv4.NewPacketConn(ipv4conn)
	p2 := ipv6.NewPacketConn(ipv6conn)
	p1.SetControlMessage(ipv4.FlagInterface, true)
	p2.SetControlMessage(ipv6.FlagInterface, true)
	if iface != nil {
		if err := p1.JoinGroup(iface, &net.UDPAddr{IP: mdnsGroupIPv4}); err != nil {
			return nil, err
		}
		if err := p2.JoinGroup(iface, &net.UDPAddr{IP: mdnsGroupIPv6}); err != nil {
			return nil, err
		}
	} else {
		ifaces, err := net.Interfaces()
		if err != nil {
			return nil, err
		}
		errCount1, errCount2 := 0, 0
		for _, iface := range ifaces {
			if err := p1.JoinGroup(&iface, &net.UDPAddr{IP: mdnsGroupIPv4}); err != nil {
				errCount1++
			}
			if err := p2.JoinGroup(&iface, &net.UDPAddr{IP: mdnsGroupIPv6}); err != nil {
				errCount2++
			}
		}
		if len(ifaces) == errCount1 && len(ifaces) == errCount2 {
			return nil, fmt.Errorf("Failed to join multicast group on all interfaces!")
		}
	}

	msgCh := make(chan *incomingMsg, 32)
	ns := &netserver{ipv4pconn: p1, ipv6pconn: p2, msgCh: msgCh,
		response: &dns.Msg{}, query: &dns.Msg{}}
	ns.startReceiving()
	return ns, nil
}

func (nss *netserver) startReceiving() {
	if nss.ipv4pconn != nil {
		c := nss.ipv4pconn
		go nss._recv(func(buf []byte) (n, ifIndex int, from net.Addr, err error) {
			n, cm, from, err := c.ReadFrom(buf)
			return n, cm.IfIndex, from, err
		})
	}
	if nss.ipv6pconn != nil {
		c := nss.ipv6pconn
		go nss._recv(func(buf []byte) (n, ifIndex int, from net.Addr, err error) {
			n, cm, from, err := c.ReadFrom(buf)
			return n, cm.IfIndex, from, err
		})
	}
}

// recv is a long running routine to receive packets from an interface
func (nss *netserver) _recv(readSocket func(buf []byte) (n, ifIndex int, from net.Addr, err error)) {
	buf := make([]byte, 65536)
	for !nss.closed {
		n, ifIndex, from, err := readSocket(buf)
		if err != nil {
			netlog("[ERR] dnssd: Failed to read packet: ", err)
			continue
		}

		var msg dns.Msg
		if err := msg.Unpack(buf[:n]); err != nil {
			netlog("[ERR] dnssd: Failed to unpack packet: ", err)
			continue
		}
		nss.msgCh <- &incomingMsg{&msg, ifIndex, from}
	}
}

// Shutdown server will close currently open connections & channel
func (nss *netserver) shutdown() error {
	nss.closeLock.Lock()
	defer nss.closeLock.Unlock()

	if nss.closed {
		return nil
	}
	nss.closed = true

	if nss.ipv4pconn != nil {
		nss.ipv4pconn.Close()
	}
	if nss.ipv6pconn != nil {
		nss.ipv6pconn.Close()
	}
	return nil
}

// Should not be used according to RFC6762, but seems to be used in practice
func (nss *netserver) sendResponseQuestion(ifIndex int, q *dns.Question) {
	nss.response.Question = append(nss.response.Question, *q)
}

func (nss *netserver) sendResponseRecord(ifIndex int, rr dns.RR) {
	nss.response.Answer = append(nss.response.Answer, rr)
}

func (nss *netserver) sendKnownAnswer(ifIndex int, rr dns.RR) {
	nss.query.Answer = append(nss.query.Answer, rr)
}

func (nss *netserver) sendQuestion(ifIndex int, q *dns.Question) {
	nss.query.Question = append(nss.query.Question, *q)
}

func (nss *netserver) sendPending() error {
	nss.sendMessage(&nss.response)
	nss.sendMessage(&nss.query)
	return nil
}

// Pack the dns.Msg and write to available connections (multicast)
func (nss *netserver) sendMessage(msgp **dns.Msg) error {
	msg := *msgp
	if len(msg.Answer) == 0 && len(msg.Question) == 0 {
		return nil
	}
	newMsg := &dns.Msg{}
	*msgp = newMsg
	newMsg.Response = (msgp == &nss.response)

	netlog("TX:", msg)
	buf, err := msg.Pack()
	if err != nil {
		log.Println("Failed to pack message!", err)
		log.Println("Failed to pack message!", *msgp)
		return err
	}
	if nss.ipv4pconn != nil {
		nss.ipv4pconn.WriteTo(buf, nil, ipv4Addr)
	}
	if nss.ipv6pconn != nil {
		nss.ipv6pconn.WriteTo(buf, nil, ipv6Addr)
	}
	return nil
}
