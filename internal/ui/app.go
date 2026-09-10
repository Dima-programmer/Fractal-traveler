package ui

import (
	"fmt"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"fractal-gen/internal/fractal"
	"fractal-gen/internal/renderer"
)

const (
	winW     = 1280
	winH     = 800
	sidebarW = 320

	// Sidebar layout
	sbContentX = winW - sidebarW + 20
	sbWidth    = sidebarW - 40
	btnW       = (sbWidth - 10) / 2
	btnH       = 26
	btnStep    = 32
	sliderStep = 28
	sbTitleY   = 10
)

// section is a sidebar group header with its vertical position.
type section struct {
	label string
	y     int
}

// Game is the main Ebiten application.
type Game struct {
	reg      *fractal.Registry
	frac     fractal.Fractal
	palettes map[string]fractal.Palette
	palNames []string
	palName  string
	view     renderer.View
	maxIter  int
	canvas   *canvas

	// input state
	dragging       bool
	lastMX, lastMY int
	zoomActive     bool
	zoomOut        bool
	zoomPX, zoomPY float64

	// fullscreen
	fullscreen bool
	fsW, fsH   int

	// sidebar widgets
	typeBtns             []*Button
	palBtns              []*Button
	paramSlid            []*Slider
	iterSlider           *Slider
	zoomSpeedSlider      *Slider
	autoCheck            *Checkbox
	fullBtn              *Button
	expBtns              []*Button
	expRes               string
	exportBtn            *Button
	pathField            *TextField
	saveBtn, loadBtn     *Button
	sections             []section
	statusText           string
	switchAlert          string
	switchAlertTimer     int
	exportWorking        bool
	exportResult         chan string

	// sidebar scroll
	sbScroll int // current scroll offset (0..sbMax)
	sbMax    int // maximum scroll offset
}

func (g *Game) Layout(w, h int) (int, int) {
	if g.fullscreen {
		if w != g.fsW || h != g.fsH {
			g.fsW, g.fsH = w, h
			g.canvas.resize(w, h)
			g.view.Aspect = float64(w) / float64(h)
			g.invalidate()
		}
		return w, h
	}
	if g.fsW != 0 || g.fsH != 0 {
		// coming back from fullscreen: restore windowed canvas size
		g.fsW, g.fsH = 0, 0
		g.canvas.resize(winW-sidebarW, winH)
		g.view.Aspect = float64(winW-sidebarW) / float64(winH)
		g.invalidate()
	}
	return winW, winH
}

// viewW/H return the current canvas size (whole screen when fullscreen,
// otherwise viewport area without the sidebar).
func (g *Game) viewW() int {
	if g.fullscreen {
		return g.fsW
	}
	return winW - sidebarW
}

func (g *Game) viewH() int {
	if g.fullscreen {
		return g.fsH
	}
	return winH
}

// NewGame builds the application state.
func NewGame() *Game {
	initFonts()
	g := &Game{
		reg:      fractal.NewRegistry(),
		palettes: fractal.DefaultPalettes(),
		maxIter:  700,
	}
	g.palNames = make([]string, 0, len(g.palettes))
	for name := range g.palettes {
		g.palNames = append(g.palNames, name)
	}
	sort.Strings(g.palNames)
	g.palName = "Bloom"
	g.frac = g.reg.Create("mandelbrot")

	g.view = renderer.View{
		CenterX: -0.55, CenterY: 0.0,
		HalfWidth: 1.7,
		Aspect:    float64(winW-sidebarW) / float64(winH),
	}

	g.canvas = newCanvas(winW-sidebarW, winH)
	g.exportResult = make(chan string, 1)

	g.buildWidgets()
	g.statusText = "Готово"
	return g
}

func (g *Game) buildWidgets() {
	g.typeBtns = nil
	g.palBtns = nil
	g.paramSlid = nil

	// Fractal type buttons, 2 per row.
	keys := g.reg.Keys()
	sort.Strings(keys)
	for _, k := range keys {
		f := g.reg.Create(k)
		b := &Button{
			Label:  f.Name(),
			Active: g.frac.Key() == k,
		}
		kk := k
		b.OnClick = func() { g.switchType(kk) }
		g.typeBtns = append(g.typeBtns, b)
	}

	// Parameter sliders (positions computed in layoutSidebar).
	for _, p := range g.frac.Params() {
		s := &Slider{
			Label: p.Name,
			Min:   p.Min, Max: p.Max, Step: p.Step,
			Value: g.frac.GetParam(p.Key),
		}
		kk := p.Key
		s.OnChanged = func(v float64) {
			g.frac.SetParam(kk, v)
			g.statusText = fmt.Sprintf("%s = %g", kk, v)
			g.invalidate()
		}
		g.paramSlid = append(g.paramSlid, s)
	}

	// Palette buttons, 2 per row.
	for _, name := range g.palNames {
		b := &Button{
			Label:  name,
			Active: name == g.palName,
		}
		nn := name
		b.OnClick = func() { g.switchPalette(nn) }
		g.palBtns = append(g.palBtns, b)
	}

	// Max iterations
	if g.iterSlider == nil {
		g.iterSlider = &Slider{
			Label: "Итерации",
			Min:   50, Max: 3000, Step: 50,
			Value:  float64(g.maxIter),
			Format: "%.0f",
		}
		g.iterSlider.OnChanged = func(v float64) {
			g.maxIter = int(math.Round(v))
			g.statusText = fmt.Sprintf("maxIter = %d", g.maxIter)
			g.invalidate()
		}
	}
	g.iterSlider.Value = float64(g.maxIter)
	g.iterSlider.Min = 50
	g.iterSlider.Max = 3000
	g.iterSlider.Step = 50

	// Auto background render while the view changes
	if g.autoCheck == nil {
		g.autoCheck = &Checkbox{Label: "Фоновый рендер"}
		g.autoCheck.OnToggle = func(on bool) {
			g.canvas.setAutoFull(on)
			g.statusText = fmt.Sprintf("Фоновый рендер: %v", on)
		}
	}
	g.autoCheck.Checked = true

	// Smooth zoom speed (Ctrl+ЛКМ), percent per second
	if g.zoomSpeedSlider == nil {
		g.zoomSpeedSlider = &Slider{
			Label: "Скорость зума",
			Min:   5, Max: 100, Step: 5,
			Value:  40,
			Format: "%.0f%%/с",
		}
	}
	g.zoomSpeedSlider.Value = 40

	// Render button
	if g.fullBtn == nil {
		g.fullBtn = &Button{Label: "Полный рендер"}
		g.fullBtn.OnClick = func() { g.canvas.requestFull(); g.statusText = "Полный рендер…" }
	}

	// Export section
	if len(g.expBtns) == 0 {
		g.expRes = "FullHD"
		formats := []struct {
			label string
			w, h  int
		}{
			{"FullHD", 1920, 1080},
			{"2K", 2560, 1440},
			{"4K", 3840, 2160},
		}
		for i, f := range formats {
			b := &Button{
				Label:  f.label,
				Active: g.expRes == f.label,
			}
			label := f.label
			b.OnClick = func() {
				g.expRes = label
				for _, eb := range g.expBtns {
					eb.Active = eb.Label == g.expRes
				}
			}
			_ = i
			g.expBtns = append(g.expBtns, b)
		}
		g.exportBtn = &Button{Label: "Экспорт PNG"}
		g.exportBtn.OnClick = g.exportPNG
	}

	// Save / load
	if g.pathField == nil {
		g.pathField = &TextField{Label: "Файл"}
		g.pathField.Text = "presets/mandelbrot.fractal"
		g.saveBtn = &Button{Label: "Сохранить"}
		g.saveBtn.OnClick = g.saveScene
		g.loadBtn = &Button{Label: "Загрузить"}
		g.loadBtn.OnClick = g.loadScene
	}

	g.layoutSidebar()
}

