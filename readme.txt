png2prg 0.4 by burglar
usage: ./png2prg [-help -h -d -q -v -force-bg-col 0 -force-charcol 5 -o outfile.prg -td testdata] FILE [FILE..]

Png2prg converts a 320x200 image (png/gif/jpeg) to a c64 single- or multicolor
bitmap or charset. It will find the best matching palette and backgroundcolor
automatically, no need to modify your source images or configure a palette.
Vice screenshots with default borders (384x272) are automatically cropped.

The resulting .prg includes the 2-byte start address and optional displayer.

This tool can be used in all buildchains on most platforms.

Single or MultiColor Bitmap:

Png2prg automatically detects single color bitmaps based on the maximum
amount of colors per character in the bitmap.

  Bitmap: $2000 - $3f3f
  Screen: $3f40 - $4327
  D800:   $4328 - $470f (multicolor only)
  D021:   $4710         (multicolor only)

Background Color:

To give you more control, you can force a specific background color for
multicolor pictures, by specifying -force-bg-col 0 for black, 1 for white, 2 for red, etc.

Single or MultiColor Charset:

Currently only pictures with max 4 colors can be converted to charset.
You can use -force-bg-col and -force-char-col to control colors.

MultiColor charsets are packed, they only contain unique characaters.
SingleColor charsets are *not* packed, primary use is 1x1 charsets.

  Charset:   $2000-$27ff
  Screen:    $2800-$2be7 (multicolor only)
  D021:      $2be8       (multicolor only)
  CharColor: $2be9       (multicolor only)
  D022:      $2bea       (multicolor only)
  D023:      $2beb       (multicolor only)

Animation (MultiColor Bitmap Only for now):

If multiple files are added, they are treated as animation frames.
The base image will be exported and a separate .prg per frame.

The frame files are following this format.
Each frame consists of 1 or more chunks. A chunk looks like this:

  .byte $03    // number of chars in this chunk
               // $00 marks end of frame
               // $ff marks end of all frames
  .word bitmap // bitmap address of this chunk (the high byte is <$20)
  .word screen // screenram address (the high byte is <$04)

  For each char in this chunk:

    .byte 0,0,15,7,8,8,0,0 // pixels
    .byte $64              // screenram colors
    .byte $01              // colorram color
    ...                    // next chars

  ...          // next chunks
  .byte 0      // end of frame
  ...          // next frames
  .byte $ff    // end of all frames

Options:

  -d	display
  -display
    	include displayer
  -force-bgcol int
    	force background color -1: off 0: black 1: white 2: red, etc (default -1)
  -force-charcol int
    	force multicolor charset d800 color -1: off 0: black 1: white 2: red, etc (default -1)
  -h	help
  -help
    	help
  -o string
    	out
  -out string
    	specify outfile.prg, by default it changes extension to .prg
  -q	quiet
  -quiet
    	quiet
  -targetdir string
    	specify targetdir
  -td string
    	targetdir
  -v	verbose
  -verbose
    	verbose
