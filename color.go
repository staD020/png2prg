package png2prg

import (
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// A C64Color is one of the 16 C64 colors, 0 is black, 1 white, 2 red, ...
type C64Color byte

func (c C64Color) String() string {
	switch c & 0x0f {
	case 0:
		return "black"
	case 1:
		return "white"
	case 2:
		return "red"
	case 3:
		return "cyan"
	case 4:
		return "purple"
	case 5:
		return "green"
	case 6:
		return "blue"
	case 7:
		return "yellow"
	case 8:
		return "orange"
	case 9:
		return "brown"
	case 10:
		return "lightred"
	case 11:
		return "darkgrey"
	case 12:
		return "grey"
	case 13:
		return "lightgreen"
	case 14:
		return "lightblue"
	case 15:
		return "lightgrey"
	default:
		return "unknown color"
	}
}

// A Color contains a mapped C64Color and its embedded color.Color value, usually a color.RGBA.
type Color struct {
	color.Color
	C64Color C64Color
	BitPair  byte
}

func (c Color) String() string {
	return fmt.Sprintf("%d,%s", c.C64Color, rgbaString(c))
}

// NewColor returns a new C64Color/color.Color pair.
func NewColor(c64color C64Color, col color.Color) Color {
	return Color{C64Color: c64color, Color: col}
}

// rgbaString returns a human readable #rrggbb string of col.
func rgbaString(col color.Color) string {
	r, g, b, _ := col.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", byte(r), byte(g), byte(b))
}

// Distance returns the absolute rgb distance between c and col.
func (c Color) Distance(col color.Color) int {
	r1, g1, b1, _ := col.RGBA()
	r2, g2, b2, _ := c.RGBA()
	d := int(math.Abs(float64(r2&0xff)-float64(r1&0xff))) +
		int(math.Abs(float64(g2&0xff)-float64(g1&0xff))) +
		int(math.Abs(float64(b2&0xff)-float64(b1&0xff)))
	return d
}

// A Palette contains RGB/C64Color to png2prg.Color maps for quick lookups.
type Palette struct {
	Name    string
	loose   bool
	c642col [MaxColors]*Color
	rgb2col map[string]Color
	mtx     *sync.RWMutex
}

func NewPalette(img image.Image, looseMatching bool) (Palette, error) {
	cols := imageColors(img)
	if len(cols) > MaxColors {
		return Palette{}, fmt.Errorf("too many colors: %d while the max is %d", len(cols), MaxColors)
	}
	p := analyzeColors(cols)
	p.loose = looseMatching
	return p, nil
}

func BlankPalette(name string, looseMatching bool) Palette {
	return Palette{
		Name:    name,
		loose:   looseMatching,
		rgb2col: make(map[string]Color),
		mtx:     &sync.RWMutex{},
	}
}

func (p Palette) NumColors() int {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	return len(p.rgb2col)
}

func (p Palette) Colors() (cc []Color) {
	p.mtx.RLock()
	for _, c := range p.c642col {
		if c != nil {
			cc = append(cc, *c)
		}
	}
	p.mtx.RUnlock()
	return cc
}

// SortColors returns the palette's colors sorted by C64Color.
func (p Palette) SortColors() []Color {
	cc := p.Colors()
	sort.Slice(cc, func(i, j int) bool { return cc[i].C64Color < cc[j].C64Color })
	return cc
}

// Add adds the Color to the Palette. If the Color was already present, it will be updated.
func (p *Palette) Add(col Color) {
	p.mtx.Lock()
	p.c642col[col.C64Color] = &col
	p.rgb2col[rgbaString(col)] = col
	p.mtx.Unlock()
}

// FromC64
func (p Palette) FromC64(col C64Color) (Color, error) {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	if v := p.c642col[col]; v != nil {
		return *p.c642col[col], nil
	}
	return Color{
		Color:    color.RGBA{},
		C64Color: col,
	}, fmt.Errorf("c64color %d not found", col)
}

func (p Palette) FromColor(col color.Color) (Color, error) {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	if v, ok := p.rgb2col[rgbaString(col)]; ok {
		return v, nil
	}
	return Color{Color: col}, fmt.Errorf("rgb %s not found", rgbaString(col))
}

// Convert converts a color to a png2prg.Color and returns it, implementing the color.Model interface.
// Finds closest match if p.loose is true.
func (p Palette) Convert(c color.Color) color.Color {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	if !p.loose {
		col, err := p.FromColor(c)
		if err != nil {
			return Color{}
		}
		return col
	}
	min := int(6e6)
	found := Color{}
	for _, src := range paletteSources {
		for _, col := range src.Colors {
			d := col.Distance(c)
			if d < min {
				found = col
				min = d
			}
		}
	}
	return found
}

// imageColors returns a slice of unique colors present in the image.
func imageColors(img image.Image) (cc []color.Color) {
	m := map[color.Color]struct{}{}
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			col := img.At(x, y)
			if _, ok := m[col]; !ok {
				m[col] = struct{}{}
				cc = append(cc, col)
			}
		}
	}
	return cc
}

