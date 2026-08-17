package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/graphics"
	"github.com/thebanri/limoni/layout"
	"github.com/thebanri/limoni/widgets"
)

type text struct {
	value string
	style cell.Style
}

func (t text) Draw(ctx cell.Context, buf *buffer.Buffer) {
	mergedStyle := ctx.Style.Merge(t.style)
	currY := ctx.Area.Y
	for _, line := range strings.Split(t.value, "\n") {
		if currY >= ctx.Area.Y+ctx.Area.Height {
			break
		}
		buf.SetString(ctx.Area.X, currY, line, mergedStyle)
		currY++
	}
}

func (t text) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	lines := strings.Split(t.value, "\n")
	maxW := 0
	for _, l := range lines {
		if len(l) > maxW {
			maxW = len(l)
		}
	}
	return uint16(maxW), uint16(len(lines))
}

type AppState struct {
	RotX, RotY, RotZ float64
	AutoRotate       bool
	RotationSpeed    int
	ZoomScale        int
	Distance         float64
	ShadingMode      string // "Wireframe", "Solid", "Lambert", "Gouraud", "Textured"
	ActiveShape      string // "Cube", "Pyramid", "Sphere", "Torus", "Custom"
	LightAngle       float64

	ShapeSelectState   *widgets.SelectState
	ShadingSelectState *widgets.SelectState
	ZoomSliderState    *widgets.SliderState
	SpeedSliderState   *widgets.SliderState

	Canvas     *widgets.Canvas
	DragActive bool
	LastDragX  int
	LastDragY  int

	FPS float64

	// New fields for Custom Model and Background overlays
	OBJModel   *graphics.Model3D
	OBJPath    string
	OverlayImg image.Image
	ImgPath    string

	ShowModelModal bool
	ModelFiles     []string
	ModelListState *widgets.ListState

	ShowImageModal bool
	ImageFiles     []string
	ImageListState *widgets.ListState

	imageCache map[string]image.Image
}

func (s *AppState) getOrLoadImage(path string) image.Image {
	if s.imageCache == nil {
		s.imageCache = make(map[string]image.Image)
	}
	if img, ok := s.imageCache[path]; ok {
		return img
	}
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		return nil
	}
	s.imageCache[path] = img
	return img
}

func scanModelFiles() []string {
	var files []string
	dirs := []string{"examples/demo", "examples/3d_viewer", "."}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if strings.HasSuffix(name, ".obj") || strings.HasSuffix(name, ".stl") || strings.HasSuffix(name, ".ply") {
				path := filepath.Join(dir, name)
				files = append(files, path)
			}
		}
	}
	return files
}

func scanImageFiles() []string {
	var files []string
	dirs := []string{"examples/demo", "examples/3d_viewer", "."}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if strings.HasSuffix(name, ".png") || strings.HasSuffix(name, ".jpg") || strings.HasSuffix(name, ".jpeg") {
				path := filepath.Join(dir, name)
				files = append(files, path)
			}
		}
	}
	return files
}

