package main

import (
	"fmt"
	"log"
	"math"
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
	if img.graphicsType == singleColorBitmap || img.graphicsType == singleColorCharset {
		bitpairs = []byte{0, 1}
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
		numColors, colorIndexes := img.countColors()
		if numColors <= 4 {
			img.preferredBitpairColors = colorIndexes
		}
	}

	for char := 0; char < 1000; char++ {
		colorIndex1, colorIndex2, err := img.multiColorIndexes(sortColors(img.charColors[char]))
		if err != nil {
			return k, fmt.Errorf("error in char %d: %v", char, err)
		}

		bitmapIndex := char * 8
		imageXIndex := img.xOffset + (int(math.Mod(float64(char), 40)) * 8)
		imageYIndex := img.yOffset + (int(math.Floor(float64(char)/40)) * 8)

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
		imageXIndex := img.xOffset + (int(math.Mod(float64(char), 40)) * 8)
		imageYIndex := img.yOffset + (int(math.Floor(float64(char)/40)) * 8)

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

	for char := 0; char < 256; char++ {

		bitmapIndex := char * 8
		imageXIndex := img.xOffset + (int(math.Mod(float64(char), 40)) * 8)
		imageYIndex := img.yOffset + (int(math.Floor(float64(char)/40)) * 8)

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
	}

	return c, nil
}

func (img *sourceImage) convertToMultiColorCharset() (c MultiColorCharset, err error) {
	c.SourceFilename = img.sourceFilename
	type charBytes [8]byte
	charMap := []charBytes{}

	_, palette := img.maxColorsPerChar()
	cc := sortColors(palette)
	// we must sort reverse to avoid a high color in bit 11
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
			log.Println("the bitpair 11 can only contain colors 0-7, singlecolor-mixed mode is not supported, you may want to condider using -bitpair-colors")
		}
	}

	c.CharColor = colorIndex2[3]
	c.BgColor = colorIndex2[0]
	c.D022Color = colorIndex2[1]
	c.D023Color = colorIndex2[2]

	for char := 0; char < 1000; char++ {

		//bitmapIndex := char * 8
		imageXIndex := img.xOffset + (int(math.Mod(float64(char), 40)) * 8)
		imageYIndex := img.yOffset + (int(math.Floor(float64(char)/40)) * 8)

		cbuf := charBytes{}
		for byteIndex := 0; byteIndex < 8; byteIndex++ {
			bmpbyte := byte(0)
			bmppattern := byte(0)
			for pixel := 0; pixel < 4; pixel++ {
				r, g, b, _ := img.image.At(imageXIndex+(pixel*2), imageYIndex+byteIndex).RGBA()
				rgb := RGB{byte(r), byte(g), byte(b)}
				bmppattern = colorIndex1[rgb]
				bmpbyte = bmpbyte | (bmppattern << (6 - (byte(pixel) * 2)))
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
		return c, fmt.Errorf("image translates to %d unique chars, the max is 256.", len(charMap))
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