// analyzeColors calculates the color distances of all colors and each of the palletSources.
// It returns the closest matching Palette.
func analyzeColors(cc []color.Color) (found Palette) {
	minDistance := int(6e9)
	for _, src := range paletteSources {
		p := Palette{Name: src.Name, rgb2col: map[string]Color{}, mtx: &sync.RWMutex{}}
		totalDistance := 0
		for _, c := range cc {
			distance := int(6e9)
			var foundCol Color
			for _, srcCol := range src.Colors {
				d := srcCol.Distance(c)
				if d < distance {
					distance = d
					foundCol = Color{Color: c, C64Color: srcCol.C64Color}
				}
				if d == 0 {
					break
				}
			}
			p.c642col[foundCol.C64Color] = &foundCol
			p.rgb2col[rgbaString(c)] = foundCol

			totalDistance += distance
		}
		fmt.Printf("palette %q distance = %d\n", p.Name, totalDistance)
		if totalDistance < minDistance {
			found = p
			minDistance = totalDistance
		}
		if minDistance == 0 {
			break
		}
	}
	return found
}

//go:embed "palettes.yaml"
var palettesYaml []byte

type paletteSource struct {
	Name   string
	Colors []Color
}

var paletteSources []paletteSource

func init() {
	var err error
	paletteSources, err = convertPaletteSources(palettesYaml)
	if err != nil {
		panic(fmt.Errorf("sourcePalettes failed: %w", err))
	}
	if len(paletteSources) == 0 {
		panic(fmt.Errorf("no palettes found in %q", palettesYaml))
	}
}

// sourcePalettes parses inputYaml, converts it to []paletteSource and returns it.
func convertPaletteSources(inputYaml []byte) (out []paletteSource, err error) {
	type paletteYaml struct {
		Name   string
		Colors []string
	}
	var ps []paletteYaml
	if err = yaml.Unmarshal(inputYaml, &ps); err != nil {
		return out, err
	}
	for _, p := range ps {
		ps := paletteSource{Name: p.Name, Colors: make([]Color, 16, 16)}
		for _, l := range p.Colors {
			a := strings.Split(l, ",")
			c64col, err := strconv.Atoi(strings.TrimSpace(a[0]))
			if err != nil {
				return out, err
			}
			rgb, err := strconv.ParseUint(strings.TrimSpace(a[1][1:]), 16, 24)
			if err != nil {
				return out, err
			}
			ps.Colors[c64col] = NewColor(C64Color(c64col), color.RGBA{byte((rgb >> 16) & 0xff), byte((rgb >> 8) & 0xff), byte(rgb & 0xff), 0x01})
		}
		out = append(out, ps)
	}
	return out, nil
}
