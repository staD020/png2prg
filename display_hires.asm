.macro vblank() {
    !:  lda $d011
        bpl !-
    !:  lda $d011
        bmi !-
}

.pc = $0801 "basic upstart"
:BasicUpstart(start)

.pc = $0810 "start"
start:
		sei
		lda #$37
		sta $01
		:vblank()
		lda #0
		sta $d011
		sta $d020
		sta $d021
		ldx #0
	!:
	.for (var i=0; i<4; i++) {
		lda $3f40+(i*$100),x
		sta $0400+(i*$100),x
	}
		inx
		bne !-
		lda #$18
		sta $d018
		lda #$c8
		sta $d016
		:vblank()
		lda #$3b
		sta $d011
	!:
		jmp !-
