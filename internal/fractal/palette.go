package fractal

import (
	"image/color"
	"math"
)

// Palette maps an iteration value in [0,1] (with weight) to a color.
type Palette interface {
	// At returns the color for normalized value t in [0.0, 1.0].
	At(t float64, isSet bool) color.RGBA
	Name() string
}

// ColorStop is a control point for gradient palettes.
type ColorStop struct {
	Pos   float64    `json:"pos"`
	Color color.RGBA `json:"color"`
}

// GradPalette is a palette built from a list of color stops.
type GradPalette struct {
	name  string
	stops []ColorStop
}

// NewGradPalette creates a palette interpolating between stops.
// Stops must be sorted by Pos in ascending order; at least two stops.
func NewGradPalette(name string, stops []ColorStop) *GradPalette {
	return &GradPalette{name: name, stops: stops}
}

func (g *GradPalette) Name() string { return g.name }

func (g *GradPalette) At(t float64, isSet bool) color.RGBA {
	if isSet {
		return color.RGBA{0, 0, 0, 255}
	}
	t = t - math.Floor(t)
	if t <= g.stops[0].Pos {
		return g.stops[0].Color
	}
	for i := 0; i < len(g.stops)-1; i++ {
		a := g.stops[i]
		b := g.stops[i+1]
		if t >= a.Pos && t <= b.Pos {
			f := 0.0
			if b.Pos != a.Pos {
				f = (t - a.Pos) / (b.Pos - a.Pos)
			}
			return lerpRGBA(a.Color, b.Color, f)
		}
	}
	return g.stops[len(g.stops)-1].Color
}

// CyclePalette is a palette defined by a periodic function (HSV hue cycling).
type CyclePalette struct {
	name string
}

// NewCyclePalette returns a palette cycling hue.
func NewCyclePalette(name string) *CyclePalette {
	return &CyclePalette{name: name}
}

func (c *CyclePalette) Name() string { return c.name }

func (c *CyclePalette) At(t float64, isSet bool) color.RGBA {
	if isSet {
		return color.RGBA{0, 0, 0, 255}
	}
	return hsvToRGB(t*360.0, 1.0, 1.0)
}

// hsvToRGB converts HSV (h in [0,360]) to RGB.
func hsvToRGB(h, s, v float64) color.RGBA {
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60.0, 2.0)-1.0))
	m := v - c
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return color.RGBA{
		R: uint8((r + m) * 255),
		G: uint8((g + m) * 255),
		B: uint8((b + m) * 255),
		A: 255,
	}
}

func lerpRGBA(a, b color.RGBA, f float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(a.R) + (float64(b.R)-float64(a.R))*f),
		G: uint8(float64(a.G) + (float64(b.G)-float64(a.G))*f),
		B: uint8(float64(a.B) + (float64(b.B)-float64(a.B))*f),
		A: uint8(float64(a.A) + (float64(b.A)-float64(a.A))*f),
	}
}

// BloomGradient returns a beautiful gradient with a subtle bloom band.
func bloomStops() []ColorStop {
	return []ColorStop{
		{Pos: 0.0, Color: color.RGBA{0, 7, 100, 255}},
		{Pos: 0.16, Color: color.RGBA{32, 107, 203, 255}},
		{Pos: 0.42, Color: color.RGBA{237, 255, 255, 255}},
		{Pos: 0.6425, Color: color.RGBA{255, 170, 0, 255}},
		{Pos: 0.8575, Color: color.RGBA{0, 2, 0, 255}},
		{Pos: 1.0, Color: color.RGBA{0, 7, 100, 255}},
	}
}

// FireStops is a fire-themed gradient.
func fireStops() []ColorStop {
	return []ColorStop{
		{Pos: 0.0, Color: color.RGBA{0, 0, 0, 255}},
		{Pos: 0.3, Color: color.RGBA{64, 0, 0, 255}},
		{Pos: 0.6, Color: color.RGBA{255, 96, 0, 255}},
		{Pos: 0.85, Color: color.RGBA{255, 224, 64, 255}},
		{Pos: 1.0, Color: color.RGBA{255, 255, 255, 255}},
	}
}

// OceanStops is a deep-sea gradient.
func oceanStops() []ColorStop {
	return []ColorStop{
		{Pos: 0.0, Color: color.RGBA{0, 0, 20, 255}},
		{Pos: 0.35, Color: color.RGBA{0, 70, 140, 255}},
		{Pos: 0.7, Color: color.RGBA{80, 200, 220, 255}},
		{Pos: 1.0, Color: color.RGBA{220, 255, 250, 255}},
	}
}

// DefaultPalettes returns the built-in palette set keyed by name.
func DefaultPalettes() map[string]Palette {
	return map[string]Palette{
		"Bloom":   NewGradPalette("Bloom", bloomStops()),
		"Fire":    NewGradPalette("Fire", fireStops()),
		"Ocean":   NewGradPalette("Ocean", oceanStops()),
		"Rainbow": NewCyclePalette("Rainbow"),
		"Grayscale": NewGradPalette("Grayscale", []ColorStop{
			{Pos: 0.0, Color: color.RGBA{0, 0, 0, 255}},
			{Pos: 1.0, Color: color.RGBA{255, 255, 255, 255}},
		}),
	}
}
