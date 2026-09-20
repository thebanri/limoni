package widgets

import (
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/cell"
)

func segText(segs []rawSegment) string {
	var b strings.Builder
	for _, s := range segs {
		b.WriteString(s.Text)
	}
	return b.String()
}

// With OSC 8 the label alone is drawn, and it carries the link.
func TestMarkdownLinkBecomesAHyperlink(t *testing.T) {
	segs := parseInlineStyles("see the [changelog](https://example.com/c) for more", cell.Style{}, true)
	if got := segText(segs); got != "see the changelog for more" {
		t.Errorf("text = %q, want the URL hidden behind the label", got)
	}
	var linked int
	for _, s := range segs {
		if s.Style.LinkURL() == "https://example.com/c" {
			linked++
			if s.Text != "changelog" {
				t.Errorf("the link covers %q, want just the label", s.Text)
			}
			if !s.Style.HasModifier(cell.ModifierUnderline) {
				t.Error("a link should be recognisable without hovering it")
			}
		}
	}
	if linked != 1 {
		t.Errorf("%d linked segments, want 1", linked)
	}
}

// Without OSC 8 the address has to stay visible, or the reader loses it.
func TestMarkdownLinkKeepsTheURLVisibleWithoutSupport(t *testing.T) {
	segs := parseInlineStyles("see the [changelog](https://example.com/c) for more", cell.Style{}, false)
	got := segText(segs)
	if !strings.Contains(got, "changelog") || !strings.Contains(got, "https://example.com/c") {
		t.Errorf("text = %q, want both the label and the address", got)
	}
	for _, s := range segs {
		if s.Style.Link != 0 {
			t.Errorf("segment %q carries a link although the terminal cannot show one", s.Text)
		}
	}
}

// Anything that is not a complete inline link is text, not a broken link.
func TestMarkdownNonLinksAreLeftAlone(t *testing.T) {
	for _, src := range []string{
		"an [unclosed link",
		"a [ref][1] style link",
		"empty [](https://example.com)",
		"no target []()",
		"a [title](https://example.com \"hi\") with a title",
		"brackets [like this] alone",
	} {
		segs := parseInlineStyles(src, cell.Style{}, true)
		if got := segText(segs); got != src {
			t.Errorf("%q was rewritten to %q", src, got)
		}
		for _, s := range segs {
			if s.Style.Link != 0 {
				t.Errorf("%q produced a link", src)
			}
		}
	}
}

// The parse cache keys on the capability too: the same widget drawn into a
// terminal with hyperlinks and one without must not reuse the other's rows.
func TestMarkdownReparsesWhenTheCapabilityChanges(t *testing.T) {
	md := &Markdown{Content: "[label](https://example.com/x)"}
	md.parse(cell.Style{}, false)
	without := len(md.cachedLines[0].segments)
	md.parse(cell.Style{}, true)
	with := len(md.cachedLines[0].segments)
	if without == with {
		t.Errorf("the same %d segments for both, so the capability was ignored", with)
	}
}
