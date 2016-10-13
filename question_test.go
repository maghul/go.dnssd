package dnssd

import (
	"context"
	"fmt"
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func TestQuestionMatching(t *testing.T) {
	q := makeTestPtrQuestion("test_xyz")
	a := makeTestPtrAnswer(2, "test_xyz", "Abra", 17)

	assert.True(t, q.match(a))
}

func TestQuestionMatchingCallback1(t *testing.T) {
	q := makeTestPtrQuestion("test_xyz")
	a := makeTestPtrAnswer(2, "test_xyz", "Abra", 17)

	trace := make(chan string)
	ctx := context.Background()
	cb1 := makeCallback("test", 1, ctx, a.ifIndex, func(flags Flags, ifIndex int, rr dns.RR) {
		trace <- fmt.Sprint("cb1:", rr)
	})
	q.attach(cb1)
	assert.True(t, q.match(a))
	assert.Equal(t, "cb1:test_xyz\t17\tIN\tPTR\tAbra", <-trace)

	cb2 := makeCallback("test", 2, ctx, a.ifIndex, func(flags Flags, ifIndex int, rr dns.RR) {
		trace <- fmt.Sprint("cb2:", rr)
	})
	q.attach(cb1)
	q.attach(cb2)

	assert.True(t, q.match(a))
	assert.Equal(t, "cb1:test_xyz\t17\tIN\tPTR\tAbra", <-trace)
	assert.Equal(t, "cb2:test_xyz\t17\tIN\tPTR\tAbra", <-trace)

	q.detach(cb1)

}
