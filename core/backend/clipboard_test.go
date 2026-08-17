package backend

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
)

func TestClipboard_SetClipboardString(t *testing.T) {
	text := "Hello Limoni TUI!"
	seq := SetClipboardString(text)

	if !strings.HasPrefix(seq, "\x1b]52;c;") || !strings.HasSuffix(seq, "\x07") {
		t.Fatalf("unexpected OSC 52 structure: %q", seq)
	}

	payload := strings.TrimSuffix(strings.TrimPrefix(seq, "\x1b]52;c;"), "\x07")
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		t.Fatalf("failed to decode base64 payload: %v", err)
	}

	if string(decoded) != text {
		t.Errorf("expected %q, got %q", text, string(decoded))
	}
}

func TestClipboard_WriteClipboard(t *testing.T) {
	var buf bytes.Buffer
	err := WriteClipboard(&buf, "Sample copy")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected buffer to have written bytes")
	}
}
