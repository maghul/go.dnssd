package dnssd

import (
	"fmt"
	"net"

	"github.com/miekg/dns"
)

func makeTestNetserver() (ns *netserver, err error) {
	msgCh := make(chan *incomingMsg, 32)

	ns = &netserver{response: &dns.Msg{}, query: &dns.Msg{}}
	ns.msgCh = msgCh

	return
}

func fakeIncomingMsg(response bool) *incomingMsg {
	addr, err := net.ResolveUDPAddr("udp", "192.168.117.17:5353")
	if err != nil {
		panic(err)
	}
	msg := &dns.Msg{}
	msg.Response = response
	return &incomingMsg{msg, 2, addr}
}

func (im *incomingMsg) addRR(name string, rrtype uint16, args ...interface{}) *incomingMsg {
	var rr dns.RR
	hdr := dns.RR_Header{Name: name, Class: dns.ClassINET, Rrtype: rrtype}
	switch rrtype {
	case dns.TypeA:
		ip, err := net.ResolveIPAddr("ip4", args[0].(string))
		if err != nil {
			panic(err)
		}
		rr = &dns.A{Hdr: hdr, A: ip.IP}
	case dns.TypeAAAA:
		ip, err := net.ResolveIPAddr("ip6", args[0].(string))
		if err != nil {
			panic(err)
		}
		rr = &dns.AAAA{Hdr: hdr, AAAA: ip.IP}
	case dns.TypePTR:
		rr = &dns.PTR{Hdr: hdr, Ptr: args[0].(string)}
	case dns.TypeSRV:
		rr = &dns.SRV{Hdr: hdr,
			Weight:   uint16(args[0].(int)),
			Priority: uint16(args[1].(int)),
			Port:     uint16(args[2].(int)),
			Target:   args[3].(string)}
	case dns.TypeTXT:
		txt := make([]string, len(args))
		for ii, v := range args {
			txt[ii] = v.(string)
		}
		rr = &dns.TXT{Hdr: hdr, Txt: txt}
	default:
		panic(fmt.Sprint("Cant make RR of type ", rrtype))
	}
	im.msg.Answer = append(im.msg.Answer, rr)
	return im
}
