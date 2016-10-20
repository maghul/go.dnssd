package dnssd

import (
	"context"
)

// Called to report discovered domains.
type DomainUpdate func(err error, flags, ifIndex int, domain string)

// Asynchronously enumerate domains available for browsing and registration.
func EnumerateDomains(ctx context.Context, flags, ifIndex int, listener DomainUpdate) {
	panic("NYI")
}

func getOwnDomainname() string {
	return "local" // TODO: fix?
}
