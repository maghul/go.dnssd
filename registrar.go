package dnssd

import (
	"context"
	"errors"
	"time"

	"github.com/miekg/dns"
)

/*
Callback when a record has been registered.
record is the newly registered record.
flags is currently unused and will be set to 0.
*/
type RecordRegistered func(record dns.RR, flags Flags)

/*
Registrar function, will register dns.RR records
flags may be dnssd.SHARED or dnssd.UNIQUE.
ifIndex The index of interface to register the record to. If 0 it will be registered on all interfaces.
record is the dns.RR record to register.
*/
type RegisterRecord func(ctx context.Context, flags Flags, ifIndex int, record dns.RR)

/*
Create a DNSSDRecordRegistrar allowing efficient registration of multiple individual records.
listener will be called when a record has been registered. errc will be called
if there is an error with the registrar.
The RegisterRecord closure returned is used to record new register entries.
*/
func CreateRecordRegistrar(listener RecordRegistered, errc ErrCallback) RegisterRecord {
	ds := getDnssd()
	return func(ctx context.Context, flags Flags, ifIndex int, record dns.RR) {
		if !flags.required(Unique | Shared) {
			errc(errBadFlags)
			return
		}
		go func() {
			rrChan := make(chan dns.RR, 2)
			for count := 3; count > 0; count-- {
				ctxc, _ := context.WithTimeout(ctx, 250*time.Millisecond)
				Query(ctxc, flags, ifIndex, record.Header().Name, record.Header().Rrtype, record.Header().Class, func(flags Flags, ifIndex int, rr dns.RR) {
					rrChan <- rr
				}, errc)
				select {
				case <-ctxc.Done():
					// Timeout of request
				case rr := <-rrChan:
					// We have received a response on the record we wish to publish.
					if rr.String() == record.String() {
						listener(rr, 0)
					} else {
						err := errors.New("Could not publish record, it is in use")
						errc(err)
					}
					return
				}
			}

			publishTime := 20
			// Publish with exponential backoff: ", name, ": 0, 20, 40, 80, 160, 320, 640, 1280
			listener(record, 0)
			for count := 8; count > 0; count-- {
				ds.cmdCh <- makeCommand(ctx, nil, record, listener, errc)
				time.Sleep(time.Duration(publishTime) * time.Millisecond)
				publishTime *= 2
			}
		}()
	}
}
