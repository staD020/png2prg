.const DEBUG         = false
.const GENDEBUG      = false
.const MUSICDEBUG    = false
.const LOOP          = false
.const fade_speed    = 2
.const steps         = 16
.const charset       = $2000
.const screenram     = $2800
.const colorram      = $d800
.const colorram_src  = $2c00
.const fade_pass_address = $ac00

.import source "lib.asm"

.pc = $0801 "basic upstart"
		.byte <basicend, >basicend, <year(), >year(), $9e
		.text toIntString(start)
		.text " PNG2PRG " + versionString()
basicend:
		.byte 0, 0, 0
.pc = settings_start() "music_startsong"
music_startsong:
		.byte 0
.pc = * "music_init"
music_init:
		jmp rrts
.pc = * "music_play"
music_play:
		jmp rrts
.pc = * "charset_case"
charset_case:
		.byte 0 // 0 = uppercase, 1 = lowercase
.pc = * "frame_delay"
frame_delay:
		.byte 0
.pc = * "wait_seconds"
wait_seconds:
		.byte 0
.pc = * "no_fade"
no_fade:
		.byte 0
.pc = * "no_loop"
no_loop:
		.byte 0

		.byte 0,0,0

.pc = basicsys() "start"
start:
		sei
		jsr $e544
		lda #$35
		sta $01

		ldy #7
!loop:
		ldx #fade_speed
	!:	jsr vblank
		dex
		bne !-

		lda $d020
		and #$0f
		tax
		lda t_easyfade,x
		sta $d020
		lda $d021
		and #$0f
		tax
		lda t_easyfade,x
		sta $d021
		dey
		bne !loop-
		sta $d011
!loop:
	.for (var i=0; i<4; i++) {
		sta colorram+i*$100,y
	}
		iny
		bne !loop-

		music_init_cia(music_startsong, music_init)

		lda #<irq
		sta $fffe
		lda #>irq
		sta $ffff

		lda #$80
	!:	cmp $d012
		bne !-
	.if (MUSICDEBUG) {
		ldx #5
	!:	dex
		bne !-
	}
		lda #%00010001
		sta $dc0e
		cli
		lda no_fade
		bne !+
		jsr generate_fade_pass
	!:
		lda no_fade
		beq prep_fade
		lax #0
	!:
	.for (var i=0; i<4; i++) {
		lda colorram_src+(i*$100),x
		sta colorram+(i*$100),x
	}
		inx
		bne !-
prep_fade:
	!:
		:setBank(screenram)
		lda #toD018(screenram, charset)
		sta $d018
		lda #$c8
		sta $d016
		lda #$1b
		sta $d011

		lda no_fade
		bne !skip_fade+

.pc = * "fade_loop"
fade_loop:
smc_yval:	ldy #steps-1
		.if (DEBUG) inc $d020
		jsr generate_phase_col_tables
		.if (DEBUG) dec $d020

		ldx #fade_speed
	!:	jsr vblank
		dex
		bne !-

	!:	lda $d012
		cmp #$60
		bcc !-
		cmp #$80
		bcs !-

		.if (DEBUG) dec $d020
		jsr fade_pass
		.if (DEBUG) inc $d020

		ldy #2
	!:	ldx colorram_src+$3e8,y
		lda t_color_fade,x
		sta $d020,y
		dey
		bpl !-

		dec smc_yval+1
		lda smc_yval+1
		cmp #$ff
		beq !done+
		cmp #(steps/2)-1
		bne fade_loop

!skip_fade:
		ldy #2
	!:	lda colorram_src+$3e8,y
		sta $d020,y
		dey
		bpl !-

		lda #$ef
	!:	cmp $dc01
		bne !-
		lda no_fade
		bne !done+
		beq fade_loop
!done:
	.if (LOOP) {
		lda #$ef
	!:	cmp $dc01
		bne !-
		jsr reset_phase
		lda #steps-1
		sta smc_yval + 1
		bne fade_loop
	} else {
		sei
		lda #$37
		sta $01
		jsr vblank
		lda #0
		sta $d011
		sta $d418
		jsr $e544
		jmp $fce2
	}
.pc = * "vblank"
vblank:
		:vblank()
