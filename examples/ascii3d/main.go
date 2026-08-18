// ascii3d demonstrates real-time 3D ASCII & sub-cell rendering in Limoni TUI.
// It supports multiple rendering modes (ASCII typography, Half-Block 2x, Low-Poly, Braille 8x, Wireframe),
// GLB/GLTF/OBJ/STL/PLY model loading, Blinn-Phong specular highlights (#066aff),
// tone mapping, edge boosting, floating physics, and interactive camera controls.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/graphics"
	"github.com/thebanri/limoni/layout"
	"github.com/thebanri/limoni/widgets"
)

type PaletteOption struct {
	Name      string
	Highlight cell.Color
	BaseColor cell.Color
}

type AppState struct {
	// 3D Model
	ActiveModelName string
	ActiveModel     graphics.Model3D
	CustomFilePath  string

	// Transformation & Animation
	Scale             float64
	XOffset, YOffset  float64
	FloatIntensity    float64
	FloatSpeed        float64
	RotationIntensity float64
	AutoRotate        bool
	AutoRotateSpeed   float64
	RotX, RotY, RotZ  float64
	StartTime         time.Time

	// Camera
	FOV            float64
	CameraDistance float64
	CellAspect     float64

	// Shading & Optics
	Contrast             float64
	EdgeContrast         float64
	Exposure             float64
	EnvironmentIntensity float64
	Roughness            float64

	// Appearance
	RenderMode   int // 0: ASCII, 1: HalfBlock 2x, 2: Low-Poly, 3: Braille 8x, 4: Wireframe
	Ascii        bool
	Colored      bool
	Invert       bool
	ActiveRamp   string
	SelectedRamp int
	PaletteIndex int
	ShowHUD      bool

	// UI Controls States
	ModeSelectState    *widgets.SelectState
	ModelSelectState   *widgets.SelectState
	RampSelectState    *widgets.SelectState
	PaletteSelectState *widgets.SelectState

	ScaleSliderState      *widgets.SliderState
	ContrastSliderState   *widgets.SliderState
	EdgeContrastSlider    *widgets.SliderState
	ExposureSliderState   *widgets.SliderState
	RoughnessSliderState  *widgets.SliderState
	FloatIntSliderState   *widgets.SliderState
	FloatSpeedSliderState *widgets.SliderState
	DistanceSliderState   *widgets.SliderState
	FOVSliderState        *widgets.SliderState
	CellAspectSliderState *widgets.SliderState

	// Mouse Drag
	DragActive bool
	LastDragX  int
	LastDragY  int

	// Performance
	FPS       float64
	LastFrame time.Time

	// File browser modal
	ShowFileModal bool
	ModelFiles    []string
	FileListState *widgets.ListState
}

var renderModes = []struct {
	Name string
	Mode widgets.Ascii3DMode
}{
	{"ASCII Typography", widgets.ModeASCII},
	{"Half-Block 2x (Micro)", widgets.ModeBlock},
	{"Dithered (Retro 4x4)", widgets.ModeDithered},
	{"Braille 8x (Dots)", widgets.ModeBraille},
}

var ramps = []struct {
	Name string
	Ramp string
}{
	{"CanvasUI Exact Master Ramp", widgets.RampCanvasUI},
	{"Ultra-Dense Typography", widgets.RampTypography},
	{"Duck Classic Ramp", widgets.RampDuck},
	{"Standard ASCII (.:-=+*#%@)", widgets.RampStandard},
	{"Shading Blocks ( ░▒▓█)", widgets.RampBlocks},
	{"Binary Matrix ( 01)", widgets.RampBinary},
	{"Minimal (.:+*#)", widgets.RampMinimal},
}

var palettes = []PaletteOption{
	{Name: "Electric Blue Sheen (#066aff)", Highlight: cell.NewColorRGB(6, 106, 255), BaseColor: cell.NewColorRGB(255, 225, 20)},
	{Name: "Glossy White Sheen (#ffffff)", Highlight: cell.NewColorRGB(255, 255, 255), BaseColor: cell.NewColorRGB(255, 225, 20)},
	{Name: "Sunburst Gold (#ffe066)", Highlight: cell.NewColorRGB(255, 224, 102), BaseColor: cell.NewColorRGB(255, 225, 20)},
	{Name: "Cyberpunk Magenta (#ff0077)", Highlight: cell.NewColorRGB(255, 0, 119), BaseColor: cell.NewColorRGB(255, 225, 20)},
	{Name: "Neon Emerald (#00ff88)", Highlight: cell.NewColorRGB(0, 255, 136), BaseColor: cell.NewColorRGB(255, 225, 20)},
	{Name: "Amber Retro (#ffaa00)", Highlight: cell.NewColorRGB(255, 230, 100), BaseColor: cell.NewColorRGB(255, 170, 0)},
}