func main() {
	b := backend.NewBackend(os.Stdin, os.Stdout)
	if err := b.Setup(); err != nil {
		fmt.Fprintf(os.Stderr, "Hata: %v\n", err)
		os.Exit(1)
	}
	defer b.Close()

	t, err := terminal.New(b)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Hata: %v\n", err)
		os.Exit(1)
	}

	b.StartEventLoop()

	shapeSelect := widgets.NewSelectState()
	shadingSelect := widgets.NewSelectState()
	zoomSlider := widgets.NewSliderState(40)
	speedSlider := widgets.NewSliderState(50)
	modelListState := widgets.NewListState()
	imageListState := widgets.NewListState()

	imgFiles := scanImageFiles()
	var defaultImg image.Image
	var defaultImgPath string
	for _, p := range imgFiles {
		if strings.Contains(p, "apple.png") || defaultImg == nil {
			if file, err := os.Open(p); err == nil {
				if img, _, err := image.Decode(file); err == nil {
					defaultImg = img
					defaultImgPath = p
					if strings.Contains(p, "apple.png") {
						file.Close()
						break
					}
				}
				file.Close()
			}
		}
	}

	state := &AppState{
		RotX:               25.0,
		RotY:               45.0,
		RotZ:               0.0,
		AutoRotate:         true,
		RotationSpeed:      30,
		ZoomScale:          40,
		Distance:           3.5,
		ShadingMode:        "Textured",
		ActiveShape:        "Cube",
		ShapeSelectState:   shapeSelect,
		ShadingSelectState: shadingSelect,
		ZoomSliderState:    zoomSlider,
		SpeedSliderState:   speedSlider,
		ModelListState:     modelListState,
		ImageListState:     imageListState,
		OverlayImg:         defaultImg,
		ImgPath:            defaultImgPath,
		ImageFiles:         imgFiles,
	}

	ticker := time.NewTicker(33 * time.Millisecond)
	defer ticker.Stop()

	frameCount := 0
	lastFpsCalc := time.Now()

	draw := func() {
		drawApp(t, state)
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
				if state.ShowModelModal {
					if ev.Key.Type == backend.KeyEsc {
						state.ShowModelModal = false
						draw()
						continue
					}
					if ev.Key.Type == backend.KeyArrowUp {
						if state.ModelListState.Selected > 0 {
							state.ModelListState.Selected--
						}
					}
					if ev.Key.Type == backend.KeyArrowDown {
						if state.ModelListState.Selected < len(state.ModelFiles)-1 {
							state.ModelListState.Selected++
						}
					}
					if ev.Key.Type == backend.KeyEnter {
						if len(state.ModelFiles) > 0 && state.ModelListState.Selected >= 0 && state.ModelListState.Selected < len(state.ModelFiles) {
							path := state.ModelFiles[state.ModelListState.Selected]
							var model graphics.Model3D
							var loadErr error
							if strings.HasSuffix(strings.ToLower(path), ".stl") {
								model, loadErr = graphics.LoadSTL(path)
							} else if strings.HasSuffix(strings.ToLower(path), ".ply") {
								model, loadErr = graphics.LoadPLY(path)
							} else {
								model, loadErr = graphics.LoadOBJ(path)
							}
							if loadErr == nil {
								model.Normalize(2.2)
								state.OBJModel = &model
								state.OBJPath = path
								state.ActiveShape = "Custom"
								state.ShapeSelectState.Selected = 4
							}
						}
						state.ShowModelModal = false
						draw()
						continue
					}
					draw()
					continue
				}

				if state.ShowImageModal {
					if ev.Key.Type == backend.KeyEsc {
						state.ShowImageModal = false
						draw()
						continue
					}
					if ev.Key.Type == backend.KeyArrowUp {
						if state.ImageListState.Selected > 0 {
							state.ImageListState.Selected--
						}
					}
					if ev.Key.Type == backend.KeyArrowDown {
						if state.ImageListState.Selected < len(state.ImageFiles)-1 {
							state.ImageListState.Selected++
						}
					}
					if ev.Key.Type == backend.KeyEnter {
						if len(state.ImageFiles) > 0 && state.ImageListState.Selected >= 0 && state.ImageListState.Selected < len(state.ImageFiles) {
							path := state.ImageFiles[state.ImageListState.Selected]
							img := state.getOrLoadImage(path)
							if img != nil {
								state.OverlayImg = img
								state.ImgPath = path
								state.ShadingMode = "Dokulu"
								state.ShadingSelectState.Selected = 0
							}
						}
						state.ShowImageModal = false
						draw()
						continue
					}
					draw()
					continue
				}

				if ev.Key.Type == backend.KeyEsc || (ev.Key.Type == backend.KeyRune && ev.Key.Ch == 'q') {
					return
				}

				// Check Ctrl+E and Ctrl+S
				if ev.Key.Type == backend.KeyRune && ev.Key.Ch == 'e' && ev.Key.Ctrl {
					state.ModelFiles = scanModelFiles()
					state.ModelListState.Selected = 0
					state.ShowModelModal = true
					state.ShowImageModal = false
					draw()
					continue
				}
				if ev.Key.Type == backend.KeyRune && ev.Key.Ch == 's' && ev.Key.Ctrl {
					state.ImageFiles = scanImageFiles()
					state.ImageListState.Selected = 0
					state.ShowImageModal = true
					state.ShowModelModal = false
					draw()
					continue
				}

				if ev.Key.Type == backend.KeyTab {
					t.FocusManager().Next()
				}

				focused := t.FocusManager().Focused()
				optionsLen := 4
				if state.OBJModel != nil {
					optionsLen = 5
				}

				switch focused {
				case "shape_select":
					state.ShapeSelectState.HandleKey(ev.Key, optionsLen)
				case "shading_select":
					state.ShadingSelectState.HandleKey(ev.Key, 5)
				case "zoom_slider":
					zoomSlider.HandleKey(ev.Key, 10, 100)
				case "speed_slider":
					speedSlider.HandleKey(ev.Key, 0, 100)
				}

				if ev.Key.Type == backend.KeySpace {
					state.AutoRotate = !state.AutoRotate
				}

				normAngle := func(a float64) float64 {
					a = math.Mod(a, 360.0)
					if a < 0 {
						a += 360.0
					}
					return a
				}

				if ev.Key.Type == backend.KeyArrowUp {
					state.RotX = normAngle(state.RotX + 5)
				}
				if ev.Key.Type == backend.KeyArrowDown {
					state.RotX = normAngle(state.RotX - 5)
				}
				if ev.Key.Type == backend.KeyArrowLeft {
					state.RotY = normAngle(state.RotY - 5)
				}
				if ev.Key.Type == backend.KeyArrowRight {
					state.RotY = normAngle(state.RotY + 5)
				}

				draw()

			case backend.EventMouse:
				if state.ShowModelModal || state.ShowImageModal {
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
								state.RotY = math.Mod(state.RotY+float64(dx)*1.5, 360.0)
								if state.RotY < 0 {
									state.RotY += 360.0
								}
								state.RotX = math.Mod(state.RotX+float64(dy)*1.5, 360.0)
								if state.RotX < 0 {
									state.RotX += 360.0
								}
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
						if ev.Mouse.X >= 30 {
							if state.ZoomScale < 100 {
								state.ZoomScale += 5
								zoomSlider.Value = state.ZoomScale
							}
						}
					} else if ev.Mouse.Button == backend.MouseScrollDown {
						if ev.Mouse.X >= 30 {
							if state.ZoomScale > 10 {
								state.ZoomScale -= 5
								zoomSlider.Value = state.ZoomScale
							}
						}
					}
				}

				draw()

			case backend.EventResize:
				draw()
			}

		case <-ticker.C:
			if state.AutoRotate {
				speedFactor := float64(state.RotationSpeed) / 50.0
				state.RotY = math.Mod(state.RotY+1.5*speedFactor, 360.0)
				if state.RotY < 0 {
					state.RotY += 360.0
				}
				state.RotX = math.Mod(state.RotX+0.8*speedFactor, 360.0)
				if state.RotX < 0 {
					state.RotX += 360.0
				}
				state.RotZ = math.Mod(state.RotZ+0.4*speedFactor, 360.0)
				if state.RotZ < 0 {
					state.RotZ += 360.0
				}
			}
			state.LightAngle = math.Mod(state.LightAngle+0.05, 2*math.Pi)

			draw()

			frameCount++
			if time.Since(lastFpsCalc) >= 1*time.Second {
				state.FPS = float64(frameCount) / time.Since(lastFpsCalc).Seconds()
				frameCount = 0
				lastFpsCalc = time.Now()
			}
		}
	}
}

