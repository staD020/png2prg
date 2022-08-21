package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/gif"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
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
	version         = "1.3.5-dev"
	displayerJumpTo = "$0822"
)

type RGB struct {
	R, G, B byte
}

func (r RGB) String() string {
	return fmt.Sprintf("RGB{0x%02x, 0x%02x, 0x%02x}", r.R, r.G, r.B)
}

type colorInfo struct {
	ColorIndex byte
	RGB        RGB
}

func (c colorInfo) String() string {
	//return fmt.Sprintf("{%d, #%02x%02x%02x}", c.ColorIndex, int(c.RGB.R), int(c.RGB.G), int(c.RGB.B))
	return fmt.Sprintf("{%d, %s},", c.ColorIndex, c.RGB)
}

type GraphicsType byte

const (
	unknownGraphicsType GraphicsType = iota
	singleColorBitmap
	multiColorBitmap
	singleColorCharset
	multiColorCharset
	singleColorSprites
	multiColorSprites
)

func stringToGraphicsType(s string) GraphicsType {
	switch s {
	case "koala":
		return multiColorBitmap
	case "hires":
		return singleColorBitmap
	case "sccharset":
		return singleColorCharset
	case "mccharset":
		return multiColorCharset
	case "scsprites":
		return singleColorSprites
	case "mcsprites":
		return multiColorSprites
	}
	return unknownGraphicsType
}

func (t GraphicsType) String() string {
	switch t {
	case singleColorBitmap:
		return "hires"
	case multiColorBitmap:
		return "koala"
	case singleColorCharset:
		return "singlecolor charset"
	case multiColorCharset:
		return "multicolor charset"
	case singleColorSprites:
		return "singlecolor sprites"
	case multiColorSprites:
		return "multicolor sprites"
	default:
		return "unknown"
	}
}

type bitpairColors []byte

func (b bitpairColors) String() (s string) {
	for i, v := range b {
		s = s + strconv.Itoa(int(v))
		if i < len(b)-1 {
			s += ","
		}
	}
	return s
}

type PaletteMap map[RGB]byte

func (m PaletteMap) devString() string {
	reverse := [16]*RGB{}
	for r, c := range m {
		r := r
		reverse[c] = &r
	}
	s := ""
	for c, r := range reverse {
		if r == nil {
			continue
		}
		s += fmt.Sprintf("{%d, %s}, ", c, *r)
	}
	return strings.TrimSuffix(s, ", ")
}

func (m PaletteMap) String() string {
	reverse := [16]*RGB{}
	for r, c := range m {
		r := r
		reverse[c] = &r
	}
	s := ""
	for c, r := range reverse {
		if r == nil {
			continue
		}
		s += fmt.Sprintf("{%d, #%02x%02x%02x}, ", c, int(r.R), int(r.G), int(r.B))
	}
	return strings.TrimSuffix(s, ", ")
}

type sourceImage struct {
	sourceFilename         string
	image                  image.Image
	xOffset                int
	yOffset                int
	width                  int
	height                 int
	palette                PaletteMap
	colors                 []RGB
	charColors             [1000]PaletteMap
	backgroundCandidates   PaletteMap
	backgroundColor        colorInfo
	borderColor            colorInfo
	preferredBitpairColors bitpairColors
	graphicsType           GraphicsType
}

type MultiColorChar struct {
	CharIndex       int
	Bitmap          [8]byte
	BackgroundColor byte
	ScreenColor     byte
	D800Color       byte
}

type SingleColorChar struct {
	CharIndex   int
	Bitmap      [8]byte
	ScreenColor byte
}

type Koala struct {
	SourceFilename  string
	Bitmap          [8000]byte
	ScreenColor     [1000]byte
	D800Color       [1000]byte
	BackgroundColor byte
	BorderColor     byte
}

type Hires struct {
	SourceFilename string
	Bitmap         [8000]byte
	ScreenColor    [1000]byte
	BorderColor    byte
}

type MultiColorCharset struct {
	SourceFilename  string
	Bitmap          [0x800]byte
	Screen          [1000]byte
	CharColor       byte
	BackgroundColor byte
	D022Color       byte
	D023Color       byte
	BorderColor     byte
}

type SingleColorCharset struct {
	SourceFilename  string
	Bitmap          [0x800]byte
	Screen          [1000]byte
	CharColor       byte
	BackgroundColor byte
	BorderColor     byte
}

type SingleColorSprites struct {
	SourceFilename  string
	Bitmap          []byte
	SpriteColor     byte
	BackgroundColor byte
	Columns         byte
	Rows            byte
}

