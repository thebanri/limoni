package widgets

import (
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

func drawMarkdown(t *testing.T, src string, w, h uint16) (*Markdown, *buffer.Buffer, []string) {
	t.Helper()
	md := &Markdown{Content: src}
	area := cell.NewRect(0, 0, w, h)
	buf := buffer.NewBuffer(area)
	md.Draw(cell.NewContext(area, cell.Style{}), buf)
	rows := strings.Split(buf.Snapshot(), "\n")
	for i := range rows {
		rows[i] = strings.TrimRight(rows[i], " ")
	}
	return md, buf, rows
}

// What an LLM writes: deeper headings, numbered and nested lists, tasks,
// quotes, a fenced code block and a table all come out as blocks, not as the
// markup that spells them.
func TestMarkdownDrawsTheBlocksRepliesUse(t *testing.T) {
	src := strings.Join([]string{
		"### Steps",
		"1. Install",
		"2. Run",
		"- top",
		"  - nested",
		"- [ ] todo",
		"- [x] done",
		"> quoted words that wrap",
		"```go",
		"func main() {}",
		"```",
		"| Name | Age |",
		"| :--- | --: |",
		"| Ada  | 36  |",
		"~~gone~~ and 2 * 3 * 4",
	}, "\n")
	_, buf, rows := drawMarkdown(t, src, 16, 20)
	want := []string{
		"Steps",
		"1. Install",
		"2. Run",
		"• top",
		"  • nested",
		"• [ ] todo",
		"• [x] done",
		"│ quoted words",
		"│ that wrap",
		" func main() {}",
		" Name │ Age",
		"──────┼─────",
		" Ada  │  36",
		"gone and 2 * 3",
		"* 4",
	}
	for i, w := range want {
		if i >= len(rows) || rows[i] != w {
			t.Fatalf("row %d = %q, want %q\nscreen:\n%s", i, rows[i], w, buf.Snapshot())
		}
	}
	// The code is highlighted: "func" is a keyword, on the code background.
	if c := buf.CellAt(1, 9); c.Style.Bg != DefaultMarkdownTheme.CodeBlock.Bg || c.Style.Fg != DefaultCodeTheme[TokenKeyword].Fg {
		t.Errorf("func is not highlighted as a keyword on the code background: %+v", c.Style)
	}
	if c := buf.CellAt(0, 13); c.Style.Modifier&cell.ModifierStrikethrough == 0 {
		t.Errorf("~~gone~~ is not struck through")
	}
	if c := buf.CellAt(13, 13); c.Content != '3' || c.Style.Modifier&cell.ModifierItalic != 0 {
		t.Errorf("a star between spaces turned the text italic")
	}
}

// A reply still streaming in may have opened a code block and not closed it:
// what has arrived is code, not prose.
func TestMarkdownUnclosedFenceIsCode(t *testing.T) {
	_, buf, rows := drawMarkdown(t, "Here:\n```python\ndef f():\n    return 1", 20, 5)
	if rows[1] != " def f():" || rows[2] != "     return 1" {
		t.Fatalf("rows %q", rows[:3])
	}
	if buf.CellAt(0, 1).Style.Bg != DefaultMarkdownTheme.CodeBlock.Bg {
		t.Errorf("the unclosed block has no code background")
	}
}

// A table wider than the area gives up width from its widest column and
// marks the cut.
func TestMarkdownTableShrinksToFit(t *testing.T) {
	_, _, rows := drawMarkdown(t, "| a | a very long description |\n|---|---|\n| x | yyyyyyyyyyyyyyyyyyyyyy |", 20, 4)
	for _, r := range rows[:3] {
		if cell.StringWidth(r) > 20 {
			t.Errorf("row %q is wider than 20", r)
		}
	}
	if !strings.HasSuffix(rows[2], "…") {
		t.Errorf("the cut cell has no ellipsis: %q", rows[2])
	}
}

// The semantic tree gets the words, not the markup.
func TestMarkdownAccessibilityIsPlainText(t *testing.T) {
	md, _, _ := drawMarkdown(t, "# Title\n**bold** and `code`\n- item", 30, 6)
	node := md.AccessibilityNode(cell.NewRect(0, 0, 30, 6), false)
	if node.Value != "Title\nbold and code\nitem" {
		t.Errorf("value %q", node.Value)
	}
}
