package dnssd

import (
	"fmt"
	"net"
	"sync"

	"github.com/miekg/dns"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

type netservers struct {
	servers map[int]*netserver
	msgCh   chan *incomingMsg
}

type netserver struct {
	iface    net.Interface
	ipv4conn *net.UDPConn
	ipv6conn *net.UDPConn

	closed    bool
	msgCh     chan *incomingMsg
	closeLock sync.Mutex
}

type incomingMsg struct {
	msg     *dns.Msg
	ifIndex int
	from    net.Addr
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
	return fmt.Sprintf("IM{%d,%s,%s}", im.ifIndex, im.from, im.msg)
}

func makeNetservers() (ns *netservers, err error) {
	msgCh := make(chan *incomingMsg, 32)

	ifaces, err := net.Interfaces()
	if err != nil {
		return
	}
	ns = &netservers{make(map[int]*netserver), msgCh}

	for _, iface := range ifaces {
		ns.addInterface(iface)
		if err != nil {
			return
		}
	}
	return
}

func (nss *netservers) addInterface(iface net.Interface) (err error) {
	nss.servers[iface.Index], err = makeNewNetserver(iface, nss.msgCh)
	return
}

func (nss *netservers) sendMessage(msg *dns.Msg) error {
	for _, ns := range nss.servers {
		ns.log("netservers::sendMessage")
		ns.sendMessage(msg)
	}
	return nil
}

func (nss *netservers) shutdown() error {
	for _, ns := range nss.servers {
		ns.shutdown()
	}
	return nil
}

func makeNewNetserver(iface net.Interface, msgCh chan *incomingMsg) (*netserver, error) {

	ns := &netserver{iface: iface, msgCh: msgCh}
	var err error
	// Create wildcard connections (because :5353 can be already taken by other apps)
	ns.ipv4conn, err = net.ListenUDP("udp4", mdnsWildcardAddrIPv4)
	if err != nil {
		log.Printf("[ERR] dnssd: Failed to bind to udp4 port: %v", err)
	}
	ns.ipv6conn, err = net.ListenUDP("udp6", mdnsWildcardAddrIPv6)
	if err != nil {
		log.Printf("[ERR] dnssd: Failed to bind to udp6 port: %v", err)
	}
	if ns.ipv4conn == nil && ns.ipv6conn == nil {
		ns.log("[ERR] dnssd: Failed to bind to any udp port!")
		return nil, fmt.Errorf("[ERR] dnssd: Failed to bind to any udp port!")
	}

	// Join multicast groups to receive announcements
	p1 := ipv4.NewPacketConn(ns.ipv4conn)
	p2 := ipv6.NewPacketConn(ns.ipv6conn)

	if err := p1.JoinGroup(&iface, &net.UDPAddr{IP: mdnsGroupIPv4}); err != nil {
		return nil, err
	}
	if err := p2.JoinGroup(&iface, &net.UDPAddr{IP: mdnsGroupIPv6}); err != nil {
		return nil, err
	}

	ns.startReceiving()
	ns.log("Started server, ifIndex=", iface.Index)
	return ns, nil
}

func (nss *netserver) startReceiving() {
	if nss.ipv4conn != nil {
		go nss._recv(nss.ipv4conn)
	}
	if nss.ipv6conn != nil {
		go nss._recv(nss.ipv6conn)
	}
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
	rmsg := &incomingMsg{&msg, s.iface.Index, from}
	nss.log("RX: msg=", rmsg.msg)
	nss.msgCh <- rmsg
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
func (nss *netserver) sendMessage(msg *dns.Msg) error {
	nss.log("sendMessage", msg)
	buf, err := msg.Pack()
	if err != nil {
		nss.log("Failed to pack message!")
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

func (nss *netserver) log(msg ...interface{}) {
	//fmt.Print(s.iface.Name)
	//fmt.Println(msg)
}
