package fractal

import "math"

// Julia implements z = z^2 + c with a fixed c chosen by the user.
type Julia struct {
	power float64
	cr    float64
	ci    float64
}

// NewJulia returns a Julia fractal with a classic c.
func NewJulia() *Julia {
	return &Julia{power: 2.0, cr: -0.8, ci: 0.156}
}

func (j *Julia) Key() string  { return "julia" }
func (j *Julia) Name() string { return "Julia" }
func (j *Julia) Description() string {
	return "z = z^p + c; c выбирается кликом ПКМ"
}

func (j *Julia) Params() []Param {
	return []Param{
		{Key: "power", Name: "Степень", Min: 2, Max: 8, Default: 2, Step: 0.1},
		{Key: "c_real", Name: "Re(c)", Min: -2, Max: 2, Default: -0.8, Step: 0.001},
		{Key: "c_imag", Name: "Im(c)", Min: -2, Max: 2, Default: 0.156, Step: 0.001},
	}
}

func (j *Julia) SetParam(key string, value float64) bool {
	switch key {
	case "power":
		j.power = value
	case "c_real":
		j.cr = value
	case "c_imag":
		j.ci = value
	default:
		return false
	}
	return true
}

func (j *Julia) GetParam(key string) float64 {
	switch key {
	case "power":
		return j.power
	case "c_real":
		return j.cr
	case "c_imag":
		return j.ci
	}
	return 0
}

func (j *Julia) Escape(px, py float64, maxIter int) (float64, bool) {
	x, y := px, py
	p := j.power
	var x2, y2 float64
	for i := 0; i < maxIter; i++ {
		x2 = x * x
		y2 = y * y
		if x2+y2 > 4.0 {
			return Smooth(x, y, 4.0, i), true
		}
		if p == 2.0 {
			y = 2*x*y + j.ci
			x = x2 - y2 + j.cr
		} else {
			r := math.Pow(x2+y2, p/2.0)
			th := p * math.Atan2(y, x)
			y = r*math.Sin(th) + j.ci
			x = r*math.Cos(th) + j.cr
		}
	}
	return 0, false
}

func (j *Julia) Clone() Fractal {
	return &Julia{power: j.power, cr: j.cr, ci: j.ci}
}
