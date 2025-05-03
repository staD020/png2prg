package png2prg

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/staD020/TSCrunch"
)

const (
	koalaFadePassStart  = 0x8900
	koalaAnimationStart = 0x4800

	hiresFadePassStart  = 0xac00
	hiresAnimationStart = 0x4400
)

// WriteAnimationTo processes all images and writes the resulting .prg to w.
func (c *Converter) WriteAnimationTo(w io.Writer) (n int64, err error) {
	var kk []Koala
	var hh []Hires
	var scSprites []SingleColorSprites
	var mcSprites []MultiColorSprites
	var scCharsets []SingleColorCharset
	var petCharsets []PETSCIICharset
	var mcCharsets []MultiColorCharset
	var mixCharsets []MixedCharset
	imgs := c.images
	if len(imgs) < 1 {
		return n, fmt.Errorf("no sourceImage given")
	}

	bruteforce := func(gfxtype GraphicsType, maxColors int) error {
		if !c.opt.BruteForce {
			return nil
		}
		if err = c.BruteForceBitpairColors(gfxtype, maxColors); err != nil {
			return fmt.Errorf("BruteForceBitpairColors %q failed: %w", imgs[0].sourceFilename, err)
		}
		if err = imgs[0].setPreferredBitpairColors(c.opt.BitpairColorsString); err != nil {
			return fmt.Errorf("img.setPreferredBitpairColors %q failed: %w", c.opt.BitpairColorsString, err)
		}
		return nil
	}

	if imgs[0].graphicsType == multiColorBitmap {
		err = bruteforce(imgs[0].graphicsType, 4)
	}
	if imgs[0].graphicsType == singleColorBitmap {
		err = bruteforce(imgs[0].graphicsType, 2)
	}
	if err != nil {
		log.Printf("bruteforce failed: %f", err)
	}

	wantedGraphicsType := imgs[0].graphicsType
	c.FinalGraphicsType = imgs[0].graphicsType
	currentBitpairColors := BPColors{}
	charset := []charBytes{}

	for i, img := range imgs {
		if !c.opt.Quiet {
			fmt.Printf("processing %q frame %d\n", img.sourceFilename, i)
		}
		if i > 0 {
			if imgs[0].graphicsType != petsciiCharset && imgs[0].graphicsType != singleColorCharset {
				img.bpc = currentBitpairColors
			}
			if err := img.analyze(); err != nil {
				log.Printf("warning: skipping frame %d, analyze failed: %v", i, err)
				continue
			}
		}
		if img.graphicsType != wantedGraphicsType {
			return n, fmt.Errorf("mixed graphicsmodes detected %q != %q", img.graphicsType, wantedGraphicsType)
		}
		if len(currentBitpairColors) == 0 {
			currentBitpairColors = img.bpc
		}
		if currentBitpairColors.String() != img.bpc.String() && imgs[0].graphicsType != petsciiCharset && imgs[0].graphicsType != singleColorCharset {
			log.Printf("bitpairColors %q of the previous frame do not equal current frame %q", currentBitpairColors, img.bpc)
			log.Println("this would cause huge animation frame sizes and probably crash the displayer")
			return n, fmt.Errorf("bitpairColors differ between frames, maybe use -bitpair-colors %s to force them", currentBitpairColors)
		}

		switch img.graphicsType {
		case multiColorBitmap:
			k, err := img.Koala()
			if err != nil {
				return n, fmt.Errorf("img.Koala failed: %w", err)
			}
			kk = append(kk, k)
		case singleColorBitmap:
			h, err := img.Hires()
			if err != nil {
				return n, fmt.Errorf("img.Hires failed: %w", err)
			}
			hh = append(hh, h)
		case multiColorSprites:
			s, err := img.MultiColorSprites()
			if err != nil {
				return n, fmt.Errorf("img.MultiColorSprites failed: %w", err)
			}
			mcSprites = append(mcSprites, s)
		case singleColorSprites:
			s, err := img.SingleColorSprites()
			if err != nil {
				return n, fmt.Errorf("img.SingleColorSprites failed: %w", err)
			}
			scSprites = append(scSprites, s)
		case multiColorCharset:
			ch, err := img.MultiColorCharset(charset)
			if err != nil {
				return n, fmt.Errorf("img.multiColorCharset failed: %w", err)
			}
			mcCharsets = append(mcCharsets, ch)
			charset = ch.CharBytes()
		case singleColorCharset:
			if c.opt.GraphicsMode == "sccharset" {
				ch, err := img.SingleColorCharset(charset)
				if err != nil {
					return n, fmt.Errorf("img.SingleColorCharset failed: %w", err)
				}
				scCharsets = append(scCharsets, ch)
				charset = ch.CharBytes()
				break
			}
			if pet, err := img.PETSCIICharset(); err != nil {
				ch, err := img.SingleColorCharset(charset)
				if err != nil {
					return n, fmt.Errorf("img.SingleColorCharset failed: %w", err)
				}
				scCharsets = append(scCharsets, ch)
				charset = ch.CharBytes()
			} else {
				c.FinalGraphicsType = petsciiCharset
				petCharsets = append(petCharsets, pet)
			}
		case petsciiCharset:
			if c.opt.GraphicsMode == "sccharset" {
				ch, err := img.SingleColorCharset(charset)
				if err != nil {
					return n, fmt.Errorf("img.SingleColorCharset failed: %w", err)
				}
				scCharsets = append(scCharsets, ch)
				charset = ch.CharBytes()
				break
			}
			pet, err := img.PETSCIICharset()
			if err != nil {
				return n, fmt.Errorf("img.SingleColorCharset failed: %w", err)
			}
			c.FinalGraphicsType = petsciiCharset
			petCharsets = append(petCharsets, pet)
		case mixedCharset:
			ch, err := img.MixedCharset(charset)
			if err != nil {
				return n, fmt.Errorf("img.MixedCharset failed: %w", err)
			}
			mixCharsets = append(mixCharsets, ch)
			charset = ch.CharBytes()
		default:
			return n, fmt.Errorf("animations do not support %q yet", img.graphicsType)
		}
	}

	if c.opt.Display {
		m, err := c.writeAnimationDisplayerTo(w, imgs, kk, hh, scSprites, mcSprites, mcCharsets, scCharsets, petCharsets, mixCharsets)
		n += m
		if err != nil {
			return n, fmt.Errorf("writeAnimationDisplayerTo failed: %w", err)
		}
		return n, nil
	}

	// export separate frame data (non displayer)
	switch {
	case len(kk) > 0:
		m, err := kk[0].WriteTo(w)
		n += m
		if err != nil {
			return n, fmt.Errorf("WriteTo %q failed: %w", c.opt.OutFile, err)
		}
		if !c.opt.Quiet {
			fmt.Printf("converted %q to %q\n", kk[0].SourceFilename, c.opt.OutFile)
		}
		c.Symbols = append(c.Symbols, kk[0].Symbols()...)
		c.Symbols = append(c.Symbols, c64Symbol{"animation", koalaAnimationStart})

		frames := makeCharer(kk)
		prgs, err := processAnimation(c.opt, frames)
		if err != nil {
			return n, fmt.Errorf("processAnimation failed: %w", err)
		}

		for i := range prgs {
			m, err := w.Write(prgs[i])
			n += int64(m)
			if err != nil {
				return n, fmt.Errorf("Write failed: %w", err)
			}
			n2, err := c.WriteFrameDelayByte(w, i, len(prgs))
			n += n2
			if err != nil {
				return n, fmt.Errorf("Write failed: %w", err)
			}
		}
		return n, nil
	case len(hh) > 0:
		m, err := hh[0].WriteTo(w)
		n += int64(m)
		if err != nil {
			return n, fmt.Errorf("WriteTo %q failed: %w", c.opt.OutFile, err)
		}
		if !c.opt.Quiet {
			fmt.Printf("converted %q to %q\n", hh[0].SourceFilename, c.opt.OutFile)
		}
		c.Symbols = append(c.Symbols, hh[0].Symbols()...)
		c.Symbols = append(c.Symbols, c64Symbol{"animation", hiresAnimationStart})

		frames := makeCharer(hh)
		prgs, err := processAnimation(c.opt, frames)
		if err != nil {
			return n, fmt.Errorf("processHiresAnimation failed: %w", err)
		}
		for i := range prgs {
			m, err := w.Write(prgs[i])
			n += int64(m)
			if err != nil {
				return n, fmt.Errorf("Write failed: %w", err)
			}
			n2, err := c.WriteFrameDelayByte(w, i, len(prgs))
			n += n2
			if err != nil {
				return n, fmt.Errorf("Write failed: %w", err)
			}
		}
		return n, nil
	case len(mcSprites) > 0:
		data := [][]byte{defaultHeader()}
		for _, s := range mcSprites {
			data = append(data, s.Bitmap)
			if !c.opt.Quiet {
				fmt.Printf("converted %q to %q\n", s.SourceFilename, c.opt.OutFile)
			}
		}
		c.Symbols = append(c.Symbols, mcSprites[0].Symbols()...)
		if _, err = writeData(w, data...); err != nil {
			return n, fmt.Errorf("writeData %q failed: %w", c.opt.OutFile, err)
		}
		return n, nil
	case len(scSprites) > 0:
		data := [][]byte{defaultHeader()}
		for _, s := range scSprites {
			data = append(data, s.Bitmap)
			if !c.opt.Quiet {
				fmt.Printf("converted %q to %q\n", s.SourceFilename, c.opt.OutFile)
			}
		}
		c.Symbols = append(c.Symbols, scSprites[0].Symbols()...)
		if _, err = writeData(w, data...); err != nil {
			return n, fmt.Errorf("writeData %q failed: %w", c.opt.OutFile, err)
		}
		return n, nil
	case len(mcCharsets) > 0:
		if c.opt.NoAnimation {
			c.Symbols = append(c.Symbols,
				c64Symbol{"d800color", 0x3c00},
				c64Symbol{"bitmap", 0x4000},
				c64Symbol{"d020color", int(mcCharsets[0].BorderColor)},
				c64Symbol{"d021color", int(mcCharsets[0].BackgroundColor)},
				c64Symbol{"d022color", int(mcCharsets[0].D022Color)},
				c64Symbol{"d023color", int(mcCharsets[0].D023Color)},
			)
		} else {
			c.Symbols = append(c.Symbols,
				c64Symbol{"bitmap", 0x2000},
				c64Symbol{"screen", 0x2800},
				c64Symbol{"d800color", 0x2c00},
				c64Symbol{"animation", 0x3000},
				c64Symbol{"d020color", int(mcCharsets[0].BorderColor)},
				c64Symbol{"d021color", int(mcCharsets[0].BackgroundColor)},
				c64Symbol{"d022color", int(mcCharsets[0].D022Color)},
				c64Symbol{"d023color", int(mcCharsets[0].D023Color)},
			)
		}
		for i := 0; i < len(mcCharsets); i++ {
			c.Symbols = append(c.Symbols, c64Symbol{"screen" + strconv.Itoa(i), 0x4800 + i*0x400})
		}
		if _, err = c.WriteMultiColorCharsetAnimationTo(w, mcCharsets); err != nil {
			return n, fmt.Errorf("WriteMultiColorCharsetAnimationTo failed: %w", err)
		}
		return n, nil
	case len(mixCharsets) > 0:
		if c.opt.NoAnimation {
			c.Symbols = append(c.Symbols,
				c64Symbol{"d800color", 0x3c00},
				c64Symbol{"bitmap", 0x4000},
				c64Symbol{"d020color", int(mixCharsets[0].BorderColor)},
				c64Symbol{"d021color", int(mixCharsets[0].BackgroundColor)},
				c64Symbol{"d022color", int(mixCharsets[0].D022Color)},
				c64Symbol{"d023color", int(mixCharsets[0].D023Color)},
			)
			for i := 0; i < len(mixCharsets); i++ {
				c.Symbols = append(c.Symbols, c64Symbol{"screen" + strconv.Itoa(i), 0x4800 + i*0x800})
				c.Symbols = append(c.Symbols, c64Symbol{"colorram" + strconv.Itoa(i), 0x4c00 + i*0x800})
			}
		} else {
			c.Symbols = append(c.Symbols,
				c64Symbol{"bitmap", 0x2000},
				c64Symbol{"screen", 0x2800},
				c64Symbol{"d800color", 0x2c00},
				c64Symbol{"animation", 0x3000},
				c64Symbol{"d020color", int(mixCharsets[0].BorderColor)},
				c64Symbol{"d021color", int(mixCharsets[0].BackgroundColor)},
				c64Symbol{"d022color", int(mixCharsets[0].D022Color)},
				c64Symbol{"d023color", int(mixCharsets[0].D023Color)},
			)
		}
		for i := 0; i < len(mixCharsets); i++ {
			c.Symbols = append(c.Symbols, c64Symbol{"screen" + strconv.Itoa(i), 0x4800 + i*0x400})
		}
		if _, err = c.WriteMixedCharsetAnimationTo(w, mixCharsets); err != nil {
			return n, fmt.Errorf("WriteMixedCharsetAnimationTo failed: %w", err)
		}
		return n, nil
	case len(scCharsets) > 0:
		if c.opt.NoAnimation {
			c.Symbols = append(c.Symbols,
				c64Symbol{"d800color", 0x3c00},
				c64Symbol{"bitmap", 0x4000},
				c64Symbol{"d020color", int(scCharsets[0].BorderColor)},
				c64Symbol{"d021color", int(scCharsets[0].BackgroundColor)},
			)
			for i := 0; i < len(scCharsets); i++ {
				c.Symbols = append(c.Symbols, c64Symbol{"screen" + strconv.Itoa(i), 0x4800 + i*0x800})
				c.Symbols = append(c.Symbols, c64Symbol{"colorram" + strconv.Itoa(i), 0x4c00 + i*0x800})
			}
		} else {
			c.Symbols = append(c.Symbols,
				c64Symbol{"bitmap", 0x2000},
				c64Symbol{"screen", 0x2800},
				c64Symbol{"d800color", 0x2c00},
				c64Symbol{"animation", 0x3000},
				c64Symbol{"d020color", int(scCharsets[0].BorderColor)},
				c64Symbol{"d021color", int(scCharsets[0].BackgroundColor)},
			)
		}
		if _, err = c.WriteSingleColorCharsetAnimationTo(w, scCharsets); err != nil {
			return n, fmt.Errorf("WriteSingleColorCharsetAnimationTo failed: %w", err)
		}
		return n, nil
	case len(petCharsets) > 0:
		c.Symbols = append(c.Symbols,
			c64Symbol{"screen", 0x2800},
			c64Symbol{"d800color", 0x2c00},
			c64Symbol{"animation", 0x3000},
			c64Symbol{"d020color", int(petCharsets[0].BorderColor)},
			c64Symbol{"d021color", int(petCharsets[0].BackgroundColor)},
			c64Symbol{"lowercase", int(petCharsets[0].Lowercase)},
		)
		if _, err = c.WritePETSCIICharsetAnimationTo(w, petCharsets); err != nil {
			return n, fmt.Errorf("WritePETSCIICharsetAnimationTo failed: %w", err)
		}
		return n, nil
	}
	return n, fmt.Errorf("handleAnimation %q failed: no frames written", imgs[0].sourceFilename)
}

// writeAnimationDisplayerTo processes the images and writes the .prg including displayer to w.
func (c *Converter) writeAnimationDisplayerTo(w io.Writer, imgs []sourceImage, kk []Koala, hh []Hires, scSprites []SingleColorSprites, mcSprites []MultiColorSprites, mcCharsets []MultiColorCharset, scCharsets []SingleColorCharset, petCharsets []PETSCIICharset, mixCharsets []MixedCharset) (n int64, err error) {
	buf := &bytes.Buffer{}
	switch {
	case len(kk) > 0:
		// handle display koala animation
		c.Symbols = append(c.Symbols, kk[0].Symbols()...)
		c.Symbols = append(c.Symbols, c64Symbol{"animation", koalaAnimationStart})
		if c.opt.NoCrunch {
			m, err := c.WriteKoalaDisplayAnimTo(w, kk)
			n += m
			if err != nil {
				return n, fmt.Errorf("WriteKoalaDisplayAnimTo failed: %w", err)
			}
			return n, nil
		}
		if _, err = c.WriteKoalaDisplayAnimTo(buf, kk); err != nil {
			return n, fmt.Errorf("WriteKoalaDisplayAnimTo buf failed: %w", err)
		}
	case len(hh) > 0:
		// handle display hires animation
		c.Symbols = append(c.Symbols, hh[0].Symbols()...)
		c.Symbols = append(c.Symbols, c64Symbol{"animation", hiresAnimationStart})
		if c.opt.NoCrunch {
			m, err := c.WriteHiresDisplayAnimTo(w, hh)
			n += m
			if err != nil {
				return n, fmt.Errorf("WriteHiresDisplayAnimTo failed: %w", err)
			}
			return n, nil
		}
		if _, err = c.WriteHiresDisplayAnimTo(buf, hh); err != nil {
			return n, fmt.Errorf("WriteHiresDisplayAnimTo buf failed: %w", err)
		}
	case len(mcCharsets) > 0:
		if c.opt.NoAnimation {
			c.Symbols = append(c.Symbols,
				c64Symbol{"d800color", 0x3c00},
				c64Symbol{"bitmap", 0x4000},
				c64Symbol{"d020color", int(mcCharsets[0].BorderColor)},
				c64Symbol{"d021color", int(mcCharsets[0].BackgroundColor)},
				c64Symbol{"d022color", int(mcCharsets[0].D022Color)},
				c64Symbol{"d023color", int(mcCharsets[0].D023Color)},
			)
		} else {
			c.Symbols = append(c.Symbols,
				c64Symbol{"bitmap", 0x2000},
				c64Symbol{"screen", 0x2800},
				c64Symbol{"d800color", 0x2c00},
				c64Symbol{"animation", 0x3000},
				c64Symbol{"d020color", int(mcCharsets[0].BorderColor)},
				c64Symbol{"d021color", int(mcCharsets[0].BackgroundColor)},
				c64Symbol{"d022color", int(mcCharsets[0].D022Color)},
				c64Symbol{"d023color", int(mcCharsets[0].D023Color)},
			)
		}
		for i := 0; i < len(mcCharsets); i++ {
			c.Symbols = append(c.Symbols, c64Symbol{"screen" + strconv.Itoa(i), 0x4800 + i*0x400})
		}
		if c.opt.NoCrunch {
			m, err := c.WriteMultiColorCharsetAnimationTo(w, mcCharsets)
			n += m
			if err != nil {
				return n, fmt.Errorf("WriteMultiColorCharsetAnimationTo failed: %w", err)
			}
			return n, nil
		}
		if _, err = c.WriteMultiColorCharsetAnimationTo(buf, mcCharsets); err != nil {
			return n, fmt.Errorf("WriteMultiColorCharsetAnimationTo buf failed: %w", err)
		}
	case len(mixCharsets) > 0:
		if c.opt.NoAnimation {
			c.Symbols = append(c.Symbols,
				c64Symbol{"d800color", 0x3c00},
				c64Symbol{"bitmap", 0x4000},
				c64Symbol{"d020color", int(mixCharsets[0].BorderColor)},
				c64Symbol{"d021color", int(mixCharsets[0].BackgroundColor)},
				c64Symbol{"d022color", int(mixCharsets[0].D022Color)},
				c64Symbol{"d023color", int(mixCharsets[0].D023Color)},
			)
		} else {
			c.Symbols = append(c.Symbols,
				c64Symbol{"bitmap", 0x2000},
				c64Symbol{"screen", 0x2800},
				c64Symbol{"d800color", 0x2c00},
				c64Symbol{"animation", 0x3000},
				c64Symbol{"d020color", int(mixCharsets[0].BorderColor)},
				c64Symbol{"d021color", int(mixCharsets[0].BackgroundColor)},
				c64Symbol{"d022color", int(mixCharsets[0].D022Color)},
				c64Symbol{"d023color", int(mixCharsets[0].D023Color)},
			)
		}
		for i := 0; i < len(mixCharsets); i++ {
			c.Symbols = append(c.Symbols, c64Symbol{"screen" + strconv.Itoa(i), 0x4800 + i*0x400})
		}
		if c.opt.NoCrunch {
			m, err := c.WriteMixedCharsetAnimationTo(w, mixCharsets)
			n += m
			if err != nil {
				return n, fmt.Errorf("WriteMixedCharsetAnimationTo failed: %w", err)
			}
			return n, nil
		}
		if _, err = c.WriteMixedCharsetAnimationTo(buf, mixCharsets); err != nil {
			return n, fmt.Errorf("WriteMixedCharsetAnimationTo buf failed: %w", err)
		}
	case len(scCharsets) > 0:
		if c.opt.NoAnimation {
			c.Symbols = append(c.Symbols,
				c64Symbol{"bitmap", 0x4000},
				c64Symbol{"d800color", 0x3c00},
				c64Symbol{"d020color", int(scCharsets[0].BorderColor)},
				c64Symbol{"d021color", int(scCharsets[0].BackgroundColor)},
			)
			for i := 0; i < len(scCharsets); i++ {
				c.Symbols = append(c.Symbols, c64Symbol{"screen" + strconv.Itoa(i), 0x4800 + i*0x800})
				c.Symbols = append(c.Symbols, c64Symbol{"colorram" + strconv.Itoa(i), 0x4c00 + i*0x800})
			}
		} else {
			c.Symbols = append(c.Symbols,
				c64Symbol{"bitmap", 0x2000},
				c64Symbol{"screen", 0x2800},
				c64Symbol{"d800color", 0x2c00},
				c64Symbol{"animation", 0x3000},
				c64Symbol{"d020color", int(scCharsets[0].BorderColor)},
				c64Symbol{"d021color", int(scCharsets[0].BackgroundColor)},
			)
		}
		if c.opt.NoCrunch {
			m, err := c.WriteSingleColorCharsetAnimationTo(w, scCharsets)
			n += m
			if err != nil {
				return n, fmt.Errorf("WriteSingleColorCharsetAnimationTo failed: %w", err)
			}
			return n, nil
		}
		if _, err = c.WriteSingleColorCharsetAnimationTo(buf, scCharsets); err != nil {
			return n, fmt.Errorf("WriteSingleColorCharsetAnimationTo buf failed: %w", err)
		}
	case len(petCharsets) > 0:
		c.Symbols = append(c.Symbols,
			c64Symbol{"screen", 0x2800},
			c64Symbol{"d800color", 0x2c00},
			c64Symbol{"animation", 0x3000},
			c64Symbol{"d020color", int(petCharsets[0].BorderColor)},
			c64Symbol{"d021color", int(petCharsets[0].BackgroundColor)},
		)
		if c.opt.NoCrunch {
			m, err := c.WritePETSCIICharsetAnimationTo(w, petCharsets)
			n += m
			if err != nil {
				return n, fmt.Errorf("WritePETSCIICharsetAnimationTo failed: %w", err)
			}
			return n, nil
		}
		if _, err = c.WritePETSCIICharsetAnimationTo(buf, petCharsets); err != nil {
			return n, fmt.Errorf("WritePETSCIICharsetAnimationTo buf failed: %w", err)
		}
	default:
		return n, fmt.Errorf("animation displayers do not support %q", imgs[0].graphicsType)
	}

	tsc, err := TSCrunch.New(TSCOptions, buf)
	if err != nil {
		return n, fmt.Errorf("tscrunch.New failed: %w", err)
	}
	t1 := time.Now()
	m, err := tsc.WriteTo(w)
	n += m
	if err != nil {
		return n, fmt.Errorf("tsc.WriteTo failed: %w", err)
	}
	if !c.opt.Quiet {
		fmt.Printf("TSCrunched in %s\n", time.Since(t1))
	}
	return n, nil
}

