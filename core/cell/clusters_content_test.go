package cell

import (
	"runtime"
	"testing"
	"unicode/utf8"
)

// The cluster table is what lets one Cell hold "\U0001F469\u200D\U0001F469\u200D\U0001F466" without growing Cell
// past 16 bytes. These are the accessors the diff and the widgets use.
func TestClusterContentAndText(t *testing.T) {
	const family = "\U0001F469\u200D\U0001F469\u200D\U0001F466"

	// A single code point is stored as itself and never touches the table.
	if got := ClusterContent("a", 1); got != 'a' {
		t.Errorf("single code point = %d, want 'a'", got)
	}
	if IsCluster('a') {
		t.Error("'a' should not look like a cluster handle")
	}

	h := ClusterContent(family, 2)
	if !IsCluster(h) {
		t.Fatalf("handle %d is not above RuneClusterBase", h)
	}
	if again := ClusterContent(family, 2); again != h {
		t.Errorf("the same cluster got two handles: %d then %d", h, again)
	}
	if got := ClusterText(h); got != family {
		t.Errorf("ClusterText = %q", got)
	}
	if got := ClusterText('a'); got != "a" {
		t.Errorf("ClusterText of a plain rune = %q", got)
	}
	if got := ClusterText(RuneClusterBase + 1<<20); got != "" {
		t.Errorf("an unknown handle should read back empty, got %q", got)
	}
	if got := clusterWidth(h); got != 2 {
		t.Errorf("clusterWidth = %d, want the width it was interned with", got)
	}
}

// AppendContent is called once per emitted cell by the diff, so both its
// paths matter: ASCII inline, everything else out of line.
func TestAppendContent(t *testing.T) {
	const flag = "\U0001F1F9\U0001F1F7" // two regional indicators
	h := ClusterContent(flag, 2)

	cases := []struct {
		name string
		in   rune
		want string
	}{
		{"ascii", 'x', "x"},
		{"multi-byte rune", 'ş', "ş"},
		{"cluster handle", h, flag},
		{"unknown handle appends nothing", RuneClusterBase + 1<<20, ""},
	}
	for _, c := range cases {
		if got := string(AppendContent(nil, c.in)); got != c.want {
			t.Errorf("%s: AppendContent = %q, want %q", c.name, got, c.want)
		}
	}

	// It appends, it does not replace.
	if got := string(AppendContent([]byte("ab"), 'c')); got != "abc" {
		t.Errorf("appending to existing bytes: %q", got)
	}

	// And it does not allocate beyond growing the caller's buffer, which is
	// why the diff can call it for every cell of a full-screen redraw.
	buf := make([]byte, 0, 64)
	if n := testing.AllocsPerRun(100, func() {
		buf = AppendContent(buf[:0], h)
		runtime.GC()
	}); n != 0 {
		t.Errorf("AppendContent allocated %v times per call", n)
	}
}

func TestNextClusterInBothModes(t *testing.T) {
	const s = "e\u0301x" // e + combining acute, then x
	t.Cleanup(func() { SetGraphemeClusters(true) })

	SetGraphemeClusters(true)
	if !GraphemeClusters() {
		t.Fatal("SetGraphemeClusters(true) did not take")
	}
	cluster, width, rest := NextCluster(s)
	if cluster != "e\u0301" || width != 1 || rest != "x" {
		t.Errorf("clustered: %q %d %q", cluster, width, rest)
	}

	SetGraphemeClusters(false)
	if GraphemeClusters() {
		t.Fatal("SetGraphemeClusters(false) did not take")
	}
	cluster, width, rest = NextCluster(s)
	if cluster != "e" || width != 1 || rest != "\u0301x" {
		t.Errorf("by code point: %q %d %q", cluster, width, rest)
	}
	if c, w, r := NextCluster(""); c != "" || w != 0 || r != "" {
		t.Errorf("empty input: %q %d %q", c, w, r)
	}
}

func TestRuneWidthAgreesWithTheSlowPath(t *testing.T) {
	// The 64 KB table is generated from runeWidthSlow; if they ever disagree
	// the fast path is silently wrong for some part of the BMP.
	for r := rune(0); r <= utf8.MaxRune; r += 97 {
		if !utf8.ValidRune(r) {
			continue
		}
		if fast, slow := RuneWidth(r), runeWidthSlow(r); fast != slow {
			t.Fatalf("RuneWidth(%U) = %d, runeWidthSlow = %d", r, fast, slow)
		}
	}
}

func TestContextHelpers(t *testing.T) {
	ctx := NewContext(NewRect(1, 2, 3, 4), NewStyle().Bold())
	if ctx.Area != NewRect(1, 2, 3, 4) || !ctx.Style.HasModifier(ModifierBold) {
		t.Errorf("NewContext = %+v", ctx)
	}
	if ctx.IsFocused("") || ctx.IsFocused("anything") {
		t.Error("a context with no focused widget focuses nothing")
	}
	ctx.FocusedID = "save"
	if !ctx.IsFocused("save") || ctx.IsFocused("cancel") || ctx.IsFocused("") {
		t.Error("IsFocused compares the id, and an empty id is never focused")
	}
}

func TestCellAndStyleHelpers(t *testing.T) {
	var c Cell
	c.SetRune('ş')
	if c.Rune() != 'ş' {
		t.Errorf("Rune = %q", c.Rune())
	}
	c.Style = NewStyle().Bold()
	c.Reset()
	if c.Content != ' ' || c.Style.Modifier != ModifierReset {
		t.Errorf("Reset left %+v", c)
	}

	s := NewStyle().Dim().Italic().Underline().Blink().Reverse()
	for _, m := range []Modifier{ModifierDim, ModifierItalic, ModifierUnderline, ModifierBlink, ModifierReverse} {
		if !s.HasModifier(m) {
			t.Errorf("modifier %v missing from %v", m, s.Modifier)
		}
	}
	if s.RemoveModifier(ModifierDim).HasModifier(ModifierDim) {
		t.Error("RemoveModifier left the flag set")
	}
	if !s.AddModifier(ModifierBold).HasModifier(ModifierBold) {
		t.Error("AddModifier did not set the flag")
	}
}
