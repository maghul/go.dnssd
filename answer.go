package dnssd

import (
	"context"
	"fmt"
	"time"

	"github.com/miekg/dns"
)

// rrcache contains all received RR records solicited or not.
type answers struct {
	cache []*answer
}

type answer struct {
	ctx       context.Context // Only used by published RR entries.
	added     time.Time       // Used to count expiry from TTL
	ttl       time.Duration
	flags     Flags
	requeried int
	ifIndex   int
	rr        dns.RR
}

func matchAnswers(a1, a2 *answer) bool {
	return a1.ifIndex == a2.ifIndex && matchRRs(a1.rr, a2.rr)
}

func makeAnswers() *answers {
	return &answers{}
}

func (aa *answers) addRecord(ctx context.Context, flags Flags, ifIndex int, rr dns.RR) (*answer, bool) {
	ttl := time.Second * time.Duration(rr.Header().Ttl)
	ttl += randomDuration(ttl, 2)
	a := &answer{ctx, time.Now(), ttl, flags, 0, ifIndex, rr}
	return a, aa.add(a)
}

func (aa *answers) add(a *answer) bool {
	for ii, a2 := range aa.cache {
		// TODO: conflicting entries?
		if matchAnswers(a, a2) {
			aa.cache[ii] = a
			return false
		}
	}
	a.added = time.Now()
	a.requeried = 0
	aa.cache = append(aa.cache, a)
	return true
}

func (aa *answers) size() int {
	return len(aa.cache)
}

func (aa *answers) iterateAnswersForQuestion(q *dns.Question, f func(a *answer)) {
	for _, a := range aa.cache {
		if matchQuestionAndRR(q, a.rr) {
			f(a)
		}
	}
}

func (aa *answers) matchQuestion(q *dns.Question) []*answer {
	var matchedAnswers []*answer
	aa.iterateAnswersForQuestion(q, func(a *answer) {
		if matchQuestionAndRR(q, a.rr) {
			matchedAnswers = append(matchedAnswers, a)
		}
	})
	return matchedAnswers
}

func (aa *answers) findAnswerFromRR(rr dns.RR) (*answer, bool) {
	for _, a := range aa.cache {
		if matchRRHeader(rr.Header(), a.rr.Header()) {
			return a, true
		}
	}
	return nil, false
}

func (a *answer) String() string {
	s := ""
	if a.flags&Shared != 0 {
		s = ", Shared"
	}
	if a.flags&Unique != 0 {
		s = ", Unique"
	}
	return fmt.Sprint("Answer{if=", a.ifIndex, ", added=", a.added, s, ", rr=", a.rr, "}")
}

// Look through the cache and requery answers which are
// expiring.
// Return a time for next record event.
func (aa *answers) findOldAnswers(requery func(a *answer), remove func(a *answer)) time.Time {
	now := time.Now()
	var nextTime time.Time
	ii := 0
	for _, a := range aa.cache {

		if a.isClosed() {
			remove(a)
			continue
		}
		rt, doRequery := a.getNextCheckTime()

		if now.After(rt) {
			if doRequery {
				a.requeried++
				requery(a)
			} else {
				remove(a)
				continue
			}
		} else {
			if nextTime.IsZero() || nextTime.After(rt) {
				nextTime = rt
			}
		}
		aa.cache[ii] = a
		ii++

	}
	aa.cache = aa.cache[0:ii]
	//	aa.dump("findOldAnswers")
	return nextTime
}

func (a *answer) getNextCheckTime() (time.Time, bool) {
	rt := a.added
	if a.flags&Unique != 0 {
		// Unique records should only be requried once
		switch a.requeried {
		case 0: // At 80% of TTL
			rt = rt.Add((a.ttl * 80) / 100)
		case 1: // At 80% of TTL
			rt = rt.Add(a.ttl)

		}
		return rt, a.requeried < 1
	} else {
		switch a.requeried {
		case 0: // At 80% of TTL
			rt = rt.Add((a.ttl * 80) / 100)
		case 1: // At 80% of TTL
			rt = rt.Add((a.ttl * 85) / 100)
		case 2:
			rt = rt.Add((a.ttl * 90) / 100)
		case 3:
			rt = rt.Add((a.ttl * 95) / 100)
		case 4:
			rt = rt.Add(a.ttl)

		}
		return rt, a.requeried < 4
	}
}

func (a *answer) isClosed() bool {
	return contextIsClosed(a.ctx)
}
