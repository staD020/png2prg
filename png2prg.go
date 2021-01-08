package main

//go:generate go run generate.go

import (
	"encoding/base64"
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
)

const version = "0.6"

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
	unknownGraphicsType graphicsType = iota
	singleColorBitmap
	multiColorBitmap
	singleColorCharset
	multiColorCharset
	singleColorSprites
	multiColorSprites
)

func stringToGraphicsType(s string) graphicsType {
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
	default:
		return "unknown"
	}
}

type colorInfo struct {
	rgb        RGB
	colorIndex byte
}

type bitpairColors []byte

func (b bitpairColors) String() (s string) {
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

func processFiles(filenames ...string) (err error) {
	if len(filenames) < 1 {
		log.Println("no files supplied, nothing to do.")
		return err
	}

	imgs, err := newSourceImages(filenames...)
	switch {
	case err != nil:
		return fmt.Errorf("newSourceImages failed: %v", err)
	case len(imgs) == 0:
		return fmt.Errorf("no images found")
	case len(imgs) > 1:
		if err = handleAnimation(imgs); err != nil {
			return fmt.Errorf("handleAnimation failed: %v", err)
		}
		return nil
	}

	img := imgs[0]
	if verbose {
		log.Printf("processing file %q", img.sourceFilename)
	}
	if err = img.analyze(); err != nil {
		return fmt.Errorf("analyze %q failed: %v", img.sourceFilename, err)
	}

	var c io.WriterTo
	switch img.graphicsType {
	case multiColorBitmap:
		if c, err = img.convertToKoala(); err != nil {
			return fmt.Errorf("convertToKoala %q failed: %v", img.sourceFilename, err)
		}
	case singleColorBitmap:
		if c, err = img.convertToHires(); err != nil {
			return fmt.Errorf("convertToHires %q failed: %v", img.sourceFilename, err)
		}
	case singleColorCharset:
		if c, err = img.convertToSingleColorCharset(); err != nil {
			return fmt.Errorf("convertToSingleColorCharset %q failed: %v", img.sourceFilename, err)
		}
	case multiColorCharset:
		c, err = img.convertToMultiColorCharset()
		if err != nil {
			if graphicsMode != "" {
				return fmt.Errorf("convertToMultiColorCharset %q failed: %v", img.sourceFilename, err)
			}
			if !quiet {
				fmt.Printf("falling back to %s because convertToMultiColorCharset %q failed: %v\n", multiColorBitmap, img.sourceFilename, err)
			}
			img.graphicsType = multiColorBitmap
			err = img.findBackgroundColor()
			if err != nil {
				return fmt.Errorf("findBackgroundColor %q failed: %v", img.sourceFilename, err)
			}
			c, err = img.convertToKoala()
			if err != nil {
				return fmt.Errorf("convertToKoala %q failed: %v", img.sourceFilename, err)
			}
		}
	case singleColorSprites:
		if c, err = img.convertToSingleColorSprites(); err != nil {
			return fmt.Errorf("convertToSingleColorSprites %q failed: %v", img.sourceFilename, err)
		}
	case multiColorSprites:
		if c, err = img.convertToMultiColorSprites(); err != nil {
			return fmt.Errorf("convertToMultiColorSprites %q failed: %v", img.sourceFilename, err)
		}
	default:
		return fmt.Errorf("unsupported graphicsType for %q", img.sourceFilename)
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

// defaultHeader returns the startaddress of a file located at 0x2000
func defaultHeader() []byte {
	return []byte{0x00, 0x20}
}

func (k Koala) WriteTo(w io.Writer) (n int64, err error) {
	header := defaultHeader()
	if display {
		header = displayers[multiColorBitmap]
	}
	return writeData(w, [][]byte{header, k.Bitmap[:], k.ScreenColor[:], k.D800Color[:], []byte{k.BgColor}})
}

func (h Hires) WriteTo(w io.Writer) (n int64, err error) {
	header := defaultHeader()
	if display {
		header = displayers[singleColorBitmap]
	}
	return writeData(w, [][]byte{header, h.Bitmap[:], h.ScreenColor[:]})
}

func (c MultiColorCharset) WriteTo(w io.Writer) (n int64, err error) {
	header := defaultHeader()
	if display {
		header = displayers[multiColorCharset]
	}
	return writeData(w, [][]byte{header, c.Bitmap[:], c.Screen[:], []byte{c.CharColor, c.BgColor, c.D022Color, c.D023Color}})
}

func (c SingleColorCharset) WriteTo(w io.Writer) (n int64, err error) {
	header := defaultHeader()
	if display {
		header = displayers[singleColorCharset]
	}
	return writeData(w, [][]byte{header, c.Bitmap[:], c.Screen[:]})
}

func (s SingleColorSprites) WriteTo(w io.Writer) (n int64, err error) {
	if display && !quiet {
		fmt.Printf("no displayer support for %s, maybe try without -d/-display\n", singleColorSprites)
	}
	header := defaultHeader()
	//return writeData(w, [][]byte{header, s.Bitmap[:], []byte{s.BgColor, s.SpriteColor}})
	return writeData(w, [][]byte{header, s.Bitmap[:]})
}

func (s MultiColorSprites) WriteTo(w io.Writer) (n int64, err error) {
	if display && !quiet {
		fmt.Printf("no displayer support for %s, maybe try without -d/-display\n", multiColorSprites)
	}
	header := defaultHeader()
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
		return destfilename + outfile
	case len(outfile) == 0:
		return destfilename + filepath.Base(strings.TrimSuffix(filename, filepath.Ext(filename))+".prg")
	}
	return destfilename
}
