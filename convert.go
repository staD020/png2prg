package png2prg

import (
	"fmt"
	"image/color"
	"log"
	"slices"
	"sort"
)

// In returns true if element v is equal to an element of slice s.
func In[S ~[]E, E comparable](s S, v E) bool {
	return slices.Index(s, v) >= 0
}

type bitpairs struct {
	rgb2bitpair   map[colorKey]byte
	bitpair2color map[byte]Color
	bitpairs      []byte
}

// bitpair returns the bitpair of the Color.
func (bp bitpairs) bitpair(col color.Color) (byte, bool) {
	v, ok := bp.rgb2bitpair[ColorKey(col)]
	return v, ok
}

// color returns the Color used by bitpair.
func (bp bitpairs) color(bitpair byte) (Color, bool) {
	v, ok := bp.bitpair2color[bitpair]
	return v, ok
}

// add adds bitpair/Color to bitpairs.
func (bp *bitpairs) add(bitpair byte, col Color) {
	if bp.bitpair2color == nil {
		bp.bitpair2color = map[byte]Color{}
	}
	if bp.rgb2bitpair == nil {
		bp.rgb2bitpair = map[colorKey]byte{}
	}
	bp.bitpair2color[bitpair] = col
	bp.rgb2bitpair[ColorKey(col)] = bitpair
	for i, bitp := range bp.bitpairs {
		if bitp == bitpair {
			bp.bitpairs = slices.Delete(bp.bitpairs, i, i+1)
			return
		}
	}
}

// delete deletes the bitpair/color from bitpairs.
func (bp *bitpairs) delete(bitpair byte) {
	delete(bp.bitpair2color, bitpair)
	for rgb, bitp := range bp.rgb2bitpair {
		if bitp == bitpair {
			delete(bp.rgb2bitpair, rgb)
			bp.bitpairs = append(bp.bitpairs, bitp)
			return
		}
	}
}

// numColors returns the number of colors.
func (bp bitpairs) numColors() int {
	return len(bp.bitpair2color)
}

// colors returns all Colors, sorted by bitpair.
func (bp bitpairs) colors() (cc Colors) {
	for i := byte(0); i < 4; i++ {
		if col, ok := bp.color(i); ok {
			cc = append(cc, col)
		}
	}
	return cc
}

