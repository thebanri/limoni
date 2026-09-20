package buffer

import (
	"runtime"
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/cell"
)

func linkStyle(url string) cell.Style { return cell.NewStyle().WithLink(url) }

// The span is opened once with OSC 8, closed when the link ends, and the URL
// travels with the cells rather than with the order they happen to be drawn.
func TestDiffEmitsAndClosesOSC8(t *testing.T) {
	area := cell.NewRect(0, 0, 40, 3)
	front, back := NewBuffer(area), NewBuffer(area)
	st := linkStyle("https://example.com/a")
	for x, r := range "link" {
		front.SetCellDirect(uint16(x), 0, cell.Cell{Content: r, Style: st})
	}
	front.SetCellDirect(4, 0, cell.Cell{Content: '!', Style: cell.Style{}})

	out, err := DiffWithOptions(front, back, nil, DiffOptions{Hyperlinks: true})
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	got := string(out)

	id := cell.Link("https://example.com/a")
	open := "\x1b]8;id=" + itoa(int(id)) + ";https://example.com/a\x1b\\"
	if !strings.Contains(got, open) {
		t.Errorf("no OSC 8 open in %q", got)
	}
	if strings.Count(got, open) != 1 {
		t.Errorf("the link should be opened once, got %d in %q", strings.Count(got, open), got)
	}
	if !strings.Contains(got, "\x1b]8;;\x1b\\") {
		t.Errorf("the link is never closed: %q", got)
	}
	if i, j := strings.Index(got, open), strings.Index(got, "\x1b]8;;\x1b\\"); i > j {
		t.Errorf("closed before it was opened: %q", got)
	}
	assertSynced(t, front, back)
}

// Without the capability, the stream must be byte-for-byte what the same
// text without a link produces: not merely free of OSC 8, but free of any
// other difference the link handle might have caused in the style tracking.
func TestWithoutTheCapabilityTheStreamIsUnchanged(t *testing.T) {
	area := cell.NewRect(0, 0, 40, 3)
	render := func(st cell.Style) string {
		front, back := NewBuffer(area), NewBuffer(area)
		for x, r := range "a link that is quite long" {
			front.SetCellDirect(uint16(x), 0, cell.Cell{Content: r, Style: st})
		}
		out, err := DiffWithOptions(front, back, nil, DiffOptions{Hyperlinks: false})
		if err != nil {
			t.Fatalf("diff: %v", err)
		}
		return string(out)
	}
	green := cell.NewStyle().WithFg(cell.NewColorANSI(2))
	withLink := render(green.WithLink("https://example.com/b"))
	plain := render(green)
	if withLink != plain {
		t.Errorf("link changed the stream although the terminal cannot show it:\n with link: %q\n plain:     %q", withLink, plain)
	}
	if strings.Contains(withLink, "\x1b]8") {
		t.Errorf("OSC 8 emitted without the capability: %q", withLink)
	}
}

// A link that wraps onto the next row is one link, and the id is what tells
// the terminal so: hovering either half should highlight both.
func TestALinkSpanningRowsKeepsOneID(t *testing.T) {
	area := cell.NewRect(0, 0, 6, 2)
	front, back := NewBuffer(area), NewBuffer(area)
	st := linkStyle("https://example.com/wrapped")
	for x := uint16(0); x < 6; x++ {
		front.SetCellDirect(x, 0, cell.Cell{Content: 'a', Style: st})
		front.SetCellDirect(x, 1, cell.Cell{Content: 'b', Style: st})
	}
	out, err := DiffWithOptions(front, back, nil, DiffOptions{Hyperlinks: true})
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	got := string(out)
	id := "id=" + itoa(int(cell.Link("https://example.com/wrapped")))
	if n := strings.Count(got, id); n == 0 {
		t.Fatalf("no id in %q", got)
	}
	// However many times the link is re-opened across the rows, every open
	// carries the same id.
	for _, part := range strings.Split(got, "\x1b]8;")[1:] {
		if part[0] == ';' {
			continue // the close
		}
		if !strings.HasPrefix(part, id+";") {
			t.Errorf("an open with a different id: %q in %q", part, got)
		}
	}
}

