package png2prg

import (
	"fmt"
	"image"
	"image/color"
	"io"
)

// https://csdb.dk/release/?id=3961 zootrope by clone/wd
// https://csdb.dk/release/?id=9863 bubbling panda
// https://csdb.dk/release/?id=2118 madonna by electric (herb's shot)
// http://unusedino.de/ec64/technical/aay/c64/gfxdrl1.htm
// http://unusedino.de/ec64/technical/aay/c64/gfxdrl0.htm
// https://codebase64.org/doku.php?id=base:c64_grafix_files_specs_list_v0.03

func (img *sourceImage) isMultiColorInterlace() bool {
	for y := 0; y < FullScreenHeight; y++ {
		for x := 0; x < FullScreenWidth; x += 2 {
			if img.colorAtXY(x, y) != img.colorAtXY(x+1, y) {
				return true
			}
		}
	}
	return false
}

func (img *sourceImage) SplitInterlace() (*image.RGBA, *image.RGBA, error) {
	new0 := image.NewRGBA(image.Rect(0, 0, FullScreenWidth, FullScreenHeight))
	new1 := image.NewRGBA(image.Rect(0, 0, FullScreenWidth, FullScreenHeight))
	for y := 0; y < FullScreenHeight; y++ {
		for x := 0; x < FullScreenWidth; x += 2 {
			rgb := img.colorAtXY(x, y)
			c := color.RGBA{R: rgb.R, G: rgb.G, B: rgb.B, A: 255}
			new0.Set(x, y, c)
			new0.Set(x+1, y, c)
			rgb = img.colorAtXY(x+1, y)
			c = color.RGBA{R: rgb.R, G: rgb.G, B: rgb.B, A: 255}
			new1.Set(x, y, c)
			new1.Set(x+1, y, c)
		}
	}
	return new0, new1, nil
}

func (c *converter) WriteInterlaceTo(w io.Writer) (n int64, err error) {
	if len(c.images) != 2 {
		return n, fmt.Errorf("interlaces requires exactly 2 images at this stage, not %d", len(c.images))
	}
	img0 := &c.images[0]
	img1 := &c.images[0]

	k0, err := img0.Koala()
	if err != nil {
		return n, fmt.Errorf("img0.Koala failed: %w", err)
	}
	k1, err := img1.InterlaceKoala(*img0)
	if err != nil {
		return n, fmt.Errorf("img1.InterlaceKoala failed: %w", err)
	}

	n2, err := writeData(w, defaultHeader(), k0.Bitmap[:], k0.ScreenColor[:], k0.D800Color[:])
	n += n2
	if err != nil {
		return n, fmt.Errorf("writeData failed: %w", err)
	}

	sharedScreenRAM := k0.ScreenColor == k1.ScreenColor
	sharedColorRAM := k0.D800Color == k1.D800Color
	if !c.opt.Quiet {
		fmt.Printf("sharedScreenRAM: %v sharedColorRAM: %v\n", sharedScreenRAM, sharedColorRAM)
	}

	if c.opt.Symbols {
		bm2 := int(BitmapAddress + n)
		bs2 := int(bm2 + 0x1f40)
		bc2 := int(bm2 + 0x1f40 + 1000)
		c.Symbols = []c64Symbol{
			{"bitmap1", BitmapAddress},
			{"screenram1", BitmapScreenRAMAddress},
			{"colorram1", BitmapColorRAMAddress},
			{"bitmap2", bm2},
			{"screenram2", bs2},
			{"colorram2", bc2},
			{"d020color", int(img0.borderColor.ColorIndex)},
			{"d021color", int(img0.backgroundColor.ColorIndex)},
		}
	}
	n2, err = writeData(w, defaultHeader(), k1.Bitmap[:], k1.ScreenColor[:], k1.D800Color[:])
	n += n2
	if err != nil {
		return n, fmt.Errorf("writeData failed: %w", err)
	}
	return n, nil
}

// InterlaceKoala returns the secondary Koala, with as many bitpairs/colors the same as the first image.
func (img *sourceImage) InterlaceKoala(first sourceImage) (Koala, error) {
	k := Koala{
		BackgroundColor: img.backgroundColor.ColorIndex,
		BorderColor:     img.borderColor.ColorIndex,
		SourceFilename:  img.sourceFilename,
		opt:             img.opt,
	}

	for char := 0; char < 1000; char++ {
		colorIndex1, colorIndex2, err := first.multiColorIndexes(sortColors(first.charColors[char]))
		if err != nil {
			return k, fmt.Errorf("multiColorIndexes failed: error in char %d: %w", char, err)
		}

		bitmapIndex := char * 8
		x, y := xyFromChar(char)

		for byteIndex := 0; byteIndex < 8; byteIndex++ {
			bmpbyte := byte(0)
			for pixel := 0; pixel < 8; pixel += 2 {
				rgb := img.colorAtXY(x+pixel, y+byteIndex)
				if bmppattern, ok := colorIndex1[rgb]; ok {
					bmpbyte = bmpbyte | (bmppattern << (6 - byte(pixel)))
				} else {
					if img.opt.Verbose {
						//log.Printf("rgb %v not found in char %d.", rgb, char)
						//x, y := xyFromChar(char)
						//log.Printf("x, y = %d, %d", x, y)
						//log.Printf("colorIndex1: %v", colorIndex1)
					}
				}
			}
			k.Bitmap[bitmapIndex+byteIndex] = bmpbyte
		}

		if _, ok := colorIndex2[1]; ok {
			k.ScreenColor[char] = colorIndex2[1] << 4
		}
		if _, ok := colorIndex2[2]; ok {
			k.ScreenColor[char] = k.ScreenColor[char] | colorIndex2[2]
		}
		if _, ok := colorIndex2[3]; ok {
			k.D800Color[char] = colorIndex2[3]
		}
	}
	return k, nil
}
