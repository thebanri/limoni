package testkit

import (
	"fmt"
	"strings"

	"github.com/thebanri/limoni/core/cell"
)

// SnapshotDiff is one cell that differs between two snapshots: the row, the
// screen column the cell starts at, and the text of the cell on each side.
//
// The text is a string, not a rune, because a cell is not a rune in this
// library: a cell that holds a grapheme cluster keeps all of the cluster's
// runes, and that is what a reader has to see to know what differs.
type SnapshotDiff struct {
	Row, Column      int
	Expected, Actual string
}

// DiffSnapshot compares two snapshots cell by cell and returns the cells that
// differ, in row then column order.
//
// Snapshot writes exactly one cluster per cell — a cell that holds a wide rune
// is followed by the continuation space for the column it covers — so the rows
// are walked by cluster and one cluster is one column: Column is the column the
// cell starts at, the one a cursor would move to. Compared rune by rune, a cell
// that holds a multi-rune cluster shifted the rest of the row, so one cell that
// differed was reported as several, at columns that hold no such cell.
func DiffSnapshot(expected, actual string) []SnapshotDiff {
	return diffSnapshot(expected, actual, " ")
}

// diffSnapshot is DiffSnapshot with the text a missing cell stands for: a blank
// for DiffSnapshot, an empty string when the cells a row lacks have to be told
// apart from the blank cells the other row holds.
func diffSnapshot(expected, actual, missing string) []SnapshotDiff {
	want, got := strings.Split(expected, "\n"), strings.Split(actual, "\n")
	rows := max(len(want), len(got))
	var diffs []SnapshotDiff
	for y := 0; y < rows; y++ {
		wantCells, gotCells := snapshotCells(rowAt(want, y)), snapshotCells(rowAt(got, y))
		columns := max(len(wantCells), len(gotCells))
		for x := 0; x < columns; x++ {
			w, g := cellText(wantCells, x, missing), cellText(gotCells, x, missing)
			if w == g {
				continue
			}
			diffs = append(diffs, SnapshotDiff{Row: y, Column: x, Expected: w, Actual: g})
		}
	}
	return diffs
}

// snapshotCells splits one snapshot row into one entry per cell, so the index
// into the result is the column itself. A cell that holds a multi-rune cluster
// keeps its whole text in the one entry, the way Snapshot wrote it; a cluster
// that takes no column belongs to the cell beside it, since it holds no cell of
// its own.
func snapshotCells(row string) []string {
	cells := make([]string, 0, len(row))
	var pending string
	for rest := row; rest != ""; {
		cluster, width, next := cell.NextCluster(rest)
		if width <= 0 {
			if n := len(cells); n > 0 {
				cells[n-1] += cluster
			} else {
				pending += cluster
			}
			rest = next
			continue
		}
		cells = append(cells, pending+cluster)
		pending = ""
		rest = next
	}
	if pending != "" && len(cells) > 0 {
		cells[len(cells)-1] += pending
	}
	return cells
}

// cellText returns the text a column holds, and missing past the end of a
// shorter row. DiffSnapshot passes a blank: the snapshots pad the cells they do
// not have, so the row a message names is the row that was drawn.
func cellText(cells []string, column int, missing string) string {
	if column >= len(cells) {
		return missing
	}
	return cells[column]
}

// snapshotMismatch is the body of the error AssertSnapshot returns for two
// snapshots that are not equal.
//
// It is the cells DiffSnapshot reports. When the two differ only in trailing
// padding, DiffSnapshot reports none, since it reads a cell a shorter row lacks
// as a blank, so the cells one row has and the other lacks are reported with an
// empty text on the side that lacks them. When even that finds no cell, the
// snapshots differ in text that holds no cell, a row without cells at the end,
// and both are quoted.
func snapshotMismatch(expected, actual string) string {
	if diffs := DiffSnapshot(expected, actual); len(diffs) > 0 {
		return FormatSnapshotDiff(diffs)
	}
	if diffs := diffSnapshot(expected, actual, ""); len(diffs) > 0 {
		return FormatSnapshotDiff(diffs)
	}
	return fmt.Sprintf("no cell differs: expected=%q actual=%q\n", expected, actual)
}

// rowAt returns one row of a snapshot, or an empty row past the last one.
func rowAt(rows []string, y int) string {
	if y >= len(rows) {
		return ""
	}
	return rows[y]
}

func FormatSnapshotDiff(diffs []SnapshotDiff) string {
	var b strings.Builder
	for _, diff := range diffs {
		fmt.Fprintf(&b, "row=%d col=%d expected=%q actual=%q\n", diff.Row, diff.Column, diff.Expected, diff.Actual)
	}
	return b.String()
}
