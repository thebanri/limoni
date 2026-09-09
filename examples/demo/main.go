// Limoni Flagship Interactive Demo
// Designed with a clean, professional sidebar layout, full mouse click interaction,
// and studio-quality 3D lemon model rendering.
package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/thebanri/limoni/animation"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/graphics"
	"github.com/thebanri/limoni/layout"
	"github.com/thebanri/limoni/widgets"
)

// Theme defines colors for the UI showcase.
type Theme struct {
	Name      string
	Primary   cell.Color // Main borders and branding
	Accent    cell.Color // Lemon gold / highlighted features
	Secondary cell.Color // Cyan / diffing / active items
	Highlight cell.Color // Specular 3D highlight
	Text      cell.Color
	Muted     cell.Color
	Success   cell.Color
	BgCard    cell.Color
}

var themes = []Theme{
	{
		Name:      "Limoni Citrus (Default)",
		Primary:   cell.NewColorRGB(255, 204, 0),   // Vibrant Citrus Gold
		Accent:    cell.NewColorRGB(255, 230, 80),  // Warm Lemon
		Secondary: cell.NewColorRGB(46, 204, 113),  // Fresh Lime Green
		Highlight: cell.NewColorRGB(255, 255, 220), // Clean Warm Specular Glow
		Text:      cell.NewColorRGB(245, 245, 250),
		Muted:     cell.NewColorRGB(120, 130, 150),
		Success:   cell.NewColorRGB(39, 174, 96),
		BgCard:    cell.NewColorRGB(18, 22, 30),
	},
	{
		Name:      "Cyberpunk Neon",
		Primary:   cell.NewColorRGB(255, 0, 128), // Hot Pink
		Accent:    cell.NewColorRGB(0, 240, 255), // Neon Cyan
		Secondary: cell.NewColorRGB(255, 230, 0), // Electric Yellow
		Highlight: cell.NewColorRGB(255, 255, 255),
		Text:      cell.NewColorRGB(250, 250, 255),
		Muted:     cell.NewColorRGB(130, 80, 140),
		Success:   cell.NewColorRGB(0, 255, 170),
		BgCard:    cell.NewColorRGB(20, 10, 28),
	},
	{
		Name:      "Nord Frost",
		Primary:   cell.NewColorRGB(136, 192, 208), // Frost Cyan
		Accent:    cell.NewColorRGB(129, 161, 193), // Glacial Blue
		Secondary: cell.NewColorRGB(163, 190, 140), // Aurora Green
		Highlight: cell.NewColorRGB(236, 239, 244), // Snow White
		Text:      cell.NewColorRGB(229, 233, 240),
		Muted:     cell.NewColorRGB(94, 129, 172),
		Success:   cell.NewColorRGB(163, 190, 140),
		BgCard:    cell.NewColorRGB(36, 41, 51),
	},
	{
		Name:      "Emerald Forest",
		Primary:   cell.NewColorRGB(46, 204, 113), // Emerald
		Accent:    cell.NewColorRGB(241, 196, 15), // Goldenrod
		Secondary: cell.NewColorRGB(26, 188, 156), // Turquoise
		Highlight: cell.NewColorRGB(200, 255, 220),
		Text:      cell.NewColorRGB(235, 245, 240),
		Muted:     cell.NewColorRGB(80, 130, 110),
		Success:   cell.NewColorRGB(46, 204, 113),
		BgCard:    cell.NewColorRGB(12, 25, 20),
	},
}

type AppState struct {
	// 3D Model State
	Model           graphics.Model3D
	ModelPath       string
	AsciiMode       widgets.Ascii3DMode
	AutoRotate      bool
	Specular        bool
	NaturalColors   bool
	RotateSpeed     float64
	RotX, RotY      float64
	Scale           float64
	Exposure        float64
	Roughness       float64
	RenderModeIndex int

	// TreeView State
	TreeRoots      []widgets.TreeNode
	TreeState      *widgets.TreeViewState
	TreeShowGuides bool

	// Image Viewer State
	Images             []image.Image
	ImageNames         []string
	ActiveImageIdx     int
	ImageCircleMask    bool
	ImageHalfBlock     bool
	ImageAspectCorrect bool

	// Telemetry State
	StartTime      time.Time
	LastFrame      time.Time
	FPS            float64
	FrameCount     int
	FpsTimer       time.Time
	ElapsedSecs    float64
	LatencyHistory []float64
	AllocStats     runtime.MemStats
	ProgressVal    float64
	LiveTelemetry  bool
	ZeroAllocGuard bool
	Logs           []string

	// Settings State
	SGRMouseTracking    bool
	DiffEngineActive    bool
	Smooth60FPS         bool
	AccessibilityActive bool

	// Navigation & System
	ActiveTab             int
	Tabs                  []string
	ThemeIndex            int
	LastMousePos          cell.Point
	ExitRequested         bool
	ShowExitDialog        bool
	ExitDialogAnim        *animation.Float
	ExitDialogSelectedBtn int // 0 = Yes, 1 = No (default 1)

	// Modal & Dialog Dragging State
	IsDraggingModal bool
	ModalOffsetX    int
	ModalOffsetY    int
	DragMouseStartX int
	DragMouseStartY int
	ModalDragBaseX  int
	ModalDragBaseY  int

	// Fuzzy Search & Command Palette
	CmdPalette *widgets.CommandPaletteState
}

