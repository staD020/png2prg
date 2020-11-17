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
		ldx #0
		stx $d011
		stx $d020
		stx $d021
		lda $2be8
		ora #$08
		tay
	!:
		tya
	.for (var i=0; i<4; i++) {
		sta $d800+(i*$100),x
	}
	.for (var i=0; i<4; i++) {
		lda $2800+(i*$100),x
		sta $0400+(i*$100),x
	}
		inx
		bne !-
		lda #$18
		sta $d018
		lda #$d8
		sta $d016
		lda $2be9
		sta $d021
		lda $2bea
		sta $d022
		lda $2beb
		sta $d023
		:vblank()
		lda #$1b
		sta $d011
	!:
		jmp !-
