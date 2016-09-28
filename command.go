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
	r    interface{}
	errc ErrCallback

	keep      bool // True if the command should be running
	completed bool
	serial    int
}

var commandSerial int = 0

func makeCommand(ctx context.Context, q *dns.Msg, rr dns.RR, r interface{}, errc ErrCallback) *command {
	commandSerial++
	return &command{ctx, q, rr, r, errc, false, false, commandSerial}
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
	return !cmd.completed
}

func (cmd *command) match(q dns.Question, answer dns.RR) bool {
	if q.Qtype == answer.Header().Rrtype {
		if q.Name == answer.Header().Name {
			respond(cmd.r, answer)
			cmd.completed = !cmd.keep // if keep is false set command as completed.
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