func main() {
	b := driver.NewBackend(os.Stdin, os.Stdout)
	if err := b.Setup(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize backend: %v\n", err)
		os.Exit(1)
	}
	defer b.Close()

	term, err := terminal.New(b)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize terminal: %v\n", err)
		os.Exit(1)
	}

	b.StartEventLoop()

	state := initAppState()

	ticker := time.NewTicker(16666 * time.Microsecond) // ~40 FPS smooth tick
	defer ticker.Stop()

	draw := func() {
		renderFrame(term, state)
	}

	draw()

	for {
		if state.ExitRequested {
			return
		}

		select {
		case now := <-ticker.C:
			updateState(state, now)
			draw()

		case ev, ok := <-b.Events():
			if !ok {
				return
			}
			prevTab := state.ActiveTab

			switch ev.Type {
			case driver.EventKey:
				handleKey(ev.Key, state)
				if state.ActiveTab != prevTab {
					b.Write([]byte("\x1b[2J"))
					term.ForceFullRedraw()
					if term.FocusManager() != nil {
						term.FocusManager().Clear()
					}
				}
				draw()

			case driver.EventMouse:
				state.LastMousePos = cell.Point{X: ev.Mouse.X, Y: ev.Mouse.Y}
				handled := term.RouteMouseEvent(ev.Mouse)
				if !handled {
					if ev.Mouse.Button == driver.MouseRelease {
						state.IsDraggingModal = false
					}
				}
				if state.ActiveTab != prevTab {
					b.Write([]byte("\x1b[2J"))
					term.ForceFullRedraw()
					if term.FocusManager() != nil {
						term.FocusManager().Clear()
					}
				} else if ev.Mouse.Button == driver.MouseLeft || ev.Mouse.Button == driver.MouseScrollUp || ev.Mouse.Button == driver.MouseScrollDown || ev.Mouse.Drag {
					draw()
				}

			case driver.EventResize:
				b.Write([]byte("\x1b[2J"))
				term.ForceFullRedraw()
				draw()
			}
		}
	}
}

func initAppState() *AppState {
	state := &AppState{
		AsciiMode:           widgets.ModeASCII,
		AutoRotate:          true,
		Specular:            true,
		NaturalColors:       true,
		RotateSpeed:         28.0,
		Scale:               7.5,
		Exposure:            1.2,
		Roughness:           0.35,
		RenderModeIndex:     0,
		TreeShowGuides:      true,
		ImageCircleMask:     false,
		ImageHalfBlock:      false,
		ImageAspectCorrect:  true,
		LiveTelemetry:       true,
		ZeroAllocGuard:      true,
		SGRMouseTracking:    true,
		DiffEngineActive:    true,
		Smooth60FPS:         true,
		AccessibilityActive: true,
		ActiveTab:           0,
		Tabs: []string{
			"1. [3D] Mascot",
			"2. TreeView",
			"3. Native Images",
			"4. Telemetry",
			"5. Settings",
			"6. Exit",
		},
		ThemeIndex:     0,
		StartTime:      time.Now(),
		LastFrame:      time.Now(),
		FpsTimer:       time.Now(),
		FPS:            60.0,
		LatencyHistory: make([]float64, 40),
		Logs:           make([]string, 0, 10),
	}

	for i := range state.LatencyHistory {
		state.LatencyHistory[i] = 18.0 + float64(i%5)
	}

	// 1. Load 3D Model
	model, path, err := findAndLoadModel()
	if err != nil {
		state.Model = graphics.NewSphere(1.0, 16, 24)
		state.ModelPath = "Procedural Sphere (Fallback)"
	} else {
		state.Model = model
		state.ModelPath = path
	}

	// 2. Initialize TreeView
	state.TreeRoots = buildProjectTree()
	state.TreeState = widgets.NewTreeViewState()
	state.TreeState.Expand("root")
	state.TreeState.Expand("core")
	state.TreeState.Expand("widgets")
	state.TreeState.Select("lemon_glb")

	// 3. Load Sample Images
	loadSampleImages(state)

	// 4. Initialize Command Palette & Fuzzy Search
	state.CmdPalette = widgets.NewCommandPaletteState()
	initCommandPaletteItems(state)

	// 5. Initialize Exit Dialog Animation
	state.ExitDialogAnim = animation.NewFloat(0.0)
	state.ExitDialogSelectedBtn = 1 // Default to "No" for safety

	addLog(state, "Limoni Engine initialized successfully")
	addLog(state, fmt.Sprintf("Loaded 3D model: %s (%d faces)", state.ModelPath, len(state.Model.Faces)))

	return state
}

func findAndLoadModel() (graphics.Model3D, string, error) {
	candidates := []string{
		"examples/demo/limoni.glb",
		"limoni.glb",
		"/home/thebanri/Projects/limoni-website/public/limoni.glb",
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			if strings.HasSuffix(p, ".glb") {
				if m, err := graphics.LoadGLB(p); err == nil {
					return m, filepath.Base(p), nil
				}
			}
		}
	}

	return graphics.Model3D{}, "", fmt.Errorf("limoni.glb not found")
}

func loadSampleImages(state *AppState) {
	paths := []string{
		"examples/demo/apple.png",
		"examples/demo/profile.png",
		"apple.png",
		"profile.png",
		"examples/showcase/apple.png",
		"examples/showcase/profile.png",
	}

	for _, p := range paths {
		if f, err := os.Open(p); err == nil {
			if img, _, err := image.Decode(f); err == nil {
				state.Images = append(state.Images, img)
				state.ImageNames = append(state.ImageNames, filepath.Base(p))
			}
			f.Close()
		}
	}

	// Fallback procedural image if needed
	if len(state.Images) == 0 {
		img := image.NewRGBA(image.Rect(0, 0, 128, 128))
		state.Images = append(state.Images, img)
		state.ImageNames = append(state.ImageNames, "Procedural Fallback")
	}
}