type Char interface {
	Index() int
	Bytes() []byte
}

type Charer interface {
	Char(charIndex int) Char
}

func (c MultiColorChar) Index() int {
	return c.CharIndex
}

func (c MultiColorChar) Bytes() (buf []byte) {
	buf = append(buf, c.Bitmap[:]...)
	return append(buf, c.ScreenColor, c.D800Color)
}

func (c SingleColorChar) Index() int {
	return c.CharIndex
}

func (c SingleColorChar) Bytes() (buf []byte) {
	buf = append(buf, c.Bitmap[:]...)
	return append(buf, c.ScreenColor)
}

// processFramesOfChars creates a slice of byteslices, where each byteslice contains a frame of chunks in animation format.
// See readme for details on format.
func processFramesOfChars(opt Options, frames [][]Char) ([][]byte, error) {
	if len(frames) < 2 {
		return nil, fmt.Errorf("insufficient number of images %d < 2", len(frames))
	}
	prgs := make([][]byte, 0)
	for i, frame := range frames {
		if opt.Verbose {
			log.Printf("frame %d length in changed chars: %d", i, len(frame))
		}
		curChar := -10
		curChunk := chunk{}
		prg := []byte{}
		for _, char := range frame {
			switch {
			case curChar == char.Index()-1:
				// next char of current chunk
				curChunk.append(char)
			default:
				// new chunk
				if curChunk.charCount > 0 {
					if opt.Verbose {
						log.Println(curChunk.String())
					}
					prg = append(prg, curChunk.export()...)
				}
				curChunk = newChunk(char.Index())
				curChunk.append(char)
			}
			curChar = char.Index()
		}
		// add last chunk
		if curChunk.charCount > 0 {
			if opt.Verbose {
				log.Printf("last chunk: %s", curChunk.String())
			}
			prg = append(prg, curChunk.export()...)
		}
		// end of chunk marker
		prg = append(prg, 0x00)
		prgs = append(prgs, prg)
	}
	return prgs, nil
}

