package terminal

// FocusManager, TUI ekranında çizilen interaktif bileşenlerin odak (focus) durumlarını
// ve Tab / Shift+Tab navigasyon sırasını yönetir.
type FocusManager struct {
	focusedID  string
	focusable  []string
	scopes     map[string][]string
	scopeStack []string
}

// NewFocusManager, yeni bir FocusManager örneği oluşturur.
func NewFocusManager() *FocusManager {
	return &FocusManager{
		focusable: make([]string, 0, 16),
		scopes:    make(map[string][]string),
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
	if len(fm.scopeStack) > 0 {
		scopeID := fm.scopeStack[len(fm.scopeStack)-1]
		members := fm.scopes[scopeID]
		found := false
		for _, member := range members {
			if member == id {
				found = true
				break
			}
		}
		if !found {
			fm.scopes[scopeID] = append(members, id)
		}
	}

	// Eğer başlangıçta hiçbir odak seçili değilse, ilk odağı buraya ver
	if fm.focusedID == "" {
		fm.focusedID = id
	}
}

// BeginScope starts a focus scope. Tab navigation is restricted to widgets registered inside it.
func (fm *FocusManager) BeginScope(id string) {
	if id == "" {
		return
	}
	fm.scopeStack = append(fm.scopeStack, id)
	if _, exists := fm.scopes[id]; !exists {
		fm.scopes[id] = nil
	}
}

// EndScope returns focus navigation to the parent scope.
func (fm *FocusManager) EndScope() {
	if len(fm.scopeStack) > 0 {
		fm.scopeStack = fm.scopeStack[:len(fm.scopeStack)-1]
	}
}

func (fm *FocusManager) ActiveScope() string {
	if len(fm.scopeStack) == 0 {
		return ""
	}
	return fm.scopeStack[len(fm.scopeStack)-1]
}

// Focused, aktif olarak odaklanmış olan widget'ın ID'sini döndürür.
func (fm *FocusManager) Focused() string {
	return fm.focusedID
}

// IsFocused reports whether id currently owns the focus.
func (fm *FocusManager) IsFocused(id string) bool { return id != "" && fm.focusedID == id }

// SetFocused, aktif odaklanan widget ID'sini manuel olarak ayarlar.
func (fm *FocusManager) SetFocused(id string) {
	fm.focusedID = id
}

// Clear, çizim karesi başında odaklanabilir elemanlar listesini temizler.
func (fm *FocusManager) Clear() {
	fm.focusable = fm.focusable[:0]
	for id := range fm.scopes {
		fm.scopes[id] = fm.scopes[id][:0]
	}
	fm.scopeStack = fm.scopeStack[:0]
}

// Next, odağı listedeki bir sonraki elemana geçirir.
func (fm *FocusManager) Next() {
	items := fm.navigationItems()
	if len(items) == 0 {
		return
	}
	idx := indexOfFocus(items, fm.focusedID)
	if idx == -1 {
		fm.focusedID = items[0]
		return
	}
	fm.focusedID = items[(idx+1)%len(items)]
}

// NextExcluding advances focus while skipping IDs with the given prefix.
func (fm *FocusManager) NextExcluding(prefix string) {
	allItems := fm.navigationItems()
	items := make([]string, 0, len(allItems))
	for _, item := range allItems {
		if prefix == "" || len(item) < len(prefix) || item[:len(prefix)] != prefix {
			items = append(items, item)
		}
	}
	if len(items) == 0 {
		return
	}
	idx := indexOfFocus(items, fm.focusedID)
	if idx < 0 {
		fm.focusedID = items[0]
	} else {
		fm.focusedID = items[(idx+1)%len(items)]
	}
}

// Prev, odağı listedeki bir önceki elemana geçirir.
func (fm *FocusManager) Prev() {
	items := fm.navigationItems()
	if len(items) == 0 {
		return
	}
	idx := indexOfFocus(items, fm.focusedID)
	if idx == -1 {
		fm.focusedID = items[len(items)-1]
		return
	}
	fm.focusedID = items[(idx-1+len(items))%len(items)]
}

func (fm *FocusManager) navigationItems() []string {
	if scope := fm.ActiveScope(); scope != "" {
		return fm.scopes[scope]
	}
	return fm.focusable
}

func indexOfFocus(items []string, id string) int {
	for i, fld := range items {
		if fld == id {
			return i
		}
	}
	return -1
}
