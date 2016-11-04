package dnssd

import (
	"context"
	"fmt"
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func TestQuery1(t *testing.T) {
	ds, _ = makeTestDnssd(t)
	go ds.processing()

	rrc := make(chan string, 5)
	defer close(rrc)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errc := func(err error) {
		rrc <- fmt.Sprint("TestQuery err=", err)
	}

	Query(ctx, 0, 0, &dns.Question{Name: "turner.local.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
		func(flags Flags, ifIndex int, rr dns.RR) {
			dnssdlog.Debug.Println("-----------------> ", rr)
			rrc <- rr.String()
		}, errc)

	ds.ns.msgCh <- fakeIncomingMsg(true).addRR("turner.local.", dns.TypeA, "10.20.30.40")
	assert.Equal(t, "turner.local.\t0\tIN\tA\t10.20.30.40", <-rrc)

	assert.Equal(t, 1, len(ds.ns.query.Question))
	assert.Equal(t, ";turner.local.\tIN\t A", ds.ns.query.Question[0].String())

}
