package dnssd

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func TestRegister1(t *testing.T) {
	ds, _ = makeTestDnssd(t)
	go ds.processing()

	rrc := make(chan string, 5)
	defer close(rrc)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errc := func(err error) {
		rrc <- fmt.Sprint("TestQuery err=", err)
	}

	txt := []string{"test=hej", "tjo=hopp"}
	addrecord := Register(ctx, 0, 3, "Stryfnake", "_tuting._tcp", "", "myhost", 4711, txt, func(flags int, serviceName, regType, domain string) {
		fmt.Println("Register: serviceName=", serviceName, ", regType=", regType, ",domain=", domain)
		rrc <- fmt.Sprint("Register: serviceName=", serviceName, ", regType=", regType, ",domain=", domain)
	}, errc)
	assert.NotNil(t, ctx)

	mxRR := new(dns.MX)
	mxRR.Hdr = dns.RR_Header{Rrtype: dns.TypeMX, Class: dns.ClassINET, Ttl: 3200}
	mxRR.Mx = "xx"
	addrecord(0, mxRR)

	assertMessage(t, time.Second, "Register: serviceName=Stryfnake, regType=_tuting._tcp,domain=local", rrc)
	time.Sleep(1 * time.Millisecond)
	assert.Equal(t, 3, len(ds.ns.response.Answer))
	assertResponse(t, "Stryfnake._tuting._tcp.local.\t20\tIN\tSRV\t0 0 4711 myhost.local.", ds.ns.response.Answer)
	assertResponse(t, "Stryfnake._tuting._tcp.local.\t3200\tIN\tTXT\t\"test=hej\" \"tjo=hopp\"", ds.ns.response.Answer)
	assertResponse(t, "_tuting._tcp.local.\t3200\tIN\tPTR\tStryfnake._tuting._tcp.local.", ds.ns.response.Answer)

}

func assertMessage(t *testing.T, timeout time.Duration, expected string, msgch <-chan string) {
	tmr := time.NewTimer(timeout)
	select {
	case v := <-msgch:
		assert.Equal(t, expected, v)
	case <-tmr.C:
		assert.Fail(t, fmt.Sprint("Timeout (", timeout.String(), ") waiting for '", expected, "'"))
	}
}

func assertResponse(t *testing.T, expected string, rrs []dns.RR) {
	for _, rr := range rrs {
		if expected == rr.String() {
			return
		}
	}
	assert.Fail(t, fmt.Sprint("Could not find response '", expected, "'"))
}
