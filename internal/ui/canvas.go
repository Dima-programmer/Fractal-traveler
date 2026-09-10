package ui

import (
	"fmt"
	"image"
	"math"
	"sync"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"

	"fractal-gen/internal/fractal"
	"fractal-gen/internal/renderer"
)

// Render-region and quality rules:
//   - zooming back out (HalfWidth grew) or panning: render a region 2x the
//     viewport at normal resolution. Because the buffer is twice as wide, the
//     currently shown image keeps covering the whole canvas while the view
//     moves outwards and just "zooms out" like a picture (same apparent
//     scale), until the fresh render for the larger view arrives.
//   - zooming in (HalfWidth shrank): render a region 1.5x the viewport at
//     1.5x quality, so the magnified picture stays sharper than the screen.
const (
	zoomOutRegion  = 2.0
	zoomOutQuality = 1.0
	zoomInRegion   = 1.5
	zoomInQuality  = 1.5
)

// canvas manages the off-screen fractal image and its background rendering.
//
// Pipeline: every invalidation bumps viewGen. While a change is pending and
// no job is running, a FULL-quality buffer is rendered for a view that is
// larger than the viewport. Jobs keep restarting until the view settles — the
// completed buffer is always published (the affine transform keeps it covering
// the canvas meanwhile), so background rendering never waits for a pause.
type canvas struct {
	w, h int // viewport size in pixels

	img    *ebiten.Image // current buffer image (recreated when its size changes)
	appliedW, appliedH int // size of the content currently in img

	// All mutable state is guarded by mu. The worker goroutine renders into a
	// freshly allocated buffer and only exchanges the published slice + gen
	// counter; it never touches an already published buffer again.
	mu sync.Mutex

	viewGen    uint64 // bumped on every invalidation
	fullGen    uint64 // generation whose pixels are in fullPix
	appliedGen uint64 // generation applied to img
	fullPix    []byte // published pixels (owned by main goroutine after publish)
	pubW, pubH int    // size of the published pixels
	vPub       renderer.View // view the published pixels were rendered with
	vApp       renderer.View // view the content currently in img is valid for
	srcView    renderer.View // unenlarged view the buffer was rendered for

	rendering bool
	prog      *atomic.Int64   // renderer progress of the active job (nil when idle)
	progRows  int             // row count the progress is measured against

	autoFull  bool // render in the background while the view changes
	forceFull bool // render forced by the "Полный рендер" button
}

func newCanvas(w, h int) *canvas {
	return &canvas{
		w:        w,
		h:        h,
		viewGen:  1, // first render is wanted on startup
		autoFull: true,
	}
}

// invalidate requests a fresh render of the current view.
func (c *canvas) invalidate() {
	c.mu.Lock()
	c.viewGen++
	c.mu.Unlock()
}

// resize adapts the canvas to a new viewport size and schedules a re-render.
func (c *canvas) resize(w, h int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.w == w && c.h == h {
		return
	}
	c.w = w
	c.h = h
	c.viewGen++
	c.forceFull = true
}

// setAutoFull toggles background rendering while the view changes.
func (c *canvas) setAutoFull(on bool) {
	c.mu.Lock()
	c.autoFull = on
	if on {
		c.forceFull = true
		c.viewGen++
	}
	c.mu.Unlock()
}

// requestFull forces an immediate render.
func (c *canvas) requestFull() {
	c.mu.Lock()
	c.forceFull = true
	c.viewGen++
	c.mu.Unlock()
}

// renderParams picks the render-region factor and quality for the current
// movement direction, measured against the *source* view (the unenlarged view
// the currently applied buffer was rendered for):
//   - view moved outwards (zoom out / pan): 2x region, normal quality;
//   - view moved inwards (zoom in):          1.5x region, 1.5x quality.
func (c *canvas) renderParams(view renderer.View) (region float64, quality float64) {
	src := c.srcView
	if c.appliedGen > 0 && view.HalfWidth >= src.HalfWidth {
		return zoomOutRegion, zoomOutQuality
	}
	return zoomInRegion, zoomInQuality
}

// scaleView returns the view used for the buffer: same center, but half-extents
// scaled by k (k = region factor).
func scaleView(view renderer.View, k float64) renderer.View {
	return renderer.View{
		CenterX:   view.CenterX,
		CenterY:   view.CenterY,
		HalfWidth: view.HalfWidth * k,
		Aspect:    view.Aspect,
	}
}

