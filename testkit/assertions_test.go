package testkit

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
)

// An assertion helper that cannot fail hides the bug it was written to catch,
// so every test here is run against input that is wrong on purpose and checks
// that the helper complains, and about the right thing. The passing direction
// is checked beside it.

// wantError fails unless err is non-nil and its message holds every part.
func wantError(t *testing.T, err error, parts ...string) {
	t.Helper()
	if err == nil {
		t.Fatalf("no error, want one mentioning %q", parts)
	}
	for _, part := range parts {
		if !strings.Contains(err.Error(), part) {
			t.Errorf("error %q does not mention %q", err, part)
		}
	}
}

// failingWriter reports err for every write.
type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

func noop(*terminal.EventContext) {}

func TestAssertSnapshotNamesTheCellThatDiffers(t *testing.T) {
	term := NewTerminal(6, 2)
	term.Draw(func(frame *terminal.Frame) {
		frame.Buffer.SetString(0, 0, "hello", cell.Style{})
		frame.Buffer.SetString(0, 1, "world", cell.Style{})
	})
	// The snapshot is every cell of every row, so a row is as wide as the
	// terminal.
	if err := term.AssertSnapshot("hello \nworld "); err != nil {
		t.Fatalf("a matching frame was reported: %v", err)
	}

	err := term.AssertSnapshot("hello \nwor!d ")
	wantError(t, err, "snapshot mismatch", `row=1 col=3 expected="!" actual="l"`)
	if strings.Contains(err.Error(), "row=0") {
		t.Errorf("the row that matches was reported too: %v", err)
	}
}

func TestAssertSnapshotIsNotDefeatedByPadding(t *testing.T) {
	term := NewTerminal(6, 2)
	term.Draw(func(frame *terminal.Frame) {
		frame.Buffer.SetString(0, 0, "hello", cell.Style{})
		frame.Buffer.SetString(0, 1, "world", cell.Style{})
	})

	// Expected text with its trailing spaces trimmed is not the frame: the
	// snapshot keeps every cell, and the comparison does not forgive padding.
	if err := term.AssertSnapshot("hello\nworld"); err == nil {
		t.Error("a snapshot without the trailing cells was accepted")
	}

	// A shorter expected string lacks the row that was drawn, and the message
	// names the row and the cells in it, not just "mismatch".
	wantError(t, term.AssertSnapshot("hello "), "row=1 col=0", `actual="w"`)

	// The reverse: a row that was not drawn, expected as text.
	wantError(t, term.AssertSnapshot("hello \nworld \nextra "), "row=2 col=0", `expected="e"`)

	// Extra text in a padding cell is a difference, not padding.
	wantError(t, term.AssertSnapshot("hello!\nworld "), "row=0 col=5", `expected="!"`, `actual=" "`)
}

func TestAssertFocused(t *testing.T) {
	term := NewTerminal(10, 2)

	// Nothing has been drawn, so nothing is focused.
	wantError(t, term.AssertFocused("first"), `""`, `"first"`)

	term.Draw(func(frame *terminal.Frame) {
		frame.FocusManager.Register("first")
		frame.FocusManager.Register("second")
	})
	if got := term.Focused(); got != "first" {
		t.Fatalf("Focused = %q, want the first widget drawn", got)
	}
	if err := term.AssertFocused("first"); err != nil {
		t.Errorf("the focused widget was refused: %v", err)
	}
	// A widget that is registered but does not hold the focus is not focused.
	wantError(t, term.AssertFocused("second"), `"first"`, `"second"`)
	wantError(t, term.AssertFocused("missing"), `"first"`, `"missing"`)

	term.Frame().FocusManager.SetFocused("second")
	if got := term.Focused(); got != "second" {
		t.Fatalf("Focused = %q after SetFocused(second)", got)
	}
	if err := term.AssertFocused("second"); err != nil {
		t.Errorf("the focused widget was refused: %v", err)
	}
	wantError(t, term.AssertFocused("first"), `"second"`, `"first"`)
}

