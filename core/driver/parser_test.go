package driver

import "testing"

// UTF-8 keystrokes and single-byte ASCII control codes.
func TestParseUTF8AndAsciiControlKeys(t *testing.T) {
	cases := []struct {
		name         string
		in           []byte
		wantEvent    Event
		wantConsumed int
	}{
		{
			name:         "empty buffer",
			in:           []byte{},
			wantEvent:    Event{},
			wantConsumed: 0,
		},
		{
			name:         "standard ASCII rune",
			in:           []byte("a"),
			wantEvent:    Event{Type: EventKey, Key: KeyEvent{Type: KeyRune, Ch: 'a'}},
			wantConsumed: 1,
		},
		{
			name:         "multi-byte UTF-8 rune (2 bytes)",
			in:           []byte("ç"),
			wantEvent:    Event{Type: EventKey, Key: KeyEvent{Type: KeyRune, Ch: 'ç'}},
			wantConsumed: 2,
		},
		{
			name:         "multi-byte UTF-8 emoji (4 bytes)",
			in:           []byte("🚀"),
			wantEvent:    Event{Type: EventKey, Key: KeyEvent{Type: KeyRune, Ch: '🚀'}},
			wantConsumed: 4,
		},
		{
			name:         "incomplete UTF-8 sequence waits for more data",
			in:           []byte{0xC3},
			wantEvent:    Event{},
			wantConsumed: 0,
		},
		{
			name:         "invalid UTF-8 byte skips one byte",
			in:           []byte{0xFF},
			wantEvent:    Event{},
			wantConsumed: 1,
		},
		{
			name:         "CR carriage return",
			in:           []byte("\r"),
			wantEvent:    Event{Type: EventKey, Key: KeyEvent{Type: KeyEnter}},
			wantConsumed: 1,
		},
		{
			name:         "LF newline maps to Ctrl+J / Enter",
			in:           []byte("\n"),
			wantEvent:    Event{Type: EventKey, Key: KeyEvent{Type: KeyEnter, Ctrl: true}},
			wantConsumed: 1,
		},
		{
			name:         "ASCII 127 backspace",
			in:           []byte{127},
			wantEvent:    Event{Type: EventKey, Key: KeyEvent{Type: KeyBackspace}},
			wantConsumed: 1,
		},
		{
			name:         "ASCII 8 backspace",
			in:           []byte{'\b'},
			wantEvent:    Event{Type: EventKey, Key: KeyEvent{Type: KeyBackspace}},
			wantConsumed: 1,
		},
		{
			name:         "tab",
			in:           []byte("\t"),
			wantEvent:    Event{Type: EventKey, Key: KeyEvent{Type: KeyTab}},
			wantConsumed: 1,
		},
		{
			name:         "space",
			in:           []byte(" "),
			wantEvent:    Event{Type: EventKey, Key: KeyEvent{Type: KeySpace}},
			wantConsumed: 1,
		},
		{
			name:         "Ctrl-A (ASCII 1)",
			in:           []byte{1},
			wantEvent:    Event{Type: EventKey, Key: KeyEvent{Type: KeyRune, Ch: 'a', Ctrl: true}},
			wantConsumed: 1,
		},
		{
			name:         "Ctrl-Z (ASCII 26)",
			in:           []byte{26},
			wantEvent:    Event{Type: EventKey, Key: KeyEvent{Type: KeyRune, Ch: 'z', Ctrl: true}},
			wantConsumed: 1,
		},
	}

	for _, c := range cases {
		ev, consumed := ParseEvent(c.in)
		if consumed != c.wantConsumed || ev != c.wantEvent {
			t.Errorf("%s: got ev=%+v consumed=%d, want ev=%+v consumed=%d",
				c.name, ev, consumed, c.wantEvent, c.wantConsumed)
		}
	}
}

