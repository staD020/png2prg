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

type graphicsType int

const (
	singleColorBitmap graphicsType = iota
	multiColorBitmap
	singleColorCharset
	multiColorCharset
	singleColorSprites
	multiColorSprites
)

type colorInfo struct {
	rgb        RGB
	colorIndex byte
}

type sourceImage struct {
	sourceFilename         string
	image                  image.Image
	xOffset                int
	yOffset                int
	palette                map[RGB]byte
	charColors             [1000]map[RGB]byte
	backgroundCandidates   map[RGB]byte
	backgroundColor        colorInfo
	preferredBitpairColors []byte
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
}

var displayers = make(map[graphicsType][]byte, 0)

var outfile string
var targetdir string
var helpbool bool
var quiet bool
var verbose bool
var display bool
var forcecharcol int
var bitPairColors string

func init() {
	flag.BoolVar(&quiet, "q", false, "quiet")
	flag.BoolVar(&quiet, "quiet", false, "quiet")
	flag.BoolVar(&verbose, "v", false, "verbose")
	flag.BoolVar(&verbose, "verbose", false, "verbose")
	flag.BoolVar(&display, "d", false, "display")
	flag.BoolVar(&display, "display", false, "include displayer")
	flag.BoolVar(&helpbool, "h", false, "help")
	flag.BoolVar(&helpbool, "help", false, "help")
	flag.StringVar(&outfile, "o", "", "out")
	flag.StringVar(&outfile, "out", "", "specify outfile.prg, by default it changes extension to .prg")
	flag.StringVar(&targetdir, "td", "", "targetdir")
	flag.StringVar(&targetdir, "targetdir", "", "specify targetdir")

	flag.StringVar(&bitPairColors, "bitpair-colors", "", "prefer these colors in 2bit space, eg 0,6,14,3")
	flag.IntVar(&forcecharcol, "force-charcol", -1, "force multicolor charset d800 color -1: off 0: black 1: white 2: red, etc")
}

func main() {
	t0 := time.Now()
	flag.Parse()
	ff := flag.Args()
	if !quiet {
		fmt.Printf("png2prg %v by burglar\n", version)
	}
	if helpbool {
		help()
	}
	if len(ff) == 0 {
		printusage()
		os.Exit(0)
	}
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

	imgs, err := newSourceImages(filename)
	if err == nil {
		fmt.Printf("imgs: %d\n", len(imgs))
		return nil
	}
	if verbose {
		log.Printf("newSourceImages %q: %v", filename, err)
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
			if !quiet {
				log.Printf("falling back to multiColorBitmap because convertToMultiColorCharset %q failed: %v", filename, err)
			}
			img.graphicsType = multiColorBitmap
			img.findBackgroundColor()
			c, err = img.convertToKoala()
			if err != nil {
				return fmt.Errorf("convertToKoala %q failed: %v", filename, err)
			}
		}
	default:
		return fmt.Errorf("unsupported graphicsType for %q", filename)
	}

	destFilename := getDestinationFilename(img.sourceFilename)
	f, err := os.Create(destFilename)
	if err != nil {
		return fmt.Errorf("os.Create %q failed: %v", destFilename, err)
	}
	defer f.Close()
	if _, err = c.WriteTo(f); err != nil {
		return fmt.Errorf("WriteTo %q failed: %v", destFilename, err)
	}
	if !quiet {
		fmt.Printf("converted %q to %q\n", img.sourceFilename, destFilename)
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
	return writeData(w, [][]byte{header, c.Bitmap[:]})
}

func writeData(w io.Writer, data [][]byte) (n int64, err error) {
	for _, d := range data {
		m := 0
		m, err = w.Write(d)
		n += int64(m)
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func getDestinationFilename(filename string) (destfilename string) {
	if len(targetdir) > 0 {
		destfilename = filepath.Dir(targetdir+string(os.PathSeparator)) + string(os.PathSeparator)
	}
	if len(outfile) > 0 {
		destfilename = destfilename + outfile
	} else {
		destfilename = destfilename + filepath.Base(strings.TrimSuffix(filename, filepath.Ext(filename))+".prg")
	}
	return destfilename
}
