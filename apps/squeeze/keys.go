package main

import (
	"fmt"

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

// xtermFinal and xtermTilde are how xterm encodes special keys held with a
// modifier: CSI 1;mod A, CSI 3;mod ~.
var (
	xtermFinal = map[driver.KeyType]byte{
		driver.KeyArrowUp: 'A', driver.KeyArrowDown: 'B', driver.KeyArrowRight: 'C', driver.KeyArrowLeft: 'D',
		driver.KeyHome: 'H', driver.KeyEnd: 'F',
		driver.KeyF1: 'P', driver.KeyF2: 'Q', driver.KeyF3: 'R', driver.KeyF4: 'S',
	}
	xtermTilde = map[driver.KeyType]int{
		driver.KeyInsert: 2, driver.KeyDelete: 3, driver.KeyPageUp: 5, driver.KeyPageDown: 6,
		driver.KeyF5: 15, driver.KeyF6: 17, driver.KeyF7: 18, driver.KeyF8: 19,
		driver.KeyF9: 20, driver.KeyF10: 21, driver.KeyF11: 23, driver.KeyF12: 24,
	}
)

// xtermKey is what a terminal sends for a special key held with Ctrl or
// Shift, which x/vt drops: it sends nothing for Ctrl+Left, Shift+Enter or
// Ctrl+Delete. Empty for keys x/vt encodes itself.
func xtermKey(k driver.KeyEvent) string {
	if !k.Ctrl && !k.Shift {
		return ""
	}
	mod := 1
	if k.Shift {
		mod++
	}
	if k.Alt {
		mod += 2
	}
	if k.Ctrl {
		mod += 4
	}
	switch k.Type {
	case driver.KeyEnter:
		// A new line in the prompt: what Claude Code's /terminal-setup binds
		// Shift+Enter to, and what codex, gemini and opencode read as one.
		return "\x1b\r"
	case driver.KeyBackspace:
		if k.Ctrl {
			return "\x08"
		}
	}
	if c, ok := xtermFinal[k.Type]; ok {
		return fmt.Sprintf("\x1b[1;%d%c", mod, c)
	}
	if n, ok := xtermTilde[k.Type]; ok {
		return fmt.Sprintf("\x1b[%d;%d~", n, mod)
	}
	return ""
}

// isDetach is F12 or Ctrl+], the keys that give the keyboard back to
// squeeze. F12 because ] takes AltGr on many layouts (Turkish, German), where
// Ctrl+] cannot be typed; none of the agents uses F12.
func isDetach(k driver.KeyEvent) bool {
	return k.Type == driver.KeyF12 || (k.Ctrl && k.Ch == ']') || (k.Type == driver.KeyRune && k.Ch == 0x1d)
}