func buildProjectTree() []widgets.TreeNode {
	return []widgets.TreeNode{
		{
			ID:       "root",
			Label:    "Limoni Project Workspace",
			Icon:     "+ ",
			Expanded: true,
			Children: []widgets.TreeNode{
				{
					ID:       "core",
					Label:    "core (Hardware Subsystems)",
					Icon:     "+ ",
					Expanded: true,
					Children: []widgets.TreeNode{
						{ID: "term_engine", Label: "terminal.go (Diff Engine)", Icon: "- "},
						{ID: "c_buffer", Label: "buffer.go (Screen Double-Buffer)", Icon: "- "},
						{ID: "c_backend", Label: "driver.go (ANSI / Raw Mode)", Icon: "- "},
					},
				},
				{
					ID:       "widgets",
					Label:    "widgets (Component Library)",
					Icon:     "+ ",
					Expanded: true,
					Children: []widgets.TreeNode{
						{ID: "treeview_w", Label: "treeview.go (Hierarchical Tree)", Icon: "- "},
						{ID: "image_w", Label: "image.go (Direct Half-Block Pixels)", Icon: "- "},
						{ID: "ascii3d_w", Label: "ascii3d.go (3D Mesh Rasterizer)", Icon: "- "},
						{ID: "w_sparkline", Label: "sparkline.go (Telemetry Chart)", Icon: "- "},
						{ID: "w_checkbox", Label: "checkbox.go (Interactive Toggle)", Icon: "- "},
					},
				},
				{
					ID:       "examples",
					Label:    "examples (Interactive Demos)",
					Icon:     "+ ",
					Expanded: true,
					Children: []widgets.TreeNode{
						{ID: "lemon_glb", Label: "limoni.glb (3D Mascot Model)", Icon: "- "},
						{ID: "ex_demo", Label: "demo (Flagship Trailer Showcase)", Icon: "- "},
						{ID: "ex_showcase", Label: "showcase (Full Feature Gallery)", Icon: "- "},
					},
				},
				{
					ID:       "benchmarks",
					Label:    "benchmarks (Zero-Cheat Suite)",
					Icon:     "+ ",
					Expanded: false,
					Children: []widgets.TreeNode{
						{ID: "b_workloads", Label: "workloads.json (12 Workloads)", Icon: "- "},
						{ID: "b_compare", Label: "compare (Regression Guard)", Icon: "- "},
					},
				},
			},
		},
	}
}

func updateState(state *AppState, now time.Time) {
	dt := now.Sub(state.LastFrame).Seconds()
	state.LastFrame = now
	state.ElapsedSecs = now.Sub(state.StartTime).Seconds()

	if state.ExitDialogAnim != nil {
		state.ExitDialogAnim.Update(now)
	}

	// FPS Calculation
	state.FrameCount++
	if now.Sub(state.FpsTimer) >= 500*time.Millisecond {
		state.FPS = float64(state.FrameCount) / now.Sub(state.FpsTimer).Seconds()
		state.FrameCount = 0
		state.FpsTimer = now
	}

	// 3D Auto-rotation
	if state.AutoRotate {
		state.RotY += state.RotateSpeed * dt
		if state.RotY >= 360 {
			state.RotY -= 360
		}
	}

	// Pulse progress bar
	state.ProgressVal = 50.0 + 45.0*math.Sin(state.ElapsedSecs*1.4)

	// Sample real-time frame diff latency (~18µs)
	if state.LiveTelemetry {
		jitter := (rand.Float64() - 0.5) * 3.2
		simLatency := 18.2 + jitter
		if simLatency < 5.0 {
			simLatency = 5.0
		}
		state.LatencyHistory = append(state.LatencyHistory[1:], simLatency)
	}

	runtime.ReadMemStats(&state.AllocStats)
}

func clampDialogOffset(screen cell.Rect, width, height uint16, offsetX, offsetY int) (int, int) {
	centered := terminal.CenterRect(screen, width, height)
	minX := -int(centered.X)
	maxX := int(screen.Width) - int(centered.X) - int(width)
	minY := -int(centered.Y)
	maxY := int(screen.Height) - int(centered.Y) - int(height)
	if maxX < minX {
		maxX = minX
	}
	if maxY < minY {
		maxY = minY
	}
	if offsetX < minX {
		offsetX = minX
	}
	if offsetX > maxX {
		offsetX = maxX
	}
	if offsetY < minY {
		offsetY = minY
	}
	if offsetY > maxY {
		offsetY = maxY
	}
	return offsetX, offsetY
}

