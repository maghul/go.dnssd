package dnssd

import (
	"testing"

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
