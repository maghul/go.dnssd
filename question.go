package dnssd

import (
	"fmt"

	"github.com/miekg/dns"
)

// A DNSSD question is a registered question which is used to listen
// to incoming responses and send them to registered callbacks.
// There is one DNSSD question for each dns.Question which may have
// many attached callbacks. The question is valid for all interfaces
// whereas a callback may be filtered for interface.
type question struct {
	q  *dns.Question
	cb []*callback
}

// A collection of questions.
type questions struct {
	qmap []*question
}

func (cq *question) String() string {
	return fmt.Sprint("Question{q=", cq.q, "}")
}

func (cq *question) match(a *answer) bool {
	q := cq.q
	rr := a.rr
	if q.Qtype == rr.Header().Rrtype {
		if q.Name == rr.Header().Name {
			cq.respond(a)
			return true
		}
	}
	return false
}

// Attach a new callback
func (cq *question) attach(cb *callback) {
	for _, cba := range cq.cb {
		if cba == cb {
			return
		}
	}
	cq.cb = append(cq.cb, cb)
}

// Detach a callback.
func (cq *question) detach(cb *callback) {
	jj := 0
	for _, cba := range cq.cb {
		if cba != cb {
			cq.cb[jj] = cba
			jj++
		}
	}
	cq.cb = cq.cb[0:jj]
}

// Send an RR to all attached callbacks. If a callback
// returns false it will be removed as callback.
func (cq *question) respond(a *answer) {
	jj := 0
	for _, cba := range cq.cb {
		if cba.respond(a) {
			cq.cb[jj] = cba
			jj++
		}
	}
	cq.cb = cq.cb[0:jj]
}

// Check on callbacks and return true if any callback
// is still active.
func (cq *question) isActive() bool {
	jj := 0
	for _, cba := range cq.cb {
		if !cba.isClosed() {
			cq.cb[jj] = cba
			jj++
		}
	}
	cq.cb = cq.cb[0:jj]
	return jj > 0
}

func (qs *questions) makeQuestion(q *dns.Question) *question {
	cq := &question{q, nil}
	qs.qmap = append(qs.qmap, cq)
	return cq
}

// Find the DNSSD question registered from the dns.Question.
func (qs *questions) findQuestion(q *dns.Question) *question {
	for _, cq := range qs.qmap {
		if matchQuestions(cq.q, q) {
			return cq
		}
	}
	return nil
}

func (qs *questions) findQuestionFromRR(rr dns.RR) *question {
	for _, cq := range qs.qmap {
		if matchQuestionAndRR(cq.q, rr) {
			return cq
		}
	}
	return nil
}

func questionFromRRHeader(rrh *dns.RR_Header) *dns.Question {
	return &dns.Question{Name: rrh.Name, Qtype: rrh.Rrtype, Qclass: rrh.Class}
}
