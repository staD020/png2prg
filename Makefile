SRC=main.go png2prg.go palettes.go animation.go analyze.go convert.go doc.go GEN_display.go
DISPLAYERS=display_koala.prg display_hires.prg display_mc_charset.prg display_sc_charset.prg
ASMLIB=lib.asm
ASM=java -jar ./vendor/KickAss-5.19.jar
ASMFLAGS=-showmem -time
X64=x64sc
UPX=upx
UPXFLAGS=--best

LDFLAGS=-s -w
CGO=0
GOBUILDFLAGS=-v
TARGET=png2prg_linux_amd64

FLAGS=-d -v
FLAGSNG=-d -v -no-guess
FLAGSNG2=-d -v -bitpair-colors 0,-1,-1,-1
FLAGSFORCE=-d -v -bitpair-colors 0,11,12,15
TESTPIC=testdata/ben_daglish.png

png2prg: $(TARGET)

all: $(TARGET) png2prg_darwin_amd64 png2prg_win_amd64.exe

compress: $(TARGET).upx png2prg_darwin_amd64.upx png2prg_win_amd64.exe.upx

GEN_display.go: generate.go $(DISPLAYERS)
	go generate

%.prg: %.asm $(ASMLIB)
	$(ASM) $(ASMFLAGS) $< -o $@

%.upx: %
	$(UPX) $(UPXFLAGS) -o $@ $<
	touch $@

$(TARGET): $(SRC)
	CGO_ENABLED=$(CGO) GOOS=linux GOARCH=amd64 go build $(GOBUILDFLAGS) -ldflags="$(LDFLAGS)" -o $@ $^

png2prg_darwin_amd64: $(SRC)
	CGO_ENABLED=$(CGO) GOOS=darwin GOARCH=amd64 go build $(GOBUILDFLAGS) -ldflags="$(LDFLAGS)" -o $@ $^

png2prg_win_amd64.exe: $(SRC)
	CGO_ENABLED=$(CGO) GOOS=windows GOARCH=amd64 go build $(GOBUILDFLAGS) -ldflags="$(LDFLAGS)" -o $@ $^

test: $(TARGET)
	./$(TARGET) $(FLAGS) -o z.prg $(TESTPIC)
	$(X64) z.prg >/dev/null

testpack: $(TARGET)
	./$(TARGET) $(FLAGS) -o z.prg $(TESTPIC)
	exomizer sfx basic -q -o zz_guess.prg z.prg
	./$(TARGET) $(FLAGSNG) -o z.prg $(TESTPIC)
	exomizer sfx basic -q -o zz_noguess.prg z.prg
	./$(TARGET) $(FLAGSNG2) -o z.prg $(TESTPIC)
	exomizer sfx basic -q -o zz_noguess2.prg z.prg
	./$(TARGET) $(FLAGSFORCE) -o z.prg $(TESTPIC)
	exomizer sfx basic -q -o zz_force_manual_colors.prg z.prg
	$(X64) zz_guess.prg >/dev/null

clean:
	rm -f $(TARGET) png2prg_darwin_amd64 png2prg_win_amd64.exe GEN_*.go *.prg *.upx
