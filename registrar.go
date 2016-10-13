package dnssd

import (
	"context"
	"time"

	"github.com/miekg/dns"
)

/*
Callback when a record has been registered.
record is the newly registered record.
flags is currently unused and will be set to 0.
*/
type RecordRegistered func(record dns.RR, flags int)

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
			if flags&Unique != 0 {
				// Only probe if the record is supposed to be unique
				rrChan := make(chan dns.RR, 2)
				question := questionFromRRHeader(record.Header())
				response := func(flags Flags, ifIndex int, rr dns.RR) {
					rrChan <- rr
				}
				for count := 3; count > 0; count-- {
					ctxc, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
					cb := makeCallback("probe", record, ctxc, ifIndex, response)
					ds.cmdCh <- func() {
						dnssdlog("DNSSD PROBE=", question)
						ds.runProbe(ifIndex, question, cb)
					}

					select {
					case <-ctxc.Done():
						// Timeout of request
						cancel() // Should already be cancelled actually, govet -1!
					case rr := <-rrChan:
						// We have received a response on the record we wish to publish.
						cancel()
						if rr.String() == record.String() {
							// TODO: create a new name or report an error.
						}
						return
					}
				}
			}

			publishTime := 20
			// Publish with exponential backoff: ", name, ": 0, 20, 40, 80, 160, 320, 640, 1280
			dnssdlog("DNSSD PUBLISH=", record)
			listener(record, 0)
			for count := 8; count > 0; count-- {
				ds.cmdCh <- func() {
					ds.publish(ctx, flags, ifIndex, record)
				}
				time.Sleep(time.Duration(publishTime) * time.Millisecond)
				publishTime *= 2
			}
		}()
	}
}
