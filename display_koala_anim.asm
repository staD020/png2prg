
.const DEBUG = false
.const GENDEBUG = false
.const MUSICDEBUG = false
.const LOOP = false
.const PERFRAME = false
.const fade_speed = 1
.const steps = 16
.const bitmap     = $2000
.const screenram  = $0400
.const colorram   = $d800
.const src_screenram = $4000
.const src_colorram = $4400
.const animations = $4800
.const fade_pass_address = $8900

.const zp_anim_start  = $08
.const zp_anim_lo     = zp_anim_start + 0
.const zp_anim_hi     = zp_anim_start + 1
.const zp_bitmap_lo   = zp_anim_start + 2
.const zp_bitmap_hi   = zp_anim_start + 3
.const zp_char_lo     = zp_anim_start + 4
.const zp_char_hi     = zp_anim_start + 5
.const zp_d800_lo     = zp_anim_start + 6
.const zp_d800_hi     = zp_anim_start + 7

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
.pc = * "frame_delay"
frame_delay:
		.byte 0
.pc = * "wait_seconds"
wait_seconds:
		.byte 0
.pc = * "no_fade"
no_fade:
		.byte 0

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
		lax #0
	!:
	.for (var i=0; i<4; i++) {
		sta screenram+(i*$100),x
		sta colorram+(i*$100),x
	}
		inx
		bne !-

		lda no_fade
		beq !prep_fade+
		lda #>(screenram+$300)
		sta smc_src_screen+2
		lda #>(colorram+$300)
		sta smc_src_col+2
!prep_fade:

		ldy #4
		ldx #$e8
	!:
smc_koalasrc_col:
		lda koala_source+$2328+$300,x
smc_src_col:
		sta src_colorram+$300,x
		dex
		cpx #$ff
		bne !-
		dec smc_koalasrc_col+2
		dec smc_src_col+2
		dey
		bne !-

		ldy #4
		ldx #$e8
	!:
smc_koalasrc_screen:
		lda koala_source+$1f40+$300,x
smc_src_screen:
		sta src_screenram+$300,x
		dex
		cpx #$ff
		bne !-
		dec smc_koalasrc_screen+2
		dec smc_src_screen+2
		dey
		bne !-

		jsr vblank
		:setBank(bitmap)
		lda #toD018(screenram, bitmap)
		sta $d018
		lda #$d8
		sta $d016
		lda #$3b
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

.if (PERFRAME) {
		lda #$ef
	!:	cmp $dc01
		bne !-
	!:	cmp $dc01
		beq !-
}
		ldx src_colorram+1000
		lda t_color_fade,x
		sta $d021
		lda src_colorram+1000
		lsr
		lsr
		lsr
		lsr
		tax
		lda t_color_fade,x
		sta $d020

	!:	lda $d012
		cmp #$70
		bcc !-
		cmp #$90
		bcs !-

		.if (DEBUG) dec $d020
		jsr fade_pass
		.if (DEBUG) inc $d020

		dec smc_yval+1
		lda smc_yval+1
		cmp #$ff
		beq !done+
		cmp #(steps/2)-1
		bne fade_loop
		beq !+
!skip_fade:
		lda $4710 // src_colorram+1000
		sta $d021
		lsr
		lsr
		lsr
		lsr
		sta $d020
!:
		jsr anim_init

		// optional wait before anim start
		ldy wait_seconds
		beq !skip+
!waitloop:
		ldx #50
	!:	jsr vblank
		dex
		bne !-
		dey
		bne !waitloop-
!skip:

loop_anim:
		.if (DEBUG) inc $d020
		jsr anim_play
		.if (DEBUG) dec $d020

		lda $dc01
		cmp #$ef
		bne loop_anim
		lda no_fade
		bne !done+
		jmp fade_loop