// layoutSidebar positions every widget and builds section headers.
// Geometry note: sliders and text fields draw their label ~20px above the
// widget, so sections starting with a slider/field reserve 36px after the
// header line; button sections only need 18px.
func (g *Game) layoutSidebar() {
	g.sections = nil
	sx, sw := sbContentX, sbWidth

	y := 36 // first section header

	// — Фрактал —
	g.sections = append(g.sections, section{"Тип фрактала", y})
	y += 18
	for i, b := range g.typeBtns {
		b.X = sx + (i%2)*(btnW+10)
		b.Y = y + (i/2)*btnStep
		b.W, b.H = btnW, btnH
	}
	y += ((len(g.typeBtns)+1)/2)*btnStep + 6

	// — Параметры —
	g.sections = append(g.sections, section{"Параметры", y})
	y += 36
	for i, s := range g.paramSlid {
		s.X, s.Y, s.W, s.H = sx, y+i*sliderStep, sw, 6
	}
	y += len(g.paramSlid) * sliderStep
	if g.iterSlider != nil {
		g.iterSlider.X, g.iterSlider.Y, g.iterSlider.W, g.iterSlider.H = sx, y+10, sw, 6
		y += sliderStep + 10
	}
	y += 4

	// — Палитра —
	g.sections = append(g.sections, section{"Палитра", y})
	y += 18
	for i, b := range g.palBtns {
		b.X = sx + (i%2)*(btnW+10)
		b.Y = y + (i/2)*btnStep
		b.W, b.H = btnW, btnH
	}
	y += ((len(g.palBtns)+1)/2)*btnStep + 6

	// — Отрисовка —
	g.sections = append(g.sections, section{"Отрисовка", y})
	y += 36
	if g.zoomSpeedSlider != nil {
		g.zoomSpeedSlider.X, g.zoomSpeedSlider.Y, g.zoomSpeedSlider.W, g.zoomSpeedSlider.H = sx, y, sw, 6
		y += sliderStep
	}
	if g.autoCheck != nil {
		g.autoCheck.X, g.autoCheck.Y, g.autoCheck.W, g.autoCheck.H = sx, y, sw, 20
		y += 24
	}
	if g.fullBtn != nil {
		g.fullBtn.X, g.fullBtn.Y = sx, y
		g.fullBtn.W, g.fullBtn.H = sw, btnH
		y += btnStep + 6
	}

	// — Экспорт —
	g.sections = append(g.sections, section{"Экспорт PNG", y})
	y += 36
	if len(g.expBtns) > 0 {
		btnW3 := (sw - 2*8) / 3
		for i, b := range g.expBtns {
			b.X = sx + i*(btnW3+8)
			b.Y = y
			b.W, b.H = btnW3, btnH
		}
		y += btnStep + 4
		g.exportBtn.X, g.exportBtn.Y = sx, y
		g.exportBtn.W, g.exportBtn.H = sw, btnH
		y += btnStep + 6
	}

	// — Файл —
	g.sections = append(g.sections, section{"Сохранение / загрузка", y})
	y += 36
	if g.pathField != nil {
		g.pathField.X, g.pathField.Y = sx, y
		g.pathField.W, g.pathField.H = sw, 24
		y += 30
		g.saveBtn.X, g.saveBtn.Y = sx, y
		g.saveBtn.W, g.saveBtn.H = btnW, btnH
		g.loadBtn.X, g.loadBtn.Y = sx+btnW+10, y
		g.loadBtn.W, g.loadBtn.H = btnW, btnH
		y += btnStep + 8
	}

	// Bottom-most widget end; anything below the visible area becomes scrollable.
	if g.sbScroll > y {
		g.sbScroll = y
	}
	g.sbMax = y - (winH - 60)
	if g.sbMax < 0 {
		g.sbMax = 0
	}
	if g.sbScroll > g.sbMax {
		g.sbScroll = g.sbMax
	}
}