// Arrow keys with the full standard modifier matrix (Shift, Alt, Ctrl).
func TestParseArrowKeysAndModifiers(t *testing.T) {
	cases := []struct {
		name      string
		in        string
		wantKey   KeyType
		wantShift bool
		wantAlt   bool
		wantCtrl  bool
	}{
		{"Up plain", "\x1b[A", KeyArrowUp, false, false, false},
		{"Down plain", "\x1b[B", KeyArrowDown, false, false, false},
		{"Right plain", "\x1b[C", KeyArrowRight, false, false, false},
		{"Left plain", "\x1b[D", KeyArrowLeft, false, false, false},

		{"Up with param 1 (no mod)", "\x1b[1;1A", KeyArrowUp, false, false, false},
		{"Up with Shift (mod 2)", "\x1b[1;2A", KeyArrowUp, true, false, false},
		{"Up with Alt (mod 3)", "\x1b[1;3A", KeyArrowUp, false, true, false},
		{"Up with Shift+Alt (mod 4)", "\x1b[1;4A", KeyArrowUp, true, true, false},
		{"Up with Ctrl (mod 5)", "\x1b[1;5A", KeyArrowUp, false, false, true},
		{"Up with Shift+Ctrl (mod 6)", "\x1b[1;6A", KeyArrowUp, true, false, true},
		{"Up with Alt+Ctrl (mod 7)", "\x1b[1;7A", KeyArrowUp, false, true, true},
		{"Up with Shift+Alt+Ctrl (mod 8)", "\x1b[1;8A", KeyArrowUp, true, true, true},

		{"Down with Ctrl", "\x1b[1;5B", KeyArrowDown, false, false, true},
		{"Right with Alt", "\x1b[1;3C", KeyArrowRight, false, true, false},
		{"Left with Shift+Ctrl", "\x1b[1;6D", KeyArrowLeft, true, false, true},

		{"Up with single modifier param", "\x1b[5A", KeyArrowUp, false, false, true},
	}

	for _, c := range cases {
		ev, consumed := ParseEvent([]byte(c.in))
		if consumed != len(c.in) {
			t.Errorf("%s: consumed %d, want %d", c.name, consumed, len(c.in))
		}
		if ev.Type != EventKey || ev.Key.Type != c.wantKey {
			t.Errorf("%s: key type = %v, want %v", c.name, ev.Key.Type, c.wantKey)
		}
		if ev.Key.Shift != c.wantShift || ev.Key.Alt != c.wantAlt || ev.Key.Ctrl != c.wantCtrl {
			t.Errorf("%s: modifiers = (shift:%v, alt:%v, ctrl:%v), want (%v, %v, %v)",
				c.name, ev.Key.Shift, ev.Key.Alt, ev.Key.Ctrl, c.wantShift, c.wantAlt, c.wantCtrl)
		}
	}
}

// Navigation, editing, and keypad keys (Home, End, Insert, Delete, PageUp, PageDown).
func TestParseNavigationAndEditingKeys(t *testing.T) {
	cases := []struct {
		name      string
		in        string
		wantKey   KeyType
		wantShift bool
		wantAlt   bool
		wantCtrl  bool
	}{
		{"Home CSI H", "\x1b[H", KeyHome, false, false, false},
		{"Home CSI 1~", "\x1b[1~", KeyHome, false, false, false},
		{"Home CSI 7~", "\x1b[7~", KeyHome, false, false, false},
		{"Home with Ctrl", "\x1b[1;5H", KeyHome, false, false, true},

		{"End CSI F", "\x1b[F", KeyEnd, false, false, false},
		{"End CSI 4~", "\x1b[4~", KeyEnd, false, false, false},
		{"End CSI 8~", "\x1b[8~", KeyEnd, false, false, false},
		{"End with Shift", "\x1b[1;2F", KeyEnd, true, false, false},

		{"Insert CSI 2~", "\x1b[2~", KeyInsert, false, false, false},
		{"Insert with Ctrl", "\x1b[2;5~", KeyInsert, false, false, true},

		{"Delete CSI 3~", "\x1b[3~", KeyDelete, false, false, false},
		{"Delete with Ctrl", "\x1b[3;5~", KeyDelete, false, false, true},
		{"Delete with Alt", "\x1b[3;3~", KeyDelete, false, true, false},

		{"PageUp CSI 5~", "\x1b[5~", KeyPageUp, false, false, false},
		{"PageUp with Shift", "\x1b[5;2~", KeyPageUp, true, false, false},

		{"PageDown CSI 6~", "\x1b[6~", KeyPageDown, false, false, false},
		{"PageDown with Alt", "\x1b[6;3~", KeyPageDown, false, true, false},

		{"Shift+Tab backtab", "\x1b[Z", KeyTab, true, false, false},
	}

	for _, c := range cases {
		ev, consumed := ParseEvent([]byte(c.in))
		if consumed != len(c.in) {
			t.Errorf("%s: consumed %d, want %d", c.name, consumed, len(c.in))
		}
		if ev.Type != EventKey || ev.Key.Type != c.wantKey {
			t.Errorf("%s: key type = %v, want %v", c.name, ev.Key.Type, c.wantKey)
		}
		if ev.Key.Shift != c.wantShift || ev.Key.Alt != c.wantAlt || ev.Key.Ctrl != c.wantCtrl {
			t.Errorf("%s: modifiers = (shift:%v, alt:%v, ctrl:%v), want (%v, %v, %v)",
				c.name, ev.Key.Shift, ev.Key.Alt, ev.Key.Ctrl, c.wantShift, c.wantAlt, c.wantCtrl)
		}
	}
}

