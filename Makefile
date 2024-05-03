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
benchkoala: floris mermaid shortcircuit ste mermaid2 sander sulevi robinlevy veto miscpic
	ls -l *_p2p.prg* *_spot.kla*

# best: -bpc 0,5,11,6
floris: $(FLORIS) $(TARGET)
	spot13 $< -o floris_spot.kla
	dali -o floris_spot.kla.dali floris_spot.kla
	./$(TARGET) -v -bf -o floris_p2p.prg $<
	#./$(TARGET) -bpc 0,5,11,6 -o floris_p2p.prg $<
	#./$(TARGET) -bf -o floris_p2p.prg $<
	dali -o floris_p2p.prg.dali floris_p2p.prg
	ls -l floris*

#:  -bpc 0,7,4,12
#best: -bpc 0,6,1,12
mermaid: $(MERMAID) $(TARGET)
	spot13 $< -o mermaid_spot.kla
	dali -o mermaid_spot.kla.dali mermaid_spot.kla
	./$(TARGET) -v -bf -o mermaid_p2p.prg $<
	#./$(TARGET) -bpc 0,6,1,12 -o mermaid_p2p.prg $<
	#./$(TARGET) -bpc 0,6,4,7 -o mermaid_p2p.prg $<
	#./$(TARGET) -bf -o mermaid_p2p.prg $<
	dali -o mermaid_p2p.prg.dali mermaid_p2p.prg
	ls -l mermaid_*

#best: -bpc 1,14,11,6
#: -bpc 1,11,14,3
shortcircuit: $(SHORTCIRCUIT) $(TARGET)
	spot13 $< -o shortcircuit_spot.kla
	dali -o shortcircuit_spot.kla.dali shortcircuit_spot.kla
	./$(TARGET) -v -bf -o shortcircuit_p2p.prg $<
	#./$(TARGET) -bpc 1,14,11,6 -o shortcircuit_p2p.prg $<
	#./$(TARGET) -bpc 1,11,14,3 -o shortcircuit_p2p.prg $<
	#./$(TARGET) -bf -o shortcircuit_p2p.prg $<
	dali -o shortcircuit_p2p.prg.dali shortcircuit_p2p.prg
	ls -l shortcircuit*

# bruteforce is best: -bpc 0,9,1,3
# default: -bpc 0,4,6,11
sander: $(SANDER) $(TARGET)
	spot13 $< -o sander_spot.kla
	dali -o sander_spot.kla.dali sander_spot.kla
	./$(TARGET) -v -bf -o sander_p2p.prg $<
	#./$(TARGET) -bpc 0,9,1,3 -o sander_p2p.prg $<
	#./$(TARGET) -bpc 0,4,6,11 -o sander_p2p.prg $<
	#./$(TARGET) -bf -o sander_p2p.prg $<
	dali -o sander_p2p.prg.dali sander_p2p.prg
	ls -l sander*

# default is best: -bpc 0,1,8,2
# bruteforce is bigger: -bpc 0,11,1,2
# slowtsc best: -bpc 0,1,14,2
ste: $(STE) $(TARGET)
	spot13 $< -o ste_spot.kla
	dali -o ste_spot.kla.dali ste_spot.kla
	./$(TARGET) -v -bf -o ste_p2p.prg $<
	#./$(TARGET) -bpc 0,1,8,2 -o ste_p2p.prg $<
	#./$(TARGET) -bpc 0,11,1,2 -o ste_p2p.prg $<
	dali -o ste_p2p.prg.dali ste_p2p.prg
	ls -l ste*

