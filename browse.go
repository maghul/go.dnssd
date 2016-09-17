package dnssd

import (
	"context"
	"fmt"
	"strings"

	"github.com/miekg/dns"
)

/*
A closure that is called when a service has been updated.
found is true if a service has been discoverd, false if it has been removed.
flags may be dnssd.MORE_COMING.
ifIndex the index of the interface where the service was discovered. It should be passed to dnssd.Resolve.
serviceName The name of the service.
regType The registration type of the service, same as regType used in call to Browse.
domain The domain the service was discovered on.
*/
type ServiceUpdate func(found bool, flags Flags, ifIndex int, serviceName, regType, domain string)

/*
Browse a service. ctx is the context used to cancel a browse. flags are currently unused. ifIndex is
used to indicate which interface the service should be browsed on. regType is the service type (e g _http._tcp)
domain is the domain to browse for the service. If domain is set blank the default domain will be used. response
is a closure called when service data has been updated. errc is called when an error has occured.
*/
func Browse(ctx context.Context, flags Flags, ifIndex int, regType, domain string, response ServiceUpdate, errc ErrCallback) {

	name := fmt.Sprint(regType, ".", domain, ".")
	Query(ctx, 0, 0, name, dns.TypePTR, dns.ClassINET,
		func(err error, flags Flags, ifIndex int, rr dns.RR) {
			if err != nil {
				response(err, false, 0, 0, "", "", "")
			} else {
				ptr := rr.(*dns.PTR)
				split := strings.SplitN(ptr.Ptr, ".", 4)
				response(nil, true, 0, 0, split[0], fmt.Sprint(split[1], ".", split[2]), trimTrailingDot(split[3]))
			}
		})

}

func trimTrailingDot(s string) string {
	return strings.TrimRight(s, ".")
}
