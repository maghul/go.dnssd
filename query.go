package dnssd

import (
	"context"

	"github.com/miekg/dns"
)

const MORE_COMING = 1
const RECORD_ADDED = 8

/* This is called when a query has been resolved.
flags may be MORE_COMING or RECORD_ADDED.
ifIndex is the interface the query was responden on.
rr is a resource record matching the query.
*/
type QueryAnswered func(flags Flags, ifIndex int, rr dns.RR)

/*
Query an arbitrary record. ctx is the query context and can be used to cancel or timeout a query.
flags - Possible values are: MORE_COMING.
ifIndex - If non-zero, specifies the interface on which to issue the query (the index for a given interface is determined via the if_nametoindex() family of calls.) Passing 0 causes the name to be queried for on all interfaces. Passing -1 causes the name to be queried for only on the local host.
question - The question to query for.
response - This closure will get called when the query completes.
errc - This closure will be called when a query has an error.
*/
func Query(ctx context.Context, flags Flags, ifIndex int, serviceName string, rrtype, rrclass uint16, response QueryAnswered, errc ErrCallback) {
	ns = getNetserver()

	// send the query
	m := new(dns.Msg)
	m.Question = []dns.Question{
		dns.Question{serviceName, rrtype, rrclass},
	}
	ns.cmdCh <- &command{m, response, errc}
}

// Instruct the daemon to verify the validity of a resource record that appears to be out of date.
func ReconfirmRecord(flags Flags, ifIndex int, rr *dns.RR) {
	panic("NYI")
}
