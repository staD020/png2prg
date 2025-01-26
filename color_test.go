package png2prg

import (
	"image"
	"image/color"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testImageFile = "testdata/the_sarge_timeout.png"

func testImage(t *testing.T) image.Image {
	f, err := os.Open(testImageFile)
	require.Nil(t, err)
	defer f.Close()
	require.NotNil(t, f)
	img, _, err := image.Decode(f)
	assert.Nil(t, err)
	assert.NotNil(t, img)
	return img
}

func TestNewPalette(t *testing.T) {
	t.Parallel()
	img := testImage(t)
	p, err := NewPalette(img, false)
	assert.Nil(t, err)
	assert.NotNil(t, p)

	type tc struct {
		col  C64Color
		want string
	}
	testCases := []tc{
		{0, "0,#000000"},
		{1, "1,#ffffff"},
		{2, "2,#b56148"},
		{3, "3,#99e6f9"},
		{4, "4,#c161c9"},
		{5, "5,#79d570"},
		{6, "6,#6049ed"},
		{7, "7,#f7ff6c"},
		{8, "8,#ba8620"},
		//		{9, "9,#000000"}, // unused color in test image
		{10, "10,#e79a84"},
		{11, "11,#7a7a7a"},
		{12, "12,#a8a8a8"},
		{13, "13,#c0ffb9"},
		{14, "14,#a28fff"},
		{15, "15,#d2d2d2"},
	}
	for _, testcol := range testCases {
		got, err := p.FromC64(testcol.col)
		assert.Nil(t, err)
		assert.Equal(t, testcol.want, got.String())
	}
	assert.Equal(t, 15, p.NumColors())
	cc := p.Colors()
	assert.Len(t, cc, 15)
	black, err := p.FromColor(color.RGBA{})
	assert.Nil(t, err)
	assert.Equal(t, C64Color(0), black.C64Color)
	darkgrey, err := p.FromColor(color.RGBA{0x7a, 0x7a, 0x7a, 0x01})
	assert.Nil(t, err)
	assert.Equal(t, C64Color(11), darkgrey.C64Color)

	p.Add(Color{Color: color.RGBA{0x7a, 0x7a, 0x10, 0x01}, C64Color: 9})
	assert.Equal(t, 16, p.NumColors())
}
