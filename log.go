package dnssd

import (
	"github.com/maghul/go.slf"
)

var dnssdlog = getLogger("dnssd.dnssd", "Internal DNSSD Logging")
var qlog = getLogger("dnssd.queries", "Query Logging")
var netlog = getLogger("dnssd.net", "Networking")
var testlog = getLogger("dnssd.test", "Test Logging")
var parentlog = slf.GetLogger("dnssd")

func init() {
	r := slf.GetLogger("dnssd")
	r.SetDescription("DNSSD Parent Logging")
	r.SetLevel(slf.Parent)
}

func getLogger(name, description string) *slf.Logger {
	l := slf.GetLogger(name)
	r := slf.GetLogger("dnssd")
	l.SetParent(r)
	l.SetDescription(description)
	return l
}
