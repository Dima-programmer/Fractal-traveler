package ui

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var (
	colBg        = color.RGBA{24, 26, 32, 255}
	colPanel     = color.RGBA{32, 35, 44, 255}
	colPanelDark = color.RGBA{26, 28, 36, 255}
	colBorder    = color.RGBA{60, 64, 78, 255}
	colAccent    = color.RGBA{96, 160, 232, 255}
	colAccentDim = color.RGBA{60, 96, 140, 255}
	colText      = color.RGBA{220, 224, 232, 255}
	colTextDim   = color.RGBA{150, 155, 168, 255}
	colHover     = color.RGBA{150, 160, 180, 255}
	colSliderBg  = color.RGBA{52, 56, 68, 255}
)

// Button is a simple clickable rectangle with a label.
type Button struct {
	X, Y, W, H int
	Label      string
	Active     bool
	OnClick    func()
}

func (b *Button) Contains(x, y int) bool {
	return x >= b.X && x <= b.X+b.W && y >= b.Y-widgetYOff && y <= b.Y-widgetYOff+b.H
}

func (b *Button) Update(x, y int, pressed, justPressed bool) {
	if b.Contains(x, y) && justPressed {
		if b.OnClick != nil {
			b.OnClick()
		}
	}
}

func (b *Button) Draw(dst *ebiten.Image) {
	yy := b.Y - widgetYOff
	bg := colPanelDark
	if b.Contains(mouseX, mouseY) {
		bg = colHover
	}
	if b.Active {
		bg = colAccentDim
	}
	fillRect(dst, b.X, yy, b.W, b.H, bg)
	drawRectBorder(dst, b.X, yy, b.W, b.H, colBorder)
	clr := colText
	if b.Active {
		clr = colText
	}
	drawTextCenter(dst, b.Label, float64(b.X)+float64(b.W)/2, float64(yy)+float64(b.H)/2-4, 12, clr)
}

// Checkbox is a toggle with a label.
type Checkbox struct {
	X, Y, W, H int
	Label      string
	Checked    bool
	OnToggle   func(on bool)
}

func (c *Checkbox) Contains(x, y int) bool {
	return x >= c.X && x <= c.X+c.W && y >= c.Y-widgetYOff && y <= c.Y-widgetYOff+c.H
}

func (c *Checkbox) Update(x, y int, justPressed bool) {
	if justPressed && c.Contains(x, y) {
		c.Checked = !c.Checked
		if c.OnToggle != nil {
			c.OnToggle(c.Checked)
		}
	}
}

func (c *Checkbox) Draw(dst *ebiten.Image) {
	yy := c.Y - widgetYOff
	box := 16
	bg := colPanelDark
	fg := colAccent
	if c.Contains(mouseX, mouseY) {
		bg = colHover
	}
	fillRect(dst, c.X, yy, box, box, bg)
	drawRectBorder(dst, c.X, yy, box, box, colBorder)
	if c.Checked {
		fillRect(dst, c.X+3, yy+3, box-6, box-6, fg)
	}
	drawText(dst, c.Label, float64(c.X+box+8), float64(yy)+2, 12, colTextDim)
}

// Slider is a horizontal value slider.
type Slider struct {
	X, Y, W, H  int
	Label       string
	Min, Max    float64
	Step        float64
	Value       float64
	Format      string
	ValueString func(v float64) string
	OnChanged   func(v float64)
	dragging    bool
}

func (s *Slider) Contains(x, y int) bool {
	yy := s.Y - widgetYOff
	return x >= s.X && x <= s.X+s.W && y >= yy-4 && y <= yy+s.H+4
}

// Update handles drag interaction.
func (s *Slider) Update(x, y int, pressed bool) {
	if pressed && s.Contains(x, y) {
		s.dragging = true
	}
	if !pressed {
		s.dragging = false
	}
	if s.dragging {
		s.valueFromX(x)
	}
}

func (s *Slider) valueFromX(x int) {
	f := float64(x-s.X) / float64(s.W)
	f = math.Max(0, math.Min(1, f))
	v := s.Min + f*(s.Max-s.Min)
	if s.Step > 0 {
		v = math.Round(v/s.Step) * s.Step
		v = math.Max(s.Min, math.Min(s.Max, v))
	}
	if v != s.Value {
		s.Value = v
		if s.OnChanged != nil {
			s.OnChanged(v)
		}
	}
}

