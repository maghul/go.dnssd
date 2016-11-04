package dnssd

import (
	"fmt"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func makeTestPtrAnswer(ifIndex int, name, ptr string, ttl uint32) *answer {
	ptr1 := new(dns.PTR)
	ptr1.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: ttl} // TODO: TTL correct?
	ptr1.Ptr = ptr
	return &answer{nil, time.Now(), time.Duration(ttl) * time.Second, Shared, 0, ifIndex, ptr1}
}

func makeTestPtrQuestion(name string) *question {
	return &question{&dns.Question{Name: name, Qclass: dns.ClassINET, Qtype: dns.TypePTR}, nil}
}

func (aa *answers) dump(ref string) {
	dnssdlog.Debug.Println("--------------- START DUMP --- ", ref, " ---------------")
	for _, a := range aa.cache {
		dnssdlog.Debug.Println("DUMP: ", a)
	}
	dnssdlog.Debug.Println("---------------- END DUMP --- ", ref, " ---------------")
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

func TestAnswerString(t *testing.T) {
	now := time.Now()

	a1 := makeTestPtrAnswer(2, "hi_there", "wazzup", 3200)
	a1.added = now
	a1.flags = Shared
	expected := fmt.Sprint("Answer{if=2, added=", now, ", Shared, rr=hi_there\t3200\tIN\tPTR\twazzup}")
	assert.Equal(t, expected, a1.String())

	a1.flags = Unique
	expected = fmt.Sprint("Answer{if=2, added=", now, ", Unique, rr=hi_there\t3200\tIN\tPTR\twazzup}")
	assert.Equal(t, expected, a1.String())
}

func TestAnswerSharedGetNextTime(t *testing.T) {
	now := time.Now()

	a1 := makeTestPtrAnswer(2, "hi_there", "wazzup", 1000)
	a1.added = now
	a1.flags = Shared

	expected := now.Add(800 * time.Second)
	nt, keep := a1.getNextCheckTime()
	assert.True(t, keep)
	assert.Equal(t, expected, nt)

	a1.requeried++
	expected = now.Add(850 * time.Second)
	nt, keep = a1.getNextCheckTime()
	assert.True(t, keep)
	assert.Equal(t, expected, nt)

	a1.requeried++
	expected = now.Add(900 * time.Second)
	nt, keep = a1.getNextCheckTime()
	assert.True(t, keep)
	assert.Equal(t, expected, nt)

	a1.requeried++
	expected = now.Add(950 * time.Second)
	nt, keep = a1.getNextCheckTime()
	assert.True(t, keep)
	assert.Equal(t, expected, nt)

	a1.requeried++
	nt, keep = a1.getNextCheckTime()
	assert.False(t, keep)
}

func TestAnswerUniqueGetNextTime(t *testing.T) {
	now := time.Now()

	a1 := makeTestPtrAnswer(2, "hi_there", "wazzup", 1000)
	a1.added = now
	a1.flags = Unique

	expected := now.Add(800 * time.Second)
	nt, keep := a1.getNextCheckTime()
	assert.True(t, keep)
	assert.Equal(t, expected, nt)

	a1.requeried++
	nt, keep = a1.getNextCheckTime()
	assert.False(t, keep)
}

func testRunFindOldAnswers(aa *answers) (req, rem *answer, nt time.Time) {
	nt = aa.findOldAnswers(func(a *answer) {
		req = a
	}, func(a *answer) {
		rem = a
	})
	return req, rem, nt
}
func TestAnswerDoRequery(t *testing.T) {
	now := time.Now()
	aa := makeAnswers()

	a1 := makeTestPtrAnswer(2, "hi_there", "wazzup", 1000)
	aa.add(a1)
	a1.added = now.Add(-801 * time.Second)
	req, rem, _ := testRunFindOldAnswers(aa)

	assert.Equal(t, a1, req)
	assert.Nil(t, rem)
}

func TestAnswerDoRemoved(t *testing.T) {
	now := time.Now()
	aa := makeAnswers()

	a1 := makeTestPtrAnswer(2, "hi_there", "wazzup", 1000)
	aa.add(a1)
	a1.added = now.Add(-1001 * time.Second)
	a1.requeried = 5
	req, rem, _ := testRunFindOldAnswers(aa)
	assert.Equal(t, a1, rem)
	assert.Nil(t, req)
}

func TestAnswerDoNothing(t *testing.T) {
	now := time.Now()
	aa := makeAnswers()

	a1 := makeTestPtrAnswer(2, "hi_there", "wazzup", 1000)
	aa.add(a1)
	a1.added = now

	req, rem, nt := testRunFindOldAnswers(aa)
	assert.Nil(t, rem)
	assert.Nil(t, req)

	expected := now.Add(800 * time.Second)
	assert.Equal(t, expected, nt)
}
