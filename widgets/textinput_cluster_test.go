package widgets

import (
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

// Written as escapes so an editor cannot normalise them away.
const (
	tiFamily = "\U0001F468‍\U0001F469‍\U0001F467" // 5 code points, 2 columns
	tiFlag   = "\U0001F1F9\U0001F1F7"                       // 2 code points, 2 columns
	tiEAcute = "é"                                    // 2 code points, 1 column
)

func key(t driver.KeyType) driver.KeyEvent { return driver.KeyEvent{Type: t} }

// Editing moves by what the reader sees as one character. Walking runes left
// "man ZWJ woman ZWJ" behind after one Backspace on a family emoji.
func TestTextInputEditsWholeClusters(t *testing.T) {
	state := NewTextInputState()
	state.SetValue("a" + tiFamily + tiFlag + tiEAcute + "z")

	steps := []struct {
		key  driver.KeyType
		want string // text with | at the cursor
	}{
		{driver.KeyBackspace, "a" + tiFamily + tiFlag + tiEAcute + "|"},
		{driver.KeyArrowLeft, "a" + tiFamily + tiFlag + "|" + tiEAcute},
		{driver.KeyArrowLeft, "a" + tiFamily + "|" + tiFlag + tiEAcute},
		{driver.KeyBackspace, "a|" + tiFlag + tiEAcute},
		{driver.KeyDelete, "a|" + tiEAcute},
		{driver.KeyArrowRight, "a" + tiEAcute + "|"},
		{driver.KeyArrowLeft, "a|" + tiEAcute},
		{driver.KeyDelete, "a|"},
	}
	for i, step := range steps {
		state.HandleKey(key(step.key))
		got := string(state.Text[:state.Cursor]) + "|" + string(state.Text[state.Cursor:])
		if got != step.want {
			t.Fatalf("step %d: got %q, want %q", i, got, step.want)
		}
	}
}

// Wide characters take two columns and the cursor lands after them. Drawing
// one rune per cell put "本" on top of the second half of "日".
func TestTextInputDrawsWideCharactersAndClusters(t *testing.T) {
	state := NewTextInputState()
	state.SetValue("日本" + tiFamily + "x")
	buf := drawInput(t, TextInput{ID: "in", State: state, Focused: true}, 12)

	if got := rowText(buf, 12); got != "日本"+tiFamily+"x" {
		t.Fatalf("row reads %q", got)
	}
	// 日 0-1, 本 2-3, family 4-5, x 6, cursor at 7.
	if c := buf.Get(7, 0); c.Style.Modifier&cell.ModifierReverse == 0 {
		t.Errorf("cursor not at column 7")
	}
	if c := buf.Get(2, 0); c.Content != '本' {
		t.Errorf("column 2 holds %q, want 本", c.Content)
	}
}

// Scrolling to keep the cursor visible never draws half of a wide character.
func TestTextInputScrollsByColumns(t *testing.T) {
	state := NewTextInputState()
	state.SetValue(strings.Repeat("日", 6)) // 12 columns
	buf := drawInput(t, TextInput{ID: "in", State: state, Focused: true}, 5)
	// Cursor at column 12 needs the window to end there: columns 8..12, so
	// 日 at 8-9 and 10-11 are drawn and the cursor cell is blank.
	if got := rowText(buf, 5); got != "日日" {
		t.Fatalf("visible text %q, want 日日", got)
	}
	if c := buf.Get(4, 0); c.Style.Modifier&cell.ModifierReverse == 0 {
		t.Errorf("cursor not in the last column")
	}
}

// A secret field shows one mask per character, and the character here is a
// cluster: five masks for one family emoji would say how it is built.
func TestTextInputSecretMasksClusters(t *testing.T) {
	state := NewTextInputState()
	state.SetValue(tiFamily + "ab")
	buf := drawInput(t, TextInput{ID: "in", State: state, Secret: true}, 10)
	if got := rowText(buf, 10); got != "•••" {
		t.Fatalf("masked row %q, want three masks", got)
	}
}

// Drawing must not allocate, emoji included, once the text has been seen.
func TestTextInputDrawDoesNotAllocate(t *testing.T) {
	state := NewTextInputState()
	state.SetValue("hello " + tiFamily + " 日本 " + tiFlag)
	buf := buffer.NewBuffer(cell.NewRect(0, 0, 30, 1))
	ctx := cell.Context{Area: cell.NewRect(0, 0, 30, 1)}
	ti := TextInput{ID: "in", State: state, Focused: true, SelectionStart: 1, SelectionEnd: 4}
	ti.Draw(ctx, buf) // interns the clusters once
	if a := testing.AllocsPerRun(100, func() { ti.Draw(ctx, buf) }); a != 0 {
		t.Fatalf("Draw allocates %.0f times per frame", a)
	}
	if a := testing.AllocsPerRun(100, func() { _ = ti.AccessibilityNode(ctx.Area, true) }); a != 0 {
		t.Fatalf("AccessibilityNode allocates %.0f times", a)
	}
}

func drawInput(t *testing.T, ti TextInput, width uint16) *buffer.Buffer {
	t.Helper()
	buf := buffer.NewBuffer(cell.NewRect(0, 0, width, 1))
	ti.Draw(cell.Context{Area: cell.NewRect(0, 0, width, 1)}, buf)
	return buf
}

// rowText reads row 0 back as text, skipping continuation cells and trimming
// trailing blanks.
func rowText(buf *buffer.Buffer, width uint16) string {
	var sb strings.Builder
	for x := uint16(0); x < width; x++ {
		c := buf.Get(x, 0)
		if c.Content == cell.RuneContinuation {
			continue
		}
		sb.WriteString(cell.ClusterText(c.Content))
	}
	return strings.TrimRight(sb.String(), " ")
}
