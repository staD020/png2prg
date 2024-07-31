.const DEBUG         = false
.const GENDEBUG      = false
.const MUSICDEBUG    = false
.const LOOP          = false
.const fade_speed    = 2
.const steps         = 16
.const charset       = $4000
.const screenram     = $4800
.const colorram      = $d800
.const colorram_src  = $4c00

.const zp_start = $0334		// displaycode will be shorter if this is <$f9, but we prefer zeropage-less code to allow most sids to play.
.const zp_screen_lo = zp_start + 0
.const zp_screen_hi = zp_start + 1
.const zp_src_screen_lo = zp_start + 2
.const zp_src_screen_hi = zp_start + 3

.import source "lib.asm"

.pc = $0801 "basic upstart"
		.byte <basicend, >basicend, <year(), >year(), $9e
		.text toIntString(start)
		.text " PNG2PRG " + versionString()
basicend:
		.byte 0, 0, 0
.pc = $0819 "music_startsong"
music_startsong:
		.byte 0
.pc = * "music_init"
music_init:
		jmp rrts
.pc = * "music_play"
music_play:
		jmp rrts

.pc = basicsys() "start"
start:
		sei
		jsr $e544
		lda #$35
		sta $01

		ldy #7
!loop:
		ldx #fade_speed
	!:	jsr vblank
		dex
		bne !-

		lda $d020
		and #$0f
		tax
		lda t_easyfade,x
		sta $d020
		lda $d021
		and #$0f
		tax
		lda t_easyfade,x
		sta $d021
		dey
		bne !loop-
		sta $d011
!loop:
	.for (var i=0; i<4; i++) {
		sta colorram+i*$100,y
	}
		iny
		bne !loop-

		// default pal 50 hz: $4cc7
		lda #$c7
		sta $dc04
		lda #$4c
		sta $dc05

		lax music_startsong
		tay
		jsr music_init
		lda #<irq
		sta $fffe
		lda #>irq
		sta $ffff

		lda #$80
	!:	cmp $d012
		bne !-
	.if (MUSICDEBUG) {
		ldx #5
	!:	dex
		bne !-
	}
		lda #%00010001
		sta $dc0e
		cli

		:setBank(charset)
		lda #toD018(screenram, charset)
		sta $d018
		lda #$c8
		sta $d016
		lda #$1b
		sta $d011

loop:

		// screen is black, show new screen
		jsr vblank
		inc framecount
		ldx framecount
		cpx $3fea
		bne !nextframe+
		ldx #0
		stx framecount
		lda #>colorram_src
		sta smc_colram+2
	!nextframe:
		ldy #4
		ldx #0
	!:
smc_colram:
		lda colorram_src,x
smc_d800:
		sta $d800,x
		inx
		bne !-
		inc smc_colram+2
		inc smc_d800+2
		dey
		bne !-
		lda #>colorram
		sta smc_d800+2
		inc smc_colram+2
		inc smc_colram+2
		inc smc_colram+2
		inc smc_colram+2

		lda #toD018(screenram, charset)
		ldx framecount
		beq !skip+
		lda $d018
		clc
		adc #$20
	!skip:
		sta $d018
		// image is being displayed
		lda #$ef
	!:	cmp $dc01
		bne !-
	!:	cmp $dc01
		beq !-
		jmp loop
.pc = * "vblank"
vblank:
		:vblank()
rrts:	rts
framecount: .byte 255
// --------------------------------
.pc = * "irq"
irq:
		pha
		txa
		pha
		tya
		pha
		.if (MUSICDEBUG) dec $d020
		jsr music_play
		.if (MUSICDEBUG) inc $d020
		lda $dc0d
		pla
		tay
		pla
		tax
		pla
		rti
// --------------------------------
.pc = * "t_easyfade"
t_easyfade:
		.byte $00,$0d,$09,$0c,$02,$08,$00,$0f
		.byte $02,$00,$08,$09,$04,$03,$04,$05
// ------------------------------
.pc = charset "charset" virtual
.fill $800,0
// ------------------------------
