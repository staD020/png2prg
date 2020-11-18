.const fade_speed = 4
.const bitmap     = $2000
.const screenram  = $0400

.import source "lib.asm"

.pc = $0801 "basic upstart"
:BasicUpstart(start)

.pc = $0810 "start"
start:
		sei
		lda #$37
		sta $01
		:vblank()
		ldx #0
		stx $d011
		stx $d020
		stx $d021
	!:
	.for (var i=0; i<4; i++) {
		lda hires_source+$1f40+(i*$100),x
		sta screenram+(i*$100),x
	}
		inx
		bne !-

!loop:
smc_src:
		lda hires_source+$1f3f
smc_dest:
		sta bitmap+$1f3f
		dec smc_src+1
		lda smc_src+1
		cmp #$ff
		bne !+
		dec smc_src+2
	!:
		dec smc_dest+1
		lda smc_dest+1
		cmp #$ff
		bne !+
		dec smc_dest+2
	!:
		lda smc_dest+2
		cmp #>(bitmap-1)
		bne !loop-

		lda #toD018(screenram, bitmap)
		sta $d018
		lda #$c8
		sta $d016
		:vblank()
		lda #$3b
		sta $d011

		lda #$ef
	!:	cmp $dc01
		bne !-
		jmp $fce2

.pc = * "hires_source" virtual
hires_source:
