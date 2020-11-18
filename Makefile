SRC=png2prg.go palettes.go animation.go analyze.go convert.go doc.go GEN_display.go
DISPLAYERS=display_koala.prg display_hires.prg display_mc_charset.prg display_sc_charset.prg
ASM=java -jar ./vendor/KickAss-5.16.jar
ASMFLAGS=-showmem -time
X64=x64

LDFLAGS=-s -w
CGO=0
GOBUILDFLAGS=-v

UPX := $(shell command -v upx 2>/dev/null)
UPXFLAGS=--best

test: png2prg_linux
	./png2prg_linux -d -o z.prg testdata/wool.gif
	$(X64) z.prg >/dev/null

png2prg: png2prg_linux

all: png2prg_linux png2prg_darwin png2prg.exe

compress: png2prg_linux.upx png2prg_darwin.upx png2prg.exe.upx

GEN_display.go: generate.go $(DISPLAYERS)
	go generate

%.prg: %.asm
	$(ASM) $(ASMFLAGS) $< -o $@

%.upx: %
	upx $(UPXFLAGS) -o $@ $<

png2prg_linux: $(SRC)
	CGO_ENABLED=$(CGO) GOOS=linux GOARCH=amd64 go build $(GOBUILDFLAGS) -ldflags="$(LDFLAGS)" -o $@ $^

png2prg_darwin: $(SRC)
	CGO_ENABLED=$(CGO) GOOS=darwin GOARCH=amd64 go build $(GOBUILDFLAGS) -ldflags="$(LDFLAGS)" -o $@ $^

png2prg.exe: $(SRC)
	CGO_ENABLED=$(CGO) GOOS=windows GOARCH=amd64 go build $(GOBUILDFLAGS) -ldflags="$(LDFLAGS)" -o $@ $^

clean:
	rm -f png2prg_linux png2prg_darwin png2prg.exe GEN_*.go *.prg
