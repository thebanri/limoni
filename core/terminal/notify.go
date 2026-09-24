package terminal

import (
	"unicode/utf8"

	"github.com/thebanri/limoni/core/driver"
)

// NotifyProtocol is the escape sequence a terminal turns into a desktop
// notification.
type NotifyProtocol uint8

const (
	// NotifyNone: the terminal is not known to show notifications, and
	// Notify writes nothing.
	NotifyNone NotifyProtocol = iota
	// NotifyOSC9 is `OSC 9 ; text`, from iTerm2: one line of text, no title.
	// iTerm2, WezTerm, Ghostty, foot and kitty show it.
	NotifyOSC9
	// NotifyOSC777 is urxvt's `OSC 777 ; notify ; title ; body`.
	NotifyOSC777
	// NotifyOSC99 is kitty's own protocol, with a real title and body.
	NotifyOSC99
)

// Notify asks the terminal to show a desktop notification, and reports
// whether it wrote one. It writes nothing unless the capability profile names
// a protocol — detected for kitty, iTerm2, WezTerm, Ghostty and foot, or set
// with LIMONI_NOTIFY=9|777|99 — because a terminal that does not know the
// sequence may print it. Control characters are stripped from both strings,
// so neither can end the sequence early and smuggle in another.
func (t *Terminal) Notify(title, body string) bool {
	if t == nil || t.driver == nil {
		return false
	}
	seq := notifySequence(t.caps.Notify, title, body)
	if seq == nil {
		return false
	}
	_, _ = t.driver.Write(seq)
	return true
}

func notifySequence(p NotifyProtocol, title, body string) []byte {
	title, body = sanitizeOSCText(title), sanitizeOSCText(body)
	if title == "" && body == "" {
		return nil
	}
	var seq []byte
	switch p {
	case NotifyOSC9:
		text := body
		if title != "" && body != "" {
			text = title + ": " + body
		} else if title != "" {
			text = title
		}
		// ConEmu and Windows Terminal read "OSC 9 ; <digits> ;" as a command
		// of their own (4 is the progress bar): keep text from looking like one.
		if startsWithCommandNumber(text) {
			text = " " + text
		}
		seq = append(seq, "\x1b]9;"...)
		seq = append(seq, text...)
		seq = append(seq, '\a')
	case NotifyOSC777:
		seq = append(seq, "\x1b]777;notify;"...)
		for i := 0; i < len(title); i++ { // the title ends at the first ';'
			if title[i] == ';' {
				seq = append(seq, ',')
			} else {
				seq = append(seq, title[i])
			}
		}
		seq = append(seq, ';')
		seq = append(seq, body...)
		seq = append(seq, '\a')
	case NotifyOSC99:
		// d=0 marks the title as unfinished; the body closes the notification.
		// Both halves share an id so kitty joins them.
		if title != "" && body != "" {
			seq = append(seq, "\x1b]99;i=limoni:d=0:p=title;"...)
			seq = append(seq, title...)
			seq = append(seq, "\x1b\\\x1b]99;i=limoni:d=1:p=body;"...)
			seq = append(seq, body...)
		} else {
			seq = append(seq, "\x1b]99;i=limoni:d=1:p=title;"...)
			seq = append(seq, title...)
			seq = append(seq, body...)
		}
		seq = append(seq, "\x1b\\"...)
	default:
		return nil
	}
	return seq
}

func startsWithCommandNumber(s string) bool {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	return i > 0 && i < len(s) && s[i] == ';'
}

// sanitizeOSCText keeps what may sit inside an OSC string: it drops C0
// controls and DEL, which end or nest a sequence, the C1 controls U+0080 to
// U+009F, which a UTF-8 terminal can read as ST, and invalid UTF-8.
func sanitizeOSCText(s string) string {
	clean := true
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r < 0x20 || r == 0x7F || (r >= 0x80 && r <= 0x9F) || (r == utf8.RuneError && size == 1) {
			clean = false
			break
		}
		i += size
	}
	if clean {
		return s
	}
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if !(r < 0x20 || r == 0x7F || (r >= 0x80 && r <= 0x9F) || (r == utf8.RuneError && size == 1)) {
			out = append(out, s[i:i+size]...)
		}
		i += size
	}
	return string(out)
}

// Pointer shapes, by their CSS names, which is what kitty, foot and Ghostty
// accept in OSC 22.
const (
	PointerDefault    = "default"
	PointerText       = "text"
	PointerHand       = "pointer"
	PointerCrosshair  = "crosshair"
	PointerMove       = "move"
	PointerResizeEW   = "ew-resize"
	PointerResizeNS   = "ns-resize"
	PointerGrab       = "grab"
	PointerGrabbing   = "grabbing"
	PointerNotAllowed = "not-allowed"
	PointerWait       = "wait"
	PointerHelp       = "help"
)

// SetPointerShape sets the mouse pointer shown over the terminal with
// OSC 22, by CSS cursor name ("pointer", "text", "ew-resize"). It writes only
// when the shape changes and the capability profile allows it (kitty, foot,
// Ghostty, or LIMONI_POINTER=1); the empty string is the default pointer.
//
// Widgets do not call this: a ClickAction with Pointer set asks for a shape
// over its area, and the terminal applies it as the mouse moves.
func (t *Terminal) SetPointerShape(shape string) {
	if t == nil || t.driver == nil || !t.caps.PointerShape {
		return
	}
	if shape == PointerDefault {
		shape = ""
	}
	if shape == t.pointerShape {
		return
	}
	t.pointerShape = shape
	var buf [64]byte
	seq := append(buf[:0], "\x1b]22;"...)
	for i := 0; i < len(shape); i++ {
		if c := shape[i]; c > 0x20 && c < 0x7F {
			seq = append(seq, c)
		}
	}
	seq = append(seq, "\x1b\\"...)
	_, _ = t.driver.Write(seq)
}

// ResetPointerShape puts the default pointer back, if the application
// changed it. The application runtime calls it on exit.
func (t *Terminal) ResetPointerShape() { t.SetPointerShape("") }

// updatePointer applies the Pointer of the topmost click action under the
// mouse. While a drag holds the mouse, the shape stays as it was.
func (t *Terminal) updatePointer(ev driver.MouseEvent) {
	if !t.caps.PointerShape || t.mouseCaptureHandler != nil {
		return
	}
	t.SetPointerShape(t.frame.pointerAt(ev.X, ev.Y))
}
