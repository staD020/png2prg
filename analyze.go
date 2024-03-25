package png2prg

import (
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
)

// setPreferredBitpairColors sets img.preferredBitpairColors according to v in format "0,1,6,7".
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

// parseBitPairColors parses the commandline -bitpair-colors string and returns a byte-slice of colors.
func parseBitPairColors(bp string) ([]byte, error) {
	var result []byte
	for _, v := range strings.Split(bp, ",") {
		i, err := strconv.Atoi(v)
		if err != nil {
			return result, fmt.Errorf("strconv.Atoi conversion of %q to integers failed: %w", bp, err)
		}
		if i < -1 || i >= MaxColors {
			return result, fmt.Errorf("incorrect c64 color %d", i)
		}
		result = append(result, byte(i))
	}
	return result, nil
}

// checkBounds confirms img width and height.
// Returns error if requirements aren't met.
func (img *sourceImage) checkBounds() error {
	img.xOffset, img.yOffset = img.image.Bounds().Min.X, img.image.Bounds().Min.Y
	img.width, img.height = img.image.Bounds().Max.X-img.xOffset, img.image.Bounds().Max.Y-img.yOffset

	switch {
	case (img.width == FullScreenWidth) && (img.height == FullScreenHeight):
		return nil
	case (img.width == ViceFullScreenWidth) && (img.height == ViceFullScreenHeight):
		// default screenshot size in vice with default borders
		img.xOffset += (ViceFullScreenWidth - FullScreenWidth) / 2         // 32
		img.yOffset += ((ViceFullScreenHeight - FullScreenHeight) / 2) - 1 // 35
		if img.opt.ForceXOffset > 0 || img.opt.ForceYOffset > 0 {
			img.xOffset, img.yOffset = img.opt.ForceXOffset, img.opt.ForceYOffset
		}
		img.width, img.height = FullScreenWidth, FullScreenHeight
		return nil
	case img.hasSpriteDimensions():
		return nil
	case img.opt.CurrentGraphicsType == singleColorSprites || img.opt.CurrentGraphicsType == multiColorSprites:
		if img.opt.Verbose {
			log.Printf("sprites forced, allowing non-sprite dimension %d * %d", img.width, img.height)
		}
		if img.width%SpriteWidth == 0 {
			img.width = int(math.Floor(float64(img.width)/SpriteWidth)) * SpriteWidth
		} else {
			img.width = int(math.Floor(float64(img.width)/SpriteWidth)+1) * SpriteWidth
		}
		if img.height%SpriteHeight == 0 {
			img.height = int(math.Floor(float64(img.height)/SpriteHeight)) * SpriteHeight
		} else {
			img.height = int(math.Floor(float64(img.height)/SpriteHeight)+1) * SpriteHeight
		}
		if img.opt.Verbose {
			log.Printf("forcing dimension %d * %d", img.width, img.height)
		}
		return nil
	}
	return fmt.Errorf("image is not %dx%d, %dx%d or x*%d x y*%d pixels, but %d x %d pixels", FullScreenWidth, FullScreenHeight, ViceFullScreenWidth, ViceFullScreenHeight, SpriteWidth, SpriteHeight, img.width, img.height)
}

// hasSpriteDimensions returns true if the img is in sprite dimensions.
func (img *sourceImage) hasSpriteDimensions() bool {
	return (img.width%SpriteWidth == 0) && (img.height%SpriteHeight == 0)
}