func (s *Slider) Draw(dst *ebiten.Image) {
	yy := s.Y - widgetYOff
	if s.ValueString != nil {
		label := s.Label + ": " + s.ValueString(s.Value)
		drawText(dst, label, float64(s.X), float64(yy-20), 11, colTextDim)
	} else {
		if s.Format == "" {
			s.Format = "%.4g"
		}
		label := s.Label + ": " + fmt.Sprintf(s.Format, s.Value)
		drawText(dst, label, float64(s.X), float64(yy-20), 11, colTextDim)
	}

	// Track
	fillRect(dst, s.X, yy, s.W, s.H, colSliderBg)
	f := (s.Value - s.Min) / (s.Max - s.Min)
	knobX := s.X + int(f*float64(s.W))
	fillRect(dst, s.X, yy+1, knobX-s.X, s.H-2, colAccent)

	// Knob
	fillRect(dst, knobX-3, yy-3, 7, s.H+6, colText)
}

// TextField is a single-line editable text box.
type TextField struct {
	X, Y, W, H int
	Label      string
	Text       string
	Focused    bool
	OnChange   func(text string)
}

func (f *TextField) Contains(x, y int) bool {
	yy := f.Y - widgetYOff
	return x >= f.X && x <= f.X+f.W && y >= yy && y <= yy+f.H
}

// Update handles focus and typing. Returns true if the value changed.
func (f *TextField) Update(x, y int, pressed bool) bool {
	if pressed {
		f.Focused = f.Contains(x, y)
	}
	if !f.Focused {
		return false
	}
	changed := false
	runes := ebiten.AppendInputChars(nil)
	if len(runes) > 0 {
		f.Text += string(runes)
		changed = true
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(f.Text) > 0 {
		f.Text = f.Text[:len(f.Text)-1]
		changed = true
	}
	if changed && f.OnChange != nil {
		f.OnChange(f.Text)
	}
	return changed
}

func (f *TextField) Draw(dst *ebiten.Image) {
	yy := f.Y - widgetYOff
	if f.Label != "" {
		drawText(dst, f.Label, float64(f.X), float64(yy-18), 11, colTextDim)
	}
	bg := colPanelDark
	if f.Focused {
		bg = color.RGBA{40, 52, 68, 255}
	}
	fillRect(dst, f.X, yy, f.W, f.H, bg)
	drawRectBorder(dst, f.X, yy, f.W, f.H, colBorder)

	txt := f.Text
	if f.Focused {
		txt += "|"
	}
	drawText(dst, txt, float64(f.X+6), float64(yy)+float64(f.H)/2-8, 12, colText)
}

// keyboard shortcuts / helpers

// mouse state captured once per frame
var mouseX, mouseY int

// widgetYOff is a shared vertical scroll offset applied to all sidebar widgets.
// It is set to the sidebar scroll amount before Update/Draw and reset to 0
// afterwards, so widgets always hit-test and render scrolled together.
var widgetYOff int

// whitePixel is a shared 1x1 white image used by fillRect.
var whitePixel *ebiten.Image

func captureMouse() {
	mouseX, mouseY = ebiten.CursorPosition()
}

// fillRect draws a filled rectangle.
func fillRect(dst *ebiten.Image, x, y, w, h int, clr color.Color) {
	if w <= 0 || h <= 0 {
		return
	}
	if whitePixel == nil {
		whitePixel = ebiten.NewImage(1, 1)
		whitePixel.Fill(color.White)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(w), float64(h))
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(clr)
	op.Filter = ebiten.FilterNearest
	dst.DrawImage(whitePixel, op)
}

// drawRectBorder draws a 1px rectangle outline.
func drawRectBorder(dst *ebiten.Image, x, y, w, h int, clr color.Color) {
	fillRect(dst, x, y, w, 1, clr)
	fillRect(dst, x, y+h-1, w, 1, clr)
	fillRect(dst, x, y, 1, h, clr)
	fillRect(dst, x+w-1, y, 1, h, clr)
}

// clampString truncates a string to fit maxWidth px at size.
func clampString(s string, maxWidth, size float64) string {
	if textWidth(s, size) <= maxWidth {
		return s
	}
	for len(s) > 0 {
		s = s[:len(s)-1]
		if textWidth(s+"…", size) <= maxWidth {
			return s + "…"
		}
	}
	return s
}

func strContainsFold(s, sub string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(sub))
}
