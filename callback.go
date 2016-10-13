package dnssd

import (
	"context"
	"time"
)

type callback struct {
	ctx     context.Context
	ifIndex int
	call    QueryAnswered
}

var callbackChan chan func() = make(chan func())
var callbackThreads int = 0

func callbackThread(num int) {
	for {
		call := <-callbackChan
		dnssdlog("CALL: #", num)
		call()
	}
}

func (cb *callback) isClosed() bool {
	return contextIsClosed(cb.ctx)
}

// return false if the callback is invalid and should be removed.
func (cb *callback) respond(a *answer) bool {
	if cb.isClosed() {
		return false
	}

	if cb.ifIndex != 0 && cb.ifIndex != a.ifIndex {
		return true
	}

	flags := None
	if a.ttl > 0 {
		flags = RecordAdded
	}
	f := func() {
		cb.call(flags, a.ifIndex, a.rr)
	}

	select {
	case callbackChan <- f:
		return true
	default:
	}

	t := time.NewTimer(time.Millisecond)
	select {
	case callbackChan <- f:
	case <-t.C:
		// callback thread isn't reading. start a new one.
		callbackThreads++
		go callbackThread(callbackThreads)
		dnssdlog("------------------> started callback thread #", callbackThreads)
		callbackChan <- f
	}
	return true
}
