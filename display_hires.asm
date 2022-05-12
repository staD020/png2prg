
.const DEBUG = false
.const GENDEBUG = false
.const LOOP = false
.const fade_speed = 2
.const steps = 16
.const bitmap     = $2000
.const screenram  = $0400
.const fade_pass_address = $4800
.const src_screenram = $c000

.const zp_start = $0334
.const zp_screen_lo = zp_start + 0
.const zp_screen_hi = zp_start + 1
.const zp_src_screen_lo = zp_start + 2
.const zp_src_screen_hi = zp_start + 3

.import source "lib.asm"

.pc = $0801 "basic upstart"
		.byte <basicend, >basicend, <2022, >2022, $9e
		.text toIntString(start)
		.text " PNG2PRG " + versionString()
basicend:
		.byte 0, 0, 0
.pc = $0819 "music_init"
music_init:
		jmp rrts
.pc = * "music_play"
music_play:
		jmp rrts

.pc = * "start"
start:
		sei
		jsr vblank
		lda #$35
		sta $01
		lda #0
		sta $d011
		sta $d020
		sta $d021

		// default pal 50 hz: $4cc7
		lda #$c7
		sta $dc04
		lda #$4c
		sta $dc05

		lax #0
		tay
		jsr music_init
		lda #<irq
		sta $fffe
		lda #>irq
		sta $ffff

		lda #$80
	!:	cmp $d012
		bne !-

		lda #%00010001
		sta $dc0e
		cli

		jsr generate_fade_pass
		ldx #0
	!:
		lda #0
	.for (var i=0; i<4; i++) {
		sta screenram+(i*$100),x
	}
	.for (var i=0; i<4; i++) {
		lda koala_source+$1f40+(i*$100),x
		sta src_screenram+(i*$100),x
	}
		inx
		bne !-

		jsr vblank
		lda #toD018(screenram, bitmap)
		sta $d018
		lda #$c8
		sta $d016
		lda #$3b
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

	!:	lda $d012
		cmp #$60
		bcc !-
		cmp #$80
		bcs !-

		.if (DEBUG) dec $d020
		jsr fade_pass
		.if (DEBUG) inc $d020

		ldx src_screenram+1000
		lda t_color_fade,x
		sta $d020

		dec smc_yval+1
		lda smc_yval+1
		cmp #$ff
		beq !done+
		cmp #(steps/2)-1
		bne fade_loop

		lda #$ef
	!:	cmp $dc01
		bne !-
		beq fade_loop
!done:
	.if (LOOP) {
		lda #$ef
	!:	cmp $dc01
		bne !-
		jsr reset_phase
		lda #steps-1
		sta smc_yval + 1
		bne fade_loop
	} else {
		sei
		lda #$37
		sta $01
		jsr vblank
		lda #0
		sta $d011
		sta $d418
		jmp $fce2
	}
.pc = * "vblank"
vblank:
		:vblank()
rrts:	rts
// --------------------------------
.pc = * "irq"
irq:
		pha
		txa
		pha
		tya
		pha
		.if (DEBUG) dec $d020
		jsr music_play
		.if (DEBUG) inc $d020
		lda $dc0d
		pla
		tay
		pla
		tax
		pla
		rti
// --------------------------------
.pc = * "generate_fade_pass"

generate_fade_pass:
		lda #<screenram
		sta zp_screen_lo
		sta zp_src_screen_lo
		lda #>screenram
		sta zp_screen_hi
		lda #>src_screenram
		sta zp_src_screen_hi

		lax #$00
		tay
!loop:
		lda #$ae            // ldx zp_src_screen_lo
		jsr store_byte
		lda zp_src_screen_lo
		jsr store_byte
		lda zp_src_screen_hi
		jsr store_byte
		lda #$bd            // lda t_color_fade,x
		jsr store_byte
		lda #<t_color_fade
		jsr store_byte
		lda #>t_color_fade
		jsr store_byte
		lda #$8d            // sta screen_lo
		jsr store_byte
		lda zp_screen_lo
		jsr store_byte
		lda zp_screen_hi
		jsr store_byte

		inc zp_src_screen_lo
		inc zp_screen_lo
		bne !+
		inc zp_src_screen_hi
		inc zp_screen_hi
	!:
		cpx #$e7
		bne not_last
		cpy #$03
		beq !done+
not_last:
	.if (zp_start < $f9) {
			inx
			bne !loop-
			iny
			bne !loop-
	} else {
			inx
			beq !+
	jmploop:
			jmp !loop-
		!:
			iny
			bne jmploop
	}
!done:
		lda #$60            // rts
store_byte:
		sta fade_pass
		inc store_byte+1
		bne !+
		inc store_byte+2
	!:	rts
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
	.if (LOOP) {
		lda #<t_fadepercol
		sta smc_fadepercol1 + 1
		sta smc_fadepercol2 + 1
		rts
	}
// ------------------------------
.pc = * "t_col2index"
t_col2index:
		.fill $10, i*$10
// ------------------------------
.align $100
.pc = * "t_fadepercol"
t_fadepercol:
:colorfade_table()
// ------------------------------
.align $100
.pc = * "t_color_fade" virtual
t_color_fade:
		.fill $100, 0
// ------------------------------
.pc = bitmap "koala_source" virtual
koala_source:
// ------------------------------
.pc = fade_pass_address "fade_pass" virtual
fade_pass:
/*
.C:4800  AE 00 04    LDX $0400
.C:4803  BD 00 09    LDA $0900,X
.C:4806  8D 00 04    STA $0400
*/
	.for (var i=0; i<1000; i++) {
		ldx $0400+i
		lda t_color_fade,x
		sta $0400+i
	}
		rts