type MultiColorSprites struct {
	SourceFilename  string
	Bitmap          []byte
	SpriteColor     byte
	BackgroundColor byte
	D025Color       byte
	D026Color       byte
	Columns         byte
	Rows            byte
}

var displayers = make(map[GraphicsType][]byte, 0)
var displayersAlternative = make(map[GraphicsType][]byte, 0)

//go:embed "display_koala.prg"
var koalaDisplay []byte

//go:embed "display_hires.prg"
var hiresDisplay []byte

//go:embed "display_mc_charset.prg"
var mcCharsetDisplay []byte

//go:embed "display_sc_charset.prg"
var scCharsetDisplay []byte

//go:embed "display_mc_sprites.prg"
var mcSpritesDisplay []byte

//go:embed "display_sc_sprites.prg"
var scSpritesDisplay []byte

//go:embed "display_koala_anim.prg"
var koalaDisplayAnim []byte

//go:embed "display_koala_anim_alternative.prg"
var koalaDisplayAnimAlternative []byte

//go:embed "display_hires_anim.prg"
var hiresDisplayAnim []byte

func init() {
	displayers[multiColorBitmap] = koalaDisplay
	displayers[singleColorBitmap] = hiresDisplay
	displayers[multiColorCharset] = mcCharsetDisplay
	displayers[singleColorCharset] = scCharsetDisplay
	displayers[multiColorSprites] = mcSpritesDisplay
	displayers[singleColorSprites] = scSpritesDisplay
}

// TODO: make png2prg a lib
type converter struct {
	opt    Options
	images []sourceImage
}

type Options struct {
	OutPath             string
	TargetDir           string
	Verbose             bool
	Display             bool
	NoPackChars         bool
	NoCrunch            bool
	AlternativeFade     bool
	BitpairColorsString string
	NoGuess             bool
	GraphicsMode        string
	ForceBorderColor    int
	IncludeSID          string
	FrameDelay          int
	WaitSeconds         int
	CurrentGraphicsType GraphicsType
}

func New(in map[string]io.Reader, opt Options) (*converter, error) {
	var err error
	imgs := []sourceImage{}
	for path, r := range in {
		switch strings.ToLower(filepath.Ext(path)) {
		case ".gif":
			g, err := gif.DecodeAll(r)
			if err != nil {
				return nil, fmt.Errorf("gif.DecodeAll %q failed: %w", path, err)
			}
			if verbose {
				log.Printf("file %q has %d frames", path, len(g.Image))
			}
			for i, rawImage := range g.Image {
				img := sourceImage{
					sourceFilename: path,
					image:          rawImage,
				}
				if err = img.setPreferredBitpairColors(bitpairColorsString); err != nil {
					return nil, fmt.Errorf("setPreferredBitpairColors %q failed: %w", bitpairColorsString, err)
				}
				if err = img.checkBounds(); err != nil {
					return nil, fmt.Errorf("img.checkBounds failed %q frame %d: %w", path, i, err)
				}
				imgs = append(imgs, img)
			}
		default:
			img := sourceImage{sourceFilename: path}
			if err = img.setPreferredBitpairColors(bitpairColorsString); err != nil {
				return nil, fmt.Errorf("setPreferredBitpairColors %q failed: %w", bitpairColorsString, err)
			}
			if img.image, _, err = image.Decode(r); err != nil {
				return nil, fmt.Errorf("image.Decode failed: %w", err)
			}
			if err = img.checkBounds(); err != nil {
				return nil, fmt.Errorf("img.checkBounds failed: %w", err)
			}
			imgs = append(imgs, img)
		}
	}
	return &converter{images: imgs, opt: opt}, nil
}

func NewFromPath(filenames []string, opt Options) (*converter, error) {
	m := make(map[string]io.Reader, len(filenames))
	for _, path := range filenames {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		m[path] = f
	}
	return New(m, opt)
}

