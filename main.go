package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
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
	frameDelay          int
	waitSeconds         int
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
	flag.IntVar(&frameDelay, "frame-delay", 6, "frames to wait before displaying next animation frame")
	flag.IntVar(&waitSeconds, "wait-seconds", 0, "seconds to wait before animation starts")
}

func main() {
	t0 := time.Now()
	flag.Parse()
	filenames := flag.Args()
	if !quiet {
		fmt.Printf("png2prg %v by burglar\n", version)
	}

	if help {
		printHelp()
	}
	if len(filenames) == 0 {
		printUsage()
		os.Exit(0)
	}
	currentGraphicsType = stringToGraphicsType(graphicsMode)
	if forceBorderColor > 15 {
		forceBorderColor = -1
	}

	if includeSID != "" && !display {
		log.Printf("ignoring sid %q, it makes no sense without the -display flag set.\n", includeSID)
	}

	filenames, err := expandWildcards(filenames)
	if err != nil {
		log.Fatalf("expandWildcards failed: %v", err)
	}

	if err := processFiles(filenames); err != nil {
		log.Fatalf("processFiles failed: %v", err)
	}

	if !quiet {
		fmt.Printf("elapsed: %v\n", time.Since(t0))
	}
}

func expandWildcards(filenames []string) (result []string, err error) {
	for _, filename := range filenames {
		if !strings.ContainsAny(filename, "?*") {
			result = append(result, filename)
			continue
		}
		dir := filepath.Dir(filename)
		ff, err := os.ReadDir(dir)
		if err != nil {
			return nil, fmt.Errorf("os.ReadDir %q failed: %w", dir, err)
		}
		name := filepath.Base(filename)
		for _, f := range ff {
			if f.IsDir() {
				continue
			}
			ok, err := filepath.Match(name, f.Name())
			if err != nil {
				return nil, fmt.Errorf("filepath.Match %q failed: %w", filename, err)
			}
			if ok {
				result = append(result, filepath.Join(dir, f.Name()))
			}
		}
	}
	return result, nil
}
