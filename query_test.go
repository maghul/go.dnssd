package dnssd

import (
	"context"
	"fmt"
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func TestQuery1(t *testing.T) {
	rrc := make(chan dns.RR)
	ctx := context.Background()
	Query(ctx, 0, 0, "turner.local.", dns.TypeA, dns.ClassINET,
		func(err error, flags Flags, ifIndex int, rr dns.RR) {
			if err != nil {
				fmt.Println("TestQuery1 err=", err)
			} else {
				rrc <- rr
			}
		})
	assert.NotNil(t, ctx)
	for ii := 0; ii < 10; ii++ {
		b := <-rrc
		fmt.Println("b=", b)
	}
	println("done...")

}
