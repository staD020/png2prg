package png2prg

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLinker(t *testing.T) {
	t.Parallel()
	start := Word(0x801)
	bin := []byte{1, 2, 3, 4, 5, 6, 7, 8}
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

	l = NewLinker(0, false)
	assert.NotNil(t, l)
	assert.Equal(t, Word(0xffff), l.StartAddress())
	assert.Equal(t, Word(0x0), l.EndAddress())
	n, err = l.Write(bin)
	assert.Nil(t, err)
	assert.Equal(t, len(bin), n)
	assert.Equal(t, Word(0x0), l.StartAddress())
	assert.Equal(t, Word(0x8), l.EndAddress())

	l = NewLinker(0xffff, false)
	assert.NotNil(t, l)
	assert.Equal(t, Word(0xffff), l.StartAddress())
	assert.Equal(t, Word(0x0), l.EndAddress())
	n, err = l.Write(bin[0:2])
	assert.NotNil(t, err)
	assert.Zero(t, n)
	n, err = l.Write(bin[0:1])
	assert.Nil(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, Word(0xffff), l.StartAddress())
	assert.Equal(t, Word(0x0), l.EndAddress())
}
