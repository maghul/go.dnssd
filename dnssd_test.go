package dnssd

import (
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func makeTestDnssd(t *testing.T) (*dnssd, chan func()) {
	cmdCh := make(chan func(), 32)
	ns, err := makeTestNetserver()
	assert.NoError(t, err)
	assert.NotNil(t, ns)

	ds = &dnssd{ns: ns, cs: &questions{nil}, cmdCh: cmdCh, ctxn: initContextNotifier()}
	ds.rrc = makeAnswers() // Remote entries, lookup only
	ds.rrl = makeAnswers() // Local entries, repond and lookup.
	ds.cn = ds.ctxn.getContextNotifications()

	testlog("response=", ds.ns.response)
	return ds, cmdCh
}

func (ds *dnssd) addPublishedAnswer(name string, ifIndex int) {
	a := makeTestPtrAnswer(ifIndex, name, "hoppla", 12000)
	ds.rrl.add(a)
	testlog("response=", ds.ns.response)

}
func (ds *dnssd) runTestQuestion(name string, ifIndex int) {
	q := makeTestPtrQuestion(name)
	qs := []dns.Question{*(q.q)}
	msg := &dns.Msg{Question: qs}
	im := &incomingMsg{msg: msg, ifIndex: ifIndex}
	ds.handleIncomingMessage(im)
}

func TestHandleIncomingMessageQuestion(t *testing.T) {
	ds, _ := makeTestDnssd(t)
	ifIndex := 2
	name := "_tuting._tcp"
	ds.addPublishedAnswer(name, ifIndex)
	ds.runTestQuestion(name, ifIndex)

	responseMsg := ds.ns.response
	testlog("response=", ds.ns.response)
	assert.NotNil(t, responseMsg)
	assert.Equal(t, 1, len(responseMsg.Answer))
}

func TestTimes(t *testing.T) {
	var t1, t2 time.Time
	t1 = time.Now()

	tr := getNextTime(t1, t2)
	assert.Equal(t, t1, tr)

	tr = getNextTime(t2, t1)
	assert.Equal(t, t1, tr)
}
