package dnssd

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

/*
This is a context notifier. It will listen on multiple contexts for
the close message and when a context is closed it will notify on a context
notification channel which context closed.
*/
type contextNotifier struct {
	contextNotifierMutex sync.Mutex
	contextNotifierChan  chan func()
	nots                 []chan context.Context
	ctxs                 []context.Context
}

func initContextNotifier() *contextNotifier {
	ctxn := &contextNotifier{}

	ctxn.contextNotifierChan = make(chan func(), 8)
	go ctxn.runContextNotifier()
	return ctxn
}

func (ctxn *contextNotifier) count() int {
	return len(ctxn.ctxs)
}

func (ctxn *contextNotifier) getContextNotifications() chan context.Context {
	nc := make(chan context.Context, 2)
	ctxn.contextNotifierChan <- func() {
		ctxn.nots = append(ctxn.nots, nc)
	}
	return nc
}

func (ctxn *contextNotifier) removeContextNotifications(nc chan context.Context) {
	ctxn.contextNotifierChan <- func() {
		ctxn.nots = internalRemoveNotifications(ctxn.nots, nc)
	}
}
func (ctxn *contextNotifier) addContextForNotifications(ctx context.Context) {
	ctxn.contextNotifierChan <- func() {
		ctxn.ctxs = internalAddContext(ctxn.ctxs, ctx)
	}
}

func (ctxn *contextNotifier) removeContextForNotifications(ctx context.Context) {
	ctxn.contextNotifierChan <- func() {
		ctxn.ctxs = internalRemoveContext(ctxn.ctxs, ctx)
	}
	ctxn.sync()
}

func (ctxn *contextNotifier) runContextNotifier() {
	for {

		cases := make([]reflect.SelectCase, len(ctxn.ctxs)+1)
		cases[0] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ctxn.contextNotifierChan)}
		for i, ctx := range ctxn.ctxs {
			cases[i+1] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ctx.Done())}
		}
		chosen, value, ok := reflect.Select(cases)
		if chosen == 0 {
			// This is the contextNotifierChan
			ncf := value.Interface().(func())
			ncf()
		} else {
			// This is a ctx.Done channel
			if ok {
				fmt.Println("We should never get an OK from a context channel, only close")
			}
			ctx := ctxn.ctxs[chosen-1]
			ctxn.notify(ctx)
			ctxn.ctxs = internalRemoveContext(ctxn.ctxs, ctx)
		}
	}
}

func (ctxn *contextNotifier) notify(ctx context.Context) {
	for _, not := range ctxn.nots {
		not <- ctx
	}
}

// Ensure that there are no pending functions in the
// channel.
func (ctxn *contextNotifier) sync() {
	cb := make(chan struct{})
	ctxn.contextNotifierChan <- func() {
		close(cb)
	}
	<-cb
}

func internalRemoveNotifications(nots []chan context.Context, nc chan context.Context) []chan context.Context {
	ii := 0
	for _, rnc := range nots {
		if nc != rnc {
			nots[ii] = rnc
			ii++
		}
	}
	return nots[0:ii]
}

func internalAddContext(ctxs []context.Context, ctx context.Context) []context.Context {
	for _, rctx := range ctxs {
		if ctx == rctx {
			return ctxs
		}
	}
	return append(ctxs, ctx)
}

func internalRemoveContext(ctxs []context.Context, ctx context.Context) []context.Context {
	ii := 0
	for _, rctx := range ctxs {
		if ctx != rctx {
			ctxs[ii] = rctx
			ii++
		}
	}
	return ctxs[0:ii]
}

func contextIsClosed(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	select {
	case <-ctx.Done():
		return true
	default:
	}
	return false

}
