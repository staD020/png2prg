SRC=png2prg.go palettes.go animation.go analyze.go convert.go doc.go GEN_display.go
DISPLAYERS=display_koala.prg display_hires.prg display_mc_charset.prg display_sc_charset.prg
ASMLIB=lib.asm
ASM=java -jar ./vendor/KickAss-5.16.jar
ASMFLAGS=-showmem -time
X64=x64sc
UPX=upx
UPXFLAGS=--best

LDFLAGS=-s -w
CGO=0
GOBUILDFLAGS=-v

FLAGS=-d -v
FLAGSNG=-d -v -no-guess
FLAGSNG2=-d -v -bitpair-colors 0,-1,-1,-1
FLAGSNG3=-d -v -bitpair-colors 0,15,12,1
TESTPIC=testdata/ste_ikariwarriors.gif

png2prg: png2prg_linux

all: png2prg_linux png2prg_darwin png2prg.exe

test: png2prg_linux
	./png2prg_linux $(FLAGS) -o z.prg $(TESTPIC)
	$(X64) z.prg >/dev/null

testpack: png2prg_linux
	./png2prg_linux $(FLAGS) -o z.prg $(TESTPIC)
	exomizer sfx basic -q -o zz_guess.prg z.prg
	./png2prg_linux $(FLAGSNG) -o z.prg $(TESTPIC)
	exomizer sfx basic -q -o zz_noguess.prg z.prg
	./png2prg_linux $(FLAGSNG2) -o z.prg $(TESTPIC)
	exomizer sfx basic -q -o zz_noguess2.prg z.prg
	./png2prg_linux $(FLAGSNG3) -o z.prg $(TESTPIC)
	exomizer sfx basic -q -o zz_force_manual_colors.prg z.prg
	$(X64) zz_guess.prg >/dev/null

compress: png2prg_linux.upx png2prg_darwin.upx png2prg.exe.upx

GEN_display.go: generate.go $(DISPLAYERS)
	go generate

%.prg: %.asm $(ASMLIB)
	$(ASM) $(ASMFLAGS) $< -o $@

%.upx: %
	$(UPX) $(UPXFLAGS) -o $@ $<
	touch $@

png2prg_linux: $(SRC)
	CGO_ENABLED=$(CGO) GOOS=linux GOARCH=amd64 go build $(GOBUILDFLAGS) -ldflags="$(LDFLAGS)" -o $@ $^

png2prg_darwin: $(SRC)
	CGO_ENABLED=$(CGO) GOOS=darwin GOARCH=amd64 go build $(GOBUILDFLAGS) -ldflags="$(LDFLAGS)" -o $@ $^

png2prg.exe: $(SRC)
	CGO_ENABLED=$(CGO) GOOS=windows GOARCH=amd64 go build $(GOBUILDFLAGS) -ldflags="$(LDFLAGS)" -o $@ $^

clean:
	rm -f png2prg_linux png2prg_darwin png2prg.exe GEN_*.go *.prg *.upx
