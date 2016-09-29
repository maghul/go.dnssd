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

func (rrc *rrcache) matchQuestion(q *dns.Question) []dns.RR {
	var matchedAnswers []dns.RR
	for _, rr := range rrc.cache {
		if matchQuestionAndRR(q, rr) {
			matchedAnswers = append(matchedAnswers, rr)
		}
	}
	return matchedAnswers
}

func (rrc *rrcache) matchQuery(ifIndex int, cq *question) bool {
	for _, rr := range rrc.cache {
		// TODO:Look at TTL and expire things
		//      Send ServiceUpdate if a PTR record expires.
		if cq.match(rr) {
			return true
		}
	}
	return false
}