// processAnimation extracts the differences between the various imgs per char (single or multicolor).
// returns the converted animation in slices of byteslices, where each byteslice contains a frame of chunks in animation format.
// See readme for details on format.
func processAnimation(opt Options, imgs []Charer) ([][]byte, error) {
	if len(imgs) < 2 {
		return nil, fmt.Errorf("insufficient number of frames %d < 2", len(imgs))
	}
	if opt.Verbose {
		log.Printf("total number of frames: %d", len(imgs))
	}

	charFrames := make([][]Char, len(imgs))
	for i := 0; i < len(imgs)-1; i++ {
		charFrames[i] = []Char{}
	}

	for i := 0; i < 1000; i++ {
		for j := 0; j < len(imgs); j++ {
			// use last frame as previous frame for first frame
			// ensures clean loop
			k := len(imgs) - 1
			if j > 0 {
				k = j - 1
			}
			prevChar := imgs[k].Char(i)
			frameChar := imgs[j].Char(i)
			if prevChar != frameChar {
				charFrames[j] = append(charFrames[j], frameChar)
			}
		}
	}
	return processFramesOfChars(opt, charFrames)
}

func (c *Converter) WriteFrameDelayByte(w io.Writer, frameIndex, totalFrames int) (n int64, err error) {
	frameDelay := c.opt.FrameDelay
	if len(c.AnimItems) > 0 {
		if len(c.AnimItems) != totalFrames {
			return n, fmt.Errorf("anim-file has %d lines, but there are %d frames, this should be equal?", len(c.AnimItems), totalFrames)
		}
		frameDelay = c.AnimItems[frameIndex].FrameDelay
	}
	if _, err = w.Write([]byte{frameDelay}); err != nil {
		return n, fmt.Errorf("w.Write error: %w", err)
	}
	if c.opt.Verbose {
		fmt.Printf("frame %d will be shown for %d frames\n", frameIndex, frameDelay)
	}
	return n, nil
}

