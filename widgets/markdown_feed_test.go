package widgets

import (
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// What HTML converted to Markdown looks like — a feed reader's articles. The
// converter escapes punctuation that would otherwise be read as markup; the
// reader must see the punctuation, not the backslashes.
func TestMarkdownBackslashEscapes(t *testing.T) {
	_, _, rows := drawMarkdown(t, "## 1\\. Investigate\nAirPods \\| Photo\n\\# Comments: 3\n\\- not a list", 40, 8)
	want := []string{"1. Investigate", "", "AirPods | Photo", "# Comments: 3", "- not a list"}
	for i, w := range want {
		if rows[i] != w {
			t.Fatalf("row %d = %q, want %q\nrows %q", i, rows[i], w, rows)
		}
	}
	// An escaped list marker or heading stays text: the first column is
	// the character, not a bullet.
	if _, _, r := drawMarkdown(t, "1\\. not numbered", 30, 1); r[0] != "1. not numbered" {
		t.Errorf("got %q", r[0])
	}
}

// "* * *" is how html-to-markdown writes <hr>.
func TestMarkdownSpacedRules(t *testing.T) {
	for _, src := range []string{"* * *", "- - -", "___", "*****"} {
		if _, _, rows := drawMarkdown(t, src, 5, 1); rows[0] != "┄┄┄┄┄" {
			t.Errorf("%q drew %q", src, rows[0])
		}
	}
	if _, _, rows := drawMarkdown(t, "* * not a rule", 20, 1); strings.Contains(rows[0], "┄") {
		t.Errorf("a list item drew as a rule: %q", rows[0])
	}
}

// A word wider than the row — a long address, in a terminal that cannot hide
// it behind a hyperlink — is broken across rows. It used to be cut off at
// the edge, and the rest of it lost.
func TestMarkdownBreaksWordsWiderThanTheRow(t *testing.T) {
	long := "https://example.com/" + strings.Repeat("abcdefghij", 4)
	_, _, rows := drawMarkdown(t, "see "+long+" end", 20, 6)
	joined := strings.Join(rows, "")
	if !strings.Contains(strings.ReplaceAll(joined, " ", ""), long) {
		t.Fatalf("the address was not drawn whole:\n%s", strings.Join(rows, "\n"))
	}
	if !strings.Contains(joined, "end") {
		t.Fatalf("the text after the address was lost:\n%s", strings.Join(rows, "\n"))
	}

	// In a quoted list item the continuation ("│ ") is narrower than the
	// first row's prefix ("│ • "); the address must still be broken there.
	_, _, rows = drawMarkdown(t, "> - see "+long+" end", 20, 8)
	joined = strings.Join(rows, "")
	if !strings.Contains(strings.NewReplacer(" ", "", "│", "").Replace(joined), long) {
		t.Fatalf("the address in a quote was not drawn whole:\n%s", strings.Join(rows, "\n"))
	}
}

// Markup inside a link's text is read: `[$179 at **Amazon**](url)` is bold
// where it says so, and the whole of it is the link.
func TestMarkdownStylesInsideALink(t *testing.T) {
	segs := parseInlineStyles("[$179 at **Amazon**](https://amazon.example)", cell.Style{}, true)
	if got := segText(segs); got != "$179 at Amazon" {
		t.Fatalf("drawn as %q", got)
	}
	for _, s := range segs {
		if s.Style.Link == 0 {
			t.Errorf("%q is not part of the link", s.Text)
		}
		if s.Text == "Amazon" && s.Style.Modifier&cell.ModifierBold == 0 {
			t.Error("**Amazon** is not bold")
		}
	}
}

// A link to a part of the page goes nowhere in a terminal: its text is
// drawn, without an address or a link. And a link whose text is its address
// does not write the address twice.
func TestMarkdownLinksWithNothingToAdd(t *testing.T) {
	segs := parseInlineStyles("[Logs](#1-logs)", cell.Style{}, false)
	if got := segText(segs); got != "Logs" {
		t.Errorf("a fragment link drew as %q", got)
	}
	segs = parseInlineStyles("[Logs](#1-logs)", cell.Style{}, true)
	if segs[0].Style.Link != 0 {
		t.Error("a fragment link was made clickable")
	}
	segs = parseInlineStyles("[https://go.dev](https://go.dev)", cell.Style{}, false)
	if got := segText(segs); got != "https://go.dev" {
		t.Errorf("an address as its own text drew as %q", got)
	}
}

// A theme colours what the default theme colours, and a heading with a
// background is drawn as a label, a space either side.
func TestMarkdownTheme(t *testing.T) {
	theme := DefaultMarkdownTheme
	pink := cell.NewColorRGB(255, 105, 180)
	theme.Headings[0] = cell.Style{Fg: cell.NewColorRGB(0, 0, 0), Bg: pink}
	theme.Bullet = cell.Style{Fg: pink}
	md := &Markdown{Content: "# Title\n- item", Theme: &theme}
	area := cell.NewRect(0, 0, 20, 4)
	buf := buffer.NewBuffer(area)
	md.Draw(cell.NewContext(area, cell.Style{}), buf)
	if row := strings.TrimRight(strings.Split(buf.Snapshot(), "\n")[0], " "); row != " Title" {
		t.Fatalf("heading row %q, want the title padded", row)
	}
	if c := buf.CellAt(0, 0); c.Style.Bg != pink {
		t.Errorf("the heading's padding is not on its background: %+v", c.Style)
	}
	if c := buf.CellAt(0, 2); c.Style.Fg != pink {
		t.Errorf("the bullet is not in the theme's colour: %+v", c.Style)
	}

	// Changing the theme redraws in the new colours: the parse cache keys
	// on it.
	theme.Bullet = cell.Style{Fg: cell.NewColorRGB(1, 2, 3)}
	buf = buffer.NewBuffer(area)
	md.Draw(cell.NewContext(area, cell.Style{}), buf)
	if c := buf.CellAt(0, 2); c.Style.Fg != theme.Bullet.Fg {
		t.Errorf("a changed theme was not redrawn: %+v", c.Style)
	}
}

// A main heading is set off by one blank row: its own when text follows
// it at once, the source's when there is one. Two were added whatever the
// source had, so "# Title" and a blank line made three.
func TestMarkdownHeadingSpacing(t *testing.T) {
	for _, src := range []string{"# Title\ntext", "# Title\n\ntext"} {
		_, _, rows := drawMarkdown(t, src, 20, 4)
		if rows[0] != "Title" || rows[1] != "" || rows[2] != "text" {
			t.Errorf("%q drew %q", src, rows[:3])
		}
	}
}
