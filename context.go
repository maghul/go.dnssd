package dnssd

import (
	"context"
)

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
