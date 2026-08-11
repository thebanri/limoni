package backend

import "bytes"

var pasteStart = []byte("\x1b[200~")
var pasteEnd = []byte("\x1b[201~")

// ParseBracketedPaste parses one complete bracketed paste sequence.
func ParseBracketedPaste(buf []byte) (Event, int) {
	if !bytes.HasPrefix(buf, pasteStart) {
		return Event{}, 0
	}
	end := bytes.Index(buf[len(pasteStart):], pasteEnd)
	if end < 0 {
		return Event{}, 0
	}
	end += len(pasteStart)
	text := string(buf[len(pasteStart):end])
	return Event{Type: EventPaste, Paste: PasteEvent{Text: text}}, end + len(pasteEnd)
}
