package backend

import "testing"

func TestParseBracketedPaste(t *testing.T) {
	event, consumed := ParseBracketedPaste([]byte("\x1b[200~hello\nworld\x1b[201~"))
	if consumed == 0 || event.Type != EventPaste || event.Paste.Text != "hello\nworld" {
		t.Fatalf("paste = %+v consumed=%d", event, consumed)
	}
}
