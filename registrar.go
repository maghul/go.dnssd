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
}