func TestHoveredRegionAndAssertHovered(t *testing.T) {
	term := NewTerminal(12, 2)
	term.Draw(func(frame *terminal.Frame) {
		frame.RegisterEventRegion(terminal.EventRegion{ID: "left", Area: cell.NewRect(0, 0, 4, 1), Phase: terminal.TargetPhase, Handler: noop})
		frame.RegisterEventRegion(terminal.EventRegion{ID: "right", Area: cell.NewRect(6, 0, 4, 1), Phase: terminal.TargetPhase, Handler: noop})
	})

	// Before the pointer moves, nothing is hovered.
	if got := term.HoveredRegionID(); got != "" {
		t.Fatalf("HoveredRegionID = %q before any pointer move", got)
	}
	wantError(t, term.AssertHovered("left"), `""`, `"left"`)

	if !term.MovePointer(1, 0) {
		t.Fatal("moving onto a region reported that nothing was there")
	}
	if got := term.HoveredRegionID(); got != "left" {
		t.Fatalf("HoveredRegionID = %q, want left", got)
	}
	if err := term.AssertHovered("left"); err != nil {
		t.Errorf("the hovered region was refused: %v", err)
	}
	wantError(t, term.AssertHovered("right"), `"left"`, `"right"`)

	// The last cell of the region is inside it; the next one is not.
	term.MovePointer(3, 0)
	if err := term.AssertHovered("left"); err != nil {
		t.Errorf("the last cell of the region is not hovering it: %v", err)
	}
	if term.MovePointer(4, 0) {
		t.Error("the cell after the region was reported as a region")
	}
	wantError(t, term.AssertHovered("left"), `""`, `"left"`)

	// Moving across to the other region changes who is hovered.
	term.MovePointer(7, 0)
	if err := term.AssertHovered("right"); err != nil {
		t.Errorf("the hovered region was refused: %v", err)
	}
	wantError(t, term.AssertHovered("left"), `"right"`, `"left"`)
}

func TestClickAtReachesTheRegionUnderThePoint(t *testing.T) {
	term := NewTerminal(12, 2)
	var hits []string
	record := func(id string) func(*terminal.EventContext) {
		return func(ctx *terminal.EventContext) {
			if ctx.RegionID != id {
				t.Errorf("the handler for %q was given region %q", id, ctx.RegionID)
			}
			hits = append(hits, id)
		}
	}
	term.Draw(func(frame *terminal.Frame) {
		frame.RegisterEventRegion(terminal.EventRegion{ID: "left", Area: cell.NewRect(2, 0, 4, 1), Phase: terminal.TargetPhase, Handler: record("left")})
		frame.RegisterEventRegion(terminal.EventRegion{ID: "right", Area: cell.NewRect(6, 0, 4, 1), Phase: terminal.TargetPhase, Handler: record("right")})
	})
	at := time.Unix(1000, 0)

	for _, tc := range []struct {
		name    string
		x, y    uint16
		want    string // the region that gets the click; empty if none does
		handled bool
	}{
		{"first cell of left", 2, 0, "left", true},
		{"last cell of left", 5, 0, "left", true},
		{"first cell of right, beside left's last", 6, 0, "right", true},
		{"last cell of right", 9, 0, "right", true},
		{"one before left", 1, 0, "", false},
		{"one past right", 10, 0, "", false},
		{"the row below", 3, 1, "", false},
	} {
		hits = nil
		if got := term.ClickAt(tc.x, tc.y, at); got != tc.handled {
			t.Errorf("%s: ClickAt(%d,%d) = %v, want %v", tc.name, tc.x, tc.y, got, tc.handled)
		}
		switch {
		case tc.want == "" && len(hits) != 0:
			t.Errorf("%s: a click on nothing reached %v", tc.name, hits)
		case tc.want != "" && (len(hits) != 1 || hits[0] != tc.want):
			t.Errorf("%s: handlers called = %v, want only %q", tc.name, hits, tc.want)
		}
	}
}