// Function keys F1–F12 via SS3 (\x1bO...) and CSI tilde (\x1b[...~).
func TestParseFunctionKeys(t *testing.T) {
	cases := []struct {
		name      string
		in        string
		wantKey   KeyType
		wantShift bool
		wantAlt   bool
		wantCtrl  bool
	}{
		{"F1 SS3", "\x1bOP", KeyF1, false, false, false},
		{"F2 SS3", "\x1bOQ", KeyF2, false, false, false},
		{"F3 SS3", "\x1bOR", KeyF3, false, false, false},
		{"F4 SS3", "\x1bOS", KeyF4, false, false, false},

		{"F1 tilde", "\x1b[11~", KeyF1, false, false, false},
		{"F2 tilde", "\x1b[12~", KeyF2, false, false, false},
		{"F3 tilde", "\x1b[13~", KeyF3, false, false, false},
		{"F4 tilde", "\x1b[14~", KeyF4, false, false, false},
		{"F5 tilde", "\x1b[15~", KeyF5, false, false, false},
		{"F6 tilde", "\x1b[17~", KeyF6, false, false, false},
		{"F7 tilde", "\x1b[18~", KeyF7, false, false, false},
		{"F8 tilde", "\x1b[19~", KeyF8, false, false, false},
		{"F9 tilde", "\x1b[20~", KeyF9, false, false, false},
		{"F10 tilde", "\x1b[21~", KeyF10, false, false, false},
		{"F11 tilde", "\x1b[23~", KeyF11, false, false, false},
		{"F12 tilde", "\x1b[24~", KeyF12, false, false, false},

		{"F5 with Ctrl", "\x1b[15;5~", KeyF5, false, false, true},
		{"F12 with Shift+Alt", "\x1b[24;4~", KeyF12, true, true, false},
	}

	for _, c := range cases {
		ev, consumed := ParseEvent([]byte(c.in))
		if consumed != len(c.in) {
			t.Errorf("%s: consumed %d, want %d", c.name, consumed, len(c.in))
		}
		if ev.Type != EventKey || ev.Key.Type != c.wantKey {
			t.Errorf("%s: key type = %v, want %v", c.name, ev.Key.Type, c.wantKey)
		}
		if ev.Key.Shift != c.wantShift || ev.Key.Alt != c.wantAlt || ev.Key.Ctrl != c.wantCtrl {
			t.Errorf("%s: modifiers = (shift:%v, alt:%v, ctrl:%v), want (%v, %v, %v)",
				c.name, ev.Key.Shift, ev.Key.Alt, ev.Key.Ctrl, c.wantShift, c.wantAlt, c.wantCtrl)
		}
	}
}

// Alt + key combinations via ESC prefix (\x1b + rune).
func TestParseAltKeyCombinations(t *testing.T) {
	cases := []struct {
		name         string
		in           []byte
		wantEvent    Event
		wantConsumed int
	}{
		{
			name:         "Alt+a",
			in:           []byte("\x1ba"),
			wantEvent:    Event{Type: EventKey, Key: KeyEvent{Type: KeyRune, Ch: 'a', Alt: true}},
			wantConsumed: 2,
		},
		{
			name:         "Alt+Z",
			in:           []byte("\x1bZ"),
			wantEvent:    Event{Type: EventKey, Key: KeyEvent{Type: KeyRune, Ch: 'Z', Alt: true}},
			wantConsumed: 2,
		},
		{
			name:         "Alt+Enter (CR)",
			in:           []byte("\x1b\r"),
			wantEvent:    Event{Type: EventKey, Key: KeyEvent{Type: KeyEnter, Alt: true}},
			wantConsumed: 2,
		},
		{
			name:         "Alt+Enter (LF)",
			in:           []byte("\x1b\n"),
			wantEvent:    Event{Type: EventKey, Key: KeyEvent{Type: KeyEnter, Alt: true}},
			wantConsumed: 2,
		},
		{
			name:         "Alt+multi-byte UTF-8",
			in:           []byte("\x1bç"),
			wantEvent:    Event{Type: EventKey, Key: KeyEvent{Type: KeyRune, Ch: 'ç', Alt: true}},
			wantConsumed: 3,
		},
		{
			name:         "Alt+incomplete UTF-8 waits for continuation",
			in:           []byte{0x1b, 0xC3},
			wantEvent:    Event{},
			wantConsumed: 0,
		},
		{
			name:         "Alt+invalid UTF-8 skips sequence",
			in:           []byte{0x1b, 0xFF},
			wantEvent:    Event{},
			wantConsumed: 2,
		},
	}

	for _, c := range cases {
		ev, consumed := ParseEvent(c.in)
		if consumed != c.wantConsumed || ev != c.wantEvent {
			t.Errorf("%s: got ev=%+v consumed=%d, want ev=%+v consumed=%d",
				c.name, ev, consumed, c.wantEvent, c.wantConsumed)
		}
	}
}

// Focus events and terminal response reports.
func TestParseFocusAndCursorReplies(t *testing.T) {
	// Focus gained
	ev, consumed := ParseEvent([]byte("\x1b[I"))
	if consumed != 3 || ev.Type != EventFocus || !ev.Focus.Gained {
		t.Errorf("Focus gained: %+v (consumed %d)", ev, consumed)
	}

	// Focus lost
	ev, consumed = ParseEvent([]byte("\x1b[O"))
	if consumed != 3 || ev.Type != EventFocus || ev.Focus.Gained {
		t.Errorf("Focus lost: %+v (consumed %d)", ev, consumed)
	}

	// Cursor position report (row 24, col 80)
	ev, consumed = ParseEvent([]byte("\x1b[24;80R"))
	if consumed != 8 || ev.Type != EventReply || ev.Reply.Kind != ReplyCursor || ev.Reply.Row != 24 || ev.Reply.Col != 80 {
		t.Errorf("Cursor reply: %+v (consumed %d)", ev, consumed)
	}

	// DA1 reply: CSI ? 62 ; 1 ; 2 c
	ev, consumed = ParseEvent([]byte("\x1b[?62;1;2c"))
	expectedAttrs := uint64((1 << 1) | (1 << 2))
	if consumed != 10 || ev.Type != EventReply || ev.Reply.Kind != ReplyPrimaryDA || ev.Reply.Attributes != expectedAttrs {
		t.Errorf("DA1 reply: %+v (consumed %d)", ev, consumed)
	}

	// DECRPM mode report: CSI ? 2026 ; 1 $ y
	ev, consumed = ParseEvent([]byte("\x1b[?2026;1$y"))
	if consumed != 11 || ev.Type != EventReply || ev.Reply.Kind != ReplyMode || ev.Reply.Mode != 2026 || ev.Reply.Setting != 1 {
		t.Errorf("DECRPM reply: %+v (consumed %d)", ev, consumed)
	}

	// Kitty keyboard flags reply: CSI ? 1 u
	ev, consumed = ParseEvent([]byte("\x1b[?1u"))
	if consumed != 5 || ev.Type != EventReply || ev.Reply.Kind != ReplyKittyKeyboard || ev.Reply.Flags != 1 {
		t.Errorf("Kitty keyboard reply: %+v (consumed %d)", ev, consumed)
	}

	// DA2 / secondary device attribute query response (CSI > 1 ; 2 c) is consumed silently
	ev, consumed = ParseEvent([]byte("\x1b[>1;2c"))
	if consumed != 7 || ev.Type != EventNone {
		t.Errorf("DA2 consumed silently: %+v (consumed %d)", ev, consumed)
	}
}