// newBitpairs return bitpairs.
// It is the main function taking care of bitpair/color sorting, according to img.bpc.
// forcePreferred is used with interlaced pictures.
func (img *sourceImage) newBitpairs(char int, cc Colors, forcePreferred bool) (*bitpairs, error) {
	bp := &bitpairs{
		rgb2bitpair:   map[colorKey]byte{},
		bitpair2color: map[byte]Color{},
	}
	// init caches
	if img.bpcCache[char] == nil {
		img.bpcCache[char] = map[C64Color]byte{}
	}
	for c64col := 0; c64col < MaxColors; c64col++ {
		if img.bpcBitpairCount[c64col] == nil {
			img.bpcBitpairCount[c64col] = map[byte]int{}
		}
	}

	// update bpcCache
	defer func() {
		for bitp, col := range bp.bitpair2color {
			img.bpcCache[char][col.C64Color] = bitp
		}
	}()

	// set background
	if img.graphicsType != singleColorBitmap {
		bp.add(0, img.bg)
	}
	// which bitpairs we have left depends on graphicsType
	switch img.graphicsType {
	case singleColorBitmap:
		bp.bitpairs = []byte{0, 1}
	case singleColorCharset, singleColorSprites, ecmCharset:
		bp.bitpairs = []byte{1}
	default:
		bp.bitpairs = []byte{1, 2, 3}
	}

	if img.opt.Trd {
		x, y := xyFromChar(char)
		column, row := x/8, y/8
		if (column > 1 && column < 38) && (row > 1 && row < 22) {
			bp.add(1, *img.bpc[1])
			bp.add(2, *img.bpc[2])

			for _, col := range cc {
				if _, ok := bp.bitpair(col); ok {
					continue
				}
				if len(bp.bitpairs) == 0 {
					return nil, fmt.Errorf("too many colors in char %d, no bitpairs left", char)
				}
				bp.add(3, col)
			}
			return bp, nil
		}
		if len(img.bpc) >= 3 {
			// pretty ugly implementation to make sure the borders of the map have different screenram colors
			// this is not concurrency safe
			img.bpc[1], img.bpc[2] = img.bpc[2], img.bpc[1]
			defer func() {
				img.bpc[1], img.bpc[2] = img.bpc[2], img.bpc[1]
			}()
		}
	}

	if forcePreferred {
		// used for interlace
		if len(img.bpc) == 0 {
			return nil, fmt.Errorf("you cannot forcePreferred without setting img.bpc")
		}
		// fill preferred
		for preferBitpair, preferColor := range img.bpc {
			if preferColor == nil {
				continue
			}
			bp.add(byte(preferBitpair), *preferColor)
		}
		// fill used
		for _, col := range cc {
			// already set as preferred?
			if _, ok := bp.bitpair(col); ok {
				continue
			}
			// find spot
			if len(bp.bitpairs) == 0 {
				return nil, fmt.Errorf("too many colors in char %d, no bitpairs left", char)
			}
			// take first spot
			bp.add(bp.bitpairs[0], col)
		}
		return bp, nil
	}

	// prefill preferred and used colors
	if len(img.bpc) > 0 {
		for preferBitpair, preferColor := range img.bpc {
			if preferColor == nil {
				continue
			}
			for _, col := range cc {
				if preferColor.C64Color == col.C64Color {
					bp.add(byte(preferBitpair), col)
				}
			}
		}
	}

	// prefill secondary preferred and used colors
	if len(img.bpc2) > 0 {
		if len(img.bpc) > 0 {
			if img.bpc[0].C64Color != img.bpc2[0].C64Color {
				return nil, fmt.Errorf("cannot use different colors for bitpair 00 with -bpc %s -bpc2 %s", img.bpc, img.bpc2)
			}
		}
		for preferBitpair, preferColor := range img.bpc2 {
			if preferColor == nil {
				continue
			}
			if _, ok := bp.color(byte(preferBitpair)); ok {
				continue
			}
			for _, col := range cc {
				if preferColor.C64Color == col.C64Color {
					bp.add(byte(preferBitpair), col)
					//if img.opt.VeryVerbose {
					//	log.Printf("char %d: bpc2 match for bitpair %d color %s", char, preferBitpair, col)
					//}
				}
			}
		}
	}

	// prefill tertiary preferred and used colors
	if len(img.bpc3) > 0 {
		if len(img.bpc) > 0 {
			if img.bpc[0].C64Color != img.bpc3[0].C64Color {
				return nil, fmt.Errorf("cannot use different colors for bitpair 00 with -bpc %s -bpc3 %s", img.bpc, img.bpc3)
			}
		}
		for preferBitpair, preferColor := range img.bpc3 {
			if preferColor == nil {
				continue
			}
			if _, ok := bp.color(byte(preferBitpair)); ok {
				continue
			}
			for _, col := range cc {
				if preferColor.C64Color == col.C64Color {
					bp.add(byte(preferBitpair), col)
					if img.opt.VeryVerbose {
						log.Printf("char %d: bpc3 match for bitpair %d color %s", char, preferBitpair, col)
					}
				}
			}
		}
	}

	// bp includes bgcol, which may not be used in the char.
	if bp.numColors() > len(cc) {
		return bp, nil
	}

	if char > 0 && !img.opt.NoBitpairCounters {
		for _, col := range cc {
			if _, ok := bp.bitpair(col); ok {
				continue
			}
			if len(bp.bitpairs) == 0 {
				return nil, fmt.Errorf("too many colors in char %d, no bitpairs left", char)
			}
			if len(bp.bitpairs) > 1 {
				bpcount := img.bpcBitpairCount[byte(col.C64Color)]
				if len(bpcount) == 0 {
					continue
				}
				//log.Printf("char %d: bitpaircache for col %s", char, col)
				max := 0
				bitpair := byte(0)
				for bitp, count := range bpcount {
					if count > max || (count == max && bitp > bitpair) {
						bitpair = bitp
						max = count
					}
				}
				if max == 0 {
					continue
				}

				for _, avail := range bp.bitpairs {
					if bitpair == avail {
						bp.add(bitpair, col)
					}
				}
			}
		}
	}

	// prefer reusing bitpaircolors of previous char
	if char > 0 && !img.opt.NoPrevCharColors {
	NEXTCOL:
		for _, col := range cc {
			if _, ok := bp.bitpair(col); ok {
				continue
			}
			if len(bp.bitpairs) == 0 {
				return nil, fmt.Errorf("too many colors in char %d, no bitpairs left", char)
			}
			if prevbp, ok := img.bpcCache[char-1][col.C64Color]; ok {
				if _, ok := bp.color(prevbp); !ok {
					bp.add(prevbp, col)
					continue NEXTCOL
				}
				if char >= 40 {
					if prevbp2, ok := img.bpcCache[char-40][col.C64Color]; ok {
						if _, ok := bp.color(prevbp2); !ok {
							bp.add(prevbp2, col)
							continue NEXTCOL
						}
					}
				}
			}
		}
	}

	// finally fill or replace missing colors
	for _, col := range cc {
		if _, ok := bp.bitpair(col); ok {
			continue
		}
		if len(bp.bitpairs) == 0 {
			return bp, fmt.Errorf("too many colors in char %d, no bitpairs left", char)
		}
		//works for all general cases, but prefers bitpair 11 should be replaced first
		//bp.add(bp.bitpairs[len(bp.bitpairs)-1], col)
		// or
		//let's shift the first available one, to avoid taking bitpair 11 (d800)
		bp.add(bp.bitpairs[0], col)
	}

	return bp, nil
}

