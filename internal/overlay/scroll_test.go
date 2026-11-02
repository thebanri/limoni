//go:build linux

package overlay

import (
	"slices"
	"testing"
)

// screen builds a sample whose rows are the given text, a pixel a byte,
// with characters a code apart five levels apart: more than near allows.
func screen(lines ...string) *sample {
	s := &sample{}
	for _, l := range lines {
		row := make([]byte, 0, 4*len(l))
		for _, c := range []byte(l) {
			v := c * 5
			row = append(row, v, v, v, v)
		}
		s.rows = append(s.rows, row)
		s.skip = append(s.skip, nil)
	}
	return s
}

func TestShiftOfFindsTheRowsScrolled(t *testing.T) {
	bottom := screen("75  ", "76  ", "77  ", "78  ", "79  ", "80  ")
	up := screen("72  ", "73  ", "74  ", "75  ", "76  ", "77  ") // scrolled up three
	if k, ok := shiftOf(bottom, up, 0); !ok || k != 3 {
		t.Fatalf("shiftOf = %d, %v; want 3, true", k, ok)
	}
	if k, ok := shiftOf(bottom, bottom, 0); !ok || k != 0 {
		t.Fatalf("unchanged: shiftOf = %d, %v; want 0, true", k, ok)
	}
}

func TestShiftOfGivesUpPastAScreen(t *testing.T) {
	bottom := screen("75  ", "76  ", "77  ", "78  ")
	far := screen("10  ", "11  ", "12  ", "13  ")
	if _, ok := shiftOf(bottom, far, 0); ok {
		t.Fatal("a screen with nothing in common should not fit any shift")
	}
}

func TestShiftOfIgnoresWhatIsUnderAPicture(t *testing.T) {
	// The picture covers bytes 8..16 of every row: the overlay's own pixels,
	// which do not move with the text.
	bottom := screen("a1PPxx", "a2PPxx", "a3PPxx", "a4PPxx")
	up := screen("a0QQxx", "a1QQxx", "a2QQxx", "a3QQxx")
	for i := range bottom.rows {
		bottom.skip[i] = [][2]int{{8, 16}}
		up.skip[i] = [][2]int{{8, 16}}
	}
	if k, ok := shiftOf(bottom, up, 0); !ok || k != 1 {
		t.Fatalf("shiftOf = %d, %v; want 1, true", k, ok)
	}
}

func TestShiftOfKeepsTheShiftInForceOnABlankScreen(t *testing.T) {
	blank := screen("    ", "    ", "    ")
	if k, ok := shiftOf(blank, blank, 2); !ok || k != 2 {
		t.Fatalf("shiftOf = %d, %v; want the shift in force, 2", k, ok)
	}
	// With no shift in force, blank rows fit everywhere and say nothing.
	if k, ok := shiftOf(blank, blank, -1); ok {
		t.Fatalf("blank rows chose %d", k)
	}
}

func TestCursorSkipCoversTheCursorWherePaddingPutsIt(t *testing.T) {
	// 10x23 cells, the grid guessed 6 pixels down (centred), cursor in
	// cell 0 of row 44: Alacritty draws it at y 1012..1035, the guess at
	// 1018..1041. Both, and the row above's lowest line, are left out.
	for _, y := range []int{1012, 1020, 1034, 1040} {
		if r, ok := cursorSkip(y, 0, 44, 10, 23, 0, 6, 0, 1910); !ok || r != [2]int{0, 80} {
			t.Errorf("y %d: %v, %v; want [0 80], true", y, r, ok)
		}
	}
	for _, y := range []int{990, 1060} {
		if _, ok := cursorSkip(y, 0, 44, 10, 23, 0, 6, 0, 1910); ok {
			t.Errorf("y %d is a row away and was left out", y)
		}
	}
}

func TestShiftOfAllowsTheGPUsRounding(t *testing.T) {
	bottom := screen("75  ", "76  ", "77  ", "78  ")
	up := screen("74  ", "75  ", "76  ", "77  ")
	up.rows[2][1]++ // 76, drawn a level brighter on its new row
	if k, ok := shiftOf(bottom, up, 0); !ok || k != 1 {
		t.Fatalf("shiftOf = %d, %v; want 1, true", k, ok)
	}
}

// lines are the numbered lines of a long output, n rows up from its bottom.
func lines(rows, n int) *sample {
	var text []string
	for i := 100 - rows - n; i < 100-n; i++ {
		text = append(text, string(rune('A'+i%26))+string(rune('a'+i/26))+"  ")
	}
	s := screen(text...)
	s.offset = n
	return s
}

func TestFollowGoesFurtherThanAScreenAStepAtATime(t *testing.T) {
	const rows = 6
	bottom := lines(rows, 0)
	last := bottom
	for _, n := range []int{3, 6, 9, 12, 9, 4, 0} { // up two screens and back
		b := lines(rows, n)
		k, ok := follow(bottom, last, b, last.offset)
		if !ok || k != n {
			t.Fatalf("from %d to %d rows up: follow = %d, %v", last.offset, n, k, ok)
		}
		b.offset = k
		last = b
	}
}

func TestFollowFindsTheBottomAfterAJump(t *testing.T) {
	const rows = 6
	bottom := lines(rows, 0)
	far := lines(rows, 30) // a page up and more, out of reach
	if _, ok := follow(bottom, bottom, far, 0); ok {
		t.Fatal("a jump past a screen should not be followed")
	}
	// From there, back at a view near the bottom.
	if k, ok := follow(bottom, far, lines(rows, 2), 0); !ok || k != 2 {
		t.Fatalf("follow = %d, %v; want 2, true", k, ok)
	}
}

func TestCoveringKeepsTheOldPlaceForAMoment(t *testing.T) {
	o := &Overlay{dy: 40}
	if got := o.covering(); !slices.Equal(got, []int{40}) {
		t.Fatalf("settled: %v, want [40]", got)
	}
	o.moved()
	o.dy = 60
	if got := o.covering(); !slices.Equal(got, []int{60, 40}) {
		t.Fatalf("just moved: %v, want [60 40]", got)
	}
	o.moved()
	o.hidden = true
	if got := o.covering(); !slices.Equal(got, []int{60}) {
		t.Fatalf("just hidden: %v, want [60], where they still show", got)
	}
	o.movedAt = o.movedAt.Add(-stale)
	if got := o.covering(); len(got) != 0 {
		t.Fatalf("hidden and settled: %v, want nothing", got)
	}
}
