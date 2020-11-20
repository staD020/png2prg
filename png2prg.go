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
	sourceFilename       string
	image                image.Image
	xOffset              int
	yOffset              int
	palette              map[RGB]byte
	charColors           [1000]map[RGB]byte
	backgroundCandidates map[RGB]byte
	backgroundColor      colorInfo
	graphicsType         graphicsType
}

type Koala struct {
	SourceFilename string
	Bitmap         [8000]byte
	ScreenColor    [1000]byte
	D800Color      [1000]byte
	BgColor        byte
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

var koaladisplay []byte
var hiresdisplay []byte
var mcchardisplay []byte
var scchardisplay []byte
var outfile string
var targetdir string
var helpbool bool
var quiet bool
var verbose bool
var display bool
var forcebgcol int
var forcecharcol int

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

	flag.IntVar(&forcebgcol, "force-bgcol", -1, "force background color -1: off 0: black 1: white 2: red, etc")
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
	var err error
	if display {
		if koaladisplay, err = base64.StdEncoding.DecodeString(koaladisplayb64); err != nil {
			log.Fatalf("unable to decode koaladisplayb64: %v", err)
		}
		if hiresdisplay, err = base64.StdEncoding.DecodeString(hiresdisplayb64); err != nil {
			log.Fatalf("unable to decode hiresdisplayb64: %v", err)
		}
		if mcchardisplay, err = base64.StdEncoding.DecodeString(mcchardisplayb64); err != nil {
			log.Fatalf("unable to decode mcchardisplayb64: %v", err)
		}
		if scchardisplay, err = base64.StdEncoding.DecodeString(scchardisplayb64); err != nil {
			log.Fatalf("unable to decode scchardisplayb64: %v", err)
		}
	}

	processFiles(ff)

	if !quiet {
		fmt.Printf("elapsed: %v\n", time.Since(t0))
	}
}

func processFiles(ff []string) {
	if len(ff) < 1 {
		log.Println("no files supplied, nothing to do.")
		return
	}
	if len(ff) > 1 {
		handleAnimation(ff)
		return
	}

	filename := ff[0]
	if verbose {
		log.Printf("processing file %q", filename)
	}

	img, err := newSourceImage(filename)
	if err != nil {
		log.Fatalf("newSourceImage failed: %v", err)
	}
	err = img.analyze()
	if err != nil {
		log.Fatalf("analyze failed: %v", err)
	}

	var c io.WriterTo
	switch img.graphicsType {
	case multiColorBitmap:
		c, err = img.convertToKoala()
		if err != nil {
			log.Fatalf("convertToKoala failed: %v", err)
		}
	case singleColorBitmap:
		c, err = img.convertToHires()
		if err != nil {
			log.Fatalf("convertToHires failed: %v", err)
		}
	case singleColorCharset:
		c, err = img.convertToSingleColorCharset()
		if err != nil {
			log.Fatalf("convertToSingleColorCharset failed: %v", err)
		}
	case multiColorCharset:
		c, err = img.convertToMultiColorCharset()
		if err != nil {
			log.Fatalf("convertToMultiColorCharset failed: %v", err)
		}
	}

	w, ok := c.(io.WriterTo)
	if !ok {
		log.Fatalf("converted image is not an io.WriterTo: %v", c)
	}

	destFilename := getdestfilename(img.sourceFilename)
	f, err := os.Create(destFilename)
	if err != nil {
		log.Fatalf("os.Create %q failed: %v", destFilename, err)
	}
	defer f.Close()
	_, err = w.WriteTo(f)
	if err != nil {
		log.Fatalf("WriteTo %q failed: %v", destFilename, err)
	}
	if !quiet {
		fmt.Printf("converted %q to %q\n", img.sourceFilename, destFilename)
	}

	return
}

func (k Koala) WriteTo(w io.Writer) (n int64, err error) {
	m := 0
	header := []byte{0x00, 0x20}
	if display {
		header = koaladisplay
	}
	data := [][]byte{header, k.Bitmap[:], k.ScreenColor[:], k.D800Color[:], []byte{k.BgColor}}
	for _, d := range data {
		m, err = w.Write(d)
		n += int64(m)
		if err != nil {
			return n, err
		}
	}
	return n, err
}

func writeKoala(k Koala) {
	destFilename := getdestfilename(k.SourceFilename)
	if verbose {
		log.Printf("going to write file %q", destFilename)
	}
	f, err := os.Create(destFilename)
	check(err)
	defer f.Close()
	if display {
		_, err = f.Write(koaladisplay)
		check(err)
	} else {
		_, err = f.Write([]byte{0x00, 0x20})
		check(err)
	}
	_, err = f.Write(k.Bitmap[:])
	check(err)
	_, err = f.Write(k.ScreenColor[:])
	check(err)
	_, err = f.Write(k.D800Color[:])
	check(err)
	_, err = f.Write([]byte{k.BgColor})
	check(err)
	f.Sync()

	if !quiet {
		fmt.Printf("converted %q to koala %q\n", k.SourceFilename, destFilename)
	}
}

func (h Hires) WriteTo(w io.Writer) (n int64, err error) {
	header := []byte{0x00, 0x20}
	if display {
		header = hiresdisplay
	}
	data := [][]byte{header, h.Bitmap[:], h.ScreenColor[:]}
	for _, d := range data {
		m := 0
		m, err = w.Write(d)
		n += int64(m)
		if err != nil {
			return n, err
		}
	}
	return n, err
}

func (c MultiColorCharset) WriteTo(w io.Writer) (n int64, err error) {
	header := []byte{0x00, 0x20}
	if display {
		header = mcchardisplay
	}
	data := [][]byte{header, c.Bitmap[:], c.Screen[:], []byte{c.CharColor, c.BgColor, c.D022Color, c.D023Color}}
	for _, d := range data {
		m := 0
		m, err = w.Write(d)
		n += int64(m)
		if err != nil {
			return n, err
		}
	}
	return n, err
}

func (c SingleColorCharset) WriteTo(w io.Writer) (n int64, err error) {
	header := []byte{0x00, 0x20}
	if display {
		header = scchardisplay
	}
	data := [][]byte{header, c.Bitmap[:]}
	for _, d := range data {
		m := 0
		m, err = w.Write(d)
		n += int64(m)
		if err != nil {
			return n, err
		}
	}
	return n, err
}

func getdestfilename(filename string) (destfilename string) {
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

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