// newBitpairsFromBPColors converts img.bpc into bitpairs.
func newBitpairsFromBPColors(bpc BPColors) *bitpairs {
	bp := &bitpairs{}
	for bitp, col := range bpc {
		if col != nil {
			bp.add(byte(bitp), *col)
		}
	}
	return bp
}

// multiColorCharBytes converts the char to charBytes.
func (img *sourceImage) multiColorCharBytes(char int, bp *bitpairs) (charBytes, error) {
	b := charBytes{}
	x, y := xyFromChar(char)
	for i := 0; i < 8; i++ {
		bmpbyte := byte(0)
		for pixel := 0; pixel < 8; pixel += 2 {
			col := img.At(x+pixel, y+i)
			if bitpair, ok := bp.bitpair(col); ok {
				bmpbyte = bmpbyte | (bitpair << (6 - byte(pixel)))
			} else {
				return b, fmt.Errorf("col %s not found in char %d (x=%d y=%d)", col, char, x, y)
			}
		}
		b[i] = bmpbyte
	}
	return b, nil
}

// singleColorCharBytes converts the char to charBytes.
func (img *sourceImage) singleColorCharBytes(char int, bp *bitpairs) (charBytes, error) {
	b := charBytes{}
	x, y := xyFromChar(char)
	for i := 0; i < 8; i++ {
		bmpbyte := byte(0)
		for pixel := 0; pixel < 8; pixel++ {
			col := img.At(x+pixel, y+i)
			if bitpair, ok := bp.bitpair(col); ok {
				bmpbyte = bmpbyte | (bitpair << (7 - byte(pixel)))
			} else {
				return b, fmt.Errorf("col %s not found in char %d (x=%d y=%d)", col, char, x, y)
			}
		}
		b[i] = bmpbyte
	}
	return b, nil
}

// guessFirstBitpair2C64Color guesses by iterating over the first chars until full 4 color bitpairs are found.
func (img *sourceImage) guessFirstBitpair2C64Color() *bitpairs {
	for char := 0; char < FullScreenChars; char++ {
		x, y := xyFromChar(char)
		bp, err := img.newBitpairs(char, img.charColors[char], false)
		if err != nil {
			log.Printf("guessFirstBitpair2C64Color newBitpairs failed: error in char %d (x=%d y=%d): %v", char, x, y, err)
			continue
		}
		if bp.numColors() == 4 {
			if img.opt.Verbose {
				log.Printf("guessFirstBitpair2C64Color from first 4col char %d (x=%d y=%d): %v", char, x, y, bp.bitpair2color)
			}
			return bp
		}
	}
	return newBitpairsFromBPColors(img.bpc)
}

