package dnssd

import (
	"fmt"
	"net"
	"strings"
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

	p1 *ipv4.PacketConn
	p2 *ipv6.PacketConn

	closed    bool
	msgCh     chan *incomingMsg
	closeLock sync.Mutex

	query, response *dns.Msg
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

func (nss *netservers) sendPending() {
	for _, ns := range nss.servers {
		ns.sendPending()
	}
}

func (nss *netserver) sendPending() {
	nss.send(&nss.query, false)
	nss.send(&nss.response, true)
}

func (nss *netserver) send(msgp **dns.Msg, response bool) {
	msg := *msgp

	if msg != nil && (len(msg.Question) > 0 || len(msg.Answer) > 0) {
		nss.log("send!... ", msg)
		nss.sendMessage(msg)
	}
	msg = &dns.Msg{}
	msg.Response = response
	*msgp = msg
}

func (nss *netservers) publish(ifIndex int, rr dns.RR) {
	for _, ns := range nss.servers {
		if ifIndex == 0 || ifIndex == ns.iface.Index {
			ns.log("publish: rr=", rr)
			ns.response.Answer = append(ns.response.Answer, rr)
		}
	}
}

func (nss *netservers) addKnownAnswer(ifIndex int, rr dns.RR) {
	for _, ns := range nss.servers {
		if ifIndex == 0 || ifIndex == ns.iface.Index {
			ns.log("addKnownAnswer: rr=", rr)
			ns.query.Answer = append(ns.query.Answer, rr)
		}
	}
}

func (nss *netservers) addQuestion(ifIndex int, q *dns.Question) {
	for _, ns := range nss.servers {
		if ifIndex == 0 || ifIndex == ns.iface.Index {
			ns.log("addQuestion: q=", q)
			ns.query.Question = append(ns.query.Question, *q)
		}
	}
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
	ns.p1 = ipv4.NewPacketConn(ns.ipv4conn)
	ns.p2 = ipv6.NewPacketConn(ns.ipv6conn)

	fmt.Println("JoinGroup iface=", ns.iface, " IP=", mdnsGroupIPv4)
	if err := ns.p1.JoinGroup(&ns.iface, &net.UDPAddr{IP: mdnsGroupIPv4}); err != nil {
		return nil, err
	}
	ns.p1.SetMulticastInterface(&ns.iface)

	fmt.Println("JoinGroup iface=", ns.iface, " IP=", mdnsGroupIPv6)
	if err := ns.p2.JoinGroup(&ns.iface, &net.UDPAddr{IP: mdnsGroupIPv6}); err != nil {
		return nil, err
	}
	ns.p2.SetMulticastInterface(&ns.iface)

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
	rmsg := &incomingMsg{&msg, nss.iface.Index, from}
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
	buf, err := msg.Pack()
	if err != nil {
		nss.log("Failed to pack message! msg=", msg)
		return err
	}
	if nss.p1 != nil {
		nss.p1.WriteTo(buf, nil, ipv4Addr)
	}
	if nss.p2 != nil {
		nss.p2.WriteTo(buf, nil, ipv6Addr)
	}
	return nil
}

func (*netservers) log(msg ...interface{}) {
	//fmt.Println("NETSERVERS", msg)
}

func (nss *netserver) sendKnownAnswer(ifIndex int, rr dns.RR) {
	nss.query.Answer = append(nss.query.Answer, rr)
}

func getInterface(ifIndex int) *net.Interface {
	switch ifIndex {
	case 0:
		return nil
	case -1:
		interfaces, _ := net.Interfaces()
		for _, iface := range interfaces {
			if strings.HasPrefix(iface.Name, "lo") {
				return &iface
			}
		}
	default:
		interfaces, _ := net.Interfaces()
		return &interfaces[ifIndex]
		// TODO: Maybe use an error here instead of panicing on out-of-bounds.
	}
	return nil
}

func getOwnDomainname() string {
	return "local" // TODO: fix?
}