func scan3DFiles() []string {
	var files []string
	searchDirs := []string{"examples/demo", "examples/3d_viewer", "examples/ascii3d", "."}
	for _, dir := range searchDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			ext := strings.ToLower(filepath.Ext(e.Name()))
			if ext == ".glb" || ext == ".gltf" || ext == ".obj" || ext == ".stl" || ext == ".ply" {
				files = append(files, filepath.Join(dir, e.Name()))
			}
		}
	}
	return files
}

func getModelByName(name string) graphics.Model3D {
	switch name {
	case "Duck":
		return graphics.NewDuck()
	case "Torus":
		m := graphics.NewTorus(0.85, 0.35, 16, 16)
		m.Normalize(2.0)
		return m
	case "Sphere":
		m := graphics.NewSphere(1.0, 14, 16)
		m.Normalize(2.0)
		return m
	case "Pyramid":
		m := graphics.NewPyramid(1.8, 1.8)
		m.Normalize(2.0)
		return m
	case "Cube":
		m := graphics.NewCube(1.8)
		m.Normalize(2.0)
		return m
	default:
		return graphics.NewDuck()
	}
}

func main() {
	b := backend.NewBackend(os.Stdin, os.Stdout)
	if err := b.Setup(); err != nil {
		fmt.Fprintf(os.Stderr, "Backend setup error: %v\n", err)
		return
	}
	defer b.Close()

	t, err := terminal.New(b)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Terminal initialization error: %v\n", err)
		return
	}
	b.StartEventLoop()

	modeOptions := make([]string, len(renderModes))
	for i, m := range renderModes {
		modeOptions[i] = m.Name
	}
	modelOptions := []string{"Duck", "Torus", "Sphere", "Pyramid", "Cube", "Load File..."}
	rampOptions := make([]string, len(ramps))
	for i, r := range ramps {
		rampOptions[i] = r.Name
	}
	paletteOptions := make([]string, len(palettes))
	for i, p := range palettes {
		paletteOptions[i] = p.Name
	}

	state := &AppState{
		ActiveModelName:      "Duck",
		ActiveModel:          graphics.NewDuck(),
		Scale:                4.2,
		XOffset:              0.0,
		YOffset:              -0.1,
		FloatIntensity:       2.0,
		FloatSpeed:           2.0,
		RotationIntensity:    1.0,
		AutoRotate:           true,
		AutoRotateSpeed:      30.0,
		RotX:                 10.0,
		RotY:                 180.0,
		StartTime:            time.Now(),
		FOV:                  65.0,
		CameraDistance:       4.2,
		CellAspect:           0.50,
		Contrast:             1.2,
		EdgeContrast:         3.0,
		Exposure:             1.0,
		EnvironmentIntensity: 1.0,
		Roughness:            0.15,
		RenderMode:           0,
		Ascii:                true,
		Colored:              true,
		Invert:               false,
		ActiveRamp:           widgets.RampCanvasUI,
		SelectedRamp:         0,
		PaletteIndex:         0,
		ShowHUD:              true,

		ModeSelectState:    widgets.NewSelectState(),
		ModelSelectState:   widgets.NewSelectState(),
		RampSelectState:    widgets.NewSelectState(),
		PaletteSelectState: widgets.NewSelectState(),

		ScaleSliderState:      widgets.NewSliderState(42), // 4.2
		ContrastSliderState:   widgets.NewSliderState(12), // 1.2
		EdgeContrastSlider:    widgets.NewSliderState(30), // 3.0
		ExposureSliderState:   widgets.NewSliderState(10), // 1.0
		RoughnessSliderState:  widgets.NewSliderState(15), // 0.15
		FloatIntSliderState:   widgets.NewSliderState(20), // 2.0
		FloatSpeedSliderState: widgets.NewSliderState(20), // 2.0
		DistanceSliderState:   widgets.NewSliderState(42), // 4.2
		FOVSliderState:        widgets.NewSliderState(65), // 65 deg
		CellAspectSliderState: widgets.NewSliderState(50), // 0.50

		FileListState: widgets.NewListState(),
		LastFrame:     time.Now(),
	}

	ticker := time.NewTicker(16 * time.Millisecond) // ~60 FPS
	defer ticker.Stop()

	frameCount := 0
	lastFPSMeasure := time.Now()

	draw := func() {
		drawUI(t, state)
	}

	draw()

	for {
		select {
		case ev, ok := <-b.Events():
			if !ok {
				return
			}
			switch ev.Type {
			case backend.EventKey:
				if state.ShowFileModal {
					if ev.Key.Type == backend.KeyEsc {
						state.ShowFileModal = false
						draw()
						continue
					}
					if ev.Key.Type == backend.KeyArrowUp {
						if state.FileListState.Selected > 0 {
							state.FileListState.Selected--
						}
					}
					if ev.Key.Type == backend.KeyArrowDown {
						if state.FileListState.Selected < len(state.ModelFiles)-1 {
							state.FileListState.Selected++
						}
					}
					if ev.Key.Type == backend.KeyEnter {
						if len(state.ModelFiles) > 0 && state.FileListState.Selected >= 0 && state.FileListState.Selected < len(state.ModelFiles) {
							path := state.ModelFiles[state.FileListState.Selected]
							model, err := graphics.LoadModel(path)
							if err == nil {
								model.Normalize(2.0)
								state.ActiveModel = model
								state.ActiveModelName = filepath.Base(path)
								state.CustomFilePath = path
							}
						}
						state.ShowFileModal = false
						draw()
						continue
					}
					draw()
					continue
				}

				if ev.Key.Type == backend.KeyEsc || (ev.Key.Type == backend.KeyRune && ev.Key.Ch == 'q') {
					return
				}

				if ev.Key.Type == backend.KeyTab {
					if ev.Key.Shift {
						t.FocusManager().Prev()
					} else {
						t.FocusManager().Next()
					}
				}

				if ev.Key.Type == backend.KeyRune {
					switch ev.Key.Ch {
					case ' ':
						state.AutoRotate = !state.AutoRotate
					case 'm', 'M':
						state.RenderMode = (state.RenderMode + 1) % len(renderModes)
						state.ModeSelectState.Selected = state.RenderMode
					case 'f':
						fmt.Print("\x1b]50;#-1\x07")
					case 'F':
						fmt.Print("\x1b]50;#+1\x07")
					case 'a', 'A':
						state.Ascii = !state.Ascii
					case 'c', 'C':
						state.Colored = !state.Colored
					case 'i', 'I':
						state.Invert = !state.Invert
					case 'h', 'H':
						state.ShowHUD = !state.ShowHUD
					case 'r', 'R':
						state.RotX = 12.0
						state.RotY = -45.0
						state.RotZ = 0.0
					case 'o', 'O':
						state.ModelFiles = scan3DFiles()
						state.FileListState.Selected = 0
						state.ShowFileModal = true
						draw()
						continue
					case '+', '=':
						if state.ScaleSliderState.Value < 80 {
							state.ScaleSliderState.Value += 2
							syncStateFromSliders(state)
						}
					case '-', '_':
						if state.ScaleSliderState.Value > 8 {
							state.ScaleSliderState.Value -= 2
							syncStateFromSliders(state)
						}
					case '[', '{':
						if state.CellAspectSliderState.Value > 25 {
							state.CellAspectSliderState.Value -= 2
							syncStateFromSliders(state)
						}
					case ']', '}':
						if state.CellAspectSliderState.Value < 80 {
							state.CellAspectSliderState.Value += 2
							syncStateFromSliders(state)
						}
					}
				}

				focused := t.FocusManager().Focused()
				switch focused {
				case "mode_select":
					if state.ModeSelectState.HandleKey(ev.Key, len(renderModes)) {
						if state.ModeSelectState.Selected >= 0 && state.ModeSelectState.Selected < len(renderModes) {
							state.RenderMode = state.ModeSelectState.Selected
						}
					}
				case "model_select":
					if state.ModelSelectState.HandleKey(ev.Key, len(modelOptions)) {
						if state.ModelSelectState.Selected == 5 { // "Load File..."
							state.ModelFiles = scan3DFiles()
							state.FileListState.Selected = 0
							state.ShowFileModal = true
						} else if state.ModelSelectState.Selected >= 0 && state.ModelSelectState.Selected < 5 {
							name := modelOptions[state.ModelSelectState.Selected]
							state.ActiveModelName = name
							state.ActiveModel = getModelByName(name)
						}
					}
				case "ramp_select":
					if state.RampSelectState.HandleKey(ev.Key, len(ramps)) {
						if state.RampSelectState.Selected >= 0 && state.RampSelectState.Selected < len(ramps) {
							state.SelectedRamp = state.RampSelectState.Selected
							state.ActiveRamp = ramps[state.SelectedRamp].Ramp
						}
					}
				case "palette_select":
					if state.PaletteSelectState.HandleKey(ev.Key, len(palettes)) {
						if state.PaletteSelectState.Selected >= 0 && state.PaletteSelectState.Selected < len(palettes) {
							state.PaletteIndex = state.PaletteSelectState.Selected
						}
					}
				case "scale_slider":
					state.ScaleSliderState.HandleKey(ev.Key, 8, 80)
					state.Scale = float64(state.ScaleSliderState.Value) / 10.0
				case "aspect_slider":
					state.CellAspectSliderState.HandleKey(ev.Key, 25, 80)
					state.CellAspect = float64(state.CellAspectSliderState.Value) / 100.0
				case "contrast_slider":
					state.ContrastSliderState.HandleKey(ev.Key, 5, 30)
					state.Contrast = float64(state.ContrastSliderState.Value) / 10.0
				case "edge_slider":
					state.EdgeContrastSlider.HandleKey(ev.Key, 0, 50)
					state.EdgeContrast = float64(state.EdgeContrastSlider.Value) / 10.0
				case "exposure_slider":
					state.ExposureSliderState.HandleKey(ev.Key, 2, 30)
					state.Exposure = float64(state.ExposureSliderState.Value) / 10.0
				case "roughness_slider":
					state.RoughnessSliderState.HandleKey(ev.Key, 1, 100)
					state.Roughness = float64(state.RoughnessSliderState.Value) / 100.0
				case "float_int_slider":
					state.FloatIntSliderState.HandleKey(ev.Key, 0, 50)
					state.FloatIntensity = float64(state.FloatIntSliderState.Value) / 10.0
				case "float_spd_slider":
					state.FloatSpeedSliderState.HandleKey(ev.Key, 0, 50)
					state.FloatSpeed = float64(state.FloatSpeedSliderState.Value) / 10.0
				case "dist_slider":
					state.DistanceSliderState.HandleKey(ev.Key, 15, 90)
					state.CameraDistance = float64(state.DistanceSliderState.Value) / 10.0
				case "fov_slider":
					state.FOVSliderState.HandleKey(ev.Key, 30, 110)
					state.FOV = float64(state.FOVSliderState.Value)
				}

				// Arrow Keys for direct 3D rotation if not navigating sliders
				if focused == "" || strings.HasSuffix(focused, "_select") {
					switch ev.Key.Type {
					case backend.KeyArrowUp:
						state.RotX += 5.0
					case backend.KeyArrowDown:
						state.RotX -= 5.0
					case backend.KeyArrowLeft:
						state.RotY -= 5.0
					case backend.KeyArrowRight:
						state.RotY += 5.0
					}
				}

				draw()

			case backend.EventMouse:
				if state.ShowFileModal {
					t.RouteMouseEvent(ev.Mouse)
					draw()
					continue
				}

				handled := t.RouteMouseEvent(ev.Mouse)
				if !handled {
					if ev.Mouse.Button == backend.MouseLeft {
						if ev.Mouse.Drag {
							if state.DragActive {
								dx := int(ev.Mouse.X) - state.LastDragX
								dy := int(ev.Mouse.Y) - state.LastDragY
								state.RotY += float64(dx) * 1.6
								state.RotX += float64(dy) * 1.6
							}
							state.LastDragX = int(ev.Mouse.X)
							state.LastDragY = int(ev.Mouse.Y)
							state.DragActive = true
						} else {
							state.DragActive = false
						}
					} else if ev.Mouse.Button == backend.MouseNone {
						state.DragActive = false
					} else if ev.Mouse.Button == backend.MouseScrollUp {
						if state.ScaleSliderState.Value < 80 {
							state.ScaleSliderState.Value += 2
							state.Scale = float64(state.ScaleSliderState.Value) / 10.0
						}
					} else if ev.Mouse.Button == backend.MouseScrollDown {
						if state.ScaleSliderState.Value > 8 {
							state.ScaleSliderState.Value -= 2
							state.Scale = float64(state.ScaleSliderState.Value) / 10.0
						}
					}
				}
				draw()

			case backend.EventResize:
				draw()
			}

		case now := <-ticker.C:
			frameCount++
			if now.Sub(lastFPSMeasure) >= 500*time.Millisecond {
				state.FPS = float64(frameCount) / now.Sub(lastFPSMeasure).Seconds()
				frameCount = 0
				lastFPSMeasure = now
			}
			state.LastFrame = now
			draw()
		}
	}
}

