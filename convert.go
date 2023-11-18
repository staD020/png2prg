package png2prg

import (
	"fmt"
	"log"
	"slices"
	"sort"
)

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

func (img *sourceImage) multiColorIndexes(cc []ColorInfo, forcePreferred bool) (rgb2bitpair PaletteMap, bitpair2c64color map[byte]byte, err error) {
	rgb2bitpair = make(PaletteMap)
	bitpair2c64color = make(map[byte]byte)

	// set background
	if img.graphicsType != singleColorBitmap {
		rgb2bitpair[img.backgroundColor.RGB] = 0
		bitpair2c64color[0] = img.backgroundColor.ColorIndex
	}
	// which bitpairs do we have left, default is multicolor
	bitpairs := []byte{1, 2, 3}
	if img.graphicsType == singleColorBitmap {
		bitpairs = []byte{0, 1}
	}
	if img.graphicsType == singleColorCharset || img.graphicsType == singleColorSprites {
		bitpairs = []byte{1}
	}

	if forcePreferred {
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

	// fill or replace missing colors
	for _, ci := range cc {
		if _, ok := rgb2bitpair[ci.RGB]; !ok {
			if len(bitpairs) == 0 {
				return nil, nil, fmt.Errorf("too many colors in char, no bitpairs left")
			}
			var bitpair byte
			//works for all general cases, but prefers bitpair 11 should be replaced first
			//bitpair, bitpairs = bitpairs[len(bitpairs)-1], bitpairs[:len(bitpairs)-1]
			//let's shift the first available one, to avoid taking bitpair 11 (d800)
			bitpair, bitpairs = bitpairs[0], bitpairs[1:]
			rgb2bitpair[ci.RGB] = bitpair
			bitpair2c64color[bitpair] = ci.ColorIndex
		}
	}
	return rgb2bitpair, bitpair2c64color, nil
}

func (img *sourceImage) Koala() (Koala, error) {
	k := Koala{
		BackgroundColor: img.backgroundColor.ColorIndex,
		BorderColor:     img.borderColor.ColorIndex,
		SourceFilename:  img.sourceFilename,
		opt:             img.opt,
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

	for char := 0; char < FullScreenChars; char++ {
		rgb2bitpair, bitpair2c64color, err := img.multiColorIndexes(sortColors(img.charColors[char]), false)
		if err != nil {
			return k, fmt.Errorf("multiColorIndexes failed: error in char %d: %w", char, err)
		}

		bitmapIndex := char * 8
		x, y := xyFromChar(char)

		for byteIndex := 0; byteIndex < 8; byteIndex++ {
			bmpbyte := byte(0)
			for pixel := 0; pixel < 8; pixel += 2 {
				rgb := img.colorAtXY(x+pixel, y+byteIndex)
				if bitpair, ok := rgb2bitpair[rgb]; ok {
					bmpbyte = bmpbyte | (bitpair << (6 - byte(pixel)))
				} else {
					if img.opt.Verbose {
						//log.Printf("rgb %v not found in char %d.", rgb, char)
						//x, y := xyFromChar(char)
						//log.Printf("x, y = %d, %d", x, y)
						//log.Printf("rgb2bitpair: %v", rgb2bitpair)
					}
				}
			}
			k.Bitmap[bitmapIndex+byteIndex] = bmpbyte
		}

		if _, ok := bitpair2c64color[1]; ok {
			k.ScreenColor[char] = bitpair2c64color[1] << 4
		}
		if _, ok := bitpair2c64color[2]; ok {
			k.ScreenColor[char] = k.ScreenColor[char] | bitpair2c64color[2]
		}
		if _, ok := bitpair2c64color[3]; ok {
			k.D800Color[char] = bitpair2c64color[3]
		}
	}
	return k, nil
}

func (img *sourceImage) Hires() (Hires, error) {
	h := Hires{
		SourceFilename: img.sourceFilename,
		BorderColor:    img.borderColor.ColorIndex,
		opt:            img.opt,
	}

	for char := 0; char < FullScreenChars; char++ {
		cc := sortColors(img.charColors[char])
		if len(cc) > 2 {
			return h, fmt.Errorf("Too many hires colors in char %d", char)
		}

		rgb2bitpair, bitpair2c64color, err := img.multiColorIndexes(cc, false)
		if err != nil {
			return h, fmt.Errorf("multiColorIndexes failed: error in char %d: %v", char, err)
		}

		bitmapIndex := char * 8
		x, y := xyFromChar(char)

		for byteIndex := 0; byteIndex < 8; byteIndex++ {
			bmpbyte := byte(0)
			for pixel := 0; pixel < 8; pixel++ {
				rgb := img.colorAtXY(x+pixel, y+byteIndex)
				if bitpair, ok := rgb2bitpair[rgb]; ok {
					bmpbyte |= bitpair << (7 - byte(pixel))
				} else {
					if img.opt.Verbose {
						log.Printf("rgb: %v not found in char: %d.", rgb, char)
						log.Printf("x, y = %d, %d", x, y)
						log.Printf("rgb2bitpair: %v", rgb2bitpair)
					}
				}
			}
			h.Bitmap[bitmapIndex+byteIndex] = bmpbyte
		}

		if _, ok := bitpair2c64color[1]; ok {
			h.ScreenColor[char] = bitpair2c64color[1] << 4
		}
		if _, ok := bitpair2c64color[0]; ok {
			h.ScreenColor[char] |= bitpair2c64color[0]
		}
	}
	return h, nil
}

func (img *sourceImage) SingleColorCharset() (SingleColorCharset, error) {
	c := SingleColorCharset{
		SourceFilename: img.sourceFilename,
		BorderColor:    img.borderColor.ColorIndex,
		opt:            img.opt,
	}
	_, palette := img.maxColorsPerChar()
	cc := sortColors(palette)

	forceBgCol := -1
	if len(img.preferredBitpairColors) > 0 {
		forceBgCol = int(img.preferredBitpairColors[0])
	}

	if forceBgCol >= 0 {
		for i, col := range cc {
			if col.ColorIndex == byte(forceBgCol) {
				cc[0], cc[i] = cc[i], cc[0]
				if img.opt.Verbose {
					log.Printf("forced background color %d was found", forceBgCol)
				}
				break
			}
		}
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

	c.CharColor = bitpair2c64color[1]
	c.BackgroundColor = bitpair2c64color[0]

	if img.opt.NoPackChars {
		for char := 0; char < MaxChars; char++ {
			bitmapIndex := char * 8
			x, y := xyFromChar(char)
			for byteIndex := 0; byteIndex < 8; byteIndex++ {
				bmpbyte := byte(0)
				for pixel := 0; pixel < 8; pixel++ {
					rgb := img.colorAtXY(x+pixel, y+byteIndex)
					if bitpair, ok := rgb2bitpair[rgb]; ok {
						bmpbyte |= bitpair << (7 - byte(pixel))
					} else {
						if img.opt.Verbose {
							log.Printf("rgb %v not found in char %d.", rgb, char)
							log.Printf("x, y = %d, %d", x, y)
							log.Printf("rgb2bitpair: %v", rgb2bitpair)
						}
					}
				}
				c.Bitmap[bitmapIndex+byteIndex] = bmpbyte
			}
			c.Screen[char] = byte(char)
		}
		return c, nil
	}

	type charBytes [8]byte
	charMap := []charBytes{}

	for char := 0; char < FullScreenChars; char++ {
		cbuf := charBytes{}
		x, y := xyFromChar(char)
		for byteIndex := 0; byteIndex < 8; byteIndex++ {
			bmpbyte := byte(0)
			for pixel := 0; pixel < 8; pixel++ {
				rgb := img.colorAtXY(x+pixel, y+byteIndex)
				if bitpair, ok := rgb2bitpair[rgb]; ok {
					bmpbyte |= bitpair << (7 - byte(pixel))
				} else {
					if img.opt.Verbose {
						log.Printf("rgb %v not found in char %d.", rgb, char)
						log.Printf("x, y = %d, %d", x, y)
						log.Printf("rgb2bitpair: %v", rgb2bitpair)
					}
				}
			}
			cbuf[byteIndex] = bmpbyte
		}

		found := false
		curChar := 0
		for curChar = range charMap {
			if cbuf == charMap[curChar] {
				found = true
				break
			}
		}
		if !found {
			charMap = append(charMap, cbuf)
			curChar = len(charMap) - 1
		}
		c.Screen[char] = byte(curChar)
	}

	if len(charMap) > MaxChars {
		return c, fmt.Errorf("image packs to %d unique chars, the max is %d.", len(charMap), MaxChars)
	}

	for i := range charMap {
		for j := range charMap[i] {
			c.Bitmap[i*8+j] = charMap[i][j]
		}
	}
	if !img.opt.Quiet {
		fmt.Printf("used %d unique chars in the charset\n", len(charMap))
	}
	return c, nil
}

func (img *sourceImage) MultiColorCharset() (c MultiColorCharset, err error) {
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

	rgb2bitpair, bitpair2c64color, err := img.multiColorIndexes(cc, false)
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
			log.Println("the bitpair 11 can only contain colors 0-7, mixed sc/mc mode is not supported, you may want to consider using -bitpair-colors")
		}
	}

	type charBytes [8]byte
	charset := []charBytes{}

	c.CharColor = bitpair2c64color[3] | 8
	c.BackgroundColor = bitpair2c64color[0]
	c.D022Color = bitpair2c64color[1]
	c.D023Color = bitpair2c64color[2]
	c.BorderColor = img.borderColor.ColorIndex

	if img.opt.NoPackChars {
		for char := 0; char < MaxChars; char++ {
			bitmapIndex := char * 8
			x, y := xyFromChar(char)
			for byteIndex := 0; byteIndex < 8; byteIndex++ {
				bmpbyte := byte(0)
				for pixel := 0; pixel < 8; pixel += 2 {
					rgb := img.colorAtXY(x+pixel, y+byteIndex)
					if bitpair, ok := rgb2bitpair[rgb]; ok {
						bmpbyte |= bitpair << (6 - byte(pixel))
					} else {
						if img.opt.Verbose {
							log.Printf("rgb %v not found in char %d.", rgb, char)
							log.Printf("x, y = %d, %d", x, y)
							log.Printf("rgb2bitpair: %v", rgb2bitpair)
						}
					}
				}
				c.Bitmap[bitmapIndex+byteIndex] = bmpbyte
			}
			c.Screen[char] = byte(char)
		}
		return c, nil
	}

	for char := 0; char < FullScreenChars; char++ {
		x, y := xyFromChar(char)
		cbuf := charBytes{}
		for byteIndex := 0; byteIndex < 8; byteIndex++ {
			bmpbyte := byte(0)
			for pixel := 0; pixel < 8; pixel += 2 {
				rgb := img.colorAtXY(x+pixel, y+byteIndex)
				if bitpair, ok := rgb2bitpair[rgb]; ok {
					bmpbyte |= bitpair << (6 - byte(pixel))
				} else {
					if img.opt.Verbose {
						log.Printf("rgb %v not found in char %d.", rgb, char)
						log.Printf("x, y = %d, %d", x, y)
						log.Printf("rgb2bitpair: %v", rgb2bitpair)
					}
				}
			}
			cbuf[byteIndex] = bmpbyte
		}

		found := false
		curChar := 0
		for curChar = range charset {
			if cbuf == charset[curChar] {
				found = true
				break
			}
		}
		if !found {
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
		fmt.Printf("used %d unique chars in the charset", len(charset))
	}
	return c, nil
}

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
		for i, col := range cc {
			if col.ColorIndex == byte(forceBgCol) {
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
							if img.opt.Verbose {
								log.Printf("rgb %v not found.", rgb)
							}
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

	rgb2bitpair, bitpair2c64color, err := img.multiColorIndexes(cc, false)
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
							if img.opt.Verbose {
								log.Printf("rgb %v not found.", rgb)
							}
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