// WriteKoalaDisplayAnimTo processes kk and writes the converted animation and displayer to w.
// Optionally uses c.AnimItems for timing.
func (c *Converter) WriteKoalaDisplayAnimTo(w io.Writer, kk []Koala) (n int64, err error) {
	bgBorder := kk[0].BackgroundColor | kk[0].BorderColor<<4
	opt := kk[0].opt

	frames := makeCharer(kk)
	framePrgs, err := processAnimation(opt, frames)
	if err != nil {
		return n, err
	}

	displayer := koalaDisplayAnim
	if opt.AlternativeFade {
		displayer = koalaDisplayAnimAlternative
	}
	link := NewLinker(0, opt.VeryVerbose)
	if _, err = link.WritePrg(displayer); err != nil {
		return n, err
	}
	link.SetByte(DisplayerSettingsStart+7, byte(opt.FrameDelay), byte(opt.WaitSeconds), opt.NoFadeByte())
	if !opt.NoFade {
		link.Block(koalaFadePassStart, 0xd000)
	}
	if !opt.Quiet {
		fmt.Printf("memory usage for displayer code: %s - %s\n", link.StartAddress(), link.EndAddress())
	}
	if _, err = link.WriteMap(LinkMap{
		BitmapAddress:                kk[0].Bitmap[:],
		BitmapScreenRAMAddress:       kk[0].ScreenColor[:],
		BitmapColorRAMAddress:        kk[0].D800Color[:],
		BitmapColorRAMAddress + 1000: {bgBorder},
	}); err != nil {
		return n, fmt.Errorf("link.WriteMap error: %w", err)
	}
	if !opt.Quiet {
		fmt.Printf("memory usage for picture: 0x%04x - %s\n", BitmapAddress, link.EndAddress())
	}

	link.SetCursor(koalaAnimationStart)
	for i, bin := range framePrgs {
		if _, err = link.Write(bin); err != nil {
			return n, fmt.Errorf("link.Write error: %w", err)
		}
		if _, err = c.WriteFrameDelayByte(link, i, len(framePrgs)); err != nil {
			return n, fmt.Errorf("WriteFrameDelayByte failed: %w", err)
		}
	}
	if _, err = link.Write([]byte{0xff}); err != nil {
		return n, fmt.Errorf("link.Write error: %w", err)
	}
	if len(c.AnimItems) > 0 {
		link.SetByte(DisplayerSettingsStart+7, c.AnimItems[0].FrameDelay)
	}

	if !opt.Quiet {
		fmt.Printf("memory usage for animations: %s - %s\n", Word(koalaAnimationStart), link.EndAddress())
		fmt.Printf("memory usage for generated fadecode: %s - %s\n", Word(koalaFadePassStart), Word(0xcfff))
	}

	if err = injectSID(link, opt.IncludeSID, opt.Quiet); err != nil {
		return n, fmt.Errorf("injectSID failed: %w", err)
	}
	m, err := link.WriteTo(w)
	n += int64(m)
	return n, err
}

// exportAnims format:
//      number of chars			1 byte
//		bitmap_lo + bitmap_hi	2 byte
//		char_lo, char_hi 		2 byte
//			pixels					8 byte
//	    	screencol				1 byte
//			d800col					1 byte
// total bytes: 5 + 10 * charcount

// WriteHiresDisplayAnimTo processes hh and writes the converted animation and displayer to w.
// Optionally uses c.AnimItems for timing.
func (c *Converter) WriteHiresDisplayAnimTo(w io.Writer, hh []Hires) (n int64, err error) {
	opt := hh[0].opt
	frames := makeCharer(hh)
	framePrgs, err := processAnimation(opt, frames)
	if err != nil {
		return n, fmt.Errorf("processAnimation error: %w", err)
	}

	link := NewLinker(0, opt.VeryVerbose)
	if _, err = link.WritePrg(hiresDisplayAnim); err != nil {
		return n, fmt.Errorf("link.WritePrg error: %w", err)
	}
	if !opt.NoFade {
		link.Block(hiresFadePassStart, 0xd000)
	}
	link.SetByte(DisplayerSettingsStart+7, byte(opt.FrameDelay), byte(opt.WaitSeconds), opt.NoFadeByte())
	if !opt.Quiet {
		fmt.Printf("memory usage for displayer code: %s - %s\n", link.StartAddress(), link.EndAddress())
	}

	link.SetCursor(BitmapAddress)
	h := hh[0]
	for _, b := range [][]byte{h.Bitmap[:], h.ScreenColor[:], {h.BorderColor}} {
		if _, err = link.Write(b); err != nil {
			return n, fmt.Errorf("link.Write error: %w", err)
		}
	}
	if !opt.Quiet {
		fmt.Printf("memory usage for picture: %#04x - %s\n", BitmapAddress, link.EndAddress())
	}

	link.SetCursor(hiresAnimationStart)
	for i, bin := range framePrgs {
		if _, err = link.Write(bin); err != nil {
			return n, fmt.Errorf("link.Write error: %w", err)
		}
		if _, err = c.WriteFrameDelayByte(link, i, len(framePrgs)); err != nil {
			return n, fmt.Errorf("WriteFrameDelayByte failed: %w", err)
		}
	}
	if _, err = link.Write([]byte{0xff}); err != nil {
		return n, fmt.Errorf("link.Write error: %w", err)
	}
	if len(c.AnimItems) > 0 {
		link.SetByte(DisplayerSettingsStart+7, c.AnimItems[0].FrameDelay)
	}
	if !opt.Quiet {
		fmt.Printf("memory usage for animations: %#04x - %s\n", hiresAnimationStart, link.EndAddress())
		fmt.Printf("memory usage for generated fadecode: %#04x - %#04x\n", hiresFadePassStart, 0xcfff)
	}

	if err = injectSID(link, opt.IncludeSID, opt.Quiet); err != nil {
		return n, fmt.Errorf("injectSID failed: %w", err)
	}

	m, err := link.WriteTo(w)
	n += int64(m)
	return n, err
}

type chunk struct {
	charIndex int
	charCount byte
	bitmapLo  byte
	bitmapHi  byte
	charLo    byte
	charHi    byte
	charBytes []byte
}

// newChunk returns a new empty chunk starting at charIndex.
func newChunk(charIndex int) chunk {
	return chunk{
		charIndex: charIndex,
		bitmapLo:  byte((charIndex * 8) & 0xff),
		bitmapHi:  byte((charIndex * 8) >> 8),
		charLo:    byte(charIndex & 0xff),
		charHi:    byte((charIndex - charIndex&0xff) >> 8),
		charBytes: []byte{},
	}
}

func (c *chunk) append(char Char) {
	c.charBytes = append(c.charBytes, char.Bytes()...)
	c.charCount++
}

func (c *chunk) export() []byte {
	return append([]byte{c.charCount, c.bitmapLo, c.bitmapHi, c.charLo, c.charHi}, c.charBytes...)
}

func (c *chunk) String() string {
	return fmt.Sprintf("chunk charindex: %d charcount %d bitmap: $%x char: $%x", c.charIndex, c.charCount, int(c.bitmapHi)*256+int(c.bitmapLo), int(c.charHi)*256+int(c.charLo))
}

// Char returns the Char at index charIndex.
func (k Koala) Char(charIndex int) Char {
	c := MultiColorChar{
		CharIndex:       charIndex,
		BackgroundColor: k.BackgroundColor,
		ScreenColor:     k.ScreenColor[charIndex],
		D800Color:       k.D800Color[charIndex],
	}
	for i := range c.Bitmap {
		c.Bitmap[i] = k.Bitmap[charIndex*8+i]
	}
	return c
}

// Char returns the Char at index charIndex.
func (h Hires) Char(charIndex int) Char {
	c := SingleColorChar{
		CharIndex:   charIndex,
		ScreenColor: h.ScreenColor[charIndex],
	}
	for i := range c.Bitmap {
		c.Bitmap[i] = h.Bitmap[charIndex*8+i]
	}
	return c
}

// WriteMultiColorCharsetAnimationTo writes the MultiColorCharsets to w, optionally with displayer code.
func (c *Converter) WriteMultiColorCharsetAnimationTo(w io.Writer, cc []MultiColorCharset) (n int64, err error) {
	if len(cc) < 2 {
		return n, fmt.Errorf("not enough images %d < 2", len(cc))
	}
	opt := cc[0].opt
	var link *Linker
	displayer := mcCharsetDisplayMulti
	if opt.NoAnimation {
		link = NewLinker(0x3c00, opt.VeryVerbose)
		_, err = link.WriteMap(LinkMap{
			0x3c00: cc[0].D800Color[:],
			0x3fe8: []byte{cc[0].BorderColor, cc[0].BackgroundColor, cc[0].D022Color, cc[0].D023Color, byte(len(cc)) & 0xff},
			0x4000: cc[len(cc)-1].Bitmap[:],
		})
		if err != nil {
			return n, fmt.Errorf("link.WriteMap failed: %w", err)
		}
		for i := 0; i < len(cc); i++ {
			_, err = link.WriteMap(LinkMap{0x4800 + Word(i)*0x400: cc[i].Screen[:]})
			if err != nil {
				return n, fmt.Errorf("link.WriteMap failed: %w", err)
			}
		}
	} else {
		displayer = mcCharsetDisplayAnim
		link = NewLinker(0x2000, opt.VeryVerbose)
		_, err = link.WriteMap(LinkMap{
			0x2000: cc[len(cc)-1].Bitmap[:],
			0x2800: cc[0].Screen[:],
			0x2c00: cc[0].D800Color[:],
			0x2fe8: []byte{cc[0].BorderColor, cc[0].BackgroundColor, cc[0].D022Color, cc[0].D023Color},
		})
		if err != nil {
			return n, fmt.Errorf("link.WriteMap failed: %w", err)
		}
		buf := &bytes.Buffer{}
		curChunk := charChunk{charIndex: -10}
		flushedtotal := 0
		flushedchartotal := 0
		flushChunk := func() {
			if curChunk.charCount > 0 {
				if opt.VeryVerbose {
					log.Printf("got chunk: %v", curChunk)
				}
				b := []byte{}
				b = append(b, curChunk.charCount, curChunk.ScreenLow(), curChunk.ScreenHigh())
				b = append(b, curChunk.bytes...)
				if _, err = buf.Write(b); err != nil {
					log.Printf("buf.Write failed: %v", err)
					return
				}
				flushedchartotal += int(curChunk.charCount)
				flushedtotal++
				curChunk = charChunk{charIndex: -10}
			}
		}
		for i := 0; i < len(cc); i++ {
			// for clean loop
			prv := len(cc) - 1
			if i > 0 {
				prv = i - 1
			}
			for char := 0; char < FullScreenChars; char++ {
				if cc[i].Screen[char] != cc[prv].Screen[char] || cc[i].D800Color[char] != cc[prv].D800Color[char] {
					if opt.VeryVerbose {
						log.Printf("%d %d: cc.Screen[char] = %d | prevscreen[char] = %d", i, char, cc[i].Screen[char], cc[prv].Screen[char])
						log.Printf("%d %d: cc.D800Color[char] = %d | prevcolram[char] = %d", i, char, cc[i].D800Color[char], cc[prv].D800Color[char])
					}
					if curChunk.charCount == 0 {
						curChunk = charChunk{
							charIndex: char,
							charCount: 1,
							bytes:     []byte{cc[i].Screen[char], cc[i].D800Color[char]},
						}
					} else {
						curChunk.bytes = append(curChunk.bytes, cc[i].Screen[char], cc[i].D800Color[char])
						curChunk.charCount++
						if curChunk.charCount > 254 {
							log.Printf("large chunck detected (%d chars), flushing...", curChunk.charCount)
							flushChunk()
						}
					}
					continue
				}
				flushChunk()
			}
			flushChunk()
			if _, err = buf.Write([]byte{0x00}); err != nil { // end of frame
				return n, fmt.Errorf("buf.Write failed: %w", err)
			}
			if _, err = c.WriteFrameDelayByte(buf, i, len(cc)); err != nil {
				return n, fmt.Errorf("WriteFrameDelayByte failed: %w", err)
			}
		}
		if _, err = buf.Write([]byte{0xff}); err != nil { // end of frames
			return n, fmt.Errorf("buf.Write failed: %w", err)
		}
		_, err = link.WriteMap(LinkMap{0x3000: buf.Bytes()})
		if err != nil {
			return n, fmt.Errorf("link.WriteMap failed: %w", err)
		}
		if opt.Verbose {
			log.Printf("flushed %d chunks, %d chars", flushedtotal, flushedchartotal)
		}
	}

	if opt.Display {
		if _, err = link.WritePrg(displayer); err != nil {
			return n, fmt.Errorf("link.WritePrg failed: %w", err)
		}
		link.SetByte(DisplayerSettingsStart+7, byte(cc[0].opt.FrameDelay), byte(cc[0].opt.WaitSeconds), cc[0].opt.NoFadeByte())
		if err = injectSID(link, opt.IncludeSID, opt.Quiet); err != nil {
			return n, fmt.Errorf("injectSID failed: %w", err)
		}
	}
	return link.WriteTo(w)
}

type charChunk struct {
	charIndex int
	charCount byte
	bytes     []byte
}

func (cc charChunk) ScreenLow() byte {
	return Word(cc.charIndex).Low()
}

func (cc charChunk) ScreenHigh() byte {
	return Word(cc.charIndex).High()
}

