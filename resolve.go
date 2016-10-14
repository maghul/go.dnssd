package dnssd

import (
	"context"

	"github.com/miekg/dns"
)

/*
This closure is called when a service has been resolved. flags are currently
unused and set to 0. fullName is the full name of the service, e.g. <servicename>.<protocol>.<domain>
The name us unpacked into a latin1/iso8859-1 string which needs to be used as argument
to subsequent calls to Query. In order to see a proper UTF8 string of the full name
you nedd to call RepackToUTF8. The parameter hostName is the name of the host and
can be used to Query for IP-addresses using dns.TypeA and dns.TypeAAAA queries.
The parameter port is the port number of the service.
The TXT data is returned in txt as a string array. The form of the entries
in each string is "<key>=<value>"
*/
type ServiceResolved func(flags Flags, ifIndex int, fullName, hostName string, port uint16, txt []string)

/*
Resolve a service to SRV host name and port, as well as a TXT record. When the resolve has return the
wanted resolve the resolve should be stopped by calling the cancel function. This is
a convencience function for resolving a single SRV and TXT record. More complex
resolutions should use Query.
ctx is the context for the resolve, should be cancellable.
flags are unused currently, ifIndex is the index of the interface on which to resolve
the service. Should normally be the same returned by a browse, if 0 it will resolve
on all interfaces. The serviceName is the name of the service as returned by Browse.
regType is the registration type (for example _raop._tcp)
domain is the domain of the service, normally the domain returned by Browse should be used.
If domain is blank it will be replaced with the local domain
response is a function that will be called when a service has been resolved. May be called
several times. errc is an error callback.
*/
func Resolve(ctx context.Context, flags Flags, ifIndex int, serviceName, regType, domain string, response ServiceResolved, errc ErrCallback) {

