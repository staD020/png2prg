package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/staD020/png2prg"
)

var (
	memProfile string
	cpuProfile string
	help       bool
	parallel   bool
	numWorkers int
)

func main() {
	t0 := time.Now()
	opt := initAndParseFlags()
	filenames := flag.Args()
	if !opt.Quiet {
		fmt.Printf("png2prg %v by burg\n", png2prg.Version)
	}
	if cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			log.Fatalf("could not create CPU profile %q: %v", cpuProfile, err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatalf("could not start CPU profile: %v", err)
		}
		defer pprof.StopCPUProfile()
	}

	if help {
		png2prg.PrintHelp()
		return
	}
	if len(filenames) == 0 {
		png2prg.PrintUsage()
		return
	}
	if opt.IncludeSID != "" && !opt.Display {
		log.Printf("ignoring sid %q, it makes no sense without the -display flag set.\n", opt.IncludeSID)
	}

	filenames, err := png2prg.ExpandWildcards(filenames)
	if err != nil {
		log.Fatalf("expandWildcards failed: %v", err)
	}

	process := processAsOne
	if parallel {
		process = processInParallel
	}
	if err = process(opt, filenames...); err != nil {
		log.Fatalf("process failed: %v", err)
	}

	if !opt.Quiet {
		fmt.Printf("converted %d file(s)\n", len(filenames))
		fmt.Printf("elapsed: %v\n", time.Since(t0))
	}
}

func processAsOne(opt png2prg.Options, filenames ...string) error {
	opt.OutFile = png2prg.DestinationFilename(filenames[0], opt)
	opt.CurrentGraphicsType = png2prg.StringToGraphicsType(opt.GraphicsMode)

	p, err := png2prg.NewFromPath(opt, filenames...)
	if err != nil {
		return fmt.Errorf("NewFromPath failed: %w", err)
	}
	w, err := os.Create(opt.OutFile)
	if err != nil {
		return fmt.Errorf("os.Create failed: %w", err)
	}
	defer w.Close()
	_, err = p.WriteTo(w)
	if err != nil {
		return fmt.Errorf("WriteTo failed: %w", err)
	}
	return nil
}

func processInParallel(opt png2prg.Options, filenames ...string) error {
	wg := &sync.WaitGroup{}
	numWorkers := numWorkers
	if numWorkers > len(filenames) {
		numWorkers = len(filenames)
	}
	jobs := make(chan string, numWorkers)
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go worker(i, wg, opt, jobs)
	}
	if !opt.Quiet {
		fmt.Printf("started %d workers\n", numWorkers)
	}

	for i, filename := range filenames {
		jobs <- filename
		if i == int(len(filenames)/2) && memProfile != "" {
			f, err := os.Create(memProfile)
			if err != nil {
				return fmt.Errorf("could not create memory profile: %w", err)
			}
			defer f.Close()
			runtime.GC()
			if err := pprof.WriteHeapProfile(f); err != nil {
				return fmt.Errorf("could not write memory profile: %w", err)
			}
			if !opt.Quiet {
				fmt.Println("WriteHeapProfile done")
			}
		}
	}
	close(jobs)
	wg.Wait()
	return nil
}

func worker(i int, wg *sync.WaitGroup, opt png2prg.Options, jobs <-chan string) {
	defer wg.Done()
	for filename := range jobs {
		opt := opt
		opt.OutFile = png2prg.DestinationFilename(filename, opt)
		opt.CurrentGraphicsType = png2prg.StringToGraphicsType(opt.GraphicsMode)

		p, err := png2prg.NewFromPath(opt, filename)
		if err != nil {
			log.Printf("NewFromPath %q failed: %v", filename, err)
			continue
		}
		w, err := os.Create(opt.OutFile)
		if err != nil {
			log.Printf("os.Create %q failed: %v", opt.OutFile, err)
			continue
		}
		defer w.Close()
		_, err = p.WriteTo(w)
		if err != nil {
			log.Printf("WriteTo %q failed: %v", opt.OutFile, err)
			continue
		}
		if !opt.Quiet {
			fmt.Printf("worker %d converted %q to %q\n", i, filename, opt.OutFile)
		}
	}
}

func initAndParseFlags() (opt png2prg.Options) {
	flag.StringVar(&cpuProfile, "cpuprofile", "", "write cpu profile to `file`")
	flag.StringVar(&memProfile, "memprofile", "", "write memory profile to `file`")
	flag.BoolVar(&help, "h", false, "help")
	flag.BoolVar(&help, "help", false, "help")

	flag.BoolVar(&opt.Quiet, "q", false, "quiet")
	flag.BoolVar(&opt.Quiet, "quiet", false, "quiet, only display errors")
	flag.BoolVar(&opt.Verbose, "v", false, "verbose")
	flag.BoolVar(&opt.Verbose, "verbose", false, "verbose output")
	flag.BoolVar(&opt.Display, "d", false, "display")
	flag.BoolVar(&opt.Display, "display", false, "include displayer")
	flag.StringVar(&opt.OutFile, "o", "", "out")
	flag.StringVar(&opt.OutFile, "out", "", "specify outfile.prg, by default it changes extension to .prg")
	flag.StringVar(&opt.TargetDir, "td", "", "targetdir")
	flag.StringVar(&opt.TargetDir, "targetdir", "", "specify targetdir")
	flag.StringVar(&opt.GraphicsMode, "m", "", "mode")
	flag.StringVar(&opt.GraphicsMode, "mode", "", "force graphics mode to koala, hires, mccharset, sccharset, scsprites or mcsprites")
	flag.StringVar(&opt.BitpairColorsString, "bpc", "", "bitpair-colors")
	flag.StringVar(&opt.BitpairColorsString, "bitpair-colors", "", "prefer these colors in 2bit space, eg 0,6,14,3")
	flag.IntVar(&opt.ForceBorderColor, "force-border-color", -1, "force border color")

	flag.BoolVar(&opt.NoGuess, "ng", false, "no-guess")
	flag.BoolVar(&opt.NoGuess, "no-guess", false, "do not guess preferred bitpair-colors")
	flag.BoolVar(&opt.NoPackChars, "np", false, "no-pack")
	flag.BoolVar(&opt.NoPackChars, "no-pack", false, "do not pack chars (only for sc/mc charset)")
	flag.BoolVar(&opt.NoCrunch, "nc", false, "no-crunch")
	flag.BoolVar(&opt.NoCrunch, "no-crunch", false, "do not TSCrunch displayer")

	// flag.BoolVar(&opt.AlternativeFade, "alt-fade", false, "use alternative (less memory hungry) fade for animation displayers.")
	flag.StringVar(&opt.IncludeSID, "sid", "", "include .sid in displayer (see -help for free memory locations)")
	flag.IntVar(&opt.FrameDelay, "frame-delay", 6, "frames to wait before displaying next animation frame")
	flag.IntVar(&opt.WaitSeconds, "wait-seconds", 0, "seconds to wait before animation starts")
	w := int(runtime.NumCPU() / 2)
	flag.IntVar(&numWorkers, "w", w, "workers")
	flag.IntVar(&numWorkers, "workers", w, "number of concurrent workers in parallel mode")
	flag.BoolVar(&parallel, "p", false, "parallel")
	flag.BoolVar(&parallel, "parallel", false, "run number of workers in parallel for fast conversion, treat each image as a standalone, not to be used for animations")
	flag.Parse()
	return opt
}
