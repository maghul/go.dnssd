package dnssd

import (
	"context"
)

type callback struct {
	ctx  context.Context
	call QueryAnswered
}

func (r *callback) respond(ifIndex int, a *answer) {
	r.call(0, ifIndex, a.rr)
}

func (cb *callback) isValid() bool {
	select {
	case <-cb.ctx.Done():
		return false
	default:
	}
	return true
}
