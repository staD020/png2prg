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
}

func (c Color) String() string {
	return fmt.Sprintf("%d,%s", c.C64Color, c.RGBString())
}

func (c Color) RGBString() string {
	return rgbString(c)
}

type colorKey [3]byte

func ColorKey(c color.Color) colorKey {
	r, g, b, _ := c.RGBA()
	return colorKey{byte(r & 0xff), byte(g & 0xff), byte(b & 0xff)}
}

// NewColor returns a new C64Color/color.Color pair.
func NewColor(c64color C64Color, col color.Color) Color {
	return Color{C64Color: c64color, Color: col}
}

// rgbString returns a human readable #rrggbb string of col.
func rgbString(col color.Color) string {
	r, g, b, _ := col.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", byte(r&0xff), byte(g&0xff), byte(b&0xff))
}

// Distance returns the absolute rgb distance between c and col.
func (c Color) Distance(col color.Color) int {
	r1, g1, b1, _ := col.RGBA()
	r2, g2, b2, _ := c.RGBA()
	/*
		// todo: https://stackoverflow.com/questions/9018016/how-to-compare-two-colors-for-similarity-difference
		rmean := float64(r1&0xff) + float64(r2&0xff)/float64(2)
		r, g, b := float64(r1&0xff-r2&0xff), float64(g1&0xff-g2&0xff), float64(b1&0xff-b2&0xff)
		result := math.Sqrt((((512 + rmean) * r * r) * 256) + 4*g*g + (((767 - rmean) * b * b) * 256))
		//fmt.Printf("Distance %q %q : %.1f\n", rgbaString(c), rgbaString(col), result)
		return int(math.Floor(result))
	*/
	d := int(math.Abs(float64(r2&0xff)-float64(r1&0xff))) +
		int(math.Abs(float64(g2&0xff)-float64(g1&0xff))) +
		int(math.Abs(float64(b2&0xff)-float64(b1&0xff)))
	return d
}

// A Palette contains RGB/C64Color to png2prg.Color maps for quick lookups.
type Palette struct {
	Name    string
	loose   bool
	c642col map[C64Color]Color
	rgb2col map[colorKey]Color
}

func NewPalette(img image.Image, looseMatching, verbose bool) (p Palette, hires bool, err error) {
	cols, hires := imageColors(img)
	if len(cols) > MaxColors {
		return Palette{}, hires, fmt.Errorf("too many colors: %d while the max is %d", len(cols), MaxColors)
	}
	p = analyzeColors(cols, verbose)
	p.loose = looseMatching
	return p, hires, nil
}

func BlankPalette(name string, looseMatching bool) Palette {
	return Palette{
		Name:    name,
		loose:   looseMatching,
		c642col: make(map[C64Color]Color),
		rgb2col: make(map[colorKey]Color),
	}
}

func (p Palette) String() string {
	s := ""
	for i := C64Color(0); i < MaxColors; i++ {
		if col, ok := p.c642col[i]; ok {
			s += col.String() + " "
		}
	}
	return p.Name + ": " + s
}

func (p Palette) NumColors() int {
	return len(p.rgb2col)
}

// Colors returns the Palette's colors, the order is undefined.
func (p Palette) Colors() (cc []Color) {
	for _, c := range p.c642col {
		cc = append(cc, c)
	}
	return cc
}

// SortColors returns the Palette's colors sorted by C64Color.
func (p Palette) SortColors() []Color {
	cc := p.Colors()
	sort.Slice(cc, func(i, j int) bool { return cc[i].C64Color < cc[j].C64Color })
	return cc
}

// Add adds the Color to the Palette. If the Color was already present, it will be updated.
func (p *Palette) Add(colors ...Color) {
	for _, col := range colors {
		p.c642col[col.C64Color] = col
		p.rgb2col[ColorKey(col)] = col
	}
}

// Delete deletes the Color from the Palette. If the Color was not present, nothing happens.
func (p *Palette) Delete(colors ...Color) {
	for _, col := range colors {
		delete(p.c642col, col.C64Color)
		delete(p.rgb2col, ColorKey(col))
	}
}

func (p Palette) FromC64(col C64Color) (Color, error) {
	if v, ok := p.c642col[col]; ok {
		return v, nil
	}
	return Color{
		Color:    color.RGBA{},
		C64Color: col,
	}, fmt.Errorf("c64color %d not found", col)
}

func (p Palette) FromColor(col color.Color) (Color, error) {
	if v, ok := p.rgb2col[ColorKey(col)]; ok {
		return v, nil
	}
	return Color{Color: col}, fmt.Errorf("color %v not found", col)
}

func (p Palette) FromC64NoErr(col C64Color) Color {
	c, _ := p.FromC64(col)
	return c
}

func (p Palette) FromColorNoErr(col color.Color) Color {
	c, _ := p.FromColor(col)
	return c
}

// Convert converts a color to a png2prg.Color and returns it, implementing the color.Model interface.
// Finds closest match if p.loose is true.
func (p Palette) Convert(c color.Color) color.Color {
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
// returns the hires bool as true if hires pixels have been detected.
func imageColors(img image.Image) (cc []color.Color, hires bool) {
	m := map[color.Color]struct{}{}
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			col := img.At(x, y)
			if _, ok := m[col]; !ok {
				m[col] = struct{}{}
				cc = append(cc, col)
			}
			if x%2 == 0 {
				if col != img.At(x+1, y) {
					hires = true
				}
			}
		}
	}
	return cc, hires
}

// analyzeColors calculates the color distances of all colors and each of the palletSources.
// It returns the closest matching Palette.
func analyzeColors(cc []color.Color, verbose bool) (found Palette) {
	minDistance := int(9e8)
	for _, src := range paletteSources {
		p := BlankPalette(src.Name, false)
		totalDistance := 0
		for _, c := range cc {
			distance := int(9e8)
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
			p.Add(foundCol)
			totalDistance += distance
		}
		if verbose {
			fmt.Printf("palette %q distance = %d\n", p.Name, totalDistance)
		}
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

// ParseBPC parses the commandline -bitpair-colors string and returns an ordered Color byte-slice.
// A nil *Color (-1 in the cli) means the bitpair has no preference.
func (p Palette) ParseBPC(in string) (cc []*Color, err error) {
	for _, v := range strings.Split(in, ",") {
		i, err := strconv.Atoi(v)
		if err != nil {
			return cc, fmt.Errorf("strconv.Atoi conversion of %q to integers failed: %w", in, err)
		}
		if i < -1 || i >= MaxColors {
			return cc, fmt.Errorf("incorrect c64 color %d", i)
		}
		if i < 0 {
			cc = append(cc, nil)
			continue
		}
		col := p.FromC64NoErr(C64Color(i))
		cc = append(cc, &col)
	}
	return cc, nil
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
		panic(fmt.Errorf("convertPaletteSources failed: %w", err))
	}
	if len(paletteSources) == 0 {
		panic(fmt.Errorf("no palettes found in %q", "palettes.yaml"))
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
		count := 0
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
			ps.Colors[c64col] = NewColor(C64Color(c64col), color.RGBA{byte((rgb >> 16) & 0xff), byte((rgb >> 8) & 0xff), byte(rgb & 0xff), 0xff})
			count++
		}
		if count != MaxColors {
			return out, fmt.Errorf("each palette in palettes.yaml must have %d colors, not %d", MaxColors, count)
		}
		out = append(out, ps)
	}
	return out, nil
}
