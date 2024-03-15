
.const charset    = $2000
.const screenram  = $2800
.const colorram   = $2c00

.import source "lib.asm"

.pc = $0801 "basic upstart"
		.byte <basicend, >basicend, <year(), >year(), $9e
		.text toIntString(start)
		.text " PNG2PRG " + versionString()
basicend:
		.byte 0, 0, 0
.pc = $0822 "start"
start:
		//sei
		lda #$37
		sta $01
		jsr vblank

		ldx #0
		stx $d011
		stx $d020
		lda charset+$fe8
		sta $d021
	!:
	.for (var i=0; i<4; i++) {
		lda colorram+(i*$100),x
		sta $d800+(i*$100),x
	}
		inx
		bne !-

		jsr vblank
		lda charset+$fe9
		sta $d020
		:setBank(charset)
		lda #toD018(screenram, charset)
		sta $d018
		lda #$c8
		sta $d016
		lda #$1b
		sta $d011

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
