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
		if i < 0 || i > 15 {
			return result, fmt.Errorf("incorrect color %d", i)
		}
		result = append(result, byte(i))
	}
	return result, nil
}

func sortColors(charColors map[RGB]byte) []colorInfo {
	cc := make([]colorInfo, 4, 4)
	i := 0
	for rgb, colorIndex := range charColors {
		cc[i] = colorInfo{rgb: rgb, colorIndex: colorIndex}
		i++
	}
	sort.Slice(cc, func(i, j int) bool {
		return cc[i].colorIndex < cc[j].colorIndex
	})
	return cc
}

func (img *sourceImage) multiColorIndexes(cc []colorInfo) (map[RGB]byte, map[byte]byte, error) {
	// rgb to bitpair
	colorIndex1 := make(map[RGB]byte, 0)
	// bitpair to colorindex
	colorIndex2 := make(map[byte]byte, 0)

	if img.graphicsType == multiColorBitmap {
		colorIndex1[img.backgroundColor.rgb] = byte(0)
		colorIndex2[byte(0)] = img.backgroundColor.colorIndex
	}
	if len(img.preferredBitpairColors) == 0 {
		img.preferredBitpairColors = []byte{img.backgroundColor.colorIndex}
	}

	if img.preferredBitpairColors != nil {
		for i, col := range cc {
			for j, preferColor := range img.preferredBitpairColors {
				if preferColor < 0 {
					continue
				}
				if col.colorIndex == preferColor && i != j {
					cc[i], cc[j] = cc[j], cc[i]
					break
				}
			}
		}
	}

	for i, ci := range cc {
		if i > 3 {
			return colorIndex1, colorIndex2, fmt.Errorf("Too many colors in char")
		}
		if _, ok := colorIndex2[byte(i)]; ok {
			continue
		}
		if _, ok := colorIndex1[ci.rgb]; !ok {
			colorIndex1[ci.rgb] = byte(i)
			colorIndex2[byte(i)] = ci.colorIndex
		}
	}

	if len(colorIndex1) > 4 {
		return colorIndex1, colorIndex2, fmt.Errorf("Too many colors in char (colorIndex1)")
	}
	if len(colorIndex2) > 4 {
		return colorIndex1, colorIndex2, fmt.Errorf("Too many colors in char (colorIndex2)")
	}

	return colorIndex1, colorIndex2, nil
}

func multiColorIndexes(preloadBitpair byte, preloadRGB RGB, preloadColorIndex byte, cc []colorInfo) (map[RGB]byte, map[byte]byte, error) {
	// rgb to bitpair
	colorIndex1 := map[RGB]byte{preloadRGB: preloadBitpair}
	// bitpair to colorindex
	colorIndex2 := map[byte]byte{preloadBitpair: preloadColorIndex}
	bitpair := byte(0)
	for _, ci := range cc {
		if bitpair == preloadBitpair {
			bitpair++
		}
		if ci.colorIndex == colorIndex2[preloadBitpair] {
			continue
		}
		if bitpair > 3 {
			return colorIndex1, colorIndex2, fmt.Errorf("Too many colors in char")
		}

		if _, ok := colorIndex1[ci.rgb]; !ok {
			colorIndex1[ci.rgb] = bitpair
			colorIndex2[bitpair] = ci.colorIndex
		}
		bitpair++
	}
	return colorIndex1, colorIndex2, nil
}

func (img *sourceImage) convertToKoala() (Koala, error) {
	koala := Koala{
		BgColor:        img.backgroundColor.colorIndex,
		SourceFilename: img.sourceFilename,
	}

	for char := 0; char < 1000; char++ {
		cc := sortColors(img.charColors[char])

		colorIndex1, colorIndex2, err := img.multiColorIndexes(cc)
		if err != nil {
			return koala, fmt.Errorf("error in char %d: %v", char, err)
		}

		//colorIndex1, colorIndex2, err := multiColorIndexes(byte(0), img.backgroundColor.rgb, img.backgroundColor.colorIndex, cc)
		//if err != nil {
		//	return koala, fmt.Errorf("error in char %d: %v", char, err)
		//}

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
			koala.Bitmap[bitmapIndex+byteIndex] = bmpbyte
		}

		if _, ok := colorIndex2[1]; ok {
			koala.ScreenColor[char] = colorIndex2[1] << 4
		}
		if _, ok := colorIndex2[2]; ok {
			koala.ScreenColor[char] = koala.ScreenColor[char] | colorIndex2[2]
		}
		if _, ok := colorIndex2[3]; ok {
			koala.D800Color[char] = colorIndex2[3]
		}
	}
	return koala, nil
}

func (img *sourceImage) convertToHires() (Hires, error) {
	h := Hires{
		SourceFilename: img.sourceFilename,
	}

	for char := 0; char < 1000; char++ {
		cc := sortColors(img.charColors[char])

		// we need colorindexes to know which bit is what
		colorIndex1 := map[RGB]byte{}
		colorIndex2 := map[byte]byte{}
		bit := byte(0)
		for _, ci := range cc {
			if bit > 1 {
				return h, fmt.Errorf("Too many hires colors in char %v.", char)
			}
			if _, ok := colorIndex2[bit]; !ok {
				colorIndex1[ci.rgb] = bit
				colorIndex2[bit] = ci.colorIndex
			}
			bit++
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

	// TODO: make colors fully configurable
	// we can now force charcol and bgcol from the cli
	var colorIndex1 map[RGB]byte
	var colorIndex2 map[byte]byte

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

	if forcecharcol < 0 {
		colorIndex1, colorIndex2, err = multiColorIndexes(byte(3), cc[0].rgb, cc[0].colorIndex, cc)
	} else {
		found := false
		for _, col := range cc {
			if col.colorIndex == byte(forcecharcol) {
				colorIndex1, colorIndex2, err = multiColorIndexes(byte(3), col.rgb, col.colorIndex, cc)
				if verbose {
					log.Printf("forced character color %d was found", forcecharcol)
				}
				found = true
				break
			}
		}
		if !found {
			if !quiet {
				log.Printf("forced character color %d was NOT found", forcecharcol)
				colorIndex1, colorIndex2, err = multiColorIndexes(byte(3), cc[0].rgb, cc[0].colorIndex, cc)
			}
		}
	}
	if err != nil {
		return c, fmt.Errorf("convertToMultiColorCharset multiColorIndexes error: %v", err)
	}

	if verbose {
		log.Printf("charset colors: %v\n", cc)
		log.Printf("colorIndex1: %v\n", colorIndex1)
		log.Printf("colorIndex2: %v\n", colorIndex2)
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
