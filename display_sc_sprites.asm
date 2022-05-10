
.const fade_speed = 4
.const screenram  = $0400
.const sprites    = $2000
.const spr_xpos_start = $40
.const spr_ypos_start = $32

.const zp_start     = $10
.const zp_spr_xy_lo = zp_start + 0
.const zp_spr_xy_hi = zp_start + 1
.const zp_spr_xpos  = zp_start + 2
.const zp_spr_ypos  = zp_start + 3

.import source "lib.asm"

.pc = $0801 "basic upstart"
		.byte <basicend, >basicend, <2021, >2021, $9e
		.text toIntString(start)
		.text " PNG2PRG " + versionString()
basicend:
		.byte 0, 0, 0
.pc = $0819 "start"
start:
		sei
		lda #$37
		sta $01
		jsr vblank
		ldx #0
		stx $d011
		stx $d020
		stx $d021
		lda #1
		sta $0286
		jsr $e536

		lda #$ff
!loop:
smc_src:
		ldx spr_bitmap+$1fff
smc_dest:
		stx sprites+$1fff
		dcp smc_src+1
		bne !+
		dec smc_src+2
	!:
		dcp smc_dest+1
		bne !+
		dec smc_dest+2
	!:
		ldx smc_dest+2
		cpx #>(sprites-1)
		bne !loop-

init_sprites:
		anc #0
		sta $d010
		sta $d015
		sta $d017
		sta $d01b
		sta $d01d
		sta $d01c	// single/multicol

		ldx #7
		lda spr_spritecol
	!:	sta $d027,x
		dex
		bpl !-

		ldx #$f
		lda #0
	!:	sta $d000,x
		dex
		bpl !-

		ldx #toSpritePtr(sprites)
	.for (var i=0; i<7; i++) {
		stx screenram+$3f8+i
		inx
	}
		stx screenram+$3f8+7

		lda #<$d000
		sta zp_spr_xy_lo
		lda #>$d000
		sta zp_spr_xy_hi
		lda #$40
		sta zp_spr_xpos
		lda #$32
		sta zp_spr_ypos

		ldy #0
!loop:	ldx spr_columns
!:		lda zp_spr_xpos
		sta (zp_spr_xy_lo),y
		clc
		adc #$18
		sta zp_spr_xpos
		iny
		lda zp_spr_ypos
		sta (zp_spr_xy_lo),y
		iny
		cpy #$10
		beq !done+
		dex
		bne !-

		lda zp_spr_ypos
		clc
		adc #21
		sta zp_spr_ypos
		lda #spr_xpos_start
		sta zp_spr_xpos

		dec spr_rows
		beq !done+
		cpy #$10
		bne !loop-
!done:
		lda #toD018(screenram, $1000)
		sta $d018
		lda #$c8
		sta $d016

		jsr vblank
		lda #$1b
		sta $d011
		lda spr_bgcol
		sta $d021
		lda #$ff
		sta $d015

		lda #$ef
	!:	cmp $dc01
		bne !-
		jsr vblank
		lda #0
		sta $d011
		jmp $fce2
vblank:
		:vblank()
		rts
// -------------------------------------------

.pc = * "sprites_source" virtual
sprites_source:

spr_columns:	.byte 0
spr_rows:		.byte 0
spr_bgcol:		.byte 0
spr_spritecol:	.byte 0

spr_bitmap:
