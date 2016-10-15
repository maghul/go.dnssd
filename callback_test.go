package dnssd

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCallbackString(t *testing.T) {
	ctx := context.Background()
	var qa QueryAnswered
	cb := makeCallback("test", 17, ctx, 2, qa)

	assert.Regexp(t, "CALLBACK:closed=false, ref=test#[0-9]+ 17", cb.String())
}
