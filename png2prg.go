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
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const version = "0.4-dev"

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

// binary blob containing c64 startaddress + basicstart + koala display code
//const koaladisplayb64 = `AQgLCAoAnjIwNjIAAAAAeKk3hQGpO40R0KkYjRjQqdiNFtCtEEeNINCNIdCiAL1AP50ABL1AQJ0A
//Bb1AQZ0ABr1AQp0AB70oQ50A2L0oRJ0A2b0oRZ0A2r0oRp0A2+jQzUxgCA==`

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

	switch img.graphicsType {
	case multiColorBitmap:
		k, err := img.convertToKoala()
		if err != nil {
			log.Fatalf("convertToKoala failed: %v", err)
		}
		writeKoala(k)
	case singleColorBitmap:
		h, err := img.convertToHires()
		if err != nil {
			log.Fatalf("convertToHires failed: %v", err)
		}
		writeHires(h)
	case singleColorCharset:
		c, err := img.convertToSingleColorCharset()
		if err != nil {
			log.Fatalf("convertToSingleColorCharset failed: %v", err)
		}
		writeSingleColorCharset(c)
	case multiColorCharset:
		c, err := img.convertToMultiColorCharset()
		if err != nil {
			log.Fatalf("convertToMultiColorCharset failed: %v", err)
		}
		writeMultiColorCharset(c)
	}

	return
}

func handleAnimation(ff []string) {
	var kk []Koala
	for _, f := range ff {
		kk = append(kk, makeKoalaFromFile(f))
	}

	if len(kk) > 1 {
		animPrgs := ProcessAnimation(kk)
		writeKoala(kk[0])
		for i, prg := range animPrgs {
			writePrgFile(frameFilename(i, kk[0].SourceFilename), prg)
		}
	}
	return
}

func writePrgFile(filename string, prg []byte) {
	if verbose {
		log.Printf("going to write file %q", filename)
	}
	f, err := os.Create(filename)
	check(err)
	defer f.Close()
	_, err = f.Write([]byte{0x00, 0x20})
	check(err)
	_, err = f.Write(prg[:])
	check(err)
	f.Sync()

	if !quiet {
		fmt.Printf("write %q\n", filename)
	}
}

func frameFilename(i int, filename string) string {
	dest := getdestfilename(filename)
	return strings.TrimSuffix(getdestfilename(filename), filepath.Ext(dest)) + ".frame" + strconv.Itoa(i) + ".prg"
}

func makeKoalaFromFile(filename string) Koala {
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
	if img.graphicsType != multiColorBitmap {
		if !quiet {
			log.Printf("hmmm, %q does not appear to be a multiColorBitmap, trying anyway.", filename)
		}
	}

	k, err := img.convertToKoala()
	if err != nil {
		log.Fatalf("convertToKoala failed: %v", err)
	}
	return k
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
		fmt.Printf("converted %q to %q\n", k.SourceFilename, destFilename)
	}
}

func writeHires(h Hires) {
	destFilename := getdestfilename(h.SourceFilename)
	if verbose {
		log.Printf("going to write file %q", destFilename)
	}
	f, err := os.Create(destFilename)
	check(err)
	defer f.Close()
	if display {
		_, err = f.Write(hiresdisplay)
		check(err)
	} else {
		_, err = f.Write([]byte{0x00, 0x20})
		check(err)
	}
	_, err = f.Write(h.Bitmap[:])
	check(err)
	_, err = f.Write(h.ScreenColor[:])
	check(err)
	f.Sync()

	if !quiet {
		fmt.Printf("converted %q to %q\n", h.SourceFilename, destFilename)
	}
}

func writeMultiColorCharset(c MultiColorCharset) {
	destFilename := getdestfilename(c.SourceFilename)
	if verbose {
		log.Printf("going to write file %q", destFilename)
	}
	f, err := os.Create(destFilename)
	check(err)
	defer f.Close()
	if display {
		_, err = f.Write(mcchardisplay)
		check(err)
		paddinglength := 0x2000 - 0x7ff - len(mcchardisplay)
		padding := [0x2000]byte{}
		_, err = f.Write(padding[0:paddinglength])
		check(err)
	} else {
		_, err = f.Write([]byte{0x00, 0x20})
		check(err)
	}

	_, err = f.Write(c.Bitmap[:])
	check(err)
	_, err = f.Write(c.Screen[:])
	check(err)
	_, err = f.Write([]byte{c.CharColor})
	check(err)
	_, err = f.Write([]byte{c.BgColor})
	check(err)
	_, err = f.Write([]byte{c.D022Color})
	check(err)
	_, err = f.Write([]byte{c.D023Color})
	check(err)
	f.Sync()

	if !quiet {
		fmt.Printf("converted %q to %q\n", c.SourceFilename, destFilename)
	}
}

func writeSingleColorCharset(c SingleColorCharset) {
	destFilename := getdestfilename(c.SourceFilename)
	if verbose {
		log.Printf("going to write file %q", destFilename)
	}
	f, err := os.Create(destFilename)
	check(err)
	defer f.Close()
	if display {
		_, err = f.Write(scchardisplay)
		check(err)
		paddinglength := 0x2000 - 0x7ff - len(scchardisplay)
		padding := [0x2000]byte{}
		_, err = f.Write(padding[0:paddinglength])
		check(err)
	} else {
		_, err = f.Write([]byte{0x00, 0x20})
		check(err)
	}

	_, err = f.Write(c.Bitmap[:])
	check(err)
	f.Sync()

	if !quiet {
		fmt.Printf("converted %q to %q\n", c.SourceFilename, destFilename)
	}
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