func syncStateFromSliders(state *AppState) {
	if state == nil {
		return
	}
	if state.ScaleSliderState != nil {
		state.Scale = float64(state.ScaleSliderState.Value) / 10.0
	}
	if state.CellAspectSliderState != nil {
		state.CellAspect = float64(state.CellAspectSliderState.Value) / 100.0
	}
	if state.ContrastSliderState != nil {
		state.Contrast = float64(state.ContrastSliderState.Value) / 10.0
	}
	if state.EdgeContrastSlider != nil {
		state.EdgeContrast = float64(state.EdgeContrastSlider.Value) / 10.0
	}
	if state.ExposureSliderState != nil {
		state.Exposure = float64(state.ExposureSliderState.Value) / 10.0
	}
	if state.RoughnessSliderState != nil {
		state.Roughness = float64(state.RoughnessSliderState.Value) / 100.0
	}
	if state.FloatIntSliderState != nil {
		state.FloatIntensity = float64(state.FloatIntSliderState.Value) / 10.0
	}
	if state.FloatSpeedSliderState != nil {
		state.FloatSpeed = float64(state.FloatSpeedSliderState.Value) / 10.0
	}
	if state.DistanceSliderState != nil {
		state.CameraDistance = float64(state.DistanceSliderState.Value) / 10.0
	}
	if state.FOVSliderState != nil {
		state.FOV = float64(state.FOVSliderState.Value)
	}
}

