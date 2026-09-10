package config

import (
	"encoding/json"
	"os"

	"fractal-gen/internal/fractal"
	"fractal-gen/internal/renderer"
)

// File bundles everything needed to reproduce a fractal scene.
type File struct {
	Version int                  `json:"version"`
	Fractal string               `json:"fractal_type"`
	Params  []fractal.ParamValue `json:"params"`
	Palette string               `json:"palette"`
	CenterX float64              `json:"center_x"`
	CenterY float64              `json:"center_y"`
	HalfW   float64              `json:"half_width"`
	MaxIter int                  `json:"max_iter"`
}

// Load reads a fractal scene file.
func Load(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	f := &File{}
	if err := json.Unmarshal(data, f); err != nil {
		return nil, err
	}
	return f, nil
}

// Save writes a fractal scene file.
func Save(path string, f *File) error {
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// ToFile serializes the live state into a File.
func ToFile(frac fractal.Fractal, palName string, view renderer.View, maxIter int) *File {
	f := &File{
		Version: 1,
		Fractal: frac.Key(),
		Palette: palName,
		CenterX: view.CenterX,
		CenterY: view.CenterY,
		HalfW:   view.HalfWidth,
		MaxIter: maxIter,
	}
	for _, p := range frac.Params() {
		f.Params = append(f.Params, fractal.ParamValue{Key: p.Key, Value: frac.GetParam(p.Key)})
	}
	return f
}

// Apply mutates frac (in place) and view from a File.
func Apply(f *File, frac fractal.Fractal, view *renderer.View) error {
	for _, p := range f.Params {
		frac.SetParam(p.Key, p.Value)
	}
	view.CenterX = f.CenterX
	view.CenterY = f.CenterY
	view.HalfWidth = f.HalfW
	return nil
}