// Koala converts the img to Koala and returns it.
func (img *sourceImage) Koala() (Koala, error) {
	k := Koala{
		BackgroundColor: byte(img.bg.C64Color),
		BorderColor:     byte(img.border.C64Color),
		SourceFilename:  img.sourceFilename,
		opt:             img.opt,
	}
	prevbp := img.guessFirstBitpair2C64Color()
	for char := 0; char < FullScreenChars; char++ {
		x, y := xyFromChar(char)
		bp, err := img.newBitpairs(char, img.charColors[char], false)
		if err != nil {
			return k, fmt.Errorf("newBitpairs failed: error in char %d (x=%d y=%d): %w", char, x, y, err)
		}

		cbuf, err := img.multiColorCharBytes(char, bp)
		if err != nil {
			return k, fmt.Errorf("multiColorCharBytes failed: error in char %d (x=%d y=%d): %w", char, x, y, err)
		}
		for i := range cbuf {
			k.Bitmap[char*8+i] = cbuf[i]
		}

		if col, ok := bp.color(1); ok {
			k.ScreenColor[char] = byte(col.C64Color) << 4
		} else {
			if !k.opt.Trd && !k.opt.disableRepeatingBitpairColors {
				pcol, _ := prevbp.color(1)
				k.ScreenColor[char] = byte(pcol.C64Color) << 4
			}
		}
		if col, ok := bp.color(2); ok {
			k.ScreenColor[char] |= byte(col.C64Color)
		} else {
			if !k.opt.Trd && !k.opt.disableRepeatingBitpairColors {
				pcol, _ := prevbp.color(2)
				k.ScreenColor[char] |= byte(pcol.C64Color)
			}
		}
		if col, ok := bp.color(3); ok {
			k.D800Color[char] = byte(col.C64Color)
		} else if !k.opt.disableRepeatingBitpairColors {
			pcol, _ := prevbp.color(3)
			k.D800Color[char] = byte(pcol.C64Color)
		}

		if prevbp.numColors() != 4 && img.opt.Verbose {
			log.Printf("warning: char %d: prevbp numColors is not 4: %v", char, prevbp)
		}

		for bitp, col := range bp.bitpair2color {
			img.bpcBitpairCount[col.C64Color][bitp]++
			prevbp.add(bitp, col)
		}
	}
	if img.opt.VeryVerbose {
		for c64col, bpcols := range img.bpcBitpairCount {
			log.Printf("img.bpcBitpairCount: col %d: %v", c64col, bpcols)
		}
	}
	return k, nil
}

// Hires converts the img to Hires and returns it.
func (img *sourceImage) Hires() (Hires, error) {
	h := Hires{
		SourceFilename: img.sourceFilename,
		BorderColor:    byte(img.border.C64Color),
		opt:            img.opt,
	}

	prevbp := newBitpairsFromBPColors(img.bpc)
	for char := 0; char < FullScreenChars; char++ {
		x, y := xyFromChar(char)
		cc := img.charColors[char]
		if len(cc) > 2 {
			return h, fmt.Errorf("Too many hires colors in char %d (x=%d y=%d)", char, x, y)
		}
		bp, err := img.newBitpairs(char, cc, false)
		if err != nil {
			return h, fmt.Errorf("newBitpairs failed: error in char %d (x=%d y=%d): %w", char, x, y, err)
		}

		cbuf, err := img.singleColorCharBytes(char, bp)
		if err != nil {
			return h, fmt.Errorf("singleColorCharBytes failed: error in char %d (x=%d y=%d): %w", char, x, y, err)
		}
		for i := range cbuf {
			h.Bitmap[char*8+i] = cbuf[i]
		}

		if col, ok := bp.color(1); ok {
			h.ScreenColor[char] = byte(col.C64Color) << 4
		} else if !h.opt.disableRepeatingBitpairColors {
			h.ScreenColor[char] = byte(prevbp.bitpair2color[1].C64Color) << 4
		}
		if col, ok := bp.color(0); ok {
			h.ScreenColor[char] |= byte(col.C64Color)
		} else if !h.opt.disableRepeatingBitpairColors {
			h.ScreenColor[char] |= byte(prevbp.bitpair2color[0].C64Color)
		}
		for bitp, col := range bp.bitpair2color {
			prevbp.add(bitp, col)
		}
	}
	return h, nil
}

