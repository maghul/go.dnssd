package dnssd

import (
	"context"
	"time"
)

type callback struct {
	ctx  context.Context
	call QueryAnswered
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

func (r *callback) respond(ifIndex int, a *answer) {
	f := func() {
		r.call(0, ifIndex, a.rr)
	}

	dnssdlog("CALLBACK ", r.isValid(), ", ANSWER=", a)
	select {
	case callbackChan <- f:
		return
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
}

func (cb *callback) isValid() bool {
	select {
	case <-cb.ctx.Done():
		return false
	default:
	}
	return true
}