// update is called from Game.Update on the main goroutine.
// It publishes completed render results and starts background jobs.
func (c *canvas) update(view renderer.View, frac fractal.Fractal, pal fractal.Palette, maxIter int) {
	c.mu.Lock()

	// Publish any completed result first.
	if c.fullGen > c.appliedGen {
		nw, nh := c.pubW, c.pubH
		if c.img == nil || c.img.Bounds().Dx() != nw || c.img.Bounds().Dy() != nh {
			c.img = ebiten.NewImage(nw, nh)
		}
		c.img.WritePixels(c.fullPix)
		c.appliedGen = c.fullGen
		c.vApp = c.vPub
		c.appliedW, c.appliedH = nw, nh
	}

	// Start rendering while the view is still changing (no pause needed):
	// as soon as one job finishes the loop above publishes it and a fresh job
	// starts for the newest view, until viewGen == appliedGen.
	if !c.rendering && (c.autoFull || c.forceFull) && c.viewGen > c.appliedGen {
		c.rendering = true
		gen := c.viewGen
		fracClone := frac.Clone()
		region, quality := c.renderParams(view)
		rview := scaleView(view, region)
		nw := int(math.Ceil(float64(c.w) * region * quality))
		nh := int(math.Ceil(float64(c.h) * region * quality))
		if nw < 1 {
			nw = 1
		}
		if nh < 1 {
			nh = 1
		}
		c.forceFull = false
		c.mu.Unlock()

		r := renderer.NewRenderer(pal, fracClone)
		r.MaxIter = maxIter
		c.mu.Lock()
		c.prog = &r.Progress
		c.progRows = nh
		c.mu.Unlock()

		go func() {
			buf := image.NewRGBA(image.Rect(0, 0, nw, nh))
			r.RenderTo(buf, rview)

			c.mu.Lock()
			c.prog = nil
			c.fullPix = buf.Pix
			c.fullGen = gen
			c.pubW, c.pubH = nw, nh
			c.vPub = rview
			c.srcView = view
			c.rendering = false
			c.mu.Unlock()
		}()
		return
	}
	c.mu.Unlock()
}

// viewTransform maps a buffer pixel (u,v) in [0,fromW]x[0,fromH] that was
// rendered for view "from" into the screen position under view "to" whose
// canvas is toW x toH. Returns the DrawImageOptions for the buffer image.
func viewTransform(from renderer.View, fromW, fromH int, to renderer.View, toW, toH int) *ebiten.DrawImageOptions {
	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	if fromW <= 0 || fromH <= 0 || from.HalfWidth <= 0 || from.Aspect <= 0 || to.HalfWidth <= 0 {
		return op
	}
	// px = u*A + B
	A := from.HalfWidth * float64(toW) / (float64(fromW) * to.HalfWidth)
	B := float64(toW)/2 - from.HalfWidth*float64(toW)/(2*to.HalfWidth) +
		(from.CenterX-to.CenterX)*float64(toW)/(2*to.HalfWidth)
	// py = v*C + D
	fromHH := from.HalfWidth / from.Aspect
	toHH := to.HalfWidth / to.Aspect
	C := fromHH * float64(toH) / (float64(fromH) * toHH)
	D := float64(toH)/2 - fromHH*float64(toH)/(2*toHH) +
		(to.CenterY-from.CenterY)*float64(toH)/(2*toHH)

	// Don't let edges of the viewport show while a fresh render is pending:
	// if the rendered image maps smaller than the canvas (the view moved out
	// further than the buffer's region and the new render is not ready yet),
	// scale the image so it covers the whole canvas, centered. This keeps the
	// old frame visually "zooming out" uniformly; a small margin also hides
	// the translucent fringe produced by linear filtering at the texture edge.
	destW := A * float64(fromW)
	destH := C * float64(fromH)
	needW := float64(toW) + 2
	needH := float64(toH) + 2
	if destW < needW || destH < needH {
		s := needW / destW
		if sh := needH / destH; sh > s {
			s = sh
		}
		A *= s
		C *= s
		destW = A * float64(fromW)
		destH = C * float64(fromH)
		// Center the scaled buffer on the canvas so every corner is covered
		// (anchoring at a fixed world point would leave the far side empty).
		B = (float64(toW) - destW) / 2
		D = (float64(toH) - destH) / 2
	}

	op.GeoM.Scale(A, C)
	op.GeoM.Translate(B, D)
	return op
}

// draw renders the canvas across the full viewport area, transforming the
// latest rendered buffer to match the current view (so it pans/zooms like a
// picture until the background render for a newer view finishes).
func (c *canvas) draw(dst *ebiten.Image, view renderer.View) {
	c.mu.Lock()
	ag := c.appliedGen
	rendering := c.rendering
	aw, ah := c.appliedW, c.appliedH
	vApp := c.vApp
	var pct float64
	if c.prog != nil && c.progRows > 0 {
		pct = float64(c.prog.Load()) / float64(c.progRows)
	}
	c.mu.Unlock()

	if ag > 0 && c.img != nil {
		op := viewTransform(vApp, aw, ah, view, c.w, c.h)
		dst.DrawImage(c.img, op)
	} else {
		fillRect(dst, 0, 0, c.w, c.h, colBg)
	}

	if rendering {
		msg := fmt.Sprintf("Отрисовка… %3d%%", int(pct*100))
		bw := int(textWidth(msg, 12)) + 28
		fillRect(dst, c.w-bw-12, c.h-40, bw, 26, colPanelDark)
		drawText(dst, msg, float64(c.w-bw+2), float64(c.h-34), 12, colText)
	}
}