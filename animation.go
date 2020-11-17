package main

import (
	"fmt"
	"log"
)

type multiColorChar struct {
	CharIndex   int
	Bitmap      [8]byte
	BgColor     byte
	ScreenColor byte
	D800Color   byte
}

func ProcessAnimation(kk []Koala) [][]byte {
	if len(kk) < 2 {
		log.Fatalf("ProcessAnimation: Insufficient number of images %d < 2", len(kk))
	}
	if verbose {
		log.Printf("total number of frames: %d", len(kk))
	}

	anims := make([][]multiColorChar, len(kk))
	for j := 1; j < len(kk); j++ {
		anims[j-1] = make([]multiColorChar, 0)
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
	return exportAnims(anims)
}

// exportAnims format:
//      number of chars			1 byte
//		bitmap_lo + bitmap_hi	2 byte
//		char_lo, char_hi 		2 byte
//			pixels					8 byte
//	    	screencol				1 byte
//			d800col					1 byte
// total bytes: 5 + 10 * charcount
const chunkHeaderSize = 5
const chunkCharSize = 10

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

func (c *chunk) appendChar(char multiColorChar) {
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

func exportAnims(anims [][]multiColorChar) [][]byte {
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
				log.Printf("curChunk: %s\n", curChunk.String())
			}
			prg = append(prg, curChunk.export()...)
		}

		prg = append(prg, 0x00)
		prgs = append(prgs, prg)
	}
	prgs[len(prgs)-1] = append(prgs[len(prgs)-1])
	return prgs
}

func (k *Koala) MultiColorChar(charIndex int) multiColorChar {
	mc := multiColorChar{
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
