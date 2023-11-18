package png2prg

import (
	"fmt"
	"image"
	"image/color"
	"io"
	"log"
	"time"

	"github.com/staD020/sid"
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
	img1 := &c.images[1]

	if len(img0.palette) != len(img1.palette) {
		// sync both palettes
		if c.opt.Verbose {
			log.Printf("len(img0.palette): %d len(img1.palette): %d", len(img0.palette), len(img1.palette))
		}
		switch {
		case len(img0.palette) < len(img1.palette):
			for k, v := range img1.palette {
				if _, ok := img0.palette[k]; !ok {
					img0.palette[k] = v
				}
			}
		case len(img0.palette) > len(img1.palette):
			for k, v := range img0.palette {
				if _, ok := img1.palette[k]; !ok {
					img1.palette[k] = v
				}
			}
		}
	}

	k0, k1, sharedcolors, err := img1.InterlaceKoala(*img0)
	if err != nil {
		return n, fmt.Errorf("img1.InterlaceKoala failed: %w", err)
	}
	if sharedcolors {
		k0.D800Color = k1.D800Color
		k0.ScreenColor = k1.ScreenColor
	}
	if !sharedcolors {
		k0, k1, _, err = img0.InterlaceKoala(*img1)
	}
	sharedd800 := k0.D800Color == k1.D800Color
	sharedscreen := k0.ScreenColor == k1.ScreenColor
	sharedbitmap := k0.Bitmap == k1.Bitmap
	if !c.opt.Quiet {
		fmt.Printf("shared colorram: %v shared screenram: %v shared bitmap: %v\n", sharedd800, sharedscreen, sharedbitmap)
		if !sharedd800 && c.opt.Verbose {
			for i := range k0.D800Color {
				if k0.D800Color[i] != k1.D800Color[i] {
					fmt.Printf("char %d k0.D800Color %d k1.D800Color %d\n", i, k0.D800Color[i], k1.D800Color[i])
				}
			}
		}
	}

	bgBorder := k0.BackgroundColor | k0.BorderColor<<4
	link := NewLinker(0, c.opt.Verbose)
	if !c.opt.Display {
		if sharedcolors {
			// drazlace
			if c.opt.Symbols {
				c.Symbols = []c64Symbol{
					{"colorram1", 0x5800},
					{"screenram1", 0x5c00},
					{"bitmap1", 0x6000},
					{"d021coloraddr", 0x7f40},
					{"d016offsetaddr", 0x7f42},
					{"bitmap2", 0x8000},
					{"d016offset", c.opt.D016Offset},
					{"d020color", int(img0.borderColor.ColorIndex)},
					{"d021color", int(img0.backgroundColor.ColorIndex)},
				}
			}
			_, err = link.MapWrite(LinkMap{
				0x5800: k1.D800Color[:],
				0x5c00: k1.ScreenColor[:],
				0x6000: k0.Bitmap[:],
				0x7f40: []byte{bgBorder, 0, byte(c.opt.D016Offset)},
				0x8000: k1.Bitmap[:],
			})
			if err != nil {
				return n, fmt.Errorf("link.MapWrite failed: %w", err)
			}
			return link.WriteTo(w)
		}
		// true paint .mci format
		c.Symbols = []c64Symbol{
			{"screenram1", 0x9c00},
			{"d021coloraddr", 0x9fe8},
			{"d016offsetaddr", 0x9fe9},
			{"bitmap1", 0xa000},
			{"bitmap2", 0xc000},
			{"screenram2", 0xe000},
			{"colorram", 0xe400},
			{"d016offset", c.opt.D016Offset},
			{"d020color", int(k0.BorderColor)},
			{"d021color", int(k0.BackgroundColor)},
		}
		_, err = link.MapWrite(LinkMap{
			0x9c00: k0.ScreenColor[:],
			0x9fe8: []byte{bgBorder, byte(c.opt.D016Offset)},
			0xa000: k0.Bitmap[:],
			0xc000: k1.Bitmap[:],
			0xe000: k1.ScreenColor[:],
			0xe400: k1.D800Color[:],
		})
		if err != nil {
			return n, fmt.Errorf("link.MapWrite failed: %w", err)
		}
		return link.WriteTo(w)
	}

	link.Block(0x7f50, 0xc5b0)
	if _, err = link.WritePrg(newHeader(multiColorInterlaceBitmap)); err != nil {
		return n, fmt.Errorf("link.WritePrg failed: %w", err)
	}

	_, err = link.MapWrite(LinkMap{
		BitmapAddress: k0.Bitmap[:],
		0x4000:        k0.ScreenColor[:],
		0x4400:        k1.D800Color[:],
		0x5c00:        k1.ScreenColor[:],
		0x6000:        k1.Bitmap[:],
		0x7f40:        []byte{bgBorder, 0, byte(c.opt.D016Offset)},
	})
	if err != nil {
		return n, fmt.Errorf("link.MapWrite failed: %w", err)
	}

	if c.opt.IncludeSID != "" {
		s, err := sid.LoadSID(c.opt.IncludeSID)
		if err != nil {
			return n, fmt.Errorf("sid.LoadSID failed: %w", err)
		}
		if _, err = link.WritePrg(s.Bytes()); err != nil {
			return n, fmt.Errorf("link.WritePrg failed: %w", err)
		}
		injectSIDLinker(link, s)
		if !c.opt.Quiet {
			fmt.Printf("injected %q: %s\n", c.opt.IncludeSID, s)
		}
	}
	if c.opt.NoCrunch {
		return link.WriteTo(w)
	}

	t1 := time.Now()
	wt, err := injectCrunch(link, c.opt.Verbose)
	if err != nil {
		return n, fmt.Errorf("injectCrunch failed: %w", err)
	}
	n, err = wt.WriteTo(w)
	if err != nil {
		return n, err
	}
	if !c.opt.Quiet && c.opt.Display && !c.opt.NoCrunch {
		fmt.Printf("TSCrunched in %s\n", time.Since(t1))
	}

	return n, err
}