// Only the link changed: same glyph, same colours. The cell still has to be
// redrawn, or the previous frame's link stays under the new text.
func TestChangingOnlyTheLinkRedrawsTheCell(t *testing.T) {
	area := cell.NewRect(0, 0, 10, 1)
	front, back := NewBuffer(area), NewBuffer(area)
	old := linkStyle("https://example.com/old")
	fresh := linkStyle("https://example.com/new")
	for x := uint16(0); x < 4; x++ {
		back.SetCellDirect(x, 0, cell.Cell{Content: 'x', Style: old})
		front.SetCellDirect(x, 0, cell.Cell{Content: 'x', Style: fresh})
	}
	front.IsDirty = true
	out, err := DiffWithOptions(front, back, nil, DiffOptions{Hyperlinks: true})
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	if !strings.Contains(string(out), "https://example.com/new") {
		t.Errorf("the new link was never written: %q", out)
	}
	assertSynced(t, front, back)
}

// Two links next to each other must not run together: the second open ends
// the first, and each keeps its own id so a terminal can tell them apart.
func TestAdjacentLinksKeepTheirOwnIDs(t *testing.T) {
	area := cell.NewRect(0, 0, 40, 1)
	front, back := NewBuffer(area), NewBuffer(area)
	a, b := linkStyle("https://example.com/1"), linkStyle("https://example.com/2")
	front.SetCellDirect(0, 0, cell.Cell{Content: 'a', Style: a})
	front.SetCellDirect(1, 0, cell.Cell{Content: 'b', Style: b})

	out, err := DiffWithOptions(front, back, nil, DiffOptions{Hyperlinks: true})
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	got := string(out)
	if !strings.Contains(got, "https://example.com/1") || !strings.Contains(got, "https://example.com/2") {
		t.Fatalf("both URLs should appear: %q", got)
	}
	if cell.Link("https://example.com/1") == cell.Link("https://example.com/2") {
		t.Error("distinct URLs share a handle")
	}
}

// A style reset (ESC[0m) does not close a hyperlink, so the encoder must not
// believe it did: dropping a modifier inside a link used to leave the encoder
// thinking no link was open, and the close was never written.
func TestDroppingAModifierInsideALinkStillClosesIt(t *testing.T) {
	area := cell.NewRect(0, 0, 40, 1)
	front, back := NewBuffer(area), NewBuffer(area)
	bold := linkStyle("https://example.com/c").Bold()
	plain := linkStyle("https://example.com/c")
	front.SetCellDirect(0, 0, cell.Cell{Content: 'A', Style: bold})
	front.SetCellDirect(1, 0, cell.Cell{Content: 'b', Style: plain})
	front.SetCellDirect(2, 0, cell.Cell{Content: '.', Style: cell.Style{}})

	out, err := DiffWithOptions(front, back, nil, DiffOptions{Hyperlinks: true})
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	got := string(out)
	if n := strings.Count(got, "https://example.com/c"); n != 1 {
		t.Errorf("the link should be opened once across the reset, got %d: %q", n, got)
	}
	if !strings.Contains(got, "\x1b]8;;\x1b\\") {
		t.Errorf("the link is never closed: %q", got)
	}
}

// The draw path's budget is zero, and interning a URL that has been seen
// before must not allocate either.
func TestLinkLookupDoesNotAllocate(t *testing.T) {
	const url = "https://example.com/allocation"
	cell.Link(url) // first sight interns it
	if n := testing.AllocsPerRun(100, func() {
		_ = cell.NewStyle().WithLink(url)
		runtime.GC()
	}); n != 0 {
		t.Errorf("WithLink allocated %v times per call", n)
	}
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var b [8]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	return string(b[i:])
}
