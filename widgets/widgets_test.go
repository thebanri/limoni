package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

func TestParagraphWrapAndDraw(t *testing.T) {
	area := cell.NewRect(0, 0, 10, 5) // Width = 10
	buf := buffer.NewBuffer(area)

	// "Merhaba TUI Dunya" -> "Merhaba TUI" is 11 chars. With wrap, it splits:
	// 1. "Merhaba" (7 chars)
	// 2. "TUI" (3 chars)
	// 3. "Dunya" (5 chars)
	p := Paragraph{
		Text: "Merhaba TUI Dunya",
		Wrap: true,
	}

	ctx := cell.NewContext(area, cell.Style{})
	p.Draw(ctx, buf)

	// Row 0: "Merhaba"
	line0 := ""
	for x := uint16(0); x < 7; x++ {
		line0 += string(buf.Get(x, 0).Content)
	}
	if line0 != "Merhaba" {
		t.Errorf("Satır 0 hatalı kaydırıldı: %q", line0)
	}

	// Row 1: "TUI Dunya"
	line1 := ""
	for x := uint16(0); x < 9; x++ {
		line1 += string(buf.Get(x, 1).Content)
	}
	if line1 != "TUI Dunya" {
		t.Errorf("Satır 1 hatalı kaydırıldı: %q", line1)
	}
}

func TestListScrollingBounds(t *testing.T) {
	state := NewListState()

	// 10 total items, view height = 3.
	// Selected = 0 -> Offset = 0
	state.Select(0)
	state.ScrollTo(3, 10)
	if state.Offset != 0 {
		t.Errorf("Offset 0 olmalıydı, alınan: %d", state.Offset)
	}

	// Selected = 2 -> Offset = 0 (still visible in index 0, 1, 2)
	state.Select(2)
	state.ScrollTo(3, 10)
	if state.Offset != 0 {
		t.Errorf("Offset 0 olmalıydı, alınan: %d", state.Offset)
	}

	// Selected = 3 -> Offset = 1 (scrolled down: visible 1, 2, 3)
	state.Select(3)
	state.ScrollTo(3, 10)
	if state.Offset != 1 {
		t.Errorf("Offset 1 olmalıydı, alınan: %d", state.Offset)
	}

	// Selected = 8 -> Offset = 6 (visible 6, 7, 8)
	state.Select(8)
	state.ScrollTo(3, 10)
	if state.Offset != 6 {
		t.Errorf("Offset 6 olmalıydı, alınan: %d", state.Offset)
	}

	// Selected = 5 -> Offset = 5 (scrolled up: visible 5, 6, 7)
	state.Select(5)
	state.ScrollTo(3, 10)
	if state.Offset != 5 {
		t.Errorf("Offset 5 olmalıydı, alınan: %d", state.Offset)
	}
}

func TestListDraw(t *testing.T) {
	area := cell.NewRect(0, 0, 15, 3)
	buf := buffer.NewBuffer(area)

	items := []string{"Giris", "Ayarlar", "Destek", "Hakkinda"}
	state := NewListState()
	state.Select(1) // "Ayarlar" selected

	list := List{
		Items:           items,
		HighlightSymbol: "> ",
		State:           state,
	}

	ctx := cell.NewContext(area, cell.Style{})
	list.Draw(ctx, buf)

	// Since height = 3, visible items should be: "Giris", "Ayarlar", "Destek".
	// Since Selected = 1, item 1 should have "> " prepended.
	// Row 0: "Giris"
	// Row 1: "> Ayarlar"
	// Row 2: "Destek"
	r0 := readLine(buf, 0, 5)
	r1 := readLine(buf, 1, 9)
	r2 := readLine(buf, 2, 6)

	if r0 != "Giris" {
		t.Errorf("Satır 0 hatalı: %q", r0)
	}
	if r1 != "> Ayarlar" {
		t.Errorf("Satır 1 (seçili öğe) hatalı: %q", r1)
	}
	if r2 != "Destek" {
		t.Errorf("Satır 2 hatalı: %q", r2)
	}
}

func readLine(buf *buffer.Buffer, y uint16, length uint16) string {
	res := ""
	for x := uint16(0); x < length; x++ {
		res += string(buf.Get(x, y).Content)
	}
	return res
}
