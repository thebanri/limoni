package terminal_test

import (
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/widgets"
)

func memTerm(t *testing.T, w, h uint16) (*terminal.Terminal, *driver.MemoryTerminalIO) {
	t.Helper()
	t.Setenv("LIMONI_PROBE", "0")
	io := driver.NewMemoryTerminalIO(nil, w, h)
	b := driver.NewPortableBackend(io)
	if err := b.Setup(); err != nil {
		t.Fatal(err)
	}
	term, err := terminal.New(b)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = term.Close() })
	return term, io
}

// A click must reach the handler registered for the region it lands in, and
// nothing else.
func TestClickRegionsRouteToTheRightHandler(t *testing.T) {
	term, _ := memTerm(t, 40, 10)
	var hit []string
	if err := term.Draw(func(f *terminal.Frame) {
		f.RegisterClickHandler(cell.NewRect(0, 0, 10, 2), func(driver.MouseEvent) { hit = append(hit, "left") })
		f.RegisterClickHandler(cell.NewRect(20, 0, 10, 2), func(driver.MouseEvent) { hit = append(hit, "right") })
	}); err != nil {
		t.Fatal(err)
	}

	term.RouteMouseEvent(driver.MouseEvent{X: 5, Y: 1, Button: driver.MouseLeft})
	term.RouteMouseEvent(driver.MouseEvent{X: 25, Y: 1, Button: driver.MouseLeft})
	term.RouteMouseEvent(driver.MouseEvent{X: 15, Y: 1, Button: driver.MouseLeft}) // between them

	if strings.Join(hit, ",") != "left,right" {
		t.Errorf("clicks routed to %v", hit)
	}
}

// A captured mouse takes every event until the button is released, which is
// what makes dragging a slider keep working after the pointer leaves it.
// The capture is requested from inside a mouse handler, not while drawing:
// a request left over from Draw is cleared before routing, deliberately.
func TestMouseCaptureTakesEventsUntilRelease(t *testing.T) {
	term, _ := memTerm(t, 40, 10)
	captured := 0
	var frame *terminal.Frame
	if err := term.Draw(func(f *terminal.Frame) {
		frame = f
		f.RegisterClickHandler(cell.NewRect(0, 0, 10, 2), func(ev driver.MouseEvent) {
			f.CaptureMouse(func(driver.MouseEvent) { captured++ })
		})
	}); err != nil {
		t.Fatal(err)
	}

	term.RouteMouseEvent(driver.MouseEvent{X: 1, Y: 1, Button: driver.MouseLeft})
	// Now outside the region entirely: the capture still sees them.
	term.RouteMouseEvent(driver.MouseEvent{X: 39, Y: 9, Button: driver.MouseLeft, Drag: true})
	term.RouteMouseEvent(driver.MouseEvent{X: 39, Y: 9, Button: driver.MouseRelease})
	if captured < 2 {
		t.Errorf("the capture saw %d events, want the drag and the release", captured)
	}

	// After the release the capture is gone and events route normally again.
	before := captured
	term.RouteMouseEvent(driver.MouseEvent{X: 39, Y: 9, Button: driver.MouseLeft, Drag: true})
	if captured != before {
		t.Error("the capture outlived the release")
	}

	// TakeMouseCapture is the hook a custom event loop uses instead.
	frame.CaptureMouse(func(driver.MouseEvent) {})
	if frame.TakeMouseCapture() == nil {
		t.Error("TakeMouseCapture returned nothing after a request")
	}
	if frame.TakeMouseCapture() != nil {
		t.Error("TakeMouseCapture did not clear the request")
	}
}

func TestFrameFocusHelpers(t *testing.T) {
	term, _ := memTerm(t, 40, 10)
	check := false
	if err := term.Draw(func(f *terminal.Frame) {
		f.BeginFocusScope("dialog")
		f.RenderWidget(&widgets.Checkbox{ID: "agree", Checked: &check, Label: "Agree"}, cell.NewRect(0, 0, 20, 1))
		f.EndFocusScope()
		f.SetTheme(widgets.DarkTheme())
	}); err != nil {
		t.Fatal(err)
	}

	fm := term.FocusManager()
	fm.Next()
	if fm.Focused() != "agree" {
		t.Errorf("focus landed on %q, want the checkbox inside the scope", fm.Focused())
	}
}

// The transition state drives the dither fade between screens; turning it
// off must restore a clean, fully-drawn frame.
func TestTransitionStateResetsCleanly(t *testing.T) {
	term, _ := memTerm(t, 20, 5)
	if term.IsTransitionActive() {
		t.Fatal("a fresh terminal is mid-transition")
	}
	term.SetTransitionActive(true)
	term.SetTransitionProgress(0.5)
	if !term.IsTransitionActive() {
		t.Error("SetTransitionActive(true) did not take")
	}
	term.SetTransitionActive(false)
	if term.IsTransitionActive() {
		t.Error("SetTransitionActive(false) did not take")
	}
	if err := term.Draw(func(f *terminal.Frame) {
		f.Buffer.SetString(0, 0, "after", cell.Style{})
	}); err != nil {
		t.Fatal(err)
	}
}

func TestTerminalAccessors(t *testing.T) {
	term, _ := memTerm(t, 30, 8)
	if term.Driver() == nil || term.Backend() == nil {
		t.Error("the terminal should expose the backend it was built on")
	}
	if term.Events() == nil {
		t.Error("Events returned no channel")
	}
	if err := term.Draw(func(f *terminal.Frame) {
		f.RenderWidget(&widgets.Checkbox{ID: "c", Checked: new(bool)}, cell.NewRect(0, 0, 10, 1))
	}); err != nil {
		t.Fatal(err)
	}
	if term.LastFrameDuration() <= 0 {
		t.Error("LastFrameDuration was not measured")
	}
	if len(term.LastWidgetStats()) == 0 {
		t.Error("no widget stats recorded for a frame that drew a widget")
	}
	if regions := term.LastImageRegions(); regions != nil && len(regions) != 0 {
		t.Errorf("a frame with no images reported %d image regions", len(regions))
	}
}
