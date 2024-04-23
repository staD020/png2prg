.const DEBUG         = false
.const GENDEBUG      = false
.const MUSICDEBUG    = false
.const LOOP          = false
.const fade_speed    = 2
.const steps         = 16
.const charset       = $4000
.const screenram     = $4800
.const colorram      = $d800
.const colorram_src  = $3c00

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
.pc = $0822 "start"
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
		sta $d022
		sta $d023
		sta $d024
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

		ldx #0
	!:
	.for (var i=0; i<4; i++) {
		lda colorram_src+i*$100,x
		and #%00001000
		sta colorram+i*$100,x
	}
		inx
		bne !-

		:setBank(charset)
		lda #toD018(screenram, charset)
		sta $d018
		lda #$d8
		sta $d016
		lda #$1b
		sta $d011

.pc = * "fade_loop"
fade_loop:
smc_yval:	ldy #steps-1
		.if (DEBUG) inc $d020
		jsr generate_phase_col_tables
		.if (DEBUG) dec $d020

		ldx #fade_speed
	!:	jsr vblank
		dex
		bne !-

		ldy #3
	!:	ldx colorram_src+$3e8,y
		lda t_color_fade,x
		sta $d020,y
		dey
		bpl !-

		dec smc_yval+1
		lda smc_yval+1
		cmp #$ff
		beq !done+
		cmp #(steps/2)-1
		bne fade_loop

		ldx #0
	!:
	.for (var i=0; i<4; i++) {
		lda colorram_src+i*$100,x
		sta colorram+i*$100,x
	}
		inx
		bne !-

		// image is being displayed
		lda #$ef
	!:	cmp $dc01
		bne !-

		jsr vblank
		ldx #0
	.for (var i=0; i<4; i++) {
	!:
		lda colorram_src+i*$100,x
		and #%00001000
		sta colorram+i*$100,x
		inx
		bne !-
	}
		beq fade_loop
!done:

		// screen is black, show new screen
		jsr vblank
		inc framecount
		ldx framecount
		cpx colorram_src+$3ec
		bne !+
		ldx #0
		stx framecount
		lda #toD018(screenram, charset)
		bne !skip+
	!:
		lda $d018
		clc
		adc #$10
	!skip:
		sta $d018
		jsr reset_phase
		lda #steps-1
		sta smc_yval + 1
		jmp fade_loop
.pc = * "vblank"
vblank:
		:vblank()
rrts:	rts
framecount: .byte 0
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
.pc = * "generate_phase_col_tables"
generate_phase_col_tables:
!next_step:
//		lda #<t_color_fade
//		sta smc_totpercol + 1
!loop:
		// start with color 0
		ldx #0
	!:
		// y points to hi-nibble of color x
		ldy t_col2index,x // y = 0, $10, $20, .., $f0
smc_fadepercol1:
		lda t_fadepercol
		asl
		asl
		asl
		asl
smc_fadepercol2:
		ora t_fadepercol,y
smc_totpercol:
		sta t_color_fade
		.if (GENDEBUG) sta $d020
		inc smc_totpercol + 1
		inx
		cpx #$10
		bcc !-

		lda smc_fadepercol1 + 1
		// clc not needed , carry is always set
		adc #$0f
		sta smc_fadepercol1 + 1
		bcc !loop-

		// prepare code for next phase
		inc smc_fadepercol1 + 1
		inc smc_fadepercol2 + 1
		rts
// --------------------------------
.pc = * "reset_phase"
reset_phase:
		lda #<t_fadepercol
		sta smc_fadepercol1 + 1
		sta smc_fadepercol2 + 1
		rts
// ------------------------------
.pc = * "t_col2index"
t_col2index:
		.fill $10, i*$10
// ------------------------------
.pc = * "t_easyfade"
t_easyfade:
		.byte $00,$0d,$09,$0c,$02,$08,$00,$0f
		.byte $02,$00,$08,$09,$04,$03,$04,$05
// ------------------------------
.align $100
.pc = * "t_fadepercol"
t_fadepercol:
:colorfade_table()
// ------------------------------
.align $100
.pc = * "t_color_fade"
t_color_fade:
		.fill $100, 0
// ------------------------------
.pc = charset "charset" virtual
.fill $800,0
// ------------------------------
