SRC=*.go cmd/png2prg/*.go
DISPLAYERS=display_koala.prg display_koala_anim.prg display_hires.prg display_hires_anim.prg display_mc_charset.prg display_sc_charset.prg display_mc_sprites.prg display_sc_sprites.prg display_koala_anim_alternative.prg display_mci_bitmap.prg
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
TESTSIDMAD=testdata/madonna/holiday.sid
TESTANIM=testdata/jamesband*.png
TESTSIDANIM=testdata/Nightbreed_-_Dalezy_TRIAD.sid

png2prg: $(SRC) $(DISPLAYERS)
	CGO_ENABLED=$(CGO) go build $(GOBUILDFLAGS) -ldflags="$(LDFLAGS)" -o $@ ./cmd/png2prg

all: $(ALLTARGETS)

bench: $(DISPLAYERS)
	go test -bench Benchmark. -benchmem ./...

dist: $(ALLTARGETS) $(TARGET) readme $(TESTSID) $(TESTSID) $(TESTSIDANIM)
	mkdir -p dist/testdata
	cp readme.md dist/
	cp $(ALLTARGETS) dist/
	cp testdata/jamesband*.png dist/testdata/
	cp $(TESTPIC) dist/testdata/
	cp $(TESTSID) dist/testdata/
	cp $(TESTSID) $(TESTSIDANIM) dist/testdata/
	cp testdata/Dutch_Breeze_Soft_and_Wet.sid dist/testdata/
	cp -r testdata/evoluer dist/testdata/
	cp -r testdata/mcinterlace dist/testdata/
	cp -r testdata/drazlace dist/testdata/
	cp -r testdata/madonna dist/testdata/
	./$(TARGET) -d -q -o dist/0.madonna.prg -sid $(TESTSIDMAD) testdata/madonna/cjam_pure_madonna.png
	./$(TARGET) -d -q -o dist/5.jamesband.prg -sid $(TESTSIDANIM) $(FLAGSANIM) testdata/jamesband*.png
	./$(TARGET) -d -q -o dist/7.parrot.prg -sid $(TESTSID) testdata/mirage_parrot.png
	./$(TARGET) -d -q -o dist/6.evoluer.prg -sid testdata/evoluer/Evoluer.sid testdata/evoluer/PIC??.png
	./$(TARGET) -d -q -i -o dist/1.stoned.prg -sid $(TESTSID) testdata/drazlace/amn_stoned_frame*.png
	./$(TARGET) -d -q -i -o dist/2.zootrope.prg -sid testdata/Dutch_Breeze_Soft_and_Wet.sid testdata/drazlace/clone_zootrope.png
	./$(TARGET) -d -q -i -o dist/3.parriot.prg -sid $(TESTSID) testdata/mcinterlace/parriot*.png
	./$(TARGET) -d -q -i -o dist/4.tete.prg -sid $(TESTSID) testdata/mcinterlace/tete*.png
	rm -f dist/examples.d64
	d64 -add dist/examples.d64 dist/*.prg
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
	./$(TARGET) -h >readme.md 2>&1

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
