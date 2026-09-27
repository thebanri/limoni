package testkit

import (
	"testing"

	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/graphics"
	"github.com/thebanri/limoni/widgets"
)

// Dragging across the viewer through the real frame routing turns the drag
// into rotation, and the wheel into distance.
func TestViewer3DOrbitThroughTheFrame(t *testing.T) {
	state := &widgets.Viewer3DState{}
	viewer := &widgets.Viewer3D{Model: graphics.NewCube(2), State: state}
	term := NewTerminal(40, 20)
	area := cell.NewRect(0, 0, 40, 20)

	term.Render(viewer, area)
	if !term.Click(20, 10) {
		t.Fatal("pressing the viewer was not routed")
	}
	// A drag arrives as several motion events, each measured from the last;
	// the release, wherever it lands, does not move the model.
	for _, p := range [][2]uint16{{25, 8}, {30, 6}} {
		if !term.Mouse(driver.MouseEvent{X: p[0], Y: p[1], Button: driver.MouseLeft, Drag: true}) {
			t.Fatal("the drag was not routed to the capture")
		}
	}
	term.Mouse(driver.MouseEvent{X: 35, Y: 12, Button: driver.MouseRelease})
	// 10 cells right and 4 up at 1.5° per cell.
	if state.RotY != 15 || state.RotX != 354 {
		t.Fatalf("after dragging (20,10)->(30,6): RotX %v RotY %v, want 354 15", state.RotX, state.RotY)
	}

	term.Render(viewer, area)
	for _, notch := range []struct {
		button driver.MouseButton
		want   float64
	}{{driver.MouseScrollUp, -0.25}, {driver.MouseScrollDown, 0}, {driver.MouseScrollDown, 0.25}} {
		if !term.Mouse(driver.MouseEvent{X: 5, Y: 5, Button: notch.button}) {
			t.Fatal("the wheel was not routed")
		}
		if state.Distance != notch.want {
			t.Fatalf("distance %v after a wheel notch, want %v", state.Distance, notch.want)
		}
	}
}
