/* The dnssd package is a pure go implementation of DNS Service Discovery
also known as Bonjour(TM).
*/
package dnssd

import (
	"github.com/miekg/dns"
)

type dnssd struct {
	ns    *netservers
	cs    *questions
	cmdCh chan func()
	rrc   *answers
	rrl   *answers
}

var ds *dnssd

func getDnssd() *dnssd {
	if ds == nil {
		ns, err := makeNetservers()
		if err != nil {
			panic("Could not start netserver")
		}
		cmdCh := make(chan func(), 32)
		ds = &dnssd{ns, &questions{nil}, cmdCh, nil, nil}
		ds.rrc = makeAnswers() // Remote entries, lookup only
		ds.rrl = makeAnswers() // Local entries, repond and lookup.

		go ds.processing()
	}
	return ds
}

func (ds *dnssd) processing() {

outer:
	for {
		select {
		case cmd := <-ds.cmdCh:
			cmd()
		case im := <-ds.ns.msgCh:
			ds.handleIncomingMessage(im)
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

func (ds *dnssd) handleIncomingMessage(im *incomingMsg) {
	ds.handleResponseRecords(im, im.msg.Answer)
	ds.handleResponseRecords(im, im.msg.Ns)
	ds.handleResponseRecords(im, im.msg.Extra)
}

func (ds *dnssd) publish(ifIndex int, a *answer) {
	ds.rrl.add(a)
	ds.ns.publish(ifIndex, a.rr)
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
		a := &answer{ifIndex, rr}
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
