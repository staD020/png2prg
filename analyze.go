package main

import (
	"fmt"
	"image"
	"image/gif"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func newSourceImage(filename string) (img sourceImage, err error) {
	img.sourceFilename = filename
	if err = img.setPreferredBitpairColors(bitpairColorsString); err != nil {
		return img, fmt.Errorf("setPreferredBitpairColors failed: %w", err)
	}
	f, err := os.Open(filename)
	if err != nil {
		return img, fmt.Errorf("could not os.Open file %q: %w", filename, err)
	}
	defer f.Close()
	if img.image, _, err = image.Decode(f); err != nil {
		return img, fmt.Errorf("image.Decode %q failed: %w", filename, err)
	}
	if err = img.checkBounds(); err != nil {
		return img, fmt.Errorf("img.checkBounds failed %q: %w", filename, err)
	}
	if verbose && (img.xOffset != 0 || img.yOffset != 0) {
		log.Printf("img.xOffset, yOffset = %d, %d\n", img.xOffset, img.yOffset)
	}
	return img, nil
}

func newSourceImages(filenames []string) (imgs []sourceImage, err error) {
	for _, filename := range filenames {
		switch strings.ToLower(filepath.Ext(filename)) {
		case ".gif":
			f, err := os.Open(filename)
			if err != nil {
				return nil, fmt.Errorf("os.Open could not open file %q: %w", filename, err)
			}
			defer f.Close()

			g, err := gif.DecodeAll(f)
			if err != nil {
				return nil, fmt.Errorf("gif.DecodeAll %q failed: %w", filename, err)
			}
			if verbose {
				log.Printf("file %q has %d frames", filename, len(g.Image))
			}

			for i, rawImage := range g.Image {
				img := sourceImage{
					sourceFilename: filename,
					image:          rawImage,
				}
				if err = img.setPreferredBitpairColors(bitpairColorsString); err != nil {
					return nil, fmt.Errorf("setPreferredBitpairColors %q failed: %w", bitpairColorsString, err)
				}
				if err = img.checkBounds(); err != nil {
					return nil, fmt.Errorf("img.checkBounds failed %q frame %d: %w", filename, i, err)
				}
				imgs = append(imgs, img)
			}
		default:
			img, err := newSourceImage(filename)
			if err != nil {
				return nil, fmt.Errorf("newSourceImage %q failed: %w", filename, err)
			}
			imgs = append(imgs, img)
		}
	}
	return imgs, nil
}

func (img *sourceImage) setPreferredBitpairColors(v string) (err error) {
	if v == "" {
		return nil
	}
	if img.preferredBitpairColors, err = parseBitPairColors(v); err != nil {
		return fmt.Errorf("parseBitPairColors %q failed: %w", v, err)
	}
	if verbose {
		log.Printf("will prefer bitpair colors: %v", img.preferredBitpairColors)
	}
	return nil
}

func parseBitPairColors(bp string) ([]byte, error) {
	var result []byte
	for _, v := range strings.Split(bp, ",") {
		i, err := strconv.Atoi(v)
		if err != nil {
			return result, fmt.Errorf("strconv.Atoi conversion of %q to integers failed: %w", bp, err)
		}
		if i < -1 || i > 15 {
			return result, fmt.Errorf("incorrect color %d", i)
		}
		result = append(result, byte(i))
	}
	return result, nil
}

func (img *sourceImage) checkBounds() error {
	img.xOffset, img.yOffset = img.image.Bounds().Min.X, img.image.Bounds().Min.Y
	img.width, img.height = img.image.Bounds().Max.X-img.xOffset, img.image.Bounds().Max.Y-img.yOffset

	switch {
	case (img.width == 320) && (img.height == 200):
		return nil
	case (img.width == 384) && (img.height == 272):
		// default screenshot size in vice with default borders
		img.xOffset += (384 - 320) / 2
		img.yOffset += ((272 - 200) / 2) - 1
		// some people
		// img.xOffset, img.yOffset = 32, 36
		img.width, img.height = 320, 200
		return nil
	case img.hasSpriteDimensions():
		return nil
	case currentGraphicsType == singleColorSprites || currentGraphicsType == multiColorSprites:
		if verbose {
			log.Printf("sprites forced, allowing non-sprite dimension %d * %d", img.width, img.height)
		}
		if img.width%24 == 0 {
			img.width = int(math.Floor(float64(img.width)/24)) * 24
		} else {
			img.width = int(math.Floor(float64(img.width)/24)+1) * 24
		}
		if img.height%21 == 0 {
			img.height = int(math.Floor(float64(img.height)/21)) * 21
		} else {
			img.height = int(math.Floor(float64(img.height)/21)+1) * 21
		}
		if verbose {
			log.Printf("forcing dimension %d * %d", img.width, img.height)
		}
		return nil
	}
	return fmt.Errorf("image is not 320x200, 384x272 or x*24 x y*21 pixels, but %d x %d pixels", img.width, img.height)
}

func (img *sourceImage) hasSpriteDimensions() bool {
	return (img.width%24 == 0) && (img.height%21 == 0)
}

func (img *sourceImage) analyze() error {
	img.analyzePalette()
	if img.hasSpriteDimensions() {
		return img.analyzeSprites()
	}

	if err := img.makeCharColors(); err != nil {
		return fmt.Errorf("img.makeCharColors failed: %w", err)
	}

	max, _ := img.maxColorsPerChar()
	if verbose {
		log.Printf("max colors per char: %d\n", max)
	}
	numColors, colorIndexes, sumColors := img.countColors()
	if verbose {
		log.Printf("total colors: %d (%v)\n", numColors, colorIndexes)
	}

	switch {
	case max < 2:
		return fmt.Errorf("max colors per char %q < 2, is this a blank image?", max)
	case numColors == 2:
		img.graphicsType = singleColorCharset
	case max == 2:
		img.graphicsType = singleColorBitmap
	case numColors == 3 || numColors == 4:
		img.graphicsType = multiColorCharset
	case max > 2:
		img.graphicsType = multiColorBitmap
	}
	if !quiet {
		fmt.Printf("file %q has graphics mode: %s\n", img.sourceFilename, img.graphicsType)
	}
	if graphicsMode != "" {
		if img.graphicsType != currentGraphicsType {
			img.graphicsType = currentGraphicsType
			if !quiet {
				fmt.Printf("graphics mode forced: %s\n", img.graphicsType)
			}
			if img.graphicsType == singleColorCharset && numColors > 2 {
				return fmt.Errorf("unable to convert to %s, too many colors: %d > 2", img.graphicsType, numColors)
			}
		}
	}
	if err := img.findBorderColor(); err != nil {
		if verbose {
			log.Printf("skipping: findBorderColor failed: %v", err)
		}
	}
	if img.graphicsType == multiColorBitmap {
		if err := img.findBackgroundColor(); err != nil {
			return fmt.Errorf("findBackgroundColor failed: %w", err)
		}
	}
	if !noGuess {
		if img.graphicsType == multiColorBitmap && max < 4 {
			max = 4
		}
		img.guessPreferredBitpairColors(max, sumColors)
	}
	return nil
}

func (img *sourceImage) analyzeSprites() error {
	if img.width/24 == 0 || img.height/21 == 0 {
		return fmt.Errorf("%d X-sprites x %d Y-sprites: cant have 0 sprites", img.width/24, img.height/21)
	}

	switch {
	case len(img.palette) <= 2:
		img.graphicsType = singleColorSprites
	case len(img.palette) == 3 || len(img.palette) == 4:
		img.graphicsType = multiColorSprites
	default:
		return fmt.Errorf("too many colors %d > 4", len(img.palette))
	}

	if !quiet {
		fmt.Printf("graphics mode found: %s\n", img.graphicsType)
	}
	if graphicsMode != "" {
		if img.graphicsType != currentGraphicsType {
			img.graphicsType = currentGraphicsType
			if !quiet {
				fmt.Printf("graphics mode forced: %s\n", img.graphicsType)
			}
		}
	}

	if err := img.findBackgroundColor(); err != nil {
		return fmt.Errorf("findBackgroundColor failed: %w", err)
	}
	if noGuess {
		return nil
	}
	max, _, sumColors := img.countSpriteColors()
	img.guessPreferredBitpairColors(max, sumColors)
	return nil
}

func (img *sourceImage) guessPreferredBitpairColors(maxColors int, sumColors [16]int) {
	if len(img.preferredBitpairColors) >= maxColors {
		return
	}
	if verbose {
		log.Printf("sumColors: %v", sumColors)
	}
	if img.graphicsType == multiColorBitmap && len(img.preferredBitpairColors) == 0 {
		img.preferredBitpairColors = append(img.preferredBitpairColors, img.backgroundColor.ColorIndex)
	}
	for i := len(img.preferredBitpairColors); i < maxColors; i++ {
		max := 0
		var colorIndex byte
	NEXTCOLOR:
		for j, sum := range sumColors {
			if sum == 0 {
				continue
			}
			for _, exists := range img.preferredBitpairColors {
				if j == int(exists) {
					continue NEXTCOLOR
				}
			}
			if sum > max {
				max = sum
				colorIndex = byte(j)
			}
		}
		img.preferredBitpairColors = append(img.preferredBitpairColors, colorIndex)
		sumColors[colorIndex] = 0
	}
	if verbose {
		log.Printf("guessed some -bitpair-colors %v", img.preferredBitpairColors)
	}

	if img.graphicsType == multiColorCharset && len(img.preferredBitpairColors) == 4 {
		for i, v := range img.preferredBitpairColors {
			if v != 0 {
				continue
			}
			if verbose {
				log.Printf("but by default, prefer black as charcolor, to override use all %d -bitpair-colors %v", maxColors, img.preferredBitpairColors)
			}
			img.preferredBitpairColors[3], img.preferredBitpairColors[i] = img.preferredBitpairColors[i], img.preferredBitpairColors[3]
			if verbose {
				log.Printf("now using -bitpair-colors %v", img.preferredBitpairColors)
			}
			break
		}
		if img.preferredBitpairColors[3] > 7 {
			for i, v := range img.preferredBitpairColors {
				if v < 8 {
					img.preferredBitpairColors[3], img.preferredBitpairColors[i] = img.preferredBitpairColors[i], img.preferredBitpairColors[3]
					log.Printf("had to avoid mixed singlecolor/multicolor mode, -bitpair-colors %v", img.preferredBitpairColors)
					break
				}
			}
		}
	}
}

func (img *sourceImage) countSpriteColors() (int, []byte, [16]int) {
	m := img.palette
	sum := [16]int{}

	for y := 0; y < img.height; y++ {
		for x := 0; x < img.width; x++ {
			rgb := img.colorAtXY(x, y)
			if ci, ok := img.palette[rgb]; ok {
				sum[ci]++
				continue
			}
			panic("countSpriteColors: this should never happen")
		}
	}
	ci := []byte{}
	for _, v := range img.palette {
		ci = append(ci, v)
	}
	sort.Slice(ci, func(i, j int) bool {
		return ci[i] < ci[j]
	})
	return len(m), ci, sum
}

func (img *sourceImage) countColors() (int, []byte, [16]int) {
	m := make(map[RGB]byte, 16)
	var sum [16]int
	for i := range img.charColors {
		for rgb, colorIndex := range img.charColors[i] {
			m[rgb] = colorIndex
			sum[colorIndex]++
		}
	}
	ci := []byte{}
	for _, v := range m {
		ci = append(ci, v)
	}
	sort.Slice(ci, func(i, j int) bool {
		return ci[i] < ci[j]
	})
	return len(m), ci, sum
}

func (img *sourceImage) maxColorsPerChar() (max int, m map[RGB]byte) {
	for i := range img.charColors {
		if len(img.charColors[i]) > max {
			max = len(img.charColors[i])
			m = img.charColors[i]
		}
	}
	return max, m
}

func (img *sourceImage) findBackgroundColorCandidates() {
	backgroundCharColors := []map[RGB]byte{}
	for _, v := range img.charColors {
		if len(v) == 4 {
			backgroundCharColors = append(backgroundCharColors, v)
		}
	}

	// need to copy the map, as we delete false candidates
	candidates := make(map[RGB]byte, 16)
	switch {
	case len(backgroundCharColors) > 0:
		for k, v := range backgroundCharColors[0] {
			candidates[k] = v
		}
	default:
		for k, v := range img.palette {
			candidates[k] = v
		}
	}

	if verbose {
		log.Printf("all bgcol candidates: %v", candidates)
	}

	for _, charcolormap := range backgroundCharColors {
		for rgb := range candidates {
			if _, ok := charcolormap[rgb]; !ok {
				if verbose {
					log.Printf("not a bgcol candidate, delete: %v", rgb)
				}
				delete(candidates, rgb)
			}
		}
	}
	img.backgroundCandidates = candidates
	if verbose {
		log.Printf("final bgcol candidates = %v", img.backgroundCandidates)
	}
	return
}

func (img *sourceImage) findBackgroundColor() error {
	var sumColors [16]int
	isSprites := img.graphicsType == singleColorSprites || img.graphicsType == multiColorSprites
	if isSprites {
		_, _, sumColors = img.countSpriteColors()
	} else {
		_, _, sumColors = img.countColors()
	}

	var rgb RGB
	var colorIndex byte
	forceBgCol := -1
	switch {
	case len(img.preferredBitpairColors) > 0:
		forceBgCol = int(img.preferredBitpairColors[0])
	default:
		max := 0
		colorIndex := -1
		for color, count := range sumColors {
			if count > max {
				max = count
				colorIndex = color
			}
		}
		forceBgCol = colorIndex
	}

	if isSprites {
		for rgb, colorIndex = range img.palette {
			if colorIndex == byte(forceBgCol) {
				if verbose {
					log.Printf("findBackgroundColor: found background color %d\n", colorIndex)
				}
				img.backgroundColor = colorInfo{RGB: rgb, ColorIndex: colorIndex}
				return nil
			}
		}
		return fmt.Errorf("background color not found in sprites")
	}

	if img.backgroundCandidates == nil {
		img.findBackgroundColorCandidates()
	}

	for rgb, colorIndex = range img.backgroundCandidates {
		switch {
		case forceBgCol < 0:
			if verbose {
				log.Printf("findBackgroundColor: found background color %d\n", colorIndex)
			}
			img.backgroundColor = colorInfo{RGB: rgb, ColorIndex: colorIndex}
			return nil
		case colorIndex == byte(forceBgCol):
			if verbose {
				log.Printf("findBackgroundColor: found preferred background color %d\n", forceBgCol)
			}
			img.backgroundColor = colorInfo{RGB: rgb, ColorIndex: colorIndex}
			return nil
		}
	}

	for rgb, colorIndex = range img.backgroundCandidates {
		if !quiet {
			fmt.Printf("findBackgroundColor: we tried looking for color %d, but we have to settle for color %d\n", forceBgCol, colorIndex)
		}
		img.backgroundColor = colorInfo{RGB: rgb, ColorIndex: colorIndex}
		return nil
	}
	return fmt.Errorf("background color not found")
}

func (img *sourceImage) findBorderColor() error {
	if img.xOffset == 0 || img.yOffset == 0 {
		return fmt.Errorf("border color not found")
	}
	rgb := img.colorAtXY(-10, -10)
	if ci, ok := img.palette[rgb]; ok {
		img.borderColor = colorInfo{RGB: rgb, ColorIndex: ci}
	}
	if verbose {
		log.Printf("findBorderColor found: %v", img.borderColor)
	}
	return nil
}

func (img *sourceImage) makeCharColors() error {
	forceBgCol := -1
	if len(img.preferredBitpairColors) > 0 {
		forceBgCol = int(img.preferredBitpairColors[0])
	}
	fatalError := false
	for i := 0; i < 1000; i++ {
		charColors := img.colorMapFromChar(i)
		if forceBgCol >= 0 && len(charColors) == 4 {
			found := false
			for _, val := range charColors {
				if val == byte(forceBgCol) {
					found = true
					break
				}
			}
			if !found {
				x, y := xyFromChar(i)
				log.Printf("forced bgcol %d not possible in char %v (x=%d, y=%d)", forceBgCol, i, x, y)
				fatalError = true
			}
		}
		if len(charColors) > 4 {
			count := make(map[byte]byte, 16)
			for _, indexcolor := range charColors {
				count[indexcolor] = 1
			}
			if len(count) > 4 {
				x, y := xyFromChar(i)
				log.Printf("amount of colors in char %v (x=%d, y=%d) %d > 4 : %v", i, x, y, len(count), count)
				fatalError = true
			}
		}

		img.charColors[i] = charColors
	}
	if fatalError {
		return fmt.Errorf("fatal error: unable to convert %q", img.sourceFilename)
	}
	return nil
}

func (img *sourceImage) colorMapFromChar(char int) map[RGB]byte {
	charColors := make(map[RGB]byte, 16)
	x, y := xyFromChar(char)
	for pixely := y; pixely < y+8; pixely++ {
		for pixelx := x; pixelx < x+8; pixelx++ {
			rgb := img.colorAtXY(pixelx, pixely)
			if _, ok := charColors[rgb]; !ok {
				charColors[rgb] = img.palette[rgb]
			}
		}
	}
	return charColors
}

func (img *sourceImage) colorAtXY(x, y int) RGB {
	r, g, b, _ := img.image.At(img.xOffset+x, img.yOffset+y).RGBA()
	return RGB{byte(r), byte(g), byte(b)}
}

func xyFromChar(i int) (int, int) {
	return 8*i - (320 * int(math.Floor(float64(i/40)))),
		8 * int(math.Floor(float64(i/40)))
}

// analyzePalette finds the closest paletteMap and sets img.palette
func (img *sourceImage) analyzePalette() {
	minDistance := 9e9
	paletteName := ""
	paletteMap := make(map[RGB]byte)
	img.setSourceColors()
	for name, palette := range c64palettes {
		distance, curMap := img.distanceAndMap(palette)
		if verbose {
			log.Printf("color distance: %v => %v\n", name, distance)
		}
		if distance < minDistance {
			paletteMap, paletteName, minDistance = curMap, name, distance
		}
		if distance == 0 {
			break
		}
	}
	if verbose {
		log.Printf("%v palette found: %v distance: %v\n", img.sourceFilename, paletteName, minDistance)
		log.Printf("palette: %v\n", paletteMap)
	}
	img.palette = paletteMap
	return
}

func (img *sourceImage) setSourceColors() {
	m := make(map[RGB]bool, 16)
	for x := 0; x < img.image.Bounds().Max.X-img.xOffset; x += 2 {
		for y := 0; y < img.image.Bounds().Max.Y-img.yOffset; y++ {
			rgb := img.colorAtXY(x, y)
			if _, ok := m[rgb]; !ok {
				m[rgb] = true
			}
		}
	}
	cc := make([]RGB, 0, 16)
	for rgb := range m {
		cc = append(cc, rgb)
	}
	img.colors = cc
}

func (img *sourceImage) distanceAndMap(palette [16]colorInfo) (float64, map[RGB]byte) {
	m := make(map[RGB]byte, 16)
	totalDistance := 0.0
	for _, rgb := range img.colors {
		if _, ok := m[rgb]; !ok {
			d := 0.0
			m[rgb], d = rgb.colorIndexAndDistance(palette)
			totalDistance += d
			if len(m) == 16 {
				return totalDistance, m
			}
		}
	}
	return totalDistance, m
}

func (r RGB) colorIndexAndDistance(palette [16]colorInfo) (byte, float64) {
	distance := r.distanceTo(palette[0].RGB)
	closestColorIndex := 0
	for i := 0; i < len(palette); i++ {
		d := r.distanceTo(palette[i].RGB)
		if d < distance {
			distance = d
			closestColorIndex = i
		}
	}
	return byte(closestColorIndex), distance
}

func (r RGB) distanceTo(r2 RGB) float64 {
	dr := math.Abs(float64(r.R) - float64(r2.R))
	dg := math.Abs(float64(r.G) - float64(r2.G))
	db := math.Abs(float64(r.B) - float64(r2.B))
	return dr + dg + db
}
