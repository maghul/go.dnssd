package dnssd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFlagsToString(t *testing.T) {
	f := MoreComing
	assert.Equal(t, "MoreComing", f.String())

	f = MoreComing | Unique
	assert.Equal(t, "MoreComing | Unique", f.String())
}