func initCommandPaletteItems(state *AppState) {
	cmdItems := []widgets.CommandItem{
		// 1. Navigation Tabs
		{Label: "Tab: 1. [3D] Mascot", Detail: "Go to 3D Mascot tab [1]", Category: "Navigation", Handler: func() { state.ActiveTab = 0; addLog(state, "Jumped to 3D Mascot") }},
		{Label: "Tab: 2. TreeView", Detail: "Go to TreeView tab [2]", Category: "Navigation", Handler: func() { state.ActiveTab = 1; addLog(state, "Jumped to TreeView") }},
		{Label: "Tab: 3. Native Images", Detail: "Go to Native Images tab [3]", Category: "Navigation", Handler: func() { state.ActiveTab = 2; addLog(state, "Jumped to Native Images") }},
		{Label: "Tab: 4. Telemetry", Detail: "Go to Telemetry tab [4]", Category: "Navigation", Handler: func() { state.ActiveTab = 3; addLog(state, "Jumped to Telemetry") }},
		{Label: "Tab: 5. Settings", Detail: "Go to Settings tab [5]", Category: "Navigation", Handler: func() { state.ActiveTab = 4; addLog(state, "Jumped to Settings") }},
		{Label: "System: Exit Application", Detail: "Key 6 / q", Category: "System", Handler: func() {
			openExitDialog(state, nil)
		}},

		// 2. 3D Model Controls (Tab 0: [3D] Mascot)
		{Label: "3D: Toggle Auto-Rotation", Detail: "r / Space", Category: "3D Graphics", Handler: func() {
			state.ActiveTab = 0
			state.AutoRotate = !state.AutoRotate
			addLog(state, fmt.Sprintf("Auto-rotate: %v", state.AutoRotate))
		}},
		{Label: "3D Mode: ASCII Characters", Detail: "m", Category: "3D Graphics", Handler: func() {
			state.ActiveTab = 0
			state.RenderModeIndex = 0
			state.AsciiMode = widgets.ModeASCII
			addLog(state, "Mode: ASCII")
		}},
		{Label: "3D Mode: Half-Block Dual Pixel", Detail: "m", Category: "3D Graphics", Handler: func() {
			state.ActiveTab = 0
			state.RenderModeIndex = 1
			state.AsciiMode = widgets.ModeBlock
			addLog(state, "Mode: Half-Block")
		}},
		{Label: "3D Mode: Braille 8-Dot Matrix", Detail: "m", Category: "3D Graphics", Handler: func() {
			state.ActiveTab = 0
			state.RenderModeIndex = 2
			state.AsciiMode = widgets.ModeBraille
			addLog(state, "Mode: Braille")
		}},
		{Label: "3D: Toggle Specular Highlights", Category: "3D Graphics", Handler: func() {
			state.ActiveTab = 0
			state.Specular = !state.Specular
			addLog(state, fmt.Sprintf("Specular: %v", state.Specular))
		}},
		{Label: "3D: Toggle Natural Colors", Category: "3D Graphics", Handler: func() {
			state.ActiveTab = 0
			state.NaturalColors = !state.NaturalColors
			addLog(state, fmt.Sprintf("Natural Colors: %v", state.NaturalColors))
		}},
		{Label: "3D: Zoom In (+)", Detail: "+", Category: "3D Graphics", Handler: func() {
			state.ActiveTab = 0
			state.Scale = math.Min(15.0, state.Scale+0.5)
			addLog(state, fmt.Sprintf("3D Scale: %.1f", state.Scale))
		}},
		{Label: "3D: Zoom Out (-)", Detail: "-", Category: "3D Graphics", Handler: func() {
			state.ActiveTab = 0
			state.Scale = math.Max(2.0, state.Scale-0.5)
			addLog(state, fmt.Sprintf("3D Scale: %.1f", state.Scale))
		}},

		// 3. TreeView (Tab 1: TreeView)
		{Label: "TreeView: Toggle Hierarchy Guides", Category: "TreeView", Handler: func() {
			state.ActiveTab = 1
			state.TreeShowGuides = !state.TreeShowGuides
			addLog(state, fmt.Sprintf("Guides: %v", state.TreeShowGuides))
		}},
		{Label: "TreeView: Expand All Folders", Category: "TreeView", Handler: func() {
			state.ActiveTab = 1
			state.TreeState.Expand("root")
			state.TreeState.Expand("core")
			state.TreeState.Expand("widgets")
			state.TreeState.Expand("examples")
			state.TreeState.Expand("benchmarks")
			addLog(state, "Expanded all tree nodes")
		}},
		{Label: "TreeView: Collapse Subfolders", Category: "TreeView", Handler: func() {
			state.ActiveTab = 1
			state.TreeState.Collapse("core")
			state.TreeState.Collapse("widgets")
			state.TreeState.Collapse("examples")
			state.TreeState.Collapse("benchmarks")
			addLog(state, "Collapsed subfolders")
		}},

		// 4. Images (Tab 2: Native Images)
		{Label: "Image: Select Apple Graphic (256x256)", Category: "Images", Handler: func() {
			state.ActiveTab = 2
			state.ActiveImageIdx = 0
			addLog(state, "Asset: Apple Graphic")
		}},
		{Label: "Image: Select Profile Avatar (512x512)", Category: "Images", Handler: func() {
			state.ActiveTab = 2
			state.ActiveImageIdx = 1
			addLog(state, "Asset: Profile Avatar")
		}},
		{Label: "Image: Toggle Circular Avatar Mask", Detail: "c", Category: "Images", Handler: func() {
			state.ActiveTab = 2
			state.ImageCircleMask = !state.ImageCircleMask
			addLog(state, fmt.Sprintf("Circle Mask: %v", state.ImageCircleMask))
		}},
		{Label: "Image: Toggle Force Half-Block Override", Category: "Images", Handler: func() {
			state.ActiveTab = 2
			state.ImageHalfBlock = !state.ImageHalfBlock
			addLog(state, fmt.Sprintf("Half-Block: %v", state.ImageHalfBlock))
		}},
		{Label: "Image: Toggle Aspect Ratio Compensation", Category: "Images", Handler: func() {
			state.ActiveTab = 2
			state.ImageAspectCorrect = !state.ImageAspectCorrect
			addLog(state, fmt.Sprintf("Aspect Compensation: %v", state.ImageAspectCorrect))
		}},

		// 5. Telemetry (Tab 3: Telemetry)
		{Label: "Telemetry: Toggle Live High-Frequency Metric Sampling", Category: "Telemetry", Handler: func() {
			state.ActiveTab = 3
			state.LiveTelemetry = !state.LiveTelemetry
			addLog(state, fmt.Sprintf("Telemetry sampling: %v", state.LiveTelemetry))
		}},
		{Label: "Telemetry: Toggle Zero-Allocations Guard", Category: "Telemetry", Handler: func() {
			state.ActiveTab = 3
			state.ZeroAllocGuard = !state.ZeroAllocGuard
			addLog(state, fmt.Sprintf("Zero-alloc guard: %v", state.ZeroAllocGuard))
		}},

		// 6. Themes & Settings (Tab 4: Settings)
		{Label: "Theme: Citrus Limoni", Category: "Themes", Handler: func() {
			state.ActiveTab = 4
			state.ThemeIndex = 0
			addLog(state, "Theme: Citrus Limoni")
		}},
		{Label: "Theme: Cyberpunk Neon", Category: "Themes", Handler: func() {
			state.ActiveTab = 4
			state.ThemeIndex = 1
			addLog(state, "Theme: Cyberpunk Neon")
		}},
		{Label: "Theme: Nord Frost", Category: "Themes", Handler: func() {
			state.ActiveTab = 4
			state.ThemeIndex = 2
			addLog(state, "Theme: Nord Frost")
		}},
		{Label: "Theme: Emerald Forest", Category: "Themes", Handler: func() {
			state.ActiveTab = 4
			state.ThemeIndex = 3
			addLog(state, "Theme: Emerald Forest")
		}},

		// 7. Settings & System (Tab 4: Settings)
		{Label: "Settings: Toggle SGR Mouse Tracking", Category: "Settings", Handler: func() {
			state.ActiveTab = 4
			state.SGRMouseTracking = !state.SGRMouseTracking
			addLog(state, fmt.Sprintf("SGR Mouse: %v", state.SGRMouseTracking))
		}},
		{Label: "Settings: Toggle Hardware Diff Engine", Category: "Settings", Handler: func() {
			state.ActiveTab = 4
			state.DiffEngineActive = !state.DiffEngineActive
			addLog(state, fmt.Sprintf("Diff Engine: %v", state.DiffEngineActive))
		}},
		{Label: "Settings: Toggle Smooth 60 FPS Engine", Category: "Settings", Handler: func() {
			state.ActiveTab = 4
			state.Smooth60FPS = !state.Smooth60FPS
			addLog(state, fmt.Sprintf("60 FPS: %v", state.Smooth60FPS))
		}},
		{Label: "Settings: Toggle Semantic Accessibility Tree", Category: "Settings", Handler: func() {
			state.ActiveTab = 4
			state.AccessibilityActive = !state.AccessibilityActive
			addLog(state, fmt.Sprintf("A11y: %v", state.AccessibilityActive))
		}},
	}

	state.CmdPalette.AllItems = cmdItems
	state.CmdPalette.Filtered = widgets.FuzzyFilter("", cmdItems)
}

