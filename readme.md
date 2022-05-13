# PNG2PRG 1.0 by Burglar

Png2prg converts a 320x200 image (png/gif/jpeg) to a c64 hires or
multicolor bitmap, charset or sprites. It will find the best matching palette
and backgroundcolor automatically, no need to modify your source images or
configure a palette.
Vice screenshots with default borders (384x272) are automatically cropped.
Images in sprite dimensions will be converted to sprites.

The resulting .prg includes the 2-byte start address and optional displayer.
The displayer for koala and hires includes fullscreen fade-in/out and
optionally a .sid tune.

This tool can be used in all buildchains on most platforms.

## What it is *not*

Png2prg is not a tool to wire fullcolor images. It needs input images to
already be compliant with c64 color and size restrictions.
In verbose mode (-v) it outputs locations of color clashes, if any.

## Supported Graphics Modes

    koala:     multicolor bitmap (max 4 colors per char)
    hires:     singlecolor bitmap (max 2 colors per char)
    mccharset: multicolor charset (max 4 colors)
    sccharset: singlecolor charset (max 2 colors)
    mcsprites: multicolor sprites (max 4 colors)
    scsprites: singlecolor sprites (max 2 colors)

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

## Single or Multicolor Charset

Currently only images with max 4 colors can be converted into a charset.
Support for individual d800 colors and mixed single/multicolor chars may be
added in a future release, if the need arises.

By default charsets are packed, they only contain unique characaters.
If you do not want charpacking, eg for a 1x1 charset, please use -no-pack

    Charset:   $2000-$27ff
    Screen:    $2800-$2be7
    CharColor: $2be8
    D021:      $2be9
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

It's also possible to explicitly skip certain bitpairs preferences with -1:

    ./png2prg -bitpair-colors 0,-1,-1,3 image.png

## Sprite Animation

Each frame will be concatenated in the output .prg.
You can supply an animated .gif or multiple image files.

## Bitmap Animation (only koala)

If multiple files are added, they are treated as animation frames.
You can also supply an animated .gif.
The first image will be exported and each frame as a separate .prg,
containing the modified characters.

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
      .byte $01                  // colorram color
      ...                        // next char(s)

    ...          // next chunks
    .byte 0      // end of frame
    ...          // next frame(s)
    .byte $ff    // end of all frames

## Displayer

The -d or -display flag will link displayer code infront of the picture.
By default it will also crunch the resulting file with Antonio Savona's
[TSCrunch](https://github.com/tonysavon/TSCrunch/) with a couple of changes in my own [fork](https://github.com/staD020/TSCrunch/).

For koala and hires, the displayer also supports adding a .sid. Multispeed sids
are supported, as long as the sid initializes the CIA timers correctly.
You can use sids located from $0d00-$1fff or $9000+.
If needed, you can relocate most sids using lft's [sidreloc](http://www.linusakesson.net/software/sidreloc/index.php).

## Changes for version 1.0

 - Added fullscreen fade in/out to koala and hires displayers.
 - Added optional .sid support for koala and hires displayers.
 - Added optional crunching for all displayers using TSCrunch.

## Credits

Png2prg was written by Burglar, using the following third-party libraries:

[TSCrunch 1.3](https://github.com/tonysavon/TSCrunch/) by Antonio Savona for optional crunching when exporting
an image with a displayer.

[Colfade Doc](https://csdb.dk/release/?id=132276) by Veto for the colfade
tables used in the koala and hires displayers.

[Kick Assembler](http://www.theweb.dk/KickAssembler/) by Slammer to compile the displayers.

## Options

```
  -bitpair-colors string
      prefer these colors in 2bit space, eg 0,6,14,3
  -bpc string
      bitpair-colors
  -d  display
  -display
      include displayer
  -h  help
  -help
      help
  -m string
      mode
  -mode string
      force graphics mode to koala, hires, mccharset, sccharset, scsprites or mcsprites
  -nc
      no-crunch
  -ng
      no-guess
  -no-crunch
      do not TSCrunch koala/hires displayer
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
  -q  quiet
  -quiet
      quiet, only display errors
  -sid string
      include .sid (0x0d00-0x1fff or 0x9000+) in displayer
  -targetdir string
      specify targetdir
  -td string
      targetdir
  -v  verbose
  -verbose
      verbose output
```
