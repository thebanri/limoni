package widgets

import (
	"unicode"

	"github.com/thebanri/limoni/core/driver"
)

// Keybinding, tek bir klavye kısayolunu ve onun tetiklediği eylemi temsil eder.
type Keybinding struct {
	// Key, tuş tipidir (ör: KeyRune, KeyTab, KeyEsc, KeyArrowUp vb.)
	Key driver.KeyType
	// Ch, Key == KeyRune ise basılan karakterdir.
	Ch rune
	// Ctrl, Ctrl modifikasyonunun gerekli olup olmadığını belirtir.
	Ctrl bool
	// Shift, Shift modifikasyonunun gerekli olup olmadığını belirtir.
	Shift bool
	// Handler, bu kısayol eşleştiğinde çalıştırılacak callback fonksiyonudur.
	Handler func()
	// Label, bu kısayolun Command Palette'te gösterilecek açıklamasıdır.
	Label string
	// Category, kısayolun ait olduğu kategoridir (ör: "Navigasyon", "Görünüm").
	Category string
	// Scope, kısayolun geçerli olduğu odak kapsamıdır (ör: "settings_modal"). Boş bırakılırsa global kabul edilir.
	Scope string
	// When, kısayolun o anda etkin olup olmadığını belirler. Nil ise daima etkindir.
	When func() bool
}

// KeybindingManager, bildirimsel (declarative) olarak tanımlanan
// klavye kısayollarını merkezi bir yerde yönetir.
type KeybindingManager struct {
	bindings []Keybinding
}

// NewKeybindingManager, yeni bir KeybindingManager örneği oluşturur.
func NewKeybindingManager() *KeybindingManager {
	return &KeybindingManager{
		bindings: make([]Keybinding, 0, 32),
	}
}

// Register, yeni bir kısayol kaydeder.
func (km *KeybindingManager) Register(kb Keybinding) {
	km.bindings = append(km.bindings, kb)
}

// Handle, gelen tuş olayını aktif odak kapsamları (activeScopes) sırasına göre kontrol eder.
// Kapsamlar en içtekiden (highest priority) en dıştakine doğru taranır. En son global kapsam kontrol edilir.
func (km *KeybindingManager) Handle(ev driver.KeyEvent, activeScopes ...string) bool {
	// Kapsam kontrol sırasını oluştur: en içten en dışa, sonra global ("")
	scopesToCheck := make([]string, 0, len(activeScopes)+1)
	for i := len(activeScopes) - 1; i >= 0; i-- {
		scopesToCheck = append(scopesToCheck, activeScopes[i])
	}
	scopesToCheck = append(scopesToCheck, "") // Global fallback

	for _, targetScope := range scopesToCheck {
		for _, kb := range km.bindings {
			kbScope := kb.Scope
			if kbScope == "global" {
				kbScope = ""
			}
			normTarget := targetScope
			if normTarget == "global" {
				normTarget = ""
			}

			if kbScope != normTarget {
				continue
			}
			if kb.When != nil && !kb.When() {
				continue
			}
			if kb.Key != ev.Type {
				continue
			}
			if kb.Key == driver.KeyRune && kb.Ch != ev.Ch {
				continue
			}
			if kb.Ctrl != ev.Ctrl {
				continue
			}
			if kb.Key != driver.KeyRune && kb.Shift != ev.Shift {
				continue
			}
			if kb.Handler != nil {
				kb.Handler()
			}
			return true
		}
	}
	return false
}

// AllBindings, tüm kayıtlı kısayolları döndürür.
// Command Palette'e otomatik komut kaydı için kullanılır.
func (km *KeybindingManager) AllBindings() []Keybinding {
	return km.bindings
}

// ToCommandItems, tüm kayıtlı kısayolları CommandItem listesine dönüştürür.
// Bu liste Command Palette'e doğrudan verilebilir.
func (km *KeybindingManager) ToCommandItems() []CommandItem {
	items := make([]CommandItem, 0, len(km.bindings))
	for _, kb := range km.bindings {
		if kb.Label == "" {
			continue
		}
		detail := formatKeybinding(kb)
		items = append(items, CommandItem{
			Label:    kb.Label,
			Detail:   detail,
			Category: kb.Category,
			Handler:  kb.Handler,
		})
	}
	return items
}

// formatKeybinding, kısayolun okunabilir metin gösterimini üretir.
func formatKeybinding(kb Keybinding) string {
	s := ""
	if kb.Ctrl {
		s += "Ctrl+"
	}
	if kb.Shift {
		s += "Shift+"
	}

	switch kb.Key {
	case driver.KeyRune:
		s += string(unicode.ToUpper(kb.Ch))
	case driver.KeyTab:
		s += "Tab"
	case driver.KeyEsc:
		s += "Esc"
	case driver.KeyEnter:
		s += "Enter"
	case driver.KeySpace:
		s += "Space"
	case driver.KeyBackspace:
		s += "Backspace"
	case driver.KeyArrowUp:
		s += "↑"
	case driver.KeyArrowDown:
		s += "↓"
	case driver.KeyArrowLeft:
		s += "←"
	case driver.KeyArrowRight:
		s += "→"
	default:
		s += "?"
	}

	return s
}
