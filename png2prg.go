package main

//go:generate go run generate.go

import (
	"encoding/base64"
	"flag"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const version = "0.5-dev"

type RGB struct {
	R, G, B byte
}

type C64RGB struct {
	Name       string
	ColorIndex byte
	RGB        RGB
}

type graphicsType byte

const (
	singleColorBitmap graphicsType = iota
	multiColorBitmap
	singleColorCharset
	multiColorCharset
	singleColorSprites
	multiColorSprites
)

func (t graphicsType) String() string {
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
	}
	return ""
}

type colorInfo struct {
	rgb        RGB
	colorIndex byte
}

type bitpairColors []byte

func (b bitpairColors) String() string {
	s := ""
	for i, v := range b {
		s = s + strconv.Itoa(int(v))
		if i < len(b)-1 {
			s = s + ","
		}
	}
	return s
}

type sourceImage struct {
	sourceFilename         string
	image                  image.Image
	xOffset                int
	yOffset                int
	width                  int
	height                 int
	palette                map[RGB]byte
	charColors             [1000]map[RGB]byte
	backgroundCandidates   map[RGB]byte
	backgroundColor        colorInfo
	preferredBitpairColors bitpairColors
	graphicsType           graphicsType
}

type Koala struct {
	SourceFilename string
	Bitmap         [8000]byte
	ScreenColor    [1000]byte
	D800Color      [1000]byte
	BgColor        byte
}

type MultiColorChar struct {
	CharIndex   int
	Bitmap      [8]byte
	BgColor     byte
	ScreenColor byte
	D800Color   byte
}

type Hires struct {
	SourceFilename string
	Bitmap         [8000]byte
	ScreenColor    [1000]byte
}

type MultiColorCharset struct {
	SourceFilename string
	Bitmap         [0x800]byte
	Screen         [1000]byte
	CharColor      byte
	BgColor        byte
	D022Color      byte
	D023Color      byte
}

type SingleColorCharset struct {
	SourceFilename string
	Bitmap         [0x800]byte
	Screen         [1000]byte
}

type SingleColorSprites struct {
	SourceFilename string
	Bitmap         []byte
	SpriteColor    byte
	BgColor        byte
}

type MultiColorSprites struct {
	SourceFilename string
	Bitmap         []byte
	SpriteColor    byte
	BgColor        byte
	D025Color      byte
	D026Color      byte
}

var outfile string
var targetdir string
var help bool
var quiet bool
var verbose bool
var display bool
var noPackChars bool
var bitpairColorsString string
var noGuess bool
var graphicsMode string
var currentGraphicsType graphicsType

func init() {
	flag.BoolVar(&quiet, "q", false, "quiet")
	flag.BoolVar(&quiet, "quiet", false, "quiet")
	flag.BoolVar(&verbose, "v", false, "verbose")
	flag.BoolVar(&verbose, "verbose", false, "verbose")
	flag.BoolVar(&display, "d", false, "display")
	flag.BoolVar(&display, "display", false, "include displayer")
	flag.BoolVar(&help, "h", false, "help")
	flag.BoolVar(&help, "help", false, "help")
	flag.StringVar(&outfile, "o", "", "out")
	flag.StringVar(&outfile, "out", "", "specify outfile.prg, by default it changes extension to .prg")
	flag.StringVar(&targetdir, "td", "", "targetdir")
	flag.StringVar(&targetdir, "targetdir", "", "specify targetdir")
	flag.StringVar(&graphicsMode, "m", "", "mode")
	flag.StringVar(&graphicsMode, "mode", "", "force graphics mode koala, hires, mccharset or sccharset")

	flag.BoolVar(&noGuess, "ng", false, "no-guess")
	flag.BoolVar(&noGuess, "no-guess", false, "do not guess preferred bitpair-colors")
	flag.BoolVar(&noPackChars, "np", false, "no-pack")
	flag.BoolVar(&noPackChars, "no-pack", false, "do not pack chars (only for sc/mc charset)")
	flag.StringVar(&bitpairColorsString, "bitpair-colors", "", "prefer these colors in 2bit space, eg 0,6,14,3")
}

func main() {
	t0 := time.Now()
	flag.Parse()
	ff := flag.Args()
	if !quiet {
		fmt.Printf("png2prg %v by burglar\n", version)
	}

	if help {
		printHelp()
	}
	if len(ff) == 0 {
		printUsage()
		os.Exit(0)
	}
	setGraphicsType(graphicsMode)
	if display {
		if err := initDisplayers(); err != nil {
			log.Fatal(err)
		}
	}

	if err := processFiles(ff); err != nil {
		log.Fatal(err)
	}

	if !quiet {
		fmt.Printf("elapsed: %v\n", time.Since(t0))
	}
}

func setGraphicsType(s string) {
	switch s {
	case "koala":
		currentGraphicsType = multiColorBitmap
	case "hires":
		currentGraphicsType = singleColorBitmap
	case "sccharset":
		currentGraphicsType = singleColorCharset
	case "mccharset":
		currentGraphicsType = multiColorCharset
	case "scsprites":
		currentGraphicsType = singleColorSprites
	case "mcsprites":
		currentGraphicsType = multiColorSprites
	}
}

var displayers = make(map[graphicsType][]byte, 0)

func initDisplayers() error {
	bin, err := base64.StdEncoding.DecodeString(koaladisplayb64)
	if err != nil {
		return fmt.Errorf("unable to decode koaladisplayb64: %v", err)
	}
	displayers[multiColorBitmap] = bin
	bin, err = base64.StdEncoding.DecodeString(hiresdisplayb64)
	if err != nil {
		return fmt.Errorf("unable to decode hiresdisplayb64: %v", err)
	}
	displayers[singleColorBitmap] = bin
	bin, err = base64.StdEncoding.DecodeString(mcchardisplayb64)
	if err != nil {
		return fmt.Errorf("unable to decode mcchardisplayb64: %v", err)
	}
	displayers[multiColorCharset] = bin
	bin, err = base64.StdEncoding.DecodeString(scchardisplayb64)
	if err != nil {
		return fmt.Errorf("unable to decode scchardisplayb64: %v", err)
	}
	displayers[singleColorCharset] = bin
	return nil
}

