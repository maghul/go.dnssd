package dnssd

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func nextNotification(ctxn *contextNotifier, nc chan context.Context) (ctx context.Context, empty bool) {
	ctxn.sync()
	select {
	case ctxr := <-nc:
		return ctxr, false
	default:
		return nil, true
	}

}

func TestContextNotification(t *testing.T) {
	ctxn := initContextNotifier()

	ctx1, cancel := context.WithCancel(context.Background())

	nc := ctxn.getContextNotifications()
	ctxn.sync()
	ctxn.addContextForNotifications(ctx1)

	cancel()
	ctxr, empty := nextNotification(ctxn, nc)
	assert.False(t, empty)
	assert.Equal(t, ctx1, ctxr)

	cancel()
	ctxr, empty = nextNotification(ctxn, nc)
	assert.True(t, empty)
}

func TestContextNotificationMany(t *testing.T) {
	ctxn := initContextNotifier()

	ctx1 := context.WithValue(context.Background(), "name", "test1")
	assert.Equal(t, 0, ctxn.count())

	ctxs := []context.Context{}
	ctxs = internalAddContext(ctxs, ctx1)
	assert.Equal(t, 1, len(ctxs))

	ctxs = internalAddContext(ctxs, ctx1)
	assert.Equal(t, 1, len(ctxs))

	ctx2 := context.WithValue(context.Background(), "name", "test2")
	ctxs = internalAddContext(ctxs, ctx2)
	assert.Equal(t, 2, len(ctxs))

	ctxs = internalRemoveContext(ctxs, ctx1)
	assert.Equal(t, 1, len(ctxs))
}
