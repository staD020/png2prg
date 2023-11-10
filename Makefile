SRC=*.go cmd/png2prg/*.go
DISPLAYERS=display_koala.prg display_koala_anim.prg display_hires.prg display_hires_anim.prg display_mc_charset.prg display_sc_charset.prg display_mc_sprites.prg display_sc_sprites.prg display_koala_anim_alternative.prg display_mci_bitmap.prg
ASMLIB=lib.asm
ASM=java -jar ./tools/KickAss-5.24.jar
ASMFLAGS=-showmem -time
X64=x64sc
UPX=upx
UPXFLAGS=

LDFLAGS=-s -w
CGO=1
GOBUILDFLAGS=-v -trimpath
TARGET=png2prg_linux_amd64
ALLTARGETS=$(TARGET) png2prg_darwin_amd64 png2prg_darwin_arm64 png2prg_win_amd64.exe png2prg_win_x86.exe png2prg_linux_arm64

FLAGS=-d -v -i
FLAGSANIM=-d -v -frame-delay 8
FLAGSNG=-d -v -no-guess
FLAGSNG2=-d -v -bitpair-colors 0,-1,-1,-1
FLAGSFORCE=-d -v -bitpair-colors 0,1,8,2
TESTPIC=testdata/madonna/cjam_pure_madonna.png
#TESTPIC=testdata/ste_ghosts_goblins.gif
#TESTPIC=testdata/ilesj_orbital_impaler.png
#TESTPIC=testdata/deev_desolate_hires.png
#TESTPIC=testdata/the_sarge_steady_eddie_ready_hires.png
#TESTPIC=testdata/carrion_still_waiting.png
#TESTPIC=testdata/bizzmo_wool.gif
#TESTPIC=testdata/mirage_parrot.png
#TESTPIC=testdata/sander_ld.png
#TESTPIC=testdata/sander_sander.png
#TESTSID=testdata/Rivalry_tune_5.sid
#TESTSID=testdata/jasonpage_eighth_90.sid
#TESTSID=testdata/Nightbreed_-_Dalezy_TRIAD.sid
#TESTSID=testdata/Yie_Ar_Kung_Fu_60.sid
#TESTSID=testdata/lman_hellyeah.sid
#TESTSID=testdata/Lift_Off_V2.sid
#TESTSID=testdata/Laserdance_10.sid
#TESTSID=testdata/Commando.sid
#TESTSID=testdata/Commando_Take_Me_to_the_Bridge_Mix.sid
TESTSID=testdata/madonna/Papa_Dont_Preach.sid
#TESTANIM=testdata/sander_tankframes.gif
#TESTANIM=testdata/jamesband02.png testdata/jamesband02.png testdata/jamesband02.png testdata/jamesband02.png testdata/jamesband03.png testdata/jamesband03.png testdata/jamesband03.png testdata/jamesband03.png testdata/jamesband03.png testdata/jamesband??.png
#TESTANIM=testdata/jamesband01.png testdata/jamesband03.png testdata/jamesband01.png testdata/jamesband03.png testdata/jamesband01.png testdata/jamesband01.png testdata/jamesband01.png testdata/jamesband*.png
TESTANIM=testdata/jamesband*.png

png2prg: $(TARGET)

all: $(ALLTARGETS)

bench: $(DISPLAYERS)
	go test -bench Benchmark. -benchmem ./...

install: $(TARGET)
	sudo cp $(TARGET) /usr/local/bin/png2prg

compress: $(TARGET).upx png2prg_darwin_amd64.upx png2prg_darwin_arm64.upx png2prg_win_amd64.exe.upx png2prg_win_x86.exe.upx

%.prg: %.asm $(ASMLIB)
	$(ASM) $(ASMFLAGS) $< -o $@

%.upx: %
	$(UPX) $(UPXFLAGS) -o $@ $<
	touch $@

$(TARGET): $(SRC) $(DISPLAYERS)
	CGO_ENABLED=$(CGO) GOOS=linux GOARCH=amd64 go build $(GOBUILDFLAGS) -ldflags="$(LDFLAGS)" -o $@ ./cmd/png2prg/

png2prg_linux_arm64: $(SRC) $(DISPLAYERS)
	CGO_ENABLED=$(CGO) GOOS=linux GOARCH=arm64 go build $(GOBUILDFLAGS) -ldflags="$(LDFLAGS)" -o $@ ./cmd/png2prg/

png2prg_darwin_amd64: $(SRC) $(DISPLAYERS)
	CGO_ENABLED=$(CGO) GOOS=darwin GOARCH=amd64 go build $(GOBUILDFLAGS) -ldflags="$(LDFLAGS)" -o $@ ./cmd/png2prg/

png2prg_darwin_arm64: $(SRC) $(DISPLAYERS)
	CGO_ENABLED=$(CGO) GOOS=darwin GOARCH=arm64 go build $(GOBUILDFLAGS) -ldflags="$(LDFLAGS)" -o $@ ./cmd/png2prg/

png2prg_win_amd64.exe: $(SRC) $(DISPLAYERS)
	CGO_ENABLED=$(CGO) GOOS=windows GOARCH=amd64 go build $(GOBUILDFLAGS) -ldflags="$(LDFLAGS)" -o $@ ./cmd/png2prg/

png2prg_win_x86.exe: $(SRC) $(DISPLAYERS)
	CGO_ENABLED=$(CGO) GOOS=windows GOARCH=386 go build $(GOBUILDFLAGS) -ldflags="$(LDFLAGS)" -o $@ ./cmd/png2prg/

test: $(TARGET) $(TESTPIC) $(TESTSID)
	./$(TARGET) $(FLAGS) -sid $(TESTSID) -o z.prg $(TESTPIC)
	$(X64) z.prg >/dev/null

testanim: $(TARGET) $(TESTANIM) $(TESTSID)
	./$(TARGET) $(FLAGSANIM) -sid $(TESTSID) -o z.prg $(TESTANIM)
	$(X64) z.prg >/dev/null

evoluer: $(TARGET)
	./$(TARGET) -d -frame-delay 4 -o z.prg -sid testdata/evoluer/Evoluer.sid testdata/evoluer/PIC??.png
	$(X64) z.prg >/dev/null

testpack: $(TARGET)
	./$(TARGET) $(FLAGS) -nc -np -o z.prg $(TESTPIC)
	exomizer sfx basic -q -o zz_guess.sfx.exo z.prg
	dali --sfx 2082 -o zz_guess.sfx.dali z.prg
	./$(TARGET) $(FLAGSNG) -nc -np -o z.prg $(TESTPIC)
	exomizer sfx basic -q -o zz_noguess.sfx.exo z.prg
	dali --sfx 2082 -o zz_noguess.sfx.dali z.prg
	./$(TARGET) $(FLAGSNG2) -nc -np -o z.prg $(TESTPIC)
	exomizer sfx basic -q -o zz_noguess2.sfx.exo z.prg
	dali --sfx 2082 -o zz_noguess2.sfx.dali z.prg
	./$(TARGET) $(FLAGSFORCE) -nc -np -o z.prg $(TESTPIC)
	exomizer sfx basic -q -o zz_force_manual_colors.sfx.exo z.prg
	dali --sfx 2082 -o zz_force_manual_colors.sfx.dali z.prg
	./$(TARGET) $(FLAGS) -o z.prg $(TESTPIC)
	$(X64) zz_guess.sfx.exo >/dev/null

clean:
	rm -f $(ALLTARGETS) GEN_*.go *.prg *.exo *.dali *.upx *.sym