// WriteSingleColorCharsetAnimationTo writes the SingleColorCharset to w, optionally with displayer code.
func (c *Converter) WriteSingleColorCharsetAnimationTo(w io.Writer, cc []SingleColorCharset) (n int64, err error) {
	if len(cc) < 2 {
		return n, fmt.Errorf("not enough images %d < 2", len(cc))
	}
	opt := cc[0].opt
	var link *Linker
	displayer := scCharsetDisplayMulti
	if opt.NoAnimation {
		link = NewLinker(0x3fe8, opt.VeryVerbose)
		_, err = link.WriteMap(LinkMap{
			0x3fe8: []byte{cc[0].BorderColor, cc[0].BackgroundColor, byte(len(cc)) & 0xff},
			0x4000: cc[len(cc)-1].Bitmap[:],
		})
		if err != nil {
			return n, fmt.Errorf("link.WriteMap failed: %w", err)
		}
		for i := 0; i < len(cc); i++ {
			_, err = link.WriteMap(LinkMap{
				0x47e8 + Word(i)*0x800: []byte{cc[i].BackgroundColor | cc[i].BorderColor<<4},
				0x4800 + Word(i)*0x800: cc[i].Screen[:],
				0x4c00 + Word(i)*0x800: cc[i].D800Color[:],
			})
			if err != nil {
				return n, fmt.Errorf("link.WriteMap failed: %w", err)
			}
		}
	} else {
		displayer = scCharsetDisplayAnim
		link = NewLinker(0x2000, cc[0].opt.VeryVerbose)
		_, err = link.WriteMap(LinkMap{
			0x2000: cc[len(cc)-1].Bitmap[:],
			0x2800: cc[0].Screen[:],
			0x2c00: cc[0].D800Color[:],
			0x2fe8: []byte{cc[0].BorderColor, cc[0].BackgroundColor},
		})
		if err != nil {
			return n, fmt.Errorf("link.WriteMap failed: %w", err)
		}

		buf := &bytes.Buffer{}
		curChunk := charChunk{charIndex: -10}
		flushedtotal := 0
		flushedchartotal := 0
		flushChunk := func() {
			if curChunk.charCount > 0 {
				if cc[0].opt.VeryVerbose {
					log.Printf("got chunk: %v", curChunk)
				}
				b := append([]byte{curChunk.charCount, curChunk.ScreenLow(), curChunk.ScreenHigh()}, curChunk.bytes...)
				if _, err = buf.Write(b); err != nil {
					log.Printf("buf.Write failed: %v", err)
					return
				}
				flushedchartotal += int(curChunk.charCount)
				flushedtotal++
				curChunk = charChunk{charIndex: -10}
			}
		}
		for i := 0; i < len(cc); i++ {
			if _, err = buf.Write([]byte{cc[i].BackgroundColor | cc[i].BorderColor<<4}); err != nil {
				log.Printf("buf.Write failed: %v", err)
				return
			}
			// for clean loop
			prv := len(cc) - 1
			if i > 0 {
				prv = i - 1
			}
			for char := 0; char < FullScreenChars; char++ {
				if cc[i].Screen[char] != cc[prv].Screen[char] || cc[i].D800Color[char] != cc[prv].D800Color[char] {
					if cc[0].opt.VeryVerbose {
						log.Printf("%d %d: cc.Screen[char] = %d | prevscreen[char] = %d", i, char, cc[i].Screen[char], cc[prv].Screen[char])
						log.Printf("%d %d: cc.D800Color[char] = %d | prevcolram[char] = %d", i, char, cc[i].D800Color[char], cc[prv].D800Color[char])
					}
					if curChunk.charCount == 0 {
						curChunk = charChunk{
							charIndex: char,
							charCount: 1,
							bytes:     []byte{cc[i].Screen[char], cc[i].D800Color[char]},
						}
					} else {
						curChunk.bytes = append(curChunk.bytes, cc[i].Screen[char], cc[i].D800Color[char])
						curChunk.charCount++
						if curChunk.charCount > 254 {
							log.Printf("large chunck detected (%d chars), flushing...", curChunk.charCount)
							flushChunk()
						}
					}
					continue
				}
				flushChunk()
			}
			flushChunk()
			if _, err = buf.Write([]byte{0x00}); err != nil { // end of frame
				return n, fmt.Errorf("buf.Write failed: %w", err)
			}
			if _, err = c.WriteFrameDelayByte(buf, i, len(cc)); err != nil {
				return n, fmt.Errorf("Write failed: %w", err)
			}
		}
		if _, err = buf.Write([]byte{0xff}); err != nil { // end of frames
			return n, fmt.Errorf("buf.Write failed: %w", err)
		}
		_, err = link.WriteMap(LinkMap{0x3000: buf.Bytes()})
		if err != nil {
			return n, fmt.Errorf("link.WriteMap failed: %w", err)
		}
		if cc[0].opt.Verbose {
			log.Printf("flushed %d chunks, %d chars", flushedtotal, flushedchartotal)
		}
	}

	if cc[0].opt.Display {
		if _, err = link.WritePrg(displayer); err != nil {
			return n, fmt.Errorf("link.WritePrg failed: %w", err)
		}
		link.SetByte(DisplayerSettingsStart+7, byte(cc[0].opt.FrameDelay), byte(cc[0].opt.WaitSeconds), byte(cc[0].opt.NoFadeByte()))
		if !opt.NoFade {
			link.Block(hiresFadePassStart, 0xcfff)
		}
		if err = injectSID(link, cc[0].opt.IncludeSID, cc[0].opt.Quiet); err != nil {
			return n, fmt.Errorf("injectSID failed: %w", err)
		}
	}
	return link.WriteTo(w)
}

// WritePETSCIICharsetAnimationTo writes the PETSCIICharset to w, optionally with displayer code.
func (c *Converter) WritePETSCIICharsetAnimationTo(w io.Writer, cc []PETSCIICharset) (n int64, err error) {
	if len(cc) < 2 {
		return n, fmt.Errorf("not enough images %d < 2", len(cc))
	}
	link := NewLinker(0x2000, cc[0].opt.VeryVerbose)
	_, err = link.WriteMap(LinkMap{
		0x2800: cc[0].Screen[:],
		0x2c00: cc[0].D800Color[:],
		0x2fe8: []byte{cc[0].BorderColor, cc[0].BackgroundColor},
	})
	if err != nil {
		return n, fmt.Errorf("link.WriteMap failed: %w", err)
	}

	buf := &bytes.Buffer{}
	curChunk := charChunk{charIndex: -10}
	flushedtotal := 0
	flushedchartotal := 0
	flushChunk := func() {
		if curChunk.charCount > 0 {
			if cc[0].opt.VeryVerbose {
				log.Printf("got chunk: %v", curChunk)
			}
			b := append([]byte{curChunk.charCount, curChunk.ScreenLow(), curChunk.ScreenHigh()}, curChunk.bytes...)
			if _, err = buf.Write(b); err != nil {
				log.Printf("buf.Write failed: %v", err)
				return
			}
			flushedchartotal += int(curChunk.charCount)
			flushedtotal++
			curChunk = charChunk{charIndex: -10}
		}
	}

	for i := 0; i < len(cc); i++ {
		if _, err = buf.Write([]byte{cc[i].BackgroundColor | cc[i].BorderColor<<4}); err != nil {
			return n, fmt.Errorf("buf.Write failed: %w", err)
		}

		// for clean loop
		prv := len(cc) - 1
		if i > 0 {
			prv = i - 1
		}
		for char := 0; char < FullScreenChars; char++ {
			if cc[i].Screen[char] != cc[prv].Screen[char] || cc[i].D800Color[char] != cc[prv].D800Color[char] {
				if cc[0].opt.VeryVerbose {
					log.Printf("%d %d: cc.Screen[char] = %d | prevscreen[char] = %d", i, char, cc[i].Screen[char], cc[prv].Screen[char])
					log.Printf("%d %d: cc.D800Color[char] = %d | prevcolram[char] = %d", i, char, cc[i].D800Color[char], cc[prv].D800Color[char])
				}
				if curChunk.charCount == 0 {
					curChunk = charChunk{
						charIndex: char,
						charCount: 1,
						bytes:     []byte{cc[i].Screen[char], cc[i].D800Color[char]},
					}
				} else {
					curChunk.bytes = append(curChunk.bytes, cc[i].Screen[char], cc[i].D800Color[char])
					curChunk.charCount++
					if curChunk.charCount > 254 {
						log.Printf("large chunck detected (%d chars), flushing...", curChunk.charCount)
						flushChunk()
					}
				}
				continue
			}
			flushChunk()
		}
		flushChunk()
		if _, err = buf.Write([]byte{0x00}); err != nil { // end of chunks and frame
			return n, fmt.Errorf("buf.Write failed: %w", err)
		}
		if _, err = c.WriteFrameDelayByte(buf, i, len(cc)); err != nil {
			return n, fmt.Errorf("WriteFrameDelayByte failed: %w", err)
		}
	}
	if _, err = buf.Write([]byte{0xff}); err != nil { // end of frames
		return n, fmt.Errorf("buf.Write failed: %w", err)
	}
	if _, err = link.WriteMap(LinkMap{0x3000: buf.Bytes()}); err != nil {
		return n, fmt.Errorf("link.WriteMap failed: %w", err)
	}
	if cc[0].opt.Verbose {
		log.Printf("flushed %d chunks, %d chars", flushedtotal, flushedchartotal)
	}

	if cc[0].opt.Display {
		if _, err = link.WritePrg(petsciiCharsetDisplayAnim); err != nil {
			return n, fmt.Errorf("link.WritePrg failed: %w", err)
		}
		link.SetByte(DisplayerSettingsStart+7, byte(cc[0].Lowercase), byte(cc[0].opt.FrameDelay), byte(cc[0].opt.WaitSeconds), cc[0].opt.NoFadeByte())
		if !cc[0].opt.NoFade {
			link.Block(hiresFadePassStart, 0xcfff)
		}
		if err = injectSID(link, cc[0].opt.IncludeSID, cc[0].opt.Quiet); err != nil {
			return n, fmt.Errorf("injectSID failed: %w", err)
		}
		if len(c.AnimItems) > 0 {
			link.SetByte(DisplayerSettingsStart+8, c.AnimItems[0].FrameDelay)
		}
	}
	return link.WriteTo(w)
}

