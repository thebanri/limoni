package widgets

import (
	"testing"
	"time"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

func TestDevTools_ToggleAndRecord(t *testing.T) {
	state := NewDevToolsState()
	if state.Enabled {
		t.Fatal("expected DevTools to be disabled by default")
	}

	state.HandleKey(backend.KeyEvent{Type: backend.KeyF12})
	if !state.Enabled {
		t.Fatal("expected DevTools to be enabled after F12")
	}

	state.RecordFrame(500 * time.Microsecond)
	if state.FrameTime != 500*time.Microsecond {
		t.Errorf("expected FrameTime 500µs, got %v", state.FrameTime)
	}

	// Tab through tabs
	state.HandleKey(backend.KeyEvent{Type: backend.KeyTab})
	if state.ActiveTab != 1 {
		t.Errorf("expected active tab 1, got %d", state.ActiveTab)
	}
}

func TestDevTools_Draw(t *testing.T) {
	state := NewDevToolsState()
	state.Enabled = true
	state.FPS = 60.0
	state.FrameTime = 16 * time.Millisecond

	dt := DevTools{
		State: state,
	}

	area := cell.NewRect(0, 0, 100, 30)
	buf := buffer.NewBuffer(area)
	ctx := cell.NewContext(area, cell.Style{})

	dt.Draw(ctx, buf)

	// Check that top-right corner region has cells drawn
	c := buf.Get(area.Width-10, 2)
	if c == nil {
		t.Fatal("expected non-nil cell for drawn DevTools")
	}
}
