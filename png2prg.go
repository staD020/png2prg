package png2prg

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
	Version         = "1.3.10-dev"
	displayerJumpTo = "$0822"
	maxColors       = 16
)

var TSCOptions = TSCrunch.Options{
	PRG:     true,
	QUIET:   true,
	INPLACE: false,
	Fast:    true,
	JumpTo:  displayerJumpTo,
}

type RGB struct {
	R, G, B byte
}

func (r RGB) String() string {
	return fmt.Sprintf("RGB{0x%02x, 0x%02x, 0x%02x}", r.R, r.G, r.B)
}

type ColorInfo struct {
	ColorIndex byte
	RGB        RGB
}

func (c ColorInfo) String() string {
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

func StringToGraphicsType(s string) GraphicsType {
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
	reverse := [maxColors]*RGB{}
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
	reverse := [maxColors]*RGB{}
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
	opt                    Options
	image                  image.Image
	xOffset                int
	yOffset                int
	width                  int
	height                 int
	palette                PaletteMap
	colors                 []RGB
	charColors             [1000]PaletteMap
	backgroundCandidates   PaletteMap
	backgroundColor        ColorInfo
	borderColor            ColorInfo
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
	opt             Options
}

type Hires struct {
	SourceFilename string
	Bitmap         [8000]byte
	ScreenColor    [1000]byte
	BorderColor    byte
	opt            Options
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
	opt             Options
}

type SingleColorCharset struct {
	SourceFilename  string
	Bitmap          [0x800]byte
	Screen          [1000]byte
	CharColor       byte
	BackgroundColor byte
	BorderColor     byte
	opt             Options
}

type SingleColorSprites struct {
	SourceFilename  string
	Bitmap          []byte
	SpriteColor     byte
	BackgroundColor byte
	Columns         byte
	Rows            byte
	opt             Options
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
	opt             Options
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

type converter struct {
	opt    Options
	images []sourceImage
}

type Options struct {
	OutFile             string
	TargetDir           string
	Verbose             bool
	Quiet               bool
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
	ForceXOffset        int
	ForceYOffset        int
	CurrentGraphicsType GraphicsType
}

func New(opt Options, in ...io.Reader) (*converter, error) {
	if opt.ForceBorderColor > 15 {
		log.Printf("only values 0-15 are allowed, -force-border-color %d is not correct, now using default.", opt.ForceBorderColor)
		opt.ForceBorderColor = -1
	}

	imgs := []sourceImage{}
	for index, ir := range in {
		ii, err := NewSourceImages(opt, index, ir)
		if err != nil {
			return nil, fmt.Errorf("NewSourceImages failed: %w", err)
		}
		imgs = append(imgs, ii...)
	}
	return &converter{images: imgs, opt: opt}, nil
}

func NewSourceImages(opt Options, index int, r io.Reader) (imgs []sourceImage, err error) {
	path := fmt.Sprintf("png2prg_%02d", index)
	if n, isNamer := r.(interface{ Name() string }); isNamer {
		path = n.Name()
	}
	bin, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("io.ReadAll %q failed: %w", path, err)
	}

	// try gif first
	if g, err := gif.DecodeAll(bytes.NewReader(bin)); err == nil {
		if opt.Verbose {
			log.Printf("file %q has %d frames", path, len(g.Image))
		}
		for i, rawImage := range g.Image {
			img := sourceImage{
				sourceFilename: path,
				opt:            opt,
				image:          rawImage,
			}
			if err = img.setPreferredBitpairColors(opt.BitpairColorsString); err != nil {
				return nil, fmt.Errorf("setPreferredBitpairColors %q failed: %w", opt.BitpairColorsString, err)
			}
			if err = img.checkBounds(); err != nil {
				return nil, fmt.Errorf("img.checkBounds failed %q frame %d: %w", path, i, err)
			}
			imgs = append(imgs, img)
		}
		return imgs, nil
	}

	// should be png or jpg
	img := sourceImage{
		sourceFilename: path,
		opt:            opt,
	}
	if err = img.setPreferredBitpairColors(opt.BitpairColorsString); err != nil {
		return nil, fmt.Errorf("setPreferredBitpairColors %q failed: %w", opt.BitpairColorsString, err)
	}
	if img.image, _, err = image.Decode(bytes.NewReader(bin)); err != nil {
		return nil, fmt.Errorf("image.Decode failed: %w", err)
	}
	if err = img.checkBounds(); err != nil {
		return nil, fmt.Errorf("img.checkBounds failed: %w", err)
	}
	imgs = append(imgs, img)
	return imgs, nil
}

func NewFromPath(opt Options, filenames ...string) (*converter, error) {
	in := make([]io.Reader, 0, len(filenames))
	for _, path := range filenames {
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("os.Open failed: %w", err)
		}
		defer f.Close()
		in = append(in, f)
	}
	return New(opt, in...)
}

