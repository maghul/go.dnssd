package dnssd

import (
	"context"
	"fmt"
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func TestBrowse(t *testing.T) {
	rrc := make(chan bool)
	defer close(rrc)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	Browse(ctx, 0, 0, "_raop._tcp", "local",
		func(found bool, flags Flags, ifIndex int, serviceName, regType, domain string) {
			fmt.Println("serviceName=", serviceName)
			select {
			case rrc <- true:
			default:
			}
		}, func(err error) {
			fmt.Println("TestBrowse1 err=", err)
		})
	assert.NotNil(t, ctx)
	for ii := 0; ii < 10; ii++ {
		b := <-rrc
		fmt.Println("b=", b)
	}
	println("done...")
}

func TestBrowseAndResolve(t *testing.T) {
	rrc := make(chan bool)
	defer close(rrc)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	Browse(ctx, 0, 0, "_raop._tcp", "local",
		func(found bool, flags Flags, ifIndex int, serviceName, regType, domain string) {
			fmt.Println(">>> TEST BROWSE: serviceName=", serviceName, ", regType=", regType, ", domain=", domain)
			Resolve(ctx, 0, 0, serviceName, regType, domain,
				func(flags Flags, ifIndex int, fullName, hostName string, port uint16, txt []string) {
					fmt.Println(">>> TEST RESOLVE: serviceName=", serviceName, ", hostname=", hostName, ", port=", port)
					select {
					case rrc <- true:
					default:
					}
					fmt.Println("<<< TEST RESOLVE: serviceName=", serviceName, ", hostname=", hostName, ", port=", port)
				}, func(err error) {
					fmt.Println("!!!TEST RESOLVE: TestBrowse1 err=", err)
				})
			fmt.Println("<<< TEST BROWSE: serviceName=", serviceName, ", regType=", regType, ", domain=", domain)
		}, func(err error) {
			fmt.Println("TEST BROWSE: TestBrowseAndResolve err=", err)
		})
	assert.NotNil(t, ctx)
	for ii := 0; ii < 10; ii++ {
		<-rrc
	}
	println("done...")
}

func TestBrowseAndResolveAndLookup(t *testing.T) {
	prefix := "-------------- "
	rrc := make(chan string)

	ctx := context.Background()
	errc := func(err error) {
		fmt.Println(prefix, "TestBrowseAndResolve err=", err)
	}
	Browse(ctx, 0, 0, "_raop._tcp", "local",
		func(found bool, flags Flags, ifIndex int, serviceName, regType, domain string) {
			fmt.Println(prefix, "TEST BROWSE: ifIndex=", ifIndex, ", serviceName=", serviceName, ", regType=", regType, ", domain=", domain)
			Resolve(ctx, 0, 0, serviceName, regType, domain,
				func(flags Flags, ifIndex int, fullName, hostName string, port uint16, txt []string) {
					fmt.Println(prefix, "TEST RESOLVE: serviceName=", serviceName, ", hostname=", hostName, ", port=", port)
					Query(ctx, 0, 0, hostName, dns.TypeA, dns.ClassINET,
						func(flags Flags, ifIndex int, rr dns.RR) {
							a := rr.(*dns.A)
							fmt.Println(prefix, "TEST QUERY: serviceName=", serviceName, ", hostName=", hostName, ":", port, ", A=", a.A)
							select {
							case rrc <- fmt.Sprint("RESULT: serviceName=", serviceName, ", ifIndex=", ifIndex, ", hostName=", hostName, ":", port, ", A=", a.A):
							default:
							}

						}, errc)
					Query(ctx, 0, 0, hostName, dns.TypeAAAA, dns.ClassINET,
						func(flags Flags, ifIndex int, rr dns.RR) {
							a := rr.(*dns.AAAA)
							fmt.Println(prefix, "TEST QUERY: serviceName=", serviceName, ", hostName=", hostName, ":", port, ", AAAA=", a.AAAA)
							select {
							case rrc <- fmt.Sprint("RESULT: serviceName=", serviceName, ", ifIndex=", ifIndex, ", hostName=", hostName, ":", port, ", AAAA=", a.AAAA):
							default:
							}

						}, errc)

				}, errc)
		}, errc)
	assert.NotNil(t, ctx)
	for ii := 0; ii < 40; ii++ {
		b := <-rrc
		fmt.Println("RESULT... ", b)
	}

	println("done...")
}
