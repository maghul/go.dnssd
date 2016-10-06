package dnssd

import (
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func makeTestPtrAnswer(ifIndex int, name, ptr string, ttl uint32) *answer {
	ptr1 := new(dns.PTR)
	ptr1.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: ttl} // TODO: TTL correct?
	ptr1.Ptr = ptr
	return &answer{nil, ifIndex, ptr1}
}

func makeTestPtrQuestion(name string) *question {
	return &question{&dns.Question{Name: name, Qclass: dns.ClassINET, Qtype: dns.TypePTR}, nil}
}

func TestMatchingAnswers(t *testing.T) {
	a1 := makeTestPtrAnswer(2, "hi_there", "wazzup", 3200)
	a2 := makeTestPtrAnswer(2, "hi_there", "wazzup", 3200)

	assert.True(t, matchAnswers(a1, a2))
	a2.rr.Header().Name = "yo!"
	assert.False(t, matchAnswers(a1, a2))
}

func TestAddAnswer(t *testing.T) {
	a1 := makeTestPtrAnswer(2, "hi_there", "wazzup", 3200)
	a2 := makeTestPtrAnswer(2, "hi_there", "wazzup", 3200)
	a3 := makeTestPtrAnswer(2, "hi_there", "yowza", 3200)

	aa := makeAnswers()
	aa.add(a1)
	assert.Equal(t, 1, aa.size())
	aa.add(a2)
	assert.Equal(t, 1, aa.size())
	aa.add(a3)
	assert.Equal(t, 2, aa.size())
}

func TestMatchQuestion(t *testing.T) {
	aa := makeAnswers()

	q := &dns.Question{Name: "hi_there", Qclass: dns.ClassINET, Qtype: dns.TypePTR}
	assert.Nil(t, aa.matchQuestion(q))

	a1 := makeTestPtrAnswer(2, "hi_there", "wazzup", 3200)
	aa.add(a1)
	assert.Equal(t, 1, len(aa.matchQuestion(q)))
	aa.add(a1)
	assert.Equal(t, 1, len(aa.matchQuestion(q)))
	a2 := makeTestPtrAnswer(2, "hi_there", "yowza", 3200)
	aa.add(a2)
	assert.Equal(t, 2, len(aa.matchQuestion(q)))
}

func TestFindAnswerFromRR(t *testing.T) {
	aa := makeAnswers()

	a1 := makeTestPtrAnswer(2, "hi_there", "wazzup", 3200)
	aa.add(a1)

	qrr := makeTestPtrAnswer(0, "hi_there", "", 0)
	a, found := aa.findAnswerFromRR(qrr.rr)
	assert.True(t, found)
	assert.Equal(t, a1, a)

}
