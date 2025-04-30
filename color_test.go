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

func testImage(t *testing.T) *sourceImage {
	f, err := os.Open(testImageFile)
	require.Nil(t, err)
	defer f.Close()
	require.NotNil(t, f)
	png, _, err := image.Decode(f)
	require.Nil(t, err)
	require.NotNil(t, png)
	img, err := NewSourceImage(Options{}, 0, png)
	require.Nil(t, err)
	return &img
}

func TestNewPalette(t *testing.T) {
	t.Parallel()
	img := testImage(t)
	p, _, err := NewPalette(img, false, false)
	require.Nil(t, err)
	require.NotNil(t, p)

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
	darkgrey, err := p.FromColor(color.RGBA{0x7a, 0x7a, 0x7a, 0xff})
	assert.Nil(t, err)
	assert.Equal(t, C64Color(11), darkgrey.C64Color)

	p.Add(Color{Color: color.RGBA{0x7a, 0x7a, 0x10, 0xff}, C64Color: 9})
	assert.Equal(t, 16, p.NumColors())
}

func TestParseBPC(t *testing.T) {
	t.Parallel()
	img := testImage(t)
	p, _, err := NewPalette(img, false, false)
	require.Nil(t, err)
	require.NotNil(t, p)

	cc, err := p.ParseBPC("0,6,4,14")
	assert.Nil(t, err)
	assert.Len(t, cc, 4)

	cc, err = p.ParseBPC("0,-1,-1,14")
	assert.Nil(t, err)
	assert.Len(t, cc, 4)
	assert.Equal(t, C64Color(0), cc[0].C64Color)
	assert.Nil(t, cc[1])
	assert.Nil(t, cc[2])
	assert.Equal(t, C64Color(14), cc[3].C64Color)

	cc, err = p.ParseBPC("16,0,0,0")
	assert.NotNil(t, err)
	cc, err = p.ParseBPC("0,0,-2,0")
	assert.NotNil(t, err)
}
