package dnssd

import (
	"context"
	"fmt"
	"testing"

	"github.com/maghul/go.slf"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func TestResolve1(t *testing.T) {
	defer parentlog.SetLevel(slf.Off)

	ds, _ = makeTestDnssd(t)
	go ds.processing()

	rrc := make(chan string, 5)
	defer close(rrc)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	Resolve(ctx, 0, 0, "rafael", "_airplay._tcp", "local",
		func(flags Flags, ifIndex int, fullName, hostName string, port uint16, txt []string) {
			rrc <- fmt.Sprint("Resolved: name=", fullName, ", host=", hostName, ", port=", port, ", text=", txt)
		}, func(err error) {
			rrc <- fmt.Sprint("TestResolve1 err=", err)
		})
	ds.ns.msgCh <- fakeIncomingMsg(true).
		addRR("rafael._airplay._tcp.local.", dns.TypeSRV, 0, 0, 4711, "www.facebook.it").
		addRR("rafael._airplay._tcp.local.", dns.TypeTXT, "hi=there")

	assert.Equal(t, "Resolved: name=rafael._airplay._tcp.local., host=www.facebook.it, port=4711, text=[hi=there]", <-rrc)
}
