package widgets

import (
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// A wide mask rune in a secret field draws as itself. The mask is written a
// cell and its continuation at a time, and the continuation used to blank
// the glyph before it.
func TestTextInputDrawsWideMask(t *testing.T) {
	st := NewTextInputState()
	area := cell.NewRect(0, 0, 12, 1)
	buf := buffer.NewBuffer(area)
	st.SetValue("ab")
	TextInput{ID: "pw", State: st, Secret: true, MaskRune: '●', SelectionStart: -1, SelectionEnd: -1}.Draw(cell.NewContext(area, cell.Style{}), buf)
	TextInput{ID: "pw", State: st, Secret: true, MaskRune: '中', SelectionStart: -1, SelectionEnd: -1}.Draw(cell.NewContext(area, cell.Style{}), buf)
	if got := strings.TrimRight(buf.Snapshot(), " "); got != "中 中" {
		t.Errorf("wide mask drawn %q", got)
	}
}