func drawUI(t *terminal.Terminal, state *AppState) {
	syncStateFromSliders(state)
	t.Draw(func(f *terminal.Frame) {
		f.SetTheme(widgets.DarkTheme())
		area := f.Buffer.Area
		accentCol := palettes[state.PaletteIndex].Highlight

		// Modals
		if state.ShowFileModal {
			modalW, modalH := uint16(64), uint16(16)
			if modalW > area.Width-4 {
				modalW = area.Width - 4
			}
			if modalH > area.Height-4 {
				modalH = area.Height - 4
			}
			modalArea := terminal.CenterRect(area, modalW, modalH)
			f.RegisterModal("file_select_modal", modalArea, func() {
				state.ShowFileModal = false
			})
		}

		rootLay := layout.NewFlexLayout(
			layout.Vertical,
			0,
			layout.Fixed(3), // Header
			layout.Fill(),   // Body
			layout.Fixed(1), // Footer
		)
		chunks := rootLay.Split(area)

		// 1. Header
		polyCount := len(state.ActiveModel.Faces)
		vertCount := len(state.ActiveModel.Vertices)
		modeNames := []string{"ASCII Typography", "Half-Block 2x (Micro-Cell)", "Braille 8x (Ultra-Dots)", "Solid Blocks"}
		modeStr := modeNames[state.RenderMode%len(modeNames)]
		headerSub := fmt.Sprintf("Model: %s | Poly: %d | Vert: %d | FPS: %.1f | Scale: %.1f | Mode: %s [M] | [H] Panel: %s",
			state.ActiveModelName, polyCount, vertCount, state.FPS, state.Scale,
			modeStr,
			ternary(state.ShowHUD, "Open", "Hidden"),
		)
		f.RenderWidget(widgets.Block{
			Title:          " ◆ LIMONI TUI ◆ REAL-TIME 3D ASCII OBJECT ENGINE ",
			TitleAlignment: widgets.AlignCenter,
			Borders:        widgets.BorderAll,
			BorderSymbols:  widgets.SymbolsRounded,
			BorderStyle:    cell.Style{Fg: accentCol},
			Child:          &widgets.Paragraph{Text: " " + headerSub, Style: cell.Style{Fg: cell.NewColorRGB(190, 205, 230)}},
		}, chunks[0])

		// 2. Main Body Split
		if state.ShowHUD {
			sidebarW := uint16(34)
			if area.Width < 80 {
				sidebarW = 28
			}
			bodyLay := layout.NewFlexLayout(
				layout.Horizontal,
				1,
				layout.Fixed(sidebarW), // Sidebar controls
				layout.Fill(),          // 3D Viewport
			)
			bodyChunks := bodyLay.Split(chunks[1])
			drawControlPanel(f, state, bodyChunks[0], t.FocusManager().Focused(), accentCol)
			draw3DViewport(f, state, bodyChunks[1], accentCol)
		} else {
			draw3DViewport(f, state, chunks[1], accentCol)
		}

		// 3. Footer
		footerText := " [M] Mode (ASCII/2x/8x) | [f/F] Font Size | [+/-] Scale | [[/]] Aspect | [Space] AutoRot | [C] Color | [H] Panel | [Esc] Exit"
		f.RenderWidget(widgets.Block{
			Borders: widgets.BorderNone,
			Child:   &widgets.Paragraph{Text: footerText, Style: cell.Style{Fg: cell.NewColorRGB(140, 155, 175)}},
		}, chunks[2])

		// File Selection Modal
		if state.ShowFileModal {
			drawModelModal(f, state, area, accentCol)
		}
	})
}