func drawApp(t *terminal.Terminal, state *AppState) {
	t.Draw(func(f *terminal.Frame) {
		area := f.Buffer.Area
		f.SetTheme(widgets.DarkTheme())

		accentColor := cell.NewColorRGB(0, 255, 128)

		// Update values from widgets
		state.ZoomScale = state.ZoomSliderState.Value
		state.RotationSpeed = state.SpeedSliderState.Value

		options := []string{"Cube", "Pyramid", "Sphere", "Torus"}
		if state.OBJModel != nil {
			options = []string{"Cube", "Pyramid", "Sphere", "Torus", "Custom"}
		}
		selectedIdx := state.ShapeSelectState.Selected % len(options)
		if selectedIdx == 4 {
			state.ActiveShape = "Custom"
		} else {
			state.ActiveShape = options[selectedIdx]
		}

		shadings := []string{"Textured", "Wireframe", "Solid", "Lambert", "Gouraud"}
		state.ShadingMode = shadings[state.ShadingSelectState.Selected%len(shadings)]

		modelModalW, modelModalH := uint16(72), uint16(16)
		imageModalW, imageModalH := uint16(72), uint16(16)

		if state.ShowModelModal {
			modalArea := terminal.CenterRect(area, modelModalW, modelModalH)
			f.RegisterModal("model_select_modal", modalArea, func() {
				state.ShowModelModal = false
			})
		}
		if state.ShowImageModal {
			modalArea := terminal.CenterRect(area, imageModalW, imageModalH)
			f.RegisterModal("image_select_modal", modalArea, func() {
				state.ShowImageModal = false
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

		// Header
		f.RenderWidget(widgets.Block{
			Title:          " LIMONI TUI - 3D GEOMETRY VISUALIZER ",
			TitleAlignment: widgets.AlignCenter,
			Borders:        widgets.BorderAll,
			BorderSymbols:  widgets.SymbolsRounded,
			BorderStyle:    cell.Style{Fg: accentColor},
			Child:          text{value: " Real-time 3D vector projection, directional lighting, and Gouraud shading ", style: cell.Style{Fg: cell.NewColorRGB(200, 200, 200)}},
		}, chunks[0])

		bodyLay := layout.NewFlexLayout(
			layout.Horizontal,
			0,
			layout.Fixed(30),
			layout.Fill(),
		)
		bodyChunks := bodyLay.Split(chunks[1])

		drawControls(f, state, bodyChunks[0], t.FocusManager().Focused())
		draw3DCanvas(f, state, bodyChunks[1])

		// Footer
		footerText := fmt.Sprintf(" [Tab] Focus | [Ctrl+E] Load 3D | [Ctrl+S] Load Texture | [Space] Auto Rotate | FPS: %.1f", state.FPS)
		f.RenderWidget(widgets.Block{
			Borders: widgets.BorderNone,
			Style:   cell.Style{Fg: cell.NewColorRGB(20, 20, 25)},
			Child:   text{value: footerText, style: cell.Style{Fg: cell.NewColorRGB(140, 140, 140)}},
		}, chunks[2])

		// ─────────────────────────────────────────────────────
		// 3D MODEL SELECT MODAL (Ctrl+E)
		// ─────────────────────────────────────────────────────
		if state.ShowModelModal {
			modalArea := terminal.CenterRect(area, modelModalW, modelModalH)
			widgets.DrawShadow(f.Buffer, modalArea, 2, 1)

			f.RenderWidget(widgets.Block{
				Title:          " [ 3D MODEL SELECTOR ]  ↑/↓ Browse · Enter Load · Esc Close ",
				TitleAlignment: widgets.AlignLeft,
				Borders:        widgets.BorderAll,
				BorderSymbols:  widgets.SymbolsRounded,
				BorderStyle:    cell.Style{Fg: accentColor},
				Style:          cell.Style{Bg: cell.NewColorRGB(18, 20, 26)},
				Opaque:         true,
			}, modalArea)

			f.BeginFocusScope("model_select_modal")

			innerArea := cell.Rect{
				X:      modalArea.X + 2,
				Y:      modalArea.Y + 1,
				Width:  modalArea.Width - 4,
				Height: modalArea.Height - 2,
			}

			cols := layout.NewFlexLayout(
				layout.Horizontal,
				2,
				layout.Ratio(55),
				layout.Ratio(45),
			).Split(innerArea)

			modelItems := make([]string, len(state.ModelFiles))
			for i, p := range state.ModelFiles {
				modelItems[i] = filepath.Base(p)
			}
			if len(modelItems) == 0 {
				modelItems = []string{"(No 3D model files found)"}
			}
			f.RenderWidget(widgets.List{
				ID:              "model_file_list",
				Items:           modelItems,
				State:           state.ModelListState,
				Scrollbar:       true,
				Style:           cell.Style{Fg: cell.NewColorRGB(210, 215, 230)},
				SelectedStyle:   cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: cell.NewColorRGB(0, 150, 90), Modifier: cell.ModifierBold},
				HighlightSymbol: "▶ ",
			}, cols[0])

			var previewInfo string
			if len(state.ModelFiles) > 0 && state.ModelListState.Selected >= 0 && state.ModelListState.Selected < len(state.ModelFiles) {
				selPath := state.ModelFiles[state.ModelListState.Selected]
				fi, err := os.Stat(selPath)
				sizeStr := "Unknown"
				if err == nil {
					sizeStr = fmt.Sprintf("%.1f KB", float64(fi.Size())/1024.0)
				}
				previewInfo = fmt.Sprintf("File: %s\nFormat: %s\nSize: %s\n\nPress [Enter] to\nload into 3D scene.",
					filepath.Base(selPath), strings.ToUpper(strings.TrimPrefix(filepath.Ext(selPath), ".")), sizeStr)
			} else {
				previewInfo = "Select a 3D model\nfrom the list."
			}

			f.RenderWidget(widgets.Block{
				Borders: widgets.BorderNone,
				Child:   text{value: previewInfo, style: cell.Style{Fg: cell.NewColorRGB(190, 195, 210)}},
			}, cols[1])
		}

		// ─────────────────────────────────────────────────────
		// IMAGE SELECT MODAL (Ctrl+S)
		// ─────────────────────────────────────────────────────
		if state.ShowImageModal {
			modalArea := terminal.CenterRect(area, imageModalW, imageModalH)
			widgets.DrawShadow(f.Buffer, modalArea, 2, 1)

			f.RenderWidget(widgets.Block{
				Title:          " [ IMAGE / TEXTURE SELECTOR ]  ↑/↓ Browse · Enter Load · Esc Close ",
				TitleAlignment: widgets.AlignLeft,
				Borders:        widgets.BorderAll,
				BorderSymbols:  widgets.SymbolsRounded,
				BorderStyle:    cell.Style{Fg: accentColor},
				Style:          cell.Style{Bg: cell.NewColorRGB(18, 20, 26)},
				Opaque:         true,
			}, modalArea)

			f.BeginFocusScope("image_select_modal")

			innerArea := cell.Rect{
				X:      modalArea.X + 2,
				Y:      modalArea.Y + 1,
				Width:  modalArea.Width - 4,
				Height: modalArea.Height - 2,
			}

			cols := layout.NewFlexLayout(
				layout.Horizontal,
				2,
				layout.Ratio(55),
				layout.Ratio(45),
			).Split(innerArea)

			imgItems := make([]string, len(state.ImageFiles))
			for i, p := range state.ImageFiles {
				imgItems[i] = filepath.Base(p)
			}
			if len(imgItems) == 0 {
				imgItems = []string{"(No image files found)"}
			}
			f.RenderWidget(widgets.List{
				ID:              "image_file_list",
				Items:           imgItems,
				State:           state.ImageListState,
				Scrollbar:       true,
				Style:           cell.Style{Fg: cell.NewColorRGB(210, 215, 230)},
				SelectedStyle:   cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: cell.NewColorRGB(0, 150, 90), Modifier: cell.ModifierBold},
				HighlightSymbol: "▶ ",
			}, cols[0])

			var imgPreview widgets.Widget
			if len(state.ImageFiles) > 0 && state.ImageListState.Selected >= 0 && state.ImageListState.Selected < len(state.ImageFiles) {
				selPath := state.ImageFiles[state.ImageListState.Selected]
				liveImg := state.getOrLoadImage(selPath)
				if liveImg != nil {
					imgPreview = &widgets.Image{
						Img: liveImg,
					}
				} else {
					imgPreview = text{value: "Failed to load image:\n" + filepath.Base(selPath), style: cell.Style{Fg: cell.NewColorRGB(220, 80, 80)}}
				}
			} else {
				imgPreview = text{value: "Select an image\nfrom the list.", style: cell.Style{Fg: cell.NewColorRGB(130, 135, 150)}}
			}

			f.RenderWidget(widgets.Block{
				Borders: widgets.BorderNone,
				Child:   imgPreview,
			}, cols[1])
		}
	})
}

func drawControls(f *terminal.Frame, state *AppState, area cell.Rect, focused string) {
	blockBorderCol := cell.NewColorRGB(60, 65, 80)
	accentColor := cell.NewColorRGB(0, 255, 128)

	options := []string{"Cube", "Pyramid", "Sphere", "Torus"}
	if state.OBJModel != nil {
		options = []string{"Cube", "Pyramid", "Sphere", "Torus", "Custom"}
	}

	f.RenderWidget(widgets.Block{
		Title:         " SETTINGS ",
		Borders:       widgets.BorderAll,
		BorderSymbols: widgets.SymbolsRounded,
		BorderStyle:   cell.Style{Fg: blockBorderCol},
	}, area)

	innerArea := cell.NewRect(area.X+1, area.Y+1, area.Width-2, area.Height-2)

	shapeH := uint16(3)
	if state.ShapeSelectState.Open {
		shapeH = uint16(len(options) + 2)
		if shapeH > 8 {
			shapeH = 8
		}
	}

	shadings := []string{"Textured", "Wireframe", "Solid", "Lambert", "Gouraud"}
	shadingH := uint16(3)
	if state.ShadingSelectState.Open {
		shadingH = uint16(len(shadings) + 2)
		if shadingH > 8 {
			shadingH = 8
		}
	}

	ctrlLay := layout.NewFlexLayout(
		layout.Vertical,
		1,
		layout.Fixed(shapeH),   // Geometry Select
		layout.Fixed(shadingH), // Shading Select
		layout.Fixed(3),        // Zoom Slider
		layout.Fixed(3),        // Speed Slider
		layout.Fill(),          // Details
	)
	ctrlChunks := ctrlLay.Split(innerArea)

	shapeBorder := blockBorderCol
	if focused == "shape_select" || state.ShapeSelectState.Open {
		shapeBorder = accentColor
	}
	f.RenderWidget(widgets.Block{
		Title: " GEOMETRY ", Borders: widgets.BorderAll, BorderSymbols: widgets.SymbolsRounded, BorderStyle: cell.Style{Fg: shapeBorder},
		Child: widgets.Select{
			ID:            "shape_select",
			Options:       options,
			State:         state.ShapeSelectState,
			OnChange: func(index int, opt string) {
				if index == 4 && state.OBJModel != nil {
					state.ActiveShape = "Custom"
				} else if index < len(options) {
					state.ActiveShape = options[index]
				}
			},
			Style:         cell.Style{Fg: cell.NewColorRGB(200, 200, 200)},
			SelectedStyle: cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: accentColor, Modifier: cell.ModifierBold},
			HoverStyle:    cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: cell.NewColorRGB(40, 40, 50)},
		},
	}, ctrlChunks[0])

	shadingBorder := blockBorderCol
	if focused == "shading_select" || state.ShadingSelectState.Open {
		shadingBorder = accentColor
	}
	f.RenderWidget(widgets.Block{
		Title: " SHADING ", Borders: widgets.BorderAll, BorderSymbols: widgets.SymbolsRounded, BorderStyle: cell.Style{Fg: shadingBorder},
		Child: widgets.Select{
			ID:            "shading_select",
			Options:       shadings,
			State:         state.ShadingSelectState,
			OnChange: func(index int, opt string) {
				if index >= 0 && index < len(shadings) {
					state.ShadingMode = shadings[index]
				}
			},
			Style:         cell.Style{Fg: cell.NewColorRGB(200, 200, 200)},
			SelectedStyle: cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: accentColor, Modifier: cell.ModifierBold},
			HoverStyle:    cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: cell.NewColorRGB(40, 40, 50)},
		},
	}, ctrlChunks[1])

	zoomBorder := blockBorderCol
	if focused == "zoom_slider" {
		zoomBorder = accentColor
	}
	f.RenderWidget(widgets.Block{
		Title: fmt.Sprintf(" SCALE: %%%d ", state.ZoomScale), Borders: widgets.BorderAll, BorderSymbols: widgets.SymbolsRounded, BorderStyle: cell.Style{Fg: zoomBorder},
		PaddingLeft: 1, PaddingRight: 1,
		Child: widgets.Slider{
			ID:          "zoom_slider",
			State:       state.ZoomSliderState,
			Min:         10,
			Max:         100,
			TrackStyle:  cell.Style{Fg: cell.NewColorRGB(50, 50, 60)},
			FilledStyle: cell.Style{Fg: accentColor},
			ThumbStyle:  cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Modifier: cell.ModifierBold},
		},
	}, ctrlChunks[2])

	speedBorder := blockBorderCol
	if focused == "speed_slider" {
		speedBorder = accentColor
	}
	f.RenderWidget(widgets.Block{
		Title: fmt.Sprintf(" SPEED: %%%d ", state.RotationSpeed), Borders: widgets.BorderAll, BorderSymbols: widgets.SymbolsRounded, BorderStyle: cell.Style{Fg: speedBorder},
		PaddingLeft: 1, PaddingRight: 1,
		Child: widgets.Slider{
			ID:          "speed_slider",
			State:       state.SpeedSliderState,
			Min:         0,
			Max:         100,
			TrackStyle:  cell.Style{Fg: cell.NewColorRGB(50, 50, 60)},
			FilledStyle: cell.Style{Fg: accentColor},
			ThumbStyle:  cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Modifier: cell.ModifierBold},
		},
	}, ctrlChunks[3])

	modelName := state.ActiveShape
	if modelName == "Custom" {
		modelName = "Custom (" + filepath.Base(state.OBJPath) + ")"
	}
	normRotX := math.Mod(state.RotX, 360.0)
	if normRotX < 0 {
		normRotX += 360.0
	}
	normRotY := math.Mod(state.RotY, 360.0)
	if normRotY < 0 {
		normRotY += 360.0
	}
	normRotZ := math.Mod(state.RotZ, 360.0)
	if normRotZ < 0 {
		normRotZ += 360.0
	}
	infoLines := fmt.Sprintf("Model: %s\nMode : %s\nRotX : %5.1f°\nRotY : %5.1f°\nRotZ : %5.1f°",
		modelName, state.ShadingMode, normRotX, normRotY, normRotZ)
	if state.OBJPath != "" {
		infoLines += "\nFile:\n " + filepath.Base(state.OBJPath)
	}
	if state.ImgPath != "" {
		infoLines += "\nTexture:\n " + filepath.Base(state.ImgPath)
	}
	f.RenderWidget(widgets.Block{
		Title:         " DETAILS ",
		Borders:       widgets.BorderAll,
		BorderSymbols: widgets.SymbolsRounded,
		BorderStyle:   cell.Style{Fg: blockBorderCol},
		PaddingLeft:   1,
		Child:         text{value: infoLines, style: cell.Style{Fg: cell.NewColorRGB(180, 180, 190)}},
	}, ctrlChunks[4])
}