// Incomplete, lone, and malformed byte sequences.
func TestParsePartialAndMalformedSequences(t *testing.T) {
	// Lone ESC waits for esc timeout
	ev, consumed := ParseEvent([]byte{0x1b})
	if consumed != 0 || ev.Type != EventNone {
		t.Errorf("lone ESC: %+v consumed=%d", ev, consumed)
	}

	// Incomplete CSI waits for more bytes
	ev, consumed = ParseEvent([]byte("\x1b["))
	if consumed != 0 || ev.Type != EventNone {
		t.Errorf("incomplete CSI: %+v consumed=%d", ev, consumed)
	}

	// Incomplete CSI parameters wait for final byte
	ev, consumed = ParseEvent([]byte("\x1b[1;5"))
	if consumed != 0 || ev.Type != EventNone {
		t.Errorf("incomplete CSI params: %+v consumed=%d", ev, consumed)
	}

	// Incomplete SS3 waits for key byte
	ev, consumed = ParseEvent([]byte("\x1bO"))
	if consumed != 0 || ev.Type != EventNone {
		t.Errorf("incomplete SS3: %+v consumed=%d", ev, consumed)
	}

	// Unknown SS3 sequence consumes the 3 bytes without emitting an event
	ev, consumed = ParseEvent([]byte("\x1bOX"))
	if consumed != 3 || ev.Type != EventNone {
		t.Errorf("unknown SS3: %+v consumed=%d", ev, consumed)
	}

	// Unknown CSI command consumes the sequence without producing a bogus key
	ev, consumed = ParseEvent([]byte("\x1b[999z"))
	if consumed != 6 || ev.Type != EventNone {
		t.Errorf("unknown CSI: %+v consumed=%d", ev, consumed)
	}

	// Unknown tilde code consumes the sequence
	ev, consumed = ParseEvent([]byte("\x1b[99~"))
	if consumed != 5 || ev.Type != EventNone {
		t.Errorf("unknown tilde: %+v consumed=%d", ev, consumed)
	}

	// Overlong malformed CSI without final byte gives up 1 byte to prevent stall
	longBuf := make([]byte, 130)
	copy(longBuf, []byte("\x1b[1234567890"))
	ev, consumed = ParseEvent(longBuf)
	if consumed != 1 || ev.Type != EventNone {
		t.Errorf("overlong CSI without final byte: %+v consumed=%d", ev, consumed)
	}
}

// String sequences (APC, OSC, DCS, PM).
func TestParseStringSequences(t *testing.T) {
	// XTVERSION DCS reply terminated by ST (\x1b\)
	ev, consumed := ParseEvent([]byte("\x1bP>|Limoni 1.0\x1b\\"))
	if consumed != 16 || ev.Type != EventReply || ev.Reply.Kind != ReplyVersion || ev.Reply.Version != "Limoni 1.0" {
		t.Errorf("XTVERSION DCS ST: %+v (consumed %d)", ev, consumed)
	}

	// OSC color query reply terminated by BEL (\x07)
	ev, consumed = ParseEvent([]byte("\x1b]11;rgb:0000/0000/0000\x07"))
	if consumed != 24 || ev.Type != EventNone {
		t.Errorf("OSC BEL: %+v (consumed %d)", ev, consumed)
	}

	// OSC query reply terminated by ST (\x1b\)
	ev, consumed = ParseEvent([]byte("\x1b]11;rgb:0000/0000/0000\x1b\\"))
	if consumed != 25 || ev.Type != EventNone {
		t.Errorf("OSC ST: %+v (consumed %d)", ev, consumed)
	}

	// Incomplete string sequence with ESC at buffer boundary waits for ST
	ev, consumed = ParseEvent([]byte("\x1b]11\x1b"))
	if consumed != 0 || ev.Type != EventNone {
		t.Errorf("incomplete OSC at boundary: %+v (consumed %d)", ev, consumed)
	}

	// Incomplete string sequence waits for terminator
	ev, consumed = ParseEvent([]byte("\x1b]11;rgb:0000"))
	if consumed != 0 || ev.Type != EventNone {
		t.Errorf("incomplete OSC: %+v (consumed %d)", ev, consumed)
	}

	// Oversized string sequence without terminator consumes to avoid stall
	oversized := make([]byte, 4100)
	copy(oversized, []byte("\x1b]"))
	ev, consumed = ParseEvent(oversized)
	if consumed != 2 || ev.Type != EventNone {
		t.Errorf("oversized string sequence: %+v (consumed %d)", ev, consumed)
	}
}

