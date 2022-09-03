package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

	"github.com/staD020/png2prg"
)

var (
	memProfile string
	cpuProfile string
	help       bool
	numWorkers int
)

func main() {
	t0 := time.Now()
	opt := initAndParseFlags()
	filenames := flag.Args()
	if !opt.Quiet {
		fmt.Printf("multipng2prg %v by burg\n", png2prg.Version)
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
		printHelp()
		return
	}
	if len(filenames) == 0 {
		printUsage()
		return
	}
	if opt.IncludeSID != "" && !opt.Display {
		log.Printf("ignoring sid %q, it makes no sense without the -display flag set.\n", opt.IncludeSID)
	}

	filenames, err := expandWildcards(filenames)
	if err != nil {
		log.Fatalf("expandWildcards failed: %v", err)
	}

	wg := &sync.WaitGroup{}
	jobs := make(chan string, numWorkers)
	wg.Add(numWorkers)
	for worker := 0; worker < numWorkers; worker++ {
		go func(worker int) {
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
					fmt.Printf("worker %d converted %q to %q\n", worker, filename, opt.OutFile)
				}
			}
			wg.Done()
		}(worker)
	}

	for _, filename := range filenames {
		jobs <- filename
	}
	close(jobs)

	if memProfile != "" {
		time.Sleep(3 * time.Second)
		f, err := os.Create(memProfile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer f.Close()
		runtime.GC()
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
	}

	wg.Wait()

	if !opt.Quiet {
		fmt.Printf("converted %d files\n", len(filenames))
		fmt.Printf("elapsed: %v\n", time.Since(t0))
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
	// flag.IntVar(&opt.FrameDelay, "frame-delay", 6, "frames to wait before displaying next animation frame")
	// flag.IntVar(&opt.WaitSeconds, "wait-seconds", 0, "seconds to wait before animation starts")
	flag.IntVar(&numWorkers, "w", 8, "workers")
	flag.IntVar(&numWorkers, "workers", 8, "number of concurrent workers")

	flag.Parse()

	if numWorkers < 1 {
		log.Printf("warning: minimum amount of workers is 1, not %d\n", numWorkers)
		numWorkers = 1
	}
	return opt
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
