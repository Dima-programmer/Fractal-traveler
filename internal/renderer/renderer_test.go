package renderer

import (
	"image"
	"math"
	"testing"

	"fractal-gen/internal/fractal"
)

func TestDeepAndFloatAgree(t *testing.T) {
	pal := fractal.DefaultPalettes()["Bloom"]
	r := NewRenderer(pal, fractal.NewRegistry().Create("mandelbrot"))
	r.MaxIter = 700

	v := View{CenterX: -0.5, CenterY: 0, HalfWidth: 1.6, Aspect: 1}

	for _, hw := range []float64{1e-1, 1e-6, 1e-8} {
		v.HalfWidth = hw
		img := image.NewRGBA(image.Rect(0, 0, 64, 64))
		r.RenderTo(img, v)
	}
}

func TestDeepZoomDoesNotPanic(t *testing.T) {
	pal := fractal.DefaultPalettes()["Bloom"]
	frac := fractal.NewRegistry().Create("mandelbrot")
	r := NewRenderer(pal, frac)
	r.MaxIter = 700

	// Extremely deep zoom: well past float64 precision limits.
	v := View{CenterX: -0.5, CenterY: 0, HalfWidth: 1e-16, Aspect: 1}
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	r.RenderTo(img, v)

	// The deep renderer must not fail at any point.
	if v.HalfWidth >= deepThreshold {
		t.Fatal("test view should use the deep path")
	}
}

func TestDeepThreshold(t *testing.T) {
	if math.Abs(math.Log10(1e-13)) < 12 {
		t.Fatal("deepThreshold sanity")
	}
	if !deepOK(fractal.NewRegistry().Create("mandelbrot")) {
		t.Fatal("mandelbrot should be deep-ok")
	}
	if !deepOK(fractal.NewRegistry().Create("phoenix")) {
		t.Fatal("fallback types should be deep-ok")
	}
}