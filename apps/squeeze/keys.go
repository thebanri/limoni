package main

import (
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/thebanri/limoni/core/driver"
)

// specialKeys maps Limoni's keys to the emulator's, which encodes them as
// the program asked for (application cursor keys and the rest).
var specialKeys = map[driver.KeyType]rune{
	driver.KeyEsc:        uv.KeyEscape,
	driver.KeyEnter:      uv.KeyEnter,
	driver.KeyBackspace:  uv.KeyBackspace,
	driver.KeyTab:        uv.KeyTab,
	driver.KeySpace:      uv.KeySpace,
	driver.KeyArrowUp:    uv.KeyUp,
	driver.KeyArrowDown:  uv.KeyDown,
	driver.KeyArrowLeft:  uv.KeyLeft,
	driver.KeyArrowRight: uv.KeyRight,
	driver.KeyHome:       uv.KeyHome,
	driver.KeyEnd:        uv.KeyEnd,
	driver.KeyPageUp:     uv.KeyPgUp,
	driver.KeyPageDown:   uv.KeyPgDown,
	driver.KeyInsert:     uv.KeyInsert,
	driver.KeyDelete:     uv.KeyDelete,
	driver.KeyF1:         uv.KeyF1,
	driver.KeyF2:         uv.KeyF2,
	driver.KeyF3:         uv.KeyF3,
	driver.KeyF4:         uv.KeyF4,
	driver.KeyF5:         uv.KeyF5,
	driver.KeyF6:         uv.KeyF6,
	driver.KeyF7:         uv.KeyF7,
	driver.KeyF8:         uv.KeyF8,
	driver.KeyF9:         uv.KeyF9,
	driver.KeyF10:        uv.KeyF10,
	driver.KeyF11:        uv.KeyF11,
	driver.KeyF12:        uv.KeyF12,
}

// toEmulator turns a key the user pressed into the key the agent receives.
func toEmulator(k driver.KeyEvent) (uv.KeyPressEvent, bool) {
	var mod uv.KeyMod
	if k.Ctrl {
		mod |= uv.ModCtrl
	}
	if k.Alt {
		mod |= uv.ModAlt
	}
	if k.Shift {
		mod |= uv.ModShift
	}
	if code, ok := specialKeys[k.Type]; ok {
		return uv.KeyPressEvent{Code: code, Mod: mod}, true
	}
	if k.Type != driver.KeyRune {
		return uv.KeyPressEvent{}, false
	}
	ev := uv.KeyPressEvent{Code: k.Ch, Mod: mod}
	if !k.Ctrl && !k.Alt {
		ev.Text = string(k.Ch)
	}
	return ev, true
}

// isDetach is Ctrl+], the key that gives the keyboard back to squeeze.
func isDetach(k driver.KeyEvent) bool {
	return (k.Ctrl && k.Ch == ']') || (k.Type == driver.KeyRune && k.Ch == 0x1d)
}
