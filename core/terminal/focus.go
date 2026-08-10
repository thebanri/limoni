package terminal

import (
	"github.com/thebanri/limoni/core/cell"
)

type FocusDirection int

const (
	DirUp FocusDirection = iota
	DirDown
	DirLeft
	DirRight
)

// FocusManager, TUI ekranında çizilen interaktif bileşenlerin odak (focus) durumlarını
// ve Tab / Shift+Tab navigasyon sırasını yönetir.
type FocusManager struct {
	focusedID  string
	focusable  []string
	scopes     map[string][]string
	scopeStack []string
	bounds     map[string]cell.Rect
}

// NewFocusManager, yeni bir FocusManager örneği oluşturur.
func NewFocusManager() *FocusManager {
	return &FocusManager{
		focusable: make([]string, 0, 16),
		scopes:    make(map[string][]string),
		bounds:    make(map[string]cell.Rect),
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

// ActiveScopes returns a copy of the current active focus scope stack.
func (fm *FocusManager) ActiveScopes() []string {
	if len(fm.scopeStack) == 0 {
		return nil
	}
	res := make([]string, len(fm.scopeStack))
	copy(res, fm.scopeStack)
	return res
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
	clear(fm.bounds)
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

// RegisterBounds registers the screen area bounds of a focusable widget.
func (fm *FocusManager) RegisterBounds(id string, bounds cell.Rect) {
	if id == "" {
		return
	}
	fm.bounds[id] = bounds
}

// MoveFocus2D shifts focus to the spatially closest widget in the given direction.
func (fm *FocusManager) MoveFocus2D(dir FocusDirection) bool {
	if fm.focusedID == "" {
		items := fm.navigationItems()
		if len(items) > 0 {
			fm.focusedID = items[0]
			return true
		}
		return false
	}

	currentRect, ok := fm.bounds[fm.focusedID]
	if !ok {
		return false
	}

	c1x := float64(currentRect.X) + float64(currentRect.Width)/2
	c1y := float64(currentRect.Y) + float64(currentRect.Height)/2

	var bestID string
	minScore := -1.0

	items := fm.navigationItems()
	for _, id := range items {
		if id == fm.focusedID {
			continue
		}
		r, ok := fm.bounds[id]
		if !ok {
			continue
		}

		c2x := float64(r.X) + float64(r.Width)/2
		c2y := float64(r.Y) + float64(r.Height)/2

		var primaryDist, orthogonalDist float64
		inDirection := false

		switch dir {
		case DirUp:
			primaryDist = c1y - c2y
			orthogonalDist = c1x - c2x
			if primaryDist > 0 {
				inDirection = true
			}
		case DirDown:
			primaryDist = c2y - c1y
			orthogonalDist = c1x - c2x
			if primaryDist > 0 {
				inDirection = true
			}
		case DirLeft:
			primaryDist = c1x - c2x
			orthogonalDist = c1y - c2y
			if primaryDist > 0 {
				inDirection = true
			}
		case DirRight:
			primaryDist = c2x - c1x
			orthogonalDist = c1y - c2y
			if primaryDist > 0 {
				inDirection = true
			}
		}

		if !inDirection {
			continue
		}

		if orthogonalDist < 0 {
			orthogonalDist = -orthogonalDist
		}

		// Spatial scoring formula: S = primary + 2 * orthogonal
		score := primaryDist + 2.0*orthogonalDist
		if minScore < 0 || score < minScore {
			minScore = score
			bestID = id
		}
	}

	if bestID != "" {
		fm.focusedID = bestID
		return true
	}
	return false
}
