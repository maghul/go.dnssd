package dnssd

import (
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func TestCharEncoding(t *testing.T) {
	base := "000420129182\\@K\\195\\182ket"
	ref := "000420129182@Köket"

	v := RepackToUTF8(base)
	assert.Equal(t, ref, v)
}
func TestCharEncoding2(t *testing.T) {
	base := "000420129182\\@K\\111ket"
	ref := "000420129182@Koket"

	v := RepackToUTF8(base)
	assert.Equal(t, ref, v)
}

func TestMatchQuestionAndRR(t *testing.T) {
	q := &dns.Question{Name: "hi_there", Qclass: dns.ClassINET, Qtype: dns.TypePTR}
	a1 := makeTestPtrAnswer(2, "hi_there", "wazzup", 3200)

	assert.True(t, matchQuestionAndRR(q, a1.rr))
}

func TestMatchRRHeader(t *testing.T) {
	a1 := makeTestPtrAnswer(2, "hi_there", "wazzup", 3200)
	a2 := makeTestPtrAnswer(2, "hi_there", "wazzup", 3200)
	assert.True(t, matchRRHeader(a1.rr.Header(), a2.rr.Header()))
}

func TestMatchRRs(t *testing.T) {
	a1 := makeTestPtrAnswer(2, "hi_there", "wazzup", 3200)
	a2 := makeTestPtrAnswer(2, "hi_there", "wazzup", 3200)
	assert.True(t, matchRRs(a1.rr, a2.rr))
}

func TestMatchQuestions(t *testing.T) {
	q1 := &dns.Question{Name: "hi_there", Qclass: dns.ClassINET, Qtype: dns.TypePTR}
	q2 := &dns.Question{Name: "hi_there", Qclass: dns.ClassINET, Qtype: dns.TypePTR}
	assert.True(t, matchQuestions(q1, q2))
}

func TestTimes(t *testing.T) {
	var t1, t2 time.Time
	t1 = time.Now()

	tr := getNextTime(t1, t2)
	assert.Equal(t, t1, tr)

	tr = getNextTime(t2, t1)
	assert.Equal(t, t1, tr)
}
