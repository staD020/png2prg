SRC=*.go cmd/png2prg/*.go tools/rom_charset_lowercase.prg tools/rom_charset_uppercase.prg
DISPLAYERS=display_koala.prg display_koala_anim.prg display_hires.prg display_hires_anim.prg display_mc_charset.prg display_sc_charset.prg display_mc_sprites.prg display_sc_sprites.prg display_koala_anim_alternative.prg display_mci_bitmap.prg display_mixed_charset.prg display_petscii_charset.prg
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
TESTSID=testdata/Rivalry_tune_5.sid
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
	cp -r testdata/mixedcharset/*.png dist/testdata/mixedcharset/
	cp -r testdata/hirescharset/*.png dist/testdata/hirescharset/
	./$(TARGET) -d -q -o dist/0.neo.prg testdata/mixedcharset/hein_neo.png
	./$(TARGET) -d -q -o dist/1.wrath.prg testdata/mixedcharset/joe_wrath.png
	./$(TARGET) -d -q -o dist/2.huntress.prg testdata/mixedcharset/huntress.gif
	./$(TARGET) -d -q -o dist/3.ohno.prg testdata/hirescharset/ohno_logo.png
	./$(TARGET) -d -q -o dist/4.gestalt.prg testdata/hirescharset/gestalt.png
	./$(TARGET) -d -q -o dist/5.samar.prg testdata/hirescharset/jetan_samar.png
	./$(TARGET) -d -q -o dist/6.apace.prg testdata/mixedcharset/zscs_apace.png
	./$(TARGET) -d -q -o dist/7.ocd.prg testdata/mixedcharset/ocd.png
	./$(TARGET) -d -q -o dist/8.hyper.prg testdata/mixedcharset/hein_hyper.png
	./$(TARGET) -d -q -o dist/9.extend.prg testdata/hirescharset/extend.png
	./$(TARGET) -d -q -o dist/10.mega.prg testdata/mixedcharset/sarge_mega.png
	./$(TARGET) -d -q -o dist/11.fair.prg testdata/mixedcharset/soya_fair.png
	./$(TARGET) -d -q -o dist/12.shine.prg -bpc 3 testdata/mixedcharset/shine.png
	./$(TARGET) -d -q -o dist/13.hibiscus.prg testdata/petscii/hein_hibiscus.png
	./$(TARGET) -d -q -o dist/14.submarine.prg testdata/petscii/submarine.png
	./$(TARGET) -d -q -o dist/15.triad.prg testdata/petscii/triad.png
	./$(TARGET) -d -q -o dist/16.proxima.prg testdata/petscii/proxima.png
	./$(TARGET) -d -q -o dist/17.artline.prg testdata/petscii/artline.png
	rm -f dist/examples.d64
	d64 -add dist/examples.d64 dist/?.*.prg dist/1?.*.prg
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

clean:
	rm -f $(ALLTARGETS) $(TARGET) q*.prg display*.prg *.exo *.dali *.upx *.sym
	rm -rf dist
