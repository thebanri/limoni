package terminal_test

import (
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/widgets"
)

// Compiling is not verifying: this draws through a real Terminal and reads
// the bytes it puts on the wire, both with the capability and without it.
func drawWithLinks(t *testing.T, links bool, draw func(f *terminal.Frame)) string {
	t.Helper()
	t.Setenv("LIMONI_PROBE", "0")
	io := driver.NewMemoryTerminalIO(nil, 60, 6)
	b := driver.NewPortableBackend(io)
	if err := b.Setup(); err != nil {
		t.Fatal(err)
	}
	term, err := terminal.New(b)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = term.Close() }()

	caps := term.Capabilities()
	caps.Hyperlinks = links
	term.SetCapabilities(caps)

	before := len(io.Output())
	if err := term.Draw(draw); err != nil {
		t.Fatalf("draw: %v", err)
	}
	return string(io.Output()[before:])
}

func TestFrameWritesOSC8ForALinkedStyle(t *testing.T) {
	const url = "https://example.com/frame"
	draw := func(f *terminal.Frame) {
		f.Buffer.SetString(1, 1, "click me", cell.NewStyle().WithLink(url))
	}

	got := drawWithLinks(t, true, draw)
	if !strings.Contains(got, url) {
		t.Errorf("no OSC 8 on the wire: %q", got)
	}
	if !strings.Contains(got, "\x1b]8;;\x1b\\") {
		t.Errorf("the link was never closed: %q", got)
	}

	if off := drawWithLinks(t, false, draw); strings.Contains(off, "\x1b]8") {
		t.Errorf("OSC 8 written to a terminal that cannot show it: %q", off)
	}
}

// The whole path a reader actually meets: markdown source with an inline
// link, drawn by the widget, into a terminal that supports OSC 8.
func TestMarkdownLinkReachesTheWire(t *testing.T) {
	const url = "https://example.com/markdown"
	draw := func(f *terminal.Frame) {
		f.RenderWidget(widgets.NewMarkdown("see the [changelog]("+url+") today"), f.Area())
	}

	got := drawWithLinks(t, true, draw)
	if !strings.Contains(got, url) {
		t.Errorf("the markdown link never reached the terminal: %q", got)
	}
	// With the link clickable, the address is not also printed as text.
	if strings.Contains(stripOSC8(got), url) {
		t.Errorf("the URL was printed as text as well: %q", got)
	}

	off := drawWithLinks(t, false, draw)
	if strings.Contains(off, "\x1b]8") {
		t.Errorf("OSC 8 without the capability: %q", off)
	}
	if !strings.Contains(off, "example.com/markdown") {
		t.Errorf("without hyperlinks the address must stay readable: %q", off)
	}
}

// stripOSC8 removes the hyperlink sequences, leaving what the reader sees.
func stripOSC8(s string) string {
	var b strings.Builder
	for {
		i := strings.Index(s, "\x1b]8;")
		if i < 0 {
			b.WriteString(s)
			return b.String()
		}
		b.WriteString(s[:i])
		rest := s[i:]
		j := strings.Index(rest, "\x1b\\")
		if j < 0 {
			return b.String()
		}
		s = rest[j+2:]
	}
}
