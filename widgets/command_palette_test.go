package widgets

import (
	"testing"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// testCommandItems, komut paleti testleri için ortak bir komut seti döndürür.
func testCommandItems() []CommandItem {
	return []CommandItem{
		{Label: "Giriş Sekmesine Git", Detail: "", Category: "Navigasyon",
			Handler: func() {}},
		{Label: "Ayarlar Sekmesine Git", Detail: "", Category: "Navigasyon",
			Handler: func() {}},
		{Label: "Yardım Panelini Aç", Detail: "?", Category: "Görünüm",
			Handler: func() {}},
		{Label: "Hata Ayıklama Modunu Aç/Kapa", Detail: "Ctrl+D", Category: "Görünüm",
			Handler: func() {}},
	}
}

func TestCommandPaletteDrawRuneMarksWideContinuation(t *testing.T) {
	buf := buffer.NewBuffer(cell.NewRect(0, 0, 4, 1))
	width := drawRune(buf, 1, 0, '🔍', cell.Style{})
	if width != 2 {
		t.Fatalf("wide rune width = %d; 2 bekleniyordu", width)
	}
	if buf.Get(1, 0).Content != '🔍' {
		t.Fatalf("wide rune ana hücreye yazılmadı")
	}
	if buf.Get(2, 0).Content != cell.RuneContinuation {
		t.Fatalf("wide rune devam hücresi RuneContinuation olmalı")
	}
}

func TestCommandPalette_DebugArea(t *testing.T) {
	state := NewCommandPaletteState()
	state.AllItems = testCommandItems()
	state.Open()

	area := cell.NewRect(0, 0, 100, 30)
	got := (CommandPalette{State: state}).DebugArea(area)

	if got.X != 20 || got.Y != 2 || got.Width != 60 || got.Height != 8 {
		t.Fatalf("DebugArea = %+v; Rect{20,2,60,8} bekleniyordu", got)
	}
}

func TestCommandPaletteRegistersFocusAndKeepsArrowNavigationInState(t *testing.T) {
	state := NewCommandPaletteState()
	state.AllItems = testCommandItems()
	state.Open()
	registered := ""
	ctx := cell.NewContext(cell.NewRect(0, 0, 100, 30), cell.Style{})
	ctx.RegisterFocus = func(id string) { registered = id }
	(CommandPalette{ID: "command_palette", State: state}).Draw(ctx, buffer.NewBuffer(ctx.Area))
	if registered != "command_palette" {
		t.Fatalf("registered focus = %q, want command_palette", registered)
	}
	if !state.HandleKey(backend.KeyEvent{Type: backend.KeyArrowDown}) || state.Selected != 1 {
		t.Fatalf("arrow down selected = %d, want 1", state.Selected)
	}
	if !state.HandleKey(backend.KeyEvent{Type: backend.KeyArrowUp}) || state.Selected != 0 {
		t.Fatalf("arrow up selected = %d, want 0", state.Selected)
	}
}

func TestCommandPalettePositionBottom(t *testing.T) {
	state := NewCommandPaletteState()
	state.AllItems = testCommandItems()
	state.Open()
	position := &CommandPalettePosition{Bottom: 1}
	got := (CommandPalette{State: state, Position: position}).DebugArea(cell.NewRect(0, 0, 100, 30))
	if got.X != 20 || got.Y != 21 || got.Width != 60 || got.Height != 8 {
		t.Fatalf("bottom position = %+v; want Rect{20,21,60,8}", got)
	}
}

func TestCommandPaletteState_New(t *testing.T) {
	state := NewCommandPaletteState()
	if state.IsOpen {
		t.Fatal("yeni palet kapalı olmalı")
	}
	if state.MaxVisible != 10 {
		t.Fatalf("MaxVisible = %d; 10 bekleniyordu", state.MaxVisible)
	}
	if state.Selected != 0 {
		t.Fatalf("Selected = %d; 0 bekleniyordu", state.Selected)
	}
	if state.ScrollOffset != 0 {
		t.Fatalf("ScrollOffset = %d; 0 bekleniyordu", state.ScrollOffset)
	}
	if state.Query.Value() != "" {
		t.Fatalf("Query = %q; boş bekleniyordu", state.Query.Value())
	}
}

func TestCommandPaletteState_OpenResetsState(t *testing.T) {
	state := NewCommandPaletteState()
	state.AllItems = testCommandItems()

	// Önce kirli durum oluştur
	state.Query.SetValue("yardım")
	state.Selected = 3
	state.ScrollOffset = 2

	state.Open()
	if !state.IsOpen {
		t.Fatal("Open sonrası palet açık olmalı")
	}
	if state.Query.Value() != "" {
		t.Fatalf("Open sonrası Query = %q; boş bekleniyordu", state.Query.Value())
	}
	if state.Selected != 0 {
		t.Fatalf("Open sonrası Selected = %d; 0 bekleniyordu", state.Selected)
	}
	if state.ScrollOffset != 0 {
		t.Fatalf("Open sonrası ScrollOffset = %d; 0 bekleniyordu", state.ScrollOffset)
	}
	// Boş sorgu tüm öğeleri getirmeli
	if len(state.Filtered) != len(state.AllItems) {
		t.Fatalf("Filtered uzunluğu = %d; %d bekleniyordu", len(state.Filtered), len(state.AllItems))
	}
}

func TestCommandPaletteState_CloseAndToggle(t *testing.T) {
	state := NewCommandPaletteState()
	state.AllItems = testCommandItems()

	state.Toggle()
	if !state.IsOpen {
		t.Fatal("Toggle açık duruma geçirmeli")
	}
	state.Toggle()
	if state.IsOpen {
		t.Fatal("Toggle kapalı duruma geçirmeli")
	}
	state.Close()
	if state.IsOpen {
		t.Fatal("Close sonrası palet kapalı olmalı")
	}
}

func TestCommandPaletteState_HandleKey_Closed(t *testing.T) {
	state := NewCommandPaletteState()
	state.AllItems = testCommandItems()

	// Kapalıyken hiçbir tuş tüketilmemeli
	if state.HandleKey(backend.KeyEvent{Type: backend.KeyEsc}) {
		t.Fatal("kapalı palet Esc'i tüketmemeli")
	}
	if state.HandleKey(backend.KeyEvent{Type: backend.KeyEnter}) {
		t.Fatal("kapalı palet Enter'ı tüketmemeli")
	}
}

func TestCommandPaletteState_HandleKey_CtrlPTogglesClosed(t *testing.T) {
	state := NewCommandPaletteState()
	state.AllItems = testCommandItems()
	state.Open()

	if !state.HandleKey(backend.KeyEvent{Type: backend.KeyRune, Ch: 'p', Ctrl: true}) {
		t.Fatal("açık palet Ctrl+P olayını tüketmeli")
	}
	if state.IsOpen {
		t.Fatal("açık palet Ctrl+P ile kapanmalı")
	}
}

func TestCommandPaletteState_HandleKey_Esc(t *testing.T) {
	state := NewCommandPaletteState()
	state.AllItems = testCommandItems()
	state.Open()

	if !state.HandleKey(backend.KeyEvent{Type: backend.KeyEsc}) {
		t.Fatal("açık palet Esc'i tüketmeli")
	}
	if state.IsOpen {
		t.Fatal("Esc sonrası palet kapanmalı")
	}
}

func TestCommandPaletteState_HandleKey_EnterRunsHandler(t *testing.T) {
	state := NewCommandPaletteState()
	state.AllItems = testCommandItems()
	state.Open()

	ran := false
	state.Filtered[1].Handler = func() { ran = true }

	// İkinci öğeyi seç
	state.HandleKey(backend.KeyEvent{Type: backend.KeyArrowDown})
	if state.Selected != 1 {
		t.Fatalf("Selected = %d; 1 bekleniyordu", state.Selected)
	}

	if !state.HandleKey(backend.KeyEvent{Type: backend.KeyEnter}) {
		t.Fatal("Enter tüketilmeli")
	}
	if !ran {
		t.Fatal("seçili öğenin handler'ı çalışmalı")
	}
	if state.IsOpen {
		t.Fatal("Enter sonrası palet kapanmalı")
	}
}

func TestCommandPaletteState_HandleKey_EnterNoSelection(t *testing.T) {
	state := NewCommandPaletteState()
	state.AllItems = []CommandItem{} // Boş liste
	state.Open()

	if !state.HandleKey(backend.KeyEvent{Type: backend.KeyEnter}) {
		t.Fatal("Enter tüketilmeli")
	}
	// Panik olmamalı, palet kapanmalı
	if state.IsOpen {
		t.Fatal("Enter sonrası palet kapanmalı")
	}
}

func TestCommandPaletteState_HandleKey_Navigation(t *testing.T) {
	state := NewCommandPaletteState()
	state.AllItems = testCommandItems()
	state.Open()

	// Yukarı: sınırda kalmalı
	state.HandleKey(backend.KeyEvent{Type: backend.KeyArrowUp})
	if state.Selected != 0 {
		t.Fatalf("üst sınırda Selected = %d; 0 bekleniyordu", state.Selected)
	}

	// Aşağı
	state.HandleKey(backend.KeyEvent{Type: backend.KeyArrowDown})
	if state.Selected != 1 {
		t.Fatalf("Selected = %d; 1 bekleniyordu", state.Selected)
	}

	// Son öğeye git
	for i := 0; i < 10; i++ {
		state.HandleKey(backend.KeyEvent{Type: backend.KeyArrowDown})
	}
	if state.Selected != len(state.Filtered)-1 {
		t.Fatalf("alt sınırda Selected = %d; %d bekleniyordu", state.Selected, len(state.Filtered)-1)
	}

	// Alt sınırda daha aşağı inmemeli
	state.HandleKey(backend.KeyEvent{Type: backend.KeyArrowDown})
	if state.Selected != len(state.Filtered)-1 {
		t.Fatalf("alt sınır aşıldı: Selected = %d", state.Selected)
	}
}

func TestCommandPaletteState_HandleKey_ScrollOffset(t *testing.T) {
	state := NewCommandPaletteState()
	state.MaxVisible = 2
	// 5 öğe oluştur
	items := make([]CommandItem, 5)
	for i := range items {
		items[i] = CommandItem{Label: "Komut", Category: "A"}
	}
	state.AllItems = items
	state.Open()

	// 3 kez aşağı in: Selected=3, ScrollOffset 2 olmalı (3 - 2 + 1)
	for i := 0; i < 3; i++ {
		state.HandleKey(backend.KeyEvent{Type: backend.KeyArrowDown})
	}
	if state.Selected != 3 {
		t.Fatalf("Selected = %d; 3 bekleniyordu", state.Selected)
	}
	if state.ScrollOffset != 2 {
		t.Fatalf("ScrollOffset = %d; 2 bekleniyordu", state.ScrollOffset)
	}

	// Yukarı çıkınca offset geri gelmeli
	state.HandleKey(backend.KeyEvent{Type: backend.KeyArrowUp})
	if state.Selected != 2 {
		t.Fatalf("Selected = %d; 2 bekleniyordu", state.Selected)
	}
	// Selected=2, MaxVisible=2: görünür pencere [2,3] -> offset 2 kalmalı
	if state.ScrollOffset != 2 {
		t.Fatalf("ScrollOffset = %d; 2 bekleniyordu", state.ScrollOffset)
	}
	state.HandleKey(backend.KeyEvent{Type: backend.KeyArrowUp})
	// Selected=1: ScrollOffset Selected'a iner -> 1
	if state.ScrollOffset != 1 {
		t.Fatalf("ScrollOffset = %d; 1 bekleniyordu", state.ScrollOffset)
	}
	state.HandleKey(backend.KeyEvent{Type: backend.KeyArrowUp})
	// Selected=0: ScrollOffset 0'a iner
	if state.ScrollOffset != 0 {
		t.Fatalf("ScrollOffset = %d; 0 bekleniyordu", state.ScrollOffset)
	}
}

func TestCommandPaletteState_HandleKey_FiltersOnTyping(t *testing.T) {
	state := NewCommandPaletteState()
	state.AllItems = testCommandItems()
	state.Open()

	// "yard" yazınca sadece Yardım eşleşmeli
	for _, r := range "yard" {
		state.HandleKey(backend.KeyEvent{Type: backend.KeyRune, Ch: r})
	}
	if len(state.Filtered) != 1 {
		t.Fatalf("Filtered uzunluğu = %d; 1 bekleniyordu", len(state.Filtered))
	}
	if state.Filtered[0].Label != "Yardım Panelini Aç" {
		t.Fatalf("Filtered[0] = %q; 'Yardım Panelini Aç' bekleniyordu", state.Filtered[0].Label)
	}
	if state.Selected != 0 {
		t.Fatalf("filtre sonrası Selected = %d; 0 bekleniyordu", state.Selected)
	}
}

func TestCommandPaletteState_HandleKey_BackspaceRefilters(t *testing.T) {
	state := NewCommandPaletteState()
	state.AllItems = testCommandItems()
	state.Open()

	for _, r := range "yard" {
		state.HandleKey(backend.KeyEvent{Type: backend.KeyRune, Ch: r})
	}
	if len(state.Filtered) != 1 {
		t.Fatalf("filtreleme öncesi Filtered = %d; 1 bekleniyordu", len(state.Filtered))
	}

	// Backspace ile "yar" kalmalı -> "Yardım Panelini Aç" ve "Ayarlar Sekmesine Git" eşleşir
	state.HandleKey(backend.KeyEvent{Type: backend.KeyBackspace})
	if len(state.Filtered) != 2 {
		t.Fatalf("backspace sonrası Filtered = %d; 2 bekleniyordu", len(state.Filtered))
	}

	// Tümünü sil -> tüm öğeler geri gelmeli
	for state.Query.Value() != "" {
		state.HandleKey(backend.KeyEvent{Type: backend.KeyBackspace})
	}
	if len(state.Filtered) != len(state.AllItems) {
		t.Fatalf("boş sorgu sonrası Filtered = %d; %d bekleniyordu", len(state.Filtered), len(state.AllItems))
	}
}

func TestCommandPaletteState_HandleKey_ReturnsTrueWhenOpen(t *testing.T) {
	state := NewCommandPaletteState()
	state.AllItems = testCommandItems()
	state.Open()

	// Açıkken herhangi bir tuş tüketilmeli (yazma dahil)
	if !state.HandleKey(backend.KeyEvent{Type: backend.KeyRune, Ch: 'x'}) {
		t.Fatal("açık palet tuş girişini tüketmeli")
	}
}

func TestComputeMatchPositions(t *testing.T) {
	positions := computeMatchPositions("gsg", "grafik sekmesine git")
	// g-s-g sırasıyla eşleşmeli
	if len(positions) != 3 {
		t.Fatalf("positions uzunluğu = %d; 3 bekleniyordu", len(positions))
	}
	// 'g' (0), 's' (7), 'g' (17) konumları
	for _, p := range []int{0, 7, 17} {
		if !positions[p] {
			t.Fatalf("konum %d eşleşme olarak işaretlenmeli", p)
		}
	}
}

func TestComputeMatchPositions_EmptyQuery(t *testing.T) {
	positions := computeMatchPositions("", "herhangi bir metin")
	if len(positions) != 0 {
		t.Fatalf("boş sorgu için positions uzunluğu = %d; 0 bekleniyordu", len(positions))
	}
}
