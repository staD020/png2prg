package png2prg

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/staD020/TSCrunch"
	"github.com/staD020/sid"
)

const (
	koalaFadePassStart  = 0x8900
	koalaAnimationStart = 0x4800

	hiresFadePassStart  = 0xac00
	hiresAnimationStart = 0x4400
)

func (c *converter) WriteAnimationTo(w io.Writer) (n int64, err error) {
	var kk []Koala
	var hh []Hires
	var scSprites []SingleColorSprites
	var mcSprites []MultiColorSprites
	imgs := c.images
	if len(imgs) < 1 {
		return n, fmt.Errorf("no sourceImage given")
	}
	wantedGraphicsType := imgs[0].graphicsType
	currentBitpairColors := bitpairColors{}
	for i, img := range imgs {
		if !c.opt.Quiet {
			fmt.Printf("processing %q frame %d\n", img.sourceFilename, i)
		}
		if i > 0 {
			if err := img.analyze(); err != nil {
				return n, fmt.Errorf("warning: skipping frame %d, analyze failed: %w", i, err)
			}
		}
		if img.graphicsType != wantedGraphicsType {
			return n, fmt.Errorf("mixed graphicsmodes detected %q != %q", img.graphicsType, wantedGraphicsType)
		}
		if len(currentBitpairColors) == 0 {
			currentBitpairColors = img.preferredBitpairColors
		}
		if currentBitpairColors.String() != img.preferredBitpairColors.String() {
			log.Printf("bitpairColors %q of the previous frame do not equal current frame %q", currentBitpairColors, img.preferredBitpairColors)
			log.Println("this would cause huge animation frame sizes and probably crash the displayer")
			return n, fmt.Errorf("bitpairColors differ between frames, maybe use -bitpair-colors %s", currentBitpairColors)
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
		default:
			return n, fmt.Errorf("animations do not support %q yet", img.graphicsType)
		}
	}

	if c.opt.Display {
		m, err := c.writeAnimationDisplayerTo(w, imgs, kk, hh, scSprites, mcSprites)
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

		frames := make([]Charer, len(kk))
		for i := range kk {
			frames[i] = kk[i]
		}
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

		frames := make([]Charer, len(hh))
		for i := range hh {
			frames[i] = hh[i]
		}
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
	}
	return n, fmt.Errorf("handleAnimation %q failed: no frames written", imgs[0].sourceFilename)
}

func (c *converter) writeAnimationDisplayerTo(w io.Writer, imgs []sourceImage, kk []Koala, hh []Hires, scSprites []SingleColorSprites, mcSprites []MultiColorSprites) (n int64, err error) {
	buf := &bytes.Buffer{}
	switch {
	case len(kk) > 0:
		// handle display koala animation
		c.Symbols = append(c.Symbols, kk[0].Symbols()...)
		c.Symbols = append(c.Symbols, c64Symbol{"animation", koalaAnimationStart})
		if c.opt.NoCrunch {
			m, err := WriteKoalaDisplayAnimTo(w, kk)
			n += m
			if err != nil {
				return n, fmt.Errorf("WriteKoalaDisplayAnimTo failed: %w", err)
			}
			return n, nil
		}
		if _, err = WriteKoalaDisplayAnimTo(buf, kk); err != nil {
			return n, fmt.Errorf("WriteKoalaDisplayAnimTo buf failed: %w", err)
		}
	case len(hh) > 0:
		// handle display hires animation
		c.Symbols = append(c.Symbols, hh[0].Symbols()...)
		c.Symbols = append(c.Symbols, c64Symbol{"animation", hiresAnimationStart})
		if c.opt.NoCrunch {
			m, err := WriteHiresDisplayAnimTo(w, hh)
			n += m
			if err != nil {
				return n, fmt.Errorf("WriteHiresDisplayAnimTo failed: %w", err)
			}
			return n, nil
		}
		if _, err = WriteHiresDisplayAnimTo(buf, hh); err != nil {
			return n, fmt.Errorf("WriteHiresDisplayAnimTo buf failed: %w", err)
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

func WriteKoalaDisplayAnimTo(w io.Writer, kk []Koala) (n int64, err error) {
	bgBorder := kk[0].BackgroundColor | kk[0].BorderColor<<4
	opt := kk[0].opt

	frames := make([]Charer, len(kk))
	for i := range kk {
		frames[i] = kk[i]
	}
	framePrgs, err := processAnimation(opt, frames)
	if err != nil {
		return n, err
	}

	displayer := koalaDisplayAnim
	if opt.AlternativeFade {
		displayer = koalaDisplayAnimAlternative
	}
	link := NewLinker(0, opt.Verbose)
	if _, err = link.WritePrg(displayer); err != nil {
		return n, err
	}
	link.Block(koalaFadePassStart, 0xd000)
	link.SetByte(0x820, byte(opt.FrameDelay))
	link.SetByte(0x821, byte(opt.WaitSeconds))
	if !opt.Quiet {
		fmt.Printf("memory usage for displayer code: %s - %s\n", link.StartAddress(), link.EndAddress())
	}

	link.SetCursor(BitmapAddress)
	k := kk[0]
	for _, b := range [][]byte{k.Bitmap[:], k.ScreenColor[:], k.D800Color[:], {bgBorder}} {
		if _, err = link.Write(b); err != nil {
			return n, fmt.Errorf("link.Write error: %w", err)
		}
	}
	if !opt.Quiet {
		fmt.Printf("memory usage for picture: 0x%04x - %s\n", BitmapAddress, link.EndAddress())
	}

	link.SetCursor(koalaAnimationStart)
	for _, bin := range framePrgs {
		if _, err = link.Write(bin); err != nil {
			return n, fmt.Errorf("link.Write error: %w", err)
		}
	}
	if _, err = link.Write([]byte{0xff}); err != nil {
		return n, fmt.Errorf("link.Write error: %w", err)
	}

	if !opt.Quiet {
		fmt.Printf("memory usage for animations: %s - %s\n", Word(koalaAnimationStart), link.EndAddress())
		fmt.Printf("memory usage for generated fadecode: %s - %s\n", Word(koalaFadePassStart), Word(0xcfff))
	}

	if opt.IncludeSID != "" {
		s, err := sid.LoadSID(opt.IncludeSID)
		if err != nil {
			return n, fmt.Errorf("sid.LoadSID failed: %w", err)
		}
		if _, err = link.WritePrg(s.Bytes()); err != nil {
			return n, fmt.Errorf("link.WritePrg failed: %w", err)
		}
		injectSIDLinker(link, s)
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

func WriteHiresDisplayAnimTo(w io.Writer, hh []Hires) (n int64, err error) {
	opt := hh[0].opt
	frames := make([]Charer, len(hh))
	for i := range hh {
		frames[i] = hh[i]
	}
	framePrgs, err := processAnimation(opt, frames)
	if err != nil {
		return n, fmt.Errorf("processAnimation error: %w", err)
	}

	link := NewLinker(0, opt.Verbose)
	if _, err = link.WritePrg(hiresDisplayAnim); err != nil {
		return n, fmt.Errorf("link.WritePrg error: %w", err)
	}
	link.Block(hiresFadePassStart, 0xd000)
	link.SetByte(0x820, byte(opt.FrameDelay))
	link.SetByte(0x821, byte(opt.WaitSeconds))
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
		fmt.Printf("memory usage for picture: 0x%04x - %s\n", BitmapAddress, link.EndAddress())
	}

	link.SetCursor(hiresAnimationStart)
	for _, bin := range framePrgs {
		if _, err = link.Write(bin); err != nil {
			return n, fmt.Errorf("link.Write error: %w", err)
		}
	}
	if _, err = link.Write([]byte{0xff}); err != nil {
		return n, fmt.Errorf("link.Write error: %w", err)
	}
	if !opt.Quiet {
		fmt.Printf("memory usage for animations: 0x%04x - %s\n", hiresAnimationStart, link.EndAddress())
		fmt.Printf("memory usage for generated fadecode: 0x%04x - 0x%04x\n", hiresFadePassStart, 0xcfff)
	}

	if opt.IncludeSID != "" {
		s, err := sid.LoadSID(opt.IncludeSID)
		if err != nil {
			return n, fmt.Errorf("sid.LoadSID failed: %w", err)
		}
		if _, err = link.WritePrg(s.Bytes()); err != nil {
			return n, fmt.Errorf("link.WritePrg failed: %w", err)
		}
		injectSIDLinker(link, s)
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
