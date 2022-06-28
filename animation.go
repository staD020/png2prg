package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func handleAnimation(imgs []sourceImage) error {
	var kk []Koala
	var hh []Hires
	var scSprites []SingleColorSprites
	var mcSprites []MultiColorSprites
	if len(imgs) < 1 {
		return fmt.Errorf("no sourceImage given")
	}
	for i, img := range imgs {
		if verbose {
			log.Printf("processing %q frame %d\n", img.sourceFilename, i)
		}
		if err := img.analyze(); err != nil {
			log.Printf("warning: skipping frame %d, analyze failed: %v", i, err)
			continue
		}
		switch img.graphicsType {
		case multiColorBitmap:
			k, err := img.convertToKoala()
			if err != nil {
				return fmt.Errorf("convertToKoala failed: %w", err)
			}
			kk = append(kk, k)
		case singleColorBitmap:
			h, err := img.convertToHires()
			if err != nil {
				return fmt.Errorf("convertToHires failed: %w", err)
			}
			hh = append(hh, h)
		case multiColorSprites:
			s, err := img.convertToMultiColorSprites()
			if err != nil {
				return fmt.Errorf("convertToMultiColorSprites failed: %w", err)
			}
			mcSprites = append(mcSprites, s)
		case singleColorSprites:
			s, err := img.convertToSingleColorSprites()
			if err != nil {
				return fmt.Errorf("convertToSingleColorSprites failed: %w", err)
			}
			scSprites = append(scSprites, s)
		default:
			return fmt.Errorf("animations do not support %q yet", img.graphicsType)
		}
	}

	destFilename := destinationFilename(imgs[0].sourceFilename)
	f, err := os.Create(destFilename)
	if err != nil {
		return fmt.Errorf("os.Create %q failed: %w", destFilename, err)
	}
	defer f.Close()

	if display && len(kk) > 0 {
		// handle display koala animation
		_, err := WriteKoalaDisplayAnimTo(f, kk)
		if err != nil {
			return fmt.Errorf("WriteKoalaDisplayAnimTo %q failed: %w", f.Name(), err)
		}
		return nil
	}

	switch {
	case len(kk) > 0:
		_, err = kk[0].WriteTo(f)
		if err != nil {
			return fmt.Errorf("WriteTo %q failed: %w", destFilename, err)
		}
		if !quiet {
			fmt.Printf("converted %q to %q\n", kk[0].SourceFilename, destFilename)
		}

		prgs, err := processKoalaAnimation(kk)
		if err != nil {
			return fmt.Errorf("processKoalaAnimation failed: %w", err)
		}

		for i, prg := range prgs {
			if err = writePrgFile(frameFilename(i, kk[0].SourceFilename), prg); err != nil {
				return fmt.Errorf("writePrgFile failed: %w", err)
			}
		}
		return nil
	case len(hh) > 0:
		_, err = hh[0].WriteTo(f)
		if err != nil {
			return fmt.Errorf("WriteTo %q failed: %w", destFilename, err)
		}
		if !quiet {
			fmt.Printf("converted %q to %q\n", hh[0].SourceFilename, destFilename)
		}

		prgs, err := processHiresAnimation(hh)
		if err != nil {
			return fmt.Errorf("processHiresAnimation failed: %w", err)
		}
		for i, prg := range prgs {
			if err = writePrgFile(frameFilename(i, hh[0].SourceFilename), prg); err != nil {
				return fmt.Errorf("writePrgFile failed: %w", err)
			}
		}
		return nil
	case len(mcSprites) > 0:
		header := defaultHeader()
		_, err = writeData(f, [][]byte{header})
		if err != nil {
			return fmt.Errorf("writeData %q failed: %w", destFilename, err)
		}
		for _, s := range mcSprites {
			_, err = writeData(f, [][]byte{s.Bitmap})
			if err != nil {
				return fmt.Errorf("writeData %q failed: %w", destFilename, err)
			}
			if !quiet {
				fmt.Printf("converted %q to %q\n", s.SourceFilename, destFilename)
			}
		}
		return nil
	case len(scSprites) > 0:
		header := defaultHeader()
		_, err = writeData(f, [][]byte{header})
		if err != nil {
			return fmt.Errorf("writeData %q failed: %w", destFilename, err)
		}
		for _, s := range scSprites {
			_, err = writeData(f, [][]byte{s.Bitmap})
			if err != nil {
				return fmt.Errorf("writeData %q failed: %w", destFilename, err)
			}
			if !quiet {
				fmt.Printf("converted %q to %q\n", s.SourceFilename, destFilename)
			}
		}
		return nil
	}
	return fmt.Errorf("handleAnimation %q failed: no frames written", imgs[0].sourceFilename)
}