func openExitDialog(state *AppState, term *terminal.Terminal) {
	state.ShowExitDialog = true
	state.ModalOffsetX = 0
	state.ModalOffsetY = 0
	state.ExitDialogSelectedBtn = 1 // Default to "No" for safety
	if state.ExitDialogAnim == nil {
		state.ExitDialogAnim = animation.NewFloat(0.0)
	}
	state.ExitDialogAnim.AnimateTo(1.0, 220*time.Millisecond, animation.EaseOutCubic)
	if term != nil && term.FocusManager() != nil {
		term.FocusManager().SetFocused("exit_dialog_btn_1")
	}
}

func closeExitDialog(state *AppState, term *terminal.Terminal) {
	if state.ExitDialogAnim != nil {
		state.ExitDialogAnim.AnimateTo(0.0, 180*time.Millisecond, animation.EaseInCubic)
	} else {
		state.ShowExitDialog = false
		if term != nil {
			term.ForceFullRedraw()
		}
	}
}

func handleKey(key driver.KeyEvent, state *AppState) {
	// If Command Palette is open, feed keys directly to it
	if state.CmdPalette != nil && state.CmdPalette.IsOpen {
		if state.CmdPalette.HandleKey(key) {
			return
		}
	}

	// Toggle Command Palette with Ctrl+P or '/'
	if (key.Type == driver.KeyRune && key.Ch == 'p' && key.Ctrl) || (key.Type == driver.KeyRune && key.Ch == '/' && !state.ShowExitDialog) {
		state.CmdPalette.Toggle()
		return
	}

	// Esc key
	if key.Type == driver.KeyEsc {
		if state.CmdPalette != nil && state.CmdPalette.IsOpen {
			state.CmdPalette.Close()
			return
		}
		if state.ShowExitDialog {
			closeExitDialog(state, nil)
			return
		}
		openExitDialog(state, nil)
		return
	}

	// If Exit Dialog is open:
	if state.ShowExitDialog {
		switch key.Type {
		case driver.KeyArrowLeft:
			state.ExitDialogSelectedBtn = 0
			return
		case driver.KeyArrowRight:
			state.ExitDialogSelectedBtn = 1
			return
		case driver.KeyTab:
			if state.ExitDialogSelectedBtn == 0 {
				state.ExitDialogSelectedBtn = 1
			} else {
				state.ExitDialogSelectedBtn = 0
			}
			return
		case driver.KeyEnter, driver.KeySpace:
			if state.ExitDialogSelectedBtn == 0 {
				state.ExitRequested = true
			} else {
				closeExitDialog(state, nil)
			}
			return
		case driver.KeyEsc:
			closeExitDialog(state, nil)
			return
		case driver.KeyRune:
			if key.Ch == 'y' || key.Ch == 'Y' {
				state.ExitRequested = true
				return
			}
			if key.Ch == 'n' || key.Ch == 'N' {
				closeExitDialog(state, nil)
				return
			}
		}
		return
	}

	// 'q' or 'Q' or '6' opens exit confirmation dialog
	if key.Type == driver.KeyRune && (key.Ch == 'q' || key.Ch == 'Q' || key.Ch == '6') {
		openExitDialog(state, nil)
		return
	}

	// If on TreeView tab, delegate navigation keys to TreeView
	if state.ActiveTab == 1 {
		if state.TreeState.HandleKey(key, state.TreeRoots) {
			addLog(state, fmt.Sprintf("Tree: %s", state.TreeState.SelectedID))
			return
		}
	}

	switch key.Type {
	case driver.KeyTab:
		state.ActiveTab = (state.ActiveTab + 1) % 5
		addLog(state, fmt.Sprintf("Tab: %s", state.Tabs[state.ActiveTab]))
	case driver.KeyArrowRight:
		state.ActiveTab = (state.ActiveTab + 1) % 5
	case driver.KeyArrowLeft:
		state.ActiveTab = (state.ActiveTab - 1 + 5) % 5
	case driver.KeyArrowUp:
		state.RotX += 10.0
	case driver.KeyArrowDown:
		state.RotX -= 10.0
	case driver.KeySpace:
		switch state.ActiveTab {
		case 0:
			state.AutoRotate = !state.AutoRotate
			addLog(state, fmt.Sprintf("Auto-rotate: %v", state.AutoRotate))
		case 1:
			if state.TreeState.SelectedID != "" {
				state.TreeState.Toggle(state.TreeState.SelectedID, true)
			}
		case 2:
			state.ImageCircleMask = !state.ImageCircleMask
			addLog(state, fmt.Sprintf("Circle Mask: %v", state.ImageCircleMask))
		}
	case driver.KeyRune:
		switch key.Ch {
		case '1', '2', '3', '4', '5':
			idx := int(key.Ch - '1')
			if idx < 5 {
				state.ActiveTab = idx
				addLog(state, fmt.Sprintf("Jumped to Tab: %s", state.Tabs[idx]))
			}
		case 'm', 'M':
			state.RenderModeIndex = (state.RenderModeIndex + 1) % 3
			switch state.RenderModeIndex {
			case 0:
				state.AsciiMode = widgets.ModeASCII
			case 1:
				state.AsciiMode = widgets.ModeBlock
			case 2:
				state.AsciiMode = widgets.ModeBraille
			}
			addLog(state, fmt.Sprintf("3D Mode: %d", state.RenderModeIndex))
		case 'c', 'C':
			state.ImageCircleMask = !state.ImageCircleMask
			addLog(state, fmt.Sprintf("Image Circle Mask: %v", state.ImageCircleMask))
		case 't', 'T':
			state.ThemeIndex = (state.ThemeIndex + 1) % len(themes)
			addLog(state, fmt.Sprintf("Theme: %s", themes[state.ThemeIndex].Name))
		case 'r', 'R':
			state.AutoRotate = !state.AutoRotate
			addLog(state, fmt.Sprintf("Auto-rotate: %v", state.AutoRotate))
		case '+', '=':
			state.Scale = math.Min(15.0, state.Scale+0.5)
		case '-', '_':
			state.Scale = math.Max(2.0, state.Scale-0.5)
		}
	}
}

