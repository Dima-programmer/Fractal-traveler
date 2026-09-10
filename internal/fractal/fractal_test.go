package fractal

import (
	"testing"
)

func TestEscapeMandelbrot(t *testing.T) {
	m := NewMandelbrot()
	// Origin is inside the set
	n, trapped := m.Escape(0, 0, 700)
	if trapped {
		t.Errorf("origin should be trapped, got escaped at %v", n)
	}
	// A point far outside escapes quickly
	n, trapped = m.Escape(2.0, 0.0, 700)
	if !trapped {
		t.Errorf("point (2,0) should escape")
	}
	if !(n > 0) {
		t.Errorf("escape count should be positive, got %v", n)
	}
}

func TestEscapeJulia(t *testing.T) {
	j := NewJulia()
	// Far point must escape quickly.
	_, trapped := j.Escape(2.0, 2.0, 700)
	if !trapped {
		t.Errorf("julia (2,2) should escape")
	}
}

func TestEscapeAll(t *testing.T) {
	reg := NewRegistry()
	types := []string{"mandelbrot", "julia", "burning_ship", "newton", "tricorn", "phoenix"}
	for _, key := range types {
		f := reg.Create(key)
		if f == nil {
			t.Fatalf("fractal %q not created", key)
		}
		if f.Key() != key {
			t.Errorf("key mismatch: %q != %q", f.Key(), key)
		}
		// Every fractal must handle points without panicking
		for _, pt := range [][2]float64{{-1.5, -1.5}, {0, 0}, {0.5, 0.3}, {1.2, -0.7}} {
			_, _ = f.Escape(pt[0], pt[1], 256)
		}
		cl := f.Clone()
		if cl.Key() != f.Key() {
			t.Errorf("clone key mismatch")
		}
	}
}

func TestPalettes(t *testing.T) {
	p := DefaultPalettes()
	if len(p) < 4 {
		t.Errorf("expected at least 4 palettes, got %d", len(p))
	}
	for name, pal := range p {
		// Interior must be black
		c := pal.At(0, true)
		if c.A != 255 {
			t.Errorf("%s: interior must be opaque", name)
		}
		// Exterior samples must be valid colors
		for _, t := range []float64{0.0, 0.5, 1.0, 1.5} {
			_ = pal.At(t, false)
		}
	}
}
