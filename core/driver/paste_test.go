package driver

import "testing"

// Bracketed paste wraps pasted text between \x1b[200~ and \x1b[201~.
func TestParseBracketedPaste(t *testing.T) {
	// Standard multi-line text paste
	event, consumed := ParseBracketedPaste([]byte("\x1b[200~hello\nworld\x1b[201~"))
	if consumed != 23 || event.Type != EventPaste || event.Paste.Text != "hello\nworld" {
		t.Fatalf("paste = %+v consumed=%d, want 'hello\\nworld' (23 bytes)", event, consumed)
	}

	// Pasted text containing embedded escape sequences must survive unparsed.
	embeddedEscapes := "\x1b[1;5A\x1b[31mred\x1b[0m\x1b[<0;10;20M"
	raw := "\x1b[200~" + embeddedEscapes + "\x1b[201~"
	event, consumed = ParseBracketedPaste([]byte(raw))
	if consumed != len(raw) || event.Type != EventPaste || event.Paste.Text != embeddedEscapes {
		t.Errorf("paste with embedded escapes = %+v, want %q", event, embeddedEscapes)
	}

	// Incomplete paste buffer without end delimiter waits for more data
	incomplete := []byte("\x1b[200~halfway pasted text")
	event, consumed = ParseBracketedPaste(incomplete)
	if consumed != 0 || event.Type != EventNone {
		t.Errorf("incomplete paste: got %+v consumed=%d, want 0", event, consumed)
	}

	// Buffer without start delimiter returns 0
	nonPaste := []byte("regular text without escape")
	event, consumed = ParseBracketedPaste(nonPaste)
	if consumed != 0 || event.Type != EventNone {
		t.Errorf("non-paste buffer: got %+v consumed=%d, want 0", event, consumed)
	}

	// Split input: delivering paste byte-by-byte produces nothing until the end marker arrives
	for i := 1; i < len(raw); i++ {
		ev, n := ParseBracketedPaste([]byte(raw[:i]))
		if n != 0 || ev.Type != EventNone {
			t.Errorf("split paste (len %d): got %+v consumed=%d, want 0", i, ev, n)
		}
	}
	ev, n := ParseBracketedPaste([]byte(raw))
	if n != len(raw) || ev.Paste.Text != embeddedEscapes {
		t.Errorf("final split paste: got %+v consumed=%d", ev, n)
	}
}