// analyze validates the image and guesses img.graphicsType, etc.
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

	maxcolsperchar, _ := img.maxColorsPerChar()
	if img.opt.Verbose {
		log.Printf("max colors per char: %d", maxcolsperchar)
	}
	numColors, colorIndexes, sumColors := img.countColors()
	if img.opt.Verbose {
		log.Printf("total colors: %d (%v)", numColors, colorIndexes)
	}

	img.findBackgroundColorCandidates(true)
	numbgcolcandidateshires := len(img.backgroundCandidates)
	if img.opt.Verbose {
		log.Printf("numbgcolcandidateshires: %d", numbgcolcandidateshires)
	}
	img.findBackgroundColorCandidates(false)
	numbgcolcandidates := len(img.backgroundCandidates)

	switch {
	case numColors == 2:
		img.graphicsType = singleColorCharset
	case maxcolsperchar <= 2 && numbgcolcandidateshires != 1:
		img.graphicsType = singleColorBitmap
		if err = img.findECMColors(); err != nil {
			log.Printf("img.findECMColors failed: %v", err)
		} else {
			img.graphicsType = ecmCharset
		}
	case maxcolsperchar <= 2 && numbgcolcandidateshires == 1:
		img.findBackgroundColorCandidates(true)
		img.graphicsType = singleColorCharset
		if len(img.preferredBitpairColors) == 0 && len(img.backgroundCandidates) == 1 {
			for _, bgcol := range img.backgroundCandidates {
				img.preferredBitpairColors = append(img.preferredBitpairColors, bgcol)
			}
		}
	case numColors == 3 || numColors == 4:
		img.graphicsType = multiColorCharset
		if len(img.preferredBitpairColors) == 0 && len(img.backgroundCandidates) == 1 {
			for _, bgcol := range img.backgroundCandidates {
				img.preferredBitpairColors = append(img.preferredBitpairColors, bgcol)
			}
		}
	case maxcolsperchar > 2 && maxcolsperchar <= 4:
		img.graphicsType = multiColorBitmap
		if img.isMultiColorInterlace() {
			img.graphicsType = multiColorInterlaceBitmap
		}
		if numbgcolcandidates > 2 {
			img.graphicsType = mixedCharset
		}
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
		}
	}
	if err = img.findBorderColor(); err != nil {
		if img.opt.Verbose {
			log.Printf("skipping: findBorderColor failed: %v", err)
		}
	}

	switch img.graphicsType {
	case multiColorBitmap:
		if err = img.findBackgroundColor(); err != nil {
			return fmt.Errorf("findBackgroundColor failed: %w", err)
		}
	case ecmCharset:
		if err = img.findECMColors(); err != nil {
			return fmt.Errorf("findECMColors failed: %w", err)
		}
		if img.opt.Verbose {
			log.Printf("img.ecmColors: %v", img.ecmColors)
		}
	}

	if img.opt.NoGuess {
		return nil
	}
	if img.graphicsType == multiColorBitmap {
		maxcolsperchar = 4
	}
	img.guessPreferredBitpairColors(maxcolsperchar, sumColors)
	return nil
}