func (c *converter) WriteTo(w io.Writer) (n int64, err error) {
	if len(c.images) == 0 {
		return 0, fmt.Errorf("no images found")
	}
	if len(c.images) > 1 {
		return 0, fmt.Errorf("%d images found", len(c.images))
	}

	img := c.images[0]
	if verbose {
		log.Printf("processing file %q", img.sourceFilename)
	}
	if err = img.analyze(); err != nil {
		return 0, fmt.Errorf("analyze %q failed: %w", img.sourceFilename, err)
	}

	var wt io.WriterTo
	switch img.graphicsType {
	case multiColorBitmap:
		if wt, err = img.convertToKoala(); err != nil {
			return 0, fmt.Errorf("convertToKoala %q failed: %w", img.sourceFilename, err)
		}
	case singleColorBitmap:
		if wt, err = img.convertToHires(); err != nil {
			return 0, fmt.Errorf("convertToHires %q failed: %w", img.sourceFilename, err)
		}
	case singleColorCharset:
		if wt, err = img.convertToSingleColorCharset(); err != nil {
			if graphicsMode != "" {
				return 0, fmt.Errorf("convertToSingleColorCharset %q failed: %w", img.sourceFilename, err)
			}
			if !quiet {
				fmt.Printf("falling back to %s because convertToSingleColorCharset %q failed: %v\n", singleColorBitmap, img.sourceFilename, err)
			}
			img.graphicsType = singleColorBitmap
			if wt, err = img.convertToHires(); err != nil {
				return 0, fmt.Errorf("convertToHires %q failed: %w", img.sourceFilename, err)
			}
		}
	case multiColorCharset:
		if wt, err = img.convertToMultiColorCharset(); err != nil {
			if graphicsMode != "" {
				return 0, fmt.Errorf("convertToMultiColorCharset %q failed: %w", img.sourceFilename, err)
			}
			if !quiet {
				fmt.Printf("falling back to %s because convertToMultiColorCharset %q failed: %v\n", multiColorBitmap, img.sourceFilename, err)
			}
			img.graphicsType = multiColorBitmap
			err = img.findBackgroundColor()
			if err != nil {
				return 0, fmt.Errorf("findBackgroundColor %q failed: %w", img.sourceFilename, err)
			}
			if wt, err = img.convertToKoala(); err != nil {
				return 0, fmt.Errorf("convertToKoala %q failed: %w", img.sourceFilename, err)
			}
		}
	case singleColorSprites:
		if wt, err = img.convertToSingleColorSprites(); err != nil {
			return 0, fmt.Errorf("convertToSingleColorSprites %q failed: %w", img.sourceFilename, err)
		}
	case multiColorSprites:
		if wt, err = img.convertToMultiColorSprites(); err != nil {
			return 0, fmt.Errorf("convertToMultiColorSprites %q failed: %w", img.sourceFilename, err)
		}
	default:
		return 0, fmt.Errorf("unsupported graphicsType for %q", img.sourceFilename)
	}

	if display && !noCrunch {
		wt, err = injectCrunch(wt)
		if err != nil {
			return 0, fmt.Errorf("injectCrunch failed: %w", err)
		}
		if !quiet {
			fmt.Println("packing with TSCrunch...")
		}
	}
	return wt.WriteTo(w)
}

