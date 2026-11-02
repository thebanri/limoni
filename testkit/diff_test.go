package testkit

import (
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// A snapshot row is one cluster per cell: a cell that holds a grapheme cluster
// writes every rune of the cluster, and a cell that holds a wide rune is
// followed by the continuation space for the column it covers. Every test here
// checks the column a reader counts to, the one a terminal cursor would move
// to, not the rune index.

// differAt is the single cell two snapshots differ in, or a failure when the
// comparison reported another number of cells.
func differAt(t *testing.T, expected, actual string) SnapshotDiff {
	t.Helper()
	diffs := DiffSnapshot(expected, actual)
	if len(diffs) != 1 {
		t.Fatalf("one cell differs, got %d:\n%s", len(diffs), FormatSnapshotDiff(diffs))
	}
	return diffs[0]
}

// drawnSnapshot is a real frame: the text the way Buffer.Snapshot writes it,
// continuation spaces included, which is what AssertSnapshot compares.
func drawnSnapshot(t *testing.T, width uint16, text string) string {
	t.Helper()
	buf := buffer.NewBuffer(cell.NewRect(0, 0, width, 1))
	buf.SetString(0, 0, text, cell.Style{})
	return buf.Snapshot()
}

func TestDiffSnapshotColumnsCountCellsNotRunes(t *testing.T) {
	// 本 starts at the third cell, which is column 2 on screen. The snapshots
	// are the real shape, the continuation space included: "日 本   ".
	lost := drawnSnapshot(t, 6, "日本")
	got := drawnSnapshot(t, 6, "日X")
	if lost != "日 本   " || got != "日 X   " {
		t.Fatalf("snapshots = %q and %q, want the six cells with the continuation spaces", lost, got)
	}
	diff := differAt(t, lost, got)
	if diff.Column != 2 {
		t.Errorf("column = %d, want 2", diff.Column)
	}
	if diff.Expected != "本" || diff.Actual != "X" {
		t.Errorf("expected %q actual %q, want %q and %q", diff.Expected, diff.Actual, "本", "X")
	}
}

func TestDiffSnapshotReportsAClusterCellOnce(t *testing.T) {
	// A test writes the precomposed form; the buffer keeps the cell as the two
	// runes it was drawn with. That is one cell, so one row entry: rune by rune
	// the two rows disagreed from the cluster onwards.
	diff := differAt(t, "éxyz", "e\u0301xyz")
	if diff.Column != 0 {
		t.Errorf("column = %d, want 0", diff.Column)
	}
	if diff.Expected != "é" || diff.Actual != "e\u0301" {
		t.Errorf("expected %q actual %q, want %q and %q", diff.Expected, diff.Actual, "é", "e\u0301")
	}
}

func TestDiffSnapshotColumnsAfterACluster(t *testing.T) {
	// q replaces z in the fourth cell, at column 3. Counting the cluster's two
	// runes as two columns put it at 4.
	diff := differAt(t, "e\u0301xyz", "e\u0301xyq")
	if diff.Column != 3 {
		t.Errorf("column = %d, want 3", diff.Column)
	}
	if diff.Expected != "z" || diff.Actual != "q" {
		t.Errorf("expected %q actual %q, want %q and %q", diff.Expected, diff.Actual, "z", "q")
	}
}

func TestDiffSnapshotReportsAnEmojiCellOnce(t *testing.T) {
	// A family emoji is five runes in one cell that takes two columns, and its
	// continuation space is in the snapshot after it. A frame that lost it has
	// two blank cells there, and x is still in the fourth cell, at column 3.
	family := "👨‍👩‍👧"
	diff := differAt(t, "   x", family+"  x")
	if diff.Column != 0 {
		t.Errorf("column = %d, want 0", diff.Column)
	}
	if diff.Actual != family {
		t.Errorf("actual = %q, want the whole cluster %q", diff.Actual, family)
	}
}

func TestDiffSnapshotReadsTheBufferSnapshotItIsGiven(t *testing.T) {
	// The end to end shape: a cluster, a wide rune and the cell after them, and
	// the one cell that differs between two real frames. This is the test that
	// catches the column moving when a cell takes more than one rune or column:
	// rune by rune q sat at index 4, and counting the wide rune's width as well
	// put it at 4 too, when it is in column 3.
	lost := drawnSnapshot(t, 6, "e\u0301日x")
	got := drawnSnapshot(t, 6, "e\u0301日q")
	if lost != "e\u0301日 x  " || got != "e\u0301日 q  " {
		t.Fatalf("snapshots = %q and %q, want the six cells with the continuation space", lost, got)
	}
	diff := differAt(t, lost, got)
	if diff.Column != 3 {
		t.Errorf("column = %d, want 3", diff.Column)
	}
	if diff.Expected != "x" || diff.Actual != "q" {
		t.Errorf("expected %q actual %q, want %q and %q", diff.Expected, diff.Actual, "x", "q")
	}
}

func TestDiffSnapshotReportsEveryRowItIsGiven(t *testing.T) {
	diffs := DiffSnapshot("ab\ncd", "ab\ndd")
	if len(diffs) != 1 {
		t.Fatalf("one cell differs, got %d:\n%s", len(diffs), FormatSnapshotDiff(diffs))
	}
	if diffs[0].Row != 1 || diffs[0].Column != 0 {
		t.Errorf("row=%d column=%d, want row 1 column 0", diffs[0].Row, diffs[0].Column)
	}
	if diffs[0].Expected != "c" || diffs[0].Actual != "d" {
		t.Errorf("expected %q actual %q, want %q and %q", diffs[0].Expected, diffs[0].Actual, "c", "d")
	}
}

func TestDiffSnapshotOfMatchingRowsIsEmpty(t *testing.T) {
	for _, snapshot := range []string{
		"",
		"abc",
		"a\tb",
		"日本",
		"e\u0301xyz",
		"👨‍👩‍👧 x",
		"ab\ncd",
	} {
		if diffs := DiffSnapshot(snapshot, snapshot); len(diffs) != 0 {
			t.Errorf("%q compared with itself reported %d cells", snapshot, len(diffs))
		}
	}
}

func TestDiffSnapshotReportsAnAbsentCellAsABlank(t *testing.T) {
	// A row shorter than the other ends in a blank cell, the way the shorter
	// row's cells are padded.
	diff := differAt(t, "ab", "abc")
	if diff.Column != 2 || diff.Expected != " " || diff.Actual != "c" {
		t.Errorf("row=%d column=%d expected=%q actual=%q, want column 2, blank and %q",
			diff.Row, diff.Column, diff.Expected, diff.Actual, "c")
	}
}

func TestFormatSnapshotDiffNamesRowColumnAndBothCells(t *testing.T) {
	diffs := []SnapshotDiff{{Row: 2, Column: 5, Expected: "z", Actual: "q"}}
	got := FormatSnapshotDiff(diffs)
	want := `row=2 col=5 expected="z" actual="q"` + "\n"
	if got != want {
		t.Errorf("FormatSnapshotDiff = %q, want %q", got, want)
	}
}

func TestDiffSnapshotKeepsAZeroWidthClusterWithItsNeighbour(t *testing.T) {
	// A combining mark that starts a row cannot be dropped: it belongs to the
	// cell after it, and losing it would shift the row.
	diffs := DiffSnapshot("\u0301a", "a")
	if len(diffs) != 1 {
		t.Fatalf("one cell differs, got %d:\n%s", len(diffs), FormatSnapshotDiff(diffs))
	}
	if diffs[0].Column != 0 || diffs[0].Expected != "\u0301a" || diffs[0].Actual != "a" {
		t.Errorf("row=%d column=%d expected=%q actual=%q, want column 0, %q and %q",
			diffs[0].Row, diffs[0].Column, diffs[0].Expected, diffs[0].Actual, "\u0301a", "a")
	}
}

func TestDiffSnapshotMessageStaysOneLinePerCell(t *testing.T) {
	// The message a failing assertion prints: a header is not added, one line
	// per cell.
	got := FormatSnapshotDiff(DiffSnapshot("ab", "cb"))
	if lines := strings.Split(strings.TrimSuffix(got, "\n"), "\n"); len(lines) != 1 {
		t.Errorf("message has %d lines, want 1:\n%s", len(lines), got)
	}
}
