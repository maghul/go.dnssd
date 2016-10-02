package dnssd

import (
	"fmt"

	"github.com/miekg/dns"
)

type question struct {
	q  *dns.Question
	cb []*callback
}

type questions []*question

func makeQuestion(q *dns.Question) *question {
	return &question{q, nil}
}

func (cq *question) String() string {
	return fmt.Sprint("Question{q=", cq.q, "}")
}

func (cq *question) match(a *answer) bool {
	q := cq.q
	rr := a.rr
	if q.Qtype == rr.Header().Rrtype {
		if q.Name == rr.Header().Name {
			ifIndex := 0 // TODO: should be part of answer record
			cq.respond(ifIndex, a)
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

// Send an RR to all attached callbacks
func (cq *question) respond(ifIndex int, a *answer) {
	jj := 0
	dnssdlog("QUESTION respond cq=", cq, ", callbacks=", len(cq.cb), ", answer=", a)
	for _, cba := range cq.cb {
		if cba.isValid() {
			cba.respond(ifIndex, a)
			cq.cb[jj] = cba
			jj++
		} else {
			dnssdlog("QUESTION   Removing callback ", cba)
		}
	}
	cq.cb = cq.cb[0:jj]
}

func (cqs *questions) findQuestion(q *dns.Question) *question {
	for _, cq := range *cqs {
		if matchQuestions(cq.q, q) {
			return cq
		}
	}
	return nil
}
