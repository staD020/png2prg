SRC=png2prg.go palettes.go animation.go analyze.go convert.go doc.go GEN_display.go
DISPLAYERS=display_koala.prg display_hires.prg display_mc_charset.prg display_sc_charset.prg
ASM=java -jar ./vendor/KickAss-5.16.jar
ASMFLAGS=-showmem -time

LDFLAGS=-s -w
CGO=0
GOBUILDFLAGS=-v

#UPX := $(shell command -v upx 2>/dev/null)
#UPXFLAGS=--best

all: png2prg_linux png2prg_darwin png2prg.exe

png2prg: png2prg_linux

GEN_display.go: generate.go $(DISPLAYERS)
	go generate

%.prg: %.asm
	$(ASM) $(ASMFLAGS) $< -o $@

png2prg_linux: $(SRC)
ifdef UPX
	@rm -f $@ $@.raw
	CGO_ENABLED=$(CGO) GOOS=linux GOARCH=amd64 go build $(GOBUILDFLAGS) -ldflags="$(LDFLAGS)" -o $@.raw $^
	upx $(UPXFLAGS) -o $@ $@.raw
	@rm $@.raw
else
	CGO_ENABLED=$(CGO) GOOS=linux GOARCH=amd64 go build $(GOBUILDFLAGS) -ldflags="$(LDFLAGS)" -o $@ $^
endif

png2prg_darwin: $(SRC)
ifdef UPX
	@rm -f $@ $@.raw
	CGO_ENABLED=$(CGO) GOOS=darwin GOARCH=amd64 go build $(GOBUILDFLAGS) -ldflags="$(LDFLAGS)" -o $@.raw $^
	upx $(UPXFLAGS) -o $@ $@.raw
	@rm $@.raw
else
	CGO_ENABLED=$(CGO) GOOS=darwin GOARCH=amd64 go build $(GOBUILDFLAGS) -ldflags="$(LDFLAGS)" -o $@ $^
endif

png2prg.exe: $(SRC)
ifdef UPX
	@rm -f $@ $@.raw
	CGO_ENABLED=$(CGO) GOOS=windows GOARCH=amd64 go build $(GOBUILDFLAGS) -ldflags="$(LDFLAGS)" -o $@.raw $^
	upx $(UPXFLAGS) -o $@ $@.raw
	@rm $@.raw
else
	CGO_ENABLED=$(CGO) GOOS=windows GOARCH=amd64 go build $(GOBUILDFLAGS) -ldflags="$(LDFLAGS)" -o $@ $^
endif

clean:
	rm -f png2prg_linux png2prg_darwin png2prg.exe GEN_*.go *.prg
