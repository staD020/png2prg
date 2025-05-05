package png2prg

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"sort"
	"strconv"
	"sync"

	"github.com/staD020/TSCrunch"
)

type bruteResult struct {
	bpc               string
	noprevcharcols    bool
	nobitpaircounters bool
	length            int
}

// BruteForceBitpairColors bruteforces all possible bitpair color combinations.
// Sets img.bpc to the best result.
func (c *Converter) BruteForceBitpairColors(gfxtype GraphicsType, maxColors int) error {
	if maxColors > 4 {
		return fmt.Errorf("maxColors has a max of 4, but it is %d", maxColors)
	}
	if gfxtype == unknownGraphicsType {
		return fmt.Errorf("BruteForceBitpairColors failed: unknownGraphicsType")
	}
	origOpt := c.opt
	num := c.opt.NumWorkers
	jobs := make(chan sourceImage, num)
	result := make(chan bruteResult, num)
	wg := &sync.WaitGroup{}
	wg.Add(num)
	for i := 1; i <= num; i++ {
		go c.bruteWorker(i, wg, jobs, result)
	}
	out := []bruteResult{}
	go func() {
		for v := range result {
			out = append(out, v)
		}
		wg.Done()
	}()
	if !c.opt.Quiet {
		fmt.Printf("started %d brute-force workers\n", num)
	}

	colors := c.images[0].SortedColors()
	const permuteDepth = 8
	if len(colors) > permuteDepth {
		colors = colors[0:permuteDepth]
	}
	if c.opt.Verbose {
		log.Printf("bruteforce colors: %v", colors)
	}

	count := 0
	total := 0
	done := map[[4]C64Color]bool{}
	for p := make([]int, len(colors)); p[0] < len(p); PermuteNext(p) {
		count++
		s := Permutation(colors, p)
		if len(s) > maxColors {
			s = s[:maxColors]
		}
		tmp := [4]C64Color{}
		for i := range tmp {
			if i < len(s) {
				tmp[i] = s[i].C64Color
			}
		}
		if _, ok := done[tmp]; ok {
			continue
		}
		done[tmp] = true

		if gfxtype == multiColorBitmap || gfxtype == singleColorCharset || gfxtype == multiColorCharset || gfxtype == mixedCharset {
			// skip impossible bgcolors
			bgok := false
			for _, col := range c.images[0].bgCandidates {
				if col.C64Color == s[0].C64Color {
					bgok = true
				}
			}
			if !bgok {
				continue
			}
		}
		if gfxtype == multiColorCharset || gfxtype == mixedCharset {
			// skip impossible d800 colors
			if len(s) > 3 {
				if s[3].C64Color > 7 {
					continue
				}
			}
		}

		bitpaircols := ""
		for i, col := range s {
			bitpaircols += strconv.Itoa(int(col.C64Color))
			if i < len(s)-1 {
				bitpaircols += ","
			}
		}

		opt := c.opt
		opt.GraphicsMode = gfxtype.String()
		opt.CurrentGraphicsType = gfxtype
		opt.BitpairColorsString = bitpaircols
		opt.Display = false
		opt.NoCrunch = true
		opt.Verbose = false
		opt.VeryVerbose = false
		opt.Quiet = true
		// prefilled NewSourceImage, no need to redo the same work
		img := sourceImage{
			sourceFilename: fmt.Sprintf("png2prg_bf_%d", count),
			opt:            opt,
			image:          c.images[0].image,
			p:              c.images[0].p,
			hiresPixels:    c.images[0].hiresPixels,
			graphicsType:   c.images[0].graphicsType,
			charColors:     c.images[0].charColors,
			sumColors:      c.images[0].sumColors,
		}
		if err := img.checkBounds(); err != nil {
			if c.opt.Verbose {
				log.Printf("skipping permutation %q because img.checkBounds failed: %v", bitpaircols, err)
			}
			continue
		}
		if err := img.setPreferredBitpairColors(opt.BitpairColorsString, opt.BitpairColorsString2); err != nil {
			if c.opt.Verbose {
				log.Printf("skipping permutation %q because setPreferredBitpairColors failed: %v", bitpaircols, err)
			}
			continue
		}
		jobs <- img
		total++
		if !origOpt.Quiet && total%10 == 0 {
			fmt.Print(".")
		}
	}
	close(jobs)
	wg.Wait()
	wg.Add(1)
	close(result)
	wg.Wait()
	c.opt = origOpt
	if !c.opt.Quiet {
		fmt.Println()
	}
	sort.Slice(out, func(i, j int) bool { return out[i].length < out[j].length })
	if !c.opt.Quiet && len(out) > 5 {
		threshold := out[0].length + 5
		d := 0
		for i := range out {
			if i > 0 && out[i].length < threshold && d < 10 {
				d++
				fmt.Printf("you may want to manually try -bpc %s (%d bytes)\n", out[i].bpc, out[i].length)
			}
		}
	}
	if c.opt.Verbose {
		for i := range out {
			extra := ""
			if out[i].noprevcharcols {
				extra = "-npcc"
			}
			if out[i].nobitpaircounters {
				if extra != "" {
					extra += " "
				}
				extra += "-nbc"
			}

			log.Printf("%d: -bpc %s %s (length: %d)", i, out[i].bpc, extra, out[i].length)
			if !c.opt.VeryVerbose && i == 9 {
				break
			}
		}
		log.Printf("-brute-force mode tried %d permutations, %d attempts and got %d results, use -vv to display all", count, total, len(out))
	}
	if len(out) == 0 {
		return fmt.Errorf("no color options found to brute-force")
	}
	if !c.opt.Quiet {
		fmt.Printf("\nbrute-force winner %q -bpc %s (%d bytes)\n\n", c.opt.OutFile, out[0].bpc, out[0].length)
	}
	c.opt.BitpairColorsString = out[0].bpc
	c.opt.NoPrevCharColors = out[0].noprevcharcols
	c.opt.NoBitpairCounters = out[0].nobitpaircounters
	c.images[0].opt.BitpairColorsString = out[0].bpc
	var bpc BPColors
	var err error
	if bpc, err = c.images[0].p.ParseBPC(c.opt.BitpairColorsString); err != nil {
		return fmt.Errorf("p.ParseBPC failed: %w", err)
	}
	if len(bpc) > 0 {
		if bpc[0] != nil {
			c.images[0].bg = *bpc[0]
		}
		c.images[0].ecmColors = bpc.Colors()
	}
	return nil
}