func (g *Game) switchType(key string) {
	if key == g.frac.Key() {
		return
	}
	g.frac = g.reg.Create(key)
	switch key {
	case "julia":
		g.view.CenterX = 0
		g.view.CenterY = 0
		g.view.HalfWidth = 1.5
	default:
		if g.view.HalfWidth > 3.0 || g.view.HalfWidth < 0.02 {
			g.view.HalfWidth = 1.7
		}
	}
	g.rebuildDependentWidgets()
	g.sbScroll = 0
	g.statusText = g.frac.Name()
	g.invalidate()
}

func (g *Game) switchPalette(name string) {
	g.palName = name
	for _, b := range g.palBtns {
		b.Active = b.Label == name
	}
	g.statusText = "Палитра: " + name
	g.invalidate()
}

func (g *Game) rebuildDependentWidgets() {
	for _, b := range g.typeBtns {
		b.Active = b.Label == g.frac.Name()
	}
	g.paramSlid = nil
	for _, p := range g.frac.Params() {
		s := &Slider{
			Label: p.Name,
			Min:   p.Min, Max: p.Max, Step: p.Step,
			Value: g.frac.GetParam(p.Key),
		}
		kk := p.Key
		s.OnChanged = func(v float64) {
			g.frac.SetParam(kk, v)
			g.statusText = fmt.Sprintf("%s = %g", kk, v)
			g.invalidate()
		}
		g.paramSlid = append(g.paramSlid, s)
	}
	g.layoutSidebar()
}

// toggleFullscreen switches between windowed and fullscreen layout. In
// fullscreen the canvas covers the entire screen and the sidebar is hidden.
func (g *Game) toggleFullscreen() {
	g.fullscreen = !g.fullscreen
	if g.fullscreen {
		ebiten.SetFullscreen(true)
	} else {
		ebiten.SetFullscreen(false)
	}
}

// invalidate marks the canvas for re-render.
func (g *Game) invalidate() {
	g.canvas.invalidate()
}

// Update handles input and rendering updates.
func (g *Game) Update() error {
	captureMouse()
	pressed := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	justPressed := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	overSidebar := !g.fullscreen && mouseX >= winW-sidebarW

	// Sidebar interaction (skip while dragging the viewport)
	if !g.dragging && overSidebar {
		widgetYOff = g.sbScroll
		g.updateSidebarWidgets(pressed, justPressed)
		widgetYOff = 0
	}

	// Viewport input (pan / zoom / julia param)
	g.handleViewport()

	// Keyboard shortcuts
	if !g.pathField.Focused {
		if inpututil.IsKeyJustPressed(ebiten.KeyEqual) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadAdd) {
			g.maxIter = int(math.Min(3000, float64(g.maxIter)+100))
			g.iterSlider.Value = float64(g.maxIter)
			g.invalidate()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyMinus) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadSubtract) {
			g.maxIter = int(math.Max(50, float64(g.maxIter)-100))
			g.iterSlider.Value = float64(g.maxIter)
			g.invalidate()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyR) {
			g.canvas.requestFull()
		}
	}

	// Fullscreen toggle (works even while a text field is focused)
	if inpututil.IsKeyJustPressed(ebiten.KeyF11) {
		g.toggleFullscreen()
	}

	if g.switchAlertTimer > 0 {
		g.switchAlertTimer--
		if g.switchAlertTimer == 0 {
			g.switchAlert = ""
		}
	}

	select {
	case msg := <-g.exportResult:
		g.switchAlert = msg
		g.switchAlertTimer = 240
	default:
	}

	g.canvas.update(g.view, g.frac, g.palettes[g.palName], g.maxIter)
	return nil
}

