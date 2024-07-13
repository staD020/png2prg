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
	mkdir -p dist/testdata/charanim
	cp -r testdata/mixedcharset/*.png dist/testdata/mixedcharset/
	cp -r testdata/mixedcharset/*.gif dist/testdata/mixedcharset/
	cp -r testdata/hirescharset/*.png dist/testdata/hirescharset/
	cp -r testdata/petscii/*.png dist/testdata/petscii/
	cp -r testdata/charanim/*.png dist/testdata/charanim/
	cp testdata/floris_untitled.png dist/testdata/
	cp testdata/floris_untitled.png dist/testdata/
	cp testdata/short_circuit.png dist/testdata/
	cp testdata/sander_ld.png dist/testdata/
	cp testdata/mermaid_weee.png dist/testdata/
	cp testdata/robinlevy_deadlock.png dist/testdata/
	cp testdata/veto_room_with_view.png dist/testdata/
	cp testdata/talent_vangelis320x200.png dist/testdata/
	cp testdata/hend_temple320x200.png dist/testdata/
	cp testdata/jonegg_thanos320x200.png dist/testdata/
	cp testdata/leon_solar.png dist/testdata/
	cp testdata/cisco_heat.png dist/testdata/
	cp testdata/sulevi_daylight.png dist/testdata/
	cp testdata/yiearkungfu.png dist/testdata/
	cp testdata/the_sarge_lee320x200.png dist/testdata/
	cp testdata/mirage_parrot320x200.png dist/testdata/
	cp testdata/dragonslair320x200.png dist/testdata/
	cp testdata/sir_scorpion320x200.png dist/testdata/
	./$(TARGET) -d -q -bf -nbc -o dist/01.floris.prg testdata/floris_untitled.png
	./$(TARGET) -d -q -bf -o dist/02.mermaid.prg testdata/mermaid_song_of_the_sunset.png
	./$(TARGET) -d -q -bf -o dist/03.shortcircuit.prg testdata/short_circuit.png
	./$(TARGET) -d -q -bf -nbc -o dist/04.sander.prg testdata/sander_ld.png
	./$(TARGET) -d -q -bpc 15,12,3,5 -o dist/05.mermaid2.prg testdata/mermaid_weee.png
	./$(TARGET) -d -q -bf -o dist/06.robinlevy.prg testdata/robinlevy_deadlock.png
	./$(TARGET) -d -q -bf -npcc -o dist/07.veto.prg testdata/veto_room_with_view.png
	./$(TARGET) -d -q -bf -o dist/08.talent.prg testdata/talent_vangelis320x200.png
	./$(TARGET) -d -q -bf -o dist/09.hend.prg testdata/hend_temple320x200.png
	./$(TARGET) -d -q -bf -npcc -o dist/10.jonegg.prg testdata/jonegg_thanos320x200.png
	./$(TARGET) -d -q -bf -o dist/11.leon.prg testdata/leon_solar.png
	./$(TARGET) -d -q -bf -o dist/12.ciscoheat.prg testdata/cisco_heat.png
	./$(TARGET) -d -q -bf -o dist/13.sulevi.prg testdata/sulevi_daylight.png
	./$(TARGET) -d -q -bf -o dist/14.yiear.prg testdata/yiearkungfu.png
	./$(TARGET) -d -q -bf -nbc -o dist/15.thesarge.prg testdata/the_sarge_lee320x200.png
	./$(TARGET) -d -q -bf -o dist/16.mirage.prg testdata/mirage_parrot320x200.png
	./$(TARGET) -d -q -bf -o dist/17.dragonslair.prg testdata/dragonslair320x200.png
	./$(TARGET) -d -q -bf -nbc -o dist/18.scorpion.prg testdata/sir_scorpion320x200.png
	./$(TARGET) -d -q -o dist/phatchar.prg testdata/charanim/phatchar*.png
	rm -f dist/examples.d64
	d64 -add dist/examples.d64 dist/0?.*.prg dist/1?.*.prg dist/phatchar.prg
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
benchkoala: floris mermaid shortcircuit ste mermaid2 sander sulevi robinlevy veto miscpic jonegg leon talent cisco yiear hend sarge mirage dragon scorpion
	ls -l *_p2p.prg* *_spot.kla*

#MISCPIC=testdata/carrion_still_waiting.png
#MISCPIC=testdata/the_sarge_obscena_vaselina_palette1.png
#MISCPIC=testdata/archmage_mc_god.png
#MISCPIC=testdata/facet_turning_point_320x200.png
#MISCPIC=testdata/bizzmo_wool_colodore.png
#MISCPIC=testdata/joe_earth.png
#MISCPIC=testdata/tentacles.png
#MISCPIC=testdata/hend_temple320x200.png
#MISCPIC=testdata/the_sarge_therapy.png
#MISCPIC=testdata/focuslogo320x200.png
#MISCPIC=testdata/cisco_heat.png
#MISCPIC=testdata/yiearkungfu.png
#MISCPIC=testdata/the_sarge_lee320x200.png
#MISCPIC=testdata/mikael_pretzelpilami320x200.png
#MISCPIC=testdata/veto_eye320x200.png
#MISCPIC=testdata/jonegg_tapper320x200.png
#MISCPIC=testdata/fungus/scorpionpic/vice320x200.png
#MISCPIC=testdata/fungus/steel/vice320x200.png
#MISCPIC=testdata/mirage_culture320x200.png
MISCPIC=testdata/joe_hatching320x200.png

P2PBENCHOPTS=-bf

# bestbf: -bpc 0,11,14,6
# best: -bpc 0,5,11,6 -nbc
floris: $(FLORIS) $(TARGET)
	spot13 $< -o $@_spot.kla
	dali -o $@_spot.kla.dali $@_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o $@_p2p.prg $<
	./$(TARGET) -bpc 0,5,11,6 -nbc -o $@_p2pbest.prg $<
	Png2prg-1.6 -o $@_p2p16.prg $<
	dali -o $@_p2p16.prg.dali $@_p2p16.prg
	dali -o $@_p2pbest.prg.dali $@_p2pbest.prg
	dali -o $@_p2p.prg.dali $@_p2p.prg
	ls -l $@_*

#best: -bpc 0,1,12,4
mermaid: $(MERMAID) $(TARGET)
	spot13 $< -o $@_spot.kla
	dali -o $@_spot.kla.dali $@_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o $@_p2p.prg $<
	./$(TARGET) -bpc 0,1,12,4 -o $@_p2pbest.prg $<
	Png2prg-1.6 -o $@_p2p16.prg $<
	dali -o $@_p2p16.prg.dali $@_p2p16.prg
	dali -o $@_p2pbest.prg.dali $@_p2pbest.prg
	dali -o $@_p2p.prg.dali $@_p2p.prg
	ls -l $@_*

#best bf: -bpc 1,6,11,0
#: -bpc 1,11,14,3
shortcircuit: $(SHORTCIRCUIT) $(TARGET)
	spot13 $< -o $@_spot.kla
	dali -o $@_spot.kla.dali $@_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o $@_p2p.prg $<
	./$(TARGET) -bpc 1,6,11,0 -o $@_p2pbest.prg $<
	Png2prg-1.6 -o $@_p2p16.prg $<
	dali -o $@_p2p16.prg.dali $@_p2p16.prg
	dali -o $@_p2pbest.prg.dali $@_p2pbest.prg
	dali -o $@_p2p.prg.dali $@_p2p.prg
	ls -l $@_*

# bruteforce: -bpc 0,6,4,7
# best-bf-nbc: -bpc 0,9,1,3 -nbc
sander: $(SANDER) $(TARGET)
	spot13 $< -o $@_spot.kla
	dali -o $@_spot.kla.dali $@_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o $@_p2p.prg $<
	./$(TARGET) -bpc 0,9,1,3 -nbc -o $@_p2pbest.prg $<
	Png2prg-1.6 -o $@_p2p16.prg $<
	dali -o $@_p2p16.prg.dali $@_p2p16.prg
	dali -o $@_p2pbest.prg.dali $@_p2pbest.prg
	dali -o $@_p2p.prg.dali $@_p2p.prg
	ls -l $@_*

# bruteforce: -bpc 0,2,14,11
# bf npcc winner: -bpc 0,6,14,1 -npcc
ste: $(STE) $(TARGET)
	spot13 $< -o $@_spot.kla
	dali -o $@_spot.kla.dali $@_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o $@_p2p.prg $<
	./$(TARGET) -bpc 0,6,14,1 -npcc -o $@_p2pbest.prg $<
	Png2prg-1.6 -o $@_p2p16.prg $<
	dali -o $@_p2p16.prg.dali $@_p2p16.prg
	dali -o $@_p2pbest.prg.dali $@_p2pbest.prg
	dali -o $@_p2p.prg.dali $@_p2p.prg
	ls -l $@_*

#best: -bpc 15,12,3,5
#15,5,12,3
#15,3,10,5
mermaid2: $(MERMAID2) $(TARGET)
	spot13 $< -o $@_spot.kla
	dali -o $@_spot.kla.dali $@_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o $@_p2p.prg $<
	./$(TARGET) -bpc 15,12,3,5 -o $@_p2pbest.prg $<
	Png2prg-1.6 -o $@_p2p16.prg $<
	dali -o $@_p2p16.prg.dali $@_p2p16.prg
	dali -o $@_p2pbest.prg.dali $@_p2pbest.prg
	dali -o $@_p2p.prg.dali $@_p2p.prg
	ls -l $@_*

# best bf: -bpc 3,13,1,7
sulevi: $(SULEVI) $(TARGET)
	spot13 $< -o $@_spot.kla
	dali -o $@_spot.kla.dali $@_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o $@_p2p.prg $<
	./$(TARGET) -bpc 3,13,1,7 -o $@_p2pbest.prg $<
	Png2prg-1.6 -o $@_p2p16.prg $<
	dali -o $@_p2p16.prg.dali $@_p2p16.prg
	dali -o $@_p2pbest.prg.dali $@_p2pbest.prg
	dali -o $@_p2p.prg.dali $@_p2p.prg
	ls -l $@_*

# best bf: -bpc 0,12,9,11
# default: -bpc 0,15,12,11
ROBIN=testdata/robinlevy_deadlock.png
# best bf: 0,14,15,3
#ROBIN=testdata/robinlevy_huntersmoon.png
robinlevy: $(ROBIN) $(TARGET)
	spot13 $< -o $@_spot.kla
	dali -o $@_spot.kla.dali $@_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o $@_p2p.prg $<
	./$(TARGET) -bpc 0,12,9,11 -o $@_p2pbest.prg $<
	Png2prg-1.6 -o $@_p2p16.prg $<
	dali -o $@_p2p16.prg.dali $@_p2p16.prg
	dali -o $@_p2pbest.prg.dali $@_p2pbest.prg
	dali -o $@_p2p.prg.dali $@_p2p.prg
	ls -l $@_*

# best: -bpc 9,0,6,5 (also bf, but non deterministic)
# best bf: 9,15,10,0
# default: 9,0,12,15

# best -bf -npcc: -bpc 9,10,15,5 -npcc
VETO=testdata/veto_room_with_view.png
veto: $(VETO) $(TARGET)
	spot13 $< -o $@_spot.kla
	dali -o $@_spot.kla.dali $@_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o $@_p2p.prg $<
	./$(TARGET) -bpc 9,10,15,5 -npcc -o $@_p2pbest.prg $<
	Png2prg-1.6 -o $@_p2p16.prg $<
	dali -o $@_p2p16.prg.dali $@_p2p16.prg
	dali -o $@_p2pbest.prg.dali $@_p2pbest.prg
	dali -o $@_p2p.prg.dali $@_p2p.prg
	ls -l $@_*

# best -bf: -bpc 8,0,6,5
LEON=testdata/leon_solar.png
leon: $(LEON) $(TARGET)
	spot13 $< -o $@_spot.kla
	dali -o $@_spot.kla.dali $@_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o $@_p2p.prg $<
	./$(TARGET) -bpc 8,0,6,5 -o $@_p2pbest.prg $<
	Png2prg-1.6 -o $@_p2p16.prg $<
	dali -o $@_p2p16.prg.dali $@_p2p16.prg
	dali -o $@_p2pbest.prg.dali $@_p2pbest.prg
	dali -o $@_p2p.prg.dali $@_p2p.prg
	ls -l $@_*

miscpic: $(MISCPIC) $(TARGET)
	spot13 $< -o $@_spot.kla
	dali -o $@_spot.kla.dali $@_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o $@_p2p.prg $<
	dali -o $@_p2p.prg.dali $@_p2p.prg
	 ./$(TARGET) -o $@_p2pdefault.prg $<
	dali -o $@_p2pdefault.prg.dali $@_p2pdefault.prg
	Png2prg-1.6 -o $@_p2p16.prg $<
	dali -o $@_p2p16.prg.dali $@_p2p16.prg
	ls -l $@_*

#
# best default: -bpc 14,0,15,11
# best bf npcc: -bpc 14,8,7,0 -npcc
JONEGG=testdata/jonegg_thanos320x200.png
jonegg: $(JONEGG) $(TARGET)
	spot13 $< -o $@_spot.kla
	dali -o $@_spot.kla.dali $@_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o $@_p2p.prg $<
	./$(TARGET) -bpc 14,8,7,0 -npcc -o $@_p2pbest.prg $<
	Png2prg-1.6 -o $@_p2p16.prg $<
	dali -o $@_p2p16.prg.dali $@_p2p16.prg
	dali -o $@_p2pbest.prg.dali $@_p2pbest.prg
	dali -o $@_p2p.prg.dali $@_p2p.prg
	ls -l $@_*

# best bf: -bpc 11,12,15,10
TALENT=testdata/talent_vangelis320x200.png
talent: $(TALENT) $(TARGET)
	spot13 $< -o $@_spot.kla
	dali -o $@_spot.kla.dali $@_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o $@_p2p.prg $<
	./$(TARGET) -bpc 11,12,15,10 -o $@_p2pbest.prg $<
	Png2prg-1.6 -o $@_p2p16.prg $<
	dali -o $@_p2p16.prg.dali $@_p2p16.prg
	dali -o $@_p2pbest.prg.dali $@_p2pbest.prg
	dali -o $@_p2p.prg.dali $@_p2p.prg
	ls -l $@_*

# best bf: -bpc 0,8,6,2
CISCO=testdata/cisco_heat.png
cisco: $(CISCO) $(TARGET)
	spot13 $< -o $@_spot.kla
	dali -o $@_spot.kla.dali $@_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o $@_p2p.prg $<
	./$(TARGET) -bpc 0,8,6,2 -o $@_p2pbest.prg $<
	Png2prg-1.6 -o $@_p2p16.prg $<
	dali -o $@_p2p16.prg.dali $@_p2p16.prg
	dali -o $@_p2pbest.prg.dali $@_p2pbest.prg
	dali -o $@_p2p.prg.dali $@_p2p.prg
	ls -l $@_*

# best bf: -bpc 6,8,0,15
YIEAR=testdata/yiearkungfu.png
yiear: $(YIEAR) $(TARGET)
	spot13 $< -o $@_spot.kla
	dali -o $@_spot.kla.dali $@_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o $@_p2p.prg $<
	./$(TARGET) -bpc 6,8,0,15 -o $@_p2pbest.prg $<
	Png2prg-1.6 -o $@_p2p16.prg $<
	dali -o $@_p2p16.prg.dali $@_p2p16.prg
	dali -o $@_p2pbest.prg.dali $@_p2pbest.prg
	dali -o $@_p2p.prg.dali $@_p2p.prg
	ls -l $@_*

# best bf: -bpc 12,1,9,11
HEND=testdata/hend_temple320x200.png
hend: $(HEND) $(TARGET)
	spot13 $< -o $@_spot.kla
	dali -o $@_spot.kla.dali $@_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o $@_p2p.prg $<
	./$(TARGET) -bpc 12,1,9,11 -o $@_p2pbest.prg $<
	Png2prg-1.6 -o $@_p2p16.prg $<
	dali -o $@_p2p16.prg.dali $@_p2p16.prg
	dali -o $@_p2pbest.prg.dali $@_p2pbest.prg
	dali -o $@_p2p.prg.dali $@_p2p.prg
	ls -l $@_*

# best bf -nbc: -bpc 0,5,4,6 -nbc
SARGE=testdata/the_sarge_lee320x200.png
sarge: $(SARGE) $(TARGET)
	spot13 $< -o $@_spot.kla
	dali -o $@_spot.kla.dali $@_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o $@_p2p.prg $<
	./$(TARGET) -bpc 0,5,4,6 -nbc -o $@_p2pbest.prg $<
	Png2prg-1.6 -o $@_p2p16.prg $<
	dali -o $@_p2p16.prg.dali $@_p2p16.prg
	dali -o $@_p2pbest.prg.dali $@_p2pbest.prg
	dali -o $@_p2p.prg.dali $@_p2p.prg
	ls -l $@_*

# best bf: -bpc 15,4,9,7
MIRAGE=testdata/mirage_parrot320x200.png
mirage: $(MIRAGE) $(TARGET)
	spot13 $< -o $@_spot.kla
	dali -o $@_spot.kla.dali $@_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o $@_p2p.prg $<
	./$(TARGET) -bpc 15,4,9,7 -o $@_p2pbest.prg $<
	Png2prg-1.6 -o $@_p2p16.prg $<
	dali -o $@_p2p16.prg.dali $@_p2p16.prg
	dali -o $@_p2pbest.prg.dali $@_p2pbest.prg
	dali -o $@_p2p.prg.dali $@_p2p.prg
	ls -l $@_*

# best bf: -bpc 0,12,11,2
DRAGON=testdata/dragonslair320x200.png
dragon: $(DRAGON) $(TARGET)
	spot13 $< -o $@_spot.kla
	dali -o $@_spot.kla.dali $@_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o $@_p2p.prg $<
	dali -o $@_p2p.prg.dali $@_p2p.prg
	./$(TARGET) -bpc 0,12,11,2 -o $@_p2pbest.prg $<
	dali -o $@_p2pbest.prg.dali $@_p2pbest.prg
	Png2prg-1.6 -o $@_p2p16.prg $<
	dali -o $@_p2p16.prg.dali $@_p2p16.prg
	ls -l $@_*

SCORPION=testdata/sir_scorpion320x200.png
scorpion:$(SCORPION) $(TARGET)
	spot13 $< -o $@_spot.kla
	dali -o $@_spot.kla.dali $@_spot.kla
	./$(TARGET) $(P2PBENCHOPTS) -o $@_p2p.prg $<
	dali -o $@_p2p.prg.dali $@_p2p.prg
	./$(TARGET) -bf -nbc -o $@_p2pbest.prg $<
	dali -o $@_p2pbest.prg.dali $@_p2pbest.prg
	Png2prg-1.6 -o $@_p2p16.prg $<
	dali -o $@_p2p16.prg.dali $@_p2p16.prg
	ls -l $@_*

clean:
	rm -f $(ALLTARGETS) $(TARGET) q*.prg display*.prg *.exo *.dali *.upx *.sym *_p2p.prg *_p2pbest.prg *_spot.kla *_p2p16.prg
	rm -rf dist
