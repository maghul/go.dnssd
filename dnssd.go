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

func (c *netserver) processing() {
	var cs []*command
	for {
		select {
		case cmd := <-c.cmdCh:
			err := c.sendQuery(cmd.q)
			if err != nil {
				respondWithError(cmd.r, err)
			} else {
				cs = append(cs, cmd)
			}
		case msg := <-c.msgCh:
			fmt.Println("MSG:", msg)
			sections := append(msg.Answer, msg.Ns...)
			sections = append(sections, msg.Extra...)
			for _, c := range cs {
				for _, answer := range sections {
					for _, q := range c.q.Question {
						if q.Qtype == answer.Header().Rrtype {
							if q.Name == answer.Header().Name {
								respond(c.r, answer)
							}
						}
					}
				}
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