// updateSidebarWidgets updates all sidebar widgets (called with widgetYOff set).
func (g *Game) updateSidebarWidgets(pressed, justPressed bool) {
	for _, b := range g.typeBtns {
		b.Update(mouseX, mouseY, pressed, justPressed)
	}
	for _, b := range g.palBtns {
		b.Update(mouseX, mouseY, pressed, justPressed)
	}
	g.exportBtn.Update(mouseX, mouseY, pressed, justPressed)
	for _, b := range g.expBtns {
		b.Update(mouseX, mouseY, pressed, justPressed)
	}
	g.saveBtn.Update(mouseX, mouseY, pressed, justPressed)
	g.loadBtn.Update(mouseX, mouseY, pressed, justPressed)
	g.fullBtn.Update(mouseX, mouseY, pressed, justPressed)
	g.autoCheck.Update(mouseX, mouseY, justPressed)
	g.pathField.Update(mouseX, mouseY, pressed)
	g.iterSlider.Update(mouseX, mouseY, pressed)
	g.zoomSpeedSlider.Update(mouseX, mouseY, pressed)
	for _, s := range g.paramSlid {
		s.Update(mouseX, mouseY, pressed)
	}
}

// handleViewport processes mouse input over the fractal area.
func (g *Game) handleViewport() {
	mx, my := mouseX, mouseY
	vw, vh := g.viewW(), g.viewH()
	inViewport := g.fullscreen || mx < winW-sidebarW
	ctrl := ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyMeta)
	shift := ebiten.IsKeyPressed(ebiten.KeyShiftLeft) || ebiten.IsKeyPressed(ebiten.KeyShiftRight)

	// Auto-zoom toggle: first Ctrl+ЛКМ click starts a continuous zoom INTO
	// the clicked point; Ctrl+Shift+ЛКМ similarly but zooming OUT. A second
	// click with the same modifier stops it. Speed via the slider.
	zoomClick := inViewport && ctrl && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	if zoomClick {
		switch {
		case g.zoomActive && g.zoomOut == shift:
			g.zoomActive = false
			g.statusText = "авто-зум остановлен"
		case !g.zoomActive:
			g.zoomActive = true
			g.zoomOut = shift
			g.zoomPX = float64(mx) / float64(vw)
			g.zoomPY = float64(my) / float64(vh)
			if shift {
				g.statusText = fmt.Sprintf("авто-отдаление %.3e", g.view.HalfWidth)
			} else {
				g.statusText = fmt.Sprintf("авто-зум %.3e", g.view.HalfWidth)
			}
		default:
			// direction changed: re-anchor, keep zooming
			g.zoomOut = shift
			g.zoomPX = float64(mx) / float64(vw)
			g.zoomPY = float64(my) / float64(vh)
			if shift {
				g.statusText = fmt.Sprintf("авто-отдаление %.3e", g.view.HalfWidth)
			} else {
				g.statusText = fmt.Sprintf("авто-зум %.3e", g.view.HalfWidth)
			}
		}
		g.invalidate()
	}

	tps := float64(ebiten.DefaultTPS)
	if tps < 1 {
		tps = 1
	}

	// Continuous zoom while active (in or out).
	if g.zoomActive {
		var factor float64
		if g.zoomOut {
			factor = math.Pow(1+g.zoomSpeedSlider.Value/100.0, 1.0/tps)
		} else {
			factor = math.Pow(1/(1+g.zoomSpeedSlider.Value/100.0), 1.0/tps)
		}
		g.view = g.view.ZoomAt(factor, g.zoomPX, g.zoomPY)
		g.invalidate()
		g.statusText = fmt.Sprintf("%s %.3e", map[bool]string{true: "авто-отдаление", false: "авто-зум"}[g.zoomOut], g.view.HalfWidth)
	}

	// Zoom (mouse wheel)
	if inViewport && !ctrl && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		if _, yoff := ebiten.Wheel(); yoff != 0 {
			px := float64(mx) / float64(vw)
			py := float64(my) / float64(vh)
			factor := math.Pow(0.85, yoff)
			g.view = g.view.ZoomAt(factor, px, py)
			g.invalidate()
			g.statusText = fmt.Sprintf("zoom %.3e", g.view.HalfWidth)
		}
	} else if !zoomClick && !g.zoomActive && !g.fullscreen {
		// Scroll the sidebar with the mouse wheel.
		if _, yoff := ebiten.Wheel(); yoff != 0 && g.sbMax > 0 {
			g.sbScroll -= int(yoff * 40)
			if g.sbScroll < 0 {
				g.sbScroll = 0
			}
			if g.sbScroll > g.sbMax {
				g.sbScroll = g.sbMax
			}
		}
	}

	// Pan dragging (disabled while Ctrl+ЛКМ zooming)
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && !ctrl && !g.zoomActive {
		if inViewport {
			if !g.dragging {
				g.dragging = true
				g.lastMX, g.lastMY = mx, my
			} else {
				dx := mx - g.lastMX
				dy := my - g.lastMY
				if dx != 0 || dy != 0 {
					g.view = g.view.Pan(float64(dx), float64(dy), float64(vw), float64(vh))
					g.lastMX, g.lastMY = mx, my
					g.invalidate()
					g.statusText = fmt.Sprintf("(%g, %g)  halfw %g", g.view.CenterX, g.view.CenterY, g.view.HalfWidth)
				}
			}
		}
	} else {
		g.dragging = false
	}

	// Right click: set Julia parameter from cursor position
	if inViewport && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		if g.frac.Key() == "julia" {
			px, py := g.view.PixelToComplex(float64(mx), float64(my), float64(vw), float64(vh))
			g.frac.SetParam("c_real", px)
			g.frac.SetParam("c_imag", py)
			g.rebuildDependentWidgets()
			g.statusText = fmt.Sprintf("c = %g + %gi", px, py)
			g.invalidate()
		}
	}
}

