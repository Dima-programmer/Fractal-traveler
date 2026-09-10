<div align="center">

# FractalForge — Fractal Generator & Explorer

[🇷🇺 Русский](README.ru.md) · [English 🇬🇧](README.md)

</div>

![Mandelbrot set](docs/mandelbrot.png)

Interactive fractal generator and explorer built with **Go** and **[Ebitengine](https://ebitengine.org/)**.

Explore the classic fractals, travel across the complex plane in real time, and
dive to arbitrary depths with high-precision rendering.

## Features

- **6 built-in fractals:** Mandelbrot set, Julia set, Burning Ship, Newton, Tricorn (Mandelbar), Phoenix
- **Adjustable parameters** for every fractal — degree, constant values, palette
- **Smooth real-time navigation:** pan with the mouse, zoom with the wheel
- **Continuous background rendering** — the view is refreshed while you move,
  not only when you stop
- **Auto-zoom mode** you can toggle with hotkeys
- **Fullscreen mode** (F11) — the canvas fills the whole screen, UI panel is hidden
- **Deep zoom to arbitrary precision** — automatic switch to high-precision
  (big.Float) rendering when the zoom level exceeds floating-point limits
- **Multiple color palettes,** fully customizable via an in-app palette editor
- **Export to PNG** — Full HD, 2K and 4K presets
- **Save & load scenes** (JSON) to restore your favorite viewpoints

## Screenshots

| Interface | Close-up |
|-----------|----------|
| ![Interface](docs/window.png) | ![Close-up](docs/mandelbrot.png) |

## Controls

| Action | Keys / Mouse |
|--------|--------------|
| Pan | Left mouse button |
| Zoom in / out | Mouse wheel |
| Toggle auto-zoom in | Ctrl + Left mouse button |
| Toggle auto-zoom out | Ctrl + Shift + Left mouse button |
| Pick a parameter point / separate action | Right mouse button |
| Force full re-render | R |
| Fullscreen toggle | F11 |

## Building

Requires **Go 1.26+**.

```sh
go build -o fractalforge .
```

## Running

```sh
go run .
```

## Pre-built binaries

No build required — download **fractalforge.exe** from the [Releases](https://github.com/Dima-programmer/Fractal-traveler/releases) page and run it directly.

## Project layout

```
internal/
  fractal/    fractal definitions and registry
  renderer/   pixel rendering (float and high-precision paths)
  ui/         Ebitengine UI: canvas, sidebar, widgets, export
presets/      bundled scene presets
```

## License

[MIT](LICENSE)