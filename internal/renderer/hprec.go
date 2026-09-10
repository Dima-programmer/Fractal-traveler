package renderer

import (
	"image"
	"image/color"
	"math"
	"math/big"
	"sync"

	"fractal-gen/internal/fractal"
)

// deepThreshold: below this HalfWidth float64 can no longer tell neighbouring
// pixels apart (offset relative to |center| approaches 2^-52), so we switch to
// arbitrary-precision arithmetic.
const deepThreshold = 1e-13

// deepOK reports whether r.Frac can be computed in arbitrary precision
// (mandelbrot/julia with an integer power; everything else).
func deepOK(f fractal.Fractal) bool {
	switch f.Key() {
	case "mandelbrot", "julia":
		p := f.GetParam("power")
		return p >= 2 && p <= 8 && p == math.Trunc(p)
	default:
		return true
	}
}

// deepPrec computes the number of mantissa bits needed to resolve a single
// pixel step at the given view.
func deepPrec(v View, width int) int {
	step := 2 * v.HalfWidth / float64(width)
	if step <= 0 {
		return 4000
	}
	center := 1 + math.Abs(v.CenterX) + math.Abs(v.CenterY)
	p := 32 + int(math.Ceil(math.Log2(center))) + int(math.Ceil(-math.Log2(step)))
	if p < 96 {
		p = 96
	}
	if p > 4000 {
		p = 4000
	}
	return p
}

// bigF is a thin wrapper keeping the shared precision on every Float.
type bigF struct {
	prec uint
}

func (b bigF) New(v float64) *big.Float { return new(big.Float).SetPrec(b.prec).SetFloat64(v) }

// bigc is a complex number in big arithmetic.
type bigc struct {
	r, i *big.Float
}

