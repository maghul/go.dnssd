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
	added     time.Time
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
func (aa *answers) addRecord(ifIndex int, rr dns.RR) (*answer, bool) {
	a := &answer{nil, time.Now(), 0, ifIndex, rr}
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
	return fmt.Sprint("Answer{if=", a.ifIndex, ", added=", a.added, ", rr=", a.rr, "}")
}

// Look through the cache and requery answers which are
// expiring.
func (aa *answers) findOldAnswers(requery func(a *answer), remove func(a *answer)) {
	now := time.Now()
	ii := 0
	for _, a := range aa.cache {

		switch a.checkRecord(now) {
		case -1:
			remove(a)
			continue
		case 1:
			a.requeried++
			requery(a)
			fallthrough
		case 0:
			aa.cache[ii] = a
			ii++
		}
	}
	aa.cache = aa.cache[0:ii]
	//	aa.dump("findOldAnswers")
}

// Check the record:
// 1 - means requery/republish
// 0 - means OK, no changes
// -1 - means remove
func (a *answer) checkRecord(now time.Time) int {
	if a.isClosed() {
		return -1
	}
	rt := a.added
	ttl := time.Second * time.Duration(a.rr.Header().Ttl)
	switch a.requeried {
	case 0: // At 50% of TTL
		rt = rt.Add(ttl / 2)
	case 1: // At 80% of TTL
		rt = rt.Add((ttl * 4) / 5)
	case 2:
		rt = rt.Add(ttl)

	}
	if now.After(rt) {
		dnssdlog("checkRecord: TRIGGER!: a=", a, ", requeried=", a.requeried, ",  rt=", rt, ", ttl=", ttl)
		if a.requeried < 2 {
			a.requeried++
			return 1 // Requery
		} else {
			return -1 // Remove
		}
	}
	return 0 // OK, do nothing
}

func (a *answer) isClosed() bool {
	return contextIsClosed(a.ctx)
}
