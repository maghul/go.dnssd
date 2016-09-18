package dnssd

import (
	"github.com/miekg/dns"
)

// rrcache contains all received RR records solicited or not.
type rrcache struct {
	cache []dns.RR
}

func (rrc *rrcache) add(rr dns.RR) {
	rrc.cache = append(rrc.cache, rr)
}

func (rrc *rrcache) matchQuestion(cmd *command) bool {
	for _, rr := range rrc.cache {
		// TODO:Look at TTL and expire things
		//      Send ServiceUpdate if a PTR record expires.
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
		// TODO: Check these RR for TTL=0 and expire RRs
		//       send a ServiceUpdate with found=false for expired PTR records
		for _, q := range cmd.q.Question {
			rrc.add(rr)
			cmd.match(q, rr)
		}
	}
}
