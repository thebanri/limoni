package widgets

import (
	"testing"
	"time"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

func TestSpinnerSetFrameAtCycles(t *testing.T) {
	set := SpinnerSet{Frames: []string{"a", "b", "c"}, Interval: 100 * time.Millisecond}

	cases := []struct {
		elapsed time.Duration
		want    string
	}{
		{0, "a"},
		{99 * time.Millisecond, "a"},
		{100 * time.Millisecond, "b"},
		{250 * time.Millisecond, "c"},
		{300 * time.Millisecond, "a"},
		{-50 * time.Millisecond, "a"},
	}
	for _, tc := range cases {
		if got := set.FrameAt(tc.elapsed); got != tc.want {
			t.Errorf("FrameAt(%v) = %q, want %q", tc.elapsed, got, tc.want)
		}
	}
}

func TestSpinnerSetFrameAtEmptyAndZeroInterval(t *testing.T) {
	if got := (SpinnerSet{}).FrameAt(time.Second); got != "" {
		t.Fatalf("empty set returned %q, want empty", got)
	}
	set := SpinnerSet{Frames: []string{"x", "y"}}
	if got := set.FrameAt(150 * time.Millisecond); got != "y" {
		t.Fatalf("zero interval fell back incorrectly: %q", got)
	}
}

func TestSpinnerExplicitFrameIsDeterministic(t *testing.T) {
	set := SpinnerSet{Frames: []string{"1", "2", "3"}}
	for i, want := range []string{"1", "2", "3", "1", "2"} {
		if got := (Spinner{Set: set, Frame: i}).CurrentFrame(); got != want {
			t.Errorf("Frame %d = %q, want %q", i, got, want)
		}
	}
	// Negative indices wrap rather than panic.
	if got := (Spinner{Set: set, Frame: -1}).CurrentFrame(); got != "3" {
		t.Errorf("Frame -1 = %q, want %q", got, "3")
	}
}

func TestSpinnerClockDrivenFrame(t *testing.T) {
	start := time.Unix(0, 0)
	set := SpinnerSet{Frames: []string{"a", "b", "c"}, Interval: 10 * time.Millisecond}

	s := Spinner{
		Set:   set,
		Since: start,
		now:   func() time.Time { return start.Add(25 * time.Millisecond) },
	}
	if got := s.CurrentFrame(); got != "c" {
		t.Fatalf("clock-driven frame = %q, want %q", got, "c")
	}
}

func TestSpinnerDefaultsToBraille(t *testing.T) {
	got := (Spinner{}).CurrentFrame()
	if got != SpinnerBraille.Frames[0] {
		t.Fatalf("default spinner frame = %q, want %q", got, SpinnerBraille.Frames[0])
	}
}

func TestSpinnerDrawWithLabel(t *testing.T) {
	area := cell.NewRect(0, 0, 20, 1)
	buf := buffer.NewBuffer(area)

	Spinner{Set: SpinnerLine, Frame: 1, Label: "Loading"}.
		Draw(cell.NewContext(area, cell.Style{}), buf)

	if got, want := row(buf, 0, 0, 9), "/ Loading"; got != want {
		t.Fatalf("spinner row = %q, want %q", got, want)
	}
}

// The label must not shift when the glyph width changes between frames.
func TestSpinnerLabelPositionStableAcrossFrames(t *testing.T) {
	// A set mixing a single-width and a double-width glyph.
	set := SpinnerSet{Frames: []string{".", "日"}, Interval: time.Millisecond}
	area := cell.NewRect(0, 0, 20, 1)

	positions := make([]int, 0, len(set.Frames))
	for frame := range set.Frames {
		buf := buffer.NewBuffer(area)
		Spinner{Set: set, Frame: frame, Label: "X"}.
			Draw(cell.NewContext(area, cell.Style{}), buf)

		found := -1
		for x := uint16(0); x < area.Width; x++ {
			if c := buf.Get(x, 0); c != nil && c.Content == 'X' {
				found = int(x)
				break
			}
		}
		if found < 0 {
			t.Fatalf("frame %d did not render the label", frame)
		}
		positions = append(positions, found)
	}

	if positions[0] != positions[1] {
		t.Fatalf("label moved between frames: %v", positions)
	}
}

func TestSpinnerDrawWithoutLabel(t *testing.T) {
	area := cell.NewRect(0, 0, 5, 1)
	buf := buffer.NewBuffer(area)

	Spinner{Set: SpinnerLine, Frame: 2}.Draw(cell.NewContext(area, cell.Style{}), buf)

	if got, want := row(buf, 0, 0, 3), "-  "; got != want {
		t.Fatalf("spinner without label = %q, want %q", got, want)
	}
}

func TestSpinnerClipsInNarrowArea(t *testing.T) {
	area := cell.NewRect(0, 0, 2, 1)
	buf := buffer.NewBuffer(area)

	Spinner{Set: SpinnerLine, Frame: 0, Label: "Loading"}.
		Draw(cell.NewContext(area, cell.Style{}), buf)

	if got, want := row(buf, 0, 0, 2), "| "; got != want {
		t.Fatalf("clipped spinner = %q, want %q", got, want)
	}
}

func TestSpinnerSetWidth(t *testing.T) {
	if got := SpinnerLine.Width(); got != 1 {
		t.Fatalf("SpinnerLine.Width() = %d, want 1", got)
	}
	mixed := SpinnerSet{Frames: []string{".", "日"}}
	if got := mixed.Width(); got != 2 {
		t.Fatalf("mixed-width set Width() = %d, want 2", got)
	}
}

func TestSpinnerSizeHint(t *testing.T) {
	s := Spinner{Set: SpinnerLine, Label: "Loading"}
	// 1 glyph + 1 gap + 7 label
	if w, h := s.SizeHint(cell.NewRect(0, 0, 40, 3)); w != 9 || h != 1 {
		t.Fatalf("SizeHint = (%d,%d), want (9,1)", w, h)
	}
	if w, _ := s.SizeHint(cell.NewRect(0, 0, 4, 1)); w != 4 {
		t.Fatalf("SizeHint in a narrow area = %d, want 4", w)
	}
}

// Every built-in set must be non-empty with a positive interval, so none of
// them silently render nothing.
func TestBuiltinSpinnerSetsAreValid(t *testing.T) {
	sets := map[string]SpinnerSet{
		"Braille": SpinnerBraille,
		"Dots":    SpinnerDots,
		"Line":    SpinnerLine,
		"Arrow":   SpinnerArrow,
		"Bar":     SpinnerBar,
		"Pulse":   SpinnerPulse,
		"Clock":   SpinnerClock,
	}
	for name, set := range sets {
		if len(set.Frames) == 0 {
			t.Errorf("%s has no frames", name)
		}
		if set.Interval <= 0 {
			t.Errorf("%s has a non-positive interval", name)
		}
		if set.Width() == 0 {
			t.Errorf("%s reports zero width", name)
		}
	}
}

func BenchmarkSpinnerDraw(b *testing.B) {
	area := cell.NewRect(0, 0, 40, 1)
	buf := buffer.NewBuffer(area)
	ctx := cell.NewContext(area, cell.Style{})
	s := Spinner{Set: SpinnerBraille, Label: "Fetching results"}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Frame = i
		s.Draw(ctx, buf)
	}
}