func drawControlPanel(f *terminal.Frame, state *AppState, area cell.Rect, focused string, accentCol cell.Color) {
	f.RenderWidget(widgets.Block{
		Title:         " PARAMETERS ",
		Borders:       widgets.BorderAll,
		BorderSymbols: widgets.SymbolsRounded,
		BorderStyle:   cell.Style{Fg: cell.NewColorRGB(60, 75, 100)},
	}, area)

	inner := cell.NewRect(area.X+1, area.Y+1, area.Width-2, area.Height-2)

	modelOpts := []string{"Duck", "Torus", "Sphere", "Pyramid", "Cube", "Load File..."}
	rampOpts := make([]string, len(ramps))
	for i, r := range ramps {
		rampOpts[i] = r.Name
	}
	palOpts := make([]string, len(palettes))
	for i, p := range palettes {
		palOpts[i] = p.Name
	}

	modeH := uint16(3)
	if state.ModeSelectState.Open {
		modeH = 8
	}
	modelH := uint16(3)
	if state.ModelSelectState.Open {
		modelH = 8
	}
	rampH := uint16(3)
	if state.RampSelectState.Open {
		rampH = 8
	}
	palH := uint16(3)
	if state.PaletteSelectState.Open {
		palH = 8
	}

	lay := layout.NewFlexLayout(
		layout.Vertical,
		0,
		layout.Fixed(modeH),  // Mode Select
		layout.Fixed(modelH), // Model
		layout.Fixed(rampH),  // Ramp
		layout.Fixed(palH),   // Palette
		layout.Fixed(2),      // Scale Slider
		layout.Fixed(2),      // Cell Aspect Slider
		layout.Fixed(2),      // Contrast Slider
		layout.Fixed(2),      // Edge Contrast Slider
		layout.Fixed(2),      // Exposure Slider
		layout.Fixed(2),      // Roughness Slider
		layout.Fixed(2),      // Float Intensity Slider
		layout.Fixed(2),      // Float Speed Slider
		layout.Fixed(2),      // Camera Distance
		layout.Fixed(2),      // FOV Slider
		layout.Fill(),        // Quick Status
	)
	rows := lay.Split(inner)

	// Mode Select with OnChange
	modeOpts := make([]string, len(renderModes))
	for i, m := range renderModes {
		modeOpts[i] = m.Name
	}
	f.RenderWidget(widgets.Block{
		Title: " RENDER MODE ", Borders: widgets.BorderAll, BorderSymbols: widgets.SymbolsRounded,
		BorderStyle: cell.Style{Fg: ternaryStyle(focused == "mode_select" || state.ModeSelectState.Open, accentCol, cell.NewColorRGB(60, 70, 90))},
		Child: widgets.Select{
			ID:      "mode_select",
			Options: modeOpts,
			State:   state.ModeSelectState,
			OnChange: func(index int, option string) {
				state.ModeSelectState.Selected = index
				if index >= 0 && index < len(renderModes) {
					state.RenderMode = index
				}
			},
			Style:         cell.Style{Fg: cell.NewColorRGB(200, 210, 230)},
			SelectedStyle: cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: accentCol, Modifier: cell.ModifierBold},
		},
	}, rows[0])

	// Model Select with OnChange
	f.RenderWidget(widgets.Block{
		Title: " 3D GEOMETRY ", Borders: widgets.BorderAll, BorderSymbols: widgets.SymbolsRounded,
		BorderStyle: cell.Style{Fg: ternaryStyle(focused == "model_select" || state.ModelSelectState.Open, accentCol, cell.NewColorRGB(60, 70, 90))},
		Child: widgets.Select{
			ID:      "model_select",
			Options: modelOpts,
			State:   state.ModelSelectState,
			OnChange: func(index int, option string) {
				state.ModelSelectState.Selected = index
				if index == 5 {
					state.ModelFiles = scan3DFiles()
					state.FileListState.Selected = 0
					state.ShowFileModal = true
				} else if index >= 0 && index < 5 {
					state.ActiveModelName = option
					state.ActiveModel = getModelByName(option)
				}
			},
			Style:         cell.Style{Fg: cell.NewColorRGB(200, 210, 230)},
			SelectedStyle: cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: accentCol, Modifier: cell.ModifierBold},
		},
	}, rows[1])

	// Ramp Select with OnChange
	f.RenderWidget(widgets.Block{
		Title: " ASCII RAMP ", Borders: widgets.BorderAll, BorderSymbols: widgets.SymbolsRounded,
		BorderStyle: cell.Style{Fg: ternaryStyle(focused == "ramp_select" || state.RampSelectState.Open, accentCol, cell.NewColorRGB(60, 70, 90))},
		Child: widgets.Select{
			ID:      "ramp_select",
			Options: rampOpts,
			State:   state.RampSelectState,
			OnChange: func(index int, option string) {
				state.RampSelectState.Selected = index
				if index >= 0 && index < len(ramps) {
					state.SelectedRamp = index
					state.ActiveRamp = ramps[index].Ramp
				}
			},
			Style:         cell.Style{Fg: cell.NewColorRGB(200, 210, 230)},
			SelectedStyle: cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: accentCol, Modifier: cell.ModifierBold},
		},
	}, rows[2])

	// Palette Select with OnChange
	f.RenderWidget(widgets.Block{
		Title: " HIGHLIGHT COLOR ", Borders: widgets.BorderAll, BorderSymbols: widgets.SymbolsRounded,
		BorderStyle: cell.Style{Fg: ternaryStyle(focused == "palette_select" || state.PaletteSelectState.Open, accentCol, cell.NewColorRGB(60, 70, 90))},
		Child: widgets.Select{
			ID:      "palette_select",
			Options: palOpts,
			State:   state.PaletteSelectState,
			OnChange: func(index int, option string) {
				state.PaletteSelectState.Selected = index
				if index >= 0 && index < len(palettes) {
					state.PaletteIndex = index
				}
			},
			Style:         cell.Style{Fg: cell.NewColorRGB(200, 210, 230)},
			SelectedStyle: cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: accentCol, Modifier: cell.ModifierBold},
		},
	}, rows[3])

	// Sliders
	renderSliderRow(f, rows[4], "scale_slider", fmt.Sprintf("Scale: %.1f", state.Scale), state.ScaleSliderState, 8, 80, accentCol, focused, func() {
		syncStateFromSliders(state)
	})
	renderSliderRow(f, rows[5], "aspect_slider", fmt.Sprintf("Cell Aspect: %.2f", state.CellAspect), state.CellAspectSliderState, 25, 80, accentCol, focused, func() {
		syncStateFromSliders(state)
	})
	renderSliderRow(f, rows[6], "contrast_slider", fmt.Sprintf("Contrast: %.1f", state.Contrast), state.ContrastSliderState, 5, 30, accentCol, focused, func() {
		syncStateFromSliders(state)
	})
	renderSliderRow(f, rows[7], "edge_slider", fmt.Sprintf("Edge Boost: %.1f", state.EdgeContrast), state.EdgeContrastSlider, 0, 50, accentCol, focused, func() {
		syncStateFromSliders(state)
	})
	renderSliderRow(f, rows[8], "exposure_slider", fmt.Sprintf("Exposure: %.1f", state.Exposure), state.ExposureSliderState, 2, 30, accentCol, focused, func() {
		syncStateFromSliders(state)
	})
	renderSliderRow(f, rows[9], "roughness_slider", fmt.Sprintf("Roughness: %.2f", state.Roughness), state.RoughnessSliderState, 1, 100, accentCol, focused, func() {
		syncStateFromSliders(state)
	})
	renderSliderRow(f, rows[10], "float_int_slider", fmt.Sprintf("Float Amp: %.1f", state.FloatIntensity), state.FloatIntSliderState, 0, 50, accentCol, focused, func() {
		syncStateFromSliders(state)
	})
	renderSliderRow(f, rows[11], "float_spd_slider", fmt.Sprintf("Float Spd: %.1f", state.FloatSpeed), state.FloatSpeedSliderState, 0, 50, accentCol, focused, func() {
		syncStateFromSliders(state)
	})
	renderSliderRow(f, rows[12], "dist_slider", fmt.Sprintf("Camera Dist: %.1f", state.CameraDistance), state.DistanceSliderState, 15, 90, accentCol, focused, func() {
		syncStateFromSliders(state)
	})
	renderSliderRow(f, rows[13], "fov_slider", fmt.Sprintf("FOV: %.0f°", state.FOV), state.FOVSliderState, 30, 110, accentCol, focused, func() {
		syncStateFromSliders(state)
	})

	// Status Info
	infoStr := fmt.Sprintf("ASCII: %s  Color: %s\nInvert: %s AutoRot: %s\nRot: X:%.0f° Y:%.0f°",
		boolToTag(state.Ascii), boolToTag(state.Colored),
		boolToTag(state.Invert), boolToTag(state.AutoRotate),
		state.RotX, state.RotY,
	)
	f.RenderWidget(widgets.Block{
		Title:       " STATE ",
		Borders:     widgets.BorderAll,
		BorderStyle: cell.Style{Fg: cell.NewColorRGB(50, 60, 80)},
		PaddingLeft: 1,
		Child:       &widgets.Paragraph{Text: infoStr, Style: cell.Style{Fg: cell.NewColorRGB(160, 175, 195)}},
	}, rows[14])
}

