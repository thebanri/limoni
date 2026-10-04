package widgets

import (
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

func drawTextArea(a TextArea, w, h uint16) (*buffer.Buffer, []string) {
	area := cell.NewRect(0, 0, w, h)
	buf := buffer.NewBuffer(area)
	a.Draw(cell.NewContext(area, cell.Style{}), buf)
	rows := strings.Split(buf.Snapshot(), "\n")
	for i := range rows {
		rows[i] = strings.TrimRight(rows[i], " ")
	}
	return buf, rows
}

// A long line wraps at the width, the view follows the cursor down, the
// cursor is drawn where it is, and a frame allocates nothing.
func TestTextAreaWrapsScrollsAndShowsTheCursor(t *testing.T) {
	st := NewTextAreaState()
	st.SetValue("one two three\nfour")
	a := TextArea{ID: "msg", State: st, Focused: true}

	buf, rows := drawTextArea(a, 6, 2)
	// "one two three" is 13 columns: rows "one tw", "o thre", "e", then
	// "four". The cursor is after "four", on the last row, so the view
	// shows the last two.
	if rows[0] != "e" || rows[1] != "four" {
		t.Fatalf("rows %q, want the last two wrapped rows", rows)
	}
	if c := buf.CellAt(4, 1); c.Style.Modifier&cell.ModifierReverse == 0 {
		t.Errorf("no cursor after the last word")
	}

	st.HandleKey(driver.KeyEvent{Type: driver.KeyArrowUp})
	if st.Cursor != 4 {
		t.Errorf("up from column 4 of line 2 went to %d, want 4 (column 4 of line 1)", st.Cursor)
	}
	st.Cursor = 0
	_, rows = drawTextArea(a, 6, 2)
	if rows[0] != "one tw" {
		t.Errorf("with the cursor at the start the view did not scroll back: %q", rows)
	}

	area := cell.NewRect(0, 0, 6, 2)
	b := buffer.NewBuffer(area)
	ctx := cell.NewContext(area, cell.Style{})
	if n := testing.AllocsPerRun(20, func() { a.Draw(ctx, b) }); n != 0 {
		t.Errorf("a frame allocated %v times", n)
	}
}

func TestTextAreaPlaceholder(t *testing.T) {
	buf, rows := drawTextArea(TextArea{State: NewTextAreaState(), Placeholder: "Message"}, 12, 1)
	if rows[0] != "Message" || buf.CellAt(0, 0).Style.Modifier&cell.ModifierDim == 0 {
		t.Errorf("placeholder row %q", rows[0])
	}
}
