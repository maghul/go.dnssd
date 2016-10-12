package dnssd

import (
	"context"
	"fmt"
	"math/rand"
	"os"

	"github.com/miekg/dns"
)

/*
Called when a service has been registered. Flags are currently unused and always 0, serviceName
is the name registered. It  may have been automatically chosen if the name was blank in the call to
Register. The regType is the same as passed to Register. The domain parameter is the name of the
domain the service was registered to, will be the default domain if domain was blank in the call
to Register
*/
type ServiceRegistered func(flags int, serviceName, regType, domain string)

/*
Add an additional record to the service registration. This will be registered
using the same context as the service was registered with*/
type AddRecord func(flags int, rr dns.RR)

/*
Register a service. ctx is the context and is used to cancel a registration.
ifIndex is the interface to publish the service on, 0 for all interfaces and -1 for localhost.
serviceName is the name of the service. if left blank the computer name will be used and
propagated to the ServiceRegistered callback. flags can be 0 or set to NoAutoRename. regType is
the service registration type.
domain is the domain of the service, usually left blank.
host is the name of the server being registered. usually left blank for the local machine name.
port is the port of the service.
txtRecord is the content of the TXT record.
listener is a closure that will be called when the service has been registered.
errc is a closure that will be called if there was an error registering the service.
The return from the func is an AddRecord func that can be called to add additional records
that will be associated with this service.
*/
func Register(ctx context.Context, flags Flags, ifIndex int, serviceName, regType, domain, host string, port uint16, txt []string,
	listener ServiceRegistered, errc ErrCallback) AddRecord {

	if flags != None && flags != NoAutoRename {
		errc(errBadFlags)
		return nil
	}

	if domain == "" {
		domain = getOwnDomainname()
	}

	if host == "" {
		h, err := os.Hostname()
		if err != nil {
			errc(err)
			return nil
		}
		host = h
	}
	if serviceName == "" {
		serviceName = getManufacturedServiceName(host)
	}

	fullRegType := fmt.Sprintf("%s.%s.", regType, domain)
	fullName := ConstructFullName(serviceName, regType, domain)
	target := fmt.Sprintf("%s.%s.", host, domain)

	recordsRegistered := uint8(0)
	registrar := CreateRecordRegistrar(func(record dns.RR, flags int) {
		fmt.Println("REGISTER: rr=", record)
		recordsRegistered = recordsRegistered | flag(record)
		if recordsRegistered == 7 {
			listener(0, serviceName, regType, domain)
		}
	}, func(err error) {
		errc(err)
	})

	ptrRR := new(dns.PTR)
	ptrRR.Hdr = dns.RR_Header{Name: fullRegType, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: 3200} // TODO: TTL correct?
	ptrRR.Ptr = fullName
	fmt.Println("ptrRR=", ptrRR)
	registrar(ctx, Shared, ifIndex, ptrRR)

	srvRR := new(dns.SRV)
	srvRR.Hdr = dns.RR_Header{Name: fullName, Rrtype: dns.TypeSRV, Class: dns.ClassINET, Ttl: 20} // TODO: TTL correct?
	srvRR.Target = target
	srvRR.Port = port
	srvRR.Priority = 0 // TODO: correct?
	srvRR.Weight = 0   // TODO: correct?
	fmt.Println("srvRR=", srvRR)
	registrar(ctx, Unique, ifIndex, srvRR)

	if txt != nil {
		txtRR := new(dns.TXT)
		txtRR.Hdr = dns.RR_Header{Name: fullName, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 3200} // TODO: TTL correct?
		txtRR.Txt = txt
		fmt.Println("txtRR=", txtRR)
		registrar(ctx, Unique, ifIndex, txtRR)
	}

	return func(flags int, rr dns.RR) {
		header := rr.Header()
		if header.Name == "" {
			header.Name = serviceName
		} else {
			if header.Name != serviceName {
				panic(fmt.Sprint("AddRecord header name '", header.Name, "' != serviceName '", serviceName, "'"))
			}
		}
		fmt.Println("AddRecord=", rr)
	}
}

func getManufacturedServiceName(hostname string) string {
	// TODO: make a bit better.
	return fmt.Sprintf("%s%x%x", hostname, rand.Uint32(), rand.Uint32())
}

func flag(rr dns.RR) uint8 {
	switch rr.Header().Rrtype {
	case dns.TypePTR:
		return 1
	case dns.TypeSRV:
		return 2
	case dns.TypeTXT:
		return 4
	}
	return 0
}