func (c *converter) WriteTo(w io.Writer) (n int64, err error) {
	if len(c.images) == 0 {
		return 0, fmt.Errorf("no images found")
	}
	if len(c.images) > 1 {
		return c.WriteAnimationTo(w)
	}

	img := c.images[0]
	if c.opt.Verbose {
		log.Printf("processing file %q", img.sourceFilename)
	}
	if err = img.analyze(); err != nil {
		return 0, fmt.Errorf("analyze %q failed: %w", img.sourceFilename, err)
	}

	var wt io.WriterTo
	switch img.graphicsType {
	case multiColorBitmap:
		if wt, err = img.Koala(); err != nil {
			return 0, fmt.Errorf("img.Koala %q failed: %w", img.sourceFilename, err)
		}
	case singleColorBitmap:
		if wt, err = img.Hires(); err != nil {
			return 0, fmt.Errorf("img.Hires %q failed: %w", img.sourceFilename, err)
		}
	case singleColorCharset:
		if wt, err = img.SingleColorCharset(); err != nil {
			if c.opt.GraphicsMode != "" {
				return 0, fmt.Errorf("img.SingleColorCharset %q failed: %w", img.sourceFilename, err)
			}
			if !c.opt.Quiet {
				fmt.Printf("falling back to %s because img.SingleColorCharset %q failed: %v\n", singleColorBitmap, img.sourceFilename, err)
			}
			img.graphicsType = singleColorBitmap
			if wt, err = img.Hires(); err != nil {
				return 0, fmt.Errorf("img.Hires %q failed: %w", img.sourceFilename, err)
			}
		}
	case multiColorCharset:
		if wt, err = img.MultiColorCharset(); err != nil {
			if c.opt.GraphicsMode != "" {
				return 0, fmt.Errorf("img.MultiColorCharset %q failed: %w", img.sourceFilename, err)
			}
			if !c.opt.Quiet {
				fmt.Printf("falling back to %s because img.MultiColorCharset %q failed: %v\n", multiColorBitmap, img.sourceFilename, err)
			}
			img.graphicsType = multiColorBitmap
			err = img.findBackgroundColor()
			if err != nil {
				return 0, fmt.Errorf("findBackgroundColor %q failed: %w", img.sourceFilename, err)
			}
			if wt, err = img.Koala(); err != nil {
				return 0, fmt.Errorf("img.Koala %q failed: %w", img.sourceFilename, err)
			}
		}
	case singleColorSprites:
		if wt, err = img.SingleColorSprites(); err != nil {
			return 0, fmt.Errorf("img.SingleColorSprites %q failed: %w", img.sourceFilename, err)
		}
	case multiColorSprites:
		if wt, err = img.MultiColorSprites(); err != nil {
			return 0, fmt.Errorf("img.MultiColorSprites %q failed: %w", img.sourceFilename, err)
		}
	default:
		return 0, fmt.Errorf("unsupported graphicsType for %q", img.sourceFilename)
	}

	if c.opt.Display && !c.opt.NoCrunch {
		wt, err = injectCrunch(wt, c.opt.Verbose)
		if err != nil {
			return 0, fmt.Errorf("injectCrunch failed: %w", err)
		}
		if !c.opt.Quiet {
			fmt.Println("packing with TSCrunch...")
		}
	}
	n, err = wt.WriteTo(w)
	if err != nil {
		return n, fmt.Errorf("WriteTo failed: %w", err)
	}
	return n, nil
}