func draw3DCanvas(f *terminal.Frame, state *AppState, area cell.Rect) {
	if area.Width < 4 || area.Height < 4 {
		return
	}

	canvasW := area.Width - 2
	canvasH := area.Height - 2

	if state.Canvas == nil {
		state.Canvas = widgets.NewCanvas(canvasW, canvasH)
	} else {
		state.Canvas.Reset(canvasW, canvasH)
	}

	canvas := state.Canvas
	virtualW := int(canvasW) * 2
	virtualH := int(canvasH) * 4

	// Generate Shape Geometry
	var vertices []graphics.Vertex3D
	var faces [][]int

	if state.ActiveShape == "Custom" && state.OBJModel != nil {
		vertices = state.OBJModel.Vertices
		faces = state.OBJModel.Faces
	} else {
		var model graphics.Model3D
		switch state.ActiveShape {
		case "Cube":
			model = graphics.NewCube(2.0)
		case "Pyramid":
			model = graphics.NewPyramid(2.0, 2.0)
		case "Sphere":
			model = graphics.NewSphere(1.1, 14, 14)
		case "Torus":
			model = graphics.NewTorus(0.8, 0.35, 14, 14)
		}
		vertices = model.Vertices
		faces = model.Faces
	}

	if len(vertices) == 0 {
		return
	}

	// Project and Rotate Vertices
	rotated := make([]graphics.Vertex3D, len(vertices))
	projected := make([]struct {
		x, y    float64
		z       float64
		visible bool
	}, len(vertices))

	canvasWFloat := float64(virtualW)
	canvasHFloat := float64(virtualH)

	for i, v := range vertices {
		v = v.RotateY(state.RotY)
		v = v.RotateX(state.RotX)
		v = v.RotateZ(state.RotZ)
		rotated[i] = v

		scale := canvasHFloat * 0.40 * (float64(state.ZoomScale) / 40.0)
		px, py, visible := graphics.Project(v, canvasWFloat, canvasHFloat, state.Distance, scale)
		projected[i] = struct {
			x, y    float64
			z       float64
			visible bool
		}{x: px, y: py, z: v.Z, visible: visible}
	}

	// Colors
	faceColors := []cell.Color{
		cell.NewColorRGB(0, 255, 128), // Mint Green
		cell.NewColorRGB(0, 128, 255), // Sky Blue
		cell.NewColorRGB(255, 0, 128), // Hot Pink
		cell.NewColorRGB(255, 255, 0), // Yellow
		cell.NewColorRGB(255, 128, 0), // Orange
		cell.NewColorRGB(128, 0, 255), // Purple
	}

	lightDir := graphics.Vector3D{
		X: math.Cos(state.LightAngle),
		Y: 0.8,
		Z: math.Sin(state.LightAngle),
	}.Normalize()
	light := graphics.Light{
		Direction: lightDir,
		Ambient:   0.3,
		Diffuse:   0.7,
	}

	wireStyle := cell.Style{Fg: cell.NewColorRGB(0, 255, 128)}
	if state.ShadingMode != "Wireframe" {
		wireStyle = cell.Style{Fg: cell.NewColorRGB(40, 45, 55)}
	}

	// Render faces
	for faceIdx, face := range faces {
		if len(face) < 3 {
			continue
		}

		p0 := projected[face[0]]
		p1 := projected[face[1]]
		p2 := projected[face[2]]

		if !p0.visible || !p1.visible || !p2.visible {
			continue
		}

		var p3 struct {
			x, y    float64
			z       float64
			visible bool
		}
		isQuad := len(face) == 4
		if isQuad {
			p3 = projected[face[3]]
			if !p3.visible {
				continue
			}
		}

		// Backface culling: only render front-facing polygons to keep models completely solid & opaque
		cross1 := (p1.x-p0.x)*(p2.y-p0.y) - (p1.y-p0.y)*(p2.x-p0.x)
		isFrontFacing := cross1 < 0
		if isQuad {
			cross2 := (p2.x-p0.x)*(p3.y-p0.y) - (p2.y-p0.y)*(p3.x-p0.x)
			isFrontFacing = isFrontFacing || cross2 < 0
		}
		if !isFrontFacing {
			continue
		}

		col := faceColors[faceIdx%len(faceColors)]
		faceStyle := cell.Style{Fg: col}

		v0, v1, v2 := rotated[face[0]], rotated[face[1]], rotated[face[2]]
		norm0 := graphics.CalculateNormal(v0, v1, v2)

		switch state.ShadingMode {
		case "Textured":
			if state.OverlayImg != nil {
				if isQuad {
					uv0 := graphics.UV{U: 0.0, V: 1.0}
					uv1 := graphics.UV{U: 1.0, V: 1.0}
					uv2 := graphics.UV{U: 1.0, V: 0.0}
					uv3 := graphics.UV{U: 0.0, V: 0.0}

					canvas.DrawTexturedTriangle(
						graphics.Vertex2D{X: p0.x, Y: p0.y},
						graphics.Vertex2D{X: p1.x, Y: p1.y},
						graphics.Vertex2D{X: p2.x, Y: p2.y},
						uv0, uv1, uv2, state.OverlayImg,
					)
					canvas.DrawTexturedTriangle(
						graphics.Vertex2D{X: p0.x, Y: p0.y},
						graphics.Vertex2D{X: p2.x, Y: p2.y},
						graphics.Vertex2D{X: p3.x, Y: p3.y},
						uv0, uv2, uv3, state.OverlayImg,
					)
				} else {
					uv0 := graphics.UV{U: 0.0, V: 1.0}
					uv1 := graphics.UV{U: 1.0, V: 1.0}
					uv2 := graphics.UV{U: 0.5, V: 0.0}

					canvas.DrawTexturedTriangle(
						graphics.Vertex2D{X: p0.x, Y: p0.y},
						graphics.Vertex2D{X: p1.x, Y: p1.y},
						graphics.Vertex2D{X: p2.x, Y: p2.y},
						uv0, uv1, uv2, state.OverlayImg,
					)
				}
			} else {
				if isQuad {
					canvas.DrawFilledTriangleDepth(graphics.Vertex2D{X: p0.x, Y: p0.y}, graphics.Vertex2D{X: p1.x, Y: p1.y}, graphics.Vertex2D{X: p2.x, Y: p2.y}, p0.z, p1.z, p2.z, faceStyle)
					canvas.DrawFilledTriangleDepth(graphics.Vertex2D{X: p0.x, Y: p0.y}, graphics.Vertex2D{X: p2.x, Y: p2.y}, graphics.Vertex2D{X: p3.x, Y: p3.y}, p0.z, p2.z, p3.z, faceStyle)
				} else {
					canvas.DrawFilledTriangleDepth(graphics.Vertex2D{X: p0.x, Y: p0.y}, graphics.Vertex2D{X: p1.x, Y: p1.y}, graphics.Vertex2D{X: p2.x, Y: p2.y}, p0.z, p1.z, p2.z, faceStyle)
				}
			}

		case "Solid":
			if isQuad {
				canvas.DrawFilledTriangleDepth(graphics.Vertex2D{X: p0.x, Y: p0.y}, graphics.Vertex2D{X: p1.x, Y: p1.y}, graphics.Vertex2D{X: p2.x, Y: p2.y}, p0.z, p1.z, p2.z, faceStyle)
				canvas.DrawFilledTriangleDepth(graphics.Vertex2D{X: p0.x, Y: p0.y}, graphics.Vertex2D{X: p2.x, Y: p2.y}, graphics.Vertex2D{X: p3.x, Y: p3.y}, p0.z, p2.z, p3.z, faceStyle)
			} else {
				canvas.DrawFilledTriangleDepth(graphics.Vertex2D{X: p0.x, Y: p0.y}, graphics.Vertex2D{X: p1.x, Y: p1.y}, graphics.Vertex2D{X: p2.x, Y: p2.y}, p0.z, p1.z, p2.z, faceStyle)
			}

		case "Lambert":
			if isQuad {
				canvas.DrawLambertTriangleDepth(graphics.Vertex2D{X: p0.x, Y: p0.y}, graphics.Vertex2D{X: p1.x, Y: p1.y}, graphics.Vertex2D{X: p2.x, Y: p2.y}, p0.z, p1.z, p2.z, norm0, light, faceStyle)
				v3 := rotated[face[3]]
				norm1 := graphics.CalculateNormal(v0, v2, v3)
				canvas.DrawLambertTriangleDepth(graphics.Vertex2D{X: p0.x, Y: p0.y}, graphics.Vertex2D{X: p2.x, Y: p2.y}, graphics.Vertex2D{X: p3.x, Y: p3.y}, p0.z, p2.z, p3.z, norm1, light, faceStyle)
			} else {
				canvas.DrawLambertTriangleDepth(graphics.Vertex2D{X: p0.x, Y: p0.y}, graphics.Vertex2D{X: p1.x, Y: p1.y}, graphics.Vertex2D{X: p2.x, Y: p2.y}, p0.z, p1.z, p2.z, norm0, light, faceStyle)
			}

		case "Gouraud":
			var n0, n1, n2, n3 graphics.Vector3D
			if state.ActiveShape == "Sphere" || state.ActiveShape == "Torus" {
				n0 = graphics.Vector3D{X: v0.X, Y: v0.Y, Z: v0.Z}.Normalize()
				n1 = graphics.Vector3D{X: v1.X, Y: v1.Y, Z: v1.Z}.Normalize()
				n2 = graphics.Vector3D{X: v2.X, Y: v2.Y, Z: v2.Z}.Normalize()
				if isQuad {
					v3 := rotated[face[3]]
					n3 = graphics.Vector3D{X: v3.X, Y: v3.Y, Z: v3.Z}.Normalize()
				}
			} else {
				n0, n1, n2, n3 = norm0, norm0, norm0, norm0
			}

			c0 := graphics.ApplyShade(col, light.CalculateIntensity(n0))
			c1 := graphics.ApplyShade(col, light.CalculateIntensity(n1))
			c2 := graphics.ApplyShade(col, light.CalculateIntensity(n2))
			if isQuad {
				c3 := graphics.ApplyShade(col, light.CalculateIntensity(n3))
				canvas.DrawGouraudTriangleDepth(graphics.Vertex2D{X: p0.x, Y: p0.y}, graphics.Vertex2D{X: p1.x, Y: p1.y}, graphics.Vertex2D{X: p2.x, Y: p2.y}, p0.z, p1.z, p2.z, c0, c1, c2, cell.Style{})
				canvas.DrawGouraudTriangleDepth(graphics.Vertex2D{X: p0.x, Y: p0.y}, graphics.Vertex2D{X: p2.x, Y: p2.y}, graphics.Vertex2D{X: p3.x, Y: p3.y}, p0.z, p2.z, p3.z, c0, c2, c3, cell.Style{})
			} else {
				canvas.DrawGouraudTriangleDepth(graphics.Vertex2D{X: p0.x, Y: p0.y}, graphics.Vertex2D{X: p1.x, Y: p1.y}, graphics.Vertex2D{X: p2.x, Y: p2.y}, p0.z, p1.z, p2.z, c0, c1, c2, cell.Style{})
			}
		}

		canvas.DrawLine(int(p0.x), int(p0.y), int(p1.x), int(p1.y), wireStyle)
		canvas.DrawLine(int(p1.x), int(p1.y), int(p2.x), int(p2.y), wireStyle)
		if isQuad {
			canvas.DrawLine(int(p2.x), int(p2.y), int(p3.x), int(p3.y), wireStyle)
			canvas.DrawLine(int(p3.x), int(p3.y), int(p0.x), int(p0.y), wireStyle)
		} else {
			canvas.DrawLine(int(p2.x), int(p2.y), int(p0.x), int(p0.y), wireStyle)
		}
	}

	f.RenderWidget(widgets.Block{
		Title:          fmt.Sprintf(" 3D CANLI ÖNİZLEME (%s) ", state.ActiveShape),
		TitleAlignment: widgets.AlignCenter,
		Borders:        widgets.BorderAll,
		BorderSymbols:  widgets.SymbolsRounded,
		BorderStyle:    cell.Style{Fg: cell.NewColorRGB(0, 255, 128)},
		Child:          canvas,
	}, area)
}

