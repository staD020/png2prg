
.const fade_speed = 4
.const charset    = $2000
.const screenram  = $0400

.import source "lib.asm"

.pc = $0801 "basic upstart"
//:BasicUpstart(start)
.byte $17,$08,$e5,$07, $9e,$32,$30,$37
.byte $33,$20,$50,$4e, $47,$32,$50,$52
.byte $47,$20,$30,$2e, $36

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
	!:
		lda #1
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
		cpx #>(charset-1)
		bne !loop-

		lda #toD018(screenram, charset)
		sta $d018
		lda #$c8
		sta $d016
		jsr vblank
		lda #$1b
		sta $d011
		lda #$ef
	!:	cmp $dc01
		bne !-
		jmp $fce2
vblank:
		:vblank()
		rts

.pc = * "charset_source" virtual
charset_source:
