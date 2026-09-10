package fractal

import "math"

// BurningShip implements z = (|Re(z)| + i|Im(z)|)^2 + c.
type BurningShip struct{}

// NewBurningShip returns a BurningShip fractal.
func NewBurningShip() *BurningShip { return &BurningShip{} }

func (b *BurningShip) Key() string         { return "burning_ship" }
func (b *BurningShip) Name() string        { return "Burning Ship" }
func (b *BurningShip) Description() string { return "z = (|Re| + i|Im|)^2 + c" }

func (b *BurningShip) Params() []Param { return nil }

func (b *BurningShip) SetParam(key string, value float64) bool { return false }
func (b *BurningShip) GetParam(key string) float64             { return 0 }

func (b *BurningShip) Escape(cx, cy float64, maxIter int) (float64, bool) {
	var x, y float64
	var x2, y2 float64
	for i := 0; i < maxIter; i++ {
		x2 = x * x
		y2 = y * y
		if x2+y2 > 4.0 {
			return Smooth(x, y, 4.0, i), true
		}
		x = math.Abs(x)
		y = math.Abs(y)
		y = 2*x*y + cy
		x = x2 - y2 + cx
	}
	return 0, false
}

func (b *BurningShip) Clone() Fractal { return &BurningShip{} }