// ClickAt exists so a test can say when a click happened: two clicks on one
// region no more than 500ms apart are a double click, by the clock the caller
// supplies and not by the wall clock.
func TestClickAtUsesTheTimestampItIsGiven(t *testing.T) {
	term := NewTerminal(12, 2)
	var seen []string
	var times []time.Time
	handler := func(ctx *terminal.EventContext) {
		seen = append(seen, fmt.Sprintf("%s:%d", ctx.RegionID, ctx.ClickCount))
		times = append(times, ctx.EventTime)
	}
	term.Draw(func(frame *terminal.Frame) {
		frame.RegisterEventRegion(terminal.EventRegion{ID: "a", Area: cell.NewRect(0, 0, 4, 1), Phase: terminal.TargetPhase, Handler: handler})
		frame.RegisterEventRegion(terminal.EventRegion{ID: "b", Area: cell.NewRect(6, 0, 4, 1), Phase: terminal.TargetPhase, Handler: handler})
	})

	start := time.Unix(1000, 0)
	clicks := []struct {
		x    uint16
		at   time.Duration
		want string
	}{
		{1, 0, "a:1"},                       // the first click
		{1, 100 * time.Millisecond, "a:2"},  // the same region, soon after
		{1, 700 * time.Millisecond, "a:1"},  // 600ms after the last click
		{7, 800 * time.Millisecond, "b:1"},  // another region
		{1, 900 * time.Millisecond, "a:1"},  // back, but the last click was on b
		{1, 1400 * time.Millisecond, "a:2"}, // exactly 500ms after the last click
	}
	for i, c := range clicks {
		seen, times = nil, nil
		at := start.Add(c.at)
		if !term.ClickAt(c.x, 0, at) {
			t.Fatalf("click %d at column %d was not handled", i, c.x)
		}
		if len(seen) != 1 || seen[0] != c.want {
			t.Errorf("click %d: handler saw %v, want %q", i, seen, c.want)
		}
		if len(times) == 1 && !times[0].Equal(at) {
			t.Errorf("click %d: the handler saw time %v, want the one given, %v", i, times[0], at)
		}
	}
}
func TestWriteAccessibilityLineMode(t *testing.T) {
	term := NewTerminal(10, 2)
	draw := func(label string) {
		term.Draw(func(frame *terminal.Frame) {
			frame.RegisterAccessibility(accessibility.Node{ID: "save", Role: accessibility.RoleButton, Label: label, Bounds: cell.NewRect(0, 0, 4, 1)})
		})
	}
	draw("Save")

	var out bytes.Buffer
	if err := term.WriteAccessibilityLineMode(&out, accessibility.Mode{}); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), "button#save \"Save\" bounds=0,0 4x1\n"; got != want {
		t.Errorf("written = %q, want %q", got, want)
	}
	if got, want := out.String(), term.AccessibilityLineMode(accessibility.Mode{})+"\n"; got != want {
		t.Errorf("the writer and the string disagree: %q and %q", got, want)
	}

	// The mode is the caller's: ASCIIOnly turns a letter that is not ASCII
	// into a question mark, as AccessibilityLineMode does for the same frame.
	draw("Café")
	out.Reset()
	if err := term.WriteAccessibilityLineMode(&out, accessibility.Mode{ASCIIOnly: true}); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), "button#save \"Caf?\" bounds=0,0 4x1\n"; got != want {
		t.Errorf("written in ASCII-only mode = %q, want %q", got, want)
	}

	// A writer that fails has its error returned, not swallowed.
	broken := errors.New("disk full")
	if err := term.WriteAccessibilityLineMode(failingWriter{broken}, accessibility.Mode{}); !errors.Is(err, broken) {
		t.Errorf("error = %v, want the writer's", err)
	}

	// A frame with no semantic nodes writes nothing.
	out.Reset()
	term.Draw(nil)
	if err := term.WriteAccessibilityLineMode(&out, accessibility.Mode{}); err != nil || out.Len() != 0 {
		t.Errorf("an empty frame wrote %q, err %v", out.String(), err)
	}
}

func TestAssertModalIsolation(t *testing.T) {
	term := NewTerminal(20, 6)

	// No layers at all.
	wantError(t, term.AssertModalIsolation("confirm"), `""`, `"confirm"`)

	term.Draw(func(frame *terminal.Frame) {
		frame.RegisterLayer("menu", terminal.LayerPopup, cell.NewRect(0, 0, 6, 3), 50, nil)
	})
	// A popup is a layer but not a modal, even with the ID that was asked for.
	wantError(t, term.AssertModalIsolation("menu"), `""`, `"menu"`)

	term.Draw(func(frame *terminal.Frame) {
		frame.RegisterLayer("menu", terminal.LayerPopup, cell.NewRect(0, 0, 6, 3), 50, nil)
		frame.RegisterLayer("settings", terminal.LayerModal, cell.NewRect(2, 1, 12, 4), 100, nil)
		frame.RegisterLayer("confirm", terminal.LayerModal, cell.NewRect(4, 2, 8, 2), 200, nil)
	})
	if err := term.AssertModalIsolation("confirm"); err != nil {
		t.Errorf("the topmost modal was refused: %v", err)
	}
	// A modal that is underneath another is not isolating anything.
	wantError(t, term.AssertModalIsolation("settings"), `"confirm"`, `"settings"`)
	wantError(t, term.AssertModalIsolation("menu"), `"confirm"`, `"menu"`)
}

// Every helper answers a nil Terminal with a zero value or an error rather than
// a panic, so a test that failed to build its terminal reports that.
func TestAssertionsOnANilTerminal(t *testing.T) {
	var term *Terminal
	if got := term.Focused(); got != "" {
		t.Errorf("Focused = %q", got)
	}
	if got := term.HoveredRegionID(); got != "" {
		t.Errorf("HoveredRegionID = %q", got)
	}
	if term.ClickAt(0, 0, time.Unix(1000, 0)) {
		t.Error("ClickAt handled a click")
	}
	wantError(t, term.AssertFocused("a"), `"a"`)
	wantError(t, term.AssertHovered("a"), `"a"`)
	wantError(t, term.AssertSnapshot("a"), "snapshot mismatch")
	wantError(t, term.AssertModalIsolation("a"), "nil")

	var out bytes.Buffer
	if err := term.WriteAccessibilityLineMode(&out, accessibility.Mode{}); err != nil || out.Len() != 0 {
		t.Errorf("a nil terminal wrote %q, err %v", out.String(), err)
	}
}