// injectCrunch drains the input io.WriterTo and returns a new TSCrunch WriterTo.
func injectCrunch(c io.WriterTo, verbose bool) (io.WriterTo, error) {
	buf := &bytes.Buffer{}
	if _, err := c.WriteTo(buf); err != nil {
		return nil, fmt.Errorf("WriteTo buffer failed: %w", err)
	}
	opt := TSCOptions
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
	if !k.opt.Display {
		return writeData(w, defaultHeader(), k.Bitmap[:], k.ScreenColor[:], k.D800Color[:], []byte{bgBorder})
	}
	header := newHeader(multiColorBitmap)
	if k.opt.IncludeSID == "" {
		header = zeroFill(header, 0x2000-0x7ff-len(header))
		return writeData(w, header, k.Bitmap[:], k.ScreenColor[:], k.D800Color[:], []byte{bgBorder})
	}

	s, err := sid.LoadSID(k.opt.IncludeSID)
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
		if !k.opt.Quiet {
			fmt.Printf("injected %q: %s\n", k.opt.IncludeSID, s)
		}
		header = zeroFill(header, 0x2000-0x7ff-len(header))
		return writeData(w, header, k.Bitmap[:], k.ScreenColor[:], k.D800Color[:], []byte{bgBorder})
	case load < 0x8f00:
		return 0, fmt.Errorf("sid LoadAddress %s is causing memory overlap for sid %s", load, s)
	}

	header = zeroFill(header, 0x2000-0x7ff-len(header))
	buf := make([]byte, load-0x4711)
	n, err = writeData(w, header, k.Bitmap[:], k.ScreenColor[:], k.D800Color[:], []byte{bgBorder}, buf, s.RawBytes())
	if err != nil {
		return n, err
	}
	if !k.opt.Quiet {
		fmt.Printf("injected %q: %s\n", k.opt.IncludeSID, s)
	}
	return n, nil
}

func (h Hires) WriteTo(w io.Writer) (n int64, err error) {
	if !h.opt.Display {
		return writeData(w, defaultHeader(), h.Bitmap[:], h.ScreenColor[:], []byte{h.BorderColor})
	}
	header := newHeader(singleColorBitmap)
	if h.opt.IncludeSID == "" {
		header = zeroFill(header, 0x2000-0x7ff-len(header))
		return writeData(w, header, h.Bitmap[:], h.ScreenColor[:], []byte{h.BorderColor})
	}

	s, err := sid.LoadSID(h.opt.IncludeSID)
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
		if !h.opt.Quiet {
			fmt.Printf("injected %q: %s\n", h.opt.IncludeSID, s)
		}
		header = zeroFill(header, 0x2000-0x7ff-len(header))
		return writeData(w, header, h.Bitmap[:], h.ScreenColor[:], []byte{h.BorderColor})
	case load < 0x6c00:
		return 0, fmt.Errorf("sid LoadAddress %s is causing memory overlap for sid %s", load, s)
	}

	header = zeroFill(header, 0x2000-0x7ff-len(header))
	buf := make([]byte, load-0x4329)
	n, err = writeData(w, header, h.Bitmap[:], h.ScreenColor[:], []byte{h.BorderColor}, buf, s.RawBytes())
	if err != nil {
		return n, err
	}
	if !h.opt.Quiet {
		fmt.Printf("injected %q: %s\n", h.opt.IncludeSID, s)
	}
	return n, nil
}

func (c MultiColorCharset) WriteTo(w io.Writer) (n int64, err error) {
	header := defaultHeader()
	if c.opt.Display {
		header = newHeader(multiColorCharset)
	}
	return writeData(w, header, c.Bitmap[:], c.Screen[:], []byte{c.CharColor, c.BackgroundColor, c.D022Color, c.D023Color, c.BorderColor})
}

func (c SingleColorCharset) WriteTo(w io.Writer) (n int64, err error) {
	header := defaultHeader()
	if c.opt.Display {
		header = newHeader(singleColorCharset)
	}
	return writeData(w, header, c.Bitmap[:], c.Screen[:], []byte{c.CharColor, c.BackgroundColor})
}

func (s SingleColorSprites) WriteTo(w io.Writer) (n int64, err error) {
	header := defaultHeader()
	if s.opt.Display {
		header = newHeader(singleColorSprites)
		header = append(header, s.Columns, s.Rows, s.BackgroundColor, s.SpriteColor)
	}
	return writeData(w, header, s.Bitmap[:])
}

func (s MultiColorSprites) WriteTo(w io.Writer) (n int64, err error) {
	header := defaultHeader()
	if s.opt.Display {
		header = newHeader(multiColorSprites)
		header = append(header, s.Columns, s.Rows, s.BackgroundColor, s.D025Color, s.SpriteColor, s.D026Color)
	}
	return writeData(w, header, s.Bitmap[:])
}

func writeData(w io.Writer, data ...[]byte) (n int64, err error) {
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

func DestinationFilename(filename string, opt Options) (destfilename string) {
	if len(opt.TargetDir) > 0 {
		destfilename = filepath.Dir(opt.TargetDir+string(os.PathSeparator)) + string(os.PathSeparator)
	}
	if len(opt.OutFile) > 0 {
		return destfilename + opt.OutFile
	}
	return destfilename + filepath.Base(strings.TrimSuffix(filename, filepath.Ext(filename))+".prg")
}
