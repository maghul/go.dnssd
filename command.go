package dnssd

import (
	"context"
	"fmt"

	"github.com/miekg/dns"
)

type command struct {
	ctx       context.Context
	q         *dns.Msg // For queries
	rr        dns.RR   // For registering records.
	r         interface{}
	errc      ErrCallback
	completed bool
	serial    int
}

var commandSerial int = 0

func makeCommand(ctx context.Context, q *dns.Msg, rr dns.RR, r interface{}, errc ErrCallback) *command {
	commandSerial++
	return &command{ctx, q, rr, r, errc, false, commandSerial}
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
	fmt.Println("ISVALID: ", cmd.completed)
	return !cmd.completed
}

func (cmd *command) match(q dns.Question, answer dns.RR) bool {
	if q.Qtype == answer.Header().Rrtype {
		if q.Name == answer.Header().Name {
			fmt.Println("MATCH: ", cmd)
			respond(cmd.r, answer)
			cmd.completed = true
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
