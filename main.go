package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

var (
	outfile             string
	targetdir           string
	help                bool
	quiet               bool
	verbose             bool
	display             bool
	noPackChars         bool
	noCrunch            bool
	bitpairColorsString string
	noGuess             bool
	graphicsMode        string
	forceBorderColor    int
	includeSID          string
	currentGraphicsType graphicsType
)

func init() {
	flag.BoolVar(&quiet, "q", false, "quiet")
	flag.BoolVar(&quiet, "quiet", false, "quiet, only display errors")
	flag.BoolVar(&verbose, "v", false, "verbose")
	flag.BoolVar(&verbose, "verbose", false, "verbose output")
	flag.BoolVar(&display, "d", false, "display")
	flag.BoolVar(&display, "display", false, "include displayer")
	flag.BoolVar(&help, "h", false, "help")
	flag.BoolVar(&help, "help", false, "help")
	flag.StringVar(&outfile, "o", "", "out")
	flag.StringVar(&outfile, "out", "", "specify outfile.prg, by default it changes extension to .prg")
	flag.StringVar(&targetdir, "td", "", "targetdir")
	flag.StringVar(&targetdir, "targetdir", "", "specify targetdir")
	flag.StringVar(&graphicsMode, "m", "", "mode")
	flag.StringVar(&graphicsMode, "mode", "", "force graphics mode to koala, hires, mccharset, sccharset, scsprites or mcsprites")

	flag.BoolVar(&noGuess, "ng", false, "no-guess")
	flag.BoolVar(&noGuess, "no-guess", false, "do not guess preferred bitpair-colors")
	flag.BoolVar(&noPackChars, "np", false, "no-pack")
	flag.BoolVar(&noPackChars, "no-pack", false, "do not pack chars (only for sc/mc charset)")
	flag.BoolVar(&noCrunch, "nc", false, "no-crunch")
	flag.BoolVar(&noCrunch, "no-crunch", false, "do not TSCrunch koala/hires displayer")
	flag.StringVar(&bitpairColorsString, "bpc", "", "bitpair-colors")
	flag.StringVar(&bitpairColorsString, "bitpair-colors", "", "prefer these colors in 2bit space, eg 0,6,14,3")
	flag.IntVar(&forceBorderColor, "force-border-color", -1, "force border color")
	flag.StringVar(&includeSID, "sid", "", "include .sid (0x0d00-0x1fff or 0x9000+) in displayer")
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
	currentGraphicsType = stringToGraphicsType(graphicsMode)
	if forceBorderColor > 15 {
		forceBorderColor = -1
	}

	if err := processFiles(ff); err != nil {
		log.Fatalf("processFiles failed: %v", err)
	}

	if !quiet {
		fmt.Printf("elapsed: %v\n", time.Since(t0))
	}
}
