# PNG2PRG 1.4.1-dev by burg

Png2prg converts a 320x200 image (png/gif/jpeg) to a c64 hires or
multicolor bitmap, charset or sprites prg. It will find the best matching
palette and backgroundcolor automatically, no need to modify your source images
or configure a palette.
Vice screenshots with default borders (384x272) are automatically cropped.
Vice's main screen's offset is at x=32, y=35.
Images in sprite dimensions will be converted to sprites.

The resulting .prg includes the 2-byte start address and optional displayer.
The displayers for koala, hires, mcibitmap and animations include fullscreen
fade-in/out and optionally a .sid tune.

This tool can be used in all buildchains on most platforms.

## What it is *not*

Png2prg is not a tool to wire fullcolor images. It needs input images to
already be compliant with c64 color and size restrictions.
In verbose mode (-v) it outputs locations of color clashes, if any.

If you do need to wire fullcolor images, check out Youth's [Retropixels](https://www.micheldebree.nl/retropixels/).

## Supported Graphics Modes

    koala:     multicolor bitmap (max 4 colors per char)
    hires:     singlecolor bitmap (max 2 colors per char)
    mccharset: multicolor charset (max 4 colors)
    sccharset: singlecolor charset (max 2 colors)
    mcsprites: multicolor sprites (max 4 colors)
    scsprites: singlecolor sprites (max 2 colors)
    mcibitmap: 320x200 multicolor interlace bitmap (max 4 colors per char/frame)

Png2prg is mostly able to autodetect the correct graphics mode, but you can
also force a specific graphics mode with the -mode flag:

    ./png2prg -m koala image.png

## Koala or Hires Bitmap

    Bitmap: $2000 - $3f3f
    Screen: $3f40 - $4327
    D020:   $4328         (singlecolor only)
    D800:   $4328 - $470f (multicolor only)
    D021:   $4710         (multicolor only, low-nibble)
    D020:   $4710         (multicolor only, high-nibble)

## Multicolor Interlace Bitmap

Experimental interlace support, you can supply one 320x200 multicolor
image with max 4 colors per 8x8 pixel char per frame of which at least
2 are shared (the D021 and D800 colors).

Or supply both frames in regular koala specs (-interlace flag required).
When making screenshots in vice, please disable the d016 pixel shift manually.

    ./png2prg -i testdata/madonna/frame_0.png testdata/madonna/frame_1.png

### Drazlace (shared screenram and colorram for both frames)

    ./png2prg testdata/madonna/cjam_pure_madonna.png

    D800:    $5800 - $5be7
    Screen:  $5c00 - $5fe7
    Bitmap1: $6000 - $7f3f
    D021:    $7f40         (low-nibble)
    D020:    $7f40         (high-nibble)
    D016Offset: $7f42
    Bitmap2: $8000 - $9f3f

### Multicolor Interlace (shared colorram, true paint .mci format)

    ./png2prg -i -d016 1 testdata/mcinterlace/parriot?.png

    Screen1: $9c00 - $9fe7
    D021:    $9fe8         (low-nibble)
    D020:    $9fe8         (high-nibble)
    D016Offset: $9fe9
    Bitmap1: $a000 - $bf3f
    Bitmap2: $c000 - $df3f
    Screen2: $e000 - $e3e7
    D800:    $e400 - $e7e7

## Single or Multicolor Charset

Currently only images with max 4 colors can be converted into a charset.
Support for individual d800 colors and mixed single/multicolor chars may be
added in a future release, if the need arises.

By default charsets are packed, they only contain unique characters.
If you do not want charpacking, eg for a 1x1 charset, please use -no-pack

    Charset:   $2000-$27ff
    Screen:    $2800-$2be7
    CharColor: $2be8
    D021:      $2be9
    D020:      $2bea       (singlecolor only)
    D022:      $2bea       (multicolor only)
    D023:      $2beb       (multicolor only)
    D020:      $2bec       (multicolor only)

## Single or Multicolor Sprites

If the source image size is a multiple of a 24x21 pixel sprite,
the image is considered to contain sprites.

The image will be converted left to right, top to bottom.

    Sprite 1: $2000-$203f
    Sprite 2: $2040-$207f
    ...

## Bitpair Colors

