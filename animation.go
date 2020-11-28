package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func handleAnimation(ff []string) error {
	var kk []Koala
	for _, f := range ff {
		img, err := newSourceImage(f)
		if err != nil {
			return fmt.Errorf("newSourceImage %q failed: %v", f, err)
		}
		err = img.analyze()
		if err != nil {
			return fmt.Errorf("analyze %q failed: %v", f, err)
		}

		switch img.graphicsType {
		case multiColorBitmap:
			k, err := img.convertToKoala()
			if err != nil {
				return fmt.Errorf("convertToKoala %q failed: %v", f, err)
			}
			kk = append(kk, k)
		default:
			return fmt.Errorf("unsupported graphicsType %q", f)
		}
	}

	if len(kk) == 0 {
		log.Print("nothing to do")
		return nil
	}
	destFilename := getDestinationFilename(kk[0].SourceFilename)
	f, err := os.Create(destFilename)
	if err != nil {
		return fmt.Errorf("os.Create %q failed: %v", destFilename, err)
	}
	defer f.Close()
	_, err = kk[0].WriteTo(f)
	if err != nil {
		return fmt.Errorf("WriteTo %q failed: %v", destFilename, err)
	}
	if !quiet {
		fmt.Printf("converted %q to %q\n", kk[0].SourceFilename, destFilename)
	}

	prgs, err := ProcessAnimation(kk)
	if err != nil {
		return fmt.Errorf("ProcessAnimation failed: %v", err)
	}

	for i, prg := range prgs {
		if err = writePrgFile(frameFilename(i, kk[0].SourceFilename), prg); err != nil {
			return fmt.Errorf("writePrgFile failed: %v", err)
		}
	}
	return nil
}

func frameFilename(i int, filename string) string {
	d := getDestinationFilename(filename)
	return strings.TrimSuffix(d, filepath.Ext(d)) + ".frame" + strconv.Itoa(i) + ".prg"
}

func writePrgFile(filename string, prg []byte) error {
	if verbose {
		log.Printf("going to write file %q", filename)
	}
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("os.Create %q failed: %v", filename, err)
	}
	defer f.Close()
	if _, err = f.Write([]byte{0x00, 0x20}); err != nil {
		return fmt.Errorf("Write startaddress to %q failed: %v", filename, err)
	}
	if _, err = f.Write(prg[:]); err != nil {
		return fmt.Errorf("Write prg to %q failed: %v", filename, err)
	}

	if !quiet {
		fmt.Printf("write %q\n", filename)
	}
	return nil
}

func ProcessAnimation(kk []Koala) ([][]byte, error) {
	if len(kk) < 2 {
		return nil, fmt.Errorf("ProcessAnimation: Insufficient number of images %d < 2", len(kk))
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
	return exportAnims(anims), nil
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
	c := chunk{
		CharIndex: charIndex,
		BitmapLo:  byte((charIndex * 8) & 0xff),
		BitmapHi:  byte((charIndex * 8) >> 8),
		CharLo:    byte(charIndex & 0xff),
		CharHi:    byte((charIndex - charIndex&0xff) >> 8),
		Chars:     make([]byte, 0),
	}
	return c
}

func (c *chunk) appendChar(char MultiColorChar) {
	c.Chars = append(c.Chars, char.Bitmap[:]...)
	c.Chars = append(c.Chars, char.ScreenColor, char.D800Color)
	c.CharCount++
}

func (c *chunk) export() []byte {
	bin := []byte{c.CharCount, c.BitmapLo, c.BitmapHi, c.CharLo, c.CharHi}
	return append(bin, c.Chars...)
}

func (c *chunk) String() string {
	return fmt.Sprintf("chunk charindex: %d charcount %d bitmap: $%x char: $%x\n", c.CharIndex, c.CharCount, int(c.BitmapHi)*256+int(c.BitmapLo), int(c.CharHi)*256+int(c.CharLo)) +
		fmt.Sprintf("%v", c.Chars)
}

func exportAnims(anims [][]MultiColorChar) [][]byte {
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
				curChunk.appendChar(char)
				curChar = char.CharIndex
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
				curChar = char.CharIndex
			}
		}
		// add last chunk
		if curChunk.CharCount > 0 {
			if verbose {
				log.Printf("curChunk: %s", curChunk.String())
			}
			prg = append(prg, curChunk.export()...)
		}

		prg = append(prg, 0x00)
		prgs = append(prgs, prg)
	}
	prgs[len(prgs)-1] = append(prgs[len(prgs)-1])
	return prgs
}

func (k *Koala) MultiColorChar(charIndex int) MultiColorChar {
	mc := MultiColorChar{
		CharIndex:   charIndex,
		Bitmap:      [8]byte{},
		BgColor:     k.BgColor,
		ScreenColor: k.ScreenColor[charIndex],
		D800Color:   k.D800Color[charIndex],
	}
	for i := range mc.Bitmap {
		mc.Bitmap[i] = k.Bitmap[charIndex*8+i]
	}
	return mc
}
