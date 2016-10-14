package dnssd

import (
	"context"
	"fmt"
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func XTestBrowse(t *testing.T) {
	rrc := make(chan bool)
	ctx := context.Background()
	Browse(ctx, 0, 0, "_raop._tcp", "local",
		func(found bool, flags Flags, ifIndex int, serviceName, regType, domain string) {
			fmt.Println("serviceName=", serviceName)
			rrc <- true
		}, func(err error) {
			fmt.Println("TestBrowse1 err=", err)
		})
	assert.NotNil(t, ctx)
	for ii := 0; ii < 1; ii++ {
		b := <-rrc
		fmt.Println("b=", b)
	}
	println("done...")
}

func NoTestBrowseAndResolve(t *testing.T) {
	rrc := make(chan bool)
	ctx := context.Background()
	Browse(ctx, 0, 0, "_raop._tcp", "local",
		func(found bool, flags Flags, ifIndex int, serviceName, regType, domain string) {
			fmt.Println("serviceName=", serviceName, ", regType=", regType, ", domain=", domain)
			Resolve(ctx, 0, 0, serviceName, regType, domain,
				func(flags Flags, ifIndex int, fullName, hostName string, port uint16, txt []string) {
					fmt.Println("serviceName=", serviceName, ", hostname=", hostName, ", port=", port)
				}, func(err error) {
					fmt.Println("TestBrowse1 err=", err)
				})
		}, func(err error) {
			fmt.Println("TestBrowseAndResolve err=", err)
		})
	assert.NotNil(t, ctx)
	for ii := 0; ii < 1; ii++ {
		b := <-rrc
		fmt.Println("b=", b)
	}
	println("done...")
}

func NoTestBrowseAndResolveAndLookup(t *testing.T) {
	prefix := "-------------- "
	rrc := make(chan bool)
	ctx := context.Background()
	errc := func(err error) {
		fmt.Println(prefix, "TestBrowseAndResolve err=", err)
	}
	Browse(ctx, 0, 0, "_raop._tcp", "local",
		func(found bool, flags Flags, ifIndex int, serviceName, regType, domain string) {
			fmt.Println(prefix, "serviceName=", serviceName, ", regType=", regType, ", domain=", domain)
			Resolve(ctx, 0, 0, serviceName, regType, domain,
				func(flags Flags, ifIndex int, fullName, hostName string, port uint16, txt []string) {
					fmt.Println(prefix, "serviceName=", serviceName, ", hostname=", hostName, ", port=", port)
					Query(ctx, 0, 0, hostName, dns.TypeA, dns.ClassINET,
						func(flags Flags, ifIndex int, rr dns.RR) {
							a := rr.(*dns.A)
							fmt.Println(prefix, "!!!! ", serviceName, hostName, ":", port, a.A)
							rrc <- true

						}, errc)
					Query(ctx, 0, 0, hostName, dns.TypeAAAA, dns.ClassINET,
						func(flags Flags, ifIndex int, rr dns.RR) {
							a := rr.(*dns.AAAA)
							fmt.Println(prefix, "!!!! ", serviceName, hostName, ":", port, a.AAAA)
							rrc <- true

						}, errc)

				}, errc)
		}, errc)
	assert.NotNil(t, ctx)
	for ii := 0; ii < 10; ii++ {
		b := <-rrc
		fmt.Println("b=", b)
	}
	println("done...")
}
