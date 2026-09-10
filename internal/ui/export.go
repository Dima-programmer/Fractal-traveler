package ui

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"fractal-gen/internal/config"
	"fractal-gen/internal/renderer"
)

// exportSize returns the pixel size for the chosen export format.
func exportSize(res string) (int, int) {
	switch res {
	case "2K":
		return 2560, 1440
	case "4K":
		return 3840, 2160
	default:
		return 1920, 1080
	}
}

// exportPNG renders the current scene to a PNG file at the chosen format size.
func (g *Game) exportPNG() {
	w, h := exportSize(g.expRes)

	path := "export.png"
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		os.MkdirAll(dir, 0o755)
	}

	v := g.view
	v.Aspect = float64(w) / float64(h)
	r := renderer.NewRenderer(g.palettes[g.palName], g.frac.Clone())
	r.MaxIter = g.maxIter
	img := image.NewRGBA(image.Rect(0, 0, w, h))

	g.exportWorking = true
	go func() {
		r.RenderTo(img, v)
		f, err := os.Create(path)
		if err == nil {
			err = png.Encode(f, img)
			f.Close()
		}
		g.exportWorking = false
		var msg string
		if err != nil {
			msg = "Ошибка экспорта: " + err.Error()
		} else {
			msg = fmt.Sprintf("Экспорт готов: %s (%s %dx%d)", path, g.expRes, w, h)
		}
		g.exportResult <- msg
	}()
}

// saveScene writes the current scene to a .fractal file.
func (g *Game) saveScene() {
	f := config.ToFile(g.frac, g.palName, g.view, g.maxIter)
	if err := config.Save(g.pathField.Text, f); err != nil {
		g.switchAlert = "Ошибка: " + err.Error()
	} else {
		g.switchAlert = "Сохранено: " + g.pathField.Text
	}
	g.switchAlertTimer = 120
}

// loadScene loads a .fractal file and applies it.
func (g *Game) loadScene() {
	f, err := config.Load(g.pathField.Text)
	if err != nil {
		g.switchAlert = "Ошибка: " + err.Error()
		g.switchAlertTimer = 120
		return
	}
	if !g.reg.IsRegistered(f.Fractal) {
		g.switchAlert = "Неизвестный тип: " + f.Fractal
		g.switchAlertTimer = 120
		return
	}
	if f.MaxIter > 0 {
		g.maxIter = f.MaxIter
		g.iterSlider.Value = float64(g.maxIter)
	}
	if f.Palette != "" && g.palettes[f.Palette] != nil {
		g.palName = f.Palette
	}
	g.frac = g.reg.Create(f.Fractal)
	config.Apply(f, g.frac, &g.view)
	g.iterSlider.Value = float64(g.maxIter)
	g.sbScroll = 0
	g.rebuildDependentWidgets()
	for _, b := range g.palBtns {
		b.Active = b.Label == g.palName
	}
	g.switchAlert = "Загружено: " + g.pathField.Text
	g.switchAlertTimer = 120
	g.invalidate()
}