// bruteWorker is launched to receive sourceImages, process and crunch them and deliver results to the result channel.
func (c *Converter) bruteWorker(i int, wg *sync.WaitGroup, jobs <-chan sourceImage, result chan bruteResult) {
	defer wg.Done()
NEXTJOB:
	for img := range jobs {
		if img.opt.Verbose {
			fmt.Printf("worker %d received: -bpc %s\n", i, img.bpc)
		}
		var err error
		if err = img.analyze(); err != nil {
			if img.opt.VeryVerbose {
				log.Printf("img.analyze %q failed: %v", img.sourceFilename, err)
				continue NEXTJOB
			}
		}
		var wt io.WriterTo
		switch img.graphicsType {
		case multiColorBitmap:
			if wt, err = img.Koala(); err != nil {
				if img.opt.VeryVerbose {
					log.Printf("img.Koala %q failed: %v", img.sourceFilename, err)
				}
				continue NEXTJOB
			}
		case singleColorBitmap:
			if wt, err = img.Hires(); err != nil {
				if img.opt.VeryVerbose {
					log.Printf("img.Hires %q failed: %v", img.sourceFilename, err)
				}
				continue NEXTJOB
			}
		case singleColorCharset:
			if wt, err = img.SingleColorCharset(nil); err != nil {
				if img.opt.VeryVerbose {
					log.Printf("worker %d img.SingleColorCharset %q failed: %v", i, img.sourceFilename, err)
				}
				continue NEXTJOB
			}
		case multiColorCharset:
			if wt, err = img.MultiColorCharset(nil); err != nil {
				if img.opt.VeryVerbose {
					log.Printf("img.MultiColorCharset %q failed: %v", img.sourceFilename, err)
				}
				continue NEXTJOB
			}
		case ecmCharset:
			if wt, err = img.ECMCharset(nil); err != nil {
				if img.opt.VeryVerbose {
					log.Printf("img.ECMCharset %q failed: %v", img.sourceFilename, err)
				}
				continue NEXTJOB
			}
		case mixedCharset:
			if len(img.bpc) > 3 {
				if img.bpc[3].C64Color > 7 {
					continue NEXTJOB
				}
			}
			if wt, err = img.MixedCharset(nil); err != nil {
				if img.opt.VeryVerbose {
					log.Printf("img.MixedCharset %q failed: %v", img.sourceFilename, err)
				}
				continue NEXTJOB
			}
		default:
			log.Printf("skip unsupported bruteforce graphicsType: %s", img.graphicsType)
			continue NEXTJOB
		}
		buf := bytes.Buffer{}
		if _, err = wt.WriteTo(&buf); err != nil {
			panic(err)
		}

		tscopt := TSCrunch.Options{PRG: true, QUIET: true, Fast: true}
		tsc, err := TSCrunch.New(tscopt, &buf)
		if err != nil {
			panic(err)
		}
		compressed := bytes.Buffer{}
		_, err = tsc.WriteTo(&compressed)
		if err != nil {
			panic(err)
		}
		result <- bruteResult{
			bpc:               img.opt.BitpairColorsString,
			noprevcharcols:    img.opt.NoPrevCharColors,
			nobitpaircounters: img.opt.NoBitpairCounters,
			length:            compressed.Len(),
		}
	}
}

// SortedColors returns Colors sorted by number of chars each Color is used in.
func (img *sourceImage) SortedColors() Colors {
	type sumcol struct {
		col   C64Color
		count int
	}
	sc := []sumcol{}
	for col, count := range img.sumColors {
		if count > 0 {
			sc = append(sc, sumcol{col: C64Color(col), count: count})
		}
	}
	sort.Slice(sc, func(i, j int) bool { return sc[i].count > sc[j].count })
	result := make(Colors, len(sc))
	for i, scol := range sc {
		result[i] = img.p.FromC64NoErr(scol.col)
	}
	return result
}
