// Package png2prg provides png/gif/jpg to c64 .prg conversion.
// A single png2prg instance cannot be used concurrently, but each instance is standalone, many can be used in parallel.

package png2prg

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/staD020/TSCrunch"
	"github.com/staD020/sid"
)

const (
	Version              = "1.11.5-dev"
	MaxColors            = 16
	MaxChars             = 256
	MaxECMChars          = 64
	FullScreenChars      = 1000
	FullScreenWidth      = 320
	FullScreenHeight     = 200
	ViceFullScreenWidth  = 384
	ViceFullScreenHeight = 272
	SpriteWidth          = 24
	SpriteHeight         = 21

	BitmapAddress           = 0x2000
	BitmapScreenRAMAddress  = 0x3f40
	BitmapColorRAMAddress   = 0x4328
	CharsetScreenRAMAddress = 0x2800
	CharsetColorRAMAddress  = 0x2c00

	DisplayerSettingsStart = 0x081a
	displayerJumpTo        = "$0829"
)

// An Options struct contains all settings to be used for an instance of png2prg.
// The default empty/false settings are in general fine.
// You may want to set Quiet to suppress logging to stdout and Display to true if you want include the displayer.
type Options struct {
	OutFile              string
	TargetDir            string
	Verbose              bool
	VeryVerbose          bool
	Quiet                bool
	Display              bool
	BruteForce           bool
	NumWorkers           int
	NoPackChars          bool
	NoPackEmptyChar      bool
	ForcePackEmptyChar   bool
	NoPrevCharColors     bool
	NoBitpairCounters    bool
	NoCrunch             bool
	Symbols              bool
	NoFade               bool
	BitpairColorsString  string
	BitpairColorsString2 string
	BitpairColorsString3 string
	NoGuess              bool
	GraphicsMode         string
	Interlace            bool
	D016Offset           int
	ForceBorderColor     int
	IncludeSID           string
	NoAnimation          bool
	NoLoop               bool
	FrameDelay           byte
	WaitSeconds          int
	ForceXOffset         int
	ForceYOffset         int
	CurrentGraphicsType  GraphicsType

	Trd                           bool // has side effect of enforcing screenram colors in level area
	disableRepeatingBitpairColors bool // koala/hires animations should not want this optimization
}

func (o Options) NoFadeByte() byte {
	if o.NoFade {
		return 1
	}
	return 0
}

func (o Options) NoLoopByte() byte {
	if o.NoLoop {
		return 1
	}
	return 0
}

// A GraphicsType represents a supported c64 graphics type.
type GraphicsType byte

const (
	unknownGraphicsType GraphicsType = iota
	singleColorBitmap
	multiColorBitmap
	singleColorCharset
	multiColorCharset
	singleColorSprites
	multiColorSprites
	multiColorInterlaceBitmap // https://csdb.dk/release/?id=3961
	mixedCharset
	petsciiCharset
	ecmCharset
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
	case "mcibitmap":
		return multiColorInterlaceBitmap
	case "mixedcharset":
		return mixedCharset
	case "petscii":
		return petsciiCharset
	case "ecm":
		return ecmCharset
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
	case multiColorInterlaceBitmap:
		return "mcibitmap"
	case mixedCharset:
		return "mixed charset"
	case petsciiCharset:
		return "petscii"
	case ecmCharset:
		return "ecm"
	default:
		return "unknown"
	}
}

type sourceImage struct {
	sourceFilename  string
	opt             Options
	image           image.Image
	hiresPixels     bool
	xOffset         int
	yOffset         int
	width           int
	height          int
	graphicsType    GraphicsType
	p               Palette
	bpc             BPColors
	bpc2            BPColors
	bpc3            BPColors
	bpcCache        [FullScreenChars]map[C64Color]byte
	bpcBitpairCount [MaxColors]map[byte]int
	border          Color
	bg              Color
	bgCandidates    Colors
	charColors      [FullScreenChars]Colors
	sumColors       [MaxColors]int
	ecmColors       Colors
}

func (img *sourceImage) At(x, y int) color.Color {
	r, g, b, _ := img.image.At(img.xOffset+x, img.yOffset+y).RGBA()
	return color.RGBA{byte(r), byte(g), byte(b), 0xff}
}

func (img *sourceImage) ColorModel() color.Model {
	return img.p
}

