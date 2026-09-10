package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"

	"os"
	"strconv"
)

var (
	faceSmall  text.Face
	faceNormal text.Face
	faceTitle  text.Face

	// Global vertical shift for text to align with UI elements.
	// Negative moves text upward. Can be overridden by env FRACTAL_TEXT_SHIFT.
	textYShift int
)

func initFonts() {
	tt, err := opentype.Parse(goregular.TTF)
	if err != nil {
		panic("ui: cannot parse Go fonts: " + err.Error())
	}
	mk := func(size float64) text.Face {
		f, err := opentype.NewFace(tt, &opentype.FaceOptions{
			Size:    size,
			DPI:     72,
			Hinting: font.HintingFull,
		})
		if err != nil {
			panic("ui: cannot create face: " + err.Error())
		}
		return text.NewGoXFace(f)
	}
	faceSmall = mk(12)
	faceNormal = mk(14)
	faceTitle = mk(18)
}

func init() {
	textYShift = -4 // default small upward shift
	if v := os.Getenv("FRACTAL_TEXT_SHIFT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			textYShift = n
		}
	}
}

// drawText draws a single line of text. x,y is the top-left origin.
func drawText(dst *ebiten.Image, s string, x, y float64, size float64, clr color.Color) {
	face := faceFor(size)
	op := &text.DrawOptions{}
	op.DrawImageOptions.GeoM.Translate(x, y+float64(textYShift))
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(dst, s, face, op)
}

// drawTextRight draws text right-aligned at the given right edge.
func drawTextRight(dst *ebiten.Image, s string, x, y float64, size float64, clr color.Color) {
	face := faceFor(size)
	w, _ := text.Measure(s, face, size)
	drawText(dst, s, x-w, y, size, clr)
}

// drawTextCenter draws text centered around the given x position.
func drawTextCenter(dst *ebiten.Image, s string, cx, y float64, size float64, clr color.Color) {
	face := faceFor(size)
	w, _ := text.Measure(s, face, size)
	drawText(dst, s, cx-w/2, y, size, clr)
}

// textWidth returns the pixel width of a string at the given size.
func textWidth(s string, size float64) float64 {
	w, _ := text.Measure(s, faceFor(size), size)
	return w
}

func faceFor(size float64) text.Face {
	switch {
	case size <= 12:
		return faceSmall
	case size <= 14:
		return faceNormal
	default:
		return faceTitle
	}
}

// sizedLine returns the approximate line height for a face size.
func sizedLine(size float64) float64 {
	return size * 1.2
}
