
.const DEBUG = false
.const GENDEBUG = false
.const LOOP = true
.const fade_speed = 1
.const steps = 16
.const bitmap     = $2000
.const screenram  = $0400
.const fade_pass_address = $4000
.const src_screenram = $c000

.const zp_start = 10
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
.pc = $0819 "start"
start:
		sei
		jsr generate_fade_pass
		jsr vblank
		ldx #0
		stx $d011
		stx $d020
		stx $d021
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
		lda #$ff
!loop:
smc_src:	ldx koala_source+$1f3f
smc_dest:	stx bitmap+$1f3f
		dcp smc_src+1
		bne !+
		dec smc_src+2
	!:	dcp smc_dest+1
		bne !+
		dec smc_dest+2
	!:	ldx smc_dest+2
		cpx #>(bitmap-1)
		bne !loop-

		lda #toD018(screenram, bitmap)
		sta $d018
		lda #$c8
		sta $d016
		jsr vblank
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

		lda #$70
	!:	cmp $d012
		bne !-

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
		jmp $fce2
	}
.pc = * "vblank"
vblank:
		:vblank()
		rts
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
		lda #$af            // lax zp_src_screen_lo
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
		inx
		bne !loop-
		iny
		bne !loop-
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
.pc = * "koala_source" virtual
koala_source:
// ------------------------------
.align $100
.pc = * "t_color_fade" virtual
t_color_fade:
		.fill $100, 0
// ------------------------------
.pc = fade_pass_address "fade_pass" virtual
fade_pass:
/*
.C:4800  AF 00 04    LAX $0400
.C:4803  BD 00 09    LDA $0900,X
.C:4806  8D 00 04    STA $0400
.C:4809  AF 00 D8    LAX $D800
.C:480c  BD 00 09    LDA $0900,X
.C:480f  8D 00 D8    STA $D800
*/
	.for (var i=0; i<1000; i++) {
		lax $0400+i
		lda t_color_fade,x
		sta $0400+i
	}
		rts
