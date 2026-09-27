package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/graphics"
)

func TestViewer3DStateKeys(t *testing.T) {
	s := &Viewer3DState{}
	for _, k := range []driver.KeyType{driver.KeyArrowRight, driver.KeyArrowRight, driver.KeyArrowUp} {
		if !s.HandleKey(driver.KeyEvent{Type: k}) {
			t.Fatalf("key %v not handled", k)
		}
	}
	if s.RotY != 10 || s.RotX != 355 {
		t.Errorf("RotX %v RotY %v after →→↑, want 355 10", s.RotX, s.RotY)
	}
	s.HandleKey(driver.KeyEvent{Type: driver.KeyRune, Ch: '-'})
	if s.Distance != 0.25 {
		t.Errorf("distance %v after -, want 0.25", s.Distance)
	}
	if s.HandleKey(driver.KeyEvent{Type: driver.KeyRune, Ch: 'x'}) {
		t.Error("x was reported as handled")
	}
	var none *Viewer3DState
	if none.HandleKey(driver.KeyEvent{Type: driver.KeyArrowLeft}) {
		t.Error("a nil state handled a key")
	}
}

// Zooming in stops once the camera would pass the closest distance, so
// zooming back out responds straight away.
func TestViewer3DStateZoomClamp(t *testing.T) {
	s := &Viewer3DState{}
	v := &Viewer3D{Model: graphics.NewCube(2), State: s}
	area := cell.NewRect(0, 0, 20, 10)
	v.Draw(cell.NewContext(area, cell.Style{}), buffer.NewBuffer(area))
	for i := 0; i < 100; i++ {
		s.HandleKey(driver.KeyEvent{Type: driver.KeyRune, Ch: '+'})
	}
	if want := orbitMinDist - 3.5; s.Distance != want {
		t.Fatalf("distance %v after zooming all the way in, want %v", s.Distance, want)
	}
	s.HandleKey(driver.KeyEvent{Type: driver.KeyRune, Ch: '-'})
	if want := orbitMinDist - 3.5 + orbitZoomStep; s.Distance != want {
		t.Errorf("distance %v after one step out, want %v", s.Distance, want)
	}
}

// A state's angles draw the same picture as the same angles on the viewer.
func TestViewer3DStateAddsToAngles(t *testing.T) {
	area := cell.NewRect(0, 0, 30, 15)
	draw := func(v *Viewer3D) string {
		buf := buffer.NewBuffer(area)
		v.Draw(cell.NewContext(area, cell.Style{}), buf)
		out := ""
		for y := uint16(0); y < area.Height; y++ {
			out += rowOf(buf, y, area.Width) + "\n"
		}
		return out
	}
	want := draw(&Viewer3D{Model: graphics.NewCube(2), RotX: 30, RotY: 40})
	got := draw(&Viewer3D{Model: graphics.NewCube(2), RotX: 10, RotY: 40, State: &Viewer3DState{RotX: 20}})
	if got != want {
		t.Errorf("RotX 10 plus state 20 drew differently from RotX 30:\n%s\nwant:\n%s", got, want)
	}
}

// Pressing a viewer with a State still focuses it: the orbit region sits on
// top of the click-to-focus one.
func TestViewer3DStatePressFocuses(t *testing.T) {
	area := cell.NewRect(0, 0, 20, 10)
	var handler func(driver.MouseEvent)
	focused := ""
	ctx := cell.NewContext(area, cell.Style{})
	ctx.RegisterMouse = func(_ cell.Rect, h func(driver.MouseEvent)) { handler = h }
	ctx.CaptureMouse = func(func(driver.MouseEvent)) {}
	ctx.SetFocus = func(id string) { focused = id }
	v := &Viewer3D{ID: "model", Model: graphics.NewCube(2), State: &Viewer3DState{}}
	v.Draw(ctx, buffer.NewBuffer(area))
	handler(driver.MouseEvent{Button: driver.MouseLeft, X: 3, Y: 3})
	if focused != "model" {
		t.Errorf("focused %q after pressing the viewer, want model", focused)
	}
}

// A viewer already closer than the zoom floor keeps its own distance when a
// State is attached, and zooming in does not push it back out.
func TestViewer3DStateKeepsACloseViewer(t *testing.T) {
	s := &Viewer3DState{}
	v := &Viewer3D{Model: graphics.NewCube(2), Distance: 0.3, State: s}
	area := cell.NewRect(0, 0, 20, 10)
	v.Draw(cell.NewContext(area, cell.Style{}), buffer.NewBuffer(area))
	if s.Distance != 0 {
		t.Fatalf("state distance %v after drawing a viewer at 0.3, want 0", s.Distance)
	}
	s.HandleKey(driver.KeyEvent{Type: driver.KeyRune, Ch: '+'})
	if s.Distance > 0 {
		t.Errorf("zooming in moved the camera out: state distance %v", s.Distance)
	}
}

// Zooming in before the first frame is clamped once the viewer is drawn, so
// zooming out afterwards responds straight away.
func TestViewer3DStateZoomBeforeFirstFrame(t *testing.T) {
	s := &Viewer3DState{}
	for i := 0; i < 100; i++ {
		s.HandleKey(driver.KeyEvent{Type: driver.KeyRune, Ch: '+'})
	}
	v := &Viewer3D{Model: graphics.NewCube(2), State: s}
	area := cell.NewRect(0, 0, 20, 10)
	v.Draw(cell.NewContext(area, cell.Style{}), buffer.NewBuffer(area))
	if want := orbitMinDist - 3.5; s.Distance != want {
		t.Fatalf("state distance %v after the first frame, want %v", s.Distance, want)
	}
}