func renderSliderRow(f *terminal.Frame, area cell.Rect, id string, label string, sliderState *widgets.SliderState, minVal, maxVal int, accentCol cell.Color, focused string, onChange func()) {
	if area.Height < 2 {
		return
	}
	f.RenderWidget(&widgets.Paragraph{
		Text:  label,
		Style: cell.Style{Fg: cell.NewColorRGB(180, 190, 210)},
	}, cell.NewRect(area.X, area.Y, area.Width, 1))

	sliderStyle := cell.Style{Fg: cell.NewColorRGB(45, 55, 70)}
	filledStyle := cell.Style{Fg: accentCol}
	if focused == id {
		sliderStyle = cell.Style{Fg: cell.NewColorRGB(70, 85, 110)}
	}

	f.RenderWidget(widgets.Slider{
		ID:          id,
		State:       sliderState,
		Min:         minVal,
		Max:         maxVal,
		TrackStyle:  sliderStyle,
		FilledStyle: filledStyle,
		ThumbStyle:  cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Modifier: cell.ModifierBold},
		OnChange: func(val int) {
			if onChange != nil {
				onChange()
			}
		},
	}, cell.NewRect(area.X, area.Y+1, area.Width, 1))
}

func draw3DViewport(f *terminal.Frame, state *AppState, area cell.Rect, accentCol cell.Color) {
	f.RenderWidget(widgets.Block{
		Title:          fmt.Sprintf(" 3D VIEWPORT (%dx%d) ", area.Width-2, area.Height-2),
		TitleAlignment: widgets.AlignCenter,
		Borders:        widgets.BorderAll,
		BorderSymbols:  widgets.SymbolsRounded,
		BorderStyle:    cell.Style{Fg: accentCol},
	}, area)

	innerArea := cell.NewRect(area.X+1, area.Y+1, area.Width-2, area.Height-2)
	if innerArea.Width < 2 || innerArea.Height < 2 {
		return
	}

	elapsedSec := time.Since(state.StartTime).Seconds()
	pal := palettes[state.PaletteIndex]
	currentMode := renderModes[state.RenderMode%len(renderModes)].Mode

	asciiWidget := widgets.Ascii3D{
		Model:                state.ActiveModel,
		Mode:                 currentMode,
		Scale:                state.Scale,
		XOffset:              state.XOffset,
		YOffset:              state.YOffset,
		FloatIntensity:       state.FloatIntensity,
		FloatSpeed:           state.FloatSpeed,
		RotationIntensity:    state.RotationIntensity,
		AutoRotate:           state.AutoRotate,
		AutoRotateSpeed:      state.AutoRotateSpeed,
		Time:                 elapsedSec,
		RotX:                 state.RotX,
		RotY:                 state.RotY,
		RotZ:                 state.RotZ,
		FOV:                  state.FOV,
		CameraDistance:       state.CameraDistance,
		CellAspect:           state.CellAspect,
		Contrast:             state.Contrast,
		EdgeContrast:         state.EdgeContrast,
		Exposure:             state.Exposure,
		EnvironmentIntensity: state.EnvironmentIntensity,
		Roughness:            state.Roughness,
		Ascii:                state.Ascii,
		Colored:              state.Colored,
		Invert:               state.Invert,
		Color:                pal.BaseColor,
		Highlight:            pal.Highlight,
		Ramp:                 state.ActiveRamp,
	}

	f.RenderWidget(asciiWidget, innerArea)
}

