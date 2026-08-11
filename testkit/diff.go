package testkit

import (
	"fmt"
	"strings"
)

type SnapshotDiff struct {
	Row, Column      int
	Expected, Actual rune
}

func DiffSnapshot(expected, actual string) []SnapshotDiff {
	want, got := strings.Split(expected, "\n"), strings.Split(actual, "\n")
	max := len(want)
	if len(got) > max {
		max = len(got)
	}
	var diffs []SnapshotDiff
	for y := 0; y < max; y++ {
		var ws, gs string
		if y < len(want) {
			ws = want[y]
		}
		if y < len(got) {
			gs = got[y]
		}
		wr, gr := []rune(ws), []rune(gs)
		width := len(wr)
		if len(gr) > width {
			width = len(gr)
		}
		for x := 0; x < width; x++ {
			a, b := ' ', ' '
			if x < len(wr) {
				a = wr[x]
			}
			if x < len(gr) {
				b = gr[x]
			}
			if a != b {
				diffs = append(diffs, SnapshotDiff{Row: y, Column: x, Expected: a, Actual: b})
			}
		}
	}
	return diffs
}

func FormatSnapshotDiff(diffs []SnapshotDiff) string {
	var b strings.Builder
	for _, diff := range diffs {
		fmt.Fprintf(&b, "row=%d col=%d expected=%q actual=%q\n", diff.Row, diff.Column, diff.Expected, diff.Actual)
	}
	return b.String()
}
