package main

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
)

func parseBitPairColors(bp string) ([]byte, error) {
	var result []byte
	for _, v := range strings.Split(bp, ",") {
		i, err := strconv.Atoi(v)
		if err != nil {
			return result, fmt.Errorf("strconv.Atoi conversion of %q to integers failed: %v", bp, err)
		}
		if i < -1 || i > 15 {
			return result, fmt.Errorf("incorrect color %d", i)
		}
		result = append(result, byte(i))
	}
	return result, nil
}

func sortColors(charColors map[RGB]byte) (cc []colorInfo) {
	i := 0
	for rgb, colorIndex := range charColors {
		cc = append(cc, colorInfo{rgb: rgb, colorIndex: colorIndex})
		i++
	}
	sort.Slice(cc, func(i, j int) bool {
		return cc[i].colorIndex < cc[j].colorIndex
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
		colorIndex1[img.backgroundColor.rgb] = byte(0)
		colorIndex2[byte(0)] = img.backgroundColor.colorIndex
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
				if preferColor == ci.colorIndex {
					colorIndex1[ci.rgb] = byte(preferBitpair)
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
		if _, ok := colorIndex1[ci.rgb]; !ok {
			// col not found
			if len(bitpairs) == 0 {
				return nil, nil, fmt.Errorf("too many colors in char, no bitpairs left")
			}
			var bitpair byte
			bitpair, bitpairs = bitpairs[len(bitpairs)-1], bitpairs[:len(bitpairs)-1]
			colorIndex1[ci.rgb] = bitpair
			colorIndex2[bitpair] = ci.colorIndex
		}
	}
	return colorIndex1, colorIndex2, nil
}

func (img *sourceImage) convertToKoala() (Koala, error) {
	k := Koala{
		BgColor:        img.backgroundColor.colorIndex,
		SourceFilename: img.sourceFilename,
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
			bmppattern := byte(0)
			for pixel := 0; pixel < 4; pixel++ {
				r, g, b, _ := img.image.At(imageXIndex+(pixel*2), imageYIndex+byteIndex).RGBA()
				rgb := RGB{byte(r), byte(g), byte(b)}
				bmppattern = colorIndex1[rgb]
				bmpbyte = bmpbyte | (bmppattern << (6 - (byte(pixel) * 2)))
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
				bmppattern := colorIndex1[rgb]
				bmpbyte = bmpbyte | (bmppattern << (7 - byte(pixel)))
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
	}
	_, palette := img.maxColorsPerChar()
	cc := sortColors(palette)

	forceBgCol := -1
	if len(img.preferredBitpairColors) > 0 {
		forceBgCol = int(img.preferredBitpairColors[0])
	}

	if forceBgCol >= 0 {
		for i, col := range cc {
			if col.colorIndex == byte(forceBgCol) {
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
			colorIndex1[ci.rgb] = bit
			colorIndex2[bit] = ci.colorIndex
		}
		bit++
	}

	//c.CharColor = colorIndex2[1]
	//c.BgColor = colorIndex2[0]

	if noPackChars {
		for char := 0; char < 256; char++ {
			bitmapIndex := char * 8
			imageXIndex, imageYIndex := img.xyOffsetFromChar(char)
			for byteIndex := 0; byteIndex < 8; byteIndex++ {
				bmpbyte := byte(0)
				for pixel := 0; pixel < 8; pixel++ {
					r, g, b, _ := img.image.At(imageXIndex+pixel, imageYIndex+byteIndex).RGBA()
					rgb := RGB{byte(r), byte(g), byte(b)}
					bmppattern := colorIndex1[rgb]
					bmpbyte = bmpbyte | (bmppattern << (7 - byte(pixel)))
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
				bmppattern := colorIndex1[rgb]
				bmpbyte = bmpbyte | (bmppattern << (7 - byte(pixel)))
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
		return cc[i].colorIndex > cc[j].colorIndex
	})

	if len(img.preferredBitpairColors) == 0 {
		for _, v := range cc {
			img.preferredBitpairColors = append(img.preferredBitpairColors, v.colorIndex)
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
	c.BgColor = colorIndex2[0]
	c.D022Color = colorIndex2[1]
	c.D023Color = colorIndex2[2]

	if noPackChars {
		for char := 0; char < 256; char++ {
			bitmapIndex := char * 8
			imageXIndex, imageYIndex := img.xyOffsetFromChar(char)
			for byteIndex := 0; byteIndex < 8; byteIndex++ {
				bmpbyte := byte(0)
				for pixel := 0; pixel < 4; pixel++ {
					r, g, b, _ := img.image.At(imageXIndex+(pixel*2), imageYIndex+byteIndex).RGBA()
					rgb := RGB{byte(r), byte(g), byte(b)}
					bmppattern := colorIndex1[rgb]
					bmpbyte |= bmppattern << (6 - (byte(pixel) * 2))
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
			for pixel := 0; pixel < 4; pixel++ {
				r, g, b, _ := img.image.At(imageXIndex+(pixel*2), imageYIndex+byteIndex).RGBA()
				rgb := RGB{byte(r), byte(g), byte(b)}
				bmppattern := colorIndex1[rgb]
				bmpbyte |= bmppattern << (6 - (byte(pixel) * 2))
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
	s := SingleColorSprites{SourceFilename: img.sourceFilename}
	maxX := img.width / 24
	maxY := img.height / 21
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
			if col.colorIndex == byte(forceBgCol) {
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
			return s, fmt.Errorf("Too many colors.")
		}
		if _, ok := colorIndex2[bit]; !ok {
			colorIndex1[ci.rgb] = bit
			colorIndex2[bit] = ci.colorIndex
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
						bmppattern := colorIndex1[rgb]
						bmpbyte = bmpbyte | (bmppattern << (7 - byte(pixel)))
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
			img.preferredBitpairColors = append(img.preferredBitpairColors, v.colorIndex)
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
		s.BgColor = img.preferredBitpairColors[0]
	}

	maxX := img.width / 24
	maxY := img.height / 21
	if maxX == 0 || maxY == 0 {
		return s, fmt.Errorf("%d Xsprites x %d Ysprites: cant have 0 sprites", maxX, maxY)
	}

	for spriteY := 0; spriteY < maxY; spriteY++ {
		for spriteX := 0; spriteX < maxX; spriteX++ {
			for y := 0; y < 21; y++ {
				yOffset := img.yOffset + y + spriteY*21
				for x := 0; x < 3; x++ {
					xOffset := img.xOffset + x*8 + spriteX*24
					bmpbyte := byte(0)
					for pixel := 0; pixel < 4; pixel++ {
						r, g, b, _ := img.image.At(xOffset+(pixel*2), yOffset).RGBA()
						rgb := RGB{byte(r), byte(g), byte(b)}
						bmppattern := colorIndex1[rgb]
						bmpbyte |= bmppattern << (6 - (byte(pixel) * 2))
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
