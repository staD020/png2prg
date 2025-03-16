package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
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
	parallel   bool
	altOffset  bool
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

	filenames, err := expandWildcards(filenames)
	if err != nil {
		log.Printf("expandWildcards failed: %v", err)
	}
	if len(filenames) == 0 {
		log.Printf("no files found")
		return
	}

	process := processAsOne
	if parallel {
		process = processInParallel
	}
	if err = process(&opt, filenames...); err != nil {
		log.Fatalf("process failed: %v", err)
		return
	}

	if !opt.Quiet {
		fmt.Printf("converted %d file(s)\n", len(filenames))
		fmt.Printf("elapsed: %v\n", time.Since(t0))
	}
}

// processAsOne converts the filenames as single entity, this may be 2 images for interlace and 2 or more for animations.
// It reads the files(s) from filesystem and stores the resulting .prg.
// returns error on failure.
func processAsOne(opt *png2prg.Options, filenames ...string) error {
	opt.OutFile = png2prg.DestinationFilename(filenames[0], *opt)
	opt.CurrentGraphicsType = png2prg.StringToGraphicsType(opt.GraphicsMode)

	if altOffset {
		opt.ForceXOffset, opt.ForceYOffset = 32, 36
		if !opt.Quiet {
			fmt.Printf("using alternate x, y offset %d, %d\n", 32, 36)
			fmt.Printf("proper vice screenshot offset is %d, %d. please fix your images to convert without the -alt-offset flag.\n", 32, 35)
		}
	}

	p, err := png2prg.NewFromPath(*opt, filenames...)
	if err != nil {
		return fmt.Errorf("NewFromPath failed: %w", err)
	}
	buf := bytes.Buffer{}
	if _, err := p.WriteTo(&buf); err != nil {
		return fmt.Errorf("WriteTo failed: %w", err)
	}
	w, err := os.Create(opt.OutFile)
	if err != nil {
		return fmt.Errorf("os.Create failed: %w", err)
	}
	defer w.Close()
	n, err := io.Copy(w, &buf)
	if err != nil {
		return fmt.Errorf("WriteTo failed: %w", err)
	}
	if !opt.Quiet {
		fmt.Printf("write %d bytes to %q in %q format.\n", n, opt.OutFile, p.FinalGraphicsType)
	}
	if opt.Symbols && len(p.Symbols) > 0 {
		fn := strings.TrimSuffix(opt.OutFile, ".prg") + ".sym"
		wsym, err := os.Create(fn)
		if err != nil {
			return fmt.Errorf("os.Create failed: %w", err)
		}
		defer wsym.Close()
		if _, err = fmt.Fprintf(wsym, "gfxmode = %q\n", p.FinalGraphicsType); err != nil {
			return fmt.Errorf("write to symbols file failed: %w", err)
		}
		if _, err = p.WriteSymbolsTo(wsym); err != nil {
			return fmt.Errorf("p.WriteSymbolsTo failed: %w", err)
		}
		if !opt.Quiet {
			fmt.Printf("write %q\n", fn)
		}
	}
	return nil
}

// processInParallel processes all filenames in parallel.
// It starts the workers and feeds filenames to them for processing.
// The function returns when all jobs are finished.
func processInParallel(opt *png2prg.Options, filenames ...string) error {
	num := opt.NumWorkers
	if num > len(filenames) {
		num = len(filenames)
	}
	if opt.BruteForce && opt.NumWorkers > 2 {
		opt.NumWorkers = 2
		if !opt.Quiet {
			fmt.Printf("restricting each parallel thread to %d bruteforce workers\n", opt.NumWorkers)
		}
	}
	jobs := make(chan string, num)
	wg := &sync.WaitGroup{}
	wg.Add(num)
	for i := 1; i <= num; i++ {
		go worker(i, *opt, wg, jobs)
	}
	defer func() {
		close(jobs)
		wg.Wait()
	}()
	if !opt.Quiet {
		fmt.Printf("started %d workers\n", num)
	}

	for i, filename := range filenames {
		jobs <- filename
		if memProfile != "" && i == int(len(filenames)/2) {
			if err := writeMemProfile(memProfile); err != nil {
				return fmt.Errorf("writeMemProfile failed: %w", err)
			}
			if !opt.Quiet {
				fmt.Println("writeMemProfile done")
			}
		}
	}
	if !opt.Quiet {
		fmt.Printf("sent all %d files to %d workers\n", len(filenames), num)
	}
	return nil
}