rrts:	rts
// --------------------------------
// we're using non zeropage addresses here to avoid collissions with .sids
.pc = * "zp_start"
zp_start:
zp_screen_lo: .byte 0
zp_screen_hi: .byte 0
zp_src_screen_lo: .byte 0
zp_src_screen_hi: .byte 0
// --------------------------------
.pc = * "irq"
irq:
		pha
		txa
		pha
		tya
		pha
		.if (MUSICDEBUG) dec $d020
		jsr music_play
		.if (MUSICDEBUG) inc $d020
		lda $dc0d
		pla
		tay
		pla
		tax
		pla
		rti
// --------------------------------
.pc = * "generate_fade_pass"

generate_fade_pass:
		lda #<colorram
		sta zp_screen_lo
		sta zp_src_screen_lo
		lda #>colorram
		sta zp_screen_hi
		lda #>colorram_src
		sta zp_src_screen_hi

		lax #$00
		tay
!loop:
		lda #$ae            // ldx zp_src_screen_lo
		jsr store_byte
		lda zp_src_screen_lo
		jsr store_byte
		lda zp_src_screen_hi
		jsr store_byte
		lda #$bd            // lda t_color_fade,x
		jsr store_byte
		lda #<t_color_fade
		jsr store_byte
		lda #>t_color_fade
		jsr store_byte
		lda #$8d            // sta screen_lo
		jsr store_byte
		lda zp_screen_lo
		jsr store_byte
		lda zp_screen_hi
		jsr store_byte

		inc zp_src_screen_lo
		inc zp_screen_lo
		bne !+
		inc zp_src_screen_hi
		inc zp_screen_hi
	!:
		cpx #$e7
		bne not_last
		cpy #$03
		beq !done+
not_last:
	.if (zp_start < $f9) {
			inx
			bne !loop-
			iny
			bne !loop-
	} else {
			inx
			beq !+
	jmploop:
			jmp !loop-
		!:
			iny
			bne jmploop
	}
!done:
		lda #$60            // rts
store_byte:
		sta fade_pass
		inc store_byte+1
		bne !+
		inc store_byte+2
	!:	rts
// --------------------------------
.pc = * "generate_phase_col_tables"
generate_phase_col_tables:
!next_step:
//		lda #<t_color_fade
//		sta smc_totpercol + 1
!loop:
		// start with color 0
		ldx #0
	!:
		// y points to hi-nibble of color x
		ldy t_col2index,x // y = 0, $10, $20, .., $f0
smc_fadepercol1:
		lda t_fadepercol
		asl
		asl
		asl
		asl
smc_fadepercol2:
		ora t_fadepercol,y
smc_totpercol:
		sta t_color_fade
		.if (GENDEBUG) sta $d020
		inc smc_totpercol + 1
		inx
		cpx #$10
		bcc !-

		lda smc_fadepercol1 + 1
		// clc not needed , carry is always set
		adc #$0f
		sta smc_fadepercol1 + 1
		bcc !loop-

		// prepare code for next phase
		inc smc_fadepercol1 + 1
		inc smc_fadepercol2 + 1
		rts
// --------------------------------
.pc = * "reset_phase"
reset_phase:
	.if (LOOP) {
		lda #<t_fadepercol
		sta smc_fadepercol1 + 1
		sta smc_fadepercol2 + 1
		rts
	}
// ------------------------------
.pc = * "t_col2index"
t_col2index:
		.fill $10, i*$10
// ------------------------------
.pc = * "t_easyfade"
t_easyfade:
		.byte $00,$0d,$09,$0c,$02,$08,$00,$0f
		.byte $02,$00,$08,$09,$04,$03,$04,$05
// ------------------------------
.align $100
.pc = * "t_fadepercol"
t_fadepercol:
:colorfade_table()
// ------------------------------
.align $100
.pc = * "t_color_fade"
t_color_fade:
		.fill $100, 0
// ------------------------------
.pc = charset "charset" virtual
.fill $800,0
// ------------------------------
.pc = fade_pass_address "fade_pass" virtual
fade_pass:
	.for (var i=0; i<1000; i++) {
		ldx colorram_src+i
		lda t_color_fade,x
		sta colorram+i
	}
		rts
