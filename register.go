package dnssd

import (
	"context"

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
propagated to the ServiceRegistered callback. flags can be 0 or set to NO_AUTO_RENAME. regType is
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
}