func addLog(state *AppState, msg string) {
	t := time.Now().Format("15:04:05")
	entry := fmt.Sprintf("[%s] %s", t, msg)
	state.Logs = append(state.Logs, entry)
	if len(state.Logs) > 8 {
		state.Logs = state.Logs[len(state.Logs)-8:]
	}
}

func renderFrame(term *terminal.Terminal, state *AppState) {
	theme := themes[state.ThemeIndex]

	term.Draw(func(f *terminal.Frame) {
		rootArea := f.Buffer.Area
		if rootArea.Width < 70 || rootArea.Height < 20 {
			f.Buffer.SetString(2, 2, "Please enlarge terminal window (minimum 80x24 recommended).", cell.Style{Fg: theme.Accent})
			return
		}

		// 1. Root Vertical Flex Layout:
		// - Header (Fixed 3 rows)
		// - Body (Fill)
		// - Footer (Fixed 1 row)
		rootChunks := layout.VBox(rootArea,
			layout.Fixed(3), // Top Header
			layout.Fill(),   // Main Body
			layout.Fixed(1), // Bottom Status Bar
		)
		headerArea := rootChunks[0]
		bodyArea := rootChunks[1]
		footerArea := rootChunks[2]

		// 2. HEADER BLOCK
		headerBlock := widgets.Block{
			Title:          " LIMONI TUI ENGINE • Flagship Interactive Showcase ",
			TitleAlignment: widgets.AlignLeft,
			Borders:        widgets.BorderAll,
			BorderSymbols:  widgets.SymbolsRounded,
			BorderStyle:    cell.Style{Fg: theme.Primary},
			TitleStyle:     cell.Style{Fg: theme.Accent, Modifier: cell.ModifierBold},
			Child:          label{text: " Hardware ANSI Double-Buffer Diffing • Zero-Alloc Hot Loops • 3D GLB Pipeline ", style: cell.Style{Fg: cell.NewColorRGB(220, 225, 235)}},
		}
		f.RenderWidget(headerBlock, headerArea)

		// Header telemetry badges and search button on top-right
		if headerArea.Width > 75 {
			searchBtnArea := cell.NewRect(headerArea.X+headerArea.Width-50, headerArea.Y+1, 15, 1)
			f.Buffer.SetString(searchBtnArea.X, searchBtnArea.Y, " Search [^P] ", cell.Style{Fg: cell.NewColorRGB(0, 0, 0), Bg: theme.Accent, Modifier: cell.ModifierBold})
			registerTargetClick(f, searchBtnArea, func(ev driver.MouseEvent) {
				state.CmdPalette.Toggle()
			})

			fpsStr := fmt.Sprintf(" FPS: %4.1f ", state.FPS)
			f.Buffer.SetString(headerArea.X+headerArea.Width-32, headerArea.Y+1, fpsStr, cell.Style{Fg: cell.NewColorRGB(0, 0, 0), Bg: theme.Secondary, Modifier: cell.ModifierBold})

			themeBadge := fmt.Sprintf(" %s ", theme.Name)
			if len(themeBadge) > 16 {
				themeBadge = themeBadge[:16] + " "
			}
			f.Buffer.SetString(headerArea.X+headerArea.Width-18, headerArea.Y+1, themeBadge, cell.Style{Fg: cell.NewColorRGB(0, 0, 0), Bg: theme.Accent, Modifier: cell.ModifierBold})
		}

		// 3. BODY HORIZONTAL FLEX LAYOUT:
		// - Left Sidebar Menu (Fixed 24 columns)
		// - Right Main Content Viewport (Fill)
		bodyChunks := layout.HBoxWithGap(bodyArea, 1,
			layout.Fixed(24),
			layout.Fill(),
		)
		sidebarArea := bodyChunks[0]
		contentArea := bodyChunks[1]

		// Draw Left Sidebar Navigation Buttons
		drawSidebarMenu(f, sidebarArea, state, theme)

		// Draw Right Main Content based on active tab
		switch state.ActiveTab {
		case 0:
			drawTab3D(f, contentArea, state, theme)
		case 1:
			drawTabTreeView(f, contentArea, state, theme)
		case 2:
			drawTabImage(f, contentArea, state, theme)
		case 3:
			drawTabTelemetry(f, contentArea, state, theme)
		case 4:
			drawTabSettings(f, contentArea, state, theme)
		}

		// 4. FOOTER STATUS BAR
		drawFooter(f, footerArea, state, theme)

		// 5. MODAL OVERLAY: EXIT CONFIRMATION DIALOG
		if state.ShowExitDialog {
			dialogW, dialogH := uint16(48), uint16(9)
			dialogArea := terminal.CenterRect(f.Buffer.Area, dialogW, dialogH)
			state.ModalOffsetX, state.ModalOffsetY = clampDialogOffset(f.Buffer.Area, dialogW, dialogH, state.ModalOffsetX, state.ModalOffsetY)
			dialogArea.X = uint16(int(dialogArea.X) + state.ModalOffsetX)
			dialogArea.Y = uint16(int(dialogArea.Y) + state.ModalOffsetY)

			progress := 1.0
			if state.ExitDialogAnim != nil {
				progress = state.ExitDialogAnim.Value()
			}
			animatedArea := terminal.ScaleRect(dialogArea, progress)

			if progress <= 0.001 && state.ExitDialogAnim != nil && !state.ExitDialogAnim.IsAnimating() {
				state.ShowExitDialog = false
				if term.FocusManager() != nil {
					term.FocusManager().SetFocused("")
				}
				term.ForceFullRedraw()
			} else {
				f.RegisterModal("exit_dialog", dialogArea, func() {
					closeExitDialog(state, term)
				})

				// Title bar area for dragging
				titleBarArea := cell.NewRect(animatedArea.X, animatedArea.Y, animatedArea.Width, 1)
				registerTargetClick(f, titleBarArea, func(ev driver.MouseEvent) {
					if ev.Button != driver.MouseLeft {
						return
					}
					state.IsDraggingModal = true
					state.DragMouseStartX = int(ev.X)
					state.DragMouseStartY = int(ev.Y)
					state.ModalDragBaseX = state.ModalOffsetX
					state.ModalDragBaseY = state.ModalOffsetY
					f.CaptureMouse(func(dragEv driver.MouseEvent) {
						if dragEv.Button == driver.MouseRelease {
							state.IsDraggingModal = false
							return
						}
						if dragEv.Drag {
							state.ModalOffsetX, state.ModalOffsetY = clampDialogOffset(f.Buffer.Area, dialogW, dialogH,
								state.ModalDragBaseX+int(dragEv.X)-state.DragMouseStartX,
								state.ModalDragBaseY+int(dragEv.Y)-state.DragMouseStartY)
						}
					})
				})

				if animatedArea.Width >= 6 && animatedArea.Height >= 3 {
					exitDialog := widgets.Dialog{
						ID:          "exit_dialog",
						Title:       " ▲ SYSTEM EXIT ",
						Message:     "Are you sure you want to exit the application?",
						SubMessage:  "The session and all unsaved state will be terminated.",
						Style:       cell.Style{Fg: cell.NewColorRGB(220, 220, 220), Bg: cell.NewColorRGB(25, 25, 25)},
						HeaderStyle: cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: cell.NewColorRGB(220, 60, 60)},
						BorderStyle: cell.Style{Fg: cell.NewColorRGB(220, 60, 60)},
						ButtonStyle: cell.Style{Fg: cell.NewColorRGB(200, 205, 215), Bg: cell.NewColorRGB(45, 50, 60)},
						ButtonFocusedStyle: cell.Style{
							Fg:       cell.NewColorRGB(255, 255, 255),
							Bg:       theme.Accent,
							Modifier: cell.ModifierBold,
						},
						Shadow:        true,
						FocusedButton: state.ExitDialogSelectedBtn,
						OnButtonHover: func(idx int) {
							state.ExitDialogSelectedBtn = idx
						},
						Buttons: []widgets.DialogButton{
							{
								Text: "Yes",
								FocusedStyle: cell.Style{
									Fg:       cell.NewColorRGB(255, 255, 255),
									Bg:       cell.NewColorRGB(220, 55, 55),
									Modifier: cell.ModifierBold,
								},
								Handler: func() {
									state.ExitRequested = true
								},
							},
							{
								Text: "No",
								FocusedStyle: cell.Style{
									Fg:       cell.NewColorRGB(255, 255, 255),
									Bg:       theme.Accent,
									Modifier: cell.ModifierBold,
								},
								Handler: func() {
									closeExitDialog(state, term)
								},
							},
						},
					}
					f.BeginFocusScope("exit_dialog")
					f.RenderWidget(exitDialog, animatedArea)
				}
			}
		}

		// 6. COMMAND PALETTE / FUZZY SEARCH OVERLAY
		if state.CmdPalette != nil && state.CmdPalette.IsOpen {
			palette := widgets.CommandPalette{
				ID:       "command_palette",
				State:    state.CmdPalette,
				Position: &widgets.CommandPalettePosition{Top: 4},
			}
			palArea := palette.DebugArea(f.Buffer.Area)
			if palArea.Width > 0 && palArea.Height > 0 {
				f.RegisterModal("command_palette", palArea, func() {
					state.CmdPalette.Close()
				})
			}
			f.BeginLayer("command_palette")
			f.RenderWidget(palette, f.Buffer.Area)
			f.EndLayer()
		}
	})
}

