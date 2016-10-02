package dnssd

import (
	"fmt"

	"github.com/miekg/dns"
)

// rrcache contains all received RR records solicited or not.
type answers struct {
	cache []*answer
}

type answer struct {
	ifIndex int
	rr      dns.RR
}

func (aa *answers) add(a *answer) bool {
	for _, a2 := range aa.cache {
		// TODO: conflicting entries?
		if matchRRs(a.rr, a2.rr) {
			return false
		}
	}
	aa.cache = append(aa.cache, a)
	return true
}

func (aa *answers) matchQuestion(q *dns.Question) []*answer {
	var matchedAnswers []*answer
	for _, a := range aa.cache {
		if matchQuestionAndRR(q, a.rr) {
			matchedAnswers = append(matchedAnswers, a)
		}
	}
	return matchedAnswers
}

func (aa *answers) matchQuery(ifIndex int, cq *question) bool {
	for _, rr := range aa.cache {
		// TODO:Look at TTL and expire things
		//      Send ServiceUpdate if a PTR record expires.
		if cq.match(rr) {
			return true
		}
	}
	return false
}

func rrs(aa []*answer) []dns.RR {
	r := make([]dns.RR, len(aa))
	for ii, a := range aa {
		r[ii] = a.rr
	}
	return r
}

func (a *answer) String() string {
	return fmt.Sprint("Answer{rr=", a.rr, "}")
}
