package png2prg

import (
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
)

func (img *sourceImage) setPreferredBitpairColors(v string) (err error) {
	if v == "" {
		return nil
	}
	if img.preferredBitpairColors, err = parseBitPairColors(v); err != nil {
		return fmt.Errorf("parseBitPairColors %q failed: %w", v, err)
	}
	if img.opt.Verbose {
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
		img.xOffset += (384 - 320) / 2       // 32
		img.yOffset += ((272 - 200) / 2) - 1 // 35
		// some people
		// img.xOffset, img.yOffset = 32, 36
		img.width, img.height = 320, 200
		return nil
	case img.hasSpriteDimensions():
		return nil
	case img.opt.CurrentGraphicsType == singleColorSprites || img.opt.CurrentGraphicsType == multiColorSprites:
		if img.opt.Verbose {
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
		if img.opt.Verbose {
			log.Printf("forcing dimension %d * %d", img.width, img.height)
		}
		return nil
	}
	return fmt.Errorf("image is not 320x200, 384x272 or x*24 x y*21 pixels, but %d x %d pixels", img.width, img.height)
}

func (img *sourceImage) hasSpriteDimensions() bool {
	return (img.width%24 == 0) && (img.height%21 == 0)
}

func (img *sourceImage) analyze() (err error) {
	if err = img.analyzePalette(); err != nil {
		return fmt.Errorf("analyzePalette failed: %w", err)
	}
	if img.hasSpriteDimensions() {
		return img.analyzeSprites()
	}

	if err = img.makeCharColors(); err != nil {
		return fmt.Errorf("img.makeCharColors failed: %w", err)
	}

	max, _ := img.maxColorsPerChar()
	if img.opt.Verbose {
		log.Printf("max colors per char: %d\n", max)
	}
	numColors, colorIndexes, sumColors := img.countColors()
	if img.opt.Verbose {
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
	if !img.opt.Quiet {
		fmt.Printf("file %q has graphics mode: %s\n", img.sourceFilename, img.graphicsType)
	}
	if img.opt.GraphicsMode != "" {
		if img.graphicsType != img.opt.CurrentGraphicsType {
			img.graphicsType = img.opt.CurrentGraphicsType
			if !img.opt.Quiet {
				fmt.Printf("graphics mode forced: %s\n", img.graphicsType)
			}
			if img.graphicsType == singleColorCharset && numColors > 2 {
				return fmt.Errorf("unable to convert to %s, too many colors: %d > 2", img.graphicsType, numColors)
			}
		}
	}
	if err = img.findBorderColor(); err != nil {
		if img.opt.Verbose {
			log.Printf("skipping: findBorderColor failed: %v", err)
		}
	}
	if img.graphicsType == multiColorBitmap {
		if err = img.findBackgroundColor(); err != nil {
			return fmt.Errorf("findBackgroundColor failed: %w", err)
		}
	}
	if !img.opt.NoGuess {
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

	if !img.opt.Quiet {
		fmt.Printf("graphics mode found: %s\n", img.graphicsType)
	}
	if img.opt.GraphicsMode != "" {
		if img.graphicsType != img.opt.CurrentGraphicsType {
			img.graphicsType = img.opt.CurrentGraphicsType
			if !img.opt.Quiet {
				fmt.Printf("graphics mode forced: %s\n", img.graphicsType)
			}
		}
	}

	if err := img.findBackgroundColor(); err != nil {
		return fmt.Errorf("findBackgroundColor failed: %w", err)
	}
	if img.opt.NoGuess {
		return nil
	}
	max, _, sumColors := img.countSpriteColors()
	img.guessPreferredBitpairColors(max, sumColors)
	return nil
}

func (img *sourceImage) guessPreferredBitpairColors(maxColors int, sumColors [maxColors]int) {
	if len(img.preferredBitpairColors) >= maxColors {
		return
	}
	if img.opt.Verbose {
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

	// swap bitpair 01 with bitpair 11, it generally improves packing
	if len(img.preferredBitpairColors) == 4 {
		img.preferredBitpairColors[1], img.preferredBitpairColors[3] = img.preferredBitpairColors[3], img.preferredBitpairColors[1]
	}
	if img.opt.Verbose {
		log.Printf("guessed some -bitpair-colors %s", img.preferredBitpairColors)
	}

	if img.graphicsType == multiColorCharset && len(img.preferredBitpairColors) == 4 {
		for i, v := range img.preferredBitpairColors {
			if v != 0 {
				continue
			}
			if img.opt.Verbose {
				log.Printf("but by default, prefer black as charcolor, to override use all %d -bitpair-colors %v", maxColors, img.preferredBitpairColors)
			}
			img.preferredBitpairColors[3], img.preferredBitpairColors[i] = img.preferredBitpairColors[i], img.preferredBitpairColors[3]
			if img.opt.Verbose {
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

func (img *sourceImage) countSpriteColors() (int, []byte, [maxColors]int) {
	m := img.palette
	sum := [maxColors]int{}

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

func (img *sourceImage) countColors() (int, []byte, [maxColors]int) {
	m := make(PaletteMap, maxColors)
	var sum [maxColors]int
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

func (img *sourceImage) maxColorsPerChar() (max int, m PaletteMap) {
	for i := range img.charColors {
		if len(img.charColors[i]) > max {
			max = len(img.charColors[i])
			m = img.charColors[i]
		}
	}
	return max, m
}

func (img *sourceImage) findBackgroundColorCandidates() {
	backgroundCharColors := []PaletteMap{}
	for _, v := range img.charColors {
		if len(v) == 4 {
			backgroundCharColors = append(backgroundCharColors, v)
		}
	}

	// need to copy the map, as we delete false candidates
	candidates := make(PaletteMap, maxColors)
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
	if img.opt.Verbose {
		log.Printf("all BackgroundColor candidates: %v", candidates)
	}

	for _, charcolormap := range backgroundCharColors {
		for rgb := range candidates {
			if _, ok := charcolormap[rgb]; !ok {
				delete(candidates, rgb)
			}
		}
	}
	img.backgroundCandidates = candidates
	if img.opt.Verbose {
		log.Printf("final BackgroundColor candidates = %v", img.backgroundCandidates)
	}
	return
}

func (img *sourceImage) findBackgroundColor() error {
	var sumColors [maxColors]int
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
				if img.opt.Verbose {
					log.Printf("findBackgroundColor: found background color %d\n", colorIndex)
				}
				img.backgroundColor = ColorInfo{RGB: rgb, ColorIndex: colorIndex}
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
			if img.opt.Verbose {
				log.Printf("findBackgroundColor: found background color %d\n", colorIndex)
			}
			img.backgroundColor = ColorInfo{RGB: rgb, ColorIndex: colorIndex}
			return nil
		case colorIndex == byte(forceBgCol):
			if img.opt.Verbose {
				log.Printf("findBackgroundColor: found preferred background color %d\n", forceBgCol)
			}
			img.backgroundColor = ColorInfo{RGB: rgb, ColorIndex: colorIndex}
			return nil
		}
	}

	for rgb, colorIndex = range img.backgroundCandidates {
		if img.opt.Verbose {
			log.Printf("findBackgroundColor: we tried looking for color %d, but we have to settle for color %d\n", forceBgCol, colorIndex)
		}
		img.backgroundColor = ColorInfo{RGB: rgb, ColorIndex: colorIndex}
		return nil
	}
	return fmt.Errorf("background color not found")
}

func (img *sourceImage) findBorderColor() error {
	if img.opt.ForceBorderColor >= 0 && img.opt.ForceBorderColor < maxColors {
		for rgb, ci := range img.palette {
			if ci == byte(img.opt.ForceBorderColor) {
				img.borderColor = ColorInfo{RGB: rgb, ColorIndex: ci}
				if img.opt.Verbose {
					log.Printf("force BorderColor: %v", img.borderColor)
				}
				return nil
			}
		}
		img.borderColor = ColorInfo{RGB: RGB{0x12, 0x34, 0x56}, ColorIndex: byte(img.opt.ForceBorderColor)}
		if img.opt.Verbose {
			log.Printf("BorderColor %d not found in palette: %s", img.opt.ForceBorderColor, img.palette)
			log.Printf("forcing BorderColor %d anyway: %v", img.opt.ForceBorderColor, img.borderColor)
		}
		return nil
	}
	if img.xOffset == 0 || img.yOffset == 0 {
		return fmt.Errorf("border color not found")
	}
	rgb := img.colorAtXY(-10, -10)
	if ci, ok := img.palette[rgb]; ok {
		img.borderColor = ColorInfo{RGB: rgb, ColorIndex: ci}
		if img.opt.Verbose {
			log.Printf("findBorderColor found: %s", img.borderColor)
		}
		return nil
	}
	return fmt.Errorf("border color not found")
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
				log.Printf("forced BackgroundColor %d not possible in char %v (x=%d, y=%d)", forceBgCol, i, x, y)
				fatalError = true
			}
		}
		if len(charColors) > 4 {
			count := make(map[byte]byte, maxColors)
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

func (img *sourceImage) colorMapFromChar(char int) PaletteMap {
	charColors := make(PaletteMap, maxColors)
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
func (img *sourceImage) analyzePalette() error {
	minDistance := int(9e6)
	paletteName := ""
	paletteMap := make(PaletteMap)
	if err := img.setSourceColors(); err != nil {
		return fmt.Errorf("setSourceColors failed: %w", err)
	}
	for name, palette := range C64Palettes {
		distance, curMap := img.distanceAndMap(palette)
		if img.opt.Verbose {
			log.Printf("%q distance: %v\n", name, distance)
		}
		if distance < minDistance {
			paletteMap, paletteName, minDistance = curMap, name, distance
		}
		if distance == 0 {
			break
		}
	}

	m := [maxColors]bool{}
	for _, ci := range paletteMap {
		if m[ci] {
			log.Printf("source colors: %#v", img.colors)
			log.Printf("palette: %s", paletteMap)
			return fmt.Errorf("unable to properly detect palette.")
		}
		m[ci] = true
	}

	if !img.opt.Quiet {
		fmt.Printf("palette found: %s distance: %d\n", paletteName, minDistance)
	}
	if img.opt.Verbose {
		log.Printf("palette: %s", paletteMap)
	}
	img.palette = paletteMap
	return nil
}

func (img *sourceImage) setSourceColors() error {
	m := make(map[RGB]bool, maxColors)
	for x := 0; x < img.image.Bounds().Max.X-img.xOffset; x++ {
		for y := 0; y < img.image.Bounds().Max.Y-img.yOffset; y++ {
			rgb := img.colorAtXY(x, y)
			if _, ok := m[rgb]; !ok {
				m[rgb] = true
			}
		}
	}
	cc := make([]RGB, 0, maxColors)
	for rgb := range m {
		cc = append(cc, rgb)
	}
	img.colors = cc
	if len(m) > maxColors {
		return fmt.Errorf("image uses %d colors, the maximum is %d.", len(m), maxColors)
	}
	return nil
}

func (img *sourceImage) distanceAndMap(palette [maxColors]ColorInfo) (int, PaletteMap) {
	m := make(PaletteMap, maxColors)
	totalDistance := 0
	for _, rgb := range img.colors {
		if _, ok := m[rgb]; !ok {
			d := 0
			m[rgb], d = rgb.colorIndexAndDistance(palette)
			totalDistance += d
		}
	}
	return totalDistance, m
}

func (r RGB) colorIndexAndDistance(palette [maxColors]ColorInfo) (byte, int) {
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

func (r RGB) distanceTo(r2 RGB) int {
	dr := int(math.Abs(float64(r.R) - float64(r2.R)))
	dg := int(math.Abs(float64(r.G) - float64(r2.G)))
	db := int(math.Abs(float64(r.B) - float64(r2.B)))
	return dr + dg + db
}
