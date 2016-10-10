package dnssd

import (
	"context"
	"fmt"
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func NoTestQuery1(t *testing.T) {
	rrc := make(chan dns.RR)
	ctx := context.Background()
	Query(ctx, 0, 0, &dns.Question{Name: "turner.local.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
		func(flags Flags, ifIndex int, rr dns.RR) {
			rrc <- rr
		}, func(err error) {
			fmt.Println("TestQuery1 err=", err)
		})
	assert.NotNil(t, ctx)
	for ii := 0; ii < 1; ii++ {
		b := <-rrc
		fmt.Println("b=", b)
	}
	println("done...")

}
