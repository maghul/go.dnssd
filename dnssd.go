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
	for {
		select {
		case cmd := <-ds.cmdCh:
			cmd()
		case im := <-ds.ns.msgCh:
			ds.handleResponseRecords(im.ifIndex, im.msg.Answer)
			ds.handleResponseRecords(im.ifIndex, im.msg.Ns)
			ds.handleResponseRecords(im.ifIndex, im.msg.Extra)
		}
	}
}

func (ds *dnssd) publish(a *answer) {
	ds.rrl.add(a)
	// TODO: We may want to batch these.
	resp := new(dns.Msg)
	resp.MsgHdr.Response = true
	resp.Answer = []dns.RR{a.rr}
	go ds.ns.sendMessage(resp)
}

// Check all cached RR entries and send a question for more
// data.
func (ds *dnssd) runQuery(ifIndex int, q *dns.Question, cb *callback) {
	matchedAnswers := ds.rrc.matchQuestion(q)

	// Check the cache for all entries matching and respond with these.
	for _, a := range matchedAnswers {
		dnssdlog("ANSWER ", a)
		cb.respond(a)
	}

	// Find a currently running query and attach this command.
	cq := ds.cs.findQuestion(q)
	if cq == nil {
		cq = ds.cs.makeQuestion(q)
		cq.attach(cb)
		queryMsg := new(dns.Msg)
		queryMsg.MsgHdr.Response = false
		queryMsg.Question = []dns.Question{*q}
		queryMsg.Answer = rrs(matchedAnswers)
		ds.ns.sendMessage(queryMsg)
	} else {
		cq.attach(cb)
	}
}

// Start a probe query. Will not check the cache
func (ds *dnssd) runProbe(ifIndex int, q *dns.Question, cb *callback) {
	cq := ds.cs.makeQuestion(q)
	cq.attach(cb)
	ds.ns.addQuestion(ifIndex, q)
}

func (ds *dnssd) handleResponseRecords(ifIndex int, rrs []dns.RR) {
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