func processFiles(filenames []string) (err error) {
	if len(filenames) < 1 {
		log.Println("no files supplied, nothing to do.")
		return nil
	}

	imgs, err := newSourceImages(filenames)
	switch {
	case err != nil:
		return fmt.Errorf("newSourceImages failed: %w", err)
	case len(imgs) == 0:
		return fmt.Errorf("no images found")
	case len(imgs) > 1:
		if err = handleAnimation(imgs); err != nil {
			return fmt.Errorf("handleAnimation failed: %w", err)
		}
		return nil
	}

	img := imgs[0]
	if verbose {
		log.Printf("processing file %q", img.sourceFilename)
	}
	if err = img.analyze(); err != nil {
		return fmt.Errorf("analyze %q failed: %w", img.sourceFilename, err)
	}

	var c io.WriterTo
	switch img.graphicsType {
	case multiColorBitmap:
		if c, err = img.convertToKoala(); err != nil {
			return fmt.Errorf("convertToKoala %q failed: %w", img.sourceFilename, err)
		}
	case singleColorBitmap:
		if c, err = img.convertToHires(); err != nil {
			return fmt.Errorf("convertToHires %q failed: %w", img.sourceFilename, err)
		}
	case singleColorCharset:
		if c, err = img.convertToSingleColorCharset(); err != nil {
			if graphicsMode != "" {
				return fmt.Errorf("convertToSingleColorCharset %q failed: %w", img.sourceFilename, err)
			}
			if !quiet {
				fmt.Printf("falling back to %s because convertToSingleColorCharset %q failed: %v\n", singleColorBitmap, img.sourceFilename, err)
			}
			img.graphicsType = singleColorBitmap
			if c, err = img.convertToHires(); err != nil {
				return fmt.Errorf("convertToHires %q failed: %w", img.sourceFilename, err)
			}
		}
	case multiColorCharset:
		if c, err = img.convertToMultiColorCharset(); err != nil {
			if graphicsMode != "" {
				return fmt.Errorf("convertToMultiColorCharset %q failed: %w", img.sourceFilename, err)
			}
			if !quiet {
				fmt.Printf("falling back to %s because convertToMultiColorCharset %q failed: %v\n", multiColorBitmap, img.sourceFilename, err)
			}
			img.graphicsType = multiColorBitmap
			err = img.findBackgroundColor()
			if err != nil {
				return fmt.Errorf("findBackgroundColor %q failed: %w", img.sourceFilename, err)
			}
			if c, err = img.convertToKoala(); err != nil {
				return fmt.Errorf("convertToKoala %q failed: %w", img.sourceFilename, err)
			}
		}
	case singleColorSprites:
		if c, err = img.convertToSingleColorSprites(); err != nil {
			return fmt.Errorf("convertToSingleColorSprites %q failed: %w", img.sourceFilename, err)
		}
	case multiColorSprites:
		if c, err = img.convertToMultiColorSprites(); err != nil {
			return fmt.Errorf("convertToMultiColorSprites %q failed: %w", img.sourceFilename, err)
		}
	default:
		return fmt.Errorf("unsupported graphicsType for %q", img.sourceFilename)
	}

	if display && !noCrunch {
		c, err = injectCrunch(c)
		if err != nil {
			return fmt.Errorf("injectCrunch failed: %w", err)
		}
		if !quiet {
			fmt.Println("packing with TSCrunch...")
		}
	}

	destFilename := destinationFilename(img.sourceFilename)
	f, err := os.Create(destFilename)
	if err != nil {
		return fmt.Errorf("os.Create %q failed: %w", destFilename, err)
	}
	defer f.Close()

	if _, err = c.WriteTo(f); err != nil {
		return fmt.Errorf("WriteTo %q failed: %w", destFilename, err)
	}
	if !quiet {
		fmt.Printf("converted %q to %q in %q format\n", img.sourceFilename, destFilename, img.graphicsType)
	}

	return nil
}

