package dnssd

import (
	"context"
	"net"

	"github.com/miekg/dns"
)

func publishIpAddr(rgr RegisterRecord, name string, ip net.IP) {
	hdr := dns.RR_Header{}
	hdr.Name = name
	hdr.Class = dns.ClassINET
	hdr.Ttl = 3200

	ctx := context.Background()
	ip4 := ip.To4()
	if ip4 != nil {
		hdr.Rrtype = dns.TypeA
		rgr(ctx, 0, &dns.A{Hdr: hdr, A: ip4})
	}
	ip6 := ip.To16()
	if ip6 != nil {
		hdr.Rrtype = dns.TypeAAAA
		rgr(ctx, 0, &dns.AAAA{Hdr: hdr, AAAA: ip6})
	}

}

func startup() {

	rgr := CreateRecordRegistrar(func(record dns.RR, flags int) {

	}, func(err error) {

	})

	//	publishIpAddr(rgr, "durer.local.", net.ParseIP("10.223.10.110"))
	publishIpAddr(rgr, "flurer.local.", net.ParseIP("10.123.10.110"))

}
