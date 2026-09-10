package fractal

import "math"

// Mandelbrot implements the classic z = z^2 + c set.
type Mandelbrot struct {
	power float64
}

// NewMandelbrot returns a Mandelbrot with default params.
func NewMandelbrot() *Mandelbrot {
	return &Mandelbrot{power: 2.0}
}

func (m *Mandelbrot) Key() string         { return "mandelbrot" }
func (m *Mandelbrot) Name() string        { return "Mandelbrot" }
func (m *Mandelbrot) Description() string { return "Классический набор: z = z^p + c" }

func (m *Mandelbrot) Params() []Param {
	return []Param{
		{Key: "power", Name: "Степень", Min: 2, Max: 8, Default: 2, Step: 0.1},
	}
}

func (m *Mandelbrot) SetParam(key string, value float64) bool {
	switch key {
	case "power":
		m.power = value
		return true
	}
	return false
}

func (m *Mandelbrot) GetParam(key string) float64 {
	if key == "power" {
		return m.power
	}
	return 0
}

func (m *Mandelbrot) Escape(cx, cy float64, maxIter int) (float64, bool) {
	var x, y float64
	p := m.power
	var x2, y2 float64
	for i := 0; i < maxIter; i++ {
		x2 = x * x
		y2 = y * y
		if x2+y2 > 4.0 {
			return Smooth(x, y, 4.0, i), true
		}
		// z^p via angle/magnitude for non-integer powers
		if p == 2.0 {
			y = 2*x*y + cy
			x = x2 - y2 + cx
		} else {
			r := math.Pow(x2+y2, p/2.0)
			th := p * math.Atan2(y, x)
			y = r*math.Sin(th) + cy
			x = r*math.Cos(th) + cx
		}
	}
	return 0, false
}

func (m *Mandelbrot) Clone() Fractal {
	return &Mandelbrot{power: m.power}
}