// Draw renders one frame.
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(colBg)
	g.canvas.draw(screen, g.view)
	if !g.fullscreen {
		g.drawSidebar(screen)
	}

	// overlay status in viewport corner
	vh := g.viewH()
	drawText(screen, "ЛКМ панорама · Ctrl+ЛКМ зум · Ctrl+Shift+ЛКМ отдаление · колесо зум · ПКМ параметр · R рендер · F11 fullscreen", 16, float64(vh)-28, 12, colTextDim)
	if g.switchAlert != "" {
		fillRect(screen, 16, 12, int(textWidth(g.switchAlert, 14))+24, 30, colPanelDark)
		drawText(screen, g.switchAlert, 28, 18, 14, colText)
	}
}

func (g *Game) drawSidebar(dst *ebiten.Image) {
	sx := winW - sidebarW
	fillRect(dst, sx, 0, sidebarW, winH, colPanel)
	fillRect(dst, sx, 0, 1, winH, colBorder)

	cx := sbContentX

	drawText(dst, "FractalForge", float64(cx), sbTitleY, 18, colText)

	widgetYOff = g.sbScroll
	for _, sec := range g.sections {
		drawText(dst, sec.label, float64(cx), float64(sec.y-widgetYOff), 12, colAccent)
	}

	for _, b := range g.typeBtns {
		b.Draw(dst)
	}
	for _, s := range g.paramSlid {
		s.Draw(dst)
	}
	g.iterSlider.Draw(dst)
	for _, b := range g.palBtns {
		b.Draw(dst)
	}
	g.zoomSpeedSlider.Draw(dst)
	g.autoCheck.Draw(dst)
	g.fullBtn.Draw(dst)
	for _, b := range g.expBtns {
		b.Draw(dst)
	}
	g.exportBtn.Draw(dst)
	g.pathField.Draw(dst)
	g.saveBtn.Draw(dst)
	g.loadBtn.Draw(dst)
	widgetYOff = 0

	// status line at the bottom of the sidebar
	drawText(dst, clampString(g.statusText, sbWidth, 12), float64(cx), winH-20, 12, colTextDim)
}