// RenderToDeep renders dst with arbitrary-precision arithmetic for the given
// view, used when float64 precision is exhausted. Supports all built-in types
// (mandelbrot/julia for integer power).
func (r *Renderer) RenderToDeep(dst *image.RGBA, v View) {
	width := dst.Bounds().Dx()
	height := dst.Bounds().Dy()
	if r.Workers <= 0 {
		r.Workers = 1
	}

	prec := deepPrec(v, width)
	bf := bigF{prec: uint(prec)}

	// Row start values, computed exactly so that per-pixel offsets keep full
	// precision relative to the center.
	stepX := bf.New(2 * v.HalfWidth / float64(width))
	stepY := bf.New(2 * (v.HalfWidth / v.Aspect) / float64(height))
	startX := bf.New(v.CenterX)
	startX.Sub(startX, bf.New(v.HalfWidth))
	startY := bf.New(v.CenterY)
	startY.Add(startY, bf.New(v.HalfWidth/v.Aspect))

	type rowJob int
	rowCh := make(chan rowJob, r.Workers)
	var wg sync.WaitGroup
	wg.Add(r.Workers)

	r.Progress.Store(0)

	escape := deepEscape(r.Frac, bf, r.MaxIter)

	for wi := 0; wi < r.Workers; wi++ {
		go func() {
			defer wg.Done()
			buf := make([]uint8, width*4)
			for j := range rowCh {
				y := int(j)
				cx := new(big.Float).SetPrec(bf.prec).Set(startX)
				cy := new(big.Float).SetPrec(bf.prec).Set(startY)
				for i := 0; i < y; i++ {
					cy.Sub(cy, stepY)
				}
				for x := 0; x < width; x++ {
					n, trapped := escape(cx, cy)
					if x > 0 {
						cx.Add(cx, stepX)
					}
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

// deepEscape returns a per-point escape evaluator computing in big arithmetic.
func deepEscape(f fractal.Fractal, bf bigF, maxIter int) func(cx, cy *big.Float) (float64, bool) {
	switch f.Key() {
	case "mandelbrot":
		p := int(f.GetParam("power"))
		return func(cx, cy *big.Float) (float64, bool) {
			return mandelEscapePower(bf, cx, cy, cx, cy, p, maxIter)
		}
	case "julia":
		p := int(f.GetParam("power"))
		cr := bf.New(f.GetParam("c_real"))
		ci := bf.New(f.GetParam("c_imag"))
		return func(cx, cy *big.Float) (float64, bool) {
			return mandelEscapePower(bf, cx, cy, cr, ci, p, maxIter)
		}
	case "burning_ship":
		return func(cx, cy *big.Float) (float64, bool) {
			return bsEscape(bf, cx, cy, maxIter)
		}
	case "tricorn":
		return func(cx, cy *big.Float) (float64, bool) {
			return tricornEscape(bf, cx, cy, maxIter)
		}
	case "phoenix":
		cr := bf.New(f.GetParam("c_real"))
		ci := bf.New(f.GetParam("c_imag"))
		pr := bf.New(f.GetParam("p_real"))
		pi := bf.New(f.GetParam("p_imag"))
		return func(cx, cy *big.Float) (float64, bool) {
			return phoenixEscape(bf, cx, cy, cr, ci, pr, pi, maxIter)
		}
	case "newton":
		return func(cx, cy *big.Float) (float64, bool) {
			return newtonEscape(bf, cx, cy, maxIter)
		}
	default:
		return func(cx, cy *big.Float) (float64, bool) {
			x, _ := cx.Float64()
			y, _ := cy.Float64()
			return f.Escape(x, y, maxIter)
		}
	}
}

// escapeCheck adds x²+y² and compares against 4.
func escapeCheck(bf bigF, x, y *big.Float) bool {
	x2 := new(big.Float).SetPrec(bf.prec).Mul(x, x)
	y2 := new(big.Float).SetPrec(bf.prec).Mul(y, y)
	s := new(big.Float).SetPrec(bf.prec).Add(x2, y2)
	return s.Cmp(bf.New(4)) > 0
}

func smoothVal(bf bigF, x, y *big.Float, i int) float64 {
	fx, _ := x.Float64()
	fy, _ := y.Float64()
	return fractal.Smooth(fx, fy, 4.0, i)
}

// powZ2 computes z^2 into (pr, pi).
func powZ2(bf bigF, x, y, pr, pi *big.Float) {
	x2 := new(big.Float).SetPrec(bf.prec).Mul(x, x)
	y2 := new(big.Float).SetPrec(bf.prec).Mul(y, y)
	xy := new(big.Float).SetPrec(bf.prec).Mul(x, y)
	pr.Sub(x2, y2)
	pi.SetPrec(bf.prec).Mul(bf.New(2), xy)
}

func mandelEscapePower(bf bigF, zx, zy, cx, cy *big.Float, power, maxIter int) (float64, bool) {
	x := bf.New(0)
	y := bf.New(0)
	for i := 0; i < maxIter; i++ {
		if escapeCheck(bf, x, y) {
			return smoothVal(bf, x, y, i), true
		}
		// z = z^p + c (integer p), p-1 complex multiplies.
		var rp, ip *big.Float
		if power == 2 {
			rp, ip = new(big.Float).SetPrec(bf.prec), new(big.Float).SetPrec(bf.prec)
			powZ2(bf, x, y, rp, ip)
		} else {
			rp = new(big.Float).SetPrec(bf.prec).Set(x)
			ip = new(big.Float).SetPrec(bf.prec).Set(y)
			for k := 1; k < power; k++ {
				nr := new(big.Float).SetPrec(bf.prec).Mul(rp, x)
				ni := new(big.Float).SetPrec(bf.prec).Mul(ip, y)
				nr2 := new(big.Float).SetPrec(bf.prec).Mul(rp, y)
				ni2 := new(big.Float).SetPrec(bf.prec).Mul(ip, x)
				// (rp+ip*i)*(x+y*i) = (rp*x-ip*y) + (rp*y+ip*x)*i
				xr := new(big.Float).SetPrec(bf.prec).Sub(nr, ni)
				yi := new(big.Float).SetPrec(bf.prec).Add(nr2, ni2)
				rp.Set(xr)
				ip.Set(yi)
			}
			// rp+ip*i = z^p
		}
		x.SetPrec(bf.prec).Add(rp, cx)
		y.SetPrec(bf.prec).Add(ip, cy)
	}
	return 0, false
}

func bsEscape(bf bigF, cx, cy *big.Float, maxIter int) (float64, bool) {
	x := bf.New(0)
	y := bf.New(0)
	for i := 0; i < maxIter; i++ {
		if escapeCheck(bf, x, y) {
			return smoothVal(bf, x, y, i), true
		}
		x.Abs(x)
		y.Abs(y)
		x2 := new(big.Float).SetPrec(bf.prec).Mul(x, x)
		y2 := new(big.Float).SetPrec(bf.prec).Mul(y, y)
		xy := new(big.Float).SetPrec(bf.prec).Mul(x, y)
		y.SetPrec(bf.prec).Mul(bf.New(2), xy)
		y.Add(y, cy)
		x.SetPrec(bf.prec).Sub(x2, y2)
		x.Add(x, cx)
	}
	return 0, false
}

func tricornEscape(bf bigF, cx, cy *big.Float, maxIter int) (float64, bool) {
	x := bf.New(0)
	y := bf.New(0)
	for i := 0; i < maxIter; i++ {
		if escapeCheck(bf, x, y) {
			return smoothVal(bf, x, y, i), true
		}
		x2 := new(big.Float).SetPrec(bf.prec).Mul(x, x)
		y2 := new(big.Float).SetPrec(bf.prec).Mul(y, y)
		xy := new(big.Float).SetPrec(bf.prec).Mul(x, y)
		ny := new(big.Float).SetPrec(bf.prec).Mul(bf.New(-2), xy)
		x.SetPrec(bf.prec).Sub(x2, y2)
		x.Add(x, cx)
		y.Set(ny)
		y.Add(y, cy)
	}
	return 0, false
}

func phoenixEscape(bf bigF, cx, cy, cr, ci, pr, pi *big.Float, maxIter int) (float64, bool) {
	x := new(big.Float).SetPrec(bf.prec).Set(cx)
	y := new(big.Float).SetPrec(bf.prec).Set(cy)
	var pxr, pyi *big.Float // previous z
	pxr = bf.New(0)
	pyi = bf.New(0)
	for i := 0; i < maxIter; i++ {
		if escapeCheck(bf, x, y) {
			return smoothVal(bf, x, y, i), true
		}
		x2 := new(big.Float).SetPrec(bf.prec).Mul(x, x)
		y2 := new(big.Float).SetPrec(bf.prec).Mul(y, y)
		xy := new(big.Float).SetPrec(bf.prec).Mul(x, y)
		// tx = x² - y² + cr + pr*pxr - pi*pyi
		tx := new(big.Float).SetPrec(bf.prec).Sub(x2, y2)
		tx.Add(tx, cr)
		t1 := new(big.Float).SetPrec(bf.prec).Mul(pr, pxr)
		t2 := new(big.Float).SetPrec(bf.prec).Mul(pi, pyi)
		tx.Add(tx, t1)
		tx.Sub(tx, t2)
		// ty = 2xy + ci + pr*pyi + pi*pxr
		ty := new(big.Float).SetPrec(bf.prec).Mul(bf.New(2), xy)
		ty.Add(ty, ci)
		t3 := new(big.Float).SetPrec(bf.prec).Mul(pr, pyi)
		t4 := new(big.Float).SetPrec(bf.prec).Mul(pi, pxr)
		ty.Add(ty, t3)
		ty.Add(ty, t4)
		pxr.Set(x)
		pyi.Set(y)
		x.Set(tx)
		y.Set(ty)
	}
	return 0, false
}

func newtonEscape(bf bigF, cx, cy *big.Float, maxIter int) (float64, bool) {
	x := new(big.Float).SetPrec(bf.prec).Set(cx)
	y := new(big.Float).SetPrec(bf.prec).Set(cy)
	tol := bf.New(1e-9)
	big12 := bf.New(1e12)
	for i := 0; i < maxIter; i++ {
		x2 := new(big.Float).SetPrec(bf.prec).Mul(x, x)
		y2 := new(big.Float).SetPrec(bf.prec).Mul(y, y)
		r2 := new(big.Float).SetPrec(bf.prec).Add(x2, y2)
		if r2.Cmp(big12) > 0 {
			return float64(i), true
		}
		// z2 = z^2
		z2r := new(big.Float).SetPrec(bf.prec).Sub(x2, y2)
		xy := new(big.Float).SetPrec(bf.prec).Mul(x, y)
		z2i := new(big.Float).SetPrec(bf.prec).Mul(bf.New(2), xy)
		// z3 = z^3 = z2*z
		z3r := new(big.Float).SetPrec(bf.prec).Mul(z2r, x)
		t := new(big.Float).SetPrec(bf.prec).Mul(z2i, y)
		z3r.Sub(z3r, t)
		z3i := new(big.Float).SetPrec(bf.prec).Mul(z2r, y)
		t2 := new(big.Float).SetPrec(bf.prec).Mul(z2i, x)
		z3i.Add(z3i, t2)
		// f = z3 - 1
		fr := new(big.Float).SetPrec(bf.prec).Sub(z3r, bf.New(1))
		fi := new(big.Float).SetPrec(bf.prec).Set(z3i)
		// f' = 3 z2
		dr := new(big.Float).SetPrec(bf.prec).Mul(bf.New(3), z2r)
		di := new(big.Float).SetPrec(bf.prec).Mul(bf.New(3), z2i)
		// den = |f'|²
		den := new(big.Float).SetPrec(bf.prec).Mul(dr, dr)
		t3 := new(big.Float).SetPrec(bf.prec).Mul(di, di)
		den.Add(den, t3)
		if den.Sign() == 0 {
			return float64(i), true
		}
		// f/f' = (fr+fi*i)/(dr+di*i) = ((fr*dr+fi*di)/den) + ((fi*dr-fr*di)/den)*i
		numr := new(big.Float).SetPrec(bf.prec).Mul(fr, dr)
		t4 := new(big.Float).SetPrec(bf.prec).Mul(fi, di)
		numr.Add(numr, t4)
		numi := new(big.Float).SetPrec(bf.prec).Mul(fi, dr)
		t5 := new(big.Float).SetPrec(bf.prec).Mul(fr, di)
		numi.Sub(numi, t5)
		qr := new(big.Float).SetPrec(bf.prec).Quo(numr, den)
		qi := new(big.Float).SetPrec(bf.prec).Quo(numi, den)
		// z = z - f/f'
		x.Sub(x, qr)
		y.Sub(y, qi)
		// convergence check |f| < tol
		if fr.Abs(fr).Cmp(tol) < 0 && fi.Abs(fi).Cmp(tol) < 0 {
			return float64(i), true
		}
	}
	return 0, false
}