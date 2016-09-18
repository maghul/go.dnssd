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
		func(err error, found bool, flags Flags, ifIndex int, serviceName, regType, domain string) {
			if err != nil {
				fmt.Println("TestBrowse1 err=", err)
			} else {
				fmt.Println("serviceName=", serviceName)
				rrc <- true
			}
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
		func(err error, found bool, flags Flags, ifIndex int, serviceName, regType, domain string) {
			if err != nil {
				fmt.Println("TestBrowseAndResolve err=", err)
			} else {
				fmt.Println("serviceName=", serviceName, ", regType=", regType, ", domain=", domain)
				Resolve(ctx, 0, 0, serviceName, regType, domain,
					func(err error, flags Flags, ifIndex int, fullName, hostName string, port uint16, txt []string) {
						if err != nil {
							fmt.Println("TestBrowse1 err=", err)
						} else {
							fmt.Println("serviceName=", serviceName, ", hostname=", hostName, ", port=", port)
						}
					})
			}
		})
	assert.NotNil(t, ctx)
	for ii := 0; ii < 1; ii++ {
		b := <-rrc
		fmt.Println("b=", b)
	}
	println("done...")
}

func TestBrowseAndResolveAndLookup(t *testing.T) {
	prefix := "-------------- "
	rrc := make(chan bool)
	ctx := context.Background()
	Browse(ctx, 0, 0, "_raop._tcp", "local",
		func(err error, found bool, flags Flags, ifIndex int, serviceName, regType, domain string) {
			if err != nil {
				fmt.Println(prefix, "TestBrowseAndResolve err=", err)
				return
			}
			fmt.Println(prefix, "serviceName=", serviceName, ", regType=", regType, ", domain=", domain)
			Resolve(ctx, 0, 0, serviceName, regType, domain,
				func(err error, flags Flags, ifIndex int, fullName, hostName string, port uint16, txt []string) {
					if err != nil {
						fmt.Println(prefix, "TestBrowse1 err=", err)
						return
					}
					fmt.Println(prefix, "serviceName=", serviceName, ", hostname=", hostName, ", port=", port)
					Query(ctx, 0, 0, hostName, dns.TypeA, dns.ClassINET,
						func(err error, flags Flags, ifIndex int, rr dns.RR) {
							if err != nil {
								fmt.Println(prefix, "TestQuery1 err=", err)
								return
							}
							a := rr.(*dns.A)
							fmt.Println(prefix, "!!!! ", serviceName, hostName, ":", port, a.A)
							rrc <- true

						})
					Query(ctx, 0, 0, hostName, dns.TypeAAAA, dns.ClassINET,
						func(err error, flags Flags, ifIndex int, rr dns.RR) {
							if err != nil {
								fmt.Println(prefix, "TestQuery1 err=", err)
								return
							}
							a := rr.(*dns.AAAA)
							fmt.Println(prefix, "!!!! ", serviceName, hostName, ":", port, a.AAAA)
							rrc <- true

						})

				})
		})
	assert.NotNil(t, ctx)
	for ii := 0; ii < 10; ii++ {
		b := <-rrc
		fmt.Println("b=", b)
	}
	println("done...")
}
