package main

import (
	"flag"
	"fmt"
	"os"
)

func printusage() {
	fmt.Println("usage: ./png2prg [-help -h -d -q -v -force-bg-col 0 -force-charcol 5 -o outfile.prg -td testdata] FILE [FILE..]")
}

func help() {
	printusage()
	fmt.Println()
	fmt.Println("Png2prg converts a 320x200 image (png/gif/jpeg) to a c64 single- or multicolor")
	fmt.Println("bitmap or charset. It will find the best matching palette and backgroundcolor")
	fmt.Println("automatically, no need to modify your source images or configure a palette.")
	fmt.Println("Vice screenshots with default borders (384x272) are automatically cropped.")
	fmt.Println()
	fmt.Println("The resulting .prg includes the 2-byte start address and optional displayer.")
	fmt.Println()
	fmt.Println("This tool can be used in all buildchains on most platforms.")
	fmt.Println()
	fmt.Println("Single or MultiColor Bitmap:")
	fmt.Println()
	fmt.Println("Png2prg automatically detects single color bitmaps based on the maximum")
	fmt.Println("amount of colors per character in the bitmap.")
	fmt.Println()
	fmt.Println("  Bitmap: $2000 - $3f3f")
	fmt.Println("  Screen: $3f40 - $4327")
	fmt.Println("  D800:   $4328 - $470f (multicolor only)")
	fmt.Println("  D021:   $4710         (multicolor only)")
	fmt.Println()
	fmt.Println("Background Color:")
	fmt.Println()
	fmt.Println("To give you more control, you can force a specific background color for")
	fmt.Println("multicolor pictures, by specifying -force-bg-col 0 for black, 1 for white, 2 for red, etc.")
	fmt.Println()
	fmt.Println("Single or MultiColor Charset:")
	fmt.Println()
	fmt.Println("Currently only pictures with max 4 colors can be converted to charset.")
	fmt.Println("You can use -force-bg-col and -force-char-col to control colors.")
	fmt.Println()
	fmt.Println("MultiColor charsets are packed, they only contain unique characaters.")
	fmt.Println("SingleColor charsets are *not* packed, primary use is 1x1 charsets.")
	fmt.Println()
	fmt.Println("  Charset:   $2000-$27ff")
	fmt.Println("  Screen:    $2800-$2be7 (multicolor only)")
	fmt.Println("  D021:      $2be8       (multicolor only)")
	fmt.Println("  CharColor: $2be9       (multicolor only)")
	fmt.Println("  D022:      $2bea       (multicolor only)")
	fmt.Println("  D023:      $2beb       (multicolor only)")
	fmt.Println()
	fmt.Println("Animation (MultiColor Bitmap Only for now):")
	fmt.Println()
	fmt.Println("If multiple files are added, they are treated as animation frames.")
	fmt.Println("The base image will be exported and a separate .prg per frame.")
	fmt.Println()
	fmt.Println("The frame files are following this format.")
	fmt.Println("Each frame consists of 1 or more chunks. A chunk looks like this:")
	fmt.Println()
	fmt.Println("  .byte $03    // number of chars in this chunk")
	fmt.Println("               // $00 marks end of frame")
	fmt.Println("               // $ff marks end of all frames")
	fmt.Println("  .word bitmap // bitmap address of this chunk (the high byte is <$20)")
	fmt.Println("  .word screen // screenram address (the high byte is <$04)")
	fmt.Println()
	fmt.Println("  For each char in this chunk:")
	fmt.Println()
	fmt.Println("    .byte 0,0,15,7,8,8,0,0 // pixels")
	fmt.Println("    .byte $64              // screenram colors")
	fmt.Println("    .byte $01              // colorram color")
	fmt.Println("    ...                    // next chars")
	fmt.Println()
	fmt.Println("  ...          // next chunks")
	fmt.Println("  .byte 0      // end of frame")
	fmt.Println("  ...          // next frames")
	fmt.Println("  .byte $ff    // end of all frames")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println()
	flag.PrintDefaults()
	fmt.Println()
	os.Exit(0)
}
