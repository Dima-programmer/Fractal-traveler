package fractal

// Tricorn (Mandelbar) implements z = conj(z)^2 + c.
type Tricorn struct{}

// NewTricorn returns a Tricorn fractal.
func NewTricorn() *Tricorn { return &Tricorn{} }

func (t *Tricorn) Key() string         { return "tricorn" }
func (t *Tricorn) Name() string        { return "Tricorn" }
func (t *Tricorn) Description() string { return "z = conj(z)^2 + c (Mandеlbar)" }

func (t *Tricorn) Params() []Param { return nil }

func (t *Tricorn) SetParam(key string, value float64) bool { return false }
func (t *Tricorn) GetParam(key string) float64             { return 0 }

func (t *Tricorn) Escape(cx, cy float64, maxIter int) (float64, bool) {
	var x, y float64
	var x2, y2 float64
	for i := 0; i < maxIter; i++ {
		x2 = x * x
		y2 = y * y
		if x2+y2 > 4.0 {
			return Smooth(x, y, 4.0, i), true
		}
		// z = (conj(z))^2 + c ; conj(z) = (x, -y); square -> (x^2-y^2, -2xy)
		y = -2*x*y + cy
		x = x2 - y2 + cx
	}
	return 0, false
}

func (t *Tricorn) Clone() Fractal { return &Tricorn{} }
