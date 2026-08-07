package terminal

// FocusManager, TUI ekranında çizilen interaktif bileşenlerin odak (focus) durumlarını
// ve Tab / Shift+Tab navigasyon sırasını yönetir.
type FocusManager struct {
	focusedID string
	focusable []string
}

// NewFocusManager, yeni bir FocusManager örneği oluşturur.
func NewFocusManager() *FocusManager {
	return &FocusManager{
		focusable: make([]string, 0, 16),
	}
}

// Register, bu çizim karesinde odaklanabilir bir bileşenin ID'sini kaydeder.
// Eğer henüz odaklanmış bir bileşen yoksa, ilk kaydedilen bileşen otomatik odaklanır.
func (fm *FocusManager) Register(id string) {
	if id == "" {
		return
	}
	// Zaten kayıtlıysa ekleme
	for _, fld := range fm.focusable {
		if fld == id {
			return
		}
	}
	fm.focusable = append(fm.focusable, id)

	// Eğer başlangıçta hiçbir odak seçili değilse, ilk odağı buraya ver
	if fm.focusedID == "" {
		fm.focusedID = id
	}
}

// Focused, aktif olarak odaklanmış olan bileşenin ID'sini döndürür.
func (fm *FocusManager) Focused() string {
	return fm.focusedID
}

// SetFocused, aktif odaklanan bileşen ID'sini manuel olarak ayarlar.
func (fm *FocusManager) SetFocused(id string) {
	fm.focusedID = id
}

// Clear, çizim karesi başında odaklanabilir elemanlar listesini temizler.
func (fm *FocusManager) Clear() {
	fm.focusable = fm.focusable[:0]
}

// Next, odağı listedeki bir sonraki elemana geçirir.
func (fm *FocusManager) Next() {
	if len(fm.focusable) == 0 {
		return
	}
	idx := fm.indexOf(fm.focusedID)
	if idx == -1 {
		fm.focusedID = fm.focusable[0]
		return
	}
	nextIdx := (idx + 1) % len(fm.focusable)
	fm.focusedID = fm.focusable[nextIdx]
}

// Prev, odağı listedeki bir önceki elemana geçirir.
func (fm *FocusManager) Prev() {
	if len(fm.focusable) == 0 {
		return
	}
	idx := fm.indexOf(fm.focusedID)
	if idx == -1 {
		fm.focusedID = fm.focusable[len(fm.focusable)-1]
		return
	}
	prevIdx := (idx - 1 + len(fm.focusable)) % len(fm.focusable)
	fm.focusedID = fm.focusable[prevIdx]
}

func (fm *FocusManager) indexOf(id string) int {
	for i, fld := range fm.focusable {
		if fld == id {
			return i
		}
	}
	return -1
}
