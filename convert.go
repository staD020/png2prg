package png2prg

import (
	"fmt"
	"log"
	"slices"
	"sort"
)

// sortColors sorts the colors by c64 colorindex.
func sortColors(charColors PaletteMap) (cc []ColorInfo) {
	for rgb, colorIndex := range charColors {
		cc = append(cc, ColorInfo{RGB: rgb, ColorIndex: colorIndex})
	}
	sort.Slice(cc, func(i, j int) bool {
		return cc[i].ColorIndex < cc[j].ColorIndex
	})
	return cc
}

// In returns true if element v is equal to an element of slice s.
func In[S ~[]E, E comparable](s S, v E) bool {
	return slices.Index(s, v) >= 0
}

// multiColorIndexes return rgb2bitpair and bitpait2c64color maps.
// It is the main function taking care of bitpair/color sorting, according to img.preferredBitpairColors.
// forcePreferred is used with interlaced pictures.
func (img *sourceImage) multiColorIndexes(char int, cc []ColorInfo, forcePreferred bool) (rgb2bitpair PaletteMap, bitpair2c64color map[byte]byte, err error) {
	rgb2bitpair = make(PaletteMap)
	bitpair2c64color = make(map[byte]byte)
	if img.c64color2bitpairCache[char] == nil {
		img.c64color2bitpairCache[char] = make(map[byte]byte)
	}
	for c64col := 0; c64col < MaxColors; c64col++ {
		if img.c64colorBitpairCount[c64col] == nil {
			img.c64colorBitpairCount[c64col] = make(map[byte]int)
		}
	}

	// set background
	if img.graphicsType != singleColorBitmap {
		rgb2bitpair[img.backgroundColor.RGB] = 0
		bitpair2c64color[0] = img.backgroundColor.ColorIndex
	}
	// which bitpairs do we have left, default is multicolor
	bitpairs := []byte{1, 2, 3}
	switch img.graphicsType {
	case singleColorBitmap:
		bitpairs = []byte{0, 1}
	case singleColorCharset, singleColorSprites, ecmCharset:
		bitpairs = []byte{1}
	}

	if img.opt.Trd {
		x, y := xyFromChar(char)
		col := x / 8
		row := y / 8
		if (col > 1 && col < 38) && (row > 1 && row < 22) {
			rgb2bitpair[img.palette.RGB(img.preferredBitpairColors[1])] = byte(1)
			bitpair2c64color[byte(1)] = img.preferredBitpairColors[1]
			rgb2bitpair[img.palette.RGB(img.preferredBitpairColors[2])] = byte(2)
			bitpair2c64color[byte(2)] = img.preferredBitpairColors[2]
			bitpairs = []byte{3}

			for _, ci := range cc {
				if _, ok := rgb2bitpair[img.palette.RGB(ci.ColorIndex)]; ok {
					continue
				}
				if len(bitpairs) == 0 {
					return nil, nil, fmt.Errorf("too many colors, no bitpairs left")
				}
				rgb2bitpair[ci.RGB] = byte(3)
				bitpair2c64color[byte(3)] = ci.ColorIndex
				bitpairs = []byte{}
			}
			return rgb2bitpair, bitpair2c64color, nil
		}
		if len(img.preferredBitpairColors) >= 3 {
			// pretty ugly implementation to make sure the borders of the map have different screenram colors
			// this is not concurrency safe
			img.preferredBitpairColors[1], img.preferredBitpairColors[2] = img.preferredBitpairColors[2], img.preferredBitpairColors[1]
			defer func() {
				img.preferredBitpairColors[1], img.preferredBitpairColors[2] = img.preferredBitpairColors[2], img.preferredBitpairColors[1]
			}()
		}
	}

	if forcePreferred {
		// used for interlace
		if len(img.preferredBitpairColors) == 0 {
			return nil, nil, fmt.Errorf("you cannot forcePreferred without setting img.preferredBitpairColors")
		}
		// fill preferred
		for preferBitpair, preferColor := range img.preferredBitpairColors {
			if preferColor > 15 {
				continue
			}
			rgb2bitpair[img.palette.RGB(preferColor)] = byte(preferBitpair)
			bitpair2c64color[byte(preferBitpair)] = preferColor
			// remove bitpair
			for i := range bitpairs {
				if bitpairs[i] == byte(preferBitpair) {
					bitpairs = append(bitpairs[:i], bitpairs[i+1:]...)
					break
				}
			}
		}
		// fill used
		for _, ci := range cc {
			// already set as preferred?
			if _, ok := rgb2bitpair[img.palette.RGB(ci.ColorIndex)]; ok {
				continue
			}
			// find spot
			if len(bitpairs) == 0 {
				return nil, nil, fmt.Errorf("too many colors, no bitpairs left")
			}
			// take first spot
			var bitpair byte
			bitpair, bitpairs = bitpairs[0], bitpairs[1:]
			rgb2bitpair[ci.RGB] = bitpair
			bitpair2c64color[bitpair] = ci.ColorIndex
		}
		for bp, c64col := range bitpair2c64color {
			if c64col > 15 {
				continue
			}
			img.c64color2bitpairCache[char][c64col] = bp
			img.c64colorBitpairCount[c64col][bp]++
		}
		return rgb2bitpair, bitpair2c64color, nil
	}

	// prefill preferred and used colors
	if len(img.preferredBitpairColors) > 0 {
		for preferBitpair, preferColor := range img.preferredBitpairColors {
			if preferColor > 15 {
				continue
			}
		OUTER:
			for _, ci := range cc {
				if preferColor == ci.ColorIndex {
					rgb2bitpair[ci.RGB] = byte(preferBitpair)
					bitpair2c64color[byte(preferBitpair)] = preferColor

					for i := range bitpairs {
						if bitpairs[i] == byte(preferBitpair) {
							bitpairs = append(bitpairs[:i], bitpairs[i+1:]...)
							break OUTER
						}
					}
				}
			}
		}
	}
	// bitpair2c64color includes bgcol, which may not be used in the char.
	if len(bitpair2c64color) > len(cc) {
		for bp, c64col := range bitpair2c64color {
			img.c64color2bitpairCache[char][c64col] = bp
		}
		return rgb2bitpair, bitpair2c64color, nil
	}

	// try img.c64colorBitpairCount
	if char > 0 && !img.opt.NoBitpairCounters {
		for _, ci := range cc {
			if _, ok := rgb2bitpair[ci.RGB]; ok {
				continue
			}
			if len(bitpairs) == 0 {
				return nil, nil, fmt.Errorf("too many colors in char, no bitpairs left")
			}
			if len(bitpairs) > 1 {
				bitpair2c64col := img.c64colorBitpairCount[ci.ColorIndex]
				if len(bitpair2c64col) == 0 {
					continue
				}
				//log.Printf("char %d: cache for col %d could work %v, this char now %v", char, ci.ColorIndex, bitpair2c64col, rgb2bitpair)
				max := 0
				bitpair := byte(0)
				for bp, count := range bitpair2c64col {
					if count > max || (count == max && bp > bitpair) {
						bitpair = bp
						max = count
					}
				}
				if max == 0 {
					continue
				}
				for i := range bitpairs {
					if bitpairs[i] == bitpair {
						rgb2bitpair[ci.RGB] = bitpair
						bitpair2c64color[bitpair] = ci.ColorIndex
						bitpairs = append(bitpairs[:i], bitpairs[i+1:]...)
						if img.opt.VeryVerbose {
							log.Printf("char %d: bitpair counter cache hit for col %d with bitpair %d", char, ci.ColorIndex, bitpair)
						}
						break
					}
				}
			}
		}
	}

	// prefer reusing bitpaircolors of previous char
	if char > 0 && !img.opt.NoPrevCharColors {
	NEXTCOL:
		for _, ci := range cc {
			if _, ok := rgb2bitpair[ci.RGB]; ok {
				continue
			}
			if len(bitpairs) == 0 {
				return nil, nil, fmt.Errorf("too many colors in char, no bitpairs left")
			}
			if prevbitpair, ok := img.c64color2bitpairCache[char-1][ci.ColorIndex]; ok {
				for i, availbitpair := range bitpairs {
					if prevbitpair == availbitpair {
						rgb2bitpair[ci.RGB] = prevbitpair
						bitpair2c64color[prevbitpair] = ci.ColorIndex
						bitpairs = append(bitpairs[:i], bitpairs[i+1:]...)
						continue NEXTCOL
					}
				}
				if char >= 40 {
					if prevbitpair2, ok := img.c64color2bitpairCache[char-40][ci.ColorIndex]; ok {
						for i, availbitpair := range bitpairs {
							if prevbitpair2 == availbitpair {
								rgb2bitpair[ci.RGB] = prevbitpair2
								bitpair2c64color[prevbitpair2] = ci.ColorIndex
								bitpairs = append(bitpairs[:i], bitpairs[i+1:]...)
								continue NEXTCOL
							}
						}
					}
				}
				if img.opt.VeryVerbose {
					log.Printf("char %d: match for color %d not found prevbitpair %d (from bitpairs %v)", char, ci.ColorIndex, prevbitpair, bitpairs)
				}
			}
		}
	}

	// fill or replace missing colors
	for _, ci := range cc {
		if _, ok := rgb2bitpair[ci.RGB]; ok {
			continue
		}
		if len(bitpairs) == 0 {
			return nil, nil, fmt.Errorf("too many colors in char, no bitpairs left")
		}
		if img.opt.VeryVerbose {
			log.Printf("char %d: could not guess bitpair for col %d from bitpairs %v", char, ci.ColorIndex, bitpairs)
		}
		var bitpair byte
		//works for all general cases, but prefers bitpair 11 should be replaced first
		//bitpair, bitpairs = bitpairs[len(bitpairs)-1], bitpairs[:len(bitpairs)-1]
		//let's shift the first available one, to avoid taking bitpair 11 (d800)
		bitpair, bitpairs = bitpairs[0], bitpairs[1:]
		rgb2bitpair[ci.RGB] = bitpair
		bitpair2c64color[bitpair] = ci.ColorIndex
	}
	for bp, c64col := range bitpair2c64color {
		img.c64color2bitpairCache[char][c64col] = bp
		img.c64colorBitpairCount[c64col][bp]++
	}
	return rgb2bitpair, bitpair2c64color, nil
}

