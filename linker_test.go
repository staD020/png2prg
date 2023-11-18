package png2prg

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLinker(t *testing.T) {
	start := Word(0x801)
	bin := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	t.Parallel()
	l := NewLinker(start, false)
	assert.NotNil(t, l)
	n, err := l.Write(bin)
	assert.Nil(t, err)
	assert.Equal(t, len(bin), n)
	assert.Equal(t, bin, l.Bytes())
	buf := new(bytes.Buffer)
	m, err := l.WriteTo(buf)
	assert.Nil(t, err)
	assert.Equal(t, int64(10), m)
	expect := start.Bytes()
	expect = append(expect, bin...)
	assert.Equal(t, expect, buf.Bytes())
}