// SGR mouse protocol variations, dragging, releases, and modifiers.
func TestParseSGRMouse(t *testing.T) {
	// 1. Left Click Press
	ev, consumed := ParseEvent([]byte("\x1b[<0;10;20M"))
	if consumed != 11 || ev.Type != EventMouse || ev.Mouse.Button != MouseLeft || ev.Mouse.Drag || ev.Mouse.X != 9 || ev.Mouse.Y != 19 {
		t.Errorf("Left click press failed: %+v (consumed %d)", ev, consumed)
	}

	// 2. Middle Click Press
	ev, consumed = ParseEvent([]byte("\x1b[<1;10;20M"))
	if consumed != 11 || ev.Type != EventMouse || ev.Mouse.Button != MouseMiddle || ev.Mouse.Drag {
		t.Errorf("Middle click press failed: %+v", ev)
	}

	// 3. Right Click Press
	ev, consumed = ParseEvent([]byte("\x1b[<2;10;20M"))
	if consumed != 11 || ev.Type != EventMouse || ev.Mouse.Button != MouseRight || ev.Mouse.Drag {
		t.Errorf("Right click press failed: %+v", ev)
	}

	// 4. Pure Hover / Pointer Motion (no buttons down)
	ev, consumed = ParseEvent([]byte("\x1b[<35;15;25M"))
	if consumed != 12 || ev.Type != EventMouse || ev.Mouse.Button != MouseNone || ev.Mouse.Drag {
		t.Errorf("Hover motion failed: %+v (consumed %d)", ev, consumed)
	}

	// 5. Drag with Left Button
	ev, consumed = ParseEvent([]byte("\x1b[<32;15;25M"))
	if consumed != 12 || ev.Type != EventMouse || ev.Mouse.Button != MouseLeft || !ev.Mouse.Drag {
		t.Errorf("Left drag failed: %+v", ev)
	}

	// 6. Drag with Middle Button
	ev, consumed = ParseEvent([]byte("\x1b[<33;15;25M"))
	if consumed != 12 || ev.Type != EventMouse || ev.Mouse.Button != MouseMiddle || !ev.Mouse.Drag {
		t.Errorf("Middle drag failed: %+v", ev)
	}

	// 7. Drag with Right Button
	ev, consumed = ParseEvent([]byte("\x1b[<34;15;25M"))
	if consumed != 12 || ev.Type != EventMouse || ev.Mouse.Button != MouseRight || !ev.Mouse.Drag {
		t.Errorf("Right drag failed: %+v", ev)
	}

	// 8. Motion with unknown button (> 2)
	ev, consumed = ParseEvent([]byte("\x1b[<96;15;25M"))
	if consumed != 12 || ev.Type != EventMouse || ev.Mouse.Button != MouseNone || !ev.Mouse.Drag {
		t.Errorf("Unknown drag failed: %+v", ev)
	}

	// 9. Button Release ('m' terminator)
	ev, consumed = ParseEvent([]byte("\x1b[<0;10;20m"))
	if consumed != 11 || ev.Type != EventMouse || ev.Mouse.Button != MouseRelease || ev.Mouse.Drag {
		t.Errorf("Release failed: %+v", ev)
	}

	// 10. Scroll Up & Down
	ev, _ = ParseEvent([]byte("\x1b[<64;10;20M"))
	if ev.Type != EventMouse || ev.Mouse.Button != MouseScrollUp {
		t.Errorf("Scroll up failed: %+v", ev)
	}
	ev, _ = ParseEvent([]byte("\x1b[<65;10;20M"))
	if ev.Type != EventMouse || ev.Mouse.Button != MouseScrollDown {
		t.Errorf("Scroll down failed: %+v", ev)
	}

	// 11. Mouse press with unmapped button code (3)
	ev, _ = ParseEvent([]byte("\x1b[<3;10;20M"))
	if ev.Type != EventMouse || ev.Mouse.Button != MouseNone {
		t.Errorf("Unmapped press failed: %+v", ev)
	}

	// 12. Mouse press with Shift, Alt, Ctrl modifiers (0 + 4 + 8 + 16 = 28)
	ev, _ = ParseEvent([]byte("\x1b[<28;10;20M"))
	if ev.Type != EventMouse || ev.Mouse.Button != MouseLeft || !ev.Mouse.Shift || !ev.Mouse.Alt || !ev.Mouse.Ctrl {
		t.Errorf("Mouse modifiers failed: %+v", ev)
	}

	// 13. Short/malformed SGR mouse (< 3 parameters)
	ev, consumed = ParseEvent([]byte("\x1b[<0;10M"))
	if consumed != 8 || ev.Type != EventNone {
		t.Errorf("Malformed SGR mouse: %+v (consumed %d)", ev, consumed)
	}
}