func (img *sourceImage) multiColorCharBytes(char int, rgb2bitpair PaletteMap) (charBytes, error) {
	b := charBytes{}
	x, y := xyFromChar(char)
	for i := 0; i < 8; i++ {
		bmpbyte := byte(0)
		for pixel := 0; pixel < 8; pixel += 2 {
			rgb := img.colorAtXY(x+pixel, y+i)
			if bitpair, ok := rgb2bitpair[rgb]; ok {
				bmpbyte = bmpbyte | (bitpair << (6 - byte(pixel)))
			} else {
				return b, fmt.Errorf("rgb %v not found char %d (x=%d y=%d)", rgb, char, x, y)
			}
		}
		b[i] = bmpbyte
	}
	return b, nil
}

func (img *sourceImage) singleColorCharBytes(char int, rgb2bitpair PaletteMap) (charBytes, error) {
	b := charBytes{}
	x, y := xyFromChar(char)
	for i := 0; i < 8; i++ {
		bmpbyte := byte(0)
		for pixel := 0; pixel < 8; pixel++ {
			rgb := img.colorAtXY(x+pixel, y+i)
			if bitpair, ok := rgb2bitpair[rgb]; ok {
				bmpbyte = bmpbyte | (bitpair << (7 - byte(pixel)))
			} else {
				return b, fmt.Errorf("rgb %v not found char %d (x=%d y=%d)", rgb, char, x, y)
			}
		}
		b[i] = bmpbyte
	}
	return b, nil
}

