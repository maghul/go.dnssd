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
	ns       *netserver
	cs       *questions
	cmdCh    chan func()
	rrc      *answers
	rrl      *answers
	ctxn     *contextNotifier
	cn       chan context.Context
	nextSend time.Time
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

func (ds *dnssd) nextSendAt(t time.Duration) {
	nt := time.Now().Add(t)
	if ds.nextSend.IsZero() || ds.nextSend.After(nt) {
		ds.nextSend = nt
	}
}

func (ds *dnssd) processing() {
	checkTimer := time.NewTimer(10 * time.Millisecond)
	sendTimer := time.NewTimer(10 * time.Millisecond)

	var nt time.Time
	var st time.Time

	var nextSendTime time.Time

	for {

		now := time.Now()
		if nt.IsZero() {
			checkTimer.Stop()
		} else if st != nt {
			st = nt
			dnssdlog("Next Timed event occurs at ", st)
			checkTimer.Reset(st.Sub(now))
		}

		if ds.nextSend.IsZero() {
			sendTimer.Stop()
		} else {
			if now.After(ds.nextSend) {
				// Dont set an antedated timer...
				nextSendTime = time.Time{}
				ds.nextSend = nextSendTime
				ds.ns.sendPending()
			} else if nextSendTime.IsZero() || nextSendTime.After(ds.nextSend) {
				nextSendTime = ds.nextSend
				delay := nextSendTime.Sub(now)
				sendTimer.Reset(delay)
			}
		}

		select {
		case cmd := <-ds.cmdCh:
			cmd()
		case im := <-ds.ns.msgCh:
			ds.handleIncomingMessage(im)
		case ctx := <-ds.cn:
			nt = ds.handleClosedContext(ctx)
		case <-checkTimer.C:
			nt = ds.checkRunningEvents()
		case <-sendTimer.C:
			nextSendTime = time.Time{}
			ds.ns.sendPending()
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
	if isFromLocalHost(im.ifIndex, im.from) {
		return
	}

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
				if mr.flags&Unique != 0 {
					ds.nextSendAt(0)
				} else {
					ds.nextSendAt(randomDuration(500*time.Millisecond, 100))
				}
				ds.ns.sendResponseRecord(im.ifIndex, mr.rr)
				answered = true
			}
			if answered {
				ds.nextSendAt(500 * time.Millisecond)
				ds.ns.sendResponseQuestion(im.ifIndex, &q)
			}
		}
	}
}

func (ds *dnssd) publish(ctx context.Context, flags Flags, ifIndex int, record dns.RR) {
	ds.ctxn.addContextForNotifications(ctx)
	a, _ := ds.rrl.addRecord(ctx, flags, ifIndex, record)
	ds.rrl.add(a)
	ds.nextSendAt(10 * time.Millisecond)
	ds.ns.sendResponseRecord(ifIndex, a.rr)

	cq := ds.cs.findQuestionFromRR(a.rr)
	if cq != nil {
		cq.respond(a)
	}
}

// Check all cached RR entries and send a question for more
// data.
func (ds *dnssd) runQuery(ifIndex int, q *dns.Question, cb *callback) {
	// Find a currently running query and attach this command.
	cq := ds.cs.findQuestion(q)
	ds.ctxn.addContextForNotifications(cb.ctx)

	// Check the cache for all entries matching and respond with these.
	f := func(a *answer) {
		dnssdlog("ANSWER ", a)
		cb.respond(a)
		if cq == nil {
			// Only add known answers if we intend to ask a question
			ds.nextSendAt(500 * time.Millisecond)
			ds.ns.sendKnownAnswer(ifIndex, a.rr)
		}
	}
	ds.rrc.iterateAnswersForQuestion(q, f)
	ds.rrl.iterateAnswersForQuestion(q, f)

	if cq == nil {
		cq = ds.cs.makeQuestion(q)
		cq.attach(cb)
		ds.nextSendAt(10 * time.Millisecond)
		ds.ns.sendQuestion(ifIndex, q)
	} else {
		cq.attach(cb)
	}
}

// Start a probe query. Will not check the cache
func (ds *dnssd) runProbe(ifIndex int, q *dns.Question, cb *callback) {
	cq := ds.cs.makeQuestion(q)
	cq.attach(cb)
	ds.nextSendAt(10 * time.Millisecond)
	ds.ns.sendQuestion(ifIndex, q)
}

func (ds *dnssd) handleResponseRecords(im *incomingMsg, rrs []dns.RR) {
	ifIndex := im.ifIndex
	for _, rr := range rrs {

		cacheFlush := rr.Header().Class&0x8000 != 0
		flags := Shared
		if cacheFlush {
			flags = Unique
		}
		rr.Header().Class &= 0x7fff
		// TODO: Is this a response or a challenge?
		cq := ds.cs.findQuestionFromRR(rr)
		a, isNew := ds.rrc.addRecord(nil, flags, ifIndex, rr)
		if cq != nil && isNew {
			cq.respond(a)
		}

		challenge, ok := ds.rrl.findAnswerFromRR(rr)
		if ok {
			// We have a challenge record.
			if challenge.flags&Unique != 0 {
				ds.nextSendAt(0)
				dnssdlog("CHALLENGE!, ", rr, challenge)
				ds.ns.sendResponseRecord(ifIndex, challenge.rr)
			}
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
		ds.nextSendAt(100 * time.Millisecond)
		ds.ns.sendResponseRecord(a.ifIndex, a.rr)
	}, func(a *answer) {
		a.rr.Header().Ttl = 0
		dnssdlog("SENDING UNPUBLISH..", a.rr)
		ds.nextSendAt(100 * time.Millisecond)
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
			ds.nextSendAt(100 * time.Millisecond)
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
