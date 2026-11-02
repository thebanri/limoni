//go:build linux

package overlay

import "testing"

// screen builds a sample whose rows are the given text, a pixel a byte,
// with characters a code apart five levels apart: more than near allows.
func screen(lines ...string) *sample {
	s := &sample{cursor: -1}
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

func TestShiftOfSkipsTheCursorRow(t *testing.T) {
	bottom := screen("ab  ", "cd  ", "ef  ", "$_  ")
	bottom.cursor = 3
	now := screen("ab  ", "cd  ", "ef  ", "$   ") // the cursor blinked off
	if k, ok := shiftOf(bottom, now, 0); !ok || k != 0 {
		t.Fatalf("shiftOf = %d, %v; want 0, true", k, ok)
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
