package main

import (
	"flag"
	"fmt"
	"os"
)

func printUsage() {
	fmt.Println("usage: ./png2prg [-help -h -d -q -v -bitpair-colors 0,6,14,3 -o outfile.prg -td testdata] FILE [FILE..]")
}

func printHelp() {
	fmt.Println()
	fmt.Printf("# PNG2PRG %v by Burglar\n", version)
	fmt.Println()
	fmt.Println("Png2prg converts a 320x200 image (png/gif/jpeg/apng) to a c64 hires or")
	fmt.Println("multicolor bitmap or charset. It will find the best matching palette and")
	fmt.Println("backgroundcolor automatically, no need to modify your source images or")
	fmt.Println("configure a palette.")
	fmt.Println("Vice screenshots with default borders (384x272) are automatically cropped.")
	fmt.Println("Images in sprite dimensions will be converted to sprites.")
	fmt.Println()
	fmt.Println("The resulting .prg includes the 2-byte start address and optional displayer.")
	fmt.Println()
	fmt.Println("This tool can be used in all buildchains on most platforms.")
	fmt.Println()
	fmt.Println("## What it is *not*")
	fmt.Println()
	fmt.Println("Png2prg is not a tool to wire fullcolor images. It needs input images to")
	fmt.Println("already be compliant with c64 color and size restrictions.")
	fmt.Println("In verbose mode (-v) it outputs locations of color clashes, if any.")
	fmt.Println()
	fmt.Println("## Supported Graphics Modes")
	fmt.Println()
	fmt.Println("    koala:     multicolor bitmap (max 4 colors per char)")
	fmt.Println("    hires:     singlecolor bitmap (max 2 colors per char)")
	fmt.Println("    mccharset: multicolor charset (max 4 colors)")
	fmt.Println("    sccharset: singlecolor charset (max 2 colors)")
	fmt.Println("    mcsprites: multicolor sprites (max 4 colors)")
	fmt.Println("    scsprites: singlecolor sprites (max 2 colors)")
	fmt.Println()
	fmt.Println("Png2prg is mostly able to autodetect the correct graphics mode, but you can")
	fmt.Println("also force a specific graphics mode with the -mode flag:")
	fmt.Println()
	fmt.Println("    ./png2prg -m koala image.png")
	fmt.Println()
	fmt.Println("## Koala or Hires Bitmap")
	fmt.Println()
	fmt.Println("    Bitmap: $2000 - $3f3f")
	fmt.Println("    Screen: $3f40 - $4327")
	fmt.Println("    D020:   $4328         (singlecolor only)")
	fmt.Println("    D800:   $4328 - $470f (multicolor only)")
	fmt.Println("    D021:   $4710         (multicolor only, low-nibble)")
	fmt.Println("    D020:   $4710         (multicolor only, high-nibble)")
	fmt.Println()
	fmt.Println("## Single or Multicolor Charset")
	fmt.Println()
	fmt.Println("Currently only images with max 4 colors can be converted into a charset.")
	fmt.Println("Support for individual d800 colors and mixed single/multicolor chars may be")
	fmt.Println("added in a future release, if the need arises.")
	fmt.Println()
	fmt.Println("By default charsets are packed, they only contain unique characaters.")
	fmt.Println("If you do not want charpacking, eg for a 1x1 charset, please use -no-pack")
	fmt.Println()
	fmt.Println("    Charset:   $2000-$27ff")
	fmt.Println("    Screen:    $2800-$2be7")
	fmt.Println("    CharColor: $2be8")
	fmt.Println("    D021:      $2be9")
	fmt.Println("    D022:      $2bea       (multicolor only)")
	fmt.Println("    D023:      $2beb       (multicolor only)")
	fmt.Println("    D020:      $2bec       (multicolor only)")
	fmt.Println()
	fmt.Println("## Single or Multicolor Sprites")
	fmt.Println()
	fmt.Println("If the source image size is a multiple of a 24x21 pixel sprite,")
	fmt.Println("the image is considered to contain sprites.")
	fmt.Println()
	fmt.Println("The image will be converted left to right, top to bottom.")
	fmt.Println()
	fmt.Println("    Sprite 1: $2000-$203f")
	fmt.Println("    Sprite 2: $2040-$207f")
	fmt.Println("    ...")
	fmt.Println()
	fmt.Println("## Bitpair Colors")
	fmt.Println()
	fmt.Println("By default, png2prg guesses bitpair colors by itself. In most cases you")
	fmt.Println("don't need to configure anything. It will provide a mostly normalized image")
	fmt.Println("which should yield good pack results, but your miles may vary.")
	fmt.Println()
	fmt.Println("To give you more control, you can force/prefer a specific bitpair")
	fmt.Println("color-order. Use c64 colors, so 0 for black, 1 for white, 2 for red, etc.")
	fmt.Println()
	fmt.Println("The following example will force background color 0 for bitpair 00 and")
	fmt.Println("prefer colors 6,14,3 for bitpairs 01,10,11:")
	fmt.Println()
	fmt.Println("    ./png2prg -bitpair-colors 0,6,14,3 image.png")
	fmt.Println()
	fmt.Println("It's also possible to explicitly skip certain bitpairs preferences with -1:")
	fmt.Println()
	fmt.Println("    ./png2prg -bitpair-colors 0,-1,-1,3 image.png")
	fmt.Println()
	fmt.Println("## Sprite Animation")
	fmt.Println()
	fmt.Println("Each frame will be concatenated in the output .prg.")
	fmt.Println("You can supply an animated .gif, .apng or multiple image files.")
	fmt.Println()
	fmt.Println("## Bitmap Animation (only koala)")
	fmt.Println()
	fmt.Println("If multiple files are added, they are treated as animation frames.")
	fmt.Println("You can also supply an animated .gif or .apng.")
	fmt.Println("The first image will be exported and each frame as a separate .prg,")
	fmt.Println("containing the modified characters.")
	fmt.Println()
	fmt.Println("The frame files are following this format.")
	fmt.Println("Each frame consists of 1 or more chunks. A chunk looks like this:")
	fmt.Println()
	fmt.Println("    .byte $03    // number of chars in this chunk")
	fmt.Println("                 // $00 marks end of frame")
	fmt.Println("                 // $ff marks end of all frames")
	fmt.Println("    .word bitmap // bitmap address of this chunk (the high byte is <$20)")
	fmt.Println("    .word screen // screenram address (the high byte is <$04)")
	fmt.Println()
	fmt.Println("    For each char in this chunk:")
	fmt.Println()
	fmt.Println("      .byte 0,31,15,7,8,34,0,128 // pixels")
	fmt.Println("      .byte $64                  // screenram colors")
	fmt.Println("      .byte $01                  // colorram color")
	fmt.Println("      ...                        // next char(s)")
	fmt.Println()
	fmt.Println("    ...          // next chunks")
	fmt.Println("    .byte 0      // end of frame")
	fmt.Println("    ...          // next frame(s)")
	fmt.Println("    .byte $ff    // end of all frames")
	fmt.Println()
	fmt.Println("## Displayer")
	fmt.Println()
	fmt.Println("The -d or -display flag will link displayer code infront of the picture.")
	fmt.Println("By default it will also crunch the resulting file with Antonio Savona's")
	fmt.Println("[TSCrunch](https://github.com/tonysavon/TSCrunch/).")
	fmt.Println()
	fmt.Println("For koala and hires, the displayer also supports adding a .sid. Multispeed sids")
	fmt.Println("are supported, as long as the sid initializes the CIA timers correctly.")
	fmt.Println("You can use sids located from $0d00-$1fff or $9000+.")
	fmt.Println("If needed, you can relocate most sids with lft's [sidreloc](http://www.linusakesson.net/software/sidreloc/index.php).")
	fmt.Println()
	fmt.Println("## Credits")
	fmt.Println()
	fmt.Println("Png2prg was written by Burglar, using the following third-party libraries:")
	fmt.Println()
	fmt.Println("[TSCrunch](https://github.com/tonysavon/TSCrunch/) by Antonio Savona for the")
	fmt.Println("added crunching of the displayers png2prg can generate.")
	fmt.Println()
	fmt.Println("[Colfade Doc](https://csdb.dk/release/?id=132276) by Veto for the colfade")
	fmt.Println("tables used in the koala and hires displayers.")
	fmt.Println()
	fmt.Println("[Kick Assembler](http://www.theweb.dk/KickAssembler/) by Slammer to compile the displayers.")
	fmt.Println()
	fmt.Println("[APNG enhancements](https://github.com/kettek/apng) Copyright (c) 2018")
	fmt.Println("Ketchetwahmeegwun T. Southall / kts of kettek for .apng support as input format.")
	fmt.Println()
	fmt.Println("## Options")
	fmt.Println()
	fmt.Println("```")
	flag.PrintDefaults()
	fmt.Println("```")
	fmt.Println()
	os.Exit(0)
}
