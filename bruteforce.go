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
	bpc            string
	bgcol          ColorInfo
	noprevcharcols bool
	length         int
}

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

	count := 0
	total := 0
	done := map[[4]byte]bool{}
	for p := make([]int, len(colors)); p[0] < len(p); PermuteNext(p) {
		count++
		s := Permutation(colors, p)
		if len(s) > maxColors {
			s = s[:maxColors]
		}
		if len(s) < maxColors {
			log.Printf("skipping permutation %v as it does not contain %d colors", s, maxColors)
			continue
		}
		tmp := [4]byte{}
		copy(tmp[:], s)
		if _, ok := done[tmp]; ok {
			continue
		}
		done[tmp] = true

		if gfxtype == multiColorBitmap || gfxtype == singleColorCharset || gfxtype == multiColorCharset || gfxtype == mixedCharset {
			// skip impossible bgcolors
			bgok := false
			for _, col := range c.images[0].backgroundCandidates {
				if col == s[0] {
					bgok = true
				}
			}
			if !bgok {
				continue
			}
		}
		if gfxtype == multiColorCharset || gfxtype == mixedCharset {
			// skip impossible d800 colors
			if s[3] > 7 {
				continue
			}
		}

		bitpaircols := ""
		for i, col := range s {
			bitpaircols += strconv.Itoa(int(col))
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
		img, err := NewSourceImage(opt, count, c.images[0].image)
		if err != nil {
			log.Printf("skipping permutation %s because NewSourceImage failed: %v", bitpaircols, err)
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
				fmt.Printf("you may want to manually try -bpc %s\n", out[i].bpc)
			}
		}
	}
	if c.opt.Verbose {
		for i := range out {
			npcc := ""
			if out[i].noprevcharcols {
				npcc = "-npcc"
			}
			log.Printf("%d: -bpc %s %s (length: %d)", i, out[i].bpc, npcc, out[i].length)
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
		fmt.Printf("brute-force winner %q -bpc %v\n", c.opt.OutFile, out[0].bpc)
	}
	c.opt.BitpairColorsString = out[0].bpc
	c.opt.NoPrevCharColors = out[0].noprevcharcols
	c.images[0].opt.BitpairColorsString = out[0].bpc
	c.images[0].backgroundColor = out[0].bgcol
	return nil
}

func (c *Converter) bruteWorker(i int, wg *sync.WaitGroup, jobs <-chan sourceImage, result chan bruteResult) {
	defer wg.Done()
NEXTJOB:
	for img := range jobs {
		err := img.analyze()
		if err != nil {
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
		case multiColorCharset:
			if wt, err = img.MultiColorCharset(nil); err != nil {
				if img.opt.VeryVerbose {
					log.Printf("img.MultiColorCharset %q failed: %v", img.sourceFilename, err)
				}
				continue NEXTJOB
			}
		case mixedCharset:
			if len(img.preferredBitpairColors) > 3 {
				if img.preferredBitpairColors[3] > 7 {
					continue NEXTJOB
				}
			}
			if wt, err = img.MixedCharset(); err != nil {
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
			bpc:            img.opt.BitpairColorsString,
			bgcol:          ColorInfo{ColorIndex: img.preferredBitpairColors[0], RGB: img.palette.RGB(img.preferredBitpairColors[0])},
			noprevcharcols: img.opt.NoPrevCharColors,
			length:         compressed.Len(),
		}
	}
}

func (img *sourceImage) SortedColors() []byte {
	_, _, sumColors := img.countColors()
	type sumcol struct {
		col   byte
		count int
	}
	sc := []sumcol{}
	for col, count := range sumColors {
		if count > 0 {
			sc = append(sc, sumcol{col: byte(col), count: count})
		}
	}
	sort.Slice(sc, func(i, j int) bool { return sc[i].count > sc[j].count })
	result := make([]byte, len(sc))
	for i, scol := range sc {
		result[i] = scol.col
	}
	return result
}