By default, png2prg guesses bitpair colors by itself. In most cases you
don't need to configure anything. It will provide a mostly normalized image
which should yield good pack results, but your miles may vary.

To give you more control, you can force/prefer a specific bitpair
color-order. Use c64 colors, so 0 for black, 1 for white, 2 for red, etc.

The following example will force background color 0 for bitpair 00 and
prefer colors 6,14,3 for bitpairs 01,10,11:

    ./png2prg -bitpair-colors 0,6,14,3 image.png

It's also possible to explicitly skip certain bitpair preferences with -1:

    ./png2prg -bitpair-colors 0,-1,-1,3 image.png

## Sprite Animation

Each frame will be concatenated in the output .prg.
You can supply an animated .gif or multiple image files.

## Bitmap Animation (only koala and hires)

If multiple files are added, they are treated as animation frames.
You can also supply an animated .gif.
The first image will be exported with all framedata appended.
Koala animation frames start at $4711, hires at $4329.

The frame files are following this format.
Each frame consists of 1 or more chunks. A chunk looks like this:

    .byte $03    // number of chars in this chunk
                 // $00 marks end of frame
                 // $ff marks end of all frames
    .word bitmap // bitmap address of this chunk (the high byte is <$20)
    .word screen // screenram address (the high byte is <$04)

    For each char in this chunk:

      .byte 0,31,15,7,8,34,0,128 // pixels
      .byte $64                  // screenram colors
      .byte $01                  // colorram color (koala only)
      ...                        // next char(s)

    ...          // next chunks
    .byte 0      // end of frame
    ...          // next frame(s)
    .byte $ff    // end of all frames

## Displayer