func (img *sourceImage) Bounds() image.Rectangle {
	return img.image.Bounds()
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
	D800Color   byte
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

type charBytes [8]byte

func (cb charBytes) Invert() (b charBytes) {
	for i := range cb {
		b[i] = cb[i] ^ 0xff
	}
	return b
}

func (cb charBytes) String() (s string) {
	for _, b := range cb {
		s += fmt.Sprintf("%08b\n", b)
	}
	return s
}

type c64Symbol struct {
	key   string
	value int
}

type Symbolser interface {
	Symbols() []c64Symbol
}

func (img Koala) Symbols() []c64Symbol {
	return []c64Symbol{
		{"bitmap", BitmapAddress},
		{"screenram", BitmapScreenRAMAddress},
		{"colorram", BitmapColorRAMAddress},
		{"d020color", int(img.BorderColor)},
		{"d021color", int(img.BackgroundColor)},
	}
}

type Hires struct {
	SourceFilename string
	Bitmap         [8000]byte
	ScreenColor    [1000]byte
	BorderColor    byte
	opt            Options
}

func (img Hires) Symbols() []c64Symbol {
	return []c64Symbol{
		{"bitmap", BitmapAddress},
		{"screenram", BitmapScreenRAMAddress},
		{"d020color", int(img.BorderColor)},
	}
}

type MultiColorCharset struct {
	SourceFilename  string
	Bitmap          [0x800]byte
	Screen          [1000]byte
	D800Color       [1000]byte
	CharColor       byte
	BorderColor     byte
	BackgroundColor byte
	D022Color       byte
	D023Color       byte
	opt             Options
}

func (img MultiColorCharset) Symbols() []c64Symbol {
	return []c64Symbol{
		{"bitmap", BitmapAddress},
		{"screenram", CharsetScreenRAMAddress},
		{"charcolor", int(img.CharColor)},
		{"d020color", int(img.BorderColor)},
		{"d021color", int(img.BackgroundColor)},
		{"d022color", int(img.D022Color)},
		{"d023color", int(img.D023Color)},
	}
}

func (c MultiColorCharset) UsedChars() int {
	max := byte(0)
	for _, v := range c.Screen {
		if v > max {
			max = v
		}
	}
	// check for empty chars too, this is for animations
	empty := charBytes{}
	emptyCount := 0
	for i := 0; i < MaxChars; i++ {
		cb := charBytes{}
		for j := 0; j < 8; j++ {
			cb[j] = c.Bitmap[i*8+j]
		}
		if cb == empty {
			emptyCount++
			if emptyCount > 1 && i > int(max) {
				return i
			}
		}
	}
	return (int(max) + 1)
}

func (c MultiColorCharset) CharBytes() (cbs []charBytes) {
	used := c.UsedChars()
	for i := 0; i < used; i++ {
		cb := charBytes{}
		for j := 0; j < 8; j++ {
			cb[j] = c.Bitmap[(i*8)+j]
		}
		cbs = append(cbs, cb)
	}
	return cbs
}

type SingleColorCharset struct {
	SourceFilename  string
	Bitmap          [0x800]byte
	Screen          [1000]byte
	D800Color       [1000]byte
	BackgroundColor byte
	BorderColor     byte
	opt             Options
}

func (img SingleColorCharset) Symbols() []c64Symbol {
	return []c64Symbol{
		{"bitmap", BitmapAddress},
		{"screenram", CharsetScreenRAMAddress},
		{"colorram", CharsetColorRAMAddress},
		{"d020color", int(img.BorderColor)},
		{"d021color", int(img.BackgroundColor)},
	}
}

func (c SingleColorCharset) UsedChars() int {
	max := byte(0)
	for _, v := range c.Screen {
		if v > max {
			max = v
		}
	}
	// check for empty chars too, this is for animations
	empty := charBytes{}
	emptyCount := 0
	for i := 0; i < MaxChars; i++ {
		cb := charBytes{}
		for j := 0; j < 8; j++ {
			cb[j] = c.Bitmap[i*8+j]
		}
		if cb == empty {
			emptyCount++
			if emptyCount > 1 && i > int(max) {
				return i
			}
		}
	}
	return (int(max) + 1)
}

func (c SingleColorCharset) CharBytes() (cbs []charBytes) {
	used := c.UsedChars()
	for i := 0; i < used; i++ {
		cb := charBytes{}
		for j := 0; j < 8; j++ {
			cb[j] = c.Bitmap[(i*8)+j]
		}
		cbs = append(cbs, cb)
	}
	return cbs
}

type MixedCharset struct {
	SourceFilename  string
	Bitmap          [0x800]byte
	Screen          [1000]byte
	D800Color       [1000]byte
	BorderColor     byte
	BackgroundColor byte
	D022Color       byte
	D023Color       byte
	opt             Options
}

func (img MixedCharset) Symbols() []c64Symbol {
	return []c64Symbol{
		{"bitmap", BitmapAddress},
		{"screenram", CharsetScreenRAMAddress},
		{"colorram", CharsetColorRAMAddress},
		{"d020color", int(img.BorderColor)},
		{"d021color", int(img.BackgroundColor)},
		{"d022color", int(img.D022Color)},
		{"d023color", int(img.D023Color)},
	}
}

func (c MixedCharset) UsedChars() int {
	max := byte(0)
	for _, v := range c.Screen {
		if v > max {
			max = v
		}
	}
	// check for empty chars too, this is for animations
	empty := charBytes{}
	emptyCount := 0
	for i := 0; i < MaxChars; i++ {
		cb := charBytes{}
		for j := 0; j < 8; j++ {
			cb[j] = c.Bitmap[i*8+j]
		}
		if cb == empty {
			emptyCount++
			if emptyCount > 1 && i > int(max) {
				return i
			}
		}
	}
	return (int(max) + 1)
}

func (c MixedCharset) CharBytes() (cbs []charBytes) {
	used := c.UsedChars()
	for i := 0; i < used; i++ {
		cb := charBytes{}
		for j := 0; j < 8; j++ {
			cb[j] = c.Bitmap[(i*8)+j]
		}
		cbs = append(cbs, cb)
	}
	return cbs
}

type PETSCIICharset struct {
	SourceFilename  string
	Lowercase       byte // 0 = uppercase, 1 = lowercase
	Screen          [1000]byte
	D800Color       [1000]byte
	BackgroundColor byte
	BorderColor     byte
	opt             Options
}

func (img PETSCIICharset) Symbols() []c64Symbol {
	return []c64Symbol{
		{"screenram", CharsetScreenRAMAddress},
		{"colorram", CharsetColorRAMAddress},
		{"d020color", int(img.BorderColor)},
		{"d021color", int(img.BackgroundColor)},
	}
}

type ECMCharset struct {
	SourceFilename  string
	Bitmap          [MaxECMChars * 8]byte
	Screen          [1000]byte
	D800Color       [1000]byte
	BorderColor     byte
	BackgroundColor byte
	D022Color       byte
	D023Color       byte
	D024Color       byte
	opt             Options
}

func (img ECMCharset) Symbols() []c64Symbol {
	return []c64Symbol{
		{"bitmap", BitmapAddress},
		{"screenram", CharsetScreenRAMAddress},
		{"colorram", CharsetColorRAMAddress},
		{"d020color", int(img.BorderColor)},
		{"d021color", int(img.BackgroundColor)},
		{"d022color", int(img.D022Color)},
		{"d023color", int(img.D023Color)},
		{"d024color", int(img.D024Color)},
	}
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

func (img SingleColorSprites) Symbols() []c64Symbol {
	return []c64Symbol{
		{"bitmap", BitmapAddress},
		{"columns", int(img.Columns)},
		{"rows", int(img.Rows)},
		{"spritecolor", int(img.SpriteColor)},
		{"d021color", int(img.BackgroundColor)},
	}
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

func (img MultiColorSprites) Symbols() []c64Symbol {
	return []c64Symbol{
		{"bitmap", BitmapAddress},
		{"columns", int(img.Columns)},
		{"rows", int(img.Rows)},
		{"spritecolor", int(img.SpriteColor)},
		{"d021color", int(img.BackgroundColor)},
		{"d025color", int(img.D025Color)},
		{"d026color", int(img.D026Color)},
	}
}

var displayers = make(map[GraphicsType][]byte, 0)

//go:embed "display_koala.prg"
var koalaDisplay []byte

//go:embed "display_hires.prg"
var hiresDisplay []byte

//go:embed "display_mc_charset.prg"
var mcCharsetDisplay []byte

//go:embed "display_mc_charset_anim.prg"
var mcCharsetDisplayAnim []byte

//go:embed "display_mc_charset_multi.prg"
var mcCharsetDisplayMulti []byte

//go:embed "display_sc_charset.prg"
var scCharsetDisplay []byte

//go:embed "display_sc_charset_anim.prg"
var scCharsetDisplayAnim []byte

//go:embed "display_sc_charset_multi.prg"
var scCharsetDisplayMulti []byte

//go:embed "display_mc_sprites.prg"
var mcSpritesDisplay []byte

//go:embed "display_sc_sprites.prg"
var scSpritesDisplay []byte

//go:embed "display_koala_anim.prg"
var koalaDisplayAnim []byte

//go:embed "display_hires_anim.prg"
var hiresDisplayAnim []byte

//go:embed "display_mci_bitmap.prg"
var mciBitmapDisplay []byte

//go:embed "display_mixed_charset.prg"
var mixedCharsetDisplay []byte

//go:embed "display_petscii_charset.prg"
var petsciiCharsetDisplay []byte

//go:embed "display_petscii_charset_anim.prg"
var petsciiCharsetDisplayAnim []byte

//go:embed "display_ecm_charset.prg"
var ecmCharsetDisplay []byte

//go:embed "tools/rom_charset_lowercase.prg"
var romCharsetLowercasePrg []byte

//go:embed "tools/rom_charset_uppercase.prg"
var romCharsetUppercasePrg []byte

func init() {
	displayers[multiColorBitmap] = koalaDisplay
	displayers[singleColorBitmap] = hiresDisplay
	displayers[multiColorCharset] = mcCharsetDisplay
	displayers[singleColorCharset] = scCharsetDisplay
	displayers[multiColorSprites] = mcSpritesDisplay
	displayers[singleColorSprites] = scSpritesDisplay
	displayers[multiColorInterlaceBitmap] = mciBitmapDisplay
	displayers[mixedCharset] = mixedCharsetDisplay
	displayers[petsciiCharset] = petsciiCharsetDisplay
	displayers[ecmCharset] = ecmCharsetDisplay
}

// newHeader returns a copy of the displayer code for GraphicsType t as a byte slice in .prg format.
func (t GraphicsType) newHeader() []byte {
	bin := make([]byte, len(displayers[t]))
	copy(bin, displayers[t])
	return bin
}

// A Converter implements the io.WriterTo interface.
type Converter struct {
	opt               Options
	images            []sourceImage
	AnimItems         []AnimItem
	Symbols           []c64Symbol
	FinalGraphicsType GraphicsType
}

// New processes the input images or anim.csv and returns the Converter.
// Returns an error if any of the images have non-supported dimensions.
// Generally a single image is used as input. For animations an anim.csv, animated gif or multiple .pngs will do the trick.
//
// The returned Converter implements the io.WriterTo interface.
func New(opt Options, pngs ...io.Reader) (*Converter, error) {
	if opt.ForceBorderColor > 15 {
		log.Printf("-force-border-color %d is not correct, only values 0-15 are allowed, now using default.", opt.ForceBorderColor)
		opt.ForceBorderColor = -1
	}
	if opt.GraphicsMode != "" && opt.CurrentGraphicsType == unknownGraphicsType {
		opt.CurrentGraphicsType = StringToGraphicsType(opt.GraphicsMode)
	}
	c := &Converter{opt: opt}
	if len(pngs) == 1 {
		bin, err := io.ReadAll(pngs[0])
		if err != nil {
			return nil, fmt.Errorf("io.ReadAll failed: %w", err)
		}
		c.AnimItems, err = ExtractAnimationCSV(bytes.NewReader(bin))
		switch {
		case err == nil:
			pngs = []io.Reader{}
			for _, ai := range c.AnimItems {
				f, err := os.Open(ai.Filename)
				if err != nil {
					return nil, fmt.Errorf("os.Open %q failed: %w", ai.Filename, err)
				}
				defer f.Close()
				pngs = append(pngs, f)
			}
		case errors.Is(err, ErrInvalidFrameDelay):
			return nil, fmt.Errorf("ExtractAnimationCSV failed: %w", err)
		case errors.Is(err, ErrInvalidCSV):
			pngs[0] = bytes.NewReader(bin)
		}
	}
	for index, ir := range pngs {
		ii, err := NewSourceImages(opt, index, ir)
		if err != nil {
			return nil, fmt.Errorf("NewSourceImages failed: %w", err)
		}
		c.images = append(c.images, ii...)
	}
	return c, nil
}

// NewSourceImages decodes r into one or more sourceImages and returns them.
// Also validates the resolution of the images.
// Generally imgs contain 1 image, unless an animated .gif was supplied in r.
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
			if opt.VeryVerbose {
				log.Printf("processing frame %d", i)
			}
			img := sourceImage{
				sourceFilename: path,
				opt:            opt,
				image:          rawImage,
			}
			switch {
			case i == 0:
				if err = img.checkBounds(); err != nil {
					return nil, fmt.Errorf("img.checkBounds failed %q frame %d: %w", path, i, err)
				}
			case i > 0:
				img.xOffset, img.yOffset = imgs[0].xOffset, imgs[0].yOffset
				img.width, img.height = imgs[0].width, imgs[0].height
			}
			img.p, img.hiresPixels, err = NewPalette(&img, false, opt.Verbose)
			if err != nil {
				return nil, fmt.Errorf("NewPalette failed: %w", err)
			}
			if err = img.setPreferredBitpairColors(); err != nil {
				return nil, fmt.Errorf("setPreferredBitpairColors -bpc %q -bpc2 %q failed: %w", opt.BitpairColorsString, opt.BitpairColorsString2, err)
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
	if img.image, _, err = image.Decode(bytes.NewReader(bin)); err != nil {
		return nil, fmt.Errorf("image.Decode failed: %w", err)
	}
	if err = img.checkBounds(); err != nil {
		return nil, fmt.Errorf("img.checkBounds failed: %w", err)
	}
	img.p, img.hiresPixels, err = NewPalette(&img, false, opt.Verbose)
	if err != nil {
		return nil, fmt.Errorf("NewPalette failed: %w", err)
	}
	if err = img.setPreferredBitpairColors(); err != nil {
		return nil, fmt.Errorf("setPreferredBitpairColors -bpc %q -bpc2 %q failed: %w", opt.BitpairColorsString, opt.BitpairColorsString2, err)
	}
	imgs = append(imgs, img)
	return imgs, nil
}

// NewSourceImage returns a new sourceImage after bounds check.
func NewSourceImage(opt Options, index int, in image.Image) (img sourceImage, err error) {
	img = sourceImage{
		sourceFilename: fmt.Sprintf("png2prg_%02d", index),
		opt:            opt,
		image:          in,
	}
	if err = img.checkBounds(); err != nil {
		return img, fmt.Errorf("img.checkBounds failed: %w", err)
	}
	if img.p, img.hiresPixels, err = NewPalette(&img, false, opt.Verbose); err != nil {
		return img, fmt.Errorf("NewPalette failed: %w", err)
	}
	if err = img.setPreferredBitpairColors(); err != nil {
		return img, fmt.Errorf("setPreferredBitpairColors -bpc %q -bpc2 %q failed: %w", opt.BitpairColorsString, opt.BitpairColorsString2, err)
	}
	return img, nil
}

// NewFromPath is the convenience New method when input images are on disk.
// See New for detais.
func NewFromPath(opt Options, filenames ...string) (*Converter, error) {
	in := []io.Reader{}
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

// WriteTo processes the image(s) and writes the resulting .prg to w.
// Returns error when analysis or conversion fails.
func (c *Converter) WriteTo(w io.Writer) (n int64, err error) {
	if len(c.images) == 0 {
		return 0, fmt.Errorf("no images found")
	}
	img := &c.images[0]
	if c.opt.Verbose {
		log.Printf("processing file %q", img.sourceFilename)
	}
	defer func() {
		if len(c.images) == 1 {
			c.FinalGraphicsType = img.graphicsType
		}
	}()
	if err = img.analyze(); err != nil {
		return 0, fmt.Errorf("analyze %q failed: %w", img.sourceFilename, err)
	}

	if (len(c.images) == 1 && img.graphicsType == multiColorInterlaceBitmap) || (len(c.images) == 2 && c.opt.Interlace) {
		if !c.opt.Quiet {
			fmt.Printf("interlace mode\n")
		}
		var rgba0, rgba1 *image.RGBA
		if img.graphicsType == multiColorInterlaceBitmap {
			rgba0, rgba1 = img.SplitInterlace()
			c.opt.ForceBorderColor = int(img.border.C64Color)
			if !c.opt.Quiet {
				fmt.Println("interlaced pic was split")
			}
			c.opt.CurrentGraphicsType = multiColorBitmap
			c.opt.GraphicsMode = multiColorBitmap.String()

			i0, err := NewSourceImage(c.opt, 0, rgba0)
			if err != nil {
				return n, fmt.Errorf("NewSourceImage %q failed: %w", img.sourceFilename, err)
			}
			i1, err := NewSourceImage(c.opt, 1, rgba1)
			if err != nil {
				return n, fmt.Errorf("NewSourceImage %q failed: %w", img.sourceFilename, err)
			}
			c.images = []sourceImage{i0, i1}
		}

		if err = c.images[0].analyze(); err != nil {
			return n, fmt.Errorf("analyze %q failed: %w", c.images[0].sourceFilename, err)
		}
		if err = c.images[1].analyze(); err != nil {
			return n, fmt.Errorf("analyze %q failed: %w", c.images[1].sourceFilename, err)
		}
		c.FinalGraphicsType = img.graphicsType
		return c.WriteInterlaceTo(w)
	}
	if len(c.images) > 1 {
		return c.WriteAnimationTo(w)
	}

	bruteforce := func(gfxtype GraphicsType, maxColors int) error {
		if !c.opt.BruteForce {
			return nil
		}
		if err = c.BruteForceBitpairColors(gfxtype, maxColors); err != nil {
			return fmt.Errorf("BruteForceBitpairColors %q failed: %w", img.sourceFilename, err)
		}
		if err = img.setPreferredBitpairColors(); err != nil {
			return fmt.Errorf("setPreferredBitpairColors -bpc %q -bpc2 %q failed: %w", c.opt.BitpairColorsString, c.opt.BitpairColorsString2, err)
		}
		return nil
	}

	var wt io.WriterTo
	switch img.graphicsType {
	case multiColorBitmap:
		if err = bruteforce(img.graphicsType, 4); err != nil {
			return 0, err
		}
		if wt, err = img.Koala(); err != nil {
			return 0, fmt.Errorf("img.Koala %q failed: %w", img.sourceFilename, err)
		}
	case singleColorBitmap:
		if err = bruteforce(img.graphicsType, 2); err != nil {
			return 0, err
		}
		if wt, err = img.Hires(); err != nil {
			return 0, fmt.Errorf("img.Hires %q failed: %w", img.sourceFilename, err)
		}
	case singleColorCharset:
		if c.opt.GraphicsMode != "" {
			if err = bruteforce(img.graphicsType, 1); err != nil {
				return 0, err
			}
			if wt, err = img.SingleColorCharset(nil); err != nil {
				return 0, fmt.Errorf("img.SingleColorCharset %q failed: %w", img.sourceFilename, err)
			}
		} else {
			if wt, err = img.PETSCIICharset(); err != nil {
				if err = bruteforce(img.graphicsType, 1); err != nil {
					return 0, err
				}
				if wt, err = img.SingleColorCharset(nil); err != nil {
					fmt.Printf("falling back to %s because img.SingleColorCharset %q failed: %v\n", singleColorBitmap, img.sourceFilename, err)
					img.graphicsType = singleColorBitmap
					if err = bruteforce(singleColorBitmap, 2); err != nil {
						return 0, err
					}
					if wt, err = img.Hires(); err != nil {
						return 0, fmt.Errorf("img.Hires %q failed: %w", img.sourceFilename, err)
					}
				}
			} else if !c.opt.Quiet {
				fmt.Printf("detected petscii\n")
				img.graphicsType = petsciiCharset
			}
		}
	case petsciiCharset:
		if wt, err = img.PETSCIICharset(); err != nil {
			return 0, fmt.Errorf("img.PETSCIICharset %q failed: %w", img.sourceFilename, err)
		}
	case ecmCharset:
		if err = bruteforce(img.graphicsType, 4); err != nil {
			fmt.Printf("falling back to %s because bruteforce %q failed: %v\n", singleColorBitmap, img.sourceFilename, err)
			img.graphicsType = singleColorBitmap
			if err = bruteforce(img.graphicsType, 2); err != nil {
				return 0, err
			}
			if wt, err = img.Hires(); err != nil {
				return 0, fmt.Errorf("img.Hires %q failed: %w", img.sourceFilename, err)
			}
		}
		if wt, err = img.ECMCharset(nil); err != nil {
			if c.opt.GraphicsMode != "" {
				return 0, fmt.Errorf("img.ECMCharset %q failed: %w", img.sourceFilename, err)
			}
			fmt.Printf("falling back to %s because img.ECMCharset %q failed: %v\n", singleColorBitmap, img.sourceFilename, err)
			img.graphicsType = singleColorBitmap
			if err = bruteforce(img.graphicsType, 2); err != nil {
				return 0, err
			}
			if wt, err = img.Hires(); err != nil {
				return 0, fmt.Errorf("img.Hires %q failed: %w", img.sourceFilename, err)
			}
		}
	case multiColorCharset:
		if err = bruteforce(img.graphicsType, 4); err != nil {
			if c.opt.GraphicsMode != "" {
				return 0, fmt.Errorf("img.MultiColorCharset %q failed: %w", img.sourceFilename, err)
			}
			fmt.Printf("falling back to %s because bruteforce %q failed: %v\n", multiColorBitmap, img.sourceFilename, err)
			img.graphicsType = multiColorBitmap
			err = img.findBackgroundColor()
			if err != nil {
				return 0, fmt.Errorf("findBackgroundColor %q failed: %w", img.sourceFilename, err)
			}
			if err = bruteforce(img.graphicsType, 4); err != nil {
				return 0, err
			}
			if wt, err = img.Koala(); err != nil {
				return 0, fmt.Errorf("img.Koala %q failed: %w", img.sourceFilename, err)
			}
		}
		if wt, err = img.MultiColorCharset(nil); err != nil {
			if c.opt.GraphicsMode != "" {
				return 0, fmt.Errorf("img.MultiColorCharset %q failed: %w", img.sourceFilename, err)
			}
			fmt.Printf("falling back to %s because img.MultiColorCharset %q failed: %v\n", multiColorBitmap, img.sourceFilename, err)
			img.graphicsType = multiColorBitmap
			err = img.findBackgroundColor()
			if err != nil {
				return 0, fmt.Errorf("findBackgroundColor %q failed: %w", img.sourceFilename, err)
			}
			if err = bruteforce(img.graphicsType, 4); err != nil {
				return 0, err
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
	case mixedCharset:
		if err = bruteforce(img.graphicsType, 4); err != nil {
			if c.opt.GraphicsMode != "" {
				return 0, fmt.Errorf("img.MixedCharset %q failed: %w", img.sourceFilename, err)
			}
			fmt.Printf("falling back to %s because bruteforce %s for %q failed: %v\n", multiColorBitmap, mixedCharset, img.sourceFilename, err)
			img.graphicsType = multiColorBitmap
			img.findBgCandidates(false)
			if err = img.findBackgroundColor(); err != nil {
				return 0, fmt.Errorf("img.findBackgroundColor %q failed: %w", img.sourceFilename, err)
			}
			if err = bruteforce(img.graphicsType, 4); err != nil {
				return 0, err
			}
			if wt, err = img.Koala(); err != nil {
				return 0, fmt.Errorf("img.Koala %q failed: %w", img.sourceFilename, err)
			}
		}
		if wt, err = img.MixedCharset(nil); err != nil {
			if c.opt.GraphicsMode != "" {
				return 0, fmt.Errorf("img.MixedCharset %q failed: %w", img.sourceFilename, err)
			}
			fmt.Printf("falling back to %s because %s for %q failed: %v\n", multiColorBitmap, mixedCharset, img.sourceFilename, err)
			img.graphicsType = multiColorBitmap
			img.findBgCandidates(false)
			if err = img.findBackgroundColor(); err != nil {
				return 0, fmt.Errorf("img.findBackgroundColor %q failed: %w", img.sourceFilename, err)
			}
			if err = bruteforce(img.graphicsType, 4); err != nil {
				return 0, err
			}
			if wt, err = img.Koala(); err != nil {
				return 0, fmt.Errorf("img.Koala %q failed: %w", img.sourceFilename, err)
			}
		}
	default:
		return 0, fmt.Errorf("unsupported graphicsType %q for %q", img.graphicsType, img.sourceFilename)
	}

	if c.opt.Symbols {
		if s, ok := wt.(Symbolser); ok {
			c.Symbols = append(c.Symbols, s.Symbols()...)
		}
		if len(c.Symbols) == 0 {
			return 0, fmt.Errorf("symbols not supported %T for %q", wt, img.sourceFilename)
		}
	}

	t1 := time.Now()
	if c.opt.Display && !c.opt.NoCrunch {
		wt, err = injectCrunch(wt, c.opt.Verbose)
		if err != nil {
			return 0, fmt.Errorf("injectCrunch failed: %w", err)
		}
	}
	n, err = wt.WriteTo(w)
	if err != nil {
		return n, fmt.Errorf("WriteTo failed: %w", err)
	}
	if !c.opt.Quiet && c.opt.Display && !c.opt.NoCrunch {
		fmt.Printf("TSCrunched in %s\n", time.Since(t1))
	}
	return n, nil
}

// WriteSymbolsTo writes c.Symbols to w in text format.
func (c *Converter) WriteSymbolsTo(w io.Writer) (n int64, err error) {
	for _, s := range c.Symbols {
		n2 := 0
		if s.value < 16 {
			n2, err = fmt.Fprintf(w, "%s = %d\n", s.key, s.value)
		} else {
			n2, err = fmt.Fprintf(w, "%s = $%x\n", s.key, s.value)
		}
		n += int64(n2)
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

var TSCOptions = TSCrunch.Options{
	PRG:    true,
	QUIET:  true,
	Fast:   true,
	JumpTo: displayerJumpTo,
}

// injectCrunch drains the input io.WriterTo and returns a new TSCrunch WriterTo.
func injectCrunch(c io.WriterTo, verbose bool) (io.WriterTo, error) {
	buf := &bytes.Buffer{}
	if _, err := c.WriteTo(buf); err != nil {
		return nil, fmt.Errorf("WriteTo buffer failed: %w", err)
	}
	c, err := TSCrunch.New(TSCOptions, buf)
	if err != nil {
		return nil, fmt.Errorf("tscrunch.New failed: %w", err)
	}
	return c, nil
}

// defaultHeader returns the startaddress of a file located at BitmapAddress.
// .prg format essentially.
func defaultHeader() []byte {
	return []byte{BitmapAddress & 0xff, BitmapAddress >> 8}
}

// injectSID injects the sid, it's start song and init/play addresses in predefined locations in the linker.
// If sidFilename is empty, nothing happens and a nil error is returned.
// Must be called *after* displayer code is linked.
func injectSID(l *Linker, sidFilename string, quiet bool) error {
	if sidFilename == "" {
		return nil
	}
	s, err := sid.LoadSID(sidFilename)
	if err != nil {
		return fmt.Errorf("sid.LoadSID failed: %w", err)
	}
	if _, err = l.WritePrg(s.Bytes()); err != nil {
		return fmt.Errorf("link.WritePrg failed: %w", err)
	}
	startSong := s.StartSong().LowByte()
	if startSong > 0 {
		startSong--
	}
	palntsc := byte(0)
	if s.NTSC() {
		// set bit 7 of startsong as ntsc flag
		palntsc = 0x80
	}
	l.SetByte(DisplayerSettingsStart, (startSong&0x7f)|palntsc)
	init := s.InitAddress()
	l.SetByte(DisplayerSettingsStart+2, init.LowByte(), init.HighByte())
	play := s.PlayAddress()
	l.SetByte(DisplayerSettingsStart+5, play.LowByte(), play.HighByte())
	if !quiet {
		fmt.Printf("injected %q: %s\n", sidFilename, s)
	}
	return nil
}

func (k Koala) WriteTo(w io.Writer) (n int64, err error) {
	bgBorder := k.BackgroundColor | k.BorderColor<<4
	link := NewLinker(BitmapAddress, k.opt.VeryVerbose)
	_, err = link.WriteMap(LinkMap{
		BitmapAddress: k.Bitmap[:],
		0x3f40:        k.ScreenColor[:],
		0x4328:        k.D800Color[:],
		0x4710:        []byte{bgBorder},
	})
	if err != nil {
		return n, fmt.Errorf("link.WriteMap failed: %w", err)
	}
	if !k.opt.Display {
		return link.WriteTo(w)
	}

	if _, err = link.WritePrg(multiColorBitmap.newHeader()); err != nil {
		return n, fmt.Errorf("link.WritePrg failed: %w", err)
	}
	link.SetByte(DisplayerSettingsStart+9, k.opt.NoFadeByte(), k.opt.NoLoopByte())
	if !k.opt.NoFade {
		link.Block(0x4800, 0x8e50)
	}
	if err = injectSID(link, k.opt.IncludeSID, k.opt.Quiet); err != nil {
		return n, fmt.Errorf("injectSID failed: %w", err)
	}
	return link.WriteTo(w)
}

func (h Hires) WriteTo(w io.Writer) (n int64, err error) {
	link := NewLinker(BitmapAddress, h.opt.VeryVerbose)
	_, err = link.WriteMap(LinkMap{
		BitmapAddress: h.Bitmap[:],
		0x3f40:        h.ScreenColor[:],
		0x4328:        []byte{h.BorderColor},
	})
	if err != nil {
		return n, fmt.Errorf("link.WriteMap failed: %w", err)
	}
	if !h.opt.Display {
		return link.WriteTo(w)
	}

	if _, err = link.WritePrg(singleColorBitmap.newHeader()); err != nil {
		return n, fmt.Errorf("link.WritePrg failed: %w", err)
	}
	link.SetByte(DisplayerSettingsStart+9, h.opt.NoFadeByte(), h.opt.NoLoopByte())
	if !h.opt.NoFade {
		link.Block(0x4800, 0x6b29)
	}
	if err = injectSID(link, h.opt.IncludeSID, h.opt.Quiet); err != nil {
		return n, fmt.Errorf("injectSID failed: %w", err)
	}
	return link.WriteTo(w)
}

func (c MultiColorCharset) WriteTo(w io.Writer) (n int64, err error) {
	link := NewLinker(BitmapAddress, c.opt.VeryVerbose)
	_, err = link.WriteMap(LinkMap{
		BitmapAddress:           c.Bitmap[:],
		CharsetScreenRAMAddress: c.Screen[:],
		CharsetColorRAMAddress:  c.D800Color[:],
		0x2fe8:                  []byte{c.BorderColor, c.BackgroundColor, c.D022Color, c.D023Color},
	})
	if err != nil {
		return n, fmt.Errorf("link.WriteMap failed: %w", err)
	}
	if !c.opt.Display {
		return link.WriteTo(w)
	}
	if _, err = link.WritePrg(mixedCharset.newHeader()); err != nil {
		return n, fmt.Errorf("link.WritePrg failed: %w", err)
	}
	link.SetByte(DisplayerSettingsStart+10, c.opt.NoFadeByte(), c.opt.NoLoopByte())
	if err = injectSID(link, c.opt.IncludeSID, c.opt.Quiet); err != nil {
		return n, fmt.Errorf("injectSID failed: %w", err)
	}
	return link.WriteTo(w)
}

func (c SingleColorCharset) WriteTo(w io.Writer) (n int64, err error) {
	link := NewLinker(BitmapAddress, c.opt.VeryVerbose)
	_, err = link.WriteMap(LinkMap{
		BitmapAddress:           c.Bitmap[:],
		CharsetScreenRAMAddress: c.Screen[:],
		CharsetColorRAMAddress:  c.D800Color[:],
		0x2fe8:                  []byte{c.BorderColor, c.BackgroundColor},
	})
	if err != nil {
		return n, fmt.Errorf("link.WriteMap failed: %w", err)
	}
	if !c.opt.Display {
		return link.WriteTo(w)
	}
	if _, err = link.WritePrg(singleColorCharset.newHeader()); err != nil {
		return n, fmt.Errorf("link.WritePrg failed: %w", err)
	}
	link.SetByte(DisplayerSettingsStart+10, c.opt.NoFadeByte(), c.opt.NoLoopByte())
	if !c.opt.NoFade {
		link.Block(0xac00, 0xcf28)
	}
	if err = injectSID(link, c.opt.IncludeSID, c.opt.Quiet); err != nil {
		return n, fmt.Errorf("injectSID failed: %w", err)
	}
	return link.WriteTo(w)
}

func (c MixedCharset) WriteTo(w io.Writer) (n int64, err error) {
	link := NewLinker(BitmapAddress, c.opt.VeryVerbose)
	_, err = link.WriteMap(LinkMap{
		BitmapAddress:           c.Bitmap[:],
		CharsetScreenRAMAddress: c.Screen[:],
		CharsetColorRAMAddress:  c.D800Color[:],
		0x2fe8:                  []byte{c.BorderColor, c.BackgroundColor, c.D022Color, c.D023Color},
	})
	if err != nil {
		return n, fmt.Errorf("link.WriteMap failed: %w", err)
	}
	if !c.opt.Display {
		return link.WriteTo(w)
	}
	if _, err = link.WritePrg(mixedCharset.newHeader()); err != nil {
		return n, fmt.Errorf("link.WritePrg failed: %w", err)
	}
	link.SetByte(DisplayerSettingsStart+10, c.opt.NoFadeByte(), c.opt.NoLoopByte())
	if err = injectSID(link, c.opt.IncludeSID, c.opt.Quiet); err != nil {
		return 0, fmt.Errorf("injectSID failed: %w", err)
	}
	return link.WriteTo(w)
}

func (c PETSCIICharset) WriteTo(w io.Writer) (n int64, err error) {
	link := NewLinker(BitmapAddress, c.opt.VeryVerbose)
	_, err = link.WriteMap(LinkMap{
		CharsetScreenRAMAddress: c.Screen[:],
		CharsetColorRAMAddress:  c.D800Color[:],
		0x2fe8:                  []byte{c.BorderColor, c.BackgroundColor},
	})
	if err != nil {
		return n, fmt.Errorf("link.WriteMap failed: %w", err)
	}
	if !c.opt.Display {
		return link.WriteTo(w)
	}
	if _, err = link.WritePrg(petsciiCharset.newHeader()); err != nil {
		return n, fmt.Errorf("link.WritePrg failed: %w", err)
	}
	link.SetByte(DisplayerSettingsStart+7, c.Lowercase)
	link.SetByte(DisplayerSettingsStart+10, c.opt.NoFadeByte(), c.opt.NoLoopByte())
	if !c.opt.NoFade {
		link.Block(0xac00, 0xcf28)
	}
	if !c.opt.Quiet {
		if c.Lowercase == 1 {
			fmt.Println("lowercase rom charset found")
		} else {
			fmt.Println("uppercase rom charset found")
		}
	}
	if err = injectSID(link, c.opt.IncludeSID, c.opt.Quiet); err != nil {
		return n, fmt.Errorf("injectSID failed: %w", err)
	}
	return link.WriteTo(w)
}

func (c ECMCharset) WriteTo(w io.Writer) (n int64, err error) {
	link := NewLinker(BitmapAddress, c.opt.VeryVerbose)
	_, err = link.WriteMap(LinkMap{
		BitmapAddress:           c.Bitmap[:],
		CharsetScreenRAMAddress: c.Screen[:],
		CharsetColorRAMAddress:  c.D800Color[:],
		0x2fe8:                  []byte{c.BorderColor, c.BackgroundColor, c.D022Color, c.D023Color, c.D024Color},
	})
	if err != nil {
		return n, fmt.Errorf("link.WriteMap failed: %w", err)
	}
	if !c.opt.Display {
		return link.WriteTo(w)
	}
	if _, err = link.WritePrg(ecmCharset.newHeader()); err != nil {
		return n, fmt.Errorf("link.WritePrg failed: %w", err)
	}
	link.SetByte(DisplayerSettingsStart+10, c.opt.NoFadeByte(), c.opt.NoLoopByte())
	if !c.opt.NoFade {
		link.Block(0xac00, 0xcf28)
	}
	if err = injectSID(link, c.opt.IncludeSID, c.opt.Quiet); err != nil {
		return 0, fmt.Errorf("injectSID failed: %w", err)
	}
	return link.WriteTo(w)
}

func (s SingleColorSprites) WriteTo(w io.Writer) (n int64, err error) {
	header := defaultHeader()
	if s.opt.Display {
		header = singleColorSprites.newHeader()
		header = append(header, s.Columns, s.Rows, s.BackgroundColor, s.SpriteColor)
	}
	return writeData(w, header, s.Bitmap[:])
}

func (s MultiColorSprites) WriteTo(w io.Writer) (n int64, err error) {
	header := defaultHeader()
	if s.opt.Display {
		header = multiColorSprites.newHeader()
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

// DestinationFilename returns the filename based on Options given.
func DestinationFilename(filename string, opt Options) (destfilename string) {
	if opt.TargetDir != "" {
		destfilename = filepath.Dir(opt.TargetDir+string(os.PathSeparator)) + string(os.PathSeparator)
	}
	if opt.OutFile != "" {
		return destfilename + opt.OutFile
	}
	return destfilename + filepath.Base(strings.TrimSuffix(filename, filepath.Ext(filename))+".prg")
}
