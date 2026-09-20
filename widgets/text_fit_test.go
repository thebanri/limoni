package widgets

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// screenText reads every row of buf back as text, one line per row.
func screenText(buf *buffer.Buffer) string {
	var sb strings.Builder
	for y := uint16(0); y < buf.Area.Height; y++ {
		for x := uint16(0); x < buf.Area.Width; x++ {
			if c := buf.Get(x, y); c.Content != cell.RuneContinuation {
				sb.WriteString(cell.ClusterText(c.Content))
			}
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

// A table cell that has to be cut keeps whole characters. Cutting by code
// point width once returned "e" plus half of U+0301 — invalid UTF-8 — and
// counted a family emoji as six columns.
func TestTableCutsCellsAtClusters(t *testing.T) {
	table := Table{
		Rows:        []TableRow{NewRow("e\u0301e\u0301e\u0301e\u0301e\u0301e\u0301"), NewRow("日本語日本語"), NewRow("\U0001F468\u200D\U0001F469\u200D\U0001F467 family")},
		Constraints: []TableConstraint{{Type: ConstraintFixed, Value: 5}},
		State:       NewTableState(),
	}
	buf := buffer.NewBuffer(cell.NewRect(0, 0, 5, 3))
	table.Draw(cell.Context{Area: buf.Area}, buf)
	got := screenText(buf)
	if !utf8.ValidString(got) {
		t.Fatalf("table produced invalid UTF-8: %q", got)
	}
	for _, want := range []string{"e\u0301e\u0301...", "日...", "\U0001F468\u200D\U0001F469\u200D\U0001F467..."} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in\n%s", want, got)
		}
	}
}

// A toast whose title is too long ends in "…" inside its box, measured in
// columns: 日本語 counted as three runes overflowed the box by three columns.
func TestToastEllipsizesInColumns(t *testing.T) {
	tm := NewToastManager(ToastTopRight)
	tm.Info(strings.Repeat("日本語", 20), "short")
	buf := buffer.NewBuffer(cell.NewRect(0, 0, 80, 10))
	tm.Draw(cell.Context{Area: buf.Area}, buf)
	for _, line := range strings.Split(screenText(buf), "\n") {
		if strings.Contains(line, "日本語") {
			if !strings.Contains(line, "…") {
				t.Errorf("long title not ellipsized: %q", line)
			}
			if w := cell.StringWidth(strings.TrimRight(line, " ")); w > 80 {
				t.Errorf("title row is %d columns wide", w)
			}
		}
	}
}