// injectCrunch drains the input io.WriterTo and returns a new TSCrunch WriterTo.
func injectCrunch(c io.WriterTo) (io.WriterTo, error) {
	buf := &bytes.Buffer{}
	if _, err := c.WriteTo(buf); err != nil {
		return nil, fmt.Errorf("WriteTo buffer failed: %w", err)
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
	c, err := TSCrunch.New(opt, buf)
	if err != nil {
		return nil, fmt.Errorf("tscrunch.New failed: %w", err)
	}
	return c, nil
}

// defaultHeader returns the startaddress of a file located at 0x2000.
func defaultHeader() []byte {
	return []byte{0x00, 0x20}
}

func newHeader(t GraphicsType) []byte {
	bin := make([]byte, len(displayers[t]))
	copy(bin, displayers[t])
	return bin
}

func zeroFill(s []byte, n int) []byte {
	return append(s, make([]byte, n)...)
}

func injectSIDHeader(header []byte, s *sid.SID) []byte {
	startSong := s.StartSong().LowByte()
	if startSong > 0 {
		startSong--
	}
	header[0x819-0x7ff] = startSong
	init := s.InitAddress()
	header[0x81b-0x7ff] = init.LowByte()
	header[0x81c-0x7ff] = init.HighByte()
	play := s.PlayAddress()
	header[0x81e-0x7ff] = play.LowByte()
	header[0x81f-0x7ff] = play.HighByte()
	return header
}

func (k Koala) WriteTo(w io.Writer) (n int64, err error) {
	bgBorder := k.BackgroundColor | k.BorderColor<<4
	if !display {
		return writeData(w, [][]byte{defaultHeader(), k.Bitmap[:], k.ScreenColor[:], k.D800Color[:], {bgBorder}})
	}
	header := newHeader(multiColorBitmap)
	if includeSID == "" {
		header = zeroFill(header, 0x2000-0x7ff-len(header))
		return writeData(w, [][]byte{header, k.Bitmap[:], k.ScreenColor[:], k.D800Color[:], {bgBorder}})
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
	case load > 0xcff && load < 0x1fff:
		header = zeroFill(header, int(load)-0x7ff-len(header))
		header = append(header, s.RawBytes()...)
		if len(header) > 0x2000-0x7ff {
			return 0, fmt.Errorf("sid memory overflow 0x%04x for sid %s", len(header)+0x7ff, s)
		}
		if !quiet {
			fmt.Printf("injected %q: %s\n", includeSID, s)
		}
		header = zeroFill(header, 0x2000-0x7ff-len(header))
		return writeData(w, [][]byte{header, k.Bitmap[:], k.ScreenColor[:], k.D800Color[:], {bgBorder}})
	case load < 0x8f00:
		return 0, fmt.Errorf("sid LoadAddress %s is causing memory overlap for sid %s", load, s)
	}

	header = zeroFill(header, 0x2000-0x7ff-len(header))
	buf := make([]byte, load-0x4711)
	n, err = writeData(w, [][]byte{header, k.Bitmap[:], k.ScreenColor[:], k.D800Color[:], {bgBorder}, buf, s.RawBytes()})
	if err != nil {
		return n, err
	}
	if !quiet {
		fmt.Printf("injected %q: %s\n", includeSID, s)
	}
	return n, nil
}

func (h Hires) WriteTo(w io.Writer) (n int64, err error) {
	if !display {
		return writeData(w, [][]byte{defaultHeader(), h.Bitmap[:], h.ScreenColor[:], {h.BorderColor}})
	}
	header := newHeader(singleColorBitmap)
	if includeSID == "" {
		header = zeroFill(header, 0x2000-0x7ff-len(header))
		return writeData(w, [][]byte{header, h.Bitmap[:], h.ScreenColor[:], {h.BorderColor}})
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
	case load > 0xcff && load < 0x1fff:
		header = zeroFill(header, int(load)-0x7ff-len(header))
		header = append(header, s.RawBytes()...)
		if len(header) > 0x2000-0x7ff {
			return 0, fmt.Errorf("sid memory overflow 0x%04x for sid %s", len(header)+0x7ff, s)
		}
		if !quiet {
			fmt.Printf("injected %q: %s\n", includeSID, s)
		}
		header = zeroFill(header, 0x2000-0x7ff-len(header))
		return writeData(w, [][]byte{header, h.Bitmap[:], h.ScreenColor[:], {h.BorderColor}})
	case load < 0x6c00:
		return 0, fmt.Errorf("sid LoadAddress %s is causing memory overlap for sid %s", load, s)
	}

	header = zeroFill(header, 0x2000-0x7ff-len(header))
	buf := make([]byte, load-0x4329)
	n, err = writeData(w, [][]byte{header, h.Bitmap[:], h.ScreenColor[:], {h.BorderColor}, buf, s.RawBytes()})
	if err != nil {
		return n, err
	}
	if !quiet {
		fmt.Printf("injected %q: %s\n", includeSID, s)
	}
	return n, nil
}

func (c MultiColorCharset) WriteTo(w io.Writer) (n int64, err error) {
	header := defaultHeader()
	if display {
		header = newHeader(multiColorCharset)
	}
	return writeData(w, [][]byte{header, c.Bitmap[:], c.Screen[:], {c.CharColor, c.BackgroundColor, c.D022Color, c.D023Color, c.BorderColor}})
}

func (c SingleColorCharset) WriteTo(w io.Writer) (n int64, err error) {
	header := defaultHeader()
	if display {
		header = newHeader(singleColorCharset)
	}
	return writeData(w, [][]byte{header, c.Bitmap[:], c.Screen[:], {c.CharColor, c.BackgroundColor}})
}

func (s SingleColorSprites) WriteTo(w io.Writer) (n int64, err error) {
	header := defaultHeader()
	if display {
		header = newHeader(singleColorSprites)
		header = append(header, s.Columns, s.Rows, s.BackgroundColor, s.SpriteColor)
	}
	return writeData(w, [][]byte{header, s.Bitmap[:]})
}

func (s MultiColorSprites) WriteTo(w io.Writer) (n int64, err error) {
	header := defaultHeader()
	if display {
		header = newHeader(multiColorSprites)
		header = append(header, s.Columns, s.Rows, s.BackgroundColor, s.D025Color, s.SpriteColor, s.D026Color)
	}
	return writeData(w, [][]byte{header, s.Bitmap[:]})
}

func writeData(w io.Writer, data [][]byte) (n int64, err error) {
	for _, d := range data {
		var m int
		m, err = w.Write(d)
		n += int64(m)
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func destinationFilename(filename string) (destfilename string) {
	if len(targetdir) > 0 {
		destfilename = filepath.Dir(targetdir+string(os.PathSeparator)) + string(os.PathSeparator)
	}
	if len(outfile) > 0 {
		return destfilename + outfile
	}
	return destfilename + filepath.Base(strings.TrimSuffix(filename, filepath.Ext(filename))+".prg")
}
