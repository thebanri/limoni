package limoni

import (
	"testing"
)

func TestLimoniColorsAndStyles(t *testing.T) {
	// 1. Hex Color parsing
	c1 := Hex("#FF0000")
	r, g, b := c1.RGB()
	if r != 255 || g != 0 || b != 0 {
		t.Fatalf("Hex '#FF0000' RGB mismatch: %d, %d, %d", r, g, b)
	}

	// 2. Short hex parsing
	c2 := Hex("#0F0")
	r, g, b = c2.RGB()
	if r != 0 || g != 255 || b != 0 {
		t.Fatalf("Hex '#0F0' RGB mismatch: %d, %d, %d", r, g, b)
	}

	// 3. Style fluent chaining
	st := NewStyle().
		WithFg(RGB(100, 150, 200)).
		WithBg(ColorBlack).
		Bold().
		Underline()

	if !st.HasModifier(ModifierBold) {
		t.Fatalf("expected bold modifier")
	}
	if !st.HasModifier(ModifierUnderline) {
		t.Fatalf("expected underline modifier")
	}

	// 4. Standalone style helpers
	bSt := Bold()
	if !bSt.HasModifier(ModifierBold) {
		t.Fatalf("expected bold from Bold()")
	}

	fgSt := Fg(ColorRed)
	if fgSt.Fg != ColorRed {
		t.Fatalf("expected red foreground from Fg()")
	}
}

func TestLimoniLayoutSplits(t *testing.T) {
	area := NewRect(0, 0, 100, 50)

	// Horizontal split
	hChunks := SplitHorizontal(area, Percentage(30), Percentage(70))
	if len(hChunks) != 2 {
		t.Fatalf("expected 2 horizontal chunks, got %d", len(hChunks))
	}
	if hChunks[0].Width != 30 || hChunks[1].Width != 70 {
		t.Fatalf("unexpected horizontal widths: %d, %d", hChunks[0].Width, hChunks[1].Width)
	}

	// Vertical split
	vChunks := SplitVertical(area, Fixed(5), Fill())
	if len(vChunks) != 2 {
		t.Fatalf("expected 2 vertical chunks, got %d", len(vChunks))
	}
	if vChunks[0].Height != 5 || vChunks[1].Height != 45 {
		t.Fatalf("unexpected vertical heights: %d, %d", vChunks[0].Height, vChunks[1].Height)
	}
}

func TestLimoniWidgetBuilders(t *testing.T) {
	// Block
	blk := NewBlock().
		WithTitle("Başlık").
		WithTitleAlign(AlignCenter).
		Rounded().
		WithPadding(1, 2, 1, 2)

	if blk.Title != "Başlık" || blk.BorderSymbols != SymbolsRounded {
		t.Fatalf("unexpected block configuration: %+v", blk)
	}

	// Paragraph
	p := NewParagraph("Türkçe metin ve emoji 🚀").WithWrap(true)
	if p.Text != "Türkçe metin ve emoji 🚀" || !p.Wrap {
		t.Fatalf("unexpected paragraph: %+v", p)
	}

	// List
	lst := NewList("Öğe 1", "Öğe 2").WithHighlightSymbol("> ")
	if len(lst.Items) != 2 || lst.HighlightSymbol != "> " {
		t.Fatalf("unexpected list: %+v", lst)
	}

	// Table
	tbl := NewTable().
		WithHeaders("İsim", "Durum").
		WithRow("Giriş", "Aktif")
	if tbl.Header == nil || len(tbl.Rows) != 1 {
		t.Fatalf("unexpected table: %+v", tbl)
	}

	// TextInput
	inp := NewTextInput("username").WithPlaceholder("Kullanıcı adı")
	if inp.ID != "username" || inp.Placeholder != "Kullanıcı adı" {
		t.Fatalf("unexpected text input: %+v", inp)
	}

	// Markdown
	md := NewMarkdown("# Başlık\n- Madde 1")
	if md.Content != "# Başlık\n- Madde 1" {
		t.Fatalf("unexpected markdown: %+v", md)
	}
}

func TestLimoniUnicodeWidths(t *testing.T) {
	// Turkish characters should have width 1
	turkish := "çğışöüÇĞİŞÖÜ"
	if w := StringWidth(turkish); w != 12 {
		t.Fatalf("StringWidth(%q) = %d; want 12", turkish, w)
	}

	// BMP Wide Emojis should have width 2
	emojis := []rune{'✅', '❌', '⚡', '✨', '⭐', '☕', '⏰', '⏳', '♿'}
	for _, em := range emojis {
		if w := RuneWidth(em); w != 2 {
			t.Fatalf("RuneWidth(%c) = %d; want 2", em, w)
		}
	}

	// BMP Symbols with East Asian Width Neutral/Narrow should have width 1
	narrowSymbols := []rune{'✓', '✔', '⚠', '★', '☆', '⚙', '│', '─'}
	for _, sym := range narrowSymbols {
		if w := RuneWidth(sym); w != 1 {
			t.Fatalf("RuneWidth(%c) = %d; want 1", sym, w)
		}
	}

	// SMP Emojis should have width 2
	smpEmojis := []rune{'🚀', '🔥', '🎉', '🔴', '🟢', '🔘', '💡', '📦'}
	for _, em := range smpEmojis {
		if w := RuneWidth(em); w != 2 {
			t.Fatalf("RuneWidth(%c) = %d; want 2", em, w)
		}
	}
}

func TestLimoniComposableLego(t *testing.T) {
	// Test creating composable components directly via root package `limoni`
	label := Label("Hello Lego", Bold())
	padded := PadAll(label, 1)
	bordered := Border(padded, SymbolsRounded, Fg(ColorCyan))

	table := NewTable().WithHeaders("A", "B").WithRow("1", "2")
	adaptedTable := AsComponent(table)

	stack := VStack(
		FixedSize(20, 3, bordered),
		Flex(1, adaptedTable),
		Center(Label("Centered Footer")),
		AlignComponent(Label("Right aligned"), AlignRight, AlignMiddle),
		AlignComponent(Label("HAlignRight test"), HAlignRight, AlignBottom),
	).WithGap(0)

	props := stack.LayoutInfo(NewRect(0, 0, 80, 24))
	if props.MinWidth == 0 {
		t.Fatalf("expected non-zero min width for composable stack")
	}
}
