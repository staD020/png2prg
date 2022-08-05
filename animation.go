package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/staD020/TSCrunch"
	"github.com/staD020/sid"
)

const (
	bitmapStart = 0x2000

	koalaFadePassStart  = 0x8900
	koalaAnimationStart = 0x4800

	hiresFadePassStart  = 0xac00
	hiresAnimationStart = 0x4400
)

func handleAnimation(imgs []sourceImage) error {
	var kk []Koala
	var hh []Hires
	var scSprites []SingleColorSprites
	var mcSprites []MultiColorSprites
	if len(imgs) < 1 {
		return fmt.Errorf("no sourceImage given")
	}
	currentGraphicsType := imgs[0].graphicsType
	currentBitpairColors := bitpairColors{}
	for i, img := range imgs {
		if !quiet {
			fmt.Printf("processing %q frame %d\n", img.sourceFilename, i)
		}
		if img.graphicsType != currentGraphicsType {
			return fmt.Errorf("mixed graphicsmodes detected %q != %q", img.graphicsType, currentGraphicsType)
		}
		if err := img.analyze(); err != nil {
			log.Printf("warning: skipping frame %d, analyze failed: %v", i, err)
			continue
		}
		if len(currentBitpairColors) == 0 {
			currentBitpairColors = img.preferredBitpairColors
		}
		if currentBitpairColors.String() != img.preferredBitpairColors.String() {
			log.Printf("warning: bitpairColors %q of the previous frame do not equal current frame %q", currentBitpairColors, img.preferredBitpairColors)
			log.Println("this will cause huge animation frame sizes and probably crash the displayer")
			return fmt.Errorf("bitpairColors differ between frames")
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

	if display {
		if err = writeAnimationDisplayerTo(f, imgs, kk, hh, scSprites, mcSprites); err != nil {
			return fmt.Errorf("writeAnimationDisplayerTo %q failed: %w", f.Name(), err)
		}
		if !quiet {
			fmt.Printf("write %q\n", f.Name())
		}
		return nil
	}

	// export separate frame data (non displayer)
	switch {
	case len(kk) > 0:
		if _, err = kk[0].WriteTo(f); err != nil {
			return fmt.Errorf("WriteTo %q failed: %w", destFilename, err)
		}
		if !quiet {
			fmt.Printf("converted %q to %q\n", kk[0].SourceFilename, destFilename)
		}

		cc := make([]Charer, len(kk))
		for i := range kk {
			cc[i] = kk[i]
		}
		prgs, err := processAnimation(cc)
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
		if _, err = hh[0].WriteTo(f); err != nil {
			return fmt.Errorf("WriteTo %q failed: %w", destFilename, err)
		}
		if !quiet {
			fmt.Printf("converted %q to %q\n", hh[0].SourceFilename, destFilename)
		}

		cc := make([]Charer, len(hh))
		for i := range hh {
			cc[i] = hh[i]
		}
		prgs, err := processAnimation(cc)
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
		data := [][]byte{defaultHeader()}
		for _, s := range mcSprites {
			data = append(data, s.Bitmap)
			if !quiet {
				fmt.Printf("converted %q to %q\n", s.SourceFilename, destFilename)
			}
		}
		if _, err = writeData(f, data); err != nil {
			return fmt.Errorf("writeData %q failed: %w", destFilename, err)
		}
		return nil
	case len(scSprites) > 0:
		data := [][]byte{defaultHeader()}
		for _, s := range scSprites {
			data = append(data, s.Bitmap)
			if !quiet {
				fmt.Printf("converted %q to %q\n", s.SourceFilename, destFilename)
			}
		}
		if _, err = writeData(f, data); err != nil {
			return fmt.Errorf("writeData %q failed: %w", destFilename, err)
		}
		return nil
	}
	return fmt.Errorf("handleAnimation %q failed: no frames written", imgs[0].sourceFilename)
}

func writeAnimationDisplayerTo(w io.Writer, imgs []sourceImage, kk []Koala, hh []Hires, scSprites []SingleColorSprites, mcSprites []MultiColorSprites) (err error) {
	buf := &bytes.Buffer{}
	switch {
	case len(kk) > 0:
		// handle display koala animation
		if noCrunch {
			if _, err = WriteKoalaDisplayAnimTo(w, kk); err != nil {
				return fmt.Errorf("WriteKoalaDisplayAnimTo failed: %w", err)
			}
			return nil
		}
		if _, err = WriteKoalaDisplayAnimTo(buf, kk); err != nil {
			return fmt.Errorf("WriteKoalaDisplayAnimTo buf failed: %w", err)
		}
	case len(hh) > 0:
		// handle display hires animation
		if noCrunch {
			if _, err = WriteHiresDisplayAnimTo(w, hh); err != nil {
				return fmt.Errorf("WriteHiresDisplayAnimTo failed: %w", err)
			}
			return nil
		}
		if _, err = WriteHiresDisplayAnimTo(buf, hh); err != nil {
			return fmt.Errorf("WriteHiresDisplayAnimTo buf failed: %w", err)
		}
	default:
		return fmt.Errorf("animation displayers do not support %q", imgs[0].graphicsType)
	}

	opt := TSCrunch.Options{
		PRG:     true,
		QUIET:   true,
		INPLACE: false,
		JumpTo:  displayerJumpTo,
	}
	if verbose {
		opt.QUIET = false
	}
	tsc, err := TSCrunch.New(opt, buf)
	if err != nil {
		return fmt.Errorf("tscrunch.New failed: %w", err)
	}
	if !quiet {
		fmt.Println("packing with TSCrunch...")
	}
	if _, err = tsc.WriteTo(w); err != nil {
		return fmt.Errorf("tsc.WriteTo failed: %w", err)
	}
	return nil
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

	if _, err = writeData(f, [][]byte{defaultHeader(), prg}); err != nil {
		return fmt.Errorf("writeData %q failed: %w", filename, err)
	}
	if !quiet {
		fmt.Printf("write %q\n", filename)
	}
	return nil
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

func processFramesOfChars(frames [][]Char) ([][]byte, error) {
	if len(frames) < 2 {
		return nil, fmt.Errorf("insufficient number of images %d < 2", len(frames))
	}

	prgs := make([][]byte, 0)
	for i, frame := range frames {
		if verbose {
			log.Printf("frame %d length in changed chars: %d", i, len(frame))
		}

		curChar := -10
		curChunk := chunk{}
		prg := []byte{}
		for _, char := range frame {
			switch {
			case curChar == char.Index()-1:
				// next char of current chunk
				curChunk.appendChar(char)
			default:
				// new chunk
				if curChunk.charCount > 0 {
					if verbose {
						log.Println(curChunk.String())
					}
					prg = append(prg, curChunk.export()...)
				}
				curChunk = newChunk(char.Index())
				curChunk.appendChar(char)
			}
			curChar = char.Index()
		}
		// add last chunk
		if curChunk.charCount > 0 {
			if verbose {
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

func processAnimation(imgs []Charer) ([][]byte, error) {
	if len(imgs) < 2 {
		return nil, fmt.Errorf("insufficient number of images %d < 2", len(imgs))
	}
	if verbose {
		log.Printf("total number of frames: %d", len(imgs))
	}

	frames := make([][]Char, len(imgs))
	for i := 0; i < len(imgs)-1; i++ {
		frames[i] = make([]Char, 0)
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
				frames[j] = append(frames[j], frameChar)
			}
		}
	}
	return processFramesOfChars(frames)
}

func WriteKoalaDisplayAnimTo(w io.Writer, kk []Koala) (n int64, err error) {
	bgBorder := kk[0].BackgroundColor | kk[0].BorderColor<<4
	header := append([]byte{}, koalaDisplayAnim...)
	if alternativeFade {
		header = append([]byte{}, koalaDisplayAnimAlternative...)
	}
	header[0x820-0x7ff] = byte(frameDelay)
	header[0x821-0x7ff] = byte(waitSeconds)

	frames := make([]Charer, len(kk))
	for i := range kk {
		frames[i] = kk[i]
	}
	framePrgs, err := processAnimation(frames)
	if err != nil {
		return n, err
	}
	if !quiet {
		fmt.Printf("memory usage for displayer code: 0x%04x - 0x%04x\n", 0x0801, len(header)+0x7ff)
	}

	if includeSID == "" {
		buf := make([]byte, 0, 64*1024)

		header = zeroFill(header, bitmapStart-0x7ff-len(header))
		k := kk[0]
		out := [][]byte{header, k.Bitmap[:], k.ScreenColor[:], k.D800Color[:], {bgBorder}}
		for _, bin := range out {
			buf = append(buf, bin...)
		}
		if !quiet {
			fmt.Printf("memory usage for picture: 0x%04x - 0x%04x\n", bitmapStart, len(buf)+0x7ff)
		}
		buf = zeroFill(buf, koalaAnimationStart-0x7ff-len(buf))
		t1 := len(buf)
		for _, bin := range framePrgs {
			buf = append(buf, bin...)
		}
		buf = append(buf, 0xff)
		if !quiet {
			fmt.Printf("memory usage for animations: 0x%04x - 0x%04x\n", t1+0x7ff, len(buf)+0x7ff)
			fmt.Printf("memory usage for generated fadecode: 0x%04x - 0x%04x\n", koalaFadePassStart, 0xcfff)
		}
		m, err := w.Write(buf)
		n += int64(m)
		return n, err
	}

	s, err := sid.LoadSID(includeSID)
	if err != nil {
		return 0, fmt.Errorf("sid.LoadSID failed: %w", err)
	}
	header = injectSIDHeader(header, s)
	load := s.LoadAddress()
	switch {
	case int(load) < len(header)+0x7ff:
		return 0, fmt.Errorf("sid LoadAddress %s is too low for sid %s", load, s)
	case load > 0xdff && load < 0x1fff:
		header = zeroFill(header, int(load)-0x7ff-len(header))
		header = append(header, s.RawBytes()...)
		if len(header) > bitmapStart-0x7ff {
			return 0, fmt.Errorf("sid memory overflow 0x%04x for sid %s", len(header)+0x7ff, s)
		}
		if !quiet {
			fmt.Printf("memory usage for sid: %s - %s (%q by %s)\n", s.LoadAddress(), s.LoadAddress()+sid.Word(len(s.RawBytes())), s.Name(), s.Author())
		}
		header = zeroFill(header, bitmapStart-0x7ff-len(header))
		if !quiet {
			fmt.Printf("memory usage for picture: 0x%04x - 0x%04x\n", bitmapStart, 0x4711)
		}
		buf := make([]byte, koalaAnimationStart-0x4711)
		for _, bin := range framePrgs {
			buf = append(buf, bin...)
		}
		buf = append(buf, 0xff)
		if !quiet {
			fmt.Printf("memory usage for animations: 0x%04x - 0x%04x\n", koalaAnimationStart, len(buf)+0x4711)
			fmt.Printf("memory usage for generated fadecode: 0x%04x - 0x%04x\n", koalaFadePassStart, 0xcfff)
		}
		return writeData(w, [][]byte{header, kk[0].Bitmap[:], kk[0].ScreenColor[:], kk[0].D800Color[:], {bgBorder}, buf})
	case (load > koalaFadePassStart && load < 0xe000) || load < koalaAnimationStart+0x100:
		return 0, fmt.Errorf("sid LoadAddress %s is causing memory overlap for sid %s", load, s)
	}

	header = zeroFill(header, bitmapStart-0x7ff-len(header))
	if !quiet {
		fmt.Printf("memory usage for picture: 0x%04x - 0x%04x\n", bitmapStart, 0x4711)
	}

	framebuf := make([]byte, koalaAnimationStart-0x4711)
	for _, bin := range framePrgs {
		framebuf = append(framebuf, bin...)
	}
	framebuf = append(framebuf, 0xff)
	if !quiet {
		fmt.Printf("memory usage for animations: 0x%04x - 0x%04x\n", koalaAnimationStart, len(framebuf)+0x4711)
		fmt.Printf("memory usage for sid: %s - %s (%q by %s)\n", s.LoadAddress(), s.LoadAddress()+sid.Word(len(s.RawBytes())), s.Name(), s.Author())
		fmt.Printf("memory usage for generated fadecode: 0x%04x - 0x%04x\n", koalaFadePassStart, 0xcfff)
	}

	buf := make([]byte, int(load)-0x4711-len(framebuf))
	n, err = writeData(w, [][]byte{header, kk[0].Bitmap[:], kk[0].ScreenColor[:], kk[0].D800Color[:], {bgBorder}, framebuf, buf, s.RawBytes()})
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
	header := append([]byte{}, hiresDisplayAnim...)
	header[0x820-0x7ff] = byte(frameDelay)
	header[0x821-0x7ff] = byte(waitSeconds)

	frames := make([]Charer, len(hh))
	for i := range hh {
		frames[i] = hh[i]
	}
	framePrgs, err := processAnimation(frames)
	if err != nil {
		return n, err
	}
	if !quiet {
		fmt.Printf("memory usage for displayer code: 0x%04x - 0x%04x\n", 0x0801, len(header)+0x7ff)
	}

	if includeSID == "" {
		buf := make([]byte, 0, 64*1024)

		header = zeroFill(header, bitmapStart-0x7ff-len(header))
		h := hh[0]
		out := [][]byte{header, h.Bitmap[:], h.ScreenColor[:], {h.BorderColor}}

		for _, bin := range out {
			buf = append(buf, bin...)
		}
		if !quiet {
			fmt.Printf("memory usage for picture: 0x%04x - 0x%04x\n", bitmapStart, 0x4329)
		}

		buf = zeroFill(buf, hiresAnimationStart-0x7ff-len(buf))
		for _, bin := range framePrgs {
			buf = append(buf, bin...)
		}
		buf = append(buf, 0xff)

		if !quiet {
			fmt.Printf("memory usage for animations: 0x%04x - 0x%04x\n", hiresAnimationStart, len(buf)+0x7ff)
			fmt.Printf("memory usage for generated fadecode: 0x%04x - 0x%04x\n", hiresFadePassStart, 0xcfff)
		}

		m, err := w.Write(buf)
		n += int64(m)
		return n, err
	}

	s, err := sid.LoadSID(includeSID)
	if err != nil {
		return 0, fmt.Errorf("sid.LoadSID failed: %w", err)
	}
	header = injectSIDHeader(header, s)
	load := s.LoadAddress()
	switch {
	case int(load) < len(header)+0x7ff:
		return 0, fmt.Errorf("sid LoadAddress %s is too low for sid %s", load, s)
	case load > 0xdff && load < 0x1fff:
		header = zeroFill(header, int(load)-0x7ff-len(header))
		header = append(header, s.RawBytes()...)
		if len(header) > bitmapStart-0x7ff {
			return 0, fmt.Errorf("sid memory overflow 0x%04x for sid %s", len(header)+0x7ff, s)
		}
		if !quiet {
			fmt.Printf("memory usage for sid: %s - %s (%q by %s)\n", s.LoadAddress(), s.LoadAddress()+sid.Word(len(s.RawBytes())), s.Name(), s.Author())
		}
		header = zeroFill(header, bitmapStart-0x7ff-len(header))
		if !quiet {
			fmt.Printf("memory usage for picture: 0x%04x - 0x%04x\n", bitmapStart, 0x4329)
		}
		framebuf := make([]byte, hiresAnimationStart-0x4329)
		for _, bin := range framePrgs {
			framebuf = append(framebuf, bin...)
		}
		framebuf = append(framebuf, 0xff)
		if !quiet {
			fmt.Printf("memory usage for animations: 0x%04x - 0x%04x\n", hiresAnimationStart, len(framebuf)+0x4328)
			fmt.Printf("memory usage for generated fadecode: 0x%04x - 0x%04x\n", hiresFadePassStart, 0xcfff)
		}
		return writeData(w, [][]byte{header, hh[0].Bitmap[:], hh[0].ScreenColor[:], {hh[0].BorderColor}, framebuf})
	case (load > hiresFadePassStart && load < 0xe000) || load < hiresAnimationStart+0x100:
		return 0, fmt.Errorf("sid LoadAddress %s is causing memory overlap for sid %s", load, s)
	}

	header = zeroFill(header, bitmapStart-0x7ff-len(header))
	if !quiet {
		fmt.Printf("memory usage for picture: 0x%04x - 0x%04x\n", bitmapStart, 0x4329)
	}
	framebuf := make([]byte, hiresAnimationStart-0x4329)
	for _, bin := range framePrgs {
		framebuf = append(framebuf, bin...)
	}
	framebuf = append(framebuf, 0xff)
	if !quiet {
		fmt.Printf("memory usage for animations: 0x%04x - 0x%04x\n", hiresAnimationStart, len(framebuf)+0x4328)
		fmt.Printf("memory usage for sid: %s - %s (%q by %s)\n", s.LoadAddress(), s.LoadAddress()+sid.Word(len(s.RawBytes())), s.Name(), s.Author())
		fmt.Printf("memory usage for generated fadecode: 0x%04x - 0x%04x\n", hiresFadePassStart, 0xcfff)
	}

	buf := make([]byte, int(load)-0x4329-len(framebuf))
	n, err = writeData(w, [][]byte{header, hh[0].Bitmap[:], hh[0].ScreenColor[:], {hh[0].BorderColor}, framebuf, buf, s.RawBytes()})
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

func (c *chunk) appendChar(char Char) {
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
