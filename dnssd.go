/* The dnssd package is a pure go implementation of DNS Service Discovery
also known as Bonjour(TM).
*/
package dnssd

import (
	"time"

	"github.com/miekg/dns"
)

type dnssd struct {
	ns    *netserver
	cs    *questions
	cmdCh chan func()
	rrc   *answers
	rrl   *answers
}

var ds *dnssd

func getDnssd() *dnssd {
	if ds == nil {
		ns, err := makeNetserver()
		if err != nil {
			panic("Could not start netserver")
		}
		cmdCh := make(chan func(), 32)
		ds = &dnssd{ns, &questions{nil}, cmdCh, nil, nil}
		ds.rrc = makeAnswers() // Remote entries, lookup only
		ds.rrl = makeAnswers() // Local entries, repond and lookup.

		go ds.processing()
		startup()
	}
	return ds
}

func (ds *dnssd) processing() {
	t := time.NewTimer(10 * time.Millisecond)
	nt := time.Millisecond * 10
outer:
	for {
		t.Reset(nt)
		select {
		case cmd := <-ds.cmdCh:
			cmd()
		case im := <-ds.ns.msgCh:
			ds.handleIncomingMessage(im)
		case <-t.C:
			ds.cleanClosedAnswers()
		}
		for {
			select {
			case cmd := <-ds.cmdCh:
				cmd()
			case im := <-ds.ns.msgCh:
				ds.handleIncomingMessage(im)
			default:
				ds.ns.sendPending()
				continue outer
			}
		}
	}
}

func (ds *dnssd) cleanClosedAnswers() {
	closed := ds.rrl.findClosedAnswers()
	for _, a := range closed {
		a.rr.Header().Ttl = 0
		dnssdlog("SENDING UNPUBLISH..", a.rr)
		ds.ns.publish(a.ifIndex, a.rr)
	}
}

func (ds *dnssd) handleIncomingMessage(im *incomingMsg) {
	ds.cleanClosedAnswers()
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
	ds.rrl.add(a)
	ds.ns.sendResponseRecord(ifIndex, a.rr)
}

// Check all cached RR entries and send a question for more
// data.
func (ds *dnssd) runQuery(ifIndex int, q *dns.Question, cb *callback) {
	matchedAnswers := ds.rrc.matchQuestion(q)

	// Find a currently running query and attach this command.
	cq := ds.cs.findQuestion(q)

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
		a := &answer{nil, ifIndex, rr}
		isNew := ds.rrc.add(a)
		if cq != nil && isNew {
			cq.respond(a)
		}
	}
}

// Shutdown server will close currently open connections & channel
func (ds *dnssd) shutdown() error {
	close(ds.cmdCh)
	return ds.ns.shutdown()
}
