package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/staD020/png2prg"
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
	alternativeFade     bool
	bitpairColorsString string
	noGuess             bool
	graphicsMode        string
	forceBorderColor    int
	includeSID          string
	frameDelay          int
	waitSeconds         int
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
	// flag.BoolVar(&alternativeFade, "alt-fade", false, "use alternative (less memory hungry) fade for animation displayers.")
	flag.StringVar(&bitpairColorsString, "bpc", "", "bitpair-colors")
	flag.StringVar(&bitpairColorsString, "bitpair-colors", "", "prefer these colors in 2bit space, eg 0,6,14,3")
	flag.IntVar(&forceBorderColor, "force-border-color", -1, "force border color")
	flag.StringVar(&includeSID, "sid", "", "include .sid in displayer (see -help for free memory locations)")
	flag.IntVar(&frameDelay, "frame-delay", 6, "frames to wait before displaying next animation frame")
	flag.IntVar(&waitSeconds, "wait-seconds", 0, "seconds to wait before animation starts")
}

func main() {
	t0 := time.Now()
	flag.Parse()
	filenames := flag.Args()
	if !quiet {
		fmt.Printf("png2prg %v by burg\n", png2prg.Version)
	}

	if help {
		printHelp()
		return
	}
	if len(filenames) == 0 {
		printUsage()
		return
	}

	if includeSID != "" && !display {
		log.Printf("ignoring sid %q, it makes no sense without the -display flag set.\n", includeSID)
	}

	filenames, err := expandWildcards(filenames)
	if err != nil {
		log.Fatalf("expandWildcards failed: %v", err)
	}

	opt := png2prg.Options{
		OutFile:             outfile,
		TargetDir:           targetdir,
		Display:             display,
		Verbose:             verbose,
		Quiet:               quiet,
		NoPackChars:         noPackChars,
		NoCrunch:            noCrunch,
		AlternativeFade:     alternativeFade,
		BitpairColorsString: bitpairColorsString,
		NoGuess:             noGuess,
		GraphicsMode:        graphicsMode,
		CurrentGraphicsType: png2prg.StringToGraphicsType(graphicsMode),
		FrameDelay:          frameDelay,
		WaitSeconds:         waitSeconds,
		ForceBorderColor:    forceBorderColor,
		IncludeSID:          includeSID,
	}
	opt.OutFile = png2prg.DestinationFilename(filenames[0], opt)

	w, err := os.Create(opt.OutFile)
	if err != nil {
		log.Fatalf("os.Create failed: %v", err)
	}
	defer w.Close()
	p, err := png2prg.NewFromPath(opt, filenames...)
	if err != nil {
		log.Fatalf("NewFromPath failed: %v", err)
	}
	_, err = p.WriteTo(w)
	if err != nil {
		log.Fatalf("WriteTo failed: %v", err)
	}

	if !quiet {
		fmt.Printf("elapsed: %v\n", time.Since(t0))
	}

	/*
		p := "perplex 2"
		fmt.Println("palette:", p)
		for _, ci := range c64palettes[p] {
			fmt.Printf("%s\n", ci)
		}
	*/
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
