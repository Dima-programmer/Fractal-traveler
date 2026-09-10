package renderer

import (
	"image"
	"image/color"
	"runtime"
	"sync"
	"sync/atomic"

	"fractal-gen/internal/fractal"
)

// View defines which region of the complex plane is visible.
type View struct {
	// Center of the view in complex coordinates.
	CenterX, CenterY float64
	// HalfWidth is half of the visible width in complex units.
	HalfWidth float64
	// Aspect is width/height of the target image.
	Aspect float64
}

// ZoomAt returns a new View zoomed by factor f around a point (px, py)
// given in normalized coordinates [0,1] of the image. The point under the
// cursor stays fixed on screen.
func (v View) ZoomAt(f, px, py float64) View {
	oldW := v.HalfWidth
	oldH := oldW / v.Aspect
	// World coordinates of the anchor point in the old view.
	aw := v.CenterX + (px-0.5)*2.0*oldW
	ah := v.CenterY - (py-0.5)*2.0*oldH
	// New half-extents.
	newW := oldW * f
	newH := newW / v.Aspect
	// Place the anchor back at the same normalized position.
	cx := aw - (px-0.5)*2.0*newW
	cy := ah + (py-0.5)*2.0*newH
	return View{CenterX: cx, CenterY: cy, HalfWidth: newW, Aspect: v.Aspect}
}

// Pan shifts the view by an offset in pixels.
func (v View) Pan(dx, dy float64, imgW, imgH float64) View {
	dxC := dx / imgW * 2.0 * v.HalfWidth
	dyC := dy / imgH * 2.0 * (v.HalfWidth / v.Aspect)
	return View{
		CenterX:   v.CenterX - dxC,
		CenterY:   v.CenterY + dyC,
		HalfWidth: v.HalfWidth,
		Aspect:    v.Aspect,
	}
}

// PixelToComplex maps image coordinates to complex coordinates.
func (v View) PixelToComplex(x, y, w, h float64) (float64, float64) {
	px := v.CenterX + (x/w-0.5)*2.0*v.HalfWidth
	py := v.CenterY - (y/h-0.5)*2.0*(v.HalfWidth/v.Aspect)
	return px, py
}

const defaultMaxIter = 700

// Renderer computes fractal images with a pool of workers.
type Renderer struct {
	Palette fractal.Palette
	Frac    fractal.Fractal
	MaxIter int

	Workers int

	// Progress is the number of completed rows.
	Progress atomic.Int64
}

// NewRenderer creates a renderer tuned for the current machine.
func NewRenderer(pal fractal.Palette, f fractal.Fractal) *Renderer {
	return &Renderer{
		Palette: pal,
		Frac:    f,
		MaxIter: defaultMaxIter,
		Workers: runtime.NumCPU(),
	}
}

// RenderTo fills dst (in place) for the given view using all workers.
func (r *Renderer) RenderTo(dst *image.RGBA, v View) {
	width := dst.Bounds().Dx()
	height := dst.Bounds().Dy()
	if r.Workers <= 0 {
		r.Workers = 1
	}

	// Deep zoom: plain float64 loses precision below ~1e-13 half-width,
	// switching to arbitrary-precision arithmetic avoids pixelation.
	if v.HalfWidth < deepThreshold && deepOK(r.Frac) {
		r.RenderToDeep(dst, v)
		return
	}

	type rowJob int
	rowCh := make(chan rowJob, r.Workers)
	var wg sync.WaitGroup
	wg.Add(r.Workers)

	r.Progress.Store(0)

	for wi := 0; wi < r.Workers; wi++ {
		go func() {
			defer wg.Done()
			buf := make([]uint8, width*4)
			for j := range rowCh {
				y := int(j)
				for x := 0; x < width; x++ {
					cx, cy := v.PixelToComplex(float64(x), float64(y), float64(width), float64(height))
					n, trapped := r.Frac.Escape(cx, cy, r.MaxIter)
					var c color.RGBA
					if trapped {
						c = r.Palette.At(n/float64(r.MaxIter)*2.0, false)
					} else {
						c = r.Palette.At(0, true)
					}
					buf[x*4] = c.R
					buf[x*4+1] = c.G
					buf[x*4+2] = c.B
					buf[x*4+3] = 255
				}
				rowStart := dst.PixOffset(0, y)
				copy(dst.Pix[rowStart:rowStart+width*4], buf)
				r.Progress.Add(1)
			}
		}()
	}

	for y := 0; y < height; y++ {
		rowCh <- rowJob(y)
	}
	close(rowCh)
	wg.Wait()
}
