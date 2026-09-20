package cell

import "testing"

func renderRich(t *testing.T, src string, resolver StyleResolver) []Cell {
	t.Helper()
	return ParseRichText(src, Style{}, resolver)
}

func richText(cells []Cell) string {
	out := make([]rune, 0, len(cells))
	for _, c := range cells {
		out = append(out, c.Content)
	}
	return string(out)
}

// Colours can be named, numbered or hexadecimal, and the short hex form has
// to expand the way CSS does — #f00 is #ff0000, not #0f0000.
func TestParseColorForms(t *testing.T) {
	cases := []struct {
		in   string
		want Color
	}{
		{"red", NewColorANSI(1)},
		{"BrightCyan", NewColorANSI(14)},
		{"9", NewColorANSI(9)},
		{"255", NewColorANSI(255)},
		{"#ff5733", NewColorRGB(0xFF, 0x57, 0x33)},
		{"#f00", NewColorRGB(255, 0, 0)},
		{"#0f0", NewColorRGB(0, 255, 0)},
		{"256", NewColorDefault()},        // out of the ANSI range
		{"-1", NewColorDefault()},         //
		{"chartreuse", NewColorDefault()}, // not a name we know
		{"#ff57", NewColorDefault()},      // not a hex length we know
		{"", NewColorDefault()},
	}
	for _, c := range cases {
		if got := parseColor(c.in); got != c.want {
			t.Errorf("parseColor(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestRichTextTagsApplyAndUnwind(t *testing.T) {
	cells := renderRich(t, "plain <fg=red,bold>hot</> plain", nil)
	if got := richText(cells); got != "plain hot plain" {
		t.Fatalf("text = %q, the tags should not be drawn", got)
	}
	hot := cells[6] // "h" of "hot"
	if hot.Style.Fg != NewColorANSI(1) || !hot.Style.HasModifier(ModifierBold) {
		t.Errorf("inside the tag: %+v", hot.Style)
	}
	after := cells[len(cells)-1]
	if after.Style != (Style{}) {
		t.Errorf("the closing tag did not unwind the style: %+v", after.Style)
	}
}

func TestRichTextNestingAndModifiers(t *testing.T) {
	cells := renderRich(t, "<bold>a<italic>b</>c</>d", nil)
	if got := richText(cells); got != "abcd" {
		t.Fatalf("text = %q", got)
	}
	mods := func(i int) Modifier { return cells[i].Style.Modifier }
	if mods(0) != ModifierBold {
		t.Errorf("a: %v", mods(0))
	}
	if mods(1) != ModifierBold|ModifierItalic {
		t.Errorf("b: %v", mods(1))
	}
	if mods(2) != ModifierBold {
		t.Errorf("c: the inner tag should have closed: %v", mods(2))
	}
	if mods(3) != 0 {
		t.Errorf("d: everything should have closed: %v", mods(3))
	}
}

// An unknown bare tag is a class name, resolved by the application's theme.
func TestRichTextResolvesRolesAndClasses(t *testing.T) {
	resolver := func(name string) Style {
		if name == "success" {
			return NewStyle().WithFg(NewColorANSI(2)).Bold()
		}
		return Style{}
	}
	for _, src := range []string{"<success>ok</>", "<role=success>ok</>", "<class=success>ok</>"} {
		cells := renderRich(t, src, resolver)
		if richText(cells) != "ok" {
			t.Fatalf("%s: text = %q", src, richText(cells))
		}
		if cells[0].Style.Fg != NewColorANSI(2) || !cells[0].Style.HasModifier(ModifierBold) {
			t.Errorf("%s: style = %+v", src, cells[0].Style)
		}
	}
	// Without a resolver an unknown tag is simply ignored, not drawn.
	if got := richText(renderRich(t, "<success>ok</>", nil)); got != "ok" {
		t.Errorf("no resolver: %q", got)
	}
}

func TestRichTextLeavesMalformedTagsAlone(t *testing.T) {
	for _, src := range []string{"a < b", "unclosed <bold", "2 < 3 and 4 > 1"} {
		if got := richText(renderRich(t, src, nil)); got != src {
			t.Errorf("%q was rewritten to %q", src, got)
		}
	}
	// A tag with no content is text too, and so is one that is only spaces.
	for _, src := range []string{"<>", "a <  > b"} {
		if got := richText(renderRich(t, src, nil)); got != src {
			t.Errorf("%q was rewritten to %q", src, got)
		}
	}
	// The escape still works for a genuine tag-shaped literal.
	if got := richText(renderRich(t, `\<bold>`, nil)); got != "<bold>" {
		t.Errorf("escaped tag = %q", got)
	}

	// One close too many must not unwind past the base style.
	cells := renderRich(t, "</></>x", nil)
	if richText(cells) != "x" || cells[0].Style != (Style{}) {
		t.Errorf("extra closing tags: %q %+v", richText(cells), cells[0].Style)
	}
}
