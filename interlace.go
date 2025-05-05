package png2prg

import (
	"fmt"
	"image"
	"io"
	"log"
	"time"
)

// drazlace examples:
//
// https://csdb.dk/release/?id=3961 zootrope by clone/wd
// https://csdb.dk/release/?id=9863 bubbling panda
// https://csdb.dk/release/?id=2118 madonna by electric (herb's shot)
// http://unusedino.de/ec64/technical/aay/c64/gfxdrl1.htm
// http://unusedino.de/ec64/technical/aay/c64/gfxdrl0.htm
// https://codebase64.org/doku.php?id=base:c64_grafix_files_specs_list_v0.03

// SplitInterlace splits the img by even and odd pixels into 2 multicolor images.
func (img *sourceImage) SplitInterlace() (*image.RGBA, *image.RGBA) {
	new0 := image.NewRGBA(image.Rect(0, 0, FullScreenWidth, FullScreenHeight))
	new1 := image.NewRGBA(image.Rect(0, 0, FullScreenWidth, FullScreenHeight))
	for y := 0; y < FullScreenHeight; y++ {
		for x := 0; x < FullScreenWidth; x += 2 {
			c := img.At(x, y)
			new0.Set(x, y, c)
			new0.Set(x+1, y, c)
			c = img.At(x+1, y)
			new1.Set(x, y, c)
			new1.Set(x+1, y, c)
		}
	}
	return new0, new1
}

