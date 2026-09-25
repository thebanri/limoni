package driver

// parseChunk parses input that arrives already split into whole keys, as it
// does from xterm.js: every onData call carries one key's sequence, or one
// paste, and never half of one.
//
// That settles the one question a byte stream cannot answer. A terminal's
// ESC may be the Esc key or the start of a sequence still on its way, so the
// stream readers wait escTimeoutDuration for more before deciding. Here
// nothing more is coming, so an ESC left over at the end of a chunk is the
// Esc key. Without this the browser never delivered Esc at all.
func parseChunk(data []byte, emit func(Event)) {
	for len(data) > 0 {
		ev, consumed := ParseBracketedPaste(data)
		if consumed == 0 {
			ev, consumed = ParseEvent(data)
		}
		if consumed == 0 {
			if len(data) == 1 && data[0] == '\x1b' {
				emit(Event{Type: EventKey, Key: KeyEvent{Type: KeyEsc}})
			}
			return
		}
		if ev.Type != EventNone {
			emit(ev)
		}
		data = data[consumed:]
	}
}
