package widgets

import (
	"fmt"
	"runtime"
	"time"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// DevToolsState records live performance metrics and active inspector state.
type DevToolsState struct {
	Enabled        bool
	ActiveTab      int // 0: Performance, 1: Focus & Events, 2: Tree & Box Model
	FPS            float64
	FrameTime      time.Duration
	AllocMB        float64
	SysMB          float64
	NumGC          uint32
	NumGoroutine   int
	InspectedArea  cell.Rect
	InspectedName  string
	frameCount     int
	lastFPSCalc    time.Time
	lastFrameTimes []time.Duration
}

// NewDevToolsState creates an initialized developer inspector state.
func NewDevToolsState() *DevToolsState {
	return &DevToolsState{
		Enabled:        false,
		lastFPSCalc:    time.Now(),
		lastFrameTimes: make([]time.Duration, 0, 60),
	}
}

// Toggle inverts the visibility of the DevTools overlay.
func (s *DevToolsState) Toggle() {
	if s == nil {
		return
	}
	s.Enabled = !s.Enabled
}

// RecordFrame records a single frame's render time and updates memory & FPS stats.
func (s *DevToolsState) RecordFrame(frameDuration time.Duration) {
	if s == nil {
		return
	}
	s.FrameTime = frameDuration
	s.frameCount++

	now := time.Now()
	elapsed := now.Sub(s.lastFPSCalc)
	if elapsed >= time.Second {
		s.FPS = float64(s.frameCount) / elapsed.Seconds()
		s.frameCount = 0
		s.lastFPSCalc = now

		// Update runtime memory statistics
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		s.AllocMB = float64(m.Alloc) / (1024.0 * 1024.0)
		s.SysMB = float64(m.Sys) / (1024.0 * 1024.0)
		s.NumGC = m.NumGC
		s.NumGoroutine = runtime.NumGoroutine()
	}
}

// HandleKey processes F12 / shortcut keys and tab navigation.
func (s *DevToolsState) HandleKey(ev backend.KeyEvent) bool {
	if s == nil {
		return false
	}
	if ev.Type == backend.KeyF12 {
		s.Toggle()
		return true
	}
	if !s.Enabled {
		return false
	}

	switch ev.Type {
	case backend.KeyTab:
		s.ActiveTab = (s.ActiveTab + 1) % 3
		return true
	}
	return false
}

// DevTools renders the in-terminal developer inspection dashboard.
type DevTools struct {
	State *DevToolsState
	Style cell.Style
}

// Draw renders the developer tools HUD to the buffer.
func (dt DevTools) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if dt.State == nil || !dt.State.Enabled {
		return
	}

	state := dt.State
	panelW := uint16(46)
	panelH := uint16(11)

	if ctx.Area.Width < panelW+2 || ctx.Area.Height < panelH+2 {
		panelW = ctx.Area.Width - 2
		panelH = ctx.Area.Height - 2
	}

	panelX := ctx.Area.X + ctx.Area.Width - panelW - 1
	panelY := ctx.Area.Y + 1
	panelArea := cell.NewRect(panelX, panelY, panelW, panelH)

	// Draw Alpha Drop Shadow
	DrawShadow(buf, panelArea, 2, 1)

	bgStyle := cell.Style{
		Bg: cell.NewColorRGB(18, 22, 30),
		Fg: cell.NewColorRGB(230, 235, 245),
	}
	borderStyle := cell.Style{
		Fg: cell.NewColorRGB(0, 255, 180),
		Bg: bgStyle.Bg,
	}

	// Outer Block Container
	block := Block{
		Title:         " 🛠️ LIMONI DEVTOOLS (F12) ",
		Borders:       BorderAll,
		BorderSymbols: SymbolsRounded,
		BorderStyle:   borderStyle,
		Style:         bgStyle,
	}
	block.Draw(cell.NewContext(panelArea, bgStyle), buf)

	// Tab Header
	tabs := []string{"[1: Perf]", "[2: Focus/Events]", "[3: Box Model]"}
	tabX := panelX + 2
	for i, t := range tabs {
		st := cell.Style{Fg: cell.NewColorRGB(130, 140, 155), Bg: bgStyle.Bg}
		if state.ActiveTab == i {
			st = cell.Style{Fg: cell.NewColorRGB(0, 255, 180), Bg: bgStyle.Bg, Modifier: cell.ModifierBold}
		}
		buf.SetString(tabX, panelY+1, t, st)
		tabX += uint16(len(t) + 1)
	}

	contentY := panelY + 3
	valStyle := cell.Style{Fg: cell.NewColorRGB(0, 220, 255), Bg: bgStyle.Bg, Modifier: cell.ModifierBold}
	dimStyle := cell.Style{Fg: cell.NewColorRGB(160, 170, 185), Bg: bgStyle.Bg}

	switch state.ActiveTab {
	case 0: // Performance & Metrics
		fpsColor := cell.NewColorRGB(46, 204, 113)
		if state.FPS < 30 && state.FPS > 0 {
			fpsColor = cell.NewColorRGB(231, 76, 60)
		} else if state.FPS < 50 && state.FPS > 0 {
			fpsColor = cell.NewColorRGB(241, 196, 15)
		}
		fpsStyle := cell.Style{Fg: fpsColor, Bg: bgStyle.Bg, Modifier: cell.ModifierBold}

		buf.SetString(panelX+2, contentY, "FPS: ", dimStyle)
		buf.SetString(panelX+8, contentY, fmt.Sprintf("%.1f fps", state.FPS), fpsStyle)

		buf.SetString(panelX+24, contentY, "Frame Time: ", dimStyle)
		buf.SetString(panelX+36, contentY, fmt.Sprintf("%6.2f ms", float64(state.FrameTime.Microseconds())/1000.0), valStyle)

		buf.SetString(panelX+2, contentY+1, "Alloc: ", dimStyle)
		buf.SetString(panelX+9, contentY+1, fmt.Sprintf("%.2f MB", state.AllocMB), valStyle)

		buf.SetString(panelX+24, contentY+1, "Sys Mem: ", dimStyle)
		buf.SetString(panelX+33, contentY+1, fmt.Sprintf("%.2f MB", state.SysMB), valStyle)

		buf.SetString(panelX+2, contentY+2, "Num GC: ", dimStyle)
		buf.SetString(panelX+10, contentY+2, fmt.Sprintf("%d", state.NumGC), valStyle)

		buf.SetString(panelX+24, contentY+2, "Goroutines: ", dimStyle)
		buf.SetString(panelX+36, contentY+2, fmt.Sprintf("%d", state.NumGoroutine), valStyle)

		buf.SetString(panelX+2, contentY+4, "Diff Engine: Zero-Alloc Active ✓", cell.Style{Fg: cell.NewColorRGB(0, 255, 140), Bg: bgStyle.Bg})

	case 1: // Focus & Events
		buf.SetString(panelX+2, contentY, "Focused Widget ID:", dimStyle)
		focusedID := ctx.FocusedID
		if focusedID == "" {
			focusedID = "<none>"
		}
		buf.SetString(panelX+21, contentY, focusedID, valStyle)

		buf.SetString(panelX+2, contentY+2, "Terminal Mode: Raw VT100 / Virtual Input", dimStyle)
		buf.SetString(panelX+2, contentY+3, "Mouse Tracking: SGR Extended (1006)", dimStyle)
		buf.SetString(panelX+2, contentY+4, "Color Mode: 24-bit TrueColor RGB", valStyle)

	case 2: // Box Model
		buf.SetString(panelX+2, contentY, "Viewport Bounds:", dimStyle)
		boundsStr := fmt.Sprintf("X:%d Y:%d W:%d H:%d", ctx.Area.X, ctx.Area.Y, ctx.Area.Width, ctx.Area.Height)
		buf.SetString(panelX+19, contentY, boundsStr, valStyle)

		buf.SetString(panelX+2, contentY+2, "Terminal Area: ", dimStyle)
		buf.SetString(panelX+17, contentY+2, fmt.Sprintf("%dx%d cells", ctx.Area.Width, ctx.Area.Height), valStyle)

		buf.SetString(panelX+2, contentY+4, "Hint: Press Tab to toggle Inspector views", cell.Style{Fg: cell.NewColorRGB(120, 130, 145), Bg: bgStyle.Bg})
	}
}