// analyzeSprites validates the image and guesses img.graphicsType, etc.
func (img *sourceImage) analyzeSprites() error {
	if img.width/SpriteWidth == 0 || img.height/SpriteHeight == 0 {
		return fmt.Errorf("%d X-sprites x %d Y-sprites: cant have 0 sprites", img.width/SpriteWidth, img.height/SpriteHeight)
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

// guessPreferredBitpairColors guesses and sets img.preferredBitpairColors.
func (img *sourceImage) guessPreferredBitpairColors(wantedMaxColors int, sumColors [MaxColors]int) {
	if len(img.preferredBitpairColors) >= wantedMaxColors {
		return
	}
	if img.opt.Verbose {
		log.Printf("sumColors: %v", sumColors)
	}

	if img.graphicsType == multiColorBitmap && len(img.preferredBitpairColors) == 0 {
		img.preferredBitpairColors = append(img.preferredBitpairColors, img.backgroundColor.ColorIndex)
	}
	for i := len(img.preferredBitpairColors); i < wantedMaxColors; i++ {
		max := 0
		var colorIndex byte
	NEXTCOLOR:
		for col, sum := range sumColors {
			if sum == 0 {
				continue
			}
			for _, exists := range img.preferredBitpairColors {
				if col == int(exists) {
					continue NEXTCOLOR
				}
			}
			if sum > max {
				max = sum
				colorIndex = byte(col)
			}
		}
		img.preferredBitpairColors = append(img.preferredBitpairColors, colorIndex)
		sumColors[colorIndex] = 0
	}

	if !img.opt.Quiet {
		fmt.Printf("guessed some -bitpair-colors %s\n", img.preferredBitpairColors)
	}

	if img.graphicsType == multiColorCharset && len(img.preferredBitpairColors) == 4 {
		for i, v := range img.preferredBitpairColors {
			if v != 0 {
				continue
			}
			if img.opt.Verbose {
				log.Printf("but by default, prefer black as charcolor, to override use all %d -bitpair-colors %v", wantedMaxColors, img.preferredBitpairColors)
			}
			img.preferredBitpairColors[3], img.preferredBitpairColors[i] = img.preferredBitpairColors[i], img.preferredBitpairColors[3]
			if img.opt.Verbose {
				log.Printf("now using -bitpair-colors %v", img.preferredBitpairColors)
			}
			break
		}
		if img.preferredBitpairColors[3] > 7 {
		OUTER:
			for i := len(img.preferredBitpairColors) - 1; i >= 0; i-- {
				for i, v := range img.preferredBitpairColors {
					if v < 8 {
						img.preferredBitpairColors[3], img.preferredBitpairColors[i] = img.preferredBitpairColors[i], img.preferredBitpairColors[3]
						if img.opt.Verbose {
							log.Printf("had to avoid mixed singlecolor/multicolor mode, -bitpair-colors %v", img.preferredBitpairColors)
						}
						break OUTER
					}
				}
			}
		}
	}
}

// countSpriteColors returns color statistics.
func (img *sourceImage) countSpriteColors() (numColors int, usedColors []byte, sumColors [MaxColors]int) {
	m := img.palette
	for y := 0; y < img.height; y++ {
		for x := 0; x < img.width; x++ {
			rgb := img.colorAtXY(x, y)
			if ci, ok := img.palette[rgb]; ok {
				sumColors[ci]++
				continue
			}
			panic("countSpriteColors: this should never happen")
		}
	}
	for _, v := range img.palette {
		usedColors = append(usedColors, v)
	}
	sort.Slice(usedColors, func(i, j int) bool {
		return usedColors[i] < usedColors[j]
	})
	return len(m), usedColors, sumColors
}

// countColors returns color statistics.
func (img *sourceImage) countColors() (numColors int, usedColors []byte, sumColors [MaxColors]int) {
	m := make(PaletteMap, MaxColors)
	for i := range img.charColors {
		for rgb, colorIndex := range img.charColors[i] {
			m[rgb] = colorIndex
			sumColors[colorIndex]++
		}
	}
	for _, v := range m {
		usedColors = append(usedColors, v)
	}
	sort.Slice(usedColors, func(i, j int) bool {
		return usedColors[i] < usedColors[j]
	})
	return len(m), usedColors, sumColors
}

// maxColorsPerChar finds the char with the most colors and returns the color count and PalletMap.
func (img *sourceImage) maxColorsPerChar() (max int, m PaletteMap) {
	char := 0
	for i := range img.charColors {
		if len(img.charColors[i]) > max {
			max = len(img.charColors[i])
			m = img.charColors[i]
			char = i
		}
	}
	x, y := xyFromChar(char)
	if img.opt.VeryVerbose {
		log.Printf("char %d (x %d y %d) maxColorsPerChar: %d m: %s", char, x, y, max, m)
	}
	return max, m
}

// findBackgroundColorCandidates iterates over all chars with 4 colors (or 2 for hires) and sets the common color(s) in img.backgroundCandidates.
func (img *sourceImage) findBackgroundColorCandidates(hires bool) {
	backgroundCharColors := []PaletteMap{}
	for _, v := range img.charColors {
		if (hires && len(v) == 2) || (!hires && len(v) == 4) {
			backgroundCharColors = append(backgroundCharColors, v)
		}
	}
	// need to copy the map, as we delete false candidates
	candidates := make(PaletteMap, MaxColors)
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
	if img.opt.VeryVerbose {
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
	if img.opt.Verbose && len(img.backgroundCandidates) > 0 {
		log.Printf("final BackgroundColor candidates = %v", img.backgroundCandidates)
	}
	return
}

// findBackgroundColor figures out the background color (forced or detected) and checks if the background color is possible.
// It sets img.backgroundColor to the correct color, which may differ from what was wanted if that color is not possible.
// returns error if no background color is found or possible.
func (img *sourceImage) findBackgroundColor() error {
	var sumColors [MaxColors]int
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

	img.findBackgroundColorCandidates(false)
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

type sortcolor struct {
	colorIndex byte
	rgb        RGB
	count      int
}

func (img *sourceImage) findECMColors() error {
	if len(img.preferredBitpairColors) == 4 {
		log.Printf("skipping findECMColors because we have 4 img.preferredBitpairColors %s", img.preferredBitpairColors)
		img.ecmColors = img.preferredBitpairColors
		return nil
	}
	if len(img.ecmColors) > 0 {
		return nil
	}

	// find the 4 colors present in all chars
	colm := map[byte]*sortcolor{}
	for _, v := range img.charColors {
		if len(v) != 2 {
			continue
		}
		for rgb, col := range v {
			if c, ok := colm[col]; ok {
				c.count++
			} else {
				colm[col] = &sortcolor{colorIndex: col, rgb: rgb, count: 1}
			}
		}
	}

	colors := make([]*sortcolor, 0)
	for _, col := range colm {
		if col != nil {
			if col.count > 0 {
				colors = append(colors, col)
			}
		}
	}
	sort.Slice(colors, func(i, j int) bool {
		return colors[i].count > colors[j].count
	})
	if len(colors) > 7 {
		colors = colors[:7]
	}

	if img.opt.VeryVerbose {
		log.Printf("findECMColors sorted len %d: %v", len(colors), colors)
		for i, v := range colors {
			log.Printf("%d: %v", i, *v)
		}
	}

	count := 0
PERMUTE:
	for p := make([]int, len(colors)); p[0] < len(p); PermuteNext(p) {
		count++
		s := Permutation(colors, p)
		if len(s) > 4 {
			s = s[:4]
		}
		for _, v := range img.charColors {
			if len(v) != 2 {
				continue
			}
			nfound := 0
			for _, charcol := range v {
				for _, ecmcol := range s {
					if charcol == ecmcol.colorIndex {
						nfound++
					}
				}
			}
			if nfound == 0 {
				continue PERMUTE
			}
		}
		if img.opt.Verbose {
			log.Println("ecm color solution found:")
		}
		for i, v := range s {
			img.ecmColors = append(img.ecmColors, v.colorIndex)
			if img.opt.Verbose {
				log.Printf("  permutation %d -> %d: %v", count, i, *v)
			}
		}
		return nil
	}
	return fmt.Errorf("solution for ecm colors was not found")
}

func PermuteNext(p []int) {
	for i := len(p) - 1; i >= 0; i-- {
		if i == 0 || p[i] < len(p)-i-1 {
			p[i]++
			return
		}
		p[i] = 0
	}
}

func Permutation[S ~[]E, E any](orig S, p []int) (r S) {
	r = append(r, orig...)
	for i, v := range p {
		r[i], r[i+v] = r[i+v], r[i]
	}
	return r
}

// findBorderColor sets img.borderColor to opt.ForceBorderColor or detects it if a vice default screenshot is used.
// returns error if the border color is not found.
func (img *sourceImage) findBorderColor() error {
	if img.opt.ForceBorderColor >= 0 && img.opt.ForceBorderColor < MaxColors {
		for rgb, ci := range img.palette {
			if ci == byte(img.opt.ForceBorderColor) {
				img.borderColor = ColorInfo{RGB: rgb, ColorIndex: ci}
				if img.opt.Verbose {
					log.Printf("force BorderColor: %v", img.borderColor)
				}
				return nil
			}
		}
		rgb := C64Palettes["pepto"][img.opt.ForceBorderColor].RGB
		img.borderColor = ColorInfo{RGB: rgb, ColorIndex: byte(img.opt.ForceBorderColor)}
		if img.opt.Verbose {
			log.Printf("BorderColor %d not found in palette: %s", img.opt.ForceBorderColor, img.palette)
			log.Printf("forcing BorderColor %d anyway: %v", img.opt.ForceBorderColor, img.borderColor)
		}
		return nil
	}
	if img.xOffset == 0 || img.yOffset == 0 {
		return fmt.Errorf("border color not found, no border in image")
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

// makeCharColors populates img.charColors, containing the colors used in each char.
func (img *sourceImage) makeCharColors() error {
	forceBgCol := -1
	if len(img.preferredBitpairColors) > 0 {
		forceBgCol = int(img.preferredBitpairColors[0])
	}
	fatalError := false
	for i := 0; i < FullScreenChars; i++ {
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
				if img.opt.Verbose {
					log.Printf("forced BackgroundColor %d not possible in char %v (x=%d, y=%d)", forceBgCol, i, x, y)
				}
				fatalError = true
			}
		}
		if len(charColors) > 4 {
			count := make(map[byte]byte, MaxColors)
			for _, indexcolor := range charColors {
				count[indexcolor] = 1
			}
			if len(count) > 4 {
				x, y := xyFromChar(i)
				if img.opt.Verbose {
					log.Printf("amount of colors in char %v (x=%d, y=%d) %d > 4 : %v", i, x, y, len(count), count)
				}
				fatalError = true
			}
		}
		img.charColors[i] = charColors
	}
	if fatalError {
		return fmt.Errorf("fatal error: unable to convert %q, too many colors required per char", img.sourceFilename)
	}
	return nil
}

// colorMapFromChar returns the colors present it the specific char.
func (img *sourceImage) colorMapFromChar(char int) PaletteMap {
	charColors := make(PaletteMap, MaxColors)
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

// colorAtXY returns the RGB color at x,y coordinates.
func (img *sourceImage) colorAtXY(x, y int) RGB {
	r, g, b, _ := img.image.At(img.xOffset+x, img.yOffset+y).RGBA()
	return RGB{byte(r), byte(g), byte(b)}
}

// xyFromChar returns the x and y coordinates for the given char.
func xyFromChar(i int) (int, int) {
	return 8*i - (FullScreenWidth * int(math.Floor(float64(i/40)))),
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
		if img.opt.VeryVerbose {
			log.Printf("%q distance: %v\n", name, distance)
		}
		if distance < minDistance {
			paletteMap, paletteName, minDistance = curMap, name, distance
		}
		if distance == 0 {
			break
		}
	}

	m := [MaxColors]bool{}
	for rgb, ci := range paletteMap {
		if m[ci] {
			log.Printf("source colors: %s", img.colors)
			log.Printf("palette: %s", paletteMap)
			log.Printf("rgb: %s", rgb)
			return fmt.Errorf("unable to properly detect palette")
		}
		m[ci] = true
	}

	// sometimes people want to reserve a specific bitpair
OUTER:
	for _, prefCol := range img.preferredBitpairColors {
		if prefCol > 15 {
			continue
		}
		for _, col := range paletteMap {
			if prefCol == col {
				continue OUTER
			}
		}
		paletteMap[C64Palettes[paletteName][prefCol].RGB] = prefCol
	}

	if !img.opt.Quiet {
		fmt.Printf("file %q palette found: %s distance: %d\n", img.sourceFilename, paletteName, minDistance)
	}
	if img.opt.Verbose {
		log.Printf("file %q palette: %s", img.sourceFilename, paletteMap)
	}
	img.palette = paletteMap
	return nil
}

// setSourceColors parses the image and sets img.colors.
func (img *sourceImage) setSourceColors() error {
	m := make(map[RGB]bool, MaxColors)
	for x := 0; x < img.image.Bounds().Max.X-img.xOffset; x++ {
		for y := 0; y < img.image.Bounds().Max.Y-img.yOffset; y++ {
			rgb := img.colorAtXY(x, y)
			if _, ok := m[rgb]; !ok {
				m[rgb] = true
			}
		}
	}
	img.colors = make([]RGB, 0, MaxColors)
	for rgb := range m {
		img.colors = append(img.colors, rgb)
	}
	if len(m) > MaxColors {
		return fmt.Errorf("image %q uses %d colors, the maximum is %d.", img.sourceFilename, len(m), MaxColors)
	}
	return nil
}

// distanceAndMap calculates the total colordistance of the image colors compared to the input palette.
// It returns the totalDistance and PaletteMap.
func (img *sourceImage) distanceAndMap(palette [MaxColors]ColorInfo) (totalDistance int, m PaletteMap) {
	m = make(PaletteMap, MaxColors)
	for _, rgb := range img.colors {
		if _, ok := m[rgb]; !ok {
			d := 0
			m[rgb], d = rgb.colorIndexAndDistance(palette)
			totalDistance += d
		}
	}
	return totalDistance, m
}

// colorIndexAndDistance finds the closest color from the palette.
func (r RGB) colorIndexAndDistance(palette [MaxColors]ColorInfo) (closestColorIndex byte, distance int) {
	distance = r.distanceTo(palette[0].RGB)
	for i := 0; i < len(palette); i++ {
		d := r.distanceTo(palette[i].RGB)
		if d < distance {
			distance = d
			closestColorIndex = byte(i)
		}
	}
	return closestColorIndex, distance
}

// distanceTo returns the absolute difference in r and r2.
func (r RGB) distanceTo(r2 RGB) int {
	return int(math.Abs(float64(r.R)-float64(r2.R))) +
		int(math.Abs(float64(r.G)-float64(r2.G))) +
		int(math.Abs(float64(r.B)-float64(r2.B)))
}
