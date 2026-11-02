//go:build linux

package overlay

import "testing"

// rowsOf builds a view of lines in which each character is a pixel of a
// sample line, the same in all of them, as the window is read.
func rowsOf(lines ...string) *sample {
	s := &sample{}
	for _, l := range lines {
		var row []byte
		for range sampleLines {
			for _, c := range []byte(l) {
				v := c * 5
				row = append(row, v, v, v, v)
			}
		}
		s.rows = append(s.rows, row)
		s.skip = append(s.skip, nil)
	}
	return s
}

func TestHistoryFindsTextSeenBeforeAPicture(t *testing.T) {
	var h history
	// "run 2" was output and seen at the bottom, its picture below it.
	h.remember(rowsOf("    ", "run2", "proc", ":::2"), 0)
	// A picture four rows tall and "run 3" came after: seven lines up.
	h.shift(7)
	h.remember(rowsOf("    ", "run3", "proc", ":::3"), 0)
	// Scrolled up until "run 2" is back on the second row: nothing in
	// the view is in the one before it, only in what was seen at output.
	view := rowsOf("    ", "run2", "proc", ":::2")
	if k, ok := h.recall(view); !ok || k != 7 {
		t.Fatalf("recall = %d, %v; want 7, true", k, ok)
	}
}

func TestHistoryDoesNotGuessOnBlankRows(t *testing.T) {
	var h history
	h.remember(rowsOf("    ", "    ", "    "), 0)
	if k, ok := h.recall(rowsOf("    ", "    ", "    ")); ok {
		t.Fatalf("a blank view recalled %d", k)
	}
}

func TestHistoryForgetsRowsTooFarUp(t *testing.T) {
	var h history
	h.remember(rowsOf("abcd", "efgh"), 0)
	h.shift(maxHistory + 1)
	if len(h) != 0 {
		t.Fatalf("%d rows kept past maxHistory", len(h))
	}
}

func TestHistoryDoesNotChooseBetweenRepeats(t *testing.T) {
	var h history
	// The same command, run twice, ten lines apart: the view fits both.
	h.remember(rowsOf("    ", "cmd1", "proc", "    "), 0)
	h.shift(10)
	h.remember(rowsOf("    ", "cmd1", "proc", "    "), 0)
	if k, ok := h.recall(rowsOf("    ", "cmd1", "proc", "    ")); ok {
		t.Fatalf("recalled %d, one of two places that fit", k)
	}
}

func TestHistoryWantsMoreThanOneRowOfText(t *testing.T) {
	var h history
	h.remember(rowsOf("    ", "abcd", "    "), 0)
	// One row of text may be any row that reads the same: not enough.
	if k, ok := h.recall(rowsOf("    ", "    ", "abcd")); ok {
		t.Fatalf("one row recalled %d", k)
	}
}