// Kitty Keyboard protocol (CSI u) full suite.
func TestParseKittyKeyboardProtocol(t *testing.T) {
	// Shift+Enter (\x1b[13;2u)
	ev, consumed := ParseEvent([]byte("\x1b[13;2u"))
	if consumed != 7 || ev.Type != EventKey || ev.Key.Type != KeyEnter || !ev.Key.Shift {
		t.Errorf("Shift+Enter CSI u failed: %+v (consumed %d)", ev, consumed)
	}

	// Alt+Enter (\x1b[13;3u)
	ev, consumed = ParseEvent([]byte("\x1b[13;3u"))
	if consumed != 7 || ev.Type != EventKey || ev.Key.Type != KeyEnter || !ev.Key.Alt {
		t.Errorf("Alt+Enter CSI u failed: %+v", ev)
	}

	// Ctrl+Enter (\x1b[13;5u)
	ev, consumed = ParseEvent([]byte("\x1b[13;5u"))
	if consumed != 7 || ev.Type != EventKey || ev.Key.Type != KeyEnter || !ev.Key.Ctrl {
		t.Errorf("Ctrl+Enter CSI u failed: %+v", ev)
	}

	// LF (\x1b[10;5u)
	ev, consumed = ParseEvent([]byte("\x1b[10;5u"))
	if consumed != 7 || ev.Type != EventKey || ev.Key.Type != KeyEnter || !ev.Key.Ctrl {
		t.Errorf("LF CSI u failed: %+v", ev)
	}

	// Shift+Tab (\x1b[9;2u)
	ev, consumed = ParseEvent([]byte("\x1b[9;2u"))
	if consumed != 6 || ev.Type != EventKey || ev.Key.Type != KeyTab || !ev.Key.Shift {
		t.Errorf("Shift+Tab CSI u failed: %+v", ev)
	}

	// Ctrl+Esc (\x1b[27;5u)
	ev, consumed = ParseEvent([]byte("\x1b[27;5u"))
	if consumed != 7 || ev.Type != EventKey || ev.Key.Type != KeyEsc || !ev.Key.Ctrl {
		t.Errorf("Ctrl+Esc CSI u failed: %+v", ev)
	}

	// Shift+Backspace (\x1b[127;2u)
	ev, consumed = ParseEvent([]byte("\x1b[127;2u"))
	if consumed != 8 || ev.Type != EventKey || ev.Key.Type != KeyBackspace || !ev.Key.Shift {
		t.Errorf("Shift+Backspace 127 CSI u failed: %+v", ev)
	}

	// Ctrl+Backspace (\x1b[8;5u)
	ev, consumed = ParseEvent([]byte("\x1b[8;5u"))
	if consumed != 6 || ev.Type != EventKey || ev.Key.Type != KeyBackspace || !ev.Key.Ctrl {
		t.Errorf("Ctrl+Backspace 8 CSI u failed: %+v", ev)
	}

	// Shift+Space (\x1b[32;2u)
	ev, consumed = ParseEvent([]byte("\x1b[32;2u"))
	if consumed != 7 || ev.Type != EventKey || ev.Key.Type != KeySpace || !ev.Key.Shift {
		t.Errorf("Shift+Space CSI u failed: %+v", ev)
	}

	// Ctrl+a (\x1b[97;5u)
	ev, consumed = ParseEvent([]byte("\x1b[97;5u"))
	if consumed != 7 || ev.Type != EventKey || ev.Key.Type != KeyRune || ev.Key.Ch != 'a' || !ev.Key.Ctrl {
		t.Errorf("Ctrl+a CSI u failed: %+v", ev)
	}

	// Empty CSI u params
	ev, consumed = ParseEvent([]byte("\x1b[u"))
	if consumed != 3 || ev.Type != EventNone {
		t.Errorf("Empty CSI u: %+v", ev)
	}

	// Keycode < 32 not in mapping
	ev, consumed = ParseEvent([]byte("\x1b[15u"))
	if consumed != 5 || ev.Type != EventNone {
		t.Errorf("Unmapped CSI u: %+v", ev)
	}
}