func (img *sourceImage) prefBitpair2C64Color() map[byte]byte {
	bitpair2c64color := map[byte]byte{}
	j := byte(0)
	for _, col := range img.preferredBitpairColors {
		if col < 16 {
			bitpair2c64color[j] = col
		}
		j++
	}
	return bitpair2c64color
}

func (img *sourceImage) guessFirstBitpair2C64Color() map[byte]byte {
	for char := 0; char < FullScreenChars; char++ {
		x, y := xyFromChar(char)
		_, charBitpair2c64color, err := img.multiColorIndexes(char, sortColors(img.charColors[char]), false)
		if err != nil {
			log.Printf("multiColorIndexes failed: error in char %d (x=%d y=%d): %v", char, x, y, err)
			continue
		}
		if len(charBitpair2c64color) == 4 {
			if img.opt.Verbose {
				log.Printf("guessFirstBitpair2C64Color from first 4col char %d (x=%d y=%d): %v", char, x, y, charBitpair2c64color)
			}
			return charBitpair2c64color
		}
	}
	return img.prefBitpair2C64Color()
}

// Koala converts the img to Koala and returns it.
func (img *sourceImage) Koala() (Koala, error) {
	k := Koala{
		BackgroundColor: img.backgroundColor.ColorIndex,
		BorderColor:     img.borderColor.ColorIndex,
		SourceFilename:  img.sourceFilename,
		opt:             img.opt,
	}
	for col := byte(0); col < MaxColors; col++ {
		img.c64colorBitpairCount[col] = map[byte]int{}
	}

	if len(img.preferredBitpairColors) == 0 {
		numColors, colorIndexes, _ := img.countColors()
		if numColors <= 4 {
			img.preferredBitpairColors = colorIndexes
			if img.opt.Verbose {
				log.Printf("detected %d unique colors, assuming preferredBitpairColors %v", numColors, colorIndexes)
			}
		}
	}

	prevbitpair2c64color := img.guessFirstBitpair2C64Color()
	for char := 0; char < FullScreenChars; char++ {
		x, y := xyFromChar(char)
		rgb2bitpair, bitpair2c64color, err := img.multiColorIndexes(char, sortColors(img.charColors[char]), false)
		if err != nil {
			return k, fmt.Errorf("multiColorIndexes failed: error in char %d (x=%d y=%d): %w", char, x, y, err)
		}

		cbuf, err := img.multiColorCharBytes(char, rgb2bitpair)
		if err != nil {
			return k, fmt.Errorf("multiColorCharBytes failed: error in char %d (x=%d y=%d): %w", char, x, y, err)
		}
		for i := range cbuf {
			k.Bitmap[char*8+i] = cbuf[i]
		}

		if _, ok := bitpair2c64color[1]; ok {
			k.ScreenColor[char] = bitpair2c64color[1] << 4
		} else {
			if !k.opt.Trd {
				k.ScreenColor[char] = prevbitpair2c64color[1] << 4
			}
		}
		if _, ok := bitpair2c64color[2]; ok {
			k.ScreenColor[char] |= bitpair2c64color[2]
		} else {
			if !k.opt.Trd {
				k.ScreenColor[char] |= prevbitpair2c64color[2]
			}
		}
		if _, ok := bitpair2c64color[3]; ok {
			k.D800Color[char] = bitpair2c64color[3]
		} else {
			k.D800Color[char] = prevbitpair2c64color[3]
		}
		for k, v := range bitpair2c64color {
			prevbitpair2c64color[k] = v
		}
	}
	if img.opt.VeryVerbose {
		for c64col, bpcols := range img.c64colorBitpairCount {
			log.Printf("img.c64colorBitpairCount: col %d: %v", c64col, bpcols)
		}
	}
	return k, nil
}

