.importonce

.function versionString() {
	.return "0.7"
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
