package main

import (
	"fmt"
	"log"
	"sort"
)

func sortColors(charColors map[RGB]byte) (cc []colorInfo) {
	i := 0
	for rgb, colorIndex := range charColors {
		cc = append(cc, colorInfo{RGB: rgb, ColorIndex: colorIndex})
		i++
	}
	sort.Slice(cc, func(i, j int) bool {
		return cc[i].ColorIndex < cc[j].ColorIndex
	})
	return cc
}

func (img *sourceImage) multiColorIndexes(cc []colorInfo) (map[RGB]byte, map[byte]byte, error) {
	// rgb to bitpair
	colorIndex1 := make(map[RGB]byte)
	// bitpair to colorindex
	colorIndex2 := make(map[byte]byte)

	// set background
	if img.graphicsType != singleColorBitmap {
		colorIndex1[img.backgroundColor.RGB] = byte(0)
		colorIndex2[byte(0)] = img.backgroundColor.ColorIndex
	}
	// which bitpairs do we have left
	bitpairs := []byte{1, 2, 3}
	if img.graphicsType == singleColorBitmap {
		bitpairs = []byte{0, 1}
	}
	if img.graphicsType == singleColorCharset || img.graphicsType == singleColorSprites {
		bitpairs = []byte{1}
	}

	// prefill preferred and used colors
	if len(img.preferredBitpairColors) > 0 {
		for preferBitpair, preferColor := range img.preferredBitpairColors {
			if preferColor < 0 {
				continue
			}
		OUTER:
			for _, ci := range cc {
				if preferColor == ci.ColorIndex {
					colorIndex1[ci.RGB] = byte(preferBitpair)
					colorIndex2[byte(preferBitpair)] = preferColor

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
		if _, ok := colorIndex1[ci.RGB]; !ok {
			// col not found
			if len(bitpairs) == 0 {
				return nil, nil, fmt.Errorf("too many colors in char, no bitpairs left")
			}
			var bitpair byte
			bitpair, bitpairs = bitpairs[len(bitpairs)-1], bitpairs[:len(bitpairs)-1]
			colorIndex1[ci.RGB] = bitpair
			colorIndex2[bitpair] = ci.ColorIndex
		}
	}
	return colorIndex1, colorIndex2, nil
}

func (img *sourceImage) convertToKoala() (Koala, error) {
	k := Koala{
		BackgroundColor: img.backgroundColor.ColorIndex,
		BorderColor:     img.borderColor.ColorIndex,
		SourceFilename:  img.sourceFilename,
	}

	if len(img.preferredBitpairColors) == 0 {
		numColors, colorIndexes, _ := img.countColors()
		if numColors <= 4 {
			img.preferredBitpairColors = colorIndexes
			if verbose {
				log.Printf("detected %d unique colors, assuming preferredBitpairColors %v", numColors, colorIndexes)
			}
		}
	}

	for char := 0; char < 1000; char++ {
		colorIndex1, colorIndex2, err := img.multiColorIndexes(sortColors(img.charColors[char]))
		if err != nil {
			return k, fmt.Errorf("error in char %d: %v", char, err)
		}

		bitmapIndex := char * 8
		imageXIndex, imageYIndex := img.xyOffsetFromChar(char)

		for byteIndex := 0; byteIndex < 8; byteIndex++ {
			bmpbyte := byte(0)
			for pixel := 0; pixel < 8; pixel += 2 {
				r, g, b, _ := img.image.At(imageXIndex+pixel, imageYIndex+byteIndex).RGBA()
				rgb := RGB{byte(r), byte(g), byte(b)}
				if bmppattern, ok := colorIndex1[rgb]; ok {
					bmpbyte = bmpbyte | (bmppattern << (6 - byte(pixel)))
				} else {
					if verbose {
						log.Printf("rgb %v not found in char %d.", rgb, char)
						x, y := xyFromChar(char)
						log.Printf("x, y = %d, %d", x, y)
						log.Printf("colorIndex1: %v", colorIndex1)
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

func (img *sourceImage) convertToHires() (Hires, error) {
	h := Hires{
		SourceFilename: img.sourceFilename,
		BorderColor:    img.borderColor.ColorIndex,
	}

	for char := 0; char < 1000; char++ {
		cc := sortColors(img.charColors[char])
		if len(cc) > 2 {
			return h, fmt.Errorf("Too many hires colors in char %v.", char)
		}

		colorIndex1, colorIndex2, err := img.multiColorIndexes(cc)
		if err != nil {
			return h, fmt.Errorf("error in char %d: %v", char, err)
		}

		bitmapIndex := char * 8
		imageXIndex, imageYIndex := img.xyOffsetFromChar(char)

		for byteIndex := 0; byteIndex < 8; byteIndex++ {
			bmpbyte := byte(0)
			for pixel := 0; pixel < 8; pixel++ {
				r, g, b, _ := img.image.At(imageXIndex+pixel, imageYIndex+byteIndex).RGBA()
				rgb := RGB{byte(r), byte(g), byte(b)}
				if bmppattern, ok := colorIndex1[rgb]; ok {
					bmpbyte = bmpbyte | (bmppattern << (7 - byte(pixel)))
				} else {
					if verbose {
						log.Printf("rgb: %v not found in char: %d.", rgb, char)
						x, y := xyFromChar(char)
						log.Printf("x, y = %d, %d", x, y)
						log.Printf("colorIndex1: %v", colorIndex1)
					}
				}
			}
			h.Bitmap[bitmapIndex+byteIndex] = bmpbyte
		}

		if _, ok := colorIndex2[1]; ok {
			h.ScreenColor[char] = colorIndex2[1] << 4
		}
		if _, ok := colorIndex2[0]; ok {
			h.ScreenColor[char] = h.ScreenColor[char] | colorIndex2[0]
		}
	}
	return h, nil
}

func (img *sourceImage) convertToSingleColorCharset() (SingleColorCharset, error) {
	c := SingleColorCharset{
		SourceFilename: img.sourceFilename,
		BorderColor:    img.borderColor.ColorIndex,
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
				if verbose {
					log.Printf("forced background color %d was found", forceBgCol)
				}
				break
			}
		}
	}

	colorIndex1 := map[RGB]byte{}
	colorIndex2 := map[byte]byte{}
	bit := byte(0)
	for _, ci := range cc {
		if bit > 1 {
			return c, fmt.Errorf("Too many colors.")
		}
		if _, ok := colorIndex2[bit]; !ok {
			colorIndex1[ci.RGB] = bit
			colorIndex2[bit] = ci.ColorIndex
		}
		bit++
	}

	c.CharColor = colorIndex2[1]
	c.BackgroundColor = colorIndex2[0]

	if noPackChars {
		for char := 0; char < 256; char++ {
			bitmapIndex := char * 8
			imageXIndex, imageYIndex := img.xyOffsetFromChar(char)
			for byteIndex := 0; byteIndex < 8; byteIndex++ {
				bmpbyte := byte(0)
				for pixel := 0; pixel < 8; pixel++ {
					r, g, b, _ := img.image.At(imageXIndex+pixel, imageYIndex+byteIndex).RGBA()
					rgb := RGB{byte(r), byte(g), byte(b)}
					if bmppattern, ok := colorIndex1[rgb]; ok {
						bmpbyte = bmpbyte | (bmppattern << (7 - byte(pixel)))
					} else {
						if verbose {
							log.Printf("rgb %v not found in char %d.", rgb, char)
							x, y := xyFromChar(char)
							log.Printf("x, y = %d, %d", x, y)
							log.Printf("colorIndex1: %v", colorIndex1)
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

	for char := 0; char < 1000; char++ {
		imageXIndex, imageYIndex := img.xyOffsetFromChar(char)
		cbuf := charBytes{}
		for byteIndex := 0; byteIndex < 8; byteIndex++ {
			bmpbyte := byte(0)
			for pixel := 0; pixel < 8; pixel++ {
				r, g, b, _ := img.image.At(imageXIndex+pixel, imageYIndex+byteIndex).RGBA()
				rgb := RGB{byte(r), byte(g), byte(b)}
				if bmppattern, ok := colorIndex1[rgb]; ok {
					bmpbyte = bmpbyte | (bmppattern << (7 - byte(pixel)))
				} else {
					if verbose {
						log.Printf("rgb %v not found in char %d.", rgb, char)
						x, y := xyFromChar(char)
						log.Printf("x, y = %d, %d", x, y)
						log.Printf("colorIndex1: %v", colorIndex1)
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

	if len(charMap) > 256 {
		return c, fmt.Errorf("image packs to %d unique chars, the max is 256.", len(charMap))
	}

	j := 0
	for _, bytes := range charMap {
		for _, b := range bytes {
			c.Bitmap[j] = b
			j++
		}
	}

	if verbose {
		log.Printf("used %d unique chars in the charset", j/8)
	}

	return c, nil
}

func (img *sourceImage) convertToMultiColorCharset() (c MultiColorCharset, err error) {
	c.SourceFilename = img.sourceFilename
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

	colorIndex1, colorIndex2, err := img.multiColorIndexes(cc)
	if err != nil {
		return c, fmt.Errorf("multiColorIndexes failed: %v", err)
	}

	if verbose {
		log.Printf("charset colors: %v\n", cc)
		log.Printf("colorIndex1: %v\n", colorIndex1)
		log.Printf("colorIndex2: %v\n", colorIndex2)
	}
	if colorIndex2[3] > 7 {
		if !quiet {
			log.Println("the bitpair 11 can only contain colors 0-7, singlecolor-mixed mode is not supported, you may want to consider using -bitpair-colors")
		}
	}

	type charBytes [8]byte
	charMap := []charBytes{}

	c.CharColor = colorIndex2[3] | 8
	c.BackgroundColor = colorIndex2[0]
	c.D022Color = colorIndex2[1]
	c.D023Color = colorIndex2[2]
	c.BorderColor = img.borderColor.ColorIndex

	if noPackChars {
		for char := 0; char < 256; char++ {
			bitmapIndex := char * 8
			imageXIndex, imageYIndex := img.xyOffsetFromChar(char)
			for byteIndex := 0; byteIndex < 8; byteIndex++ {
				bmpbyte := byte(0)
				for pixel := 0; pixel < 8; pixel += 2 {
					r, g, b, _ := img.image.At(imageXIndex+pixel, imageYIndex+byteIndex).RGBA()
					rgb := RGB{byte(r), byte(g), byte(b)}
					if bmppattern, ok := colorIndex1[rgb]; ok {
						bmpbyte |= bmppattern << (6 - byte(pixel))
					} else {
						if verbose {
							log.Printf("rgb %v not found in char %d.", rgb, char)
							x, y := xyFromChar(char)
							log.Printf("x, y = %d, %d", x, y)
							log.Printf("colorIndex1: %v", colorIndex1)
						}
					}
				}
				c.Bitmap[bitmapIndex+byteIndex] = bmpbyte
			}
			c.Screen[char] = byte(char)
		}
		return c, nil
	}

	for char := 0; char < 1000; char++ {
		imageXIndex, imageYIndex := img.xyOffsetFromChar(char)
		cbuf := charBytes{}
		for byteIndex := 0; byteIndex < 8; byteIndex++ {
			bmpbyte := byte(0)
			for pixel := 0; pixel < 8; pixel += 2 {
				r, g, b, _ := img.image.At(imageXIndex+pixel, imageYIndex+byteIndex).RGBA()
				rgb := RGB{byte(r), byte(g), byte(b)}
				if bmppattern, ok := colorIndex1[rgb]; ok {
					bmpbyte |= bmppattern << (6 - byte(pixel))
				} else {
					if verbose {
						log.Printf("rgb %v not found in char %d.", rgb, char)
						x, y := xyFromChar(char)
						log.Printf("x, y = %d, %d", x, y)
						log.Printf("colorIndex1: %v", colorIndex1)
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

	if len(charMap) > 256 {
		return c, fmt.Errorf("image packs to %d unique chars, the max is 256.", len(charMap))
	}

	j := 0
	for _, bytes := range charMap {
		for _, b := range bytes {
			c.Bitmap[j] = b
			j++
		}
	}

	if verbose {
		log.Printf("used %d unique chars in the charset", j/8)
	}

	return c, nil
}

func (img *sourceImage) convertToSingleColorSprites() (SingleColorSprites, error) {
	maxX := img.width / 24
	maxY := img.height / 21
	s := SingleColorSprites{
		SourceFilename: img.sourceFilename,
		Columns:        byte(maxX),
		Rows:           byte(maxY),
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
				if verbose {
					log.Printf("forced background color %d was found", forceBgCol)
				}
				break
			}
		}
	}

	s.BackgroundColor = cc[0].ColorIndex
	s.SpriteColor = cc[1].ColorIndex

	colorIndex1 := map[RGB]byte{}
	colorIndex2 := map[byte]byte{}
	bit := byte(0)
	for _, ci := range cc {
		if bit > 1 {
			return s, fmt.Errorf("Too many colors.")
		}
		if _, ok := colorIndex2[bit]; !ok {
			colorIndex1[ci.RGB] = bit
			colorIndex2[bit] = ci.ColorIndex
		}
		bit++
	}

	if verbose {
		log.Printf("sprite colors: %v\n", cc)
		log.Printf("colorIndex1: %v\n", colorIndex1)
		log.Printf("colorIndex2: %v\n", colorIndex2)
	}

	for spriteY := 0; spriteY < maxY; spriteY++ {
		for spriteX := 0; spriteX < maxX; spriteX++ {
			for y := 0; y < 21; y++ {
				yOffset := img.yOffset + y + spriteY*21
				for x := 0; x < 3; x++ {
					xOffset := img.xOffset + x*8 + spriteX*24
					bmpbyte := byte(0)
					for pixel := 0; pixel < 8; pixel++ {
						r, g, b, _ := img.image.At(xOffset+pixel, yOffset).RGBA()
						rgb := RGB{byte(r), byte(g), byte(b)}
						if bmppattern, ok := colorIndex1[rgb]; ok {
							bmpbyte = bmpbyte | (bmppattern << (7 - byte(pixel)))
						} else {
							if verbose {
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
	if verbose {
		log.Printf("converted %d sprites", maxX*maxY)
	}

	return s, nil
}

func (img *sourceImage) convertToMultiColorSprites() (MultiColorSprites, error) {
	s := MultiColorSprites{SourceFilename: img.sourceFilename}

	cc := sortColors(img.palette)
	if len(img.preferredBitpairColors) == 0 {
		for _, v := range cc {
			img.preferredBitpairColors = append(img.preferredBitpairColors, v.ColorIndex)
		}
	}

	colorIndex1, colorIndex2, err := img.multiColorIndexes(cc)
	if err != nil {
		return s, fmt.Errorf("multiColorIndexes failed: %v", err)
	}

	if verbose {
		log.Printf("sprite colors: %v\n", cc)
		log.Printf("colorIndex1: %v\n", colorIndex1)
		log.Printf("colorIndex2: %v\n", colorIndex2)
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

	s.Columns = byte(img.width / 24)
	s.Rows = byte(img.height / 21)
	if s.Columns == 0 || s.Rows == 0 {
		return s, fmt.Errorf("%d Xsprites x %d Ysprites: cant have 0 sprites", s.Columns, s.Rows)
	}

	for spriteY := 0; spriteY < int(s.Rows); spriteY++ {
		for spriteX := 0; spriteX < int(s.Columns); spriteX++ {
			for y := 0; y < 21; y++ {
				yOffset := img.yOffset + y + spriteY*21
				for x := 0; x < 3; x++ {
					xOffset := img.xOffset + x*8 + spriteX*24
					bmpbyte := byte(0)
					for pixel := 0; pixel < 8; pixel += 2 {
						r, g, b, _ := img.image.At(xOffset+(pixel), yOffset).RGBA()
						rgb := RGB{byte(r), byte(g), byte(b)}
						if bmppattern, ok := colorIndex1[rgb]; ok {
							bmpbyte |= bmppattern << (6 - byte(pixel))
						} else {
							if verbose {
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
	if verbose {
		log.Printf("converted %d sprites", s.Columns*s.Rows)
	}
	return s, nil
}
