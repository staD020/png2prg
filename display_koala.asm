
.const DEBUG = false
.const GENDEBUG = false
.const LOOP = false
.const PERFRAME = false
.const fade_speed = 2
.const steps = 16
.const bitmap     = $2000
.const screenram  = $0400
.const colorram   = $d800
.const fade_pass_address = $4800
.const src_screenram = $c000
.const src_colorram = $c400

.const zp_start = 10
.const zp_screen_lo = zp_start + 0
.const zp_screen_hi = zp_start + 1
.const zp_d800_lo = zp_start + 2
.const zp_d800_hi = zp_start + 3
.const zp_src_screen_lo = zp_start + 4
.const zp_src_screen_hi = zp_start + 5
.const zp_src_d800_lo = zp_start + 6
.const zp_src_d800_hi = zp_start + 7

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
		jsr vblank
		lda #$35
		sta $01

		lda #0
		sta $d011
		sta $d020
		sta $d021
		jsr rrts // music init

		lda #$7f
		sta $dc0d
		lda $dc0d
		lda #$42
		sta $d012
		lda #<irq
		sta $fffe
		lda #>irq
		sta $ffff

		lda #1
		sta $d01a
		inc $d019
		cli

		jsr generate_fade_pass
		ldx #0
	!:
		lda #0
	.for (var i=0; i<4; i++) {
		sta screenram+(i*$100),x
		sta colorram+(i*$100),x
	}
	.for (var i=0; i<4; i++) {
		lda koala_source+$1f40+(i*$100),x
		sta src_screenram+(i*$100),x
		lda koala_source+$2328+(i*$100),x
		sta src_colorram+(i*$100),x
	}
		inx
		bne !-

		lda #toD018(screenram, bitmap)
		sta $d018
		lda #$d8
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

.if (PERFRAME) {
		lda #$ef
	!:	cmp $dc01
		bne !-
	!:	cmp $dc01
		beq !-
}
		ldx src_colorram+1000
		lda t_color_fade,x
		sta $d021
		lda src_colorram+1000
		lsr
		lsr
		lsr
		lsr
		tax
		lda t_color_fade,x
		sta $d020

		lda #$70
	!:	cmp $d012
		bne !-

		.if (DEBUG) dec $d020
		jsr fade_pass
		.if (DEBUG) inc $d020

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

		// reset phase
		lda #<t_fadepercol
		sta smc_fadepercol1 + 1
		sta smc_fadepercol2 + 1

		lda #steps-1
		sta smc_yval + 1
		bne fade_loop
	} else {
		sei
		lda #$37
		sta $01
		lda #0
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
		jsr rrts
		inc $d019
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
		sta zp_d800_lo
		sta zp_src_screen_lo
		sta zp_src_d800_lo
		lda #>screenram
		sta zp_screen_hi
		lda #>colorram
		sta zp_d800_hi
		lda #>src_screenram
		sta zp_src_screen_hi
		lda #>src_colorram
		sta zp_src_d800_hi

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

		lda #$ae            // ldx zp_src_d800_lo
		jsr store_byte
		lda zp_src_d800_lo
		jsr store_byte
		lda zp_src_d800_hi
		jsr store_byte
		lda #$bd            // lda t_color_fade,x
		jsr store_byte
		lda #<t_color_fade
		jsr store_byte
		lda #>t_color_fade
		jsr store_byte
		lda #$8d            // sta d800_lo
		jsr store_byte
		lda zp_d800_lo
		jsr store_byte
		lda zp_d800_hi
		jsr store_byte

		inc zp_src_screen_lo
		inc zp_src_d800_lo
		inc zp_screen_lo
		inc zp_d800_lo
		bne !+
		inc zp_src_screen_hi
		inc zp_src_d800_hi
		inc zp_screen_hi
		inc zp_d800_hi
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
		.if (GENDEBUG) sta $d020
		inc store_byte+1
		bne !+
		inc store_byte+2
	!:	rts
// --------------------------------
.pc = * "generate_phase_col_tables"
generate_phase_col_tables:
		//lda #<t_color_fade
		//sta smc_totpercol + 1
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
		// clc not needed, carry is always set
		adc #$0f
		sta smc_fadepercol1 + 1
		bcc !loop-

		// prepare code for next phase
		inc smc_fadepercol1 + 1
		inc smc_fadepercol2 + 1
		rts
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
.fill $2711, 0
// ------------------------------
.pc = fade_pass_address "fade_pass" virtual
fade_pass:
/*
.C:4800  AE 00 04    LDX $0400
.C:4803  BD 00 09    LDA $0900,X
.C:4806  8D 00 04    STA $0400
.C:4809  AE 00 D8    LDX $D800
.C:480c  BD 00 09    LDA $0900,X
.C:480f  8D 00 D8    STA $D800
*/
	.for (var i=0; i<1000; i++) {
		ldx $0400+i
		lda t_color_fade,x
		sta $0400+i
		ldx $d800+i
		lda t_color_fade,x
		sta $d800+i
	}
		rts