func frameFilename(i int, filename string) string {
	d := destinationFilename(filename)
	return strings.TrimSuffix(d, filepath.Ext(d)) + ".frame" + strconv.Itoa(i) + ".prg"
}

func writePrgFile(filename string, prg []byte) error {
	if verbose {
		log.Printf("going to write file %q", filename)
	}
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("os.Create %q failed: %w", filename, err)
	}
	defer f.Close()

	_, err = writeData(f, [][]byte{defaultHeader(), prg})
	if err != nil {
		return fmt.Errorf("writeData %q failed: %w", filename, err)
	}
	if !quiet {
		fmt.Printf("write %q\n", filename)
	}
	return nil
}

func processKoalaAnimation(kk []Koala) ([][]byte, error) {
	if len(kk) < 2 {
		return nil, fmt.Errorf("insufficient number of images %d < 2", len(kk))
	}
	if verbose {
		log.Printf("total number of frames: %d", len(kk))
	}

	anims := make([][]MultiColorChar, len(kk))
	for i := 0; i < len(kk)-1; i++ {
		anims[i] = make([]MultiColorChar, 0)
	}

	for i := 0; i < 1000; i++ {
		for j := 0; j < len(kk); j++ {
			k := len(kk) - 1
			if j > 0 {
				k = j - 1
			}
			prevChar := kk[k].MultiColorChar(i)
			frameChar := kk[j].MultiColorChar(i)
			if prevChar != frameChar {
				anims[j] = append(anims[j], frameChar)
			}
		}
	}
	return exportKoalaAnims(anims), nil
}

func WriteKoalaDisplayAnimTo(w io.Writer, kk []Koala) (n int64, err error) {
	bgBorder := kk[0].BackgroundColor | kk[0].BorderColor<<4
	header := append([]byte{}, koalaDisplayAnim...)
	if includeSID == "" {
		buf := make([]byte, 0, 64*1024)

		header = zeroFill(header, 0x2000-0x7ff-len(header))
		k := kk[0]
		framePrgs, err := processKoalaAnimation(kk)
		if err != nil {
			return n, err
		}
		out := [][]byte{header, k.Bitmap[:], k.ScreenColor[:], k.D800Color[:], {bgBorder}}

		for _, bin := range out {
			buf = append(buf, bin...)
		}
		buf = zeroFill(buf, 0x4800-0x7ff-len(buf))
		for _, bin := range framePrgs {
			buf = append(buf, bin...)
		}
		buf = append(buf, 0xff)
		m, err := w.Write(buf)
		n += int64(m)
		return n, err
	}
	return n, nil
}

// exportAnims format:
//      number of chars			1 byte
//		bitmap_lo + bitmap_hi	2 byte
//		char_lo, char_hi 		2 byte
//			pixels					8 byte
//	    	screencol				1 byte
//			d800col					1 byte
// total bytes: 5 + 10 * charcount

type chunk struct {
	CharIndex int
	CharCount byte
	BitmapLo  byte
	BitmapHi  byte
	CharLo    byte
	CharHi    byte
	Chars     []byte
}

func newChunk(charIndex int) chunk {
	return chunk{
		CharIndex: charIndex,
		BitmapLo:  byte((charIndex * 8) & 0xff),
		BitmapHi:  byte((charIndex * 8) >> 8),
		CharLo:    byte(charIndex & 0xff),
		CharHi:    byte((charIndex - charIndex&0xff) >> 8),
		Chars:     make([]byte, 0),
	}
}

func (c *chunk) appendChar(char MultiColorChar) {
	c.Chars = append(c.Chars, char.Bitmap[:]...)
	c.Chars = append(c.Chars, char.ScreenColor, char.D800Color)
	c.CharCount++
}

func (c *chunk) appendHiresChar(char SingleColorChar) {
	c.Chars = append(c.Chars, char.Bitmap[:]...)
	c.Chars = append(c.Chars, char.ScreenColor)
	c.CharCount++
}

