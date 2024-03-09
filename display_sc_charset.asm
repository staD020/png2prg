
.const fade_speed = 4
.const charset    = $2000
.const screenram  = $0400

.import source "lib.asm"

.pc = $0801 "basic upstart"
		.byte <basicend, >basicend, <year(), >year(), $9e
		.text toIntString(start)
		.text " PNG2PRG " + versionString()
basicend:
		.byte 0, 0, 0
.pc = $0822 "start"
start:
		sei
		lda #$37
		sta $01
		jsr vblank
		ldx #0
		stx $d011
		lda charset_source+$800+1000+2
		sta $d020
		lda charset_source+$800+1000+1
		sta $d021
	!:
		lda charset_source+$800+1000
	.for (var i=0; i<4; i++) {
		sta $d800+(i*$100),x
	}

	.for (var i=0; i<4; i++) {
		lda charset_source+$800+(i*$100),x
		sta screenram+(i*$100),x
	}
		inx
		bne !-

		lda #$ff
!loop:
smc_src:
		ldx charset_source+$7ff
smc_dest:
		stx charset+$7ff
		dcp smc_src+1
		bne !+
		dec smc_src+2
	!:
		dcp smc_dest+1
		bne !+
		dec smc_dest+2
	!:
		ldx smc_dest+2
		cpx #>(charset-1)
		bne !loop-

		jsr vblank
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

.pc = * "charset_source" virtual
charset_source:
