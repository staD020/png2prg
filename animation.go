package png2prg

import (
	"bytes"
	"fmt"
	"io"
	"log"

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
		if img.graphicsType != wantedGraphicsType {
			return n, fmt.Errorf("mixed graphicsmodes detected %q != %q", img.graphicsType, wantedGraphicsType)
		}
		if err := img.analyze(); err != nil {
			return n, fmt.Errorf("warning: skipping frame %d, analyze failed: %w", i, err)
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
		m, err := writeAnimationDisplayerTo(w, imgs, kk, hh, scSprites, mcSprites)
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

func writeAnimationDisplayerTo(w io.Writer, imgs []sourceImage, kk []Koala, hh []Hires, scSprites []SingleColorSprites, mcSprites []MultiColorSprites) (n int64, err error) {
	buf := &bytes.Buffer{}
	opt := imgs[0].opt
	switch {
	case len(kk) > 0:
		// handle display koala animation
		if opt.NoCrunch {
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
		if opt.NoCrunch {
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

	tscopt := TSCOptions
	if opt.Verbose {
		tscopt.QUIET = false
	}
	tsc, err := TSCrunch.New(tscopt, buf)
	if err != nil {
		return n, fmt.Errorf("tscrunch.New failed: %w", err)
	}
	if !opt.Quiet {
		fmt.Println("packing with TSCrunch...")
	}
	m, err := tsc.WriteTo(w)
	n += m
	if err != nil {
		return n, fmt.Errorf("tsc.WriteTo failed: %w", err)
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
	header := append([]byte{}, koalaDisplayAnim...)
	opt := kk[0].opt
	if opt.AlternativeFade {
		header = append([]byte{}, koalaDisplayAnimAlternative...)
	}
	header[0x820-0x7ff] = byte(opt.FrameDelay)
	header[0x821-0x7ff] = byte(opt.WaitSeconds)

	frames := make([]Charer, len(kk))
	for i := range kk {
		frames[i] = kk[i]
	}
	framePrgs, err := processAnimation(opt, frames)
	if err != nil {
		return n, err
	}
	if !opt.Quiet {
		fmt.Printf("memory usage for displayer code: 0x%04x - 0x%04x\n", 0x0801, len(header)+0x7ff)
	}

	if opt.IncludeSID == "" {
		buf := make([]byte, 0, 64*1024)

		header = zeroFill(header, BitmapAddress-0x7ff-len(header))
		k := kk[0]
		out := [][]byte{header, k.Bitmap[:], k.ScreenColor[:], k.D800Color[:], {bgBorder}}
		for _, bin := range out {
			buf = append(buf, bin...)
		}
		if !opt.Quiet {
			fmt.Printf("memory usage for picture: 0x%04x - 0x%04x\n", BitmapAddress, len(buf)+0x7ff)
		}
		buf = zeroFill(buf, koalaAnimationStart-0x7ff-len(buf))
		t1 := len(buf)
		for _, bin := range framePrgs {
			buf = append(buf, bin...)
		}
		buf = append(buf, 0xff)
		if !opt.Quiet {
			fmt.Printf("memory usage for animations: 0x%04x - 0x%04x\n", t1+0x7ff, len(buf)+0x7ff)
			fmt.Printf("memory usage for generated fadecode: 0x%04x - 0x%04x\n", koalaFadePassStart, 0xcfff)
		}
		m, err := w.Write(buf)
		n += int64(m)
		return n, err
	}

	s, err := sid.LoadSID(opt.IncludeSID)
	if err != nil {
		return n, fmt.Errorf("sid.LoadSID failed: %w", err)
	}
	header = injectSIDHeader(header, s)
	load := s.LoadAddress()
	switch {
	case int(load) < len(header)+0x7ff:
		return n, fmt.Errorf("sid LoadAddress %s is too low for sid %s", load, s)
	case load > 0xdff && load < 0x1fff:
		header = zeroFill(header, int(load)-0x7ff-len(header))
		header = append(header, s.RawBytes()...)
		if len(header) > BitmapAddress-0x7ff {
			return n, fmt.Errorf("sid memory overflow 0x%04x for sid %s", len(header)+0x7ff, s)
		}
		if !opt.Quiet {
			fmt.Printf("memory usage for sid: %s - %s (%q by %s)\n", s.LoadAddress(), s.LoadAddress()+sid.Word(len(s.RawBytes())), s.Name(), s.Author())
		}
		header = zeroFill(header, BitmapAddress-0x7ff-len(header))
		if !opt.Quiet {
			fmt.Printf("memory usage for picture: 0x%04x - 0x%04x\n", BitmapAddress, 0x4711)
		}
		buf := make([]byte, koalaAnimationStart-0x4711)
		for _, bin := range framePrgs {
			buf = append(buf, bin...)
		}
		buf = append(buf, 0xff)
		if !opt.Quiet {
			fmt.Printf("memory usage for animations: 0x%04x - 0x%04x\n", koalaAnimationStart, len(buf)+0x4711)
			fmt.Printf("memory usage for generated fadecode: 0x%04x - 0x%04x\n", koalaFadePassStart, 0xcfff)
		}
		return writeData(w, header, kk[0].Bitmap[:], kk[0].ScreenColor[:], kk[0].D800Color[:], []byte{bgBorder}, buf)
	case (load > koalaFadePassStart && load < 0xe000) || load < koalaAnimationStart+0x100:
		return n, fmt.Errorf("sid LoadAddress %s is causing memory overlap for sid %s", load, s)
	}

	header = zeroFill(header, BitmapAddress-0x7ff-len(header))
	if !opt.Quiet {
		fmt.Printf("memory usage for picture: 0x%04x - 0x%04x\n", BitmapAddress, 0x4711)
	}

	framebuf := make([]byte, koalaAnimationStart-0x4711)
	for _, bin := range framePrgs {
		framebuf = append(framebuf, bin...)
	}
	framebuf = append(framebuf, 0xff)
	if !opt.Quiet {
		fmt.Printf("memory usage for animations: 0x%04x - 0x%04x\n", koalaAnimationStart, len(framebuf)+0x4711)
		fmt.Printf("memory usage for sid: %s - %s (%q by %s)\n", s.LoadAddress(), s.LoadAddress()+sid.Word(len(s.RawBytes())), s.Name(), s.Author())
		fmt.Printf("memory usage for generated fadecode: 0x%04x - 0x%04x\n", koalaFadePassStart, 0xcfff)
	}

	buf := make([]byte, int(load)-0x4711-len(framebuf))
	n, err = writeData(w, header, kk[0].Bitmap[:], kk[0].ScreenColor[:], kk[0].D800Color[:], []byte{bgBorder}, framebuf, buf, s.RawBytes())
	if err != nil {
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

func WriteHiresDisplayAnimTo(w io.Writer, hh []Hires) (n int64, err error) {
	opt := hh[0].opt
	header := append([]byte{}, hiresDisplayAnim...)
	header[0x820-0x7ff] = byte(opt.FrameDelay)
	header[0x821-0x7ff] = byte(opt.WaitSeconds)

	frames := make([]Charer, len(hh))
	for i := range hh {
		frames[i] = hh[i]
	}
	framePrgs, err := processAnimation(opt, frames)
	if err != nil {
		return n, err
	}
	if !opt.Quiet {
		fmt.Printf("memory usage for displayer code: 0x%04x - 0x%04x\n", 0x0801, len(header)+0x7ff)
	}

	if opt.IncludeSID == "" {
		buf := make([]byte, 0, 64*1024)

		header = zeroFill(header, BitmapAddress-0x7ff-len(header))
		h := hh[0]
		out := [][]byte{header, h.Bitmap[:], h.ScreenColor[:], {h.BorderColor}}

		for _, bin := range out {
			buf = append(buf, bin...)
		}
		if !opt.Quiet {
			fmt.Printf("memory usage for picture: 0x%04x - 0x%04x\n", BitmapAddress, 0x4329)
		}

		buf = zeroFill(buf, hiresAnimationStart-0x7ff-len(buf))
		for _, bin := range framePrgs {
			buf = append(buf, bin...)
		}
		buf = append(buf, 0xff)

		if !opt.Quiet {
			fmt.Printf("memory usage for animations: 0x%04x - 0x%04x\n", hiresAnimationStart, len(buf)+0x7ff)
			fmt.Printf("memory usage for generated fadecode: 0x%04x - 0x%04x\n", hiresFadePassStart, 0xcfff)
		}

		m, err := w.Write(buf)
		n += int64(m)
		return n, err
	}

	s, err := sid.LoadSID(opt.IncludeSID)
	if err != nil {
		return n, fmt.Errorf("sid.LoadSID failed: %w", err)
	}
	header = injectSIDHeader(header, s)
	load := s.LoadAddress()
	switch {
	case int(load) < len(header)+0x7ff:
		return n, fmt.Errorf("sid LoadAddress %s is too low for sid %s", load, s)
	case load > 0xdff && load < 0x1fff:
		header = zeroFill(header, int(load)-0x7ff-len(header))
		header = append(header, s.RawBytes()...)
		if len(header) > BitmapAddress-0x7ff {
			return n, fmt.Errorf("sid memory overflow 0x%04x for sid %s", len(header)+0x7ff, s)
		}
		if !opt.Quiet {
			fmt.Printf("memory usage for sid: %s - %s (%q by %s)\n", s.LoadAddress(), s.LoadAddress()+sid.Word(len(s.RawBytes())), s.Name(), s.Author())
		}
		header = zeroFill(header, BitmapAddress-0x7ff-len(header))
		if !opt.Quiet {
			fmt.Printf("memory usage for picture: 0x%04x - 0x%04x\n", BitmapAddress, 0x4329)
		}
		framebuf := make([]byte, hiresAnimationStart-0x4329)
		for _, bin := range framePrgs {
			framebuf = append(framebuf, bin...)
		}
		framebuf = append(framebuf, 0xff)
		if !opt.Quiet {
			fmt.Printf("memory usage for animations: 0x%04x - 0x%04x\n", hiresAnimationStart, len(framebuf)+0x4328)
			fmt.Printf("memory usage for generated fadecode: 0x%04x - 0x%04x\n", hiresFadePassStart, 0xcfff)
		}
		return writeData(w, header, hh[0].Bitmap[:], hh[0].ScreenColor[:], []byte{hh[0].BorderColor}, framebuf)
	case (load > hiresFadePassStart && load < 0xe000) || load < hiresAnimationStart+0x100:
		return n, fmt.Errorf("sid LoadAddress %s is causing memory overlap for sid %s", load, s)
	}

	header = zeroFill(header, BitmapAddress-0x7ff-len(header))
	if !opt.Quiet {
		fmt.Printf("memory usage for picture: 0x%04x - 0x%04x\n", BitmapAddress, 0x4329)
	}
	framebuf := make([]byte, hiresAnimationStart-0x4329)
	for _, bin := range framePrgs {
		framebuf = append(framebuf, bin...)
	}
	framebuf = append(framebuf, 0xff)
	if !opt.Quiet {
		fmt.Printf("memory usage for animations: 0x%04x - 0x%04x\n", hiresAnimationStart, len(framebuf)+0x4328)
		fmt.Printf("memory usage for sid: %s - %s (%q by %s)\n", s.LoadAddress(), s.LoadAddress()+sid.Word(len(s.RawBytes())), s.Name(), s.Author())
		fmt.Printf("memory usage for generated fadecode: 0x%04x - 0x%04x\n", hiresFadePassStart, 0xcfff)
	}

	buf := make([]byte, int(load)-0x4329-len(framebuf))
	n, err = writeData(w, header, hh[0].Bitmap[:], hh[0].ScreenColor[:], []byte{hh[0].BorderColor}, framebuf, buf, s.RawBytes())
	if err != nil {
		return n, err
	}
	return n, nil
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