// SingleColorCharset converts the img to SingleColorCharset and returns it.
func (img *sourceImage) SingleColorCharset(prebuiltCharset []charBytes) (SingleColorCharset, error) {
	c := SingleColorCharset{
		SourceFilename: img.sourceFilename,
		BorderColor:    byte(img.border.C64Color),
		opt:            img.opt,
	}
	if len(img.bpc) == 0 {
		return c, fmt.Errorf("no bgcol? this should not happen.")
	}

	cc := img.maxColorsPerChar()
	forceBgCol := *img.bpc[0]

LOOP:
	for _, bgc := range img.bgCandidates {
		if bgc.C64Color == forceBgCol.C64Color {
			for i, col := range cc {
				if col.C64Color == forceBgCol.C64Color {
					cc[0], cc[i] = cc[i], cc[0]
					if img.opt.VeryVerbose {
						log.Printf("forced background color %d was found", forceBgCol)
					}
					break LOOP
				}
			}
		}
	}
	if forceBgCol.C64Color != cc[0].C64Color {
		return c, fmt.Errorf("forced background color %d was not found in (%v) with img.backgroundCandidates %s", forceBgCol, cc, img.bgCandidates)
	}
	c.BackgroundColor = byte(forceBgCol.C64Color)
	if len(cc) > 2 {
		return c, fmt.Errorf("too many colors: %d the max is 2", len(cc))
	}
	bp := &bitpairs{bitpairs: []byte{0, 1}}
	if len(cc) > 0 {
		bp.add(0, cc[0])
	}
	if len(cc) > 1 {
		bp.add(1, cc[1])
	}

	for i := 0; i < FullScreenChars; i++ {
		// disable for animations
		//c.D800Color[i] = bitpair2c64color[1]
	}

	if img.opt.NoPackChars {
		for char := 0; char < MaxChars; char++ {
			cbuf, err := img.singleColorCharBytes(char, bp)
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

	truecount := map[charBytes]int{}
	for char := 0; char < FullScreenChars; char++ {
		bp := &bitpairs{bitpairs: []byte{0, 1}}
		for _, col := range img.charColors[char] {
			if col.C64Color == cc[0].C64Color {
				bp.add(0, col)
			} else {
				bp.add(1, col)
				c.D800Color[char] = byte(col.C64Color)
			}
		}

		cbuf, err := img.singleColorCharBytes(char, bp)
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

// PETSCIICharset converts the img to PETSCIICharset and returns it.
func (img *sourceImage) PETSCIICharset() (PETSCIICharset, error) {
	c := PETSCIICharset{
		SourceFilename: img.sourceFilename,
		BorderColor:    byte(img.border.C64Color),
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
	cc := img.p.SortColors()
	// we must sort reverse to avoid a high color in bitpair 11
	sort.Slice(cc, func(i, j int) bool {
		return cc[i].C64Color > cc[j].C64Color
	})
	if len(cc) < 1 {
		return c, fmt.Errorf("not enough colors: %v", cc)
	}
	img.bg = cc[0]
	if len(img.bpc) == 0 {
		for i := range cc {
			col := cc[i]
			img.bpc = append(img.bpc, &col)
		}
	}

	bp, err := img.newBitpairs(0, cc, false)
	if err != nil {
		return c, fmt.Errorf("newBitpairs failed: %w", err)
	}

	if img.opt.Verbose {
		log.Printf("charset colors: %s\n", bp.colors())
		log.Printf("bitpairs: %v\n", bp)
	}
	if col, ok := bp.color(3); ok {
		if col.C64Color > 7 {
			if !img.opt.Quiet {
				return c, fmt.Errorf("the bitpair 11 can only contain colors 0-7, you will want to swap -bitpair-colors %s", img.bpc)
			}
		}
		c.CharColor = byte(col.C64Color) | 8
		for i := 0; i < FullScreenChars; i++ {
			c.D800Color[i] = c.CharColor
		}
	}

	if col, ok := bp.color(0); ok {
		c.BackgroundColor = byte(col.C64Color)
	}
	if col, ok := bp.color(1); ok {
		c.D022Color = byte(col.C64Color)
	}
	if col, ok := bp.color(2); ok {
		c.D023Color = byte(col.C64Color)
	}
	c.BorderColor = byte(img.border.C64Color)

	if img.opt.NoPackChars {
		for char := 0; char < MaxChars; char++ {
			cbuf, err := img.multiColorCharBytes(char, bp)
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
		cbuf, err := img.multiColorCharBytes(char, bp)
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

// MixedCharset converts the img to MixedCharset and returns it.
func (img *sourceImage) MixedCharset(prebuiltCharset []charBytes) (c MixedCharset, err error) {
	c.SourceFilename = img.sourceFilename
	c.BorderColor = byte(img.border.C64Color)
	c.opt = img.opt
	if img.opt.Verbose {
		log.Printf("img.MixedCharset: bpc: %v", img.bpc)
	}

	if len(img.bpc) > 3 {
		if col := img.bpc[3]; col != nil {
			if col.C64Color > 7 {
				if img.opt.Verbose {
					log.Printf("img.MixedCharset: detected charcol %d > 7, attempting to swap with another bitpair", col)
				}
				fixed := false
				for i := 2; i > 0; i-- {
					//for i := 1; i < 3; i++ {
					if img.bpc[i] == nil {
						continue
					}
					if img.bpc[i].C64Color < 8 {
						img.bpc[i], img.bpc[3] = img.bpc[3], img.bpc[i]
						fixed = true
						break
					}
				}
				if !fixed {
					return c, fmt.Errorf("could not find charcol %d to swap, required in mixed mode. try alternate -bitpair-colors", col)
				}
			}
		}
	}

	if len(img.bgCandidates) >= 0 {
		candidates := Colors{}
		for _, col := range img.bgCandidates {
			candidates = append(candidates, col)
		}
		sort.Slice(candidates, func(i, j int) bool { return candidates[i].C64Color > candidates[j].C64Color })
		if img.opt.Verbose {
			log.Printf("img.MixedCharset: candidates: %v", candidates)
		}

		fixpref := BPColors{}
		for _, p := range img.bpc {
			if p == nil {
				continue
			}
			if In(candidates, *p) && len(fixpref) < 3 {
				fixpref = append(fixpref, p)
			}
		}
		if len(fixpref) < len(candidates) {
			for _, col := range candidates {
				if img.bpc.Contains(col) && !fixpref.Contains(col) && len(fixpref) < 3 {
					c := col
					fixpref = append(fixpref, &c)
				}
			}
		}
		if len(fixpref) < len(candidates) {
			for _, col := range candidates {
				if !fixpref.Contains(col) && len(fixpref) < 3 {
					c := col
					fixpref = append(fixpref, &c)
				}
			}
		}
		img.bpc = fixpref
	}

	if img.opt.Verbose {
		log.Printf("img.MixedCharset: img.bpc: %v", img.bpc)
	}
	if len(img.bpc) > 0 {
		c.BackgroundColor = byte(img.bpc[0].C64Color)
	}
	if len(img.bpc) > 1 {
		c.D022Color = byte(img.bpc[1].C64Color)
	}
	if len(img.bpc) > 2 {
		c.D023Color = byte(img.bpc[2].C64Color)
	}
	if len(img.bpc) > 3 {
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
		bp := &bitpairs{bitpairs: []byte{0, 1, 2, 3}}
		if len(img.bpc) > 0 {
			if col := img.bpc[0]; col != nil {
				bp.add(0, *col)
			}
		}
		if len(img.bpc) > 1 {
			if col := img.bpc[1]; col != nil {
				bp.add(1, *col)
			}
		}
		if len(img.bpc) > 2 {
			if col := img.bpc[2]; col != nil {
				bp.add(2, *col)
			}
		}
		for _, col := range img.charColors[char] {
			if _, ok := bp.bitpair(col); !ok {
				bp.add(3, col)
				c.D800Color[char] = byte(col.C64Color)
				break
			}
		}
		if col, ok := bp.color(3); ok {
			c.D800Color[char] = byte(col.C64Color)
		}

		hires := false
		hirespixels := false
		charcol := C64Color(0)
		x, y := xyFromChar(char)
		if len(img.charColors[char]) <= 2 {
			// could be hires
		LOOP:
			for y2 := 0; y2 < 8; y2++ {
				for x2 := 0; x2 < 8; x2 += 2 {
					if img.At(x+x2, y+y2) != img.At(x+x2+1, y+y2) {
						hirespixels = true
						break LOOP
					}
				}
			}
			for _, bgcol := range img.charColors[char] {
				if bgcol.C64Color == C64Color(c.BackgroundColor) {
					for _, col := range img.charColors[char] {
						if col.C64Color != bgcol.C64Color && col.C64Color < 8 {
							hires = true
							charcol = col.C64Color
							c.D800Color[char] = byte(col.C64Color)
							bp = &bitpairs{bitpairs: []byte{0, 1}}
							bp.add(0, bgcol)
							bp.add(1, col)
							break
						}
					}
					break
				}
			}
		}

		if hirespixels && !hires {
			return c, fmt.Errorf("found hirespixels in char %d (x=%d y=%d), but colors are bad: %s please swap some -bitpair-colors %s", char, x, y, img.charColors[char], img.bpc)
		}

		var cbuf charBytes
		emptyChar := charBytes{}
		if hires {
			if img.opt.VeryVerbose {
				log.Printf("char %d (x=%d y=%d) seems to be hires, charcol %d img.charColors: %s -bpc %s", char, x, y, charcol, img.charColors[char], img.bpc)
			}
			cbuf, err = img.singleColorCharBytes(char, bp)
			if err != nil {
				return c, fmt.Errorf("singleColorCharBytes failed: error in char %d (x=%d y=%d): %w", char, x, y, err)
			}
		} else {
			c.D800Color[char] |= 8
			cbuf, err = img.multiColorCharBytes(char, bp)
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
		fmt.Printf("settled for -bitpair-colors %s\n", img.bpc)
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
		BorderColor:     byte(img.border.C64Color),
		BackgroundColor: byte(img.ecmColors[0].C64Color),
		opt:             img.opt,
	}
	if len(img.ecmColors) > 1 {
		c.D022Color = byte(img.ecmColors[1].C64Color)
	}
	if len(img.ecmColors) > 2 {
		c.D023Color = byte(img.ecmColors[2].C64Color)
	}
	if len(img.ecmColors) > 3 {
		c.D024Color = byte(img.ecmColors[3].C64Color)
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
		var orchar byte
		var foundbg bool
		var emptycharcol C64Color
		// when 2 ecm colors are used in the same char, which color to choose for bitpair 00?
		// good example: testdata/ecm/orion.png testdata/ecm/xpardey.png
		// so now we sort to at least make it deterministic.
		// this is a bit later mitigated by trying to detect flippable chars.
		bp := &bitpairs{bitpairs: []byte{0, 1}}
		for _, col := range img.charColors[char] {
			match := false
			matchi := 0
			for i, col2 := range img.ecmColors {
				if col.C64Color == col2.C64Color {
					match = true
					matchi = i
					break
				}
			}
			if match && !foundbg {
				bp.add(0, col)
				orchar = byte(matchi << 6)
				foundbg = true
				emptycharcol = col.C64Color
			} else {
				bp.add(1, col)
				c.D800Color[char] = byte(col.C64Color)
			}
		}
		if len(img.charColors[char]) == 2 && !foundbg {
			return c, fmt.Errorf("background ecm color not found in char %d (x=%d y=%d)", char, x, y)
		}

		cbuf, err := img.singleColorCharBytes(char, bp)
		if err != nil {
			return c, fmt.Errorf("singleColorCharBytes failed: error in char %d (x=%d y=%d): %w", char, x, y, err)
		}
		if !img.opt.NoPackEmptyChar {
			if cbuf == emptyChar {
				// use bitpair 11 for empty chars, usually saves 1 char
				// good example: testdata/ecm/shampoo.png
				cbuf = charBytes{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
				c.D800Color[char] = byte(emptycharcol)
			}
		}

		curChar := slices.Index(charset, cbuf)
		if curChar < 0 {
			flipped := false
			if charcol, ok := bp.color(1); ok {
				if ecmindex := slices.Index(img.ecmColors, charcol); ecmindex >= 0 {
					// try to find a bitflipped char as both background and foreground colors are ecmcolors.
					tmpchar := cbuf.Invert()
					flipChar := slices.Index(charset, tmpchar)
					if flipChar >= 0 {
						if img.opt.VeryVerbose {
							log.Printf("ecmColors: in char %d (x=%d y=%d): flip char %d ecmindex: %d ecm: %s bp: %s", char, x, y, flipChar, ecmindex, img.ecmColors, bp.colors().BPColors())
						}
						curChar = flipChar
						bgcol, _ := bp.color(0)
						c.D800Color[char] = byte(bgcol.C64Color)
						orchar = byte(ecmindex << 6)
						flipped = true
					}
				}
			}
			if !flipped {
				truecount[cbuf]++
				charset = append(charset, cbuf)
				curChar = len(charset) - 1
			}
		}
		c.Screen[char] = byte(curChar) + orchar
	}

	// detect flippers
	for i, char := range charset {
		tmpchar := char.Invert()
		if other := slices.Index(charset, tmpchar); other >= 0 {
			_ = i
			//fmt.Printf("could flip char %d with char %d\n", i, other)
			//fmt.Print(char)
			//fmt.Println("----")
			//fmt.Print(tmpchar)
			//fmt.Printf("char:\n%s\n", char)
			//fmt.Printf("inverted char:\n%s\n", tmpchar)
			/*
				for k := range c.Screen {
					if c.Screen[k] == byte(i) {
						ink := img.p.FromC64NoErr(C64Color(c.D800Color[k]))
						if In(img.ecmColors, ink) {
							fmt.Printf("char %d (num %d) ink %s ecm %s\n", k, c.Screen[k], ink, img.ecmColors)
						}
					}
				}
			*/
		}
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

	cc := img.p.SortColors()
	forceBgCol := -1
	if len(img.bpc) > 0 {
		forceBgCol = int(img.bpc[0].C64Color)
	}
	if forceBgCol >= 0 {
		for i := range cc {
			if cc[i].C64Color == C64Color(forceBgCol) {
				cc[0], cc[i] = cc[i], cc[0]
				if img.opt.Verbose {
					log.Printf("forced background color %d was found", forceBgCol)
				}
				break
			}
		}
	}

	bp := &bitpairs{bitpairs: []byte{0, 1}}
	s.BackgroundColor = byte(cc[0].C64Color)
	bp.add(0, cc[0])
	if len(cc) > 1 {
		s.SpriteColor = byte(cc[1].C64Color)
		bp.add(1, cc[1])
	}
	if len(cc) > 2 {
		return s, fmt.Errorf("Too many colors.")
	}

	if img.opt.Verbose {
		log.Printf("sprite colors: %v\n", cc)
		log.Printf("bitpairs: %v\n", bp)
	}

	for spriteY := 0; spriteY < maxY; spriteY++ {
		for spriteX := 0; spriteX < maxX; spriteX++ {
			for y := 0; y < SpriteHeight; y++ {
				yOffset := y + spriteY*SpriteHeight
				for x := 0; x < 3; x++ {
					xOffset := x*8 + spriteX*SpriteWidth
					bmpbyte := byte(0)
					for pixel := 0; pixel < 8; pixel++ {
						col := img.At(xOffset+pixel, yOffset)
						if bitpair, ok := bp.bitpair(col); ok {
							bmpbyte = bmpbyte | (bitpair << (7 - byte(pixel)))
						} else {
							return s, fmt.Errorf("col %v not found in x %d, u %d.", col, x, y)
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

	cc := img.p.SortColors()
	if len(img.bpc) == 0 {
		for _, col := range cc {
			img.bpc = append(img.bpc, &col)
		}
	}

	bp, err := img.newBitpairs(0, cc, false)
	if err != nil {
		return s, fmt.Errorf("img.newBitpairs failed: %v", err)
	}

	if img.opt.Verbose {
		log.Printf("sprite colors: %v\n", cc)
		log.Printf("bitpairs: %v\n", bp)
	}

	switch {
	case len(img.bpc) > 3:
		if img.bpc[3] != nil {
			s.D026Color = byte(img.bpc[3].C64Color)
		}
		fallthrough
	case len(img.bpc) > 2:
		if img.bpc[2] != nil {
			s.SpriteColor = byte(img.bpc[2].C64Color)
		}
		fallthrough
	case len(img.bpc) > 1:
		if img.bpc[1] != nil {
			s.D025Color = byte(img.bpc[1].C64Color)
		}
		fallthrough
	case len(img.bpc) > 0:
		if img.bpc[0] != nil {
			s.BackgroundColor = byte(img.bpc[0].C64Color)
		}
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
						col := img.At(xOffset+pixel, yOffset)
						if bitpair, ok := bp.bitpair(col); ok {
							bmpbyte |= bitpair << (6 - byte(pixel))
						} else {
							return s, fmt.Errorf("col %v not found in x %d, u %d.", col, x, y)
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
