package png2prg

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"io"

	"github.com/staD020/TSCrunch"
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

	k0, err := img0.Koala()
	if err != nil {
		return n, fmt.Errorf("img0.Koala failed: %w", err)
	}
	k1, err := img1.InterlaceKoala(*img0)
	if err != nil {
		return n, fmt.Errorf("img1.InterlaceKoala failed: %w", err)
	}

	bgBorder := k0.BackgroundColor | k0.BorderColor<<4

	if !c.opt.Display {
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
		const memoffset = 0x57fe
		header := []byte{0x00, 0x58}
		header = append(header, k1.D800Color[:]...)
		header = zeroFill(header, 0x5c00-memoffset-len(header))
		header = append(header, k1.ScreenColor[:]...)
		header = zeroFill(header, 0x6000-memoffset-len(header))
		header = append(header, k0.Bitmap[:]...)
		header = append(header, bgBorder, 0, byte(c.opt.D016Offset))
		header = zeroFill(header, 0x8000-memoffset-len(header))
		header = append(header, k1.Bitmap[:]...)
		return writeData(w, header)
	}

	buf := &bytes.Buffer{}
	header := newHeader(multiColorInterlaceBitmap)
	if c.opt.IncludeSID == "" {
		header = zeroFill(header, BitmapAddress-0x7ff-len(header))
		header = append(header, k0.Bitmap[:]...)
		header = zeroFill(header, 0x4000-0x7ff-len(header))
		header = append(header, k1.ScreenColor[:]...)
		header = zeroFill(header, 0x4400-0x7ff-len(header))
		header = append(header, k1.D800Color[:]...)
		header = append(header, bgBorder)
		header = zeroFill(header, 0x5c00-0x7ff-len(header))
		header = append(header, k1.ScreenColor[:]...)
		header = zeroFill(header, 0x6000-0x7ff-len(header))
		header = append(header, k1.Bitmap[:]...)
		header = append(header, bgBorder, 0, byte(c.opt.D016Offset))
		n, err = writeData(buf, header)
		if err != nil {
			return n, fmt.Errorf("writeData failed: %w", err)
		}
	}
	if c.opt.IncludeSID != "" {
		s, err := sid.LoadSID(c.opt.IncludeSID)
		if err != nil {
			return n, fmt.Errorf("sid.LoadSID failed: %w", err)
		}
		header = injectSIDHeader(header, s)
		load := s.LoadAddress()
		switch {
		case int(load) < len(header)+0x7ff:
			return n, fmt.Errorf("sid LoadAddress %s is too low for sid %s", load, s)
		case load > 0xcff && load < 0x1fff:
			header = zeroFill(header, int(load)-0x7ff-len(header))
			header = append(header, s.RawBytes()...)
			if len(header) > BitmapAddress-0x7ff {
				return n, fmt.Errorf("sid memory overflow 0x%04x for sid %s", len(header)+0x7ff, s)
			}
			if !c.opt.Quiet {
				fmt.Printf("injected %q: %s\n", c.opt.IncludeSID, s)
			}
		case load < 0xe000:
			return n, fmt.Errorf("sid LoadAddress %s is causing memory overlap for sid %s", load, s)
		}
		header = zeroFill(header, BitmapAddress-0x7ff-len(header))
		header = append(header, k0.Bitmap[:]...)
		header = zeroFill(header, 0x4000-0x7ff-len(header))
		header = append(header, k1.ScreenColor[:]...)
		header = zeroFill(header, 0x4400-0x7ff-len(header))
		header = append(header, k1.D800Color[:]...)
		header = append(header, bgBorder)
		header = zeroFill(header, 0x5c00-0x7ff-len(header))
		header = append(header, k1.ScreenColor[:]...)
		header = zeroFill(header, 0x6000-0x7ff-len(header))
		header = append(header, k1.Bitmap[:]...)
		header = append(header, bgBorder, 0, byte(c.opt.D016Offset))
		if load >= 0xe000 {
			header = zeroFill(header, int(load)-0x7ff-len(header))
			header = append(header, s.RawBytes()...)
		}

		n, err = writeData(buf, header)
		if err != nil {
			return n, fmt.Errorf("writeData failed: %w", err)
		}
	}

	if c.opt.NoCrunch {
		m, err := w.Write(buf.Bytes())
		return int64(m), err
	}
	tscopt := TSCOptions
	if c.opt.Verbose {
		tscopt.QUIET = false
	}
	tsc, err := TSCrunch.New(tscopt, buf)
	if err != nil {
		return n, fmt.Errorf("tscrunch.New failed: %w", err)
	}
	if !c.opt.Quiet {
		fmt.Println("packing with TSCrunch...")
	}
	m, err := tsc.WriteTo(w)
	n += m
	if err != nil {
		return n, fmt.Errorf("tsc.WriteTo failed: %w", err)
	}
	return n, nil
}

// InterlaceKoala returns the secondary Koala, with as many bitpairs/colors the same as the first image.
// it also merges possibly missing colors into k.ScreenColor and k.D800Color, use those.
func (img *sourceImage) InterlaceKoala(first sourceImage) (Koala, error) {
	k := Koala{
		BackgroundColor: img.backgroundColor.ColorIndex,
		BorderColor:     img.borderColor.ColorIndex,
		SourceFilename:  img.sourceFilename,
		opt:             img.opt,
	}

	for char := 0; char < 1000; char++ {
		rgb2bitpair, bitpair2c64color, err := first.multiColorIndexes(sortColors(first.charColors[char]), false)
		if err != nil {
			return k, fmt.Errorf("multiColorIndexes failed: error in char %d: %w", char, err)
		}
		tempbpc := bitpairColors{255, 255, 255, 255}
		for k, v := range bitpair2c64color {
			tempbpc[k] = v
		}
		img.preferredBitpairColors = tempbpc
		secondIndex1, secondIndex2, err := img.multiColorIndexes(sortColors(img.charColors[char]), true)
		if err != nil {
			return k, fmt.Errorf("multiColorIndexes failed: error in char %d: %w", char, err)
		}

		for k, v := range secondIndex1 {
			if _, ok := rgb2bitpair[k]; !ok {
				rgb2bitpair[k] = v
				bitpair2c64color[v] = secondIndex2[v]
			}
		}

		bitmapIndex := char * 8
		x, y := xyFromChar(char)

		for byteIndex := 0; byteIndex < 8; byteIndex++ {
			bmpbyte := byte(0)
			for pixel := 0; pixel < 8; pixel += 2 {
				rgb := img.colorAtXY(x+pixel, y+byteIndex)
				if bmppattern, ok := rgb2bitpair[rgb]; ok {
					bmpbyte = bmpbyte | (bmppattern << (6 - byte(pixel)))
				}
			}
			k.Bitmap[bitmapIndex+byteIndex] = bmpbyte
		}

		if _, ok := bitpair2c64color[1]; ok {
			k.ScreenColor[char] = bitpair2c64color[1] << 4
		}
		if _, ok := bitpair2c64color[2]; ok {
			k.ScreenColor[char] |= bitpair2c64color[2]
		}
		if _, ok := bitpair2c64color[3]; ok {
			k.D800Color[char] = bitpair2c64color[3]
		}
	}
	return k, nil
}
