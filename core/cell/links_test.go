package cell

import (
	"runtime"
	"strings"
	"testing"
)

func TestLinkInterningIsStableAndReversible(t *testing.T) {
	a := Link("https://example.com/one")
	b := Link("https://example.com/two")
	if a == 0 || b == 0 {
		t.Fatal("a URL should always get a handle")
	}
	if a == b {
		t.Error("two URLs share a handle")
	}
	if again := Link("https://example.com/one"); again != a {
		t.Errorf("the same URL got two handles: %d then %d", a, again)
	}
	if got := LinkURL(a); got != "https://example.com/one" {
		t.Errorf("LinkURL(%d) = %q", a, got)
	}
	if got := LinkURL(0); got != "" {
		t.Errorf("the zero handle means no link, got %q", got)
	}
	if got := LinkURL(LinkID(1 << 15)); got != "" {
		t.Errorf("an unknown handle should read back empty, got %q", got)
	}
}

func TestEmptyURLIsNotALink(t *testing.T) {
	if id := Link(""); id != 0 {
		t.Errorf("empty URL got handle %d", id)
	}
	s := NewStyle().WithLink("https://example.com/x").WithLink("")
	if s.Link != 0 {
		t.Error("an empty URL should clear the link")
	}
}

func TestStyleCarriesTheLinkThroughHelpers(t *testing.T) {
	const url = "https://example.com/style"
	s := NewStyle().WithLink(url).Bold().WithFg(NewColorANSI(4))
	if s.LinkURL() != url {
		t.Errorf("LinkURL() = %q", s.LinkURL())
	}
	if !s.HasModifier(ModifierBold) || s.Fg != NewColorANSI(4) {
		t.Error("WithLink interfered with the rest of the style")
	}

	// Merge is how a widget's style meets its container's.
	merged := NewStyle().WithFg(NewColorANSI(1)).Merge(NewStyle().WithLink(url))
	if merged.LinkURL() != url {
		t.Error("Merge dropped the link")
	}
	if merged.Fg != NewColorANSI(1) {
		t.Error("Merge lost the base colour")
	}

	var reset Style
	reset = s
	reset.Reset()
	if reset.Link != 0 {
		t.Error("Reset left a link behind, so a default style would still be clickable")
	}
}

// The table stores its own copy: a URL sliced out of a large document must
// not keep that document alive.
func TestLinkDoesNotPinTheCallersBuffer(t *testing.T) {
	const url = "https://example.com/sliced"
	doc := strings.Repeat("x", 1<<12) + url
	id := Link(doc[len(doc)-len(url):])
	if got := LinkURL(id); got != url {
		t.Fatalf("LinkURL = %q", got)
	}
	// A stored URL that still aliased the document would carry its capacity
	// with it, keeping 4 KiB alive for a 26-byte link.
	if stored := LinkURL(id); cap([]byte(stored)) > 64 {
		t.Errorf("stored URL looks like a slice of the document: cap %d", cap([]byte(stored)))
	}
}

func TestLinkLookupIsAllocationFree(t *testing.T) {
	const url = "https://example.com/hot-path"
	Link(url)
	if n := testing.AllocsPerRun(100, func() {
		_ = Link(url)
		runtime.GC()
	}); n != 0 {
		t.Errorf("Link allocated %v times per lookup; widgets call this while drawing", n)
	}
}
