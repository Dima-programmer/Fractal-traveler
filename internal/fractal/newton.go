package fractal

import "math"

// Newton solves z^3 - 1 = 0 by Newton's method.
// Points are colored by which root they converge to, plus iteration count.
type Newton struct{}

// NewNewton returns a Newton fractal.
func NewNewton() *Newton { return &Newton{} }

func (n *Newton) Key() string         { return "newton" }
func (n *Newton) Name() string        { return "Newton" }
func (n *Newton) Description() string { return "Метод Ньютона для z^3 - 1 = 0" }

func (n *Newton) Params() []Param { return nil }

func (n *Newton) SetParam(key string, value float64) bool { return false }
func (n *Newton) GetParam(key string) float64             { return 0 }

func (n *Newton) Escape(px, py float64, maxIter int) (float64, bool) {
	x, y := px, py
	tol := 1e-9
	for i := 0; i < maxIter; i++ {
		// f(z) = z^3 - 1 ; f'(z) = 3 z^2
		x2 := x * x
		y2 := y * y
		r2 := x2 + y2
		if r2 > 1e12 {
			return float64(i), true
		}
		// z^3 = (x^2 - y^2 + 2ixy)*z
		x3 := (x2-y2)*x - 2*x*y*y - 1
		y3 := 2*x*y*x + (x2-y2)*y
		// z^2
		z2r := x2 - y2
		z2i := 2 * x * y
		den := 3 * (z2r*z2r + z2i*z2i)
		if den == 0 {
			return float64(i), true
		}
		// z - f/f' = z - (z^3-1)*(conj z^2)/(3 |z^2|^2)
		// (z^3-1) * conj(z^2) / |z^2|^2
		numr := x3*z2r + y3*z2i
		numi := y3*z2r - x3*z2i
		x = x - 3*numr/den
		y = y - 3*numi/den
		// convergence check: |f(z)| < tol
		if math.Abs(x3) < tol && math.Abs(y3) < tol {
			return float64(i), true
		}
	}
	return 0, false
}

func (n *Newton) Clone() Fractal { return &Newton{} }
