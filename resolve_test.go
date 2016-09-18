package dnssd

import (
	"context"
	"fmt"
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func NoTestResolve1(t *testing.T) {
	rrc := make(chan dns.RR)
	ctx := context.Background()
	Resolve(ctx, 0, 0, "rafael", "_airplay._tcp", "local",
		func(flags Flags, ifIndex int, fullName, hostName string, port uint16, txt []string) {
			fmt.Println("Resolved: name=", fullName, ", host=", hostName, ", port=", port, ", text=", txt)
			close(rrc)
		}, func(err error) {
			fmt.Println("TestResolve1 err=", err)
		})
	assert.NotNil(t, ctx)
	for ii := 0; ii < 1; ii++ {
		b := <-rrc
		fmt.Println("b=", b)
	}
	println("done...")
}
