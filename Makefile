SRC=main.go png2prg.go palettes.go animation.go analyze.go convert.go doc.go
DISPLAYERS=display_koala.prg display_hires.prg display_mc_charset.prg display_sc_charset.prg display_mc_sprites.prg display_sc_sprites.prg
ASMLIB=lib.asm
ASM=java -jar ./tools/KickAss-5.24.jar
ASMFLAGS=-showmem -time
X64=x64sc
UPX=upx
UPXFLAGS=--best

LDFLAGS=-s -w
CGO=0
GOBUILDFLAGS=-v -trimpath
TARGET=png2prg_linux_amd64
ALLTARGETS=$(TARGET) png2prg_darwin_amd64 png2prg_darwin_arm64 png2prg_win_amd64.exe

FLAGS=-d -v
FLAGSNG=-d -v -no-guess
FLAGSNG2=-d -v -bitpair-colors 0,-1,-1,-1
FLAGSFORCE=-d -v -bitpair-colors 0,11,12,15
#TESTPIC=testdata/ste_ghosts_goblins.gif
#TESTPIC=testdata/ilesj_orbital_impaler.png
#TESTPIC=testdata/deev_desolate_hires.png
#TESTPIC=testdata/the_sarge_steady_eddie_ready_hires.png
#TESTPIC=testdata/carrion_still_waiting.png
#TESTPIC=testdata/bizzmo_wool.gif
TESTPIC=testdata/mirage_parrot.png
#TESTPIC=testdata/sander_ld.png
#TESTPIC=testdata/sander_sander.png
#TESTSID=testdata/Rivalry_tune_5.sid
#TESTSID=testdata/jasonpage_eighth_90.sid
TESTSID=testdata/Nightbreed_-_Dalezy_TRIAD.sid
#TESTSID=testdata/lman_hellyeah.sid
#TESTSID=testdata/Lift_Off_V2.sid

png2prg: $(TARGET)

all: $(ALLTARGETS)

install: $(TARGET)
	sudo cp $(TARGET) /usr/local/bin/png2prg

compress: $(TARGET).upx png2prg_darwin_amd64.upx png2prg_darwin_arm64.upx png2prg_win_amd64.exe.upx

%.prg: %.asm $(ASMLIB)
	$(ASM) $(ASMFLAGS) $< -o $@

%.upx: %
	$(UPX) $(UPXFLAGS) -o $@ $<
	touch $@

$(TARGET): $(SRC) $(DISPLAYERS) $(TESTPIC)
	CGO_ENABLED=$(CGO) GOOS=linux GOARCH=amd64 go build $(GOBUILDFLAGS) -ldflags="$(LDFLAGS)" -o $@

png2prg_darwin_amd64: $(SRC) $(DISPLAYERS)
	CGO_ENABLED=$(CGO) GOOS=darwin GOARCH=amd64 go build $(GOBUILDFLAGS) -ldflags="$(LDFLAGS)" -o $@

png2prg_darwin_arm64: $(SRC) $(DISPLAYERS)
	CGO_ENABLED=$(CGO) GOOS=darwin GOARCH=arm64 go build $(GOBUILDFLAGS) -ldflags="$(LDFLAGS)" -o $@

png2prg_win_amd64.exe: $(SRC) $(DISPLAYERS)
	CGO_ENABLED=$(CGO) GOOS=windows GOARCH=amd64 go build $(GOBUILDFLAGS) -ldflags="$(LDFLAGS)" -o $@

test: $(TARGET) $(TESTPIC)
	./$(TARGET) $(FLAGS) -sid $(TESTSID) -o z.prg $(TESTPIC)
	$(X64) z.prg >/dev/null

testpack: $(TARGET)
	./$(TARGET) $(FLAGS) -np -o z.prg $(TESTPIC)
	exomizer sfx basic -q -o zz_guess.sfx.exo z.prg
	dali --sfx 2079 -o zz_guess.sfx.dali z.prg
	./$(TARGET) $(FLAGSNG) -np -o z.prg $(TESTPIC)
	exomizer sfx basic -q -o zz_noguess.sfx.exo z.prg
	dali --sfx 2079 -o zz_noguess.sfx.dali z.prg
	./$(TARGET) $(FLAGSNG2) -np -o z.prg $(TESTPIC)
	exomizer sfx basic -q -o zz_noguess2.sfx.exo z.prg
	dali --sfx 2079 -o zz_noguess2.sfx.dali z.prg
	./$(TARGET) $(FLAGSFORCE) -np -o z.prg $(TESTPIC)
	exomizer sfx basic -q -o zz_force_manual_colors.sfx.exo z.prg
	dali --sfx 2079 -o zz_force_manual_colors.sfx.dali z.prg
	$(X64) zz_guess.sfx.exo >/dev/null

clean:
	rm -f $(TARGET) png2prg_darwin_amd64 png2prg_darwin_arm64 png2prg_win_amd64.exe GEN_*.go *.prg *.exo *.dali *.upx
