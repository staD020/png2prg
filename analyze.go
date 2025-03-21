package png2prg

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"sort"
)

// setPreferredBitpairColors sets img.preferredBitpairColors according to v in format "0,1,6,7".
func (img *sourceImage) setPreferredBitpairColors(v string) (err error) {
	if v == "" {
		return nil
	}
	if img.bpc, err = img.p.ParseBPC(v); err != nil {
		return fmt.Errorf("p.ParseBPC %q failed: %w", v, err)
	}
	if img.graphicsType == singleColorBitmap {
		if len(img.bpc) > 2 {
			img.bpc = img.bpc[0:2]
		}
	}
	if img.opt.Verbose {
		log.Printf("will prefer bitpair colors: %s", img.bpc)
	}
	return nil
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
	case (img.width >= FullScreenWidth) && (img.height >= FullScreenHeight):
		// Handle arbitrary resolutions like Marq's PETSCII editor (352x232)
		img.xOffset += (img.width - FullScreenWidth) / 2
		img.yOffset += (img.height - FullScreenHeight) / 2
		if img.opt.ForceXOffset > 0 || img.opt.ForceYOffset > 0 {
			img.xOffset, img.yOffset = img.opt.ForceXOffset, img.opt.ForceYOffset
		}
		img.width, img.height = FullScreenWidth, FullScreenHeight
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
	if img.opt.Verbose {
		log.Println("palette found:", img.p)
		log.Println("total colors:", img.p.NumColors())
	}
	if img.opt.BitpairColorsString != "" {
		if img.bpc, err = img.p.ParseBPC(img.opt.BitpairColorsString); err != nil {
			return fmt.Errorf("p.ParseBPC failed: %w", err)
		}
	}
	if img.hasSpriteDimensions() {
		return img.analyzeSprites()
	}
	if err = img.makeCharColors(); err != nil {
		return fmt.Errorf("img.makeCharColors failed: %w", err)
	}

	maxcolsperchar := len(img.maxColorsPerChar())
	if img.opt.Verbose {
		log.Printf("max colors per char: %d", maxcolsperchar)
		log.Printf("sum colors: %v", img.sumColors)
	}

	img.findBgCandidates(true)
	numbgcolcandidateshires := len(img.bgCandidates)
	if img.opt.Verbose {
		log.Printf("bgcandidates hires: %v", img.bgCandidates)
	}
	img.findBgCandidates(false)
	numbgcolcandidates := len(img.bgCandidates)
	if img.opt.Verbose {
		log.Printf("bgcandidates multicolor: %v", img.bgCandidates)
	}

	switch {
	case img.p.NumColors() == 2:
		img.graphicsType = singleColorCharset
	case maxcolsperchar <= 2 && numbgcolcandidateshires != 1:
		img.graphicsType = singleColorBitmap
		if err = img.findECMColors(); err != nil {
			if img.opt.Verbose {
				log.Printf("img.findECMColors failed: %v", err)
			}
		} else {
			img.graphicsType = ecmCharset
		}
	case maxcolsperchar <= 2 && numbgcolcandidateshires == 1:
		img.findBgCandidates(true)
		img.graphicsType = singleColorCharset
		if len(img.bpc) == 0 && len(img.bgCandidates) == 1 {
			for _, col := range img.bgCandidates {
				img.bpc = append(img.bpc, &col)
			}
		}
	case img.p.NumColors() == 3 || img.p.NumColors() == 4:
		img.findBgCandidates(false)
		img.graphicsType = multiColorCharset
		if len(img.bpc) == 0 && len(img.bgCandidates) == 1 {
			for _, col := range img.bgCandidates {
				img.bpc = append(img.bpc, &col)
			}
		}
	case maxcolsperchar > 2 && maxcolsperchar <= 4:
		img.graphicsType = multiColorBitmap
		if img.hiresPixels {
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
				fmt.Printf("graphics mode forced: %s\n", img.opt.CurrentGraphicsType)
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
	img.guessPreferredBitpairColors(maxcolsperchar, img.sumColors)
	return nil
}

// analyzeSprites validates the image and guesses img.graphicsType, etc.
func (img *sourceImage) analyzeSprites() error {
	if img.width/SpriteWidth == 0 || img.height/SpriteHeight == 0 {
		return fmt.Errorf("%d X-sprites x %d Y-sprites: cant have 0 sprites", img.width/SpriteWidth, img.height/SpriteHeight)
	}

	switch {
	case img.p.NumColors() <= 2:
		img.graphicsType = singleColorSprites
	case img.p.NumColors() == 3 || img.p.NumColors() == 4:
		img.graphicsType = multiColorSprites
	default:
		return fmt.Errorf("too many colors %d > 4", img.p.NumColors())
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
			if img.opt.CurrentGraphicsType != singleColorSprites && img.opt.CurrentGraphicsType != multiColorSprites {
				return fmt.Errorf("cannot force mode to %s for images in sprite dimensions", img.opt.CurrentGraphicsType)
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
	if len(img.bpc) >= wantedMaxColors {
		return
	}
	if img.opt.Verbose {
		log.Printf("sumColors: %v", sumColors)
	}

	if img.graphicsType == multiColorBitmap && len(img.bpc) == 0 {
		img.bpc = append(img.bpc, &img.bg)
	}
	for i := len(img.bpc); i < wantedMaxColors; i++ {
		max := 0
		var c64col byte
	NEXTCOLOR:
		for col, sum := range sumColors {
			if sum == 0 {
				continue
			}
			for _, bpccol := range img.bpc {
				if col == int(bpccol.C64Color) {
					continue NEXTCOLOR
				}
			}
			if sum > max {
				max = sum
				c64col = byte(col)
			}
		}
		col := img.p.FromC64NoErr(C64Color(c64col))
		img.bpc = append(img.bpc, &col)
		sumColors[c64col] = 0
	}

	if img.graphicsType == multiColorBitmap && len(img.bpc) == 4 {
		img.bpc[1], img.bpc[3] = img.bpc[3], img.bpc[1]
	}

	if img.graphicsType == singleColorCharset || img.graphicsType == petsciiCharset {
		if len(img.bpc) > 1 {
			img.bpc = img.bpc[0:1]
		}
	}

	if !img.opt.Quiet {
		fmt.Printf("guessed some -bitpair-colors %s\n", img.bpc)
	}

	if img.graphicsType == multiColorCharset && len(img.bpc) == 4 {
		for i, col := range img.bpc {
			if col.C64Color != 0 {
				continue
			}
			if img.opt.Verbose {
				log.Printf("but by default, prefer black as charcolor, to override use all %d -bitpair-colors %s", wantedMaxColors, img.bpc)
			}
			img.bpc[3], img.bpc[i] = img.bpc[i], img.bpc[3]
			if !img.opt.Quiet {
				fmt.Printf("now using -bitpair-colors %s\n", img.bpc)
			}
			break
		}
		if img.bpc[3].C64Color > 7 {
		OUTER:
			for i := len(img.bpc) - 1; i >= 0; i-- {
				for i, col := range img.bpc {
					if col.C64Color < 8 {
						img.bpc[3], img.bpc[i] = img.bpc[i], img.bpc[3]
						if img.opt.Verbose {
							log.Printf("had to avoid mixed singlecolor/multicolor mode, -bitpair-colors %s", img.bpc)
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
	for y := 0; y < img.height; y++ {
		for x := 0; x < img.width; x++ {
			col, err := img.p.FromColor(img.At(x, y))
			if err != nil {
				panic(fmt.Errorf("countSpriteColors: color %v not found: %w", col, err))
			}
			sumColors[col.C64Color]++
		}
	}
	for _, col := range img.p.Colors() {
		usedColors = append(usedColors, byte(col.C64Color))
	}
	sort.Slice(usedColors, func(i, j int) bool {
		return usedColors[i] < usedColors[j]
	})
	return img.p.NumColors(), usedColors, sumColors
}

// maxColorsPerChar finds the char with the most colors and returns the Color slice.
func (img *sourceImage) maxColorsPerChar() (cc Colors) {
	char := 0
	max := 0
	for i := range img.charColors {
		if len(img.charColors[i]) > max {
			cc = img.charColors[i]
			max = len(cc)
			char = i
		}
	}
	if img.opt.VeryVerbose {
		x, y := xyFromChar(char)
		log.Printf("char %d (x %d y %d) maxColorsPerChar: %d cc: %v", char, x, y, max, cc)
	}
	return cc
}

// findBgColorCandidates iterates over all chars with 4 colors (or 2 for hires) and sets the common color(s) in img.bgCandidates.
func (img *sourceImage) findBgCandidates(hires bool) {
	charcc := []Colors{}
	for _, cc := range img.charColors {
		if (hires && len(cc) == 2) || (!hires && len(cc) == 4) {
			charcc = append(charcc, cc)
		}
	}
	if len(charcc) == 0 {
		img.bgCandidates = img.p.Colors()
		if len(img.bgCandidates) < MaxColors {
			// add missing colors as all colors should be possible in this case where there is a free bitpair color.
		LOOP:
			for _, col := range paletteSources[0].Colors {
				for _, colcan := range img.bgCandidates {
					if col.C64Color == colcan.C64Color {
						continue LOOP
					}
				}
				img.bgCandidates = append(img.bgCandidates, col)
			}
		}
		return
	}
	candidates := BlankPalette("bgcol", false)
	candidates.Add(charcc[0]...)
	if img.opt.VeryVerbose {
		log.Printf("all BackgroundColor candidates: %s", candidates)
	}
	for _, cc := range charcc {
		for _, col := range candidates.Colors() {
			if !In(cc, col) {
				candidates.Delete(col)
			}
		}
	}
	img.bgCandidates = candidates.SortColors()
	if img.opt.Verbose {
		log.Printf("final BackgroundColor candidates: %s", img.bgCandidates)
	}
	return
}

// findBackgroundColor figures out the background color (forced or detected) and checks if the background color is possible.
// It sets img.backgroundColor to the correct color, which may differ from what was wanted if that color is not possible.
// returns error if no background color is found or possible.
func (img *sourceImage) findBackgroundColor() error {
	var sumColors = img.sumColors
	isSprites := img.graphicsType == singleColorSprites || img.graphicsType == multiColorSprites
	if isSprites {
		_, _, sumColors = img.countSpriteColors()
	}

	var forceBgCol Color
	noForce := true
	switch {
	case len(img.bpc) > 0:
		if img.bpc[0] != nil {
			forceBgCol = *img.bpc[0]
			noForce = false
		}
	default:
		max := 0
		mostused := Color{}
		for color, count := range sumColors {
			if count > max {
				max = count
				mostused = img.p.FromC64NoErr(C64Color(color))
			}
		}
		forceBgCol = mostused
	}

	if isSprites {
		for _, col := range img.p.Colors() {
			if col.C64Color == forceBgCol.C64Color {
				if img.opt.Verbose {
					log.Printf("findBackgroundColor: found background color %d\n", col.C64Color)
				}
				img.bg = col
				return nil
			}
		}
		return fmt.Errorf("background color not found in sprites")
	}

	img.findBgCandidates(false)
	for _, col := range img.bgCandidates {
		switch {
		case noForce:
			if img.opt.Verbose {
				log.Printf("findBackgroundColor: found background color %d\n", col)
			}
			img.bg = col
			return nil
		case col.C64Color == forceBgCol.C64Color:
			if img.opt.Verbose {
				log.Printf("findBackgroundColor: found preferred background color %d\n", forceBgCol)
			}
			img.bg = col
			return nil
		}
	}

	for _, col := range img.bgCandidates {
		if img.opt.Verbose {
			log.Printf("findBackgroundColor: we tried looking for color %d, but we have to settle for color %d\n", forceBgCol, col)
		}
		img.bg = col
		return nil
	}
	return fmt.Errorf("background color not found")
}

type sortColor struct {
	Color
	count int
}

func (img *sourceImage) findECMColors() error {
	if len(img.bpc) == 4 {
		if img.opt.Verbose {
			log.Printf("skipping findECMColors because we have 4 img.bpc %s", img.bpc)
		}
		img.ecmColors = newBitpairsFromBPColors(img.bpc).colors()
		return nil
	}
	if len(img.ecmColors) > 0 {
		return nil
	}

	// find the 4 colors present in all chars
	colm := map[C64Color]*sortColor{}
	for _, cc := range img.charColors {
		if len(cc) != 2 {
			continue
		}
		for _, col := range cc {
			if c, ok := colm[col.C64Color]; ok {
				c.count++
			} else {
				colm[col.C64Color] = &sortColor{Color: col, count: 1}
			}
		}
	}
	if len(colm) == 0 {
		for _, cc := range img.charColors {
			if len(cc) > 1 {
				continue
			}
			for _, col := range cc {
				if c, ok := colm[col.C64Color]; ok {
					c.count++
				} else {
					colm[col.C64Color] = &sortColor{Color: col, count: 1}
				}
			}
		}
	}
	colors := make([]*sortColor, 0)
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
		log.Printf("findECMColors sorted len %d: %s", len(colors), colors)
		for i, v := range colors {
			log.Printf("  %d: %v", i, *v)
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
		for _, cc := range img.charColors {
			if len(cc) != 2 {
				continue
			}
			nfound := 0
			for _, charcol := range cc {
				for _, ecmcol := range s {
					if charcol.C64Color == ecmcol.C64Color {
						nfound++
					}
				}
			}
			if nfound == 0 {
				continue PERMUTE
			}
		}
		bpc := BPColors{}
		for _, v := range s {
			if v == nil {
				continue
			}
			img.ecmColors = append(img.ecmColors, v.Color)
			bpc = append(bpc, &v.Color)
		}
		if !img.opt.Quiet {
			fmt.Printf("ecm color solution found: %s\n", bpc)
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

// findBorderColor sets img.border to opt.ForceBorderColor or detects it if a vice default screenshot is used.
// returns error if the border color is not found.
func (img *sourceImage) findBorderColor() error {
	if img.opt.ForceBorderColor >= 0 && img.opt.ForceBorderColor < MaxColors {
		for _, col := range img.p.Colors() {
			if col.C64Color == C64Color(img.opt.ForceBorderColor) {
				img.border = col
				if img.opt.Verbose {
					log.Printf("force img.border: %s", img.border)
				}
				return nil
			}
		}
		img.border = paletteSources[0].Colors[img.opt.ForceBorderColor]
		if img.opt.Verbose {
			log.Printf("-force-border-color %d not found in palette: %v", img.opt.ForceBorderColor, img.p)
			log.Printf("forcing BorderColor %d anyway: %v", img.opt.ForceBorderColor, img.border)
		}
		return nil
	}
	if img.xOffset < 1 || img.yOffset < 1 {
		return fmt.Errorf("border color not found, no border in image")
	}
	if col, err := img.p.FromColor(img.At(-1, -1)); err == nil {
		img.border = col
		if img.opt.Verbose {
			log.Printf("findBorderColor found: %s", img.border)
		}
		return nil
	}
	return fmt.Errorf("border color not found")
}

// makeCharColors parses the entire image and populates img.charColors.
func (img *sourceImage) makeCharColors() error {
	if len(img.charColors[0]) > 0 {
		// skip redoing already generated charColors
		return nil
	}
	forceBgCol := -1
	if len(img.bpc) > 0 {
		if img.bpc[0] != nil {
			forceBgCol = int(img.bpc[0].C64Color)
		}
	}
	sumColors := [MaxColors]int{}
	fatalError := false
	for char := 0; char < FullScreenChars; char++ {
		cc := img.colorsFromChar(char)
		if forceBgCol >= 0 && len(cc) == 4 {
			found := false
			for _, col := range cc {
				if col.C64Color == C64Color(forceBgCol) {
					found = true
					break
				}
			}
			if !found {
				fatalError = true
				x, y := xyFromChar(char)
				if img.opt.Verbose {
					log.Printf("forced BackgroundColor %d not possible in char %v (x=%d, y=%d)", forceBgCol, char, x, y)
				}
			}
		}
		if len(cc) > 4 {
			fatalError = true
			x, y := xyFromChar(char)
			if img.opt.Verbose {
				log.Printf("amount of colors in char %v (x=%d, y=%d) %d > 4 : %v", char, x, y, len(cc), cc)
			}
		}
		img.charColors[char] = cc
		for _, col := range cc {
			sumColors[col.C64Color]++
		}
	}
	img.sumColors = sumColors
	if fatalError {
		return fmt.Errorf("fatal errors were logged, see above")
	}
	return nil
}

// colorsFromChar returns the Colors of the specific char.
func (img *sourceImage) colorsFromChar(char int) (cc Colors) {
	pixelWidth := 2
	if img.hiresPixels {
		pixelWidth = 1
	}
	x, y := xyFromChar(char)
	m := make(map[color.Color]struct{})
	for pixely := y; pixely < y+8; pixely++ {
		for pixelx := x; pixelx < x+8; pixelx += pixelWidth {
			rgbcol := img.At(pixelx, pixely)
			if _, ok := m[rgbcol]; ok {
				continue
			}
			m[rgbcol] = struct{}{}
			col, err := img.p.FromColor(rgbcol)
			if err != nil {
				log.Panicf("char %d: color must always be found: %v", char, rgbcol)
			}
			cc = append(cc, col)
		}
	}
	sort.Slice(cc, func(i, j int) bool { return cc[i].C64Color < cc[j].C64Color })
	return cc
}

// xyFromChar returns the x and y coordinates for the given char.
func xyFromChar(i int) (int, int) {
	return xFromChar(i), yFromChar(i)
}

// xFromChar returns the x coordinate for the given char.
func xFromChar(i int) int {
	return 8*i - (FullScreenWidth * int(float64(i/40)))
}

// yFromChar returns the y coordinate for the given char.
func yFromChar(i int) int {
	return 8 * int(float64(i/40))
}