// WriteMixedCharsetAnimationTo writes the MixedCharset to w, optionally with displayer code.
func (c *Converter) WriteMixedCharsetAnimationTo(w io.Writer, cc []MixedCharset) (n int64, err error) {
	if len(cc) < 2 {
		return n, fmt.Errorf("not enough images %d < 2", len(cc))
	}
	opt := cc[0].opt
	var link *Linker
	displayer := mcCharsetDisplayMulti
	if opt.NoAnimation {
		link = NewLinker(0x3fe8, opt.VeryVerbose)
		_, err = link.WriteMap(LinkMap{
			0x3fe8: []byte{cc[0].BorderColor, cc[0].BackgroundColor, byte(len(cc)) & 0xff},
			0x4000: cc[len(cc)-1].Bitmap[:],
		})
		if err != nil {
			return n, fmt.Errorf("link.WriteMap failed: %w", err)
		}
		for i := 0; i < len(cc); i++ {
			_, err = link.WriteMap(LinkMap{
				0x4800 + Word(i)*0x800: cc[i].Screen[:],
				0x4c00 + Word(i)*0x800: cc[i].D800Color[:],
			})
			if err != nil {
				return n, fmt.Errorf("link.WriteMap failed: %w", err)
			}
		}
	} else {
		displayer = mcCharsetDisplayAnim
		link = NewLinker(0x2000, cc[0].opt.VeryVerbose)
		_, err = link.WriteMap(LinkMap{
			0x2000: cc[len(cc)-1].Bitmap[:],
			0x2800: cc[0].Screen[:],
			0x2c00: cc[0].D800Color[:],
			0x2fe8: []byte{cc[0].BorderColor, cc[0].BackgroundColor},
		})
		if err != nil {
			return n, fmt.Errorf("link.WriteMap failed: %w", err)
		}

		buf := &bytes.Buffer{}
		curChunk := charChunk{charIndex: -10}
		flushedtotal := 0
		flushedchartotal := 0
		flushChunk := func() {
			if curChunk.charCount > 0 {
				if cc[0].opt.VeryVerbose {
					log.Printf("got chunk: %v", curChunk)
				}
				b := []byte{}
				b = append(b, curChunk.charCount, curChunk.ScreenLow(), curChunk.ScreenHigh())
				b = append(b, curChunk.bytes...)
				if _, err = buf.Write(b); err != nil {
					log.Printf("buf.Write failed: %v", err)
					return
				}
				flushedchartotal += int(curChunk.charCount)
				flushedtotal++
				curChunk = charChunk{charIndex: -10}
			}
		}
		for i := 0; i < len(cc); i++ {
			// for clean loop
			prv := len(cc) - 1
			if i > 0 {
				prv = i - 1
			}
			for char := 0; char < FullScreenChars; char++ {
				if cc[i].Screen[char] != cc[prv].Screen[char] || cc[i].D800Color[char] != cc[prv].D800Color[char] {
					if cc[0].opt.VeryVerbose {
						log.Printf("%d %d: cc.Screen[char] = %d | prevscreen[char] = %d", i, char, cc[i].Screen[char], cc[prv].Screen[char])
						log.Printf("%d %d: cc.D800Color[char] = %d | prevcolram[char] = %d", i, char, cc[i].D800Color[char], cc[prv].D800Color[char])
					}
					if curChunk.charCount == 0 {
						curChunk = charChunk{
							charIndex: char,
							charCount: 1,
							bytes:     []byte{cc[i].Screen[char], cc[i].D800Color[char]},
						}
					} else {
						curChunk.bytes = append(curChunk.bytes, cc[i].Screen[char], cc[i].D800Color[char])
						curChunk.charCount++
						if curChunk.charCount > 254 {
							log.Printf("large chunck detected (%d chars), flushing...", curChunk.charCount)
							flushChunk()
						}
					}
					continue
				}
				flushChunk()
			}
			flushChunk()
			if _, err = buf.Write([]byte{0x00}); err != nil { // end of chunks and frame
				log.Printf("buf.Write failed: %v", err)
				return
			}
			if _, err = c.WriteFrameDelayByte(buf, i, len(cc)); err != nil {
				return n, fmt.Errorf("WriteFrameDelayByte failed: %w", err)
			}
		}
		if _, err = buf.Write([]byte{0xff}); err != nil { // end of frames
			log.Printf("buf.Write failed: %v", err)
			return
		}
		_, err = link.WriteMap(LinkMap{0x3000: buf.Bytes()})
		if err != nil {
			return n, fmt.Errorf("link.WriteMap failed: %w", err)
		}
		if cc[0].opt.Verbose {
			log.Printf("flushed %d chunks, %d chars", flushedtotal, flushedchartotal)
		}
	}

	if cc[0].opt.Display {
		if _, err = link.WritePrg(displayer); err != nil {
			return n, fmt.Errorf("link.WritePrg failed: %w", err)
		}
		link.SetByte(DisplayerSettingsStart+7, byte(cc[0].opt.FrameDelay), byte(cc[0].opt.WaitSeconds))
		link.Block(hiresFadePassStart, 0xcfff)
		if err = injectSID(link, cc[0].opt.IncludeSID, cc[0].opt.Quiet); err != nil {
			return n, fmt.Errorf("injectSID failed: %w", err)
		}
	}
	return link.WriteTo(w)
}

func makeCharer[S []E, E Koala | Hires](s S) []Charer {
	frames := make([]Charer, len(s))
	for i := range s {
		frames[i] = Charer(s[i])
	}
	return frames
}

type AnimItem struct {
	FrameDelay byte
	Filename   string
}

func ExtractAnimationFile(filename string) (result []AnimItem, err error) {
	f, err := os.Open(filename)
	if err != nil {
		return result, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return result, err
		}
		if len(record) < 2 {
			return result, fmt.Errorf("csv row does not contain >= 2  columns, but %d", len(record))
		}
		c, err := strconv.Atoi(record[0])
		if err != nil {
			return result, err
		}
		if c < 1 || c > 255 {
			return result, fmt.Errorf("invalid frame delay %d, the minimum is 1 and the max is 255", c)
		}
		result = append(result, AnimItem{FrameDelay: byte(c), Filename: record[1]})
	}
	return result, nil
}
