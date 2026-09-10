package fractal

import "math"

// Param describes a single adjustable parameter of a fractal.
type Param struct {
	Key     string
	Name    string
	Min     float64
	Max     float64
	Default float64
	Step    float64
}

// Fractalevaluates a single complex point and returns the escape data.
type Fractal interface {
	// Key returns a unique identifier for the fractal type.
	Key() string
	// Name returns a human-readable name.
	Name() string
	// Description returns a short text shown in the UI.
	Description() string
	// Params returns the list of adjustable parameters with defaults.
	Params() []Param
	// SetParam sets a param by key and reports whether it existed.
	SetParam(key string, value float64) bool
	// GetParam returns the current value of a param.
	GetParam(key string) float64
	// Escape computes the iteration count until the point escapes,
	// plus the fractional (smooth-coloring) part.
	// maxIter sets the bailout iteration bound.
	Escape(px, py float64, maxIter int) (n float64, trapped bool)
	// Clone returns a deep copy of the fractal (with its params).
	Clone() Fractal
}

// ParamValue is a key/value pair used for serialization.
type ParamValue struct {
	Key   string  `json:"key"`
	Value float64 `json:"value"`
}

// Smooth returns the continuous iteration value for coloring:
// n + 1 - log2(log2(|z|)) / log2(bailout) when |z| > bailout.
func Smooth(px, py, bailout float64, n int) float64 {
	x2py2 := px*px + py*py
	mod := math.Log(x2py2)
	p := math.Log2(mod / math.Log(bailout))
	if v := float64(n) + 1.0 - p; v > 0 {
		return v
	}
	return float64(n)
}
