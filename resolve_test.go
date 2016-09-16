package dnssd

import (
	"context"
	"fmt"
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func TestResolve1(t *testing.T) {
	rrc := make(chan dns.RR)
	ctx := context.Background()
	Resolve(ctx, 0, 0, "rafael", "_airplay._tcp", "local",
		func(err error, flags Flags, ifIndex int, fullName, hostName string, port uint16, txt []string) {
			if err != nil {
				fmt.Println("TestResolve1 err=", err)
			} else {
				fmt.Println("Resolved: name=", fullName, ", host=", hostName, ", port=", port, ", text=", txt)
				close(rrc)
			}
		})
	assert.NotNil(t, ctx)
	for ii := 0; ii < 1; ii++ {
		b := <-rrc
		fmt.Println("b=", b)
	}
	println("done...")
}
