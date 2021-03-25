package exfil_test

import (
	"testing"

	"github.com/jpicht/exfil/lib/exfil"
	"github.com/stretchr/testify/assert"
)

func TestPunyCodeMapping(t *testing.T) {
	assert.Equal(t, len(exfil.Mapping), len(exfil.ReverseMapping))
	for r, rr := range exfil.Mapping {
		rrr, ok := exfil.ReverseMapping[rr]
		assert.True(t, ok)
		assert.Equal(t, r, rrr)
	}
}
