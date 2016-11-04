package dnssd

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func TestBrowse(t *testing.T) {
	ds, _ = makeTestDnssd(t)
	go ds.processing()
	dnssdlog.Debug.Println("Start Browse...")

	rrc := make(chan string, 5)
	defer close(rrc)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	Browse(ctx, 0, 0, "_raop._tcp", "local",
		func(found bool, flags Flags, ifIndex int, serviceName, regType, domain string) {
			select {
			case rrc <- serviceName:
			default:
			}
		}, func(err error) {
			rrc <- fmt.Sprint("TestBrowse1 err=", err)
		})

	ds.ns.msgCh <- fakeIncomingMsg(true).addRR("_raop._tcp.local.", dns.TypePTR, "tjosan._raop._tcp.local.")
	assert.Equal(t, "tjosan", <-rrc)

	ds.ns.msgCh <- fakeIncomingMsg(true).
		addRR("_raop._tcp.local.", dns.TypePTR, "hejsan._raop._tcp.local.").
		addRR("_raop._tcp.local.", dns.TypePTR, "hoppsan._raop._tcp.local.")
	assert.Equal(t, "hejsan", <-rrc)
	assert.Equal(t, "hoppsan", <-rrc)

	assert.Equal(t, 1, len(ds.ns.query.Question))
	assert.Equal(t, ";_raop._tcp.local.\tIN\t PTR", ds.ns.query.Question[0].String())
}

func TestBrowseAndResolve(t *testing.T) {
	ds, _ = makeTestDnssd(t)
	go ds.processing()

	rrc := make(chan string, 5)
	defer close(rrc)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	Browse(ctx, 0, 0, "_raop._tcp", "local",
		func(found bool, flags Flags, ifIndex int, serviceName, regType, domain string) {
			rrc <- serviceName
			Resolve(ctx, 0, 0, serviceName, regType, domain,
				func(flags Flags, ifIndex int, fullName, hostName string, port uint16, txt []string) {
					select {
					case rrc <- fmt.Sprint(serviceName, ":", hostName, ":", port, ":", txt):
					default:
					}
				}, func(err error) {
					rrc <- fmt.Sprint("!!!TEST RESOLVE: TestBrowse1 err=", err)
				})
		}, func(err error) {
			rrc <- fmt.Sprint("TEST BROWSE: TestBrowseAndResolve err=", err)
		})

	ds.ns.msgCh <- fakeIncomingMsg(true).addRR("_raop._tcp.local.", dns.TypePTR, "tjosan._raop._tcp.local.")
	assert.Equal(t, "tjosan", <-rrc)

	ds.ns.msgCh <- fakeIncomingMsg(true).
		addRR("tjosan._raop._tcp.local.", dns.TypeSRV, 0, 0, 4711, "www.facebook.it").
		addRR("tjosan._raop._tcp.local.", dns.TypeTXT, "hi=there")

	assert.Equal(t, "tjosan:www.facebook.it:4711:[hi=there]", <-rrc)

	assert.Equal(t, 3, len(ds.ns.query.Question))
	assert.Equal(t, ";_raop._tcp.local.\tIN\t PTR", ds.ns.query.Question[0].String())
	assert.Equal(t, ";tjosan._raop._tcp.local.\tIN\t SRV", ds.ns.query.Question[1].String())
	assert.Equal(t, ";tjosan._raop._tcp.local.\tIN\t TXT", ds.ns.query.Question[2].String())
}

