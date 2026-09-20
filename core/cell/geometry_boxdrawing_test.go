package cell

import "testing"

func TestRectIntersection(t *testing.T) {
	cases := []struct {
		name string
		a, b Rect
		want Rect
	}{
		{"overlapping corner", NewRect(0, 0, 10, 10), NewRect(5, 5, 10, 10), NewRect(5, 5, 5, 5)},
		{"one inside the other", NewRect(0, 0, 10, 10), NewRect(2, 2, 3, 3), NewRect(2, 2, 3, 3)},
		{"identical", NewRect(1, 1, 4, 4), NewRect(1, 1, 4, 4), NewRect(1, 1, 4, 4)},
		{"disjoint", NewRect(0, 0, 5, 5), NewRect(10, 10, 5, 5), Rect{}},
		{"touching edges is not overlap", NewRect(0, 0, 5, 5), NewRect(5, 0, 5, 5), Rect{}},
		{"empty stays empty", NewRect(0, 0, 0, 0), NewRect(0, 0, 5, 5), Rect{}},
		{"overlap on one axis only", NewRect(0, 0, 10, 2), NewRect(3, 5, 2, 2), Rect{}},
	}
	for _, c := range cases {
		if got := c.a.Intersection(c.b); got != c.want {
			t.Errorf("%s: %+v ∩ %+v = %+v, want %+v", c.name, c.a, c.b, got, c.want)
		}
		// Intersection is symmetric, and a widget clipped by its parent
		// must not depend on which way round the call was made.
		if got := c.b.Intersection(c.a); got != c.want {
			t.Errorf("%s (reversed): got %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestRectContains(t *testing.T) {
	r := NewRect(2, 3, 4, 5) // x 2..5, y 3..7
	in := [][2]uint16{{2, 3}, {5, 7}, {3, 5}}
	out := [][2]uint16{{1, 3}, {6, 7}, {2, 2}, {2, 8}, {0, 0}}
	for _, p := range in {
		if !r.Contains(p[0], p[1]) {
			t.Errorf("%+v should contain (%d,%d)", r, p[0], p[1])
		}
	}
	for _, p := range out {
		if r.Contains(p[0], p[1]) {
			t.Errorf("%+v should not contain (%d,%d)", r, p[0], p[1])
		}
	}
	if (Rect{}).Contains(0, 0) {
		t.Error("an empty rect contains nothing, not even its own origin")
	}
}

func TestNewPointAndNewRectKeepTheirFields(t *testing.T) {
	p := NewPoint(3, 4)
	if p.X != 3 || p.Y != 4 {
		t.Errorf("NewPoint = %+v", p)
	}
	r := NewRect(1, 2, 3, 4)
	if r.X != 1 || r.Y != 2 || r.Width != 3 || r.Height != 4 {
		t.Errorf("NewRect = %+v", r)
	}
}

// What border merging is made of: a cell already holding one glyph receives
// another, and the result has to draw both sets of segments.
func TestMergeBoxDrawing(t *testing.T) {
	cases := []struct {
		name               string
		existing, incoming rune
		want               rune
	}{
		{"a rule arriving at a vertical border makes a tee", '│', '─', '┼'},
		{"a corner meeting a vertical", '│', '┌', '├'},
		{"two corners meeting side by side", '┐', '┌', '┬'},
		{"bottom corners", '┘', '└', '┴'},
		{"the same glyph twice", '│', '│', '│'},
		{"a rounded corner counts as its square twin", '╰', '─', '┴'},
		{"text is never rewritten", 'a', '─', '─'},
		{"an incoming non-border wins", '│', 'x', 'x'},
		{"heavy lines have no honest junction with light", '┃', '─', '─'},
		{"double lines likewise", '║', '─', '─'},
	}
	for _, c := range cases {
		if got := MergeBoxDrawing(c.existing, c.incoming); got != c.want {
			t.Errorf("%s: merge(%q, %q) = %q, want %q", c.name, c.existing, c.incoming, got, c.want)
		}
	}
}

// Every light glyph must round-trip: its segments map back to itself, or a
// merge would silently replace a border with a different one.
func TestBoxDrawingSegmentsRoundTrip(t *testing.T) {
	for r, segs := range boxDrawingSegments {
		got, ok := BoxDrawingSegments(r)
		if !ok || got != segs {
			t.Errorf("BoxDrawingSegments(%q) = %d,%v", r, got, ok)
		}
		if back, ok := segmentsToBoxDrawing[segs]; ok {
			if again, _ := BoxDrawingSegments(back); again != segs {
				t.Errorf("%q → %d → %q draws %d instead", r, segs, back, again)
			}
		}
	}
	if _, ok := BoxDrawingSegments('x'); ok {
		t.Error("'x' is not a box-drawing rune")
	}
}
