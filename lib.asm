.importonce

.function versionString() {
	.return "0.9"
}

.macro vblank() {
    !:  lda $d011
        bpl !-
    !:  lda $d011
        bmi !-
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
    t_facecol_0:
            .byte $0,$0,$0,$0, $0,$0,$0,$0
            .byte $0,$0,$0,$0, $0,$0,$0,$0
    t_facecol_1:
            .byte $9,$2,$8,$c, $a,$f,$7,$1
            .byte $7,$f,$a,$c, $8,$2,$9,$0
    t_facecol_2:
            .byte $0,$0,$0,$0, $0,$0,$9,$2
            .byte $9,$0,$0,$0, $0,$0,$0,$0
    t_facecol_3:
            .byte $0,$0,$6,$b, $4,$e,$5,$3
            .byte $5,$e,$4,$b, $6,$0,$0,$0
    t_facecol_4:
            .byte $0,$0,$0,$0, $0,$6,$b,$4
            .byte $b,$6,$0,$0, $0,$0,$0,$0
    t_facecol_5:
            .byte $0,$0,$0,$6, $b,$4,$e,$5
            .byte $e,$4,$b,$6, $0,$0,$0,$0
    t_facecol_6:
            .byte $0,$0,$0,$0, $0,$0,$0,$6
            .byte $0,$0,$0,$0, $0,$0,$0,$0
    t_facecol_7:
            .byte $0,$9,$2,$8, $c,$a,$f,$7
            .byte $f,$a,$c,$8, $2,$9,$0,$0
    t_facecol_8:
            .byte $0,$0,$0,$0, $0,$9,$2,$8
            .byte $2,$9,$0,$0, $0,$0,$0,$0
    t_facecol_9:
            .byte $0,$0,$0,$0, $0,$0,$0,$9
            .byte $0,$0,$0,$0, $0,$0,$0,$0
    t_facecol_a:
            .byte $0,$0,$0,$9, $2,$8,$c,$a
            .byte $c,$8,$2,$9, $0,$0,$0,$0
    t_facecol_b:
            .byte $0,$0,$0,$0, $0,$0,$6,$b
            .byte $6,$0,$0,$0, $0,$0,$0,$0
    t_facecol_c:
            .byte $0,$0,$0,$0, $9,$2,$8,$c
            .byte $8,$2,$9,$0, $0,$0,$0,$0
    t_facecol_d:
            .byte $0,$6,$b,$4, $e,$5,$3,$d
            .byte $3,$5,$e,$4, $b,$6,$0,$0
    t_facecol_e:
            .byte $0,$0,$0,$0, $6,$b,$4,$e
            .byte $4,$b,$6,$0, $0,$0,$0,$0
    t_facecol_f:
            .byte $0,$0,$9,$2, $8,$c,$a,$f
            .byte $a,$c,$8,$2, $9,$0,$0,$0
}