func TestBrowseAndResolveAndLookup(t *testing.T) {
	ds, _ = makeTestDnssd(t)
	go ds.processing()

	rrc := make(chan string, 5)
	defer close(rrc)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errc := func(err error) {
		rrc <- fmt.Sprint("TestBrowseAndResolve err=", err)
	}
	Browse(ctx, 0, 0, "_raop._tcp", "local",
		func(found bool, flags Flags, ifIndex int, serviceName, regType, domain string) {
			testlog.Debug.Println("TEST BROWSE: ifIndex=", ifIndex, ", serviceName=", serviceName, ", regType=", regType, ", domain=", domain)
			rrc <- serviceName
			Resolve(ctx, 0, ifIndex, serviceName, regType, domain,
				func(flags Flags, ifIndex int, fullName, hostName string, port uint16, txt []string) {
					testlog.Debug.Println("TEST RESOLVE: ifIndex=", ifIndex, ",serviceName=", serviceName, ", hostname=", hostName, ", port=", port)
					rrc <- fmt.Sprint(serviceName, ":", hostName, ":", port, ":", txt)
					Query(ctx, 0, ifIndex, &dns.Question{Name: hostName, Qtype: dns.TypeA, Qclass: dns.ClassINET},
						func(flags Flags, ifIndex int, rr dns.RR) {
							a := rr.(*dns.A)
							testlog.Debug.Println("TEST QUERY: ifIndex=", ifIndex, ",serviceName=", serviceName, ", hostName=", hostName, ":", port, ", A=", a.A)
							select {
							case rrc <- fmt.Sprint("RESULT: serviceName=", serviceName, ", ifIndex=", ifIndex, ", hostName=", hostName, ":", port, ", A=", a.A):
							default:
							}

						}, errc)
					Query(ctx, 0, ifIndex, &dns.Question{Name: hostName, Qtype: dns.TypeAAAA, Qclass: dns.ClassINET},
						func(flags Flags, ifIndex int, rr dns.RR) {
							a := rr.(*dns.AAAA)
							testlog.Debug.Println("TEST QUERY: ifIndex=", ifIndex, ",serviceName=", serviceName, ", hostName=", hostName, ":", port, ", AAAA=", a.AAAA)
							select {
							case rrc <- fmt.Sprint("RESULT: serviceName=", serviceName, ", ifIndex=", ifIndex, ", hostName=", hostName, ":", port, ", AAAA=", a.AAAA):
							default:
							}

						}, errc)

				}, errc)
		}, errc)

	ds.ns.msgCh <- fakeIncomingMsg(true).addRR("_raop._tcp.local.", dns.TypePTR, "tjosan._raop._tcp.local.")
	assert.Equal(t, "tjosan", <-rrc)

	ds.ns.msgCh <- fakeIncomingMsg(true).
		addRR("tjosan._raop._tcp.local.", dns.TypeSRV, 0, 0, 4711, "www.facebook.it").
		addRR("tjosan._raop._tcp.local.", dns.TypeTXT, "hi=there").
		addRR("www.facebook.it", dns.TypeA, "192.168.112.77").
		addRR("www.facebook.it", dns.TypeAAAA, "192.168.112.77")

	assert.Equal(t, "tjosan:www.facebook.it:4711:[hi=there]", <-rrc)
	assert.Equal(t, "RESULT: serviceName=tjosan, ifIndex=2, hostName=www.facebook.it:4711, A=192.168.112.77", <-rrc)
	assert.Equal(t, "RESULT: serviceName=tjosan, ifIndex=2, hostName=www.facebook.it:4711, AAAA=192.168.112.77", <-rrc)

	time.Sleep(1 * time.Millisecond)
	assert.Equal(t, 5, len(ds.ns.query.Question))
	assert.Equal(t, ";_raop._tcp.local.\tIN\t PTR", ds.ns.query.Question[0].String())
	assert.Equal(t, ";tjosan._raop._tcp.local.\tIN\t SRV", ds.ns.query.Question[1].String())
	assert.Equal(t, ";tjosan._raop._tcp.local.\tIN\t TXT", ds.ns.query.Question[2].String())
	assert.Equal(t, ";www.facebook.it\tIN\t A", ds.ns.query.Question[3].String())
	assert.Equal(t, ";www.facebook.it\tIN\t AAAA", ds.ns.query.Question[4].String())
}