func drawModelModal(f *terminal.Frame, state *AppState, screenArea cell.Rect, accentCol cell.Color) {
	modalW, modalH := uint16(60), uint16(14)
	modalArea := terminal.CenterRect(screenArea, modalW, modalH)
	widgets.DrawShadow(f.Buffer, modalArea, 2, 1)

	f.RenderWidget(widgets.Block{
		Title:          " [ OPEN 3D MODEL FILE ]  ↑/↓ Browse · Enter Load · Esc Close ",
		TitleAlignment: widgets.AlignLeft,
		Borders:        widgets.BorderAll,
		BorderSymbols:  widgets.SymbolsRounded,
		BorderStyle:    cell.Style{Fg: accentCol},
		Style:          cell.Style{Bg: cell.NewColorRGB(18, 22, 30)},
		Opaque:         true,
	}, modalArea)

	f.BeginFocusScope("file_select_modal")

	inner := cell.NewRect(modalArea.X+2, modalArea.Y+2, modalArea.Width-4, modalArea.Height-3)
	items := make([]string, len(state.ModelFiles))
	for i, path := range state.ModelFiles {
		items[i] = fmt.Sprintf("%-24s (%s)", filepath.Base(path), strings.ToUpper(strings.TrimPrefix(filepath.Ext(path), ".")))
	}
	if len(items) == 0 {
		items = []string{"(No .glb, .gltf, .obj, .stl, .ply files found in project)"}
	}

	f.RenderWidget(widgets.List{
		ID:              "model_file_list",
		Items:           items,
		State:           state.FileListState,
		Scrollbar:       true,
		Style:           cell.Style{Fg: cell.NewColorRGB(210, 220, 240)},
		SelectedStyle:   cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: accentCol, Modifier: cell.ModifierBold},
		HighlightSymbol: "▶ ",
	}, inner)
}

func boolToTag(b bool) string {
	if b {
		return "[ON]"
	}
	return "[OFF]"
}

func ternary[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}

func ternaryStyle(cond bool, a, b cell.Color) cell.Color {
	if cond {
		return a
	}
	return b
}
