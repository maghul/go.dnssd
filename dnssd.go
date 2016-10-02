/* The dnssd package is a pure go implementation of DNS Service Discovery
also known as Bonjour(TM).
*/
package dnssd

import (
	"github.com/miekg/dns"
)

type dnssd struct {
	ns    *netserver
	cs    questions
	cmdCh chan func()
	rrc   *answers
	rrl   *answers
}

var ds *dnssd

func getDnssd() *dnssd {
	if ds == nil {
		ns, err := makeNetserver(nil)
		if err != nil {
			panic("Could not start netserver")
		}
		ns.startReceiving()
		cmdCh := make(chan func(), 32)
		ds = &dnssd{ns, nil, cmdCh, nil, nil}
		ds.rrc = &answers{} // Remote entries, lookup only
		ds.rrl = &answers{} // Local entries, repond and lookup.

		go ds.processing()
	}
	return ds
}

func (ds *dnssd) processing() {
	for {
		select {
		case cmd := <-ds.cmdCh:
			cmd()
		case msg := <-ds.ns.msgCh:
			ifIndex := 0 // TODO get this from msgCh
			ds.handleResponseRecords(ifIndex, msg.Answer)
			ds.handleResponseRecords(ifIndex, msg.Ns)
			ds.handleResponseRecords(ifIndex, msg.Extra)
		}
	}
}
func (ds *dnssd) publish(a *answer) {
	ds.rrl.add(a)
	// TODO: We may want to batch these.
	resp := new(dns.Msg)
	resp.MsgHdr.Response = true
	resp.Answer = []dns.RR{a.rr}
	go ds.ns.sendUnsolicitedMessage(resp)
}

// Check all cached RR entries and send a question for more
// data.
func (ds *dnssd) runQuery(ifIndex int, q *dns.Question, cb *callback) {
	matchedAnswers := ds.rrc.matchQuestion(q)

	// Check the cache for all entries matching and respond with these.
	for _, a := range matchedAnswers {
		dnssdlog("ANSWER ", a)
		ifIndex := 0 // TODO: Should be part of the answer record
		if cb.isValid() {
			cb.respond(ifIndex, a)
		}
	}

	// Find a currently running query and attach this command.
	cq := ds.cs.findQuestion(q)
	if cq == nil {
		cq = makeQuestion(q)
		ds.cs = append(ds.cs, cq)
		cq.attach(cb)
		queryMsg := new(dns.Msg)
		queryMsg.MsgHdr.Response = false
		queryMsg.Question = []dns.Question{*q}
		queryMsg.Answer = rrs(matchedAnswers)
		dnssdlog("sendQuery", cq)
		ds.ns.sendQuery(queryMsg)
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
		cq := ds.findQuery(rr)
		a := &answer{ifIndex, rr}
		isNew := ds.rrc.add(a)
		if cq != nil && isNew {
			cq.respond(ifIndex, a)
		}
	}
}

func (ds *dnssd) findQuery(rr dns.RR) *question {
	for _, dsq := range ds.cs {
		if matchQuestionAndRR(dsq.q, rr) {
			return dsq
		}
	}
	// TODO: Just cache the rr.
	return nil
}

func matchQuestionAndRR(q *dns.Question, rr dns.RR) bool {
	return (q.Qtype == rr.Header().Rrtype) &&
		(q.Qclass == rr.Header().Class) &&
		(q.Name == rr.Header().Name)
}

func matchRRHeader(rr1, rr2 dns.RR) bool {
	return (rr1.Header().Rrtype == rr2.Header().Rrtype) &&
		(rr1.Header().Class == rr2.Header().Class) &&
		(rr1.Header().Name == rr2.Header().Name)
}

func matchRRs(rr1, rr2 dns.RR) bool {
	return rr1.String() == rr2.String()
}

func matchQuestions(q1, q2 *dns.Question) bool {
	return (q1.Qtype == q2.Qtype) &&
		(q1.Qclass == q2.Qclass) &&
		(q1.Name == q2.Name)
}

// Shutdown server will close currently open connections & channel
func (ds *dnssd) shutdown() error {
	close(ds.cmdCh)
	return ds.ns.shutdown()
}
