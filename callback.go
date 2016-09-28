package dnssd

import (
	"fmt"

	"github.com/miekg/dns"
)

type callback interface{}

func respond(r callback, rr dns.RR) {
	switch r := r.(type) {
	case QueryAnswered:
		r(0, 0, rr)
	default:
		panic(fmt.Sprint("Dont know what", r, " is"))
	}
}
