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
	if !term.Drag(30, 6) {
		t.Fatal("the drag was not routed to the capture")
	}
	// 10 cells right and 4 up at 1.5° per cell.
	if state.RotY != 15 || state.RotX != 354 {
		t.Fatalf("after dragging (20,10)->(30,6): RotX %v RotY %v, want 354 15", state.RotX, state.RotY)
	}

	term.Render(viewer, area)
	if !term.Mouse(driver.MouseEvent{X: 5, Y: 5, Button: driver.MouseScrollUp}) {
		t.Fatal("the wheel was not routed")
	}
	if state.Distance != -0.25 {
		t.Fatalf("distance after one wheel notch in: %v, want -0.25", state.Distance)
	}
}
