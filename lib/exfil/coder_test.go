package exfil_test

import (
	"testing"

	"github.com/jpicht/exfil/lib/exfil"
	"github.com/stretchr/testify/assert"
)

func TestFileEncoder(t *testing.T) {
	c, err := exfil.EncodeFile("coder_test.go")
	if err != nil {
		t.Error(err)
		return
	}

	var (
		id    uint32
		size  = 0
		bytes = 0
	)
	for fqdn := range c {
		p, err := exfil.Decode(fqdn)
		assert.Nil(t, err)
		assert.NotNil(t, p)
		if p.IsHeader() {
			size = int(p.Header().Size())
			id = p.Id()
			t.Logf("Recv File %08x '%s' (%d bytes)", p.Id(), p.Header().Name(), p.Header().Size())
		} else {
			assert.NotNil(t, c)
			assert.Equal(t, id, p.Id())
			bytes += len(p.Content().Data())
		}
	}

	assert.Equal(t, size, bytes)
}