The -d or -display flag will link displayer code infront of the picture.
By default it will also crunch the resulting file with Antonio Savona's
[TSCrunch](https://github.com/tonysavon/TSCrunch/) with a couple of changes in my own [fork](https://github.com/staD020/TSCrunch/).

For hires, koala and koala-anim the displayer also supports adding a .sid.
Multispeed sids are supported as long as the .sid initializes the CIA timers
correctly.

You can use sids located from $0e00-$1fff or $e000+ in the displayers.
More areas may be free depending on graphics type.
A memory usage map is shown on error and in -vv (very verbose) mode.

If needed, you can relocate most sids using lft's [sidreloc](http://www.linusakesson.net/software/sidreloc/index.php).

Zeropages $08-$0f are used in the animation displayers, while none are used
in hires/koala displayers, increasing sid compatibility.

## Examples

This release contains examples with all assets included for you to test with.
Also included are the assets of [Évoluer](https://csdb.dk/release/?id=220170) by The Sarge and Flotsam.

## Install from source

First [install Go 1.20 or higher](https://go.dev/dl/), then:

    go install github.com/staD020/png2prg/cmd/png2prg@latest

Png2prg was built on Linux, building on Mac should work out of the box.
For Windows, try out Windows Subsystem Linux (WSL), works pretty well.
However, natively building on Windows should be easy enough, look at
Compiling without Make below.

The compiled displayer prgs are included in the repo to ease building
and importing png2prg as a library. Java is only required to build
the displayers with KickAssembler (included in the repo).

### Compiling with Make (recommended)

    make -j

Build for all common targets:

    make all -j

### Compiling without Make

    go build ./cmd/png2prg

## Install and use as library

In your Go project's path, go get the library:

    go get github.com/staD020/png2prg

In essence png2prg implements the [io.WriterTo](https://pkg.go.dev/io#WriterTo) interface.
Typical usage could look like below. A more complex example can be found
in the [source](https://github.com/staD020/png2prg/blob/master/cmd/png2prg/main.go) of the cli tool.

```go
import (
	"fmt"
	"io"
	"github.com/staD020/png2prg"
)

func convertPNG(w io.Writer, png io.Reader) (int64, error) {
	p, err := png2prg.New(png2prg.Options{}, png)
	if err != nil {
		return 0, fmt.Errorf("png2prg.New failed: %w", err)
	}
	return p.WriteTo(w)
}
```

## Changes for version 1.4.1

 - Fix -force-border-color for singlecolor charset (thanks Raistlin).

## Changes for version 1.4

 - Support for even more far-out palette ranges (thanks Perplex).
 - Now throws an error if the palette can't be detected properly, this should
   never happen. Please let me know if you run into this error.
 - Separated library and cli tool.
 - Library supports the standard [io.Reader](https://pkg.go.dev/io#Reader) and [io.Writer](https://pkg.go.dev/io#Writer) interfaces.
 - Patched [TSCrunch](https://github.com/staD020/TSCrunch/) further to increase crunch speed and use less memory.
 - Added -parallel and -worker flags to treat each input file as standalone
   and convert all files in parallel. Gifs with multiple frames are still
   treated as animations.
 - Stop relying on .gif filename extension, detect it.
 - Add -alt-offset flag to force screenshot offset 32, 36), used by a few
   graphicians. Though, please switch to the correct 32, 35.
 - Add -symbols flag to write symbols to a .sym file.
 - Interlace support for mcibitmap (drazlace and truepaint).
 - Bugfix: allow blank images input (thanks Spider-J).
 - Allow colors not present in the image as -bitpair-colors (thanks Map).

## Changes for version 1.2

 - Added displayer for koala animations.
 - Added displayer for hires animations.
 - Added -frame-delay flag for animation displayers.
 - Added -wait-seconds flag for animation displayers.
 - Fixed bug in koala/hires displayers not allowing sids to overlap $c000-$c7ff.
 - Expanding wildcards: using pic??.png or pic*.png now also works on Windows.
 - Set bank via $dd00 in displayers.

## Changes for version 1.0

 - Added fullscreen fade in/out to koala and hires displayers.
 - Added optional .sid support for koala and hires displayers.
 - Added optional crunching for all displayers using TSCrunch.

## Credits

Png2prg was created by Burglar, using the following third-party libraries:

[TSCrunch 1.3](https://github.com/tonysavon/TSCrunch/) by Antonio Savona for optional crunching when exporting
an image with a displayer.

[Colfade Doc](https://csdb.dk/release/?id=132276) by Veto for the colfade
tables used in the koala and hires displayers.

[Kick Assembler](http://www.theweb.dk/KickAssembler/) by Slammer to compile the displayers.

[Go](https://go.dev/) by The Go Authors is the programming language used to create png2prg.

## Options

```
  -alt-offset
    	use alternate screenshot offset with x,y = 32,36
  -ao
    	alt-offset
  -bitpair-colors string
    	prefer these colors in 2bit space, eg 0,6,14,3
  -bpc string
    	bitpair-colors
  -cpuprofile file
    	write cpu profile to file
  -d	display
  -d016 int
    	d016offset (default 1)
  -d016offset int
    	number of pixels to shift with d016 when using interlace (default 1)
  -display
    	include displayer
  -force-border-color int
    	force border color (default -1)
  -frame-delay int
    	frames to wait before displaying next animation frame (default 6)
  -h	help
  -help
    	help
  -i	interlace
  -interlace
    	when you supply 2 frames, specify -interlace to treat the images as such
  -m string
    	mode
  -memprofile file
    	write memory profile to file (only in -parallel mode)
  -mode string
    	force graphics mode to koala, hires, mccharset, sccharset, scsprites or mcsprites
  -nc
    	no-crunch
  -ng
    	no-guess
  -no-crunch
    	do not TSCrunch displayer
  -no-guess
    	do not guess preferred bitpair-colors
  -no-pack
    	do not pack chars (only for sc/mc charset)
  -np
    	no-pack
  -o string
    	out
  -out string
    	specify outfile.prg, by default it changes extension to .prg
  -p	parallel
  -parallel
    	run number of workers in parallel for fast conversion, treat each image as a standalone, not to be used for animations
  -q	quiet
  -quiet
    	quiet, only display errors
  -sid string
    	include .sid in displayer (see -help for free memory locations)
  -sym
    	symbols
  -symbols
    	export symbols to .sym
  -targetdir string
    	specify targetdir
  -td string
    	targetdir
  -v	verbose
  -verbose
    	verbose output
  -vv
    	very verbose, show memory usage map in most cases and implies -verbose
  -w int
    	workers (default 12)
  -wait-seconds int
    	seconds to wait before animation starts
  -workers int
    	number of concurrent workers in parallel mode (default 12)
```

