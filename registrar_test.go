package dnssd

import (
	"context"
	"net"
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func NoTestRegistrar(t *testing.T) {
	d := make(chan bool)

	register := CreateRecordRegistrar(func(record dns.RR, flags int) {
		close(d)
	}, func(err error) {
		close(d)
	})
	assert.NotNil(t, register)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "name", "test")
	rr := new(dns.A)
	rr.Hdr = dns.RR_Header{Name: "tuting.local.", Rrtype: dns.TypeA,
		Class: dns.ClassINET, Ttl: 3600}
	rr.A = net.IPv4(10, 20, 30, 40)
	register(ctx, Shared, 0, rr)

	<-d
}

func NoTestRegistrarTwice(t *testing.T) {
	d := make(chan bool)

	register := CreateRecordRegistrar(func(record dns.RR, flags int) {
		close(d)
	}, func(err error) {
		close(d)
	})
	assert.NotNil(t, register)

	ctx := context.Background()
	ctx1 := context.WithValue(ctx, "name", "test1")
	rr := new(dns.A)
	rr.Hdr = dns.RR_Header{Name: "tuting.local.", Rrtype: dns.TypeA,
		Class: dns.ClassINET, Ttl: 3600}
	rr.A = net.IPv4(10, 20, 30, 40)
	register(ctx1, Shared, 0, rr)
	<-d

	d = make(chan bool)
	ctx2 := context.WithValue(ctx, "name", "test2")
	register(ctx2, 0, 0, rr)

	<-d
}

func NoTestRegistrarConflict(t *testing.T) {
	d := make(chan bool)

	register := CreateRecordRegistrar(func(record dns.RR, flags int) {
		close(d)
	}, func(err error) {
		close(d)
	})
	assert.NotNil(t, register)

	ctx := context.Background()
	ctx1 := context.WithValue(ctx, "name", "test1")
	rr1 := new(dns.A)
	rr1.Hdr = dns.RR_Header{Name: "tuting.local.", Rrtype: dns.TypeA,
		Class: dns.ClassINET, Ttl: 3600}
	rr1.A = net.IPv4(10, 20, 30, 40)
	register(ctx1, Shared, 0, rr1)
	<-d

	d = make(chan bool)
	rr2 := new(dns.A)
	rr2.Hdr = dns.RR_Header{Name: "tuting.local.", Rrtype: dns.TypeA,
		Class: dns.ClassINET, Ttl: 3600}
	rr2.A = net.IPv4(10, 20, 30, 41)
	ctx2 := context.WithValue(ctx, "name", "test2")
	register(ctx2, 0, 0, rr2)

	<-d
}