// Xterm modifyOtherKeys format (\x1b[27;<mod>;<keycode>~).
func TestParseModifyOtherKeys(t *testing.T) {
	// Shift+Enter (\x1b[27;2;13~)
	ev, consumed := ParseEvent([]byte("\x1b[27;2;13~"))
	if consumed != 10 || ev.Type != EventKey || ev.Key.Type != KeyEnter || !ev.Key.Shift {
		t.Errorf("Shift+Enter modifyOtherKeys failed: %+v (consumed %d)", ev, consumed)
	}

	// Ctrl+Enter (\x1b[27;5;13~)
	ev, consumed = ParseEvent([]byte("\x1b[27;5;13~"))
	if consumed != 10 || ev.Type != EventKey || ev.Key.Type != KeyEnter || !ev.Key.Ctrl {
		t.Errorf("Ctrl+Enter modifyOtherKeys failed: %+v", ev)
	}

	// Ctrl+LF (\x1b[27;5;10~)
	ev, consumed = ParseEvent([]byte("\x1b[27;5;10~"))
	if consumed != 10 || ev.Type != EventKey || ev.Key.Type != KeyEnter || !ev.Key.Ctrl {
		t.Errorf("Ctrl+LF modifyOtherKeys failed: %+v", ev)
	}

	// Shift+Tab (\x1b[27;2;9~)
	ev, consumed = ParseEvent([]byte("\x1b[27;2;9~"))
	if consumed != 9 || ev.Type != EventKey || ev.Key.Type != KeyTab || !ev.Key.Shift {
		t.Errorf("Shift+Tab modifyOtherKeys failed: %+v", ev)
	}

	// Ctrl+Esc (\x1b[27;5;27~)
	ev, consumed = ParseEvent([]byte("\x1b[27;5;27~"))
	if consumed != 10 || ev.Type != EventKey || ev.Key.Type != KeyEsc || !ev.Key.Ctrl {
		t.Errorf("Ctrl+Esc modifyOtherKeys failed: %+v", ev)
	}

	// Ctrl+a (\x1b[27;5;97~)
	ev, consumed = ParseEvent([]byte("\x1b[27;5;97~"))
	if consumed != 10 || ev.Type != EventKey || ev.Key.Type != KeyRune || ev.Key.Ch != 'a' || !ev.Key.Ctrl {
		t.Errorf("Ctrl+a modifyOtherKeys failed: %+v", ev)
	}

	// Short params (\x1b[27~)
	ev, consumed = ParseEvent([]byte("\x1b[27~"))
	if consumed != 5 || ev.Type != EventNone {
		t.Errorf("Short modifyOtherKeys: %+v", ev)
	}

	// Keycode < 32 not in mapping
	ev, consumed = ParseEvent([]byte("\x1b[27;5;1~"))
	if consumed != 9 || ev.Type != EventNone {
		t.Errorf("Unmapped modifyOtherKeys: %+v", ev)
	}
}

// Split input: delivering a sequence one byte at a time produces the exact same
// event as delivering it whole once the final byte arrives.
func TestParseSplitInputByteByByte(t *testing.T) {
	sequences := []struct {
		name      string
		raw       string
		wantEvent Event
	}{
		{
			name:      "Ctrl+Up",
			raw:       "\x1b[1;5A",
			wantEvent: Event{Type: EventKey, Key: KeyEvent{Type: KeyArrowUp, Ctrl: true}},
		},
		{
			name:      "Shift+F12",
			raw:       "\x1b[24;2~",
			wantEvent: Event{Type: EventKey, Key: KeyEvent{Type: KeyF12, Shift: true}},
		},
		{
			name: "SGR Mouse drag",
			raw:  "\x1b[<32;15;25M",
			wantEvent: Event{
				Type: EventMouse,
				Mouse: MouseEvent{
					Button: MouseLeft,
					X:      14,
					Y:      24,
					Drag:   true,
				},
			},
		},
		{
			name: "DCS Version reply",
			raw:  "\x1bP>|Limoni 1.0\x1b\\",
			wantEvent: Event{
				Type:  EventReply,
				Reply: ReplyEvent{Kind: ReplyVersion, Version: "Limoni 1.0"},
			},
		},
	}

	for _, seq := range sequences {
		// Feed prefix by prefix (1 byte, 2 bytes, ..., up to full length - 1)
		for i := 1; i < len(seq.raw); i++ {
			ev, consumed := ParseEvent([]byte(seq.raw[:i]))
			if consumed != 0 || ev.Type != EventNone {
				t.Errorf("%s (prefix len %d): should wait for more data, got ev=%+v consumed=%d",
					seq.name, i, ev, consumed)
			}
		}

		// Full sequence completes and parses correctly
		ev, consumed := ParseEvent([]byte(seq.raw))
		if consumed != len(seq.raw) || ev != seq.wantEvent {
			t.Errorf("%s (full): got ev=%+v consumed=%d, want ev=%+v consumed=%d",
				seq.name, ev, consumed, seq.wantEvent, len(seq.raw))
		}
	}
}
