package dnssd

import (
	"context"
)

/*
 alled to report discovered domains. Flags can be dnssd.MoreComing and
dnssd.RecordAdded. The flag dnssd.RecordAdded should be checked to
determine if a domain was found or lost.
*/
type DomainUpdate func(flags Flags, ifIndex int, domain string)

/*
Asynchronously enumerate domains available for browsing and registration. Flags
can be dnssd.BrowseDomains or dnssd.RegistrationDomains. ifIndex is ignored
currently but may be used in the future if there are interface specific domains.
*/
func EnumerateDomains(ctx context.Context, flags Flags, ifIndex int, listener DomainUpdate, errc ErrCallback) {
	listener(RecordAdded, ifIndex, getOwnDomainname())
}

func getOwnDomainname() string {
	return "local"
}
