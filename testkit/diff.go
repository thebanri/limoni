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
// A snapshot row is not one rune per column: a cell that holds a grapheme
// cluster writes every rune of the cluster, and a cell that holds a wide rune
// writes a continuation space for the column it covers. The rows are walked by
// cluster so that one cell is one entry, and Column is the screen column the
// cell starts at, the one a cursor would move to: a row that holds an emoji or
// a CJK cell before the difference no longer reports a column past the cell, and
// one cell that differs is reported once instead of shifting the rest of the row.
func DiffSnapshot(expected, actual string) []SnapshotDiff {
	want, got := strings.Split(expected, "\n"), strings.Split(actual, "\n")
	rows := max(len(want), len(got))
	var diffs []SnapshotDiff
	for y := 0; y < rows; y++ {
		wantCells, gotCells := snapshotCells(rowAt(want, y)), snapshotCells(rowAt(got, y))
		columns := max(len(wantCells), len(gotCells))
		for x := 0; x < columns; x++ {
			w, g := cellText(wantCells, x), cellText(gotCells, x)
			if w == g {
				continue
			}
			diffs = append(diffs, SnapshotDiff{Row: y, Column: x, Expected: w, Actual: g})
		}
	}
	return diffs
}

// snapshotCells splits one snapshot row into one entry per screen column, so the
// index into the result is the column itself: a cell that takes two columns
// leaves the column it covers empty, and a cluster keeps its whole text in the
// column it starts at.
func snapshotCells(row string) []string {
	cells := make([]string, 0, len(row))
	var pending string
	for rest := row; rest != ""; {
		cluster, width, next := cell.NextCluster(rest)
		if width <= 0 {
			// A cluster that takes no column belongs to the cell beside it, and a
			// cluster that starts the row belongs to the first cell of it.
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
		for i := 1; i < width; i++ {
			cells = append(cells, "")
		}
		rest = next
	}
	if pending != "" && len(cells) > 0 {
		cells[len(cells)-1] += pending
	}
	return cells
}

// cellText returns the text a column holds. A column a wide cell covers, and a
// column past the end of a shorter row, hold a blank: the snapshots pad the
// cells they do not have, so the row a message names is the row that was drawn.
func cellText(cells []string, column int) string {
	if column >= len(cells) || cells[column] == "" {
		return " "
	}
	return cells[column]
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
