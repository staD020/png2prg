.importonce

.function versionString() {
	.return "1.2"
}

.function year() {
	.return 2022
}

.macro vblank() {
	!:	bit $d011
		bpl !-
	!:	bit $d011
		bmi !-
}

.macro setBank(addr) {
	lda #toDD00(addr)
	sta $dd00
}

.macro setD018(screen, charset) {
	lda #toD018(screen, charset)
	sta $d018
}

.function toDD00(addr) {
	.return (>addr >> 6) ^ %11
}
.function toD018(screen, charset) {
	.return screenToD018(screen) | charsetToD018(charset)
}
.function screenToD018(addr) {
	.return ( ( addr & $3fff ) / $400 ) << 4
}
.function charsetToD018(addr) {
	.return ( ( addr & $3fff ) / $800 ) << 1
}
.function toSpritePtr(addr) {
	.return ( addr & $3fff ) / $40
}

.macro colorfade_table() {
	// veto's colfade v2
	t_facecol_0:
			.byte $0,$0,$0,$0, $0,$0,$0,$0
			.byte $0,$0,$0,$0, $0,$0,$0,$0
	t_facecol_1:
			.byte $0,$9,$2,$4, $c,$3,$d,$1
			.byte $d,$3,$c,$4, $2,$9,$0,$0
	t_facecol_2:
			.byte $0,$0,$0,$0, $0,$0,$9,$2
			.byte $9,$0,$0,$0, $0,$0,$0,$0
	t_facecol_3:
			.byte $0,$0,$0,$9, $2,$4,$c,$3
			.byte $c,$4,$2,$9, $0,$0,$0,$0
	t_facecol_4:
			.byte $0,$0,$0,$0, $0,$6,$b,$4
			.byte $b,$6,$0,$0, $0,$0,$0,$0
	t_facecol_5:
			.byte $0,$0,$0,$0, $9,$b,$4,$5
			.byte $4,$b,$9,$0, $0,$0,$0,$0
	t_facecol_6:
			.byte $0,$0,$0,$0, $0,$0,$0,$6
			.byte $0,$0,$0,$0, $0,$0,$0,$0
	t_facecol_7:
			.byte $0,$0,$9,$2, $8,$a,$f,$7
			.byte $f,$a,$8,$2, $9,$0,$0,$0
	t_facecol_8:
			.byte $0,$0,$0,$0, $0,$9,$2,$8
			.byte $2,$9,$0,$0, $0,$0,$0,$0
	t_facecol_9:
			.byte $0,$0,$0,$0, $0,$0,$9,$9
			.byte $9,$0,$0,$0, $0,$0,$0,$0
	t_facecol_a:
			.byte $0,$0,$0,$0, $9,$2,$8,$a
			.byte $8,$2,$9,$0, $0,$0,$0,$0
	t_facecol_b:
			.byte $0,$0,$0,$0, $0,$0,$6,$b
			.byte $6,$0,$0,$0, $0,$0,$0,$0
	t_facecol_c:
			.byte $0,$0,$0,$0, $9,$b,$4,$c
			.byte $4,$b,$9,$0, $0,$0,$0,$0
	t_facecol_d:
			.byte $0,$0,$6,$b, $4,$e,$3,$d
			.byte $3,$e,$4,$b, $6,$0,$0,$0
	t_facecol_e:
			.byte $0,$0,$0,$0, $6,$b,$4,$e
			.byte $4,$b,$6,$0, $0,$0,$0,$0
	t_facecol_f:
			.byte $0,$0,$0,$9, $2,$8,$a,$f
			.byte $a,$8,$2,$9, $0,$0,$0,$0
}
