// +build ignore

// This program generates display.go. It can be invoked by running
// go generate
package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

const (
	filename               = "GEN_display.go"
	koalaDisplayerFilename = "display_koala.prg"
	koalaDestinationVar    = "koaladisplayb64"
	hiresDisplayerFilename = "display_hires.prg"
	hiresDestinationVar    = "hiresdisplayb64"

	mcCharsetDisplayerFilename = "display_mc_charset.prg"
	mcCharsetDestinationVar    = "mcchardisplayb64"
	scCharsetDisplayerFilename = "display_sc_charset.prg"
	scCharsetDestinationVar    = "scchardisplayb64"

	mcSpriteDisplayerFilename = "display_mc_sprites.prg"
	mcSpriteDestinationVar    = "mcspritedisplayb64"
	scSpriteDisplayerFilename = "display_sc_sprites.prg"
	scSpriteDestinationVar    = "scspritedisplayb64"
)

func main() {
	w, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Unable to create file %q: %v", filename, err)
	}
	defer w.Close()

	fmt.Fprintln(w, "// This file was generated automatically with go generate.")
	fmt.Fprintln(w, "package main")
	fmt.Fprintln(w, "")

	type displayer struct {
		filename string
		varName  string
	}
	dd := []displayer{
		displayer{koalaDisplayerFilename, koalaDestinationVar},
		displayer{hiresDisplayerFilename, hiresDestinationVar},
		displayer{mcCharsetDisplayerFilename, mcCharsetDestinationVar},
		displayer{scCharsetDisplayerFilename, scCharsetDestinationVar},
		displayer{mcSpriteDisplayerFilename, mcSpriteDestinationVar},
		displayer{scSpriteDisplayerFilename, scSpriteDestinationVar},
	}

	for _, d := range dd {
		file, err := os.Open(d.filename)
		if err != nil {
			log.Fatalf("Unable to open file %q: %v", d.filename, err)
		}
		defer file.Close()
		bin, err := ioutil.ReadAll(file)
		if err != nil {
			log.Fatalf("Unable to ReadAll file %q: %v", d.filename, err)
		}
		str := base64.StdEncoding.EncodeToString(bin)
		fmt.Fprintf(w, "const "+d.varName+" = `")
		fmt.Fprintf(w, str+"`\n\n")
	}
}
