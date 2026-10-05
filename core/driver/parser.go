package driver

import (
	"bytes"
	"strings"
	"unicode/utf8"
)

// ParseEvent parses a single TUI event from the incoming byte stream.
// It returns the parsed event and the number of bytes it consumed.
// If the buffer holds an incomplete ANSI sequence it consumes 0 bytes, and the caller should wait for more data.
func ParseEvent(buf []byte) (Event, int) {
	if len(buf) == 0 {
		return Event{}, 0
	}
	// A bracketed paste whose end has not arrived yet: a long paste comes in
	// several reads. Parsed now, its start marker would be dropped as an
	// unknown sequence and the pasted text taken for key presses, which
	// typed shortcuts into an application.
	if bytes.HasPrefix(buf, pasteStart) {
		return Event{}, 0
	}

	// 1. Not an ANSI escape sequence: a plain UTF-8 key press
	if buf[0] != '\x1b' {
		r, size := utf8.DecodeRune(buf)
		if r == utf8.RuneError {
			// An incomplete UTF-8 character: keep waiting for data
			if !utf8.FullRune(buf) {
				return Event{}, 0
			}
			// Invalid UTF-8: consume 1 byte and move on
			return Event{}, 1
		}

		ev := Event{Type: EventKey}
		switch r {
		case '\n':
			ev.Key.Type = KeyEnter
			ev.Key.Ctrl = true // Ctrl+J (ASCII 10 LF)
		case '\r':
			ev.Key.Type = KeyEnter
		case 127, '\b':
			ev.Key.Type = KeyBackspace
		case '\t':
			ev.Key.Type = KeyTab
		case ' ':
			ev.Key.Type = KeySpace
		default:
			// Ctrl characters: ASCII 1-26 map to Ctrl-A through Ctrl-Z.
			if r >= 1 && r <= 26 {
				ev.Key.Type = KeyRune
				ev.Key.Ch = rune('a' + r - 1)
				ev.Key.Ctrl = true
			} else {
				ev.Key.Type = KeyRune
				ev.Key.Ch = r
			}
		}
		return ev, size
	}

	// 2. Sequences starting with Escape (\x1b)
	if len(buf) == 1 {
		// A lone ESC in the buffer. The event loop must wait for its timeout
		// (escTimeoutDuration) to see whether a CSI or Alt sequence follows.
		return Event{}, 0
	}

	switch buf[1] {
	case '[': // CSI (Control Sequence Introducer) sequence
		return parseCSI(buf)
	case 'O': // SS3 application cursor keys and F1-F4.
		if len(buf) < 3 {
			return Event{}, 0
		}
		ev := Event{Type: EventKey}
		switch buf[2] {
		case 'A':
			ev.Key.Type = KeyArrowUp
		case 'B':
			ev.Key.Type = KeyArrowDown
		case 'C':
			ev.Key.Type = KeyArrowRight
		case 'D':
			ev.Key.Type = KeyArrowLeft
		case 'H':
			ev.Key.Type = KeyHome
		case 'F':
			ev.Key.Type = KeyEnd
		case 'P':
			ev.Key.Type = KeyF1
		case 'Q':
			ev.Key.Type = KeyF2
		case 'R':
			ev.Key.Type = KeyF3
		case 'S':
			ev.Key.Type = KeyF4
		case 'M':
			ev.Key.Type = KeyEnter // keypad Enter in application keypad mode
		default:
			// The rest of the keypad in application keypad mode (DECKPAM) types
			// its character: ESC O o is the keypad's /.
			if i := strings.IndexByte(ss3Keypad, buf[2]); i >= 0 {
				ev.Key.Type = KeyRune
				ev.Key.Ch = rune(ss3KeypadChars[i])
				return ev, 3
			}
			return Event{}, 3
		}
		return ev, 3
	case '_', ']', 'P', '^': // APC (\x1b_), OSC (\x1b]), DCS (\x1bP), PM (\x1b^)
		return parseStringSequence(buf)
	default:
		// ESC + character (Alt + key)
		r, size := utf8.DecodeRune(buf[1:])
		if r == utf8.RuneError {
			if !utf8.FullRune(buf[1:]) {
				return Event{}, 0
			}
			return Event{}, 2
		}
		if r == '\r' || r == '\n' {
			return Event{Type: EventKey, Key: KeyEvent{Type: KeyEnter, Alt: true}}, 1 + size
		}
		ev := Event{Type: EventKey, Key: KeyEvent{Type: KeyRune, Ch: r, Alt: true}}
		return ev, 1 + size
	}
}