// InterlaceKoala returns the secondary Koala, with as many bitpairs/colors the same as the first image.
// it also merges possibly missing colors into k.ScreenColor and k.D800Color, use those.
func (img1 *sourceImage) InterlaceKoala(img0 sourceImage) (k0, k1 Koala, sharedcolors bool, err error) {
	k0 = Koala{
		BackgroundColor: img0.backgroundColor.ColorIndex,
		BorderColor:     img0.borderColor.ColorIndex,
		SourceFilename:  img0.sourceFilename,
		opt:             img0.opt,
	}
	k1 = Koala{
		BackgroundColor: img1.backgroundColor.ColorIndex,
		BorderColor:     img1.borderColor.ColorIndex,
		SourceFilename:  img1.sourceFilename,
		opt:             img1.opt,
	}
	sharedcolors = true
	chars := []int{}
	foundsharedcol := 0
	for char := 0; char < 1000; char++ {
		rgb2bitpair0, bitpair2c64color0, err := img0.multiColorIndexes(sortColors(img0.charColors[char]), false)
		if err != nil {
			return k0, k1, sharedcolors, fmt.Errorf("multiColorIndexes failed: error in char %d: %w", char, err)
		}
		tempbpc := bitpairColors{255, 255, 255, 255}
		for k, v := range bitpair2c64color0 {
			tempbpc[k] = v
		}
		img1.preferredBitpairColors = tempbpc
		rgb2bitpair1, bitpair2c64color1, err := img1.multiColorIndexes(sortColors(img1.charColors[char]), true)
		if err != nil {
			sharedcolors = false
			chars = append(chars, char)

			// detected non-shared colors, let's find and force a common d800 color
			foundsharedcolinchar := false
			forcepreferred := bitpairColors{255, 255, 255, 255}
		OUTER:
			for _, col0 := range img0.charColors[char] {
				if col0 == img1.backgroundColor.ColorIndex {
					continue
				}
				for _, col1 := range img1.charColors[char] {
					if col0 == col1 {
						foundsharedcol++
						foundsharedcolinchar = true
						//fmt.Printf("img0.charColors[%d]: %v img1.charColors[%d]: %v\n", char, img0.charColors[char], char, img1.charColors[char])
						if img1.preferredBitpairColors[3] < 16 {
							forcepreferred = bitpairColors{img1.backgroundColor.ColorIndex, 255, 255, col0}
						}
						break OUTER
					}
				}
			}
			if !foundsharedcolinchar {
				if len(img0.charColors[char]) == 4 && len(img1.charColors[char]) == 4 {
					//fmt.Printf("img0.charColors[%d]: %v img1.charColors[%d]: %v\n", char, img0.charColors[char], char, img1.charColors[char])
					return k0, k1, sharedcolors, fmt.Errorf("multiColorIndexes failed: no shared color found in char %d", char)
				}
			}

			img0.preferredBitpairColors = forcepreferred
			rgb2bitpair0, bitpair2c64color0, err = img0.multiColorIndexes(sortColors(img0.charColors[char]), true)
			if err != nil {
				return k0, k1, sharedcolors, fmt.Errorf("img0.multiColorIndexes failed: error in char %d: %w", char, err)
			}
			img1.preferredBitpairColors = forcepreferred
			rgb2bitpair1, bitpair2c64color1, err = img1.multiColorIndexes(sortColors(img1.charColors[char]), true)
			if err != nil {
				return k0, k1, sharedcolors, fmt.Errorf("img1.multiColorIndexes failed: error in char %d: %w", char, err)
			}
		}

		bitmapIndex := char * 8
		x, y := xyFromChar(char)

		for byteIndex := 0; byteIndex < 8; byteIndex++ {
			bmpbyte0 := byte(0)
			bmpbyte1 := byte(0)
			for pixel := 0; pixel < 8; pixel += 2 {
				rgb0 := img0.colorAtXY(x+pixel, y+byteIndex)
				if bmppattern, ok := rgb2bitpair0[rgb0]; ok {
					bmpbyte0 = bmpbyte0 | (bmppattern << (6 - byte(pixel)))
				}
				rgb1 := img1.colorAtXY(x+pixel, y+byteIndex)
				if bmppattern, ok := rgb2bitpair1[rgb1]; ok {
					bmpbyte1 = bmpbyte1 | (bmppattern << (6 - byte(pixel)))
				}
			}
			k0.Bitmap[bitmapIndex+byteIndex] = bmpbyte0
			k1.Bitmap[bitmapIndex+byteIndex] = bmpbyte1
		}

		if _, ok := bitpair2c64color0[1]; ok {
			k0.ScreenColor[char] = bitpair2c64color0[1] << 4
		}
		if _, ok := bitpair2c64color0[2]; ok {
			k0.ScreenColor[char] |= bitpair2c64color0[2]
		}
		if _, ok := bitpair2c64color0[3]; ok {
			k0.D800Color[char] = bitpair2c64color0[3]
		}
		if _, ok := bitpair2c64color1[1]; ok {
			k1.ScreenColor[char] = bitpair2c64color1[1] << 4
		}
		if _, ok := bitpair2c64color1[2]; ok {
			k1.ScreenColor[char] |= bitpair2c64color1[2]
		}
		if _, ok := bitpair2c64color1[3]; ok {
			k1.D800Color[char] = bitpair2c64color1[3]
		}

		// sync d800
		if k0.D800Color[char] == k0.BackgroundColor {
			k0.D800Color[char] = k1.D800Color[char]
		}
		if k1.D800Color[char] == k1.BackgroundColor {
			k1.D800Color[char] = k0.D800Color[char]
		}
	}
	if !sharedcolors && img1.opt.Verbose {
		log.Printf("cannot force the same screenram colors for %d chars", len(chars))
		log.Printf("found at least 1 shared col in %d chars", foundsharedcol)
	}
	return k0, k1, sharedcolors, nil
}
