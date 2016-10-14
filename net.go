package dnssd

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/miekg/dns"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

type netserver struct {
	ipv4conn *net.UDPConn
	ipv6conn *net.UDPConn

	closed    bool
	msgCh     chan *incomingMsg
	closeLock sync.Mutex
}

type incomingMsg struct {
	*dns.Msg
	from net.Addr
}

type netCommand int

const (
	CLOSE netCommand = iota
)

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
	return fmt.Sprintf("IM{%s,%s}", im.from, im.msg)
}

func makeNetserver(iface *net.Interface) (*netserver, error) {
	// Create wildcard connections (because :5353 can be already taken by other apps)
	ipv4conn, err := net.ListenUDP("udp4", mdnsWildcardAddrIPv4)
	if err != nil {
		log.Printf("[ERR] dnssd: Failed to bind to udp4 port: %v", err)
	}
	ipv6conn, err := net.ListenUDP("udp6", mdnsWildcardAddrIPv6)
	if err != nil {
		log.Printf("[ERR] dnssd: Failed to bind to udp6 port: %v", err)
	}
	if ipv4conn == nil && ipv6conn == nil {
		return nil, fmt.Errorf("[ERR] dnssd: Failed to bind to any udp port!")
	}

	// Join multicast groups to receive announcements
	p1 := ipv4.NewPacketConn(ipv4conn)
	p2 := ipv6.NewPacketConn(ipv6conn)
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
	return &netserver{ipv4conn: ipv4conn, ipv6conn: ipv6conn, msgCh: msgCh}, nil
}

func (nss *netserver) startReceiving() {
	if nss.ipv4conn != nil {
		go nss._recv(nss.ipv4conn)
	}
	if nss.ipv6conn != nil {
		go nss._recv(nss.ipv6conn)
	}
}

// multicastResponse us used to send a multicast response packet
func (nss *netserver) multicastResponse(msg *dns.Msg) error {
	buf, err := msg.Pack()
	if err != nil {
		log.Println("Failed to pack message!")
		return err
	}
	if nss.ipv4conn != nil {
		nss.ipv4conn.WriteTo(buf, ipv4Addr)
	}
	if nss.ipv6conn != nil {
		nss.ipv6conn.WriteTo(buf, ipv6Addr)
	}
	return nil
}

// recv is a long running routine to receive packets from an interface
func (nss *netserver) _recv(c *net.UDPConn) {
	if c == nil {
		return
	}
	buf := make([]byte, 65536)
	for !nss.closed {
		n, from, err := c.ReadFrom(buf)
		if err != nil {
			continue
		}

		if err := nss.parsePacket(buf[:n], from); err != nil {
			log.Printf("[ERR] dnssd: Failed to handle packet: %v", err)
		}
	}
}

// parsePacket is used to parse an incoming packet
func (nss *netserver) parsePacket(packet []byte, from net.Addr) error {
	var msg dns.Msg
	if err := msg.Unpack(packet); err != nil {
		log.Printf("[ERR] dnssd: Failed to unpack packet: %v", err)
		return err
	}
	nss.msgCh <- &incomingMsg{&msg, from}
	//	return s.handleQuery(&msg, from)
	return nil
}

// Shutdown server will close currently open connections & channel
func (nss *netserver) shutdown() error {
	nss.closeLock.Lock()
	defer nss.closeLock.Unlock()

	if nss.closed {
		return nil
	}
	nss.closed = true

	if nss.ipv4conn != nil {
		nss.ipv4conn.Close()
	}
	if nss.ipv6conn != nil {
		nss.ipv6conn.Close()
	}
	return nil
}

// Pack the dns.Msg and write to available connections (multicast)
func (nss *netserver) sendQuery(msg *dns.Msg) error {
	buf, err := msg.Pack()
	if err != nil {
		return err
	}
	if nss.ipv4conn != nil {
		nss.ipv4conn.WriteTo(buf, ipv4Addr)
	}
	if nss.ipv6conn != nil {
		nss.ipv6conn.WriteTo(buf, ipv6Addr)
	}
	return nil
}

func (nss *netserver) sendUnsolicitedMessage(resp *dns.Msg) {
	/*	resp := new(dns.Msg)
		resp.MsgHdr.Response = true
		resp.Answer = []dns.RR{}
		resp.Extra = []dns.RR{}
		s.composeLookupAnswers(service, resp, s.ttl)
	*/

	// From RFC6762
	//    The Multicast DNS responder MUST send at least two unsolicited
	//    responses, one second apart. To provide increased robustness against
	//    packet loss, a responder MAY send up to eight unsolicited responses,
	//    provided that the interval between unsolicited responses increases by
	//    at least a factor of two with every response sent.
	timeout := 1 * time.Second
	for i := 0; i < 3; i++ {
		if err := nss.multicastResponse(resp); err != nil {
			log.Println("[ERR] bonjour: failed to send announcement:", err.Error())
		}
		time.Sleep(timeout)
		timeout *= 2
	}
}
