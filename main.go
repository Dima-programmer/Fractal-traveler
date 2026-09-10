package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"fractal-gen/internal/ui"
)

func main() {
	ebiten.SetWindowSize(1280, 800)
	ebiten.SetWindowTitle("FractalForge — Генератор фракталов")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeDisabled)

	g := ui.NewGame()
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
