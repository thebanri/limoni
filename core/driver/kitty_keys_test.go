package driver

import "testing"

// With the first kitty keyboard level on, keys that legacy encodings cannot
// tell apart arrive as CSI u; these are the forms that level produces.
func TestParseKittyKeyboardKeys(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want KeyEvent
	}{
		// Shift+Enter as the higher protocol levels send it (level 1, which
		// Limoni turns on, leaves Enter in its legacy form).
		{"\x1b[13;2u", KeyEvent{Type: KeyEnter, Shift: true}},
		{"\x1b[27u", KeyEvent{Type: KeyEsc}},
		{"\x1b[105;5u", KeyEvent{Type: KeyRune, Ch: 'i', Ctrl: true}}, // Ctrl+I, not Tab
		{"\x1b[9;5u", KeyEvent{Type: KeyTab, Ctrl: true}},
		{"\x1b[97;3u", KeyEvent{Type: KeyRune, Ch: 'a', Alt: true}},
		// Sub-parameters (shifted key, event type) must not run into the key code.
		{"\x1b[97:65;6u", KeyEvent{Type: KeyRune, Ch: 'a', Shift: true, Ctrl: true}},
		{"\x1b[97;5:1u", KeyEvent{Type: KeyRune, Ch: 'a', Ctrl: true}},
		// Caps Lock (64) in the modifiers does not read as Shift.
		{"\x1b[97;65u", KeyEvent{Type: KeyRune, Ch: 'a'}},
		{"\x1b[57401u", KeyEvent{Type: KeyRune, Ch: '2'}}, // keypad 2
		{"\x1b[57414u", KeyEvent{Type: KeyEnter}},         // keypad Enter
		{"\x1b[1;5P", KeyEvent{Type: KeyF1, Ctrl: true}},
		{"\x1b[1;2Q", KeyEvent{Type: KeyF2, Shift: true}},
		{"\x1b[1;3S", KeyEvent{Type: KeyF4, Alt: true}},
	} {
		ev, n := ParseEvent([]byte(tc.in))
		if n != len(tc.in) || ev.Type != EventKey || ev.Key != tc.want {
			t.Errorf("%q: %+v (consumed %d), want %+v", tc.in, ev.Key, n, tc.want)
		}
	}
	// Caps Lock itself, a media key, F13: consumed, not typed.
	for _, in := range []string{"\x1b[57358u", "\x1b[57428u", "\x1b[57376u"} {
		if ev, n := ParseEvent([]byte(in)); n != len(in) || ev.Type == EventKey {
			t.Errorf("%q produced %+v", in, ev)
		}
	}
}
