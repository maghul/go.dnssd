package dnssd

import (
	"context"

	"github.com/miekg/dns"
)

type callback struct {
	ctx  context.Context
	call QueryAnswered
}

func (r *callback) respond(rr dns.RR) {
	r.call(0, 0, rr)
}

func (cb *callback) isValid() bool {
	select {
	case <-cb.ctx.Done():
		return false
	default:
	}
	return true
}