#bf best: -bpc 15,12,3,5
#default: -bpc 15,5,12,3
#15,3,10,5
mermaid2: $(MERMAID2) $(TARGET)
	spot13 $< -o mermaid2_spot.kla
	dali -o mermaid2_spot.kla.dali mermaid2_spot.kla
	./$(TARGET) -v -bf -o mermaid2_p2p.prg $<
	#./$(TARGET) -bpc 15,12,3,5 -o mermaid2_p2p.prg $<
	#./$(TARGET) -bpc 15,3,12,5 -o mermaid2_p2p.prg $<
	#./$(TARGET) -bpc 15,3,10,5 -o mermaid2_p2p.prg $<
	dali -o mermaid2_p2p.prg.dali mermaid2_p2p.prg
	ls -l mermaid2*

sulevi: $(SULEVI) $(TARGET)
	spot13 $< -o sulevi_spot.kla
	dali -o sulevi_spot.kla.dali sulevi_spot.kla
	./$(TARGET) -v -bf -o sulevi_p2p.prg $<
	#./$(TARGET) -bpc 3,10,6,1 -o sulevi_p2p.prg $<
	#./$(TARGET) -bpc 3,10,1,6 -o sulevi_p2p.prg $<
	#./$(TARGET) -bpc 3,10,0,1 -o sulevi_p2p.prg $<
	dali -o sulevi_p2p.prg.dali sulevi_p2p.prg
	ls -l sulevi*

# best bf: 0,15,12,11
# default: 0,11,12,15
ROBIN=testdata/robinlevy_deadlock.png
# best bf: 0,3,5,9
# default: 0,9,5,14
# ROBIN=testdata/robinlevy_huntersmoon.png
robinlevy: $(ROBIN) $(TARGET)
	spot13 $< -o robin_spot.kla
	dali -o robin_spot.kla.dali robin_spot.kla
	./$(TARGET) -v -bf -o robin_p2p.prg $<
	#./$(TARGET) -bpc 0,15,12,11 -o robin_p2p.prg $<
	#./$(TARGET) -bpc 0,11,12,15 -o robin_p2p.prg $<
	dali -o robin_p2p.prg.dali robin_p2p.prg
	ls -l robin*

# best: -bpc 9,0,6,5 (also bf, but non deterministic)
# best bf: 9,15,10,0
# default: 9,0,12,15
VETO=testdata/veto_room_with_view.png
veto: $(VETO) $(TARGET)
	spot13 $< -o veto_spot.kla
	dali -o veto_spot.kla.dali veto_spot.kla
	./$(TARGET) -v -bf -o veto_p2p.prg $<
	#./$(TARGET) -bpc 9,0,6,5 -o veto_p2p.prg $<
	#./$(TARGET) -bpc 9,15,10,0 -o veto_p2p.prg $<
	#./$(TARGET) -bpc 9,0,12,15 -o veto_p2p.prg $<
	dali -o veto_p2p.prg.dali veto_p2p.prg
	ls -l veto*

MISCPIC=testdata/carrion_still_waiting.png
MISCPIC=testdata/the_sarge_obscena_vaselina_palette1.png
MISCPIC=testdata/archmage_mc_god.png
MISCPIC=testdata/facet_turning_point_320x200.png
MISCPIC=testdata/bizzmo_wool_colodore.png
MISCPIC=testdata/joe_earth.png
MISCPIC=testdata/tentacles.png
#MISCPIC=testdata/the_sarge_therapy.png
miscpic: $(MISCPIC) $(TARGET)
	spot13 $< -o misc_spot.kla
	dali -o misc_spot.kla.dali misc_spot.kla
	./$(TARGET) -v -bf -o misc_p2p.prg $<
	#./$(TARGET) -bpc 10,0,13,8 -o misc_p2p.prg $<
	#./$(TARGET) -bpc 10,0,6,3 -o misc_p2p.prg $<
	#./$(TARGET) -bpc 0,13,11,6 -o misc_p2p.prg $<
	dali -o misc_p2p.prg.dali misc_p2p.prg
	ls -l misc*

clean:
	rm -f $(ALLTARGETS) $(TARGET) q*.prg display*.prg *.exo *.dali *.upx *.sym *_p2p.prg *_spot.kla
	rm -rf dist
