package dnssd

import (
	"context"
	"fmt"

	"github.com/miekg/dns"
)

type command struct {
	ctx  context.Context
	q    *dns.Msg
	r    interface{}
	errc ErrCallback
}

func (c *command) String() string {
	return fmt.Sprint("command{", c.q, "}")
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

func respond(r interface{}, rr dns.RR) {
	switch r := r.(type) {
	case QueryAnswered:
		r(0, 0, rr)
	default:
		panic(fmt.Sprint("Dont know what", r, " is"))
	}
}
