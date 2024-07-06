SRC=*.go cmd/png2prg/*.go tools/rom_charset_lowercase.prg tools/rom_charset_uppercase.prg
DISPLAYERS=display_koala.prg display_koala_anim.prg display_hires.prg display_hires_anim.prg display_mc_charset.prg display_sc_charset.prg display_mc_sprites.prg display_sc_sprites.prg display_koala_anim_alternative.prg display_mci_bitmap.prg display_mixed_charset.prg display_petscii_charset.prg display_ecm_charset.prg display_mc_charset_anim.prg
ASMLIB=lib.asm
ASM=java -jar ./tools/KickAss-5.25.jar
ASMFLAGS=-showmem -time
X64=x64sc
UPX=upx
UPXFLAGS=--best

LDFLAGS=-s -w
CGO=0
GOBUILDFLAGS=-v -trimpath
TARGET=png2prg
ALLTARGETS=png2prg_linux_amd64 png2prg_linux_arm64 png2prg_darwin_amd64 png2prg_darwin_arm64 png2prg_win_amd64.exe png2prg_win_arm64.exe png2prg_win_x86.exe

FLAGS=-d
FLAGSANIM=-frame-delay 8
FLAGSNG=-d -v -no-guess
FLAGSNG2=-d -v -bitpair-colors 0,-1,-1,-1
FLAGSFORCE=-d -v -bitpair-colors 0,8,10,2
TESTPIC=testdata/mirage_parrot.png
TESTMCI=testdata/mcinterlace/parriot?.png
#TESTSID=testdata/Lift_Off_V2.sid
TESTSID=testdata/512_Rap.sid
#TESTSID=testdata/pocket_universe_8580.sid
TESTSID2=testdata/Snake_Disco.sid
TESTSIDMAD=testdata/madonna/holiday.sid
TESTANIM=testdata/jamesband*.png
TESTSIDANIM=testdata/Nightbreed_-_Dalezy_TRIAD.sid

png2prg: $(SRC) $(DISPLAYERS)
	CGO_ENABLED=$(CGO) go build $(GOBUILDFLAGS) -ldflags="$(LDFLAGS)" -o $@ ./cmd/png2prg

all: $(ALLTARGETS)

bench: $(DISPLAYERS)
	go test -bench Benchmark. -benchmem ./...

dist: $(ALLTARGETS) $(TARGET) readme $(TESTSID) $(TESTSID2) $(TESTSIDMAD) $(TESTSIDANIM)
	mkdir -p dist/testdata
	cp readme.md dist/
	cp $(ALLTARGETS) dist/
	cp testdata/jamesband*.png dist/testdata/
	cp $(TESTPIC) dist/testdata/
	cp $(TESTSID) dist/testdata/
	cp $(TESTSID) $(TESTSIDANIM) dist/testdata/
	cp testdata/Dutch_Breeze_Soft_and_Wet.sid dist/testdata/
	cp -r testdata/evoluer dist/testdata/
	mkdir -p dist/testdata/mcinterlace
	cp -r testdata/mcinterlace/*.png dist/testdata/mcinterlace/
	cp -r testdata/drazlace dist/testdata/
	cp -r testdata/madonna dist/testdata/
	mkdir -p dist/testdata/mixedcharset
	mkdir -p dist/testdata/hirescharset
	mkdir -p dist/testdata/petscii
	cp -r testdata/mixedcharset/*.png dist/testdata/mixedcharset/
	cp -r testdata/mixedcharset/*.gif dist/testdata/mixedcharset/
	cp -r testdata/hirescharset/*.png dist/testdata/hirescharset/
	cp -r testdata/petscii/*.png dist/testdata/petscii/
	./$(TARGET) -d -q -sid $(TESTSID) -o dist/0.orion.prg testdata/ecm/orion.png
	./$(TARGET) -d -q -sid $(TESTSID) -o dist/1.wrath.prg testdata/mixedcharset/joe_wrath.png
	./$(TARGET) -d -q -sid $(TESTSID) -o dist/2.huntress.prg testdata/mixedcharset/huntress.gif
	./$(TARGET) -d -q -sid $(TESTSID) -o dist/3.ohno.prg testdata/hirescharset/ohno_logo.png
	./$(TARGET) -d -q -sid $(TESTSID) -o dist/4.gestalt.prg testdata/hirescharset/gestalt.png
	./$(TARGET) -d -q -sid $(TESTSID) -o dist/5.samar.prg testdata/hirescharset/jetan_samar.png
	./$(TARGET) -d -q -sid $(TESTSID) -o dist/6.apace.prg testdata/mixedcharset/zscs_apace.png
	./$(TARGET) -d -q -sid $(TESTSID) -o dist/7.ocd.prg testdata/mixedcharset/ocd.png
	./$(TARGET) -d -q -sid $(TESTSID) -o dist/8.hyper.prg testdata/mixedcharset/hein_hyper.png
	./$(TARGET) -d -q -sid $(TESTSID) -o dist/9.extend.prg testdata/hirescharset/extend.png
	./$(TARGET) -d -q -sid $(TESTSID) -o dist/10.mega.prg testdata/mixedcharset/sarge_mega.png
	./$(TARGET) -d -q -sid $(TESTSID) -o dist/11.fair.prg testdata/mixedcharset/soya_fair.png
	./$(TARGET) -d -q -sid $(TESTSID) -o dist/12.shine.prg -bpc 3 -npe testdata/mixedcharset/shine.png
	./$(TARGET) -d -q -sid $(TESTSID) -o dist/13.hibiscus.prg testdata/petscii/hein_hibiscus.png
	./$(TARGET) -d -q -sid $(TESTSID) -o dist/14.submarine.prg testdata/petscii/submarine.png
	./$(TARGET) -d -q -sid $(TESTSID) -o dist/15.triad.prg testdata/petscii/triad.png
	./$(TARGET) -d -q -sid $(TESTSID) -o dist/16.proxima.prg testdata/petscii/proxima.png
	./$(TARGET) -d -q -sid $(TESTSID) -o dist/17.artline.prg testdata/petscii/artline.png
	./$(TARGET) -d -q -sid $(TESTSID) -o dist/18.gary.prg testdata/petscii/gary.png
	./$(TARGET) -d -q -sid $(TESTSID) -o dist/19.ernie.prg testdata/petscii/ernie.png
	./$(TARGET) -d -q -sid $(TESTSID) -o dist/20.4nki.prg testdata/petscii/deev_4nki.png
	./$(TARGET) -d -q -sid $(TESTSID) -o dist/21.f4cg.prg testdata/mixedcharset/zscs_f4cg.png
	./$(TARGET) -d -q -sid $(TESTSID) -o dist/22.charsetcompo.prg -bpc 0 testdata/mixedcharset/charsetcompo.png
	./$(TARGET) -d -q -sid $(TESTSID) -o dist/23.neo.prg testdata/mixedcharset/hein_neo.png
	./$(TARGET) -d -q -sid $(TESTSID) -o dist/24.xpardey.prg testdata/ecm/xpardey.png
	./$(TARGET) -d -q -sid $(TESTSID) -o dist/25.pvm.prg testdata/ecm/pvm.png
	./$(TARGET) -d -q -sid $(TESTSID) -o dist/26.rebels.prg testdata/ecm/rebels.png
	./$(TARGET) -d -q -sid $(TESTSID) -o dist/27.dune.prg testdata/ecm/dune.png
	./$(TARGET) -d -q -sid $(TESTSID) -o dist/28.booze.prg testdata/mixedcharset/booze.png
	./$(TARGET) -d -q -sid $(TESTSID) -o dist/29.pretzel.prg testdata/mixedcharset/pretzel.png
	./$(TARGET) -d -q -sid $(TESTSID) -o dist/30.horizon.prg testdata/petscii/horizon.png
	./$(TARGET) -d -q -sid $(TESTSID) -o dist/31.shampoo.prg testdata/ecm/shampoo.png
	./$(TARGET) -d -q -sid $(TESTSID) -o dist/32.phatchar.prg testdata/charanim/phatchar?.png
	rm -f dist/examples.d64
	d64 -add dist/examples.d64 dist/?.*.prg dist/1?.*.prg dist/2?.*.prg dist/3?.*.prg
	rm -f dist/*.prg

.PHONY: dist readme

install: $(TARGET)
	sudo cp $(TARGET) /usr/local/bin/png2prg

displayers: $(DISPLAYERS)

compress: png2prg_linux_amd64.upx png2prg_linux_arm64.upx png2prg_darwin_amd64.upx png2prg_darwin_arm64.upx png2prg_win_amd64.exe.upx png2prg_win_x86.exe.upx

%.prg: %.asm $(ASMLIB)
	$(ASM) $(ASMFLAGS) $< -o $@

%.upx: %
	$(UPX) $(UPXFLAGS) -o $@ $<
	touch $@

png2prg_linux_amd64: $(SRC) $(DISPLAYERS)
	CGO_ENABLED=$(CGO) GOOS=linux GOARCH=amd64 go build $(GOBUILDFLAGS) -ldflags="$(LDFLAGS)" -o $@ ./cmd/png2prg/

png2prg_linux_arm64: $(SRC) $(DISPLAYERS)
	CGO_ENABLED=$(CGO) GOOS=linux GOARCH=arm64 go build $(GOBUILDFLAGS) -ldflags="$(LDFLAGS)" -o $@ ./cmd/png2prg/

png2prg_darwin_amd64: $(SRC) $(DISPLAYERS)
	CGO_ENABLED=$(CGO) GOOS=darwin GOARCH=amd64 go build $(GOBUILDFLAGS) -ldflags="$(LDFLAGS)" -o $@ ./cmd/png2prg/

png2prg_darwin_arm64: $(SRC) $(DISPLAYERS)
	CGO_ENABLED=$(CGO) GOOS=darwin GOARCH=arm64 go build $(GOBUILDFLAGS) -ldflags="$(LDFLAGS)" -o $@ ./cmd/png2prg/

png2prg_win_amd64.exe: $(SRC) $(DISPLAYERS)
	CGO_ENABLED=$(CGO) GOOS=windows GOARCH=amd64 go build $(GOBUILDFLAGS) -ldflags="$(LDFLAGS)" -o $@ ./cmd/png2prg/

png2prg_win_arm64.exe: $(SRC) $(DISPLAYERS)
	CGO_ENABLED=$(CGO) GOOS=windows GOARCH=arm64 go build $(GOBUILDFLAGS) -ldflags="$(LDFLAGS)" -o $@ ./cmd/png2prg/

png2prg_win_x86.exe: $(SRC) $(DISPLAYERS)
	CGO_ENABLED=$(CGO) GOOS=windows GOARCH=386 go build $(GOBUILDFLAGS) -ldflags="$(LDFLAGS)" -o $@ ./cmd/png2prg/

readme: $(TARGET)
	./$(TARGET) -q -h >readme.md 2>&1

roms: rom_charset_lowercase.prg rom_charset_uppercase.prg

rom_charset_lowercase.prg: testdata/rom_charset_lowercase.png $(TARGET)
	./$(TARGET) -np -m sccharset -bpc 0 -o $@ $<

rom_charset_uppercase.prg: testdata/rom_charset_uppercase.png $(TARGET)
	./$(TARGET) -np -m sccharset -bpc 0 -o $@ $<

test: $(TARGET) $(TESTPIC) $(TESTSID)
	./$(TARGET) $(FLAGS) -o q.prg -sid $(TESTSID) $(TESTPIC)
	$(X64) q.prg >/dev/null

testmci: $(TARGET) $(TESTMCI) $(TESTSID)
	./$(TARGET) $(FLAGS) -o q.prg -i -sid $(TESTSID) $(TESTMCI)
	$(X64) q.prg >/dev/null

testmadonna: $(TARGET) $(TESTPIC) $(TESTSIDMAD)
	./$(TARGET) $(FLAGS) -o q.prg -i -sid $(TESTSIDMAD) testdata/madonna/cjam_pure_madonna.png
	$(X64) q.prg >/dev/null

testanim: $(TARGET) $(TESTANIM) $(TESTSIDANIM)
	./$(TARGET) $(FLAGS) $(FLAGSANIM) -sid $(TESTSIDANIM) -o q.prg $(TESTANIM)
	$(X64) q.prg >/dev/null

evoluer: $(TARGET)
	./$(TARGET) $(FLAGS) -frame-delay 4 -o q.prg -sid testdata/evoluer/Evoluer.sid testdata/evoluer/PIC??.png
	$(X64) q.prg >/dev/null

testpack: $(TARGET)
	./$(TARGET) $(FLAGS) -nc -np -i -o q.prg $(TESTPIC)
	exomizer sfx basic -q -o qq_guess.sfx.exo q.prg
	dali --sfx 2082 -o qq_guess.sfx.dali q.prg
	./$(TARGET) $(FLAGSNG) -nc -np -i -o q.prg $(TESTPIC)
	exomizer sfx basic -q -o qq_noguess.sfx.exo q.prg
	dali --sfx 2082 -o qq_noguess.sfx.dali q.prg
	./$(TARGET) $(FLAGSNG2) -nc -np -i -o q.prg $(TESTPIC)
	exomizer sfx basic -q -o qq_noguess2.sfx.exo q.prg
	dali --sfx 2082 -o qq_noguess2.sfx.dali q.prg
	./$(TARGET) $(FLAGSFORCE) -nc -np -i -o q.prg $(TESTPIC)
	exomizer sfx basic -q -o qq_force_manual_colors.sfx.exo q.prg
	dali --sfx 2082 -o qq_force_manual_colors.sfx.dali q.prg
	./$(TARGET) $(FLAGS) -i -o q.prg $(TESTPIC)
	$(X64) qq_guess.sfx.exo >/dev/null

# https://csdb.dk/forums/?roomid=13&topicid=38311&showallposts=1
FLORIS=testdata/floris_untitled.png
MERMAID=testdata/mermaid_song_of_the_sunset.png
MERMAID2=testdata/mermaid_weee.png
SHORTCIRCUIT=testdata/short_circuit.png
STE=testdata/ste_gng.png
SANDER=testdata/sander_ld.png
SULEVI=testdata/sulevi_daylight.png
benchkoala: floris mermaid shortcircuit ste mermaid2 sander sulevi robinlevy veto miscpic jonegg leon talent cisco yiear hend
	ls -l *_p2p.prg* *_spot.kla*

MISCPIC=testdata/carrion_still_waiting.png
MISCPIC=testdata/the_sarge_obscena_vaselina_palette1.png
MISCPIC=testdata/archmage_mc_god.png
MISCPIC=testdata/facet_turning_point_320x200.png
MISCPIC=testdata/bizzmo_wool_colodore.png
MISCPIC=testdata/joe_earth.png
MISCPIC=testdata/tentacles.png
#MISCPIC=testdata/hend_temple320x200.png
#MISCPIC=testdata/the_sarge_therapy.png
#MISCPIC=testdata/focuslogo320x200.png
MISCPIC=testdata/cisco_heat.png
#MISCPIC=testdata/yiearkungfu.png
#MISCPIC=testdata/the_sarge_lee320x200.png
#MISCPIC=testdata/mikael_pretzelpilami320x200.png
#MISCPIC=testdata/veto_eye320x200.png
#MISCPIC=testdata/jonegg_tapper320x200.png

P2PBENCHOPTS=-v -bf

# bestbf: -bpc 0,11,14,6
# best: -bpc 0,5,11,6 -nbc
floris: $(FLORIS) $(TARGET)
	spot13 $< -o floris_spot.kla
	dali -o floris_spot.kla.dali floris_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o floris_p2p.prg $<
	./$(TARGET) -bpc 0,5,11,6 -nbc -o floris_p2pbest.prg $<
	dali -o floris_p2p.prg.dali floris_p2p.prg
	dali -o floris_p2pbest.prg.dali floris_p2pbest.prg
	ls -l floris*

#best: -bpc 0,1,12,4
mermaid: $(MERMAID) $(TARGET)
	spot13 $< -o mermaid_spot.kla
	dali -o mermaid_spot.kla.dali mermaid_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o mermaid_p2p.prg $<
	./$(TARGET) -bpc 0,1,12,4 -o mermaid_p2pbest.prg $<
	dali -o mermaid_p2p.prg.dali mermaid_p2p.prg
	dali -o mermaid_p2pbest.prg.dali mermaid_p2pbest.prg
	ls -l mermaid_*

#best bf: -bpc 1,6,11,0
#: -bpc 1,11,14,3
shortcircuit: $(SHORTCIRCUIT) $(TARGET)
	spot13 $< -o shortcircuit_spot.kla
	dali -o shortcircuit_spot.kla.dali shortcircuit_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o shortcircuit_p2p.prg $<
	./$(TARGET) -bpc 1,6,11,0 -o shortcircuit_p2pbest.prg $<
	#./$(TARGET) -bpc 1,14,11,6 -o shortcircuit_p2p.prg $<
	dali -o shortcircuit_p2p.prg.dali shortcircuit_p2p.prg
	dali -o shortcircuit_p2pbest.prg.dali shortcircuit_p2pbest.prg
	ls -l shortcircuit*

# bruteforce: -bpc 0,6,4,7
# best-bf-nbc: -bpc 0,9,1,3 -nbc
sander: $(SANDER) $(TARGET)
	spot13 $< -o sander_spot.kla
	dali -o sander_spot.kla.dali sander_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o sander_p2p.prg $<
	./$(TARGET) -bpc 0,9,1,3 -nbc -o sander_p2pbest.prg $<
	dali -o sander_p2p.prg.dali sander_p2p.prg
	dali -o sander_p2pbest.prg.dali sander_p2pbest.prg
	ls -l sander*

# bruteforce: -bpc 0,2,14,11
ste: $(STE) $(TARGET)
	spot13 $< -o ste_spot.kla
	dali -o ste_spot.kla.dali ste_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o ste_p2p.prg $<
	./$(TARGET) -bpc 0,2,14,11 -o ste_p2pbest.prg $<
	dali -o ste_p2p.prg.dali ste_p2p.prg
	dali -o ste_p2pbest.prg.dali ste_p2pbest.prg
	ls -l ste*

#best: -bpc 15,12,3,5
#15,5,12,3
#15,3,10,5
mermaid2: $(MERMAID2) $(TARGET)
	spot13 $< -o mermaid2_spot.kla
	dali -o mermaid2_spot.kla.dali mermaid2_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o mermaid2_p2p.prg $<
	./$(TARGET) -bpc 15,12,3,5 -o mermaid2_p2pbest.prg $<
	dali -o mermaid2_p2p.prg.dali mermaid2_p2p.prg
	dali -o mermaid2_p2pbest.prg.dali mermaid2_p2pbest.prg
	ls -l mermaid2*

# best bf: -bpc 3,13,1,7
sulevi: $(SULEVI) $(TARGET)
	spot13 $< -o sulevi_spot.kla
	dali -o sulevi_spot.kla.dali sulevi_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o sulevi_p2p.prg $<
	./$(TARGET) -bpc 3,13,1,7 -o sulevi_p2pbest.prg $<
	Png2prg-1.6 -o sulevi_p2p16.prg $<
	dali -o sulevi_p2p.prg.dali sulevi_p2p.prg
	dali -o sulevi_p2p16.prg.dali sulevi_p2p16.prg
	dali -o sulevi_p2pbest.prg.dali sulevi_p2pbest.prg
	ls -l sulevi*

# best bf: -bpc 0,12,9,11
# default: -bpc 0,15,12,11
ROBIN=testdata/robinlevy_deadlock.png
# best bf: 0,14,15,3
#ROBIN=testdata/robinlevy_huntersmoon.png
robinlevy: $(ROBIN) $(TARGET)
	spot13 $< -o robin_spot.kla
	dali -o robin_spot.kla.dali robin_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o robin_p2p.prg $<
	./$(TARGET) -bpc 0,12,9,11 -o robin_p2pbest.prg $<
	#./$(TARGET) -bpc 0,14,15,3 -o robin_p2pbest.prg $<
	dali -o robin_p2p.prg.dali robin_p2p.prg
	dali -o robin_p2pbest.prg.dali robin_p2pbest.prg
	ls -l robin*

# best: -bpc 9,0,6,5 (also bf, but non deterministic)
# best bf: 9,15,10,0
# default: 9,0,12,15

# best -bf -npcc: -bpc 9,15,10,5 -npcc
VETO=testdata/veto_room_with_view.png
veto: $(VETO) $(TARGET)
	spot13 $< -o veto_spot.kla
	dali -o veto_spot.kla.dali veto_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o veto_p2p.prg $<
	./$(TARGET) -bpc 9,10,15,5 -npcc -o veto_p2pbest.prg $<
	dali -o veto_p2p.prg.dali veto_p2p.prg
	dali -o veto_p2pbest.prg.dali veto_p2pbest.prg
	ls -l veto*

# best -bf: -bpc 8,0,6,5
LEON=testdata/leon_solar.png
leon: $(LEON) $(TARGET)
	spot13 $< -o leon_spot.kla
	dali -o leon_spot.kla.dali leon_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o leon_p2p.prg $<
	./$(TARGET) -bpc 8,0,6,5 -o leon_p2pbest.prg $<
	dali -o leon_p2p.prg.dali leon_p2p.prg
	dali -o leon_p2pbest.prg.dali leon_p2pbest.prg
	ls -l leon_*

miscpic: $(MISCPIC) $(TARGET)
	spot13 $< -o misc_spot.kla
	dali -o misc_spot.kla.dali misc_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o misc_p2p.prg $<
	#Png2prg-1.6 -v -o misc_p2p.prg $<
	#./$(TARGET) -v -o misc_p2p.prg $<
	dali -o misc_p2p.prg.dali misc_p2p.prg
	Png2prg-1.6 -v -o misc_p2p16.prg $<
	dali -o misc_p2p16.prg.dali misc_p2p16.prg
	ls -l misc*

#
# best default: -bpc 14,0,15,11
# best: -bpc 14,8,7,0 -bf -npcc
JONEGG=testdata/jonegg_thanos320x200.png
jonegg: $(JONEGG) $(TARGET)
	spot13 $< -o jonegg_spot.kla
	dali -o jonegg_spot.kla.dali jonegg_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o jonegg_p2p.prg $<
	dali -o jonegg_p2p.prg.dali jonegg_p2p.prg
	./$(TARGET) -v -bpc 14,8,7,0 -npcc -o jonegg_p2pbest.prg $<
	dali -o jonegg_p2pbest.prg.dali jonegg_p2pbest.prg
	Png2prg-1.6 -v -o jonegg_p2p16.prg $<
	dali -o jonegg_p2p16.prg.dali jonegg_p2p16.prg
	ls -l jonegg_*

# best bf: -bpc 11,12,15,10
TALENT=testdata/talent_vangelis320x200.png
talent: $(TALENT) $(TARGET)
	spot13 $< -o talent_spot.kla
	dali -o talent_spot.kla.dali talent_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o talent_p2p.prg $<
	./$(TARGET) -v -o talent_p2pbest.prg -bpc 11,12,15,10 $<
	Png2prg-1.6 -v -o talent_p2p16.prg $<
	dali -o talent_p2p16.prg.dali talent_p2p16.prg
	dali -o talent_p2pbest.prg.dali talent_p2pbest.prg
	dali -o talent_p2p.prg.dali talent_p2p.prg
	ls -l talent_*

# best bf: -bpc 0,8,6,2
CISCO=testdata/cisco_heat.png
cisco: $(CISCO) $(TARGET)
	spot13 $< -o cisco_spot.kla
	dali -o cisco_spot.kla.dali cisco_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o cisco_p2p.prg $<
	./$(TARGET) -v -bpc 0,8,6,2 -o cisco_p2pbest.prg $<
	Png2prg-1.6 -v -o cisco_p2p16.prg $<
	dali -o cisco_p2p16.prg.dali cisco_p2p16.prg
	dali -o cisco_p2pbest.prg.dali cisco_p2pbest.prg
	dali -o cisco_p2p.prg.dali cisco_p2p.prg
	ls -l cisco_*

# best bf: -bpc 6,8,0,15
YIEAR=testdata/yiearkungfu.png
yiear: $(YIEAR) $(TARGET)
	spot13 $< -o yiear_spot.kla
	dali -o yiear_spot.kla.dali yiear_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o yiear_p2p.prg $<
	./$(TARGET) -v -bpc 6,8,0,15 -o yiear_p2pbest.prg $<
	Png2prg-1.6 -v -o yiear_p2p16.prg $<
	dali -o yiear_p2p16.prg.dali yiear_p2p16.prg
	dali -o yiear_p2pbest.prg.dali yiear_p2pbest.prg
	dali -o yiear_p2p.prg.dali yiear_p2p.prg
	ls -l yiear_*

# best bf: -bpc 12,1,9,11
HEND=testdata/hend_temple320x200.png
hend: $(HEND) $(TARGET)
	spot13 $< -o hend_spot.kla
	dali -o hend_spot.kla.dali hend_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o hend_p2p.prg $<
	./$(TARGET) -v -bpc 12,1,9,11 -o hend_p2pbest.prg $<
	Png2prg-1.6 -v -o hend_p2p16.prg $<
	dali -o hend_p2p16.prg.dali hend_p2p16.prg
	dali -o hend_p2pbest.prg.dali hend_p2pbest.prg
	dali -o hend_p2p.prg.dali hend_p2p.prg
	ls -l hend_*

clean:
	rm -f $(ALLTARGETS) $(TARGET) q*.prg display*.prg *.exo *.dali *.upx *.sym *_p2p.prg *_p2pbest.prg *_spot.kla *_p2p16.prg
	rm -rf dist
