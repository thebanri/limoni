package backend

import (
	"testing"
)

func TestParseEventRunes(t *testing.T) {
	// Standart karakter vuruşu
	ev, consumed := ParseEvent([]byte("A"))
	if consumed != 1 || ev.Type != EventKey || ev.Key.Type != KeyRune || ev.Key.Ch != 'A' {
		t.Errorf("Rune çözme hatası: %v, consumed: %d", ev, consumed)
	}

	// Ctrl-A karakteri (ASCII 1)
	ev, consumed = ParseEvent([]byte{1})
	if consumed != 1 || ev.Type != EventKey || ev.Key.Type != KeyRune || ev.Key.Ch != 'a' || !ev.Key.Ctrl {
		t.Errorf("Ctrl-A çözme hatası: %v, consumed: %d", ev, consumed)
	}

	// Özel tuşlar (Backspace, Tab, Space)
	ev, consumed = ParseEvent([]byte{127}) // Backspace
	if consumed != 1 || ev.Type != EventKey || ev.Key.Type != KeyBackspace {
		t.Errorf("Backspace çözme hatası: %v", ev)
	}

	ev, consumed = ParseEvent([]byte{'\t'}) // Tab
	if consumed != 1 || ev.Type != EventKey || ev.Key.Type != KeyTab {
		t.Errorf("Tab çözme hatası: %v", ev)
	}
}

func TestParseEventArrowsAndModifiers(t *testing.T) {
	// Standart Yukarı Ok: \x1b[A
	ev, consumed := ParseEvent([]byte("\x1b[A"))
	if consumed != 3 || ev.Type != EventKey || ev.Key.Type != KeyArrowUp {
		t.Errorf("ArrowUp çözme hatası: %v, consumed: %d", ev, consumed)
	}

	// Ctrl-Yukarı Ok: \x1b[1;5A
	ev, consumed = ParseEvent([]byte("\x1b[1;5A"))
	if consumed != 6 || ev.Type != EventKey || ev.Key.Type != KeyArrowUp || !ev.Key.Ctrl || ev.Key.Shift {
		t.Errorf("Ctrl-ArrowUp çözme hatası: %v", ev)
	}

	// Shift-Alt-Aşağı Ok: \x1b[1;4B
	ev, consumed = ParseEvent([]byte("\x1b[1;4B"))
	if consumed != 6 || ev.Type != EventKey || ev.Key.Type != KeyArrowDown || !ev.Key.Shift || !ev.Key.Alt || ev.Key.Ctrl {
		t.Errorf("Shift-Alt-ArrowDown çözme hatası: %v", ev)
	}
}

func TestParseEventSpecialKeys(t *testing.T) {
	// Home Tuşu: \x1b[1~ veya \x1b[H
	ev, consumed := ParseEvent([]byte("\x1b[H"))
	if consumed != 3 || ev.Type != EventKey || ev.Key.Type != KeyHome {
		t.Errorf("Home çözme hatası: %v", ev)
	}

	// Delete Tuşu: \x1b[3~
	ev, consumed = ParseEvent([]byte("\x1b[3~"))
	if consumed != 4 || ev.Type != EventKey || ev.Key.Type != KeyDelete {
		t.Errorf("Delete çözme hatası: %v", ev)
	}

	// F5 Tuşu: \x1b[15~
	ev, consumed = ParseEvent([]byte("\x1b[15~"))
	if consumed != 5 || ev.Type != EventKey || ev.Key.Type != KeyF5 {
		t.Errorf("F5 çözme hatası: %v", ev)
	}

	// SS3 F1 Tuşu: \x1bOP
	ev, consumed = ParseEvent([]byte("\x1bOP"))
	if consumed != 3 || ev.Type != EventKey || ev.Key.Type != KeyF1 {
		t.Errorf("F1 çözme hatası: %v", ev)
	}
}

func TestParseEventMouseSGR(t *testing.T) {
	// SGR Sol Tıklama (10, 20 koordinatı, 1-tabanlı girdi: 11, 21): \x1b[<0;11;21M
	ev, consumed := ParseEvent([]byte("\x1b[<0;11;21M"))
	if consumed != 11 || ev.Type != EventMouse {
		t.Errorf("Fare çözme hatası: %v", ev)
	}
	if ev.Mouse.Button != MouseLeft || ev.Mouse.X != 10 || ev.Mouse.Y != 20 || ev.Mouse.Drag {
		t.Errorf("Fare değerleri hatalı: %v", ev.Mouse)
	}

	// SGR Tekerlek Yukarı (Shift basılı): \x1b[<68;5;5M (64 + 4 modifier = 68)
	ev, consumed = ParseEvent([]byte("\x1b[<68;5;5M"))
	if consumed != 10 || ev.Type != EventMouse || ev.Mouse.Button != MouseScrollUp || !ev.Mouse.Shift {
		t.Errorf("Fare tekerlek/Shift çözme hatası: %v", ev.Mouse)
	}

	// SGR Sol Bırakma: \x1b[<0;11;21m
	ev, consumed = ParseEvent([]byte("\x1b[<0;11;21m"))
	if consumed != 11 || ev.Type != EventMouse || ev.Mouse.Button != MouseRelease {
		t.Errorf("Fare bırakma çözme hatası: %v", ev.Mouse)
	}
}

func TestParseEventFocus(t *testing.T) {
	// Focus In: \x1b[I
	ev, consumed := ParseEvent([]byte("\x1b[I"))
	if consumed != 3 || ev.Type != EventFocus || !ev.Focus.Gained {
		t.Errorf("Focus Gained çözme hatası: %v", ev)
	}

	// Focus Out: \x1b[O
	ev, consumed = ParseEvent([]byte("\x1b[O"))
	if consumed != 3 || ev.Type != EventFocus || ev.Focus.Gained {
		t.Errorf("Focus Lost çözme hatası: %v", ev)
	}
}

func TestParseIncompleteEscape(t *testing.T) {
	// Yarım kalan dizi: \x1b[1
	_, consumed := ParseEvent([]byte("\x1b[1"))
	if consumed != 0 {
		t.Errorf("Yarım kalan dizi tüketilmemeliydi, consumed: %d", consumed)
	}
}