func (c *chunk) export() []byte {
	return append([]byte{c.CharCount, c.BitmapLo, c.BitmapHi, c.CharLo, c.CharHi}, c.Chars...)
}

func (c *chunk) String() string {
	return fmt.Sprintf("chunk charindex: %d charcount %d bitmap: $%x char: $%x", c.CharIndex, c.CharCount, int(c.BitmapHi)*256+int(c.BitmapLo), int(c.CharHi)*256+int(c.CharLo))
}

func exportKoalaAnims(anims [][]MultiColorChar) [][]byte {
	prgs := make([][]byte, 0)
	for _, anim := range anims {
		if verbose {
			log.Println("frame length in changed chars:", len(anim))
		}

		curChar := -10
		curChunk := chunk{}
		prg := []byte{}
		for _, char := range anim {
			switch {
			case curChar == char.CharIndex-1:
				// next char of current chunk
				curChunk.appendChar(char)
			default:
				// new chunk
				if curChunk.CharCount > 0 {
					if verbose {
						log.Println(curChunk.String())
					}
					prg = append(prg, curChunk.export()...)
				}
				curChunk = newChunk(char.CharIndex)
				curChunk.appendChar(char)
			}
			curChar = char.CharIndex
		}
		// add last chunk
		if curChunk.CharCount > 0 {
			if verbose {
				log.Printf("curChunk: %s", curChunk.String())
			}
			prg = append(prg, curChunk.export()...)
		}

		// end of chunk marker
		prg = append(prg, 0x00)
		prgs = append(prgs, prg)
	}
	return prgs
}

func (k *Koala) MultiColorChar(charIndex int) MultiColorChar {
	mc := MultiColorChar{
		CharIndex:       charIndex,
		Bitmap:          [8]byte{},
		BackgroundColor: k.BackgroundColor,
		ScreenColor:     k.ScreenColor[charIndex],
		D800Color:       k.D800Color[charIndex],
	}
	for i := range mc.Bitmap {
		mc.Bitmap[i] = k.Bitmap[charIndex*8+i]
	}
	return mc
}

func (h *Hires) SingleColorChar(charIndex int) SingleColorChar {
	sc := SingleColorChar{
		CharIndex:   charIndex,
		Bitmap:      [8]byte{},
		ScreenColor: h.ScreenColor[charIndex],
	}
	for i := range sc.Bitmap {
		sc.Bitmap[i] = h.Bitmap[charIndex*8+i]
	}
	return sc
}

func processHiresAnimation(hh []Hires) ([][]byte, error) {
	if len(hh) < 2 {
		return nil, fmt.Errorf("insufficient number of images %d < 2", len(hh))
	}
	if verbose {
		log.Printf("total number of frames: %d", len(hh))
	}

	anims := make([][]SingleColorChar, len(hh))
	for i := 0; i < len(hh)-1; i++ {
		anims[i] = make([]SingleColorChar, 0)
	}

	for i := 0; i < 1000; i++ {
		for j := 0; j < len(hh); j++ {
			k := len(hh) - 1
			if j > 0 {
				k = j - 1
			}
			prevChar := hh[k].SingleColorChar(i)
			frameChar := hh[j].SingleColorChar(i)
			if prevChar != frameChar {
				anims[j] = append(anims[j], frameChar)
			}
		}
	}
	return exportHiresAnims(anims), nil
}

func exportHiresAnims(anims [][]SingleColorChar) [][]byte {
	prgs := make([][]byte, 0)
	for _, anim := range anims {
		if verbose {
			log.Println("frame length in changed chars:", len(anim))
		}

		curChar := -10
		curChunk := chunk{}
		prg := []byte{}
		for _, char := range anim {
			switch {
			case curChar == char.CharIndex-1:
				// next char of current chunk
				curChunk.appendHiresChar(char)
			default:
				// new chunk
				if curChunk.CharCount > 0 {
					if verbose {
						log.Println(curChunk.String())
					}
					prg = append(prg, curChunk.export()...)
				}
				curChunk = newChunk(char.CharIndex)
				curChunk.appendHiresChar(char)
			}
			curChar = char.CharIndex
		}
		// add last chunk
		if curChunk.CharCount > 0 {
			if verbose {
				log.Printf("curChunk: %s", curChunk.String())
			}
			prg = append(prg, curChunk.export()...)
		}

		// end of chunk marker
		prg = append(prg, 0x00)
		prgs = append(prgs, prg)
	}
	return prgs
}
