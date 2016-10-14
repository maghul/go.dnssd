package dnssd

import (
	"context"
	"fmt"
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func TestRegister1(t *testing.T) {
	bc := make(chan string)

	errc := func(err error) {
		fmt.Println("TestBrowseAndResolve err=", err)
	}
	txt := []string{"test=hej", "tjo=hopp"}
	ctx := context.Background()
	addrecord := Register(ctx, 0, 3, "Stryfnake", "_tuting._tcp", "", "", 4711, txt, func(flags int, serviceName, regType, domain string) {
		fmt.Println("Register: serviceName=", serviceName, ", regType=", regType, ",domain=", domain)
		bc <- fmt.Sprint("Register: serviceName=", serviceName, ", regType=", regType, ",domain=", domain)
	}, errc)
	assert.NotNil(t, ctx)

	mxRR := new(dns.MX)
	mxRR.Hdr = dns.RR_Header{Rrtype: dns.TypeMX, Class: dns.ClassINET, Ttl: 3200}
	mxRR.Mx = "xx"
	addrecord(0, mxRR)

	r := <-bc
	fmt.Println("RESULT: ", r)
}
