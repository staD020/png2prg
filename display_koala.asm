
.const fade_speed = 4
.const bitmap     = $2000
.const screenram  = $0400
.const colorram   = $d800
.const fade_pass_address = $4000

.import source "lib.asm"

.pc = $0801 "basic upstart"
:BasicUpstart(start)

.pc = $0810 "start"
start:
		sei
		jsr vblank
		ldx #0
		stx $d011
		stx $d020
	!:
	.for (var i=0; i<4; i++) {
		lda koala_source+$1f40+(i*$100),x
		sta screenram+(i*$100),x
		lda koala_source+$2328+(i*$100),x
		sta colorram+(i*$100),x
	}
		inx
		bne !-
		lda koala_source+$2710
		sta $d021

		lda #$ff
!loop:
smc_src:
		ldx koala_source+$1f3f
smc_dest:
		stx bitmap+$1f3f
		//lda #$ff
		dcp smc_src+1
		//dec smc_src+1
		//lda smc_src+1
		//cmp #$ff
		bne !+
		dec smc_src+2
	!:
		dcp smc_dest+1
		//dec smc_dest+1
		//lda smc_dest+1
		//cmp #$ff
		bne !+
		dec smc_dest+2
	!:
		ldx smc_dest+2
		cpx #>(bitmap-1)
		bne !loop-

		lda #toD018(screenram, bitmap)
		sta $d018
		lda #$d8
		sta $d016
		jsr vblank
		lda #$3b
		sta $d011

		jsr generate_t_color_fade
		jsr generate_fade_pass

		lda #$ef
	!:	cmp $dc01
		bne !-

		ldy #10
!loop:
		ldx #fade_speed
	!:	jsr vblank
		dex
		bne !-

		ldx $d021
		lda t_color_fade,x
		sta $d021

		lda #$70
	!:  cmp $d012
		bne !-
		jsr fade_pass

		dey
		bne !loop-
		jmp $fce2

vblank:
        :vblank()
        rts
// --------------------------------
.pc = * "generate_fade_pass"

.const zp_start = $fb
.const zp_screen_lo = zp_start + 0
.const zp_screen_hi = zp_start + 1
.const zp_d800_lo = zp_start + 2
.const zp_d800_hi = zp_start + 3

generate_fade_pass:
		lda #<screenram
		sta zp_screen_lo
		sta zp_d800_lo
		lda #>screenram
		sta zp_screen_hi
		lda #>colorram
		sta zp_d800_hi

		ldx #$00
		ldy #$00
!loop:
		lda #$af            // lax screen_lo
		jsr store_byte
		lda zp_screen_lo
		jsr store_byte
		lda zp_screen_hi
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

		lda #$af            // lax d800_lo
		jsr store_byte
		lda zp_d800_lo
		jsr store_byte
		lda zp_d800_hi
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

		inc zp_screen_lo
		inc zp_d800_lo
		bne !+
		inc zp_screen_hi
		inc zp_d800_hi
	!:
		cpx #$e7
		bne not_last
		cpy #$03
		beq done
not_last:
		inx
		bne !loop-
		iny
		bne !loop-
done:
		lda #$60            // rts
store_byte:
		sta fade_pass
		inc store_byte+1
		bne !+
		inc store_byte+2
	!:	rts
/*
.C:4800  AF 00 04    LAX $0400
.C:4803  BD 00 09    LDA $0900,X
.C:4806  8D 00 04    STA $0400
.C:4809  AF 00 D8    LAX $D800
.C:480c  BD 00 09    LDA $0900,X
.C:480f  8D 00 D8    STA $D800
	.for (var i=0; i<1000; i++) {
		lax $0400+i
		lda t_color_fade,x
		sta $0400+i
		lax $d800+i
		lda t_color_fade,x
		sta $d800+i
	}
		rts
*/
// --------------------------------
.pc = * "generate_t_total_colfade"
// generate the full fade table, for both low nibble AND hi nibble colors
// this is a fast approach, but really limits the fadecolors and order.
generate_t_color_fade:
		lda #<t_color_fade
		sta smc_cf + 1

		ldx #0
!loop:
		ldy #0
	!:
		lda t_fadecol,x
		asl
		asl
		asl
		asl
		ora t_fadecol,y
smc_cf:
		sta t_color_fade
		inc smc_cf + 1
		iny
		cpy #$10
		bne !-

		inx
		cpx #$10
		bne !loop-
		rts

// ------------------------------
.pc = * "t_fadecol"
t_fadecol:
		.byte $00,$0d,$09,$0c,$02,$08,$00,$0f
		.byte $02,$00,$08,$09,$04,$03,$04,$05
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
		lax $d800+i
		lda t_color_fade,x
		sta $d800+i
	}
		rts
