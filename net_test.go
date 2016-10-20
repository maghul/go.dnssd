package dnssd

import (
	"github.com/miekg/dns"
)

func makeTestNetserver() (ns *netserver, err error) {
	msgCh := make(chan *incomingMsg, 32)

	ns = &netserver{response: &dns.Msg{}, query: &dns.Msg{}}
	ns.msgCh = msgCh

	return
}
