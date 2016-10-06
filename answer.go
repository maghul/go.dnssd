package dnssd

import (
	"context"
	"fmt"

	"github.com/miekg/dns"
)

// rrcache contains all received RR records solicited or not.
type answers struct {
	cache []*answer
}

type answer struct {
	ctx     context.Context // Only used by published RR entries.
	ifIndex int
	rr      dns.RR
}

func matchAnswers(a1, a2 *answer) bool {
	return a1.ifIndex == a2.ifIndex && matchRRs(a1.rr, a2.rr)
}

func makeAnswers() *answers {
	return &answers{}
}

func (aa *answers) add(a *answer) bool {
	for _, a2 := range aa.cache {
		// TODO: conflicting entries?
		if matchAnswers(a, a2) {
			return false
		}
	}
	aa.cache = append(aa.cache, a)
	return true
}

func (aa *answers) size() int {
	return len(aa.cache)
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

func (a *answer) String() string {
	return fmt.Sprint("Answer{rr=", a.rr, "}")
}

func (aa *answers) findClosedAnswers() []*answer {
	var closedAnswers []*answer
	ii := 0
	for _, a := range aa.cache {
		if !a.isClosed() {
			aa.cache[ii] = a
			ii++
		} else {
			closedAnswers = append(closedAnswers, a)

		}
	}
	aa.cache = aa.cache[0:ii]
	return closedAnswers
}

func (a *answer) isClosed() bool {
	return contextIsClosed(a.ctx)
}
