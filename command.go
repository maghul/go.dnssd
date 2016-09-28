package dnssd

import (
	"context"
	"fmt"

	"github.com/miekg/dns"
)

type command struct {
	ctx  context.Context
	q    *dns.Msg // For queries
	rr   dns.RR   // For registering records.
	r    callback // This is a callback.
	errc ErrCallback

	//	completed bool
	serial int
}

var commandSerial int = 0

func makeCommand(ctx context.Context, q *dns.Msg, rr dns.RR, r callback, errc ErrCallback) *command {
	commandSerial++
	return &command{ctx, q, rr, r, errc, commandSerial}
}
func (c *command) String() string {
	return fmt.Sprint("command{#", c.serial, ", q=", c.q, "}")
}

func (cmd *command) isValid() bool {
	select {
	case <-cmd.ctx.Done():
		return false
	default:
	}
	return true
}

func (cmd *command) match(q dns.Question, answer dns.RR) bool {
	if q.Qtype == answer.Header().Rrtype {
		if q.Name == answer.Header().Name {
			respond(cmd.r, answer)
			return true
		}
	}
	return false
}