// Hires converts the img to Hires and returns it.
func (img *sourceImage) Hires() (Hires, error) {
	h := Hires{
		SourceFilename: img.sourceFilename,
		BorderColor:    img.borderColor.ColorIndex,
		opt:            img.opt,
	}

	prevbitpair2c64color := img.prefBitpair2C64Color()
	for char := 0; char < FullScreenChars; char++ {
		x, y := xyFromChar(char)
		cc := sortColors(img.charColors[char])
		if len(cc) > 2 {
			return h, fmt.Errorf("Too many hires colors in char %d (x=%d y=%d)", char, x, y)
		}

		rgb2bitpair, bitpair2c64color, err := img.multiColorIndexes(char, cc, false)
		if err != nil {
			return h, fmt.Errorf("multiColorIndexes failed: error in char %d (x=%d y=%d): %w", char, x, y, err)
		}

		cbuf, err := img.singleColorCharBytes(char, rgb2bitpair)
		if err != nil {
			return h, fmt.Errorf("singleColorCharBytes failed: error in char %d (x=%d y=%d): %w", char, x, y, err)
		}
		for i := range cbuf {
			h.Bitmap[char*8+i] = cbuf[i]
		}

		if _, ok := bitpair2c64color[1]; ok {
			h.ScreenColor[char] = bitpair2c64color[1] << 4
		} else {
			h.ScreenColor[char] = prevbitpair2c64color[1] << 4
		}
		if _, ok := bitpair2c64color[0]; ok {
			h.ScreenColor[char] |= bitpair2c64color[0]
		} else {
			h.ScreenColor[char] |= prevbitpair2c64color[2]
		}
		for k, v := range bitpair2c64color {
			prevbitpair2c64color[k] = v
		}
	}
	return h, nil
}

type charBytes [8]byte