func processFiles(ff []string) (err error) {
	if len(ff) < 1 {
		log.Println("no files supplied, nothing to do.")
		return err
	}
	if len(ff) > 1 {
		handleAnimation(ff)
		return err
	}

	filename := ff[0]
	if verbose {
		log.Printf("processing file %q", filename)
	}

	var imgs []sourceImage
	if strings.ToLower(filepath.Ext(filename)) == ".gif" {
		imgs, err = newSourceImages(filename)
		if err == nil {
			fmt.Printf("imgs: %d\n", len(imgs))
			return nil
		}

		if verbose {
			log.Printf("newSourceImages %q: %v", filename, err)
		}
	}

	img, err := newSourceImage(filename)
	if err != nil {
		return fmt.Errorf("newSourceImage %q failed: %v", filename, err)
	}
	err = img.analyze()
	if err != nil {
		return fmt.Errorf("analyze %q failed: %v", filename, err)
	}

	var c io.WriterTo
	switch img.graphicsType {
	case multiColorBitmap:
		c, err = img.convertToKoala()
		if err != nil {
			return fmt.Errorf("convertToKoala %q failed: %v", filename, err)
		}
	case singleColorBitmap:
		c, err = img.convertToHires()
		if err != nil {
			return fmt.Errorf("convertToHires %q failed: %v", filename, err)
		}
	case singleColorCharset:
		c, err = img.convertToSingleColorCharset()
		if err != nil {
			return fmt.Errorf("convertToSingleColorCharset %q failed: %v", filename, err)
		}
	case multiColorCharset:
		c, err = img.convertToMultiColorCharset()
		if err != nil {
			if graphicsMode != "" {
				return fmt.Errorf("convertToMultiColorCharset %q failed: %v", filename, err)
			}
			img.graphicsType = multiColorBitmap
			if !quiet {
				log.Printf("falling back to %s because convertToMultiColorCharset %q failed: %v", img.graphicsType, filename, err)
			}
			img.findBackgroundColor()
			c, err = img.convertToKoala()
			if err != nil {
				return fmt.Errorf("convertToKoala %q failed: %v", filename, err)
			}
		}
	case singleColorSprites:
		c, err = img.convertToSingleColorSprites()
		if err != nil {
			return fmt.Errorf("convertToSingleColorSprites %q failed: %v", filename, err)
		}
	case multiColorSprites:
		c, err = img.convertToMultiColorSprites()
		if err != nil {
			return fmt.Errorf("convertToMultiColorSprites %q failed: %v", filename, err)
		}
	default:
		return fmt.Errorf("unsupported graphicsType for %q", filename)
	}

	destFilename := destinationFilename(img.sourceFilename)
	f, err := os.Create(destFilename)
	if err != nil {
		return fmt.Errorf("os.Create %q failed: %v", destFilename, err)
	}
	defer f.Close()
	if _, err = c.WriteTo(f); err != nil {
		return fmt.Errorf("WriteTo %q failed: %v", destFilename, err)
	}
	if !quiet {
		fmt.Printf("converted %q to %q in %q format\n", img.sourceFilename, destFilename, img.graphicsType)
	}

	return nil
}

func (k Koala) WriteTo(w io.Writer) (n int64, err error) {
	header := []byte{0x00, 0x20}
	if display {
		header = displayers[multiColorBitmap]
	}
	return writeData(w, [][]byte{header, k.Bitmap[:], k.ScreenColor[:], k.D800Color[:], []byte{k.BgColor}})
}

func (h Hires) WriteTo(w io.Writer) (n int64, err error) {
	header := []byte{0x00, 0x20}
	if display {
		header = displayers[singleColorBitmap]
	}
	return writeData(w, [][]byte{header, h.Bitmap[:], h.ScreenColor[:]})
}

func (c MultiColorCharset) WriteTo(w io.Writer) (n int64, err error) {
	header := []byte{0x00, 0x20}
	if display {
		header = displayers[multiColorCharset]
	}
	return writeData(w, [][]byte{header, c.Bitmap[:], c.Screen[:], []byte{c.CharColor, c.BgColor, c.D022Color, c.D023Color}})
}

func (c SingleColorCharset) WriteTo(w io.Writer) (n int64, err error) {
	header := []byte{0x00, 0x20}
	if display {
		header = displayers[singleColorCharset]
	}
	return writeData(w, [][]byte{header, c.Bitmap[:], c.Screen[:]})
}

func (s SingleColorSprites) WriteTo(w io.Writer) (n int64, err error) {
	header := []byte{0x00, 0x20}
	//return writeData(w, [][]byte{header, s.Bitmap[:], []byte{s.BgColor, s.SpriteColor}})
	return writeData(w, [][]byte{header, s.Bitmap[:]})
}

func (s MultiColorSprites) WriteTo(w io.Writer) (n int64, err error) {
	header := []byte{0x00, 0x20}
	//return writeData(w, [][]byte{header, s.Bitmap[:], []byte{s.BgColor, s.D025Color, s.SpriteColor, s.D026Color}})
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
	switch {
	case len(outfile) > 0:
		destfilename = destfilename + outfile
	case len(outfile) == 0:
		destfilename = destfilename + filepath.Base(strings.TrimSuffix(filename, filepath.Ext(filename))+".prg")
	}
	return destfilename
}
