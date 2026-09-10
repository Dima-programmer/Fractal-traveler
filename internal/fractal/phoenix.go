package fractal

// Phoenix implements z_{n+1} = z_n^2 + c + p * z_{n-1}.
type Phoenix struct {
	cr float64
	ci float64
	pr float64
	pi float64
}

// NewPhoenix returns a Phoenix fractal with defaults.
func NewPhoenix() *Phoenix {
	return &Phoenix{cr: -0.5, ci: 0.0, pr: -0.5, pi: 0.0}
}

func (f *Phoenix) Key() string         { return "phoenix" }
func (f *Phoenix) Name() string        { return "Phoenix" }
func (f *Phoenix) Description() string { return "z = z^2 + c + p·z_prev" }

func (f *Phoenix) Params() []Param {
	return []Param{
		{Key: "c_real", Name: "Re(c)", Min: -2, Max: 2, Default: -0.5, Step: 0.001},
		{Key: "c_imag", Name: "Im(c)", Min: -2, Max: 2, Default: 0, Step: 0.001},
		{Key: "p_real", Name: "Re(p)", Min: -2, Max: 2, Default: -0.5, Step: 0.001},
		{Key: "p_imag", Name: "Im(p)", Min: -2, Max: 2, Default: 0, Step: 0.001},
	}
}

func (f *Phoenix) SetParam(key string, value float64) bool {
	switch key {
	case "c_real":
		f.cr = value
	case "c_imag":
		f.ci = value
	case "p_real":
		f.pr = value
	case "p_imag":
		f.pi = value
	default:
		return false
	}
	return true
}

func (f *Phoenix) GetParam(key string) float64 {
	switch key {
	case "c_real":
		return f.cr
	case "c_imag":
		return f.ci
	case "p_real":
		return f.pr
	case "p_imag":
		return f.pi
	}
	return 0
}

func (f *Phoenix) Escape(px, py float64, maxIter int) (float64, bool) {
	x, y := px, py
	var pxr, pyi float64 // previous z
	var tx, ty float64
	for i := 0; i < maxIter; i++ {
		if x*x+y*y > 4.0 {
			return Smooth(x, y, 4.0, i), true
		}
		tx = x*x - y*y + f.cr + f.pr*pxr - f.pi*pyi
		ty = 2*x*y + f.ci + f.pr*pyi + f.pi*pxr
		pxr, pyi = x, y
		x, y = tx, ty
	}
	return 0, false
}

func (f *Phoenix) Clone() Fractal {
	return &Phoenix{cr: f.cr, ci: f.ci, pr: f.pr, pi: f.pi}
}