// SingleColorCharset converts the img to SingleColorCharset and returns it.
func (img *sourceImage) SingleColorCharset(prebuiltCharset []charBytes) (SingleColorCharset, error) {
	c := SingleColorCharset{
		SourceFilename: img.sourceFilename,
		BorderColor:    img.borderColor.ColorIndex,
		opt:            img.opt,
	}

	_, palette := img.maxColorsPerChar()
	cc := sortColors(palette)

	if len(img.preferredBitpairColors) == 0 {
		return c, fmt.Errorf("no bgcol? this should not happen.")
	}
	forceBgCol := int(img.preferredBitpairColors[0])

LOOP:
	for _, candidate := range img.backgroundCandidates {
		if candidate == byte(forceBgCol) {
			for i, col := range cc {
				if col.ColorIndex == byte(forceBgCol) {
					cc[0], cc[i] = cc[i], cc[0]
					if img.opt.VeryVerbose {
						log.Printf("forced background color %d was found", forceBgCol)
					}
					break LOOP
				}
			}
		}
	}
	if byte(forceBgCol) != cc[0].ColorIndex {
		return c, fmt.Errorf("forced background color %d was not found in (%v) with img.backgroundCandidates %s", forceBgCol, cc, img.backgroundCandidates)
	}

	rgb2bitpair := PaletteMap{}
	bitpair2c64color := map[byte]byte{}
	bit := byte(0)
	for _, ci := range cc {
		if bit > 1 {
			return c, fmt.Errorf("Too many colors.")
		}
		if _, ok := bitpair2c64color[bit]; !ok {
			rgb2bitpair[ci.RGB] = bit
			bitpair2c64color[bit] = ci.ColorIndex
		}
		bit++
	}

	c.BackgroundColor = bitpair2c64color[0]
	for i := 0; i < FullScreenChars; i++ {
		// disable for animations
		//c.D800Color[i] = bitpair2c64color[1]
	}

	if img.opt.NoPackChars {
		for char := 0; char < MaxChars; char++ {
			cbuf, err := img.singleColorCharBytes(char, rgb2bitpair)
			if err != nil {
				x, y := xyFromChar(char)
				return c, fmt.Errorf("singleColorCharBytes failed: error in char %d (x=%d y=%d): %w", char, x, y, err)
			}
			for i := range cbuf {
				c.Bitmap[char*8+i] = cbuf[i]
			}
			c.Screen[char] = byte(char)
		}
		return c, nil
	}

	charset := []charBytes{}
	if len(prebuiltCharset) > 0 {
		charset = prebuiltCharset
		if img.opt.VeryVerbose {
			log.Printf("using prebuiltCharset of %d chars", len(prebuiltCharset))
		}
	}

	truecount := make(map[charBytes]int, MaxChars)
	for char := 0; char < FullScreenChars; char++ {
		rgb2bitpair = PaletteMap{}
		bitpair2c64color = map[byte]byte{}
		for rgb, col := range img.charColors[char] {
			if col == cc[0].ColorIndex {
				rgb2bitpair[rgb] = 0
				bitpair2c64color[0] = col
			} else {
				rgb2bitpair[rgb] = 1
				bitpair2c64color[1] = col
				c.D800Color[char] = col
			}
		}

		cbuf, err := img.singleColorCharBytes(char, rgb2bitpair)
		if err != nil {
			x, y := xyFromChar(char)
			return c, fmt.Errorf("singleColorCharBytes failed: error in char %d (x=%d y=%d): %w", char, x, y, err)
		}
		if img.opt.ForcePackEmptyChar {
			emptyChar := charBytes{}
			if cbuf == emptyChar {
				cbuf = charBytes{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
				c.D800Color[char] = c.BackgroundColor
			}
		}
		truecount[cbuf]++
		curChar := slices.Index(charset, cbuf)
		if curChar < 0 {
			charset = append(charset, cbuf)
			curChar = len(charset) - 1
		}
		c.Screen[char] = byte(curChar)
	}

	if len(charset) > MaxChars {
		return c, fmt.Errorf("image packs to %d unique chars, the max is %d.", len(charset), MaxChars)
	}

	for i := range charset {
		for j := range charset[i] {
			c.Bitmap[i*8+j] = charset[i][j]
		}
	}
	if !img.opt.Quiet {
		fmt.Printf("used %d unique chars in the charset\n", len(truecount))
	}
	return c, nil
}

func romCharsetToCharBytes(romPrg []byte) (cb []charBytes) {
	buf := romPrg[2 : MaxChars*8+2]
	if len(buf)%8 != 0 {
		panic(fmt.Sprintf("romCharsetToCharBytes romPrg does not consist of 8 byte chars, %d %% 8 == %d", len(buf), len(buf)%8))
	}
	for i := 0; i < len(buf); i += 8 {
		c := charBytes{}
		for j := 0; j < 8; j++ {
			c[j] = buf[i+j]
		}
		cb = append(cb, c)
	}
	return cb
}

func (img *sourceImage) PETSCIICharset() (PETSCIICharset, error) {
	c := PETSCIICharset{
		SourceFilename: img.sourceFilename,
		BorderColor:    img.borderColor.ColorIndex,
		opt:            img.opt,
	}
	charset := romCharsetToCharBytes(romCharsetUppercasePrg)
	scc, err := img.SingleColorCharset(charset)
	if err == nil {
		c.Screen = scc.Screen
		c.D800Color = scc.D800Color
		c.BackgroundColor = scc.BackgroundColor
		return c, nil
	}
	charset = romCharsetToCharBytes(romCharsetLowercasePrg)
	scc, err = img.SingleColorCharset(charset)
	if err == nil {
		c.Screen = scc.Screen
		c.D800Color = scc.D800Color
		c.BackgroundColor = scc.BackgroundColor
		c.Lowercase = 1
		return c, nil
	}
	return c, err
}

// MultiColorCharset converts the img to MultiColorCharset and returns it.
func (img *sourceImage) MultiColorCharset(prebuiltCharset []charBytes) (c MultiColorCharset, err error) {
	c.SourceFilename = img.sourceFilename
	c.opt = img.opt
	_, palette := img.maxColorsPerChar()
	cc := sortColors(palette)
	// we must sort reverse to avoid a high color in bitpair 11
	sort.Slice(cc, func(i, j int) bool {
		return cc[i].ColorIndex > cc[j].ColorIndex
	})

	if len(img.preferredBitpairColors) == 0 {
		for _, v := range cc {
			img.preferredBitpairColors = append(img.preferredBitpairColors, v.ColorIndex)
		}
	}

	rgb2bitpair, bitpair2c64color, err := img.multiColorIndexes(0, cc, false)
	if err != nil {
		return c, fmt.Errorf("multiColorIndexes failed: %w", err)
	}

	if img.opt.Verbose {
		log.Printf("charset colors: %s\n", cc)
		log.Printf("rgb2bitpair: %v\n", rgb2bitpair)
		log.Printf("bitpair2c64color: %v\n", bitpair2c64color)
	}
	if bitpair2c64color[3] > 7 {
		if !img.opt.Quiet {
			return c, fmt.Errorf("the bitpair 11 can only contain colors 0-7, you will want to swap -bitpair-colors %s", img.preferredBitpairColors)
		}
	}

	c.CharColor = bitpair2c64color[3] | 8
	for i := 0; i < FullScreenChars; i++ {
		c.D800Color[i] = c.CharColor
	}
	c.BackgroundColor = bitpair2c64color[0]
	c.D022Color = bitpair2c64color[1]
	c.D023Color = bitpair2c64color[2]
	c.BorderColor = img.borderColor.ColorIndex

	if img.opt.NoPackChars {
		for char := 0; char < MaxChars; char++ {
			cbuf, err := img.multiColorCharBytes(char, rgb2bitpair)
			if err != nil {
				x, y := xyFromChar(char)
				return c, fmt.Errorf("multiColorCharBytes failed: error in char %d (x=%d y=%d): %w", char, x, y, err)
			}
			for i := range cbuf {
				c.Bitmap[char*8+i] = cbuf[i]
			}
			c.Screen[char] = byte(char)
		}
		return c, nil
	}

	charset := []charBytes{}
	if len(prebuiltCharset) > 0 {
		charset = prebuiltCharset
		if img.opt.VeryVerbose {
			log.Printf("using prebuiltCharset of %d chars", len(prebuiltCharset))
		}
	}
	for char := 0; char < FullScreenChars; char++ {
		cbuf, err := img.multiColorCharBytes(char, rgb2bitpair)
		if err != nil {
			x, y := xyFromChar(char)
			return c, fmt.Errorf("multiColorCharBytes failed: error in char %d (x=%d y=%d): %w", char, x, y, err)
		}
		if img.opt.ForcePackEmptyChar {
			emptyChar := charBytes{}
			if cbuf == emptyChar && c.BackgroundColor < 8 {
				cbuf = charBytes{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
				c.D800Color[char] = c.BackgroundColor
			}
		}
		curChar := slices.Index(charset, cbuf)
		if curChar < 0 {
			charset = append(charset, cbuf)
			curChar = len(charset) - 1
		}
		c.Screen[char] = byte(curChar)
	}

	if len(charset) > MaxChars {
		return c, fmt.Errorf("image packs to %d unique chars, the max is %d.", len(charset), MaxChars)
	}

	for i, bytes := range charset {
		for j, b := range bytes {
			c.Bitmap[i*8+j] = b
		}
	}
	if !img.opt.Quiet {
		fmt.Printf("used %d unique chars in the charset\n", len(charset))
	}
	return c, nil
}

func (img *sourceImage) MixedCharset(prebuiltCharset []charBytes) (c MixedCharset, err error) {
	c.SourceFilename = img.sourceFilename
	c.BorderColor = img.borderColor.ColorIndex
	c.opt = img.opt
	if img.opt.Verbose {
		log.Printf("img.MixedCharset: preferredBitpairColors: %v", img.preferredBitpairColors)
	}

	if len(img.preferredBitpairColors) > 3 {
		if img.preferredBitpairColors[3] > 7 {
			if img.opt.Verbose {
				log.Printf("img.MixedCharset: detected charcol %d > 7, attempting to swap with another bitpair", img.preferredBitpairColors[3])
			}
			fixed := false
			for i := 2; i > 0; i-- {
				//for i := 1; i < 3; i++ {
				if img.preferredBitpairColors[i] < 8 {
					img.preferredBitpairColors[i], img.preferredBitpairColors[3] = img.preferredBitpairColors[3], img.preferredBitpairColors[i]
					fixed = true
					break
				}
			}
			if !fixed {
				return c, fmt.Errorf("could not find charcol %d to swap, required in mixed mode. try alternate -bitpair-colors", img.preferredBitpairColors[3])
			}
		}
	}

	if len(img.backgroundCandidates) >= 0 {
		candidates := []byte{}
		for _, col := range img.backgroundCandidates {
			candidates = append(candidates, col)
		}
		sort.Slice(candidates, func(i, j int) bool { return candidates[i] > candidates[j] })
		if img.opt.Verbose {
			log.Printf("img.MixedCharset: candidates: %v", candidates)
		}

		fixpref := bitpairColors{}
		for _, p := range img.preferredBitpairColors {
			if In(candidates, p) && len(fixpref) < 3 {
				fixpref = append(fixpref, p)
			}
		}
		if len(fixpref) < len(candidates) {
			for _, p := range candidates {
				if In(img.preferredBitpairColors, p) && !In(fixpref, p) && len(fixpref) < 3 {
					fixpref = append(fixpref, p)
				}
			}
		}
		if len(fixpref) < len(candidates) {
			for _, p := range candidates {
				if !In(fixpref, p) && len(fixpref) < 3 {
					fixpref = append(fixpref, p)
				}
			}
		}
		img.preferredBitpairColors = fixpref
	}

	if img.opt.Verbose {
		log.Printf("img.MixedCharset: preferredBitpairColors: %v", img.preferredBitpairColors)
	}
	if len(img.preferredBitpairColors) > 0 {
		c.BackgroundColor = img.preferredBitpairColors[0]
	}
	if len(img.preferredBitpairColors) > 1 {
		c.D022Color = img.preferredBitpairColors[1]
	}
	if len(img.preferredBitpairColors) > 2 {
		c.D023Color = img.preferredBitpairColors[2]
	}
	if len(img.preferredBitpairColors) > 3 {
		return c, fmt.Errorf("d800 color for mixed charsets are deterministic, please use max 3.")
	}

	charset := []charBytes{}
	if len(prebuiltCharset) > 0 {
		charset = prebuiltCharset
		if img.opt.VeryVerbose {
			log.Printf("using prebuiltCharset of %d chars", len(prebuiltCharset))
		}
	}
	for char := 0; char < FullScreenChars; char++ {
		rgb2bitpair := PaletteMap{
			img.palette.RGB(c.BackgroundColor): 0,
			img.palette.RGB(c.D022Color):       1,
			img.palette.RGB(c.D023Color):       2,
		}
		bitpair2c64color := map[byte]byte{
			0: c.BackgroundColor,
			1: c.D022Color,
			2: c.D023Color,
		}

		for rgb, col := range img.charColors[char] {
			if _, ok := rgb2bitpair[rgb]; !ok {
				rgb2bitpair[rgb] = 3
				bitpair2c64color[3] = col
				c.D800Color[char] = bitpair2c64color[3]
				break
			}
		}
		if len(bitpair2c64color) == 4 {
			c.D800Color[char] = bitpair2c64color[3]
		}

		hires := false
		hirespixels := false
		charcol := byte(0)
		x, y := xyFromChar(char)
		if len(img.charColors[char]) <= 2 {
			// could be hires
		LOOP:
			for y2 := 0; y2 < 8; y2++ {
				for x2 := 0; x2 < 8; x2 += 2 {
					if img.colorAtXY(x+x2, y+y2) != img.colorAtXY(x+x2+1, y+y2) {
						hirespixels = true
						break LOOP
					}
				}
			}
			if bgcol, ok := img.charColors[char][img.palette.RGB(c.BackgroundColor)]; ok {
				for _, col := range img.charColors[char] {
					if col != bgcol && col < 8 {
						hires = true
						charcol = col
						c.D800Color[char] = col
						rgb2bitpair = PaletteMap{
							img.palette.RGB(c.BackgroundColor): 0,
							img.palette.RGB(charcol):           1,
						}
						break
					}
				}
			}
		}

		if hirespixels && !hires {
			return c, fmt.Errorf("found hirespixels in char %d (x=%d y=%d), but colors are bad: %s please swap some -bitpair-colors %s", char, x, y, img.charColors[char], img.preferredBitpairColors)
		}

		var cbuf charBytes
		emptyChar := charBytes{}
		if hires {
			if img.opt.VeryVerbose {
				log.Printf("char %d (x=%d y=%d) seems to be hires, charcol %d img.charColors: %v, -bpc %s", char, x, y, charcol, img.charColors[char], img.preferredBitpairColors)
			}
			cbuf, err = img.singleColorCharBytes(char, rgb2bitpair)
			if err != nil {
				return c, fmt.Errorf("singleColorCharBytes failed: error in char %d (x=%d y=%d): %w", char, x, y, err)
			}
		} else {
			c.D800Color[char] |= 8
			cbuf, err = img.multiColorCharBytes(char, rgb2bitpair)
			if err != nil {
				x, y := xyFromChar(char)
				return c, fmt.Errorf("multiColorCharBytes failed: error in char %d (x=%d y=%d): %w", char, x, y, err)
			}
		}

		if !img.opt.NoPackEmptyChar {
			if cbuf == emptyChar && c.BackgroundColor < 8 {
				cbuf = charBytes{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
				c.D800Color[char] = c.BackgroundColor
			}
		}
		curChar := slices.Index(charset, cbuf)
		if curChar < 0 {
			charset = append(charset, cbuf)
			curChar = len(charset) - 1
		}
		c.Screen[char] = byte(curChar)
	}

	if len(charset) > MaxChars {
		return c, fmt.Errorf("image packs to %d unique chars, the max is %d.", len(charset), MaxChars)
	}

	for i, bytes := range charset {
		for j, b := range bytes {
			c.Bitmap[i*8+j] = b
		}
	}
	if !img.opt.Quiet {
		fmt.Printf("settled for -bitpair-colors %s\n", img.preferredBitpairColors)
		fmt.Printf("used %d unique chars in the charset\n", len(charset))
	}

	return c, err
}

// ECMCharset converts the img to ECMCharset and returns it.
func (img *sourceImage) ECMCharset(prebuiltCharset []charBytes) (ECMCharset, error) {
	if len(img.ecmColors) < 4 {
		if img.opt.Verbose {
			log.Printf("not using all 4 img.ecmColors: %v", img.ecmColors)
		}
	}

	c := ECMCharset{
		SourceFilename:  img.sourceFilename,
		BorderColor:     img.borderColor.ColorIndex,
		BackgroundColor: img.ecmColors[0],
		opt:             img.opt,
	}
	if len(img.ecmColors) > 1 {
		c.D022Color = img.ecmColors[1]
	}
	if len(img.ecmColors) > 2 {
		c.D023Color = img.ecmColors[2]
	}
	if len(img.ecmColors) > 3 {
		c.D024Color = img.ecmColors[3]
	}

	charset := []charBytes{}
	if len(prebuiltCharset) > 0 && len(prebuiltCharset) <= MaxECMChars {
		charset = prebuiltCharset
		if img.opt.VeryVerbose {
			log.Printf("using prebuiltCharset of %d chars", len(prebuiltCharset))
		}
	}

	emptyChar := charBytes{}
	truecount := make(map[charBytes]int, MaxECMChars)
	for char := 0; char < FullScreenChars; char++ {
		x, y := xyFromChar(char)
		rgb2bitpair := PaletteMap{}
		orchar := byte(0)
		foundbg := false
		emptycharcol := byte(0)
		// when 2 ecm colors are used in the same char, which color to choose for bitpair 00?
		// good example: testdata/ecm/orion.png testdata/ecm/xpardey.png
		// so now we sort to at least make it deterministic.
		cc := sortColors(img.charColors[char])
		for _, v := range cc {
			i := slices.Index(img.ecmColors, v.ColorIndex)
			if i >= 0 && !foundbg {
				rgb2bitpair[v.RGB] = 0
				orchar = byte(i << 6)
				foundbg = true
				emptycharcol = v.ColorIndex
			} else {
				rgb2bitpair[v.RGB] = 1
				c.D800Color[char] = v.ColorIndex
			}
		}
		if len(img.charColors[char]) == 2 && !foundbg {
			return c, fmt.Errorf("background ecm color not found in char %d (x=%d y=%d)", char, x, y)
		}

		cbuf, err := img.singleColorCharBytes(char, rgb2bitpair)
		if err != nil {
			return c, fmt.Errorf("singleColorCharBytes failed: error in char %d (x=%d y=%d): %w", char, x, y, err)
		}
		if !img.opt.NoPackEmptyChar {
			if cbuf == emptyChar {
				// use bitpair 11 for empty chars, usually saves 1 char
				// good example: testdata/ecm/shampoo.png
				cbuf = charBytes{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
				c.D800Color[char] = emptycharcol
			}
		}

		truecount[cbuf]++
		curChar := slices.Index(charset, cbuf)
		if curChar < 0 {
			charset = append(charset, cbuf)
			curChar = len(charset) - 1
		}
		c.Screen[char] = byte(curChar) + orchar
	}

	if len(charset) > MaxECMChars {
		return c, fmt.Errorf("image packs to %d unique chars, the max is %d.", len(charset), MaxECMChars)
	}

	for i := range charset {
		for j := range charset[i] {
			c.Bitmap[i*8+j] = charset[i][j]
		}
	}
	if !img.opt.Quiet {
		fmt.Printf("used %d unique chars in the charset\n", len(truecount))
	}
	return c, nil
}

// SingleColorSprites converts the img to SingleColorSprites and returns it.
func (img *sourceImage) SingleColorSprites() (SingleColorSprites, error) {
	maxX := img.width / SpriteWidth
	maxY := img.height / SpriteHeight
	s := SingleColorSprites{
		SourceFilename: img.sourceFilename,
		Columns:        byte(maxX),
		Rows:           byte(maxY),
		opt:            img.opt,
	}
	if maxX == 0 || maxY == 0 {
		return s, fmt.Errorf("%d Xsprites x %d Ysprites: cant have 0 sprites", maxX, maxY)
	}

	cc := sortColors(img.palette)
	forceBgCol := -1
	if len(img.preferredBitpairColors) > 0 {
		forceBgCol = int(img.preferredBitpairColors[0])
	}
	if forceBgCol >= 0 {
		for i := range cc {
			if cc[i].ColorIndex == byte(forceBgCol) {
				cc[0], cc[i] = cc[i], cc[0]
				if img.opt.Verbose {
					log.Printf("forced background color %d was found", forceBgCol)
				}
				break
			}
		}
	}

	s.BackgroundColor = cc[0].ColorIndex
	if len(cc) > 1 {
		s.SpriteColor = cc[1].ColorIndex
	}

	rgb2bitpair := PaletteMap{}
	bitpair2c64color := map[byte]byte{}
	bit := byte(0)
	for _, ci := range cc {
		if bit > 1 {
			return s, fmt.Errorf("Too many colors.")
		}
		if _, ok := bitpair2c64color[bit]; !ok {
			rgb2bitpair[ci.RGB] = bit
			bitpair2c64color[bit] = ci.ColorIndex
		}
		bit++
	}

	if img.opt.Verbose {
		log.Printf("sprite colors: %v\n", cc)
		log.Printf("rgb2bitpair: %v\n", rgb2bitpair)
		log.Printf("bitpair2c64color: %v\n", bitpair2c64color)
	}

	for spriteY := 0; spriteY < maxY; spriteY++ {
		for spriteX := 0; spriteX < maxX; spriteX++ {
			for y := 0; y < SpriteHeight; y++ {
				yOffset := y + spriteY*SpriteHeight
				for x := 0; x < 3; x++ {
					xOffset := x*8 + spriteX*SpriteWidth
					bmpbyte := byte(0)
					for pixel := 0; pixel < 8; pixel++ {
						rgb := img.colorAtXY(xOffset+pixel, yOffset)
						if bitpair, ok := rgb2bitpair[rgb]; ok {
							bmpbyte = bmpbyte | (bitpair << (7 - byte(pixel)))
						} else {
							return s, fmt.Errorf("rgb %v not found in x %d, u %d.", rgb, x, y)
						}
					}
					s.Bitmap = append(s.Bitmap, bmpbyte)
				}
			}
			s.Bitmap = append(s.Bitmap, 0)
		}
	}
	if !img.opt.Quiet {
		fmt.Printf("converted %d sprites\n", maxX*maxY)
	}

	return s, nil
}

// MultiColorSprites converts the img to MultiColorSprites and returns it.
func (img *sourceImage) MultiColorSprites() (MultiColorSprites, error) {
	s := MultiColorSprites{
		SourceFilename: img.sourceFilename,
		opt:            img.opt,
	}

	cc := sortColors(img.palette)
	if len(img.preferredBitpairColors) == 0 {
		for _, v := range cc {
			img.preferredBitpairColors = append(img.preferredBitpairColors, v.ColorIndex)
		}
	}

	rgb2bitpair, bitpair2c64color, err := img.multiColorIndexes(0, cc, false)
	if err != nil {
		return s, fmt.Errorf("multiColorIndexes failed: %v", err)
	}

	if img.opt.Verbose {
		log.Printf("sprite colors: %v\n", cc)
		log.Printf("rgb2bitpair: %v\n", rgb2bitpair)
		log.Printf("bitpair2c64color: %v\n", bitpair2c64color)
	}

	switch {
	case len(img.preferredBitpairColors) > 3:
		s.D026Color = img.preferredBitpairColors[3]
		fallthrough
	case len(img.preferredBitpairColors) > 2:
		s.SpriteColor = img.preferredBitpairColors[2]
		fallthrough
	case len(img.preferredBitpairColors) > 1:
		s.D025Color = img.preferredBitpairColors[1]
		fallthrough
	case len(img.preferredBitpairColors) > 0:
		s.BackgroundColor = img.preferredBitpairColors[0]
	}

	s.Columns = byte(img.width / SpriteWidth)
	s.Rows = byte(img.height / SpriteHeight)
	if s.Columns == 0 || s.Rows == 0 {
		return s, fmt.Errorf("%d Xsprites x %d Ysprites: cant have 0 sprites", s.Columns, s.Rows)
	}

	for spriteY := 0; spriteY < int(s.Rows); spriteY++ {
		for spriteX := 0; spriteX < int(s.Columns); spriteX++ {
			for y := 0; y < SpriteHeight; y++ {
				yOffset := y + spriteY*SpriteHeight
				for x := 0; x < 3; x++ {
					xOffset := x*8 + spriteX*SpriteWidth
					bmpbyte := byte(0)
					for pixel := 0; pixel < 8; pixel += 2 {
						rgb := img.colorAtXY(xOffset+pixel, yOffset)
						if bitpair, ok := rgb2bitpair[rgb]; ok {
							bmpbyte |= bitpair << (6 - byte(pixel))
						} else {
							return s, fmt.Errorf("rgb %v not found in x %d, u %d.", rgb, x, y)
						}
					}
					s.Bitmap = append(s.Bitmap, bmpbyte)
				}
			}
			s.Bitmap = append(s.Bitmap, 0)
		}
	}
	if !img.opt.Quiet {
		fmt.Printf("converted %d sprites\n", s.Columns*s.Rows)
	}
	return s, nil
}