// WriteInterlaceTo converts the 2 images and writes the resulting .prg to w.
func (c *Converter) WriteInterlaceTo(w io.Writer) (n int64, err error) {
	if len(c.images) != 2 {
		return n, fmt.Errorf("interlaces requires exactly 2 images at this stage, not %d", len(c.images))
	}
	img0 := &c.images[0]
	img1 := &c.images[1]
	if img0.p.NumColors() < MaxColors {
		for _, v := range img1.p.Colors() {
			img0.p.Add(v)
		}
	}
	if img1.p.NumColors() < MaxColors {
		for _, v := range img0.p.Colors() {
			img1.p.Add(v)
		}
	}

	if c.opt.BruteForce {
		if err = c.BruteForceBitpairColors(multiColorBitmap, 4); err != nil {
			return 0, fmt.Errorf("BruteForceBitpairColors %q failed: %w", img0.sourceFilename, err)
		}
		if err = img0.setPreferredBitpairColors(); err != nil {
			return 0, fmt.Errorf("img.setPreferredBitpairColors %q failed: %w", c.opt.BitpairColorsString, err)
		}
		if err = img1.setPreferredBitpairColors(); err != nil {
			return 0, fmt.Errorf("img.setPreferredBitpairColors %q failed: %w", c.opt.BitpairColorsString, err)
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
		if err != nil {
			return 0, fmt.Errorf("img0.InterlaceKoala %q failed: %w", img0.sourceFilename, err)
		}
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
	link := NewLinker(0, c.opt.VeryVerbose)
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
					{"d020color", int(img0.border.C64Color)},
					{"d021color", int(img0.bg.C64Color)},
				}
			}
			_, err = link.WriteMap(LinkMap{
				0x5800: k1.D800Color[:],
				0x5c00: k1.ScreenColor[:],
				0x6000: k0.Bitmap[:],
				0x7f40: []byte{bgBorder, 0, byte(c.opt.D016Offset)},
				0x8000: k1.Bitmap[:],
			})
			if err != nil {
				return n, fmt.Errorf("link.WriteMap failed: %w", err)
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
		_, err = link.WriteMap(LinkMap{
			0x9c00: k0.ScreenColor[:],
			0x9fe8: []byte{bgBorder, byte(c.opt.D016Offset)},
			0xa000: k0.Bitmap[:],
			0xc000: k1.Bitmap[:],
			0xe000: k1.ScreenColor[:],
			0xe400: k1.D800Color[:],
		})
		if err != nil {
			return n, fmt.Errorf("link.WriteMap failed: %w", err)
		}
		return link.WriteTo(w)
	}

	if _, err = link.WritePrg(multiColorInterlaceBitmap.newHeader()); err != nil {
		return n, fmt.Errorf("link.WritePrg failed: %w", err)
	}
	link.SetByte(DisplayerSettingsStart+9, k0.opt.NoFadeByte())
	if !k0.opt.NoFade {
		link.Block(0x7f50, 0xc5b0)
	}
	_, err = link.WriteMap(LinkMap{
		BitmapAddress: k0.Bitmap[:],
		0x4000:        k0.ScreenColor[:],
		0x4400:        k1.D800Color[:],
		0x5c00:        k1.ScreenColor[:],
		0x6000:        k1.Bitmap[:],
		0x7f40:        []byte{bgBorder, 0, byte(c.opt.D016Offset)},
	})
	if err != nil {
		return n, fmt.Errorf("link.WriteMap failed: %w", err)
	}

	if err = injectSID(link, c.opt.IncludeSID, c.opt.Quiet); err != nil {
		return n, fmt.Errorf("injectSID failed: %w", err)
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
		BackgroundColor: byte(img0.bg.C64Color),
		BorderColor:     byte(img0.border.C64Color),
		SourceFilename:  img0.sourceFilename,
		opt:             img0.opt,
	}
	k1 = Koala{
		BackgroundColor: byte(img1.bg.C64Color),
		BorderColor:     byte(img1.border.C64Color),
		SourceFilename:  img1.sourceFilename,
		opt:             img1.opt,
	}
	sharedcolors = true
	chars := []int{}
	foundsharedcol := 0
	for char := 0; char < FullScreenChars; char++ {
		bp0, err := img0.newBitpairs(char, img0.charColors[char], false)
		if err != nil {
			return k0, k1, sharedcolors, fmt.Errorf("img0.newBitpairs failed: error in char %d: %w", char, err)
		}
		tempbpc := BPColors{nil, nil, nil, nil}
		for bitpair := range bp0.bitpair2color {
			col := bp0.bitpair2color[bitpair]
			tempbpc[bitpair] = &col
		}
		img1.bpc = tempbpc
		bp1, err := img1.newBitpairs(char, img1.charColors[char], true)
		if err != nil {
			sharedcolors = false
			chars = append(chars, char)

			// detected non-shared colors, let's find and force a common d800 color
			foundsharedcolinchar := false
			forcepreferred := BPColors{nil, nil, nil, nil}
		OUTER:
			for _, col0 := range img0.charColors[char] {
				if col0.C64Color == img1.bg.C64Color {
					continue
				}
				for _, col1 := range img1.charColors[char] {
					if col0.C64Color == col1.C64Color {
						foundsharedcol++
						foundsharedcolinchar = true
						if img1.bpc[3] != nil {
							forcepreferred = BPColors{&img1.bg, nil, nil, &col0}
						}
						break OUTER
					}
				}
			}
			if !foundsharedcolinchar {
				if len(img0.charColors[char]) == 4 && len(img1.charColors[char]) == 4 {
					return k0, k1, sharedcolors, fmt.Errorf("failed: no shared color found in char %d", char)
				}
			}

			img0.bpc = forcepreferred
			bp0, err = img0.newBitpairs(char, img0.charColors[char], true)
			if err != nil {
				return k0, k1, sharedcolors, fmt.Errorf("img0.newBitpairs failed: error in char %d: %w", char, err)
			}
			img1.bpc = forcepreferred
			bp1, err = img1.newBitpairs(char, img1.charColors[char], true)
			if err != nil {
				return k0, k1, sharedcolors, fmt.Errorf("img1.newBitpairs failed: error in char %d: %w", char, err)
			}
		}

		bitmapIndex := char * 8
		x, y := xyFromChar(char)

		for byteIndex := 0; byteIndex < 8; byteIndex++ {
			bmpbyte0 := byte(0)
			bmpbyte1 := byte(0)
			for pixel := 0; pixel < 8; pixel += 2 {
				col0 := img0.At(x+pixel, y+byteIndex)
				if bmppattern, ok := bp0.bitpair(col0); ok {
					bmpbyte0 = bmpbyte0 | (bmppattern << (6 - byte(pixel)))
				}
				col1 := img1.At(x+pixel, y+byteIndex)
				if bmppattern, ok := bp1.bitpair(col1); ok {
					bmpbyte1 = bmpbyte1 | (bmppattern << (6 - byte(pixel)))
				}
			}
			k0.Bitmap[bitmapIndex+byteIndex] = bmpbyte0
			k1.Bitmap[bitmapIndex+byteIndex] = bmpbyte1
		}

		if col, ok := bp0.color(1); ok {
			k0.ScreenColor[char] = byte(col.C64Color) << 4
		}
		if col, ok := bp0.color(2); ok {
			k0.ScreenColor[char] |= byte(col.C64Color)
		}
		if col, ok := bp0.color(3); ok {
			k0.D800Color[char] = byte(col.C64Color)
		}
		if col, ok := bp1.color(1); ok {
			k1.ScreenColor[char] = byte(col.C64Color) << 4
		}
		if col, ok := bp1.color(2); ok {
			k1.ScreenColor[char] |= byte(col.C64Color)
		}
		if col, ok := bp1.color(3); ok {
			k1.D800Color[char] = byte(col.C64Color)
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