!done:
	.if (LOOP) {
		lda #$ef
	!:	cmp $dc01
		bne !-

		// reset phase
		lda #<t_fadepercol
		sta smc_fadepercol1 + 1
		sta smc_fadepercol2 + 1

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
zp_src_d800_lo: .byte 0
zp_src_d800_hi: .byte 0
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
		lda #<screenram
		sta zp_screen_lo
		sta zp_d800_lo
		sta zp_src_screen_lo
		sta zp_src_d800_lo
		lda #>screenram
		sta zp_screen_hi
		lda #>colorram
		sta zp_d800_hi
		lda #>src_screenram
		sta zp_src_screen_hi
		lda #>src_colorram
		sta zp_src_d800_hi

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

		lda #$ae            // ldx zp_src_d800_lo
		jsr store_byte
		lda zp_src_d800_lo
		jsr store_byte
		lda zp_src_d800_hi
		jsr store_byte
		lda #$bd            // lda t_color_fade,x
		jsr store_byte
		lda #<t_color_fade
		jsr store_byte
		lda #>t_color_fade
		jsr store_byte
		lda #$8d            // sta d800_lo
		jsr store_byte
		lda zp_d800_lo
		jsr store_byte
		lda zp_d800_hi
		jsr store_byte

		inc zp_src_screen_lo
		inc zp_src_d800_lo
		inc zp_screen_lo
		inc zp_d800_lo
		bne !+
		inc zp_src_screen_hi
		inc zp_src_d800_hi
		inc zp_screen_hi
		inc zp_d800_hi
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
		.if (GENDEBUG) sta $d020
		inc store_byte+1
		bne !+
		inc store_byte+2
	!:	rts
// --------------------------------
.pc = * "generate_phase_col_tables"
generate_phase_col_tables:
		//lda #<t_color_fade
		//sta smc_totpercol + 1
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
		// sec not needed, carry is always set
		adc #$0f
		sta smc_fadepercol1 + 1
		bcc !loop-

		// prepare code for next phase
		inc smc_fadepercol1 + 1
		inc smc_fadepercol2 + 1
		rts
// ------------------------------
.pc = * "anim_play"
anim_play:
next_chunk:
		ldy #0
		lax (zp_anim_lo),y      //  a = x = number of chars in chunk
		bne plot_chunk          // #$00 = end of frame
		inc zp_anim_lo
		bne !+
		inc zp_anim_hi
	!:
		lax (zp_anim_lo),y      // framedelay
	!:	jsr vblank
		dex
		bne !-
		inc zp_anim_lo
		bne !+
		inc zp_anim_hi
	!:
		lda (zp_anim_lo),y
		cmp #$ff				// #$ff = end of all frames
		bne !skip+

.pc = * "anim_init"
anim_init:
		lda #<anim_frames
		sta zp_anim_lo
		lda #>anim_frames
		sta zp_anim_hi
!skip:	rts

plot_chunk:
		//tax                     // x = number of chars in chunk
		iny
		lda (zp_anim_lo),y
		sta zp_bitmap_lo
		iny
		lda (zp_anim_lo),y
		clc
		adc #>bitmap
		sta zp_bitmap_hi
		iny
		lda (zp_anim_lo),y
		sta zp_char_lo
		sta zp_d800_lo
		iny
		lda (zp_anim_lo),y
		//clc
		adc #>screenram
		sta zp_char_hi
		adc #>(colorram - screenram)
		sta zp_d800_hi

		lda zp_anim_lo
		//clc
		adc #5
		sta zp_anim_lo
		bcc !+
		inc zp_anim_hi
		clc
	!:

plot_next_char:
		ldy #0
	.for (var i = 0; i < 8; i++) {
		lda (zp_anim_lo),y
		sta (zp_bitmap_lo),y
		iny
	}
		lda (zp_anim_lo),y
		sta smc_keep_screencol+1
		iny
		lda (zp_anim_lo),y

		ldy #0
		sta (zp_d800_lo),y
smc_keep_screencol:
		lda #0
		sta (zp_char_lo),y

		lda zp_anim_lo
		//clc
		adc #10
		sta zp_anim_lo
		bcc !+
		clc
		inc zp_anim_hi
	!:
		lda zp_bitmap_lo
		adc #8
		sta zp_bitmap_lo
		bcc !+
		clc
		inc zp_bitmap_hi
	!:
		inc zp_char_lo
		inc zp_d800_lo
		bne !+
		inc zp_char_hi
		inc zp_d800_hi
	!:
		dex
		bne plot_next_char
		jmp next_chunk

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
// ------------------------------
.align $100
.pc = * "t_color_fade"
t_color_fade:
		.fill $100, 0
// ------------------------------
.pc = bitmap "koala_source" virtual
koala_source:
		.fill $2711, 0
// ------------------------------
.pc = animations "anim_frames" virtual
anim_frames:
		.byte 0,$ff
// ------------------------------
.pc = fade_pass_address "fade_pass" virtual
fade_pass:
/*
.C:4800  AE 00 04    LDX $0400
.C:4803  BD 00 09    LDA $0900,X
.C:4806  8D 00 04    STA $0400
.C:4809  AE 00 D8    LDX $D800
.C:480c  BD 00 09    LDA $0900,X
.C:480f  8D 00 D8    STA $D800
*/
	.for (var i=0; i<1000; i++) {
		ldx $0400+i
		lda t_color_fade,x
		sta $0400+i
		ldx $d800+i
		lda t_color_fade,x
		sta $d800+i
	}
		rts
