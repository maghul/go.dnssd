package dnssd

import (
	"bytes"
)

type Flags uint32

const (
	/*
		No flags (makes it clearer to use this symbolic constant rather than using a 0).
	*/
	None Flags = 0

	/*
		MoreComing indicates to a callback that at least one more result is queued and will be delivered
		following immediately after this one. Applications should not update their UI to display browse
		results when the MoreComing flag is set, because this would result in a great deal of ugly
		flickering on the screen. Applications should instead wait until until MoreComing is not set
		(i.e. "Finished", for now), and then update their UI. When MoreComing is not set (i.e.
		"Finished") that doesn't mean there will be no more answers EVER, just that there are no more
		answers immediately available right now at this instant. If more answers become available in
		the future they will be delivered as usual.
	*/
	MoreComing Flags = 1 << iota

	/*
		Service or domain has been added during browsing/querying or domain enumeration.
	*/
	Add Flags = 1 << iota

	/*
		Default domain has been added during domain enumeration.
	*/
	Default Flags = 1 << iota

	/*
		Auto renaming should not be performed. Only valid if a name is explicitly specified when
		registering a service (i.e. the default name is not used).
	*/
	NoAutoRename Flags = 1 << iota

	/*
		Shared flag for registering individual records on a connected DNSServiceRef. Indicates that there
		may be multiple records with this name on the network (e.g. PTR records).
	*/
	Shared Flags = 1 << iota

	/*
		Shared flag for registering individual records on a connected DNSServiceRef. Indicates that the
		record's name is to be unique on the network (e.g. SRV records).
	*/
	Unique Flags = 1 << iota

	/*
		Enumerates domains recommended for browsing.
	*/
	BrowseDomains Flags = 1 << iota

	/*
		Enumerates domains recommended for registration.
	*/
	RegistrationDomains Flags = 1 << iota

	/*
		True if a record was added. False it was lost.
	*/
	RecordAdded Flags = 1 << iota
)

func (f Flags) appendString(b *bytes.Buffer, mask Flags, name string) {
	if f&mask != 0 {
		if b.Len() > 0 {
			b.WriteString(" | ")
		}
		b.WriteString(name)
	}
}

func (f Flags) String() string {
	b := bytes.NewBufferString("")
	f.appendString(b, MoreComing, "MoreComing")
	f.appendString(b, Add, "Add")
	f.appendString(b, Default, "Default")
	f.appendString(b, NoAutoRename, "NoAutoRename")
	f.appendString(b, Shared, "Shared")
	f.appendString(b, Unique, "Unique")
	f.appendString(b, BrowseDomains, "BrowseDomains")
	f.appendString(b, RegistrationDomains, "RegistrationDomains")
	f.appendString(b, RecordAdded, "RecordAdded")
	if b.Len() == 0 {
		return "None"
	}
	return b.String()
}

func (f Flags) required(mask Flags) bool {
	return f&mask != 0
}