// maxCSIParams bounds the parameters kept from one CSI sequence. Keys use at
// most three; DA1 replies list a dozen or so attributes. Parameters beyond the
// bound are dropped rather than growing a slice on the input path.
const maxCSIParams = 32

// parseCSI decodes a control sequence introduced by ESC [.
func parseCSI(buf []byte) (Event, int) {
	if len(buf) < 3 {
		return Event{}, 0
	}

	// Find the final byte. Keys and replies end in a letter or '~'.
	endIdx := -1
	for i := 2; i < len(buf); i++ {
		c := buf[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || c == '~' {
			endIdx = i
			break
		}
	}

	// No final byte yet: wait for the rest of the sequence.
	if endIdx == -1 {
		// A DA1 reply with many attributes can run past 32 bytes; give up only
		// on something no terminal would send.
		if len(buf) > 128 {
			return Event{}, 1
		}
		return Event{}, 0
	}

	cmd := buf[endIdx]
	raw := buf[2:endIdx]
	consumed := endIdx + 1

	// A private marker ('<', '=', '>', '?') may open the parameters, and
	// intermediate bytes (0x20–0x2F, e.g. '$') may close them. Keys carry
	// neither, except SGR mouse ('<'); terminal replies carry both.
	var marker, intermediate byte
	if len(raw) > 0 && raw[0] >= '<' && raw[0] <= '?' {
		marker = raw[0]
	}
	if n := len(raw); n > 0 && raw[n-1] >= 0x20 && raw[n-1] <= 0x2F {
		intermediate = raw[n-1]
	}

	if marker == '<' && (cmd == 'M' || cmd == 'm') {
		return parseSGRMouse(raw[1:], cmd, consumed)
	}

	// Split the ;-separated parameters into a fixed array: no allocation.
	var store [maxCSIParams]int
	n := 0
	currentVal, hasVal := 0, false
	// The kitty keyboard protocol adds ':'-separated sub-parameters (shifted
	// and base-layout keys, event types). Only the first of each is kept:
	// reading "97:65" as one number would make it 9765. The one sub-parameter
	// read is the event type after the modifiers (CSI 119;1:3u is w let go).
	sub := 0 // which sub-parameter the digits belong to; 0 is the parameter
	eventType, subVal := 0, 0
	for _, c := range raw {
		switch {
		case c >= '0' && c <= '9':
			switch {
			case sub == 0:
				currentVal = currentVal*10 + int(c-'0')
				hasVal = true
			case sub == 1 && n == 1:
				subVal = subVal*10 + int(c-'0')
			}
		case c == ':':
			if sub == 1 && n == 1 {
				eventType = subVal
			}
			sub++
		case c == ';':
			if sub == 1 && n == 1 {
				eventType = subVal
			}
			if n < maxCSIParams {
				store[n] = currentVal
				n++
			}
			currentVal, hasVal, sub = 0, false, 0
		}
	}
	if sub == 1 && n == 1 {
		eventType = subVal
	}
	if hasVal && n < maxCSIParams {
		store[n] = currentVal
		n++
	}
	params := store[:n]

	if marker == '?' {
		return parseReplyCSI(params, cmd, intermediate), consumed
	}
	if cmd == 'R' && marker == 0 && intermediate == 0 && len(params) == 2 {
		return Event{Type: EventReply, Reply: ReplyEvent{Kind: ReplyCursor, Row: params[0], Col: params[1]}}, consumed
	}
	if marker != 0 || intermediate != 0 {
		// Some other reply (DA2 is CSI > ... c): consume it so its bytes never
		// reach the application as keystrokes.
		return Event{}, consumed
	}

	ev := csiKey(cmd, params)
	if ev.Type == EventKey {
		// Only applications that asked for them (Terminal.SetKeyReleases)
		// are sent repeats and releases marked as such.
		switch eventType {
		case 2:
			ev.Key.Repeat = true
		case 3:
			ev.Key.Release = true
		}
	}
	return ev, consumed
}

// kittyKeypadChars are the characters of the kitty keyboard protocol's keypad
// keys 57409 (KP_DECIMAL) to 57416 (KP_SEPARATOR). 57414, KP_ENTER, is Enter.
var kittyKeypadChars = [...]rune{'.', '/', '*', '-', '+', 0, '=', ','}

// ss3Keypad are the final bytes of the keypad's keys in application keypad
// mode (ESC O p is keypad 0), and ss3KeypadChars the characters they type.
const (
	ss3Keypad      = "pqrstuvwxyjkmnoXl"
	ss3KeypadChars = "0123456789*+-./=,"
)

// csiKey decodes the key (or focus change) a CSI sequence ending in cmd
// stands for.
func csiKey(cmd byte, params []int) Event {
	// Build the event from the final byte.
	switch cmd {
	case 'A': // Up arrow
		return makeKeyEvent(KeyArrowUp, params)
	case 'B': // Down arrow
		return makeKeyEvent(KeyArrowDown, params)
	case 'C': // Right arrow
		return makeKeyEvent(KeyArrowRight, params)
	case 'D': // Left arrow
		return makeKeyEvent(KeyArrowLeft, params)
	case 'Z': // Shift+Tab (backtab)
		return Event{Type: EventKey, Key: KeyEvent{Type: KeyTab, Shift: true}}
	case 'H': // Home
		return makeKeyEvent(KeyHome, params)
	case 'F': // End
		return makeKeyEvent(KeyEnd, params)
	case 'P': // F1, F2 and F4 with modifiers: CSI 1 ; mod P. F3's R would be
		// read as a cursor report, so terminals send CSI 13 ~ for it instead.
		return makeKeyEvent(KeyF1, params)
	case 'Q':
		return makeKeyEvent(KeyF2, params)
	case 'S':
		return makeKeyEvent(KeyF4, params)
	case 'I': // Focus Gained
		return Event{Type: EventFocus, Focus: FocusEvent{Gained: true}}
	case 'O': // Focus Lost
		return Event{Type: EventFocus, Focus: FocusEvent{Gained: false}}
	case 'u':
		// Kitty keyboard protocol / CSI u format: \x1b[<keycode>;<modifiers>u
		if len(params) == 0 {
			return Event{}
		}
		keyCode := params[0]
		mod := 0
		if len(params) > 1 {
			mod = params[1]
		}
		shift, alt, ctrl := decodeModifiers(mod)
		switch keyCode {
		case 13, 10:
			return Event{Type: EventKey, Key: KeyEvent{Type: KeyEnter, Shift: shift, Alt: alt, Ctrl: ctrl}}
		case 9:
			return Event{Type: EventKey, Key: KeyEvent{Type: KeyTab, Shift: shift, Alt: alt, Ctrl: ctrl}}
		case 27:
			return Event{Type: EventKey, Key: KeyEvent{Type: KeyEsc, Shift: shift, Alt: alt, Ctrl: ctrl}}
		case 127, 8:
			return Event{Type: EventKey, Key: KeyEvent{Type: KeyBackspace, Shift: shift, Alt: alt, Ctrl: ctrl}}
		case 32:
			return Event{Type: EventKey, Key: KeyEvent{Type: KeySpace, Shift: shift, Alt: alt, Ctrl: ctrl}}
		default:
			// Keys without a character of their own live in the Private Use
			// Area: the keypad's keys are kept (they type their character, and
			// its Enter is Enter), the rest (lock keys, media keys, F13 and up)
			// are dropped rather than typed as text.
			if keyCode >= 57344 && keyCode <= 63743 {
				switch {
				case keyCode >= 57399 && keyCode <= 57408: // KP_0 … KP_9
					return Event{Type: EventKey, Key: KeyEvent{Type: KeyRune, Ch: rune('0' + keyCode - 57399), Shift: shift, Alt: alt, Ctrl: ctrl}}
				case keyCode == 57414: // KP_ENTER
					return Event{Type: EventKey, Key: KeyEvent{Type: KeyEnter, Shift: shift, Alt: alt, Ctrl: ctrl}}
				case keyCode >= 57409 && keyCode <= 57416: // KP_DECIMAL … KP_SEPARATOR
					if ch := kittyKeypadChars[keyCode-57409]; ch != 0 {
						return Event{Type: EventKey, Key: KeyEvent{Type: KeyRune, Ch: ch, Shift: shift, Alt: alt, Ctrl: ctrl}}
					}
				}
				return Event{}
			}
			if keyCode >= 32 {
				return Event{Type: EventKey, Key: KeyEvent{Type: KeyRune, Ch: rune(keyCode), Shift: shift, Alt: alt, Ctrl: ctrl}}
			}
		}
		return Event{}
	case '~':
		// Keypad and function keys (\x1b[number~)
		if len(params) == 0 {
			return Event{}
		}
		num := params[0]
		var modParams []int
		if len(params) > 1 {
			modParams = params[1:]
		}

		var kt KeyType
		switch num {
		case 1, 7:
			kt = KeyHome
		case 2:
			kt = KeyInsert
		case 3:
			kt = KeyDelete
		case 4, 8:
			kt = KeyEnd
		case 5:
			kt = KeyPageUp
		case 6:
			kt = KeyPageDown
		case 11:
			kt = KeyF1
		case 12:
			kt = KeyF2
		case 13:
			kt = KeyF3
		case 14:
			kt = KeyF4
		case 15:
			kt = KeyF5
		case 17:
			kt = KeyF6
		case 18:
			kt = KeyF7
		case 19:
			kt = KeyF8
		case 20:
			kt = KeyF9
		case 21:
			kt = KeyF10
		case 23:
			kt = KeyF11
		case 24:
			kt = KeyF12
		case 27:
			// Xterm modifyOtherKeys format: \x1b[27;<mod>;<keycode>~
			if len(params) >= 3 {
				mod := params[1]
				keyCode := params[2]
				shift, alt, ctrl := decodeModifiers(mod)
				if keyCode == 13 || keyCode == 10 {
					return Event{Type: EventKey, Key: KeyEvent{Type: KeyEnter, Shift: shift, Alt: alt, Ctrl: ctrl}}
				} else if keyCode == 9 {
					return Event{Type: EventKey, Key: KeyEvent{Type: KeyTab, Shift: shift, Alt: alt, Ctrl: ctrl}}
				} else if keyCode == 27 {
					return Event{Type: EventKey, Key: KeyEvent{Type: KeyEsc, Shift: shift, Alt: alt, Ctrl: ctrl}}
				} else if keyCode >= 32 {
					return Event{Type: EventKey, Key: KeyEvent{Type: KeyRune, Ch: rune(keyCode), Shift: shift, Alt: alt, Ctrl: ctrl}}
				}
			}
			return Event{}
		default:
			return Event{}
		}
		return makeKeyEvent(kt, modParams)
	}

	return Event{}
}

// makeKeyEvent decodes the key modifiers and returns the KeyEvent.
func makeKeyEvent(kt KeyType, params []int) Event {
	ev := Event{Type: EventKey}
	ev.Key.Type = kt
	if len(params) > 1 {
		ev.Key.Shift, ev.Key.Alt, ev.Key.Ctrl = decodeModifiers(params[1])
	} else if len(params) == 1 {
		ev.Key.Shift, ev.Key.Alt, ev.Key.Ctrl = decodeModifiers(params[0])
	}
	return ev
}

// decodeModifiers decodes the standard VT100/xterm modifier codes.
func decodeModifiers(code int) (shift, alt, ctrl bool) {
	if code <= 1 {
		return
	}
	shift = (code-1)&1 != 0
	alt = (code-1)&2 != 0
	ctrl = (code-1)&4 != 0
	return
}

// parseReplyCSI decodes the replies to the queries in ProbeQueries that are
// CSI sequences with a '?' marker.
func parseReplyCSI(params []int, cmd, intermediate byte) Event {
	switch {
	case cmd == 'c' && intermediate == 0:
		// DA1: CSI ? class ; attr ; attr ... c
		var attrs uint64
		for i, p := range params {
			if i > 0 && p >= 0 && p < 64 {
				attrs |= 1 << uint(p)
			}
		}
		return Event{Type: EventReply, Reply: ReplyEvent{Kind: ReplyPrimaryDA, Attributes: attrs}}
	case cmd == 'y' && intermediate == '$' && len(params) >= 2:
		// DECRPM: CSI ? mode ; setting $ y
		return Event{Type: EventReply, Reply: ReplyEvent{Kind: ReplyMode, Mode: params[0], Setting: params[1]}}
	case cmd == 'u' && intermediate == 0:
		// Kitty keyboard flags: CSI ? flags u. Without the marker this would be
		// read as a key press of the code point "flags".
		flags := 0
		if len(params) > 0 {
			flags = params[0]
		}
		return Event{Type: EventReply, Reply: ReplyEvent{Kind: ReplyKittyKeyboard, Flags: flags}}
	}
	return Event{}
}

// parseSGRMouse decodes an SGR mouse report, CSI < btn ; x ; y M (press) or
// m (release). raw is the parameter text after the '<'.
func parseSGRMouse(raw []byte, cmd byte, consumed int) (Event, int) {
	var params [3]int
	n := 0
	currentVal := 0
	for _, c := range raw {
		if c >= '0' && c <= '9' {
			currentVal = currentVal*10 + int(c-'0')
		} else if c == ';' {
			if n < len(params) {
				params[n] = currentVal
			}
			n++
			currentVal = 0
		}
	}
	if n < len(params) {
		params[n] = currentVal
	}
	n++

	if n < 3 {
		return Event{}, consumed
	}
	btnCode := params[0]
	mouseX := params[1]
	mouseY := params[2]

	ev := Event{Type: EventMouse}

	// Convert 1-based terminal coordinates to 0-based ones
	if mouseX > 0 {
		ev.Mouse.X = uint16(mouseX - 1)
	}
	if mouseY > 0 {
		ev.Mouse.Y = uint16(mouseY - 1)
	}

	// Check the modifier bits (Shift: 4, Alt: 8, Ctrl: 16)
	ev.Mouse.Shift = (btnCode & 4) != 0
	ev.Mouse.Alt = (btnCode & 8) != 0
	ev.Mouse.Ctrl = (btnCode & 16) != 0

	// Clear the modifier bits to separate the button and the drag flag
	btnRaw := btnCode & ^(4 | 8 | 16)
	isMotion := (btnRaw & 32) != 0
	btnBase := btnRaw & ^32

	if cmd == 'm' {
		// Key release event
		ev.Mouse.Button = MouseRelease
		ev.Mouse.Drag = false
	} else if isMotion {
		if btnBase == 3 {
			// Motion without buttons (pure hover / pointer motion)
			ev.Mouse.Button = MouseNone
			ev.Mouse.Drag = false
		} else {
			// Motion with a button held (a real drag)
			ev.Mouse.Drag = true
			switch btnBase {
			case 0:
				ev.Mouse.Button = MouseLeft
			case 1:
				ev.Mouse.Button = MouseMiddle
			case 2:
				ev.Mouse.Button = MouseRight
			default:
				ev.Mouse.Button = MouseNone
			}
		}
	} else {
		ev.Mouse.Drag = false
		switch btnBase {
		case 0:
			ev.Mouse.Button = MouseLeft
		case 1:
			ev.Mouse.Button = MouseMiddle
		case 2:
			ev.Mouse.Button = MouseRight
		case 64:
			ev.Mouse.Button = MouseScrollUp
		case 65:
			ev.Mouse.Button = MouseScrollDown
		case 66:
			ev.Mouse.Button = MouseScrollLeft
		case 67:
			ev.Mouse.Button = MouseScrollRight
		default:
			ev.Mouse.Button = MouseNone
		}
	}

	return ev, consumed
}

// parseStringSequence handles string escape sequences: APC (\x1b_), OSC (\x1b]), DCS (\x1bP), and PM (\x1b^).
// These sequences are terminated by String Terminator (ST: \x1b\) or BEL (\x07).
// Unhandled internal terminal responses (e.g. Kitty graphics ACK, OSC queries) are cleanly consumed
// as EventNone, preventing raw response bytes from leaking into keyboard event handlers.
func parseStringSequence(buf []byte) (Event, int) {
	if len(buf) < 2 {
		return Event{}, 0
	}

	for i := 2; i < len(buf); i++ {
		// BEL (\x07) terminator (common in OSC)
		if buf[i] == '\x07' {
			return stringSequenceEvent(buf[:i]), i + 1
		}
		// ST (\x1b\) terminator
		if buf[i] == '\x1b' {
			if i+1 < len(buf) {
				if buf[i+1] == '\\' {
					return stringSequenceEvent(buf[:i]), i + 2
				}
			} else {
				// ESC at the buffer boundary; wait for next byte to check for ST
				return Event{}, 0
			}
		}
	}

	// Terminator not yet present in buffer
	if len(buf) > 4096 {
		// Avoid indefinite stall if an oversized/malformed sequence arrives
		return Event{Type: EventNone}, 2
	}
	return Event{}, 0
}

// stringSequenceEvent interprets a complete string sequence, without its
// terminator. Only the XTVERSION reply (DCS > | text) carries anything Limoni
// uses; every other one is consumed silently.
func stringSequenceEvent(seq []byte) Event {
	if len(seq) >= 4 && seq[1] == 'P' && seq[2] == '>' && seq[3] == '|' {
		return Event{Type: EventReply, Reply: ReplyEvent{Kind: ReplyVersion, Version: string(seq[4:])}}
	}
	return Event{Type: EventNone}
}
