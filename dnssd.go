/* The dnssd package is a pure go implementation of DNS Service Discovery
also known as Bonjour(TM).
*/
package dnssd

import (
	"context"
	"time"

	"github.com/miekg/dns"
)

type dnssd struct {
	ns    *netserver
	cs    *questions
	cmdCh chan func()
	rrc   *answers
	rrl   *answers
	ctxn  *contextNotifier
	cn    chan context.Context
}

var ds *dnssd

func getDnssd() *dnssd {
	if ds == nil {
		ns, err := makeNetserver()
		if err != nil {
			panic("Could not start netserver")
		}
		cmdCh := make(chan func(), 32)
		ds = &dnssd{ns: ns, cs: &questions{nil}, cmdCh: cmdCh, ctxn: initContextNotifier()}
		ds.rrc = makeAnswers() // Remote entries, lookup only
		ds.rrl = makeAnswers() // Local entries, repond and lookup.
		ds.cn = ds.ctxn.getContextNotifications()

		go ds.processing()
		startup()
	}
	return ds
}

func (ds *dnssd) processing() {
	t := time.NewTimer(10 * time.Millisecond)
	t.Stop()
	var nt time.Time
	var st time.Time
outer:
	for {

		if nt.IsZero() {
			t.Stop()
		} else if st != nt {
			st = nt
			dnssdlog("Next Timed event occurs at ", st)
			now := time.Now()
			t.Reset(st.Sub(now))
		}
		select {
		case cmd := <-ds.cmdCh:
			cmd()
		case im := <-ds.ns.msgCh:
			ds.handleIncomingMessage(im)
		case ctx := <-ds.cn:
			nt = ds.handleClosedContext(ctx)
		case <-t.C:
			nt = ds.checkRunningEvents()
		}
		for {
			select {
			case cmd := <-ds.cmdCh:
				cmd()
			case im := <-ds.ns.msgCh:
				ds.handleIncomingMessage(im)
			case ctx := <-ds.cn:
				nt = ds.handleClosedContext(ctx)
			default:
				ds.ns.sendPending()
				continue outer
			}
		}
	}
}

func (ds *dnssd) handleClosedContext(ctx context.Context) time.Time {
	dnssdlog("handleClosedContext: ctx=", ctx)
	// This will do what we want when a context has been closed
	// but it will do unnecessary scanning of all records so it
	// can be optimized.
	return ds.checkRunningEvents()
}

func (ds *dnssd) checkRunningEvents() time.Time {
	t1 := ds.updateTTLOnPublishedRecords()
	t2 := ds.requeryOldAnswers()
	nt := getNextTime(t1, t2)
	return nt
}

func (ds *dnssd) handleIncomingMessage(im *incomingMsg) {
	if im.msg.Response {
		ds.handleResponseRecords(im, im.msg.Answer)
		ds.handleResponseRecords(im, im.msg.Ns)
		ds.handleResponseRecords(im, im.msg.Extra)
	} else {
		// Check each question find matching answers and remove
		// any already known by peer.
		for _, q := range im.msg.Question {
			answered := false
			matchedResponses := ds.rrl.matchQuestion(&q)
		nextMatchedResponse:
			for _, mr := range matchedResponses {
				for _, kr := range im.msg.Answer {
					if matchRRs(mr.rr, kr) {
						// Already known by peer so...
						continue nextMatchedResponse
					}
				}
				ds.ns.sendResponseRecord(im.ifIndex, mr.rr)
				answered = true
			}
			if answered {
				ds.ns.sendResponseQuestion(im.ifIndex, &q)
			}
		}
	}
}

func (ds *dnssd) publish(ifIndex int, a *answer) {
	ds.ctxn.addContextForNotifications(a.ctx)
	ds.rrl.add(a)
	ds.ns.sendResponseRecord(ifIndex, a.rr)
}

// Check all cached RR entries and send a question for more
// data.
func (ds *dnssd) runQuery(ifIndex int, q *dns.Question, cb *callback) {
	matchedAnswers := ds.rrc.matchQuestion(q)

	// Find a currently running query and attach this command.
	cq := ds.cs.findQuestion(q)
	ds.ctxn.addContextForNotifications(cb.ctx)

	// Check the cache for all entries matching and respond with these.
	for _, a := range matchedAnswers {
		dnssdlog("ANSWER ", a)
		cb.respond(a)
		if cq == nil {
			// Only add known answers if we intend to ask a question
			ds.ns.sendKnownAnswer(ifIndex, a.rr)
		}
	}

	if cq == nil {
		cq = ds.cs.makeQuestion(q)
		cq.attach(cb)
		ds.ns.sendQuestion(ifIndex, q)
	} else {
		cq.attach(cb)
	}
}

// Start a probe query. Will not check the cache
func (ds *dnssd) runProbe(ifIndex int, q *dns.Question, cb *callback) {
	cq := ds.cs.makeQuestion(q)
	cq.attach(cb)
	ds.ns.sendQuestion(ifIndex, q)
}

func (ds *dnssd) handleResponseRecords(im *incomingMsg, rrs []dns.RR) {
	ifIndex := im.ifIndex
	for _, rr := range rrs {

		cacheFlush := rr.Header().Class&0x8000 != 0
		if cacheFlush {
			dnssdlog("DNSSD FLUSH! ", ifIndex, ", RR=", rr)
		} else {
			dnssdlog("DNSSD        ", ifIndex, ", RR=", rr)

		}
		rr.Header().Class &= 0x7fff
		// TODO: Is this a response or a challenge?
		cq := ds.cs.findQuestionFromRR(rr)
		a, isNew := ds.rrc.addRecord(ifIndex, rr)
		if cq != nil && isNew {
			cq.respond(a)
		}
	}
}

// Look through ds.rrl for records which are about to expire
// and republish them unless their context has cancelled them
// Return a time for next published record to update TTL for
func (ds *dnssd) updateTTLOnPublishedRecords() time.Time {
	return ds.rrl.findOldAnswers(func(a *answer) {
		// Republish old answers...
		a.added = time.Now()
		a.requeried = 0
		dnssdlog("SENDING REPUBLISH..", a.rr)
		ds.ns.sendResponseRecord(a.ifIndex, a.rr)
	}, func(a *answer) {
		a.rr.Header().Ttl = 0
		dnssdlog("SENDING UNPUBLISH..", a.rr)
		ds.ns.sendResponseRecord(a.ifIndex, a.rr)
	})
}

// Look through ds.rrc and check if the record is about to
// expire, if we are still interested requery otherwise just close
// the answer.
// Return a time for next record to requery.
func (ds *dnssd) requeryOldAnswers() time.Time {
	return ds.rrc.findOldAnswers(func(a *answer) {
		// Requery the record if we have a question for it...
		q := ds.cs.findQuestionFromRR(a.rr)
		if q != nil && q.isActive() {
			ds.ns.sendQuestion(a.ifIndex, q.q)
		}
	}, func(a *answer) {
		// The record has been removed
		dnssdlog("Record removed:", a)
	})
}

// Shutdown server will close currently open connections & channel
func (ds *dnssd) shutdown() error {
	close(ds.cmdCh)
	return ds.ns.shutdown()
}
