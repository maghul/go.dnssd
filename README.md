# dnssd

This is a LPGL license pure Go library for DNS Service Discovery.

It aims to be a fully compliant implementation of RFC6762 without
any dependencies except pure go packages.

Dependencies
------------
	"context/Context"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"

License
-------

The dnssd package is licensed under the LGPL with an exception that allows it to be linked statically. Please see the LICENSE file for details.

Usage
-----
The library defines the funcs Browse, Resolve, Query, Register which are
used in most normal operations. Each take a context.Context which can be
used to cancel the operation. No channels are exposed by the API and
all replies are performed using callbacks in a separate go-routine. The callback
go-routines are reused and created as needed. If callabacks are handled quickly
there will be only one go-routine.

For more specific needs the API exposes CreateRecordRegistrar for registering
arbitrary entries.

Please see godoc via godoc github.com/ermahu/dnssd` for further information.

Examples
--------

Browsing for a Service

	ctx, cancel := context.WithCancel(context.Background())
	dnssd.Browse(ctx, 0, 0, "_raop._tcp", "local",
		func(found bool, flags, ifIndex int, serviceName, regType, domain string) {
			fmt.Println("serviceName=", serviceName)
		}, func(err error) {
			fmt.Println("Error browsing: ", err)
			cancel()
		})
		
Resolving a serviceName

	ctx, cancel := context.WithCancel(context.Background())
	dnssd.Resolve(ctx, 0, 0, "rafael", "_airplay._tcp", "local",
		func(flags, ifIndex int, fullName, hostName string, port uint16, txt []string) {
			fmt.Println("Resolved: name=", fullName, ", host=", hostName, ", port=", port, ", text=", txt)
		}, func(err error) {
			fmt.Println("Error resolving: ", err)
			cancel()
		})

Registering a service

	name := "testService"
	regType := "_test._tcp"
	port := 4711
	txt := []string{"test=one", "check=two"}
	ctx, cancel := context.WithCancel(context.Background())
	dnssd.Register(ctx, 0, 3, name, regType, "", "", port, txt, 
	    func(flags int, serviceName, regType, domain string) {
			fmt.Println("Register: serviceName=", serviceName, ", regType=", regType, ",domain=", domain)
	}, func(err error) {
		fmt.Println("Error registering: ", err)
		cancel()
	})
	
Querying for an address

	ctx := context.Background()
	hostname := "myserver.local."
	dnssd.Query(ctx, 0, 0, &dns.Question{myserver, dns.TypeA, dns.ClassINET},
		func(flags, ifIndex int, rr dns.RR) {
			fmt.Println("Queried address is ",rr)
		}, func(err error) {
			fmt.Println("Error querying: ", err)
		})
		