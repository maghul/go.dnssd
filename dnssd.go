/* The dnssd package is a pure go implementation of DNS Service Discovery
also known as Bonjour(TM).
*/
package dnssd

import (
	"fmt"

	"github.com/miekg/dns"
)

type command struct {
	q *dns.Msg
	r interface{}
}

func (c *command) String() string {
	return fmt.Sprint("command{", c.q, "}")
}

var ns *netserver

func getNetserver() *netserver {
	if ns == nil {
		var err error
		ns, err = newNetserver(nil)
		if err != nil {
			panic("Could not start netserver")
		}
		ns.startReceiving()
		go ns.processing()
	}
	return ns
}

type rrcache struct {
	cache []dns.RR
}

func (rrc *rrcache) add(rr dns.RR) {
	rrc.cache = append(rrc.cache, rr)
}

func (rrc *rrcache) matchQuestion(cmd *command) bool {
	for _, rr := range rrc.cache {
		for _, q := range cmd.q.Question {
			//			fmt.Println("matchQuestion: q=", q, ", rr=", rr)
			if cmd.match(q, rr) {
				return true
			}
		}
	}
	return false
}

func (rrc *rrcache) matchAnswers(cmd *command, sections []dns.RR) {
	for _, rr := range sections {
		for _, q := range cmd.q.Question {
			rrc.add(rr)
			cmd.match(q, rr)
		}
	}
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

func (c *netserver) processing() {
	var cs []*command
	rrc := &rrcache{}

	for {
		select {
		case cmd := <-c.cmdCh:
			//			fmt.Println("COMMAND: ", cmd)

			if !rrc.matchQuestion(cmd) {
				fmt.Println("SEND-QUERY-COMMAND: ", cmd)
				err := c.sendQuery(cmd.q)
				if err != nil {
					respondWithError(cmd.r, err)
				} else {
					cs = append(cs, cmd)
				}
			}

		case msg := <-c.msgCh:
			sections := append(msg.Answer, msg.Ns...)
			sections = append(sections, msg.Extra...)
			for _, cmd := range cs {
				rrc.matchAnswers(cmd, sections)
			}
		}
	}
}

func respond(r interface{}, rr dns.RR) {
	switch r := r.(type) {
	case QueryAnswered:
		r(nil, 0, 0, rr)
	default:
		panic(fmt.Sprint("Dont know what", r, " is"))
	}
}

func respondWithError(r interface{}, err error) {
	switch r := r.(type) {
	case QueryAnswered:
		r(err, 0, 0, nil)
	default:
		panic(fmt.Sprint("Dont know what", r, " is"))
	}
}