// worker runs one worker to process incoming conversion jobs.
// The caller is expected to start 1 or more workers to process jobs in parallel.
// The caller is also expected to close(jobs) when done and wait for wg.Wait().
func worker(i int, opt png2prg.Options, wg *sync.WaitGroup, jobs <-chan string) {
	defer wg.Done()
	for filename := range jobs {
		opt := opt
		if err := processAsOne(&opt, filename); err != nil {
			log.Printf("skipping processAsOne %q failed: %v", filename, err)
		}
		if !opt.Quiet {
			fmt.Printf("worker %d converted %q to %q\n", i, filename, opt.OutFile)
		}
	}
}

func writeMemProfile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("Create failed: %w", err)
	}
	defer f.Close()
	runtime.GC()
	return pprof.WriteHeapProfile(f)
}

// expandWildcards searches all filenames for ? and * characters and manually finds matching files on the filesystem.
// It skips matching directories and does not recursively search them.
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

func initAndParseFlags() (opt png2prg.Options) {
	flag.StringVar(&cpuProfile, "cpuprofile", "", "write cpu profile to `file`")
	flag.StringVar(&memProfile, "memprofile", "", "write memory profile to `file` (only in -parallel mode)")
	flag.BoolVar(&help, "h", false, "help")
	flag.BoolVar(&help, "help", false, "help")

	flag.BoolVar(&opt.Quiet, "q", false, "quiet")
	flag.BoolVar(&opt.Quiet, "quiet", false, "quiet, only display errors")
	flag.BoolVar(&opt.Verbose, "v", false, "verbose")
	flag.BoolVar(&opt.Verbose, "verbose", false, "verbose output")
	flag.BoolVar(&opt.VeryVerbose, "vv", false, "very verbose, show memory usage map in most cases and implies -verbose")
	flag.BoolVar(&opt.Display, "d", false, "display")
	flag.BoolVar(&opt.Display, "display", false, "include displayer")
	flag.StringVar(&opt.OutFile, "o", "", "out")
	flag.StringVar(&opt.OutFile, "out", "", "specify outfile.prg, by default it changes extension to .prg")
	flag.StringVar(&opt.TargetDir, "td", "", "targetdir")
	flag.StringVar(&opt.TargetDir, "targetdir", "", "specify targetdir")
	flag.StringVar(&opt.GraphicsMode, "m", "", "mode")
	flag.StringVar(&opt.GraphicsMode, "mode", "", "force graphics mode to koala, hires, mixedcharset, sccharset, mccharset (4col), scsprites or mcsprites")
	flag.BoolVar(&opt.Interlace, "i", false, "interlace")
	flag.BoolVar(&opt.Interlace, "interlace", false, "when you supply 2 frames, specify -interlace to treat the images as such")
	flag.IntVar(&opt.D016Offset, "d016", 1, "d016offset")
	flag.IntVar(&opt.D016Offset, "d016offset", 1, "number of pixels to shift with d016 when using interlace")
	flag.StringVar(&opt.BitpairColorsString, "bpc", "", "bitpair-colors")
	flag.StringVar(&opt.BitpairColorsString, "bitpair-colors", "", "prefer these colors in 2bit space, eg 0,6,14,3")
	flag.IntVar(&opt.ForceBorderColor, "force-border-color", -1, "force border color")

	flag.BoolVar(&opt.BruteForce, "bf", false, "brute-force")
	flag.BoolVar(&opt.BruteForce, "brute-force", false, "brute force bitpair-colors")
	flag.BoolVar(&opt.NoGuess, "ng", false, "no-guess")
	flag.BoolVar(&opt.NoGuess, "no-guess", false, "do not guess preferred bitpair-colors")
	flag.BoolVar(&opt.NoPackChars, "np", false, "no-pack")
	flag.BoolVar(&opt.NoPackChars, "no-pack", false, "do not pack chars (only for sc/mc charset)")
	flag.BoolVar(&opt.NoPackEmptyChar, "npe", false, "no-pack-empty")
	flag.BoolVar(&opt.NoPackEmptyChar, "no-pack-empty", false, "do not optimize packing empty chars (only for mc/mixed/ecm charset)")
	flag.BoolVar(&opt.ForcePackEmptyChar, "fpe", false, "force-pack-empty")
	flag.BoolVar(&opt.ForcePackEmptyChar, "force-pack-empty", false, "optimize packing empty chars (only for sccharset)")
	flag.BoolVar(&opt.NoPrevCharColors, "npcc", false, "no-prev-char-colors")
	flag.BoolVar(&opt.NoPrevCharColors, "no-prev-char-colors", false, "do not look at the previous char's bitpair-colors, in some cases this optimization causes worse pack results")
	flag.BoolVar(&opt.NoBitpairCounters, "nbc", false, "no-bitpair-counters")
	flag.BoolVar(&opt.NoBitpairCounters, "no-bitpair-counters", false, "do not use c64color bitpar counters optimization")
	flag.BoolVar(&opt.NoCrunch, "nc", false, "no-crunch")
	flag.BoolVar(&opt.NoCrunch, "no-crunch", false, "do not TSCrunch displayer")
	flag.BoolVar(&opt.Symbols, "sym", false, "symbols")
	flag.BoolVar(&opt.Symbols, "symbols", false, "export symbols to .sym")

	// flag.BoolVar(&opt.AlternativeFade, "alt-fade", false, "use alternative (less memory hungry) fade for animation displayers.")
	flag.BoolVar(&opt.NoFade, "nf", false, "no-fade")
	flag.BoolVar(&opt.NoFade, "no-fade", false, "do not use fade in/out for animation displayers and free up a lot of memory")
	flag.StringVar(&opt.IncludeSID, "sid", "", "include .sid in displayer (see -help for free memory locations)")
	flag.BoolVar(&opt.NoAnimation, "na", false, "no-anim")
	flag.BoolVar(&opt.NoAnimation, "no-anim", false, "disable charset animations and store frames as separate screens")
	flag.IntVar(&opt.FrameDelay, "frame-delay", 6, "frames to wait before displaying next animation frame")
	flag.IntVar(&opt.WaitSeconds, "wait-seconds", 0, "seconds to wait before animation starts")
	w := int(runtime.NumCPU() / 2)
	if w < 1 {
		w = 1
	}
	flag.IntVar(&opt.NumWorkers, "w", w, "workers")
	flag.IntVar(&opt.NumWorkers, "workers", w, "number of concurrent workers in -parallel or -brute-force mode")
	flag.BoolVar(&parallel, "p", false, "parallel")
	flag.BoolVar(&parallel, "parallel", false, "run number of workers in parallel for fast conversion, treat each image as a standalone, not to be used for animations")
	flag.BoolVar(&altOffset, "ao", false, "alt-offset")
	flag.BoolVar(&altOffset, "alt-offset", false, "use alternate screenshot offset with x,y = 32,36")

	flag.BoolVar(&opt.Trd, "trd", false, "has side effect of enforcing screenram bitpair colors in level area")

	flag.Parse()
	if opt.VeryVerbose {
		opt.Verbose = true
	}
	return opt
}