func drawSidebarMenu(f *terminal.Frame, area cell.Rect, state *AppState, theme Theme) {
	// Sidebar menu buttons: 6 fixed height-3 blocks
	menuLay := layout.VBoxWithGap(area, 0,
		layout.Fixed(3), // 1. [3D] Mascot
		layout.Fixed(3), // 2. TreeView
		layout.Fixed(3), // 3. Native Images
		layout.Fixed(3), // 4. Telemetry
		layout.Fixed(3), // 5. Settings
		layout.Fixed(3), // 6. Exit
		layout.Fill(),   // Empty spacer
	)

	drawNavBtn := func(chunkIdx int, tabIdx int, title string) {
		btnArea := menuLay[chunkIdx]
		isActive := (state.ActiveTab == tabIdx)

		borderCol := theme.Muted
		titleStyle := cell.Style{Fg: theme.Muted}

		if isActive {
			borderCol = theme.Primary
			titleStyle = cell.Style{Fg: theme.Primary, Modifier: cell.ModifierBold}
		}

		btn := widgets.Block{
			Borders:        widgets.BorderAll,
			BorderSymbols:  widgets.SymbolsRounded,
			BorderStyle:    cell.Style{Fg: borderCol},
			Title:          title,
			TitleAlignment: widgets.AlignCenter,
			TitleStyle:     titleStyle,
		}
		f.RenderWidget(btn, btnArea)

		registerTargetClick(f, btnArea, func(ev driver.MouseEvent) {
			if tabIdx == 5 {
				// Exit Confirmation Dialog
				openExitDialog(state, nil)
			} else {
				state.ActiveTab = tabIdx
				addLog(state, fmt.Sprintf("Clicked tab: %s", state.Tabs[tabIdx]))
			}
		})
	}

	drawNavBtn(0, 0, state.Tabs[0])
	drawNavBtn(1, 1, state.Tabs[1])
	drawNavBtn(2, 2, state.Tabs[2])
	drawNavBtn(3, 3, state.Tabs[3])
	drawNavBtn(4, 4, state.Tabs[4])
	drawNavBtn(5, 5, state.Tabs[5])
}

func drawFooter(f *terminal.Frame, area cell.Rect, state *AppState, theme Theme) {
	// Left: Navigation hotkeys
	shortcuts := "[Tab/Click] Switch Tab │ [^P] Search │ [Space] Toggle │ [1-5] Tabs │ [m] 3D Mode │ [t] Theme │ [q] Exit"
	f.Buffer.SetString(area.X+1, area.Y, shortcuts, cell.Style{Fg: theme.Muted})

	// Right: Mouse pointer position & theme
	rightStr := fmt.Sprintf("Cursor: (%2d, %2d) │ %s ", state.LastMousePos.X, state.LastMousePos.Y, theme.Name)
	if uint16(len(rightStr)) < area.Width {
		startX := area.X + area.Width - uint16(len(rightStr)) - 1
		f.Buffer.SetString(startX, area.Y, rightStr, cell.Style{Fg: theme.Accent, Modifier: cell.ModifierBold})
	}
}
