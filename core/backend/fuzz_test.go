package backend

import "testing"

func FuzzParseEvent(f *testing.F) {
	// Seed corpus with valid, partial, and complex escape sequences
	f.Add([]byte("A"))
	f.Add([]byte("\x1b"))
	f.Add([]byte("\x1b["))
	f.Add([]byte("\x1b[A"))
	f.Add([]byte("\x1b[1;5A"))
	f.Add([]byte("\x1b[<0;10;20M"))
	f.Add([]byte("\x1b[200~pasted text\x1b[201~"))
	f.Add([]byte("\x1b[?2026h"))
	f.Add([]byte("Emoji 🚀 and CJK 日本語"))
	f.Add([]byte{0, 1, 2, 3, 4, 127, 255})

	f.Fuzz(func(t *testing.T, data []byte) {
		// ParseEvent must never panic regardless of input
		_, consumed := ParseEvent(data)

		// Consumed must never be negative and never exceed len(data)
		if consumed < 0 || consumed > len(data) {
			t.Fatalf("invalid consumed %d for input len %d", consumed, len(data))
		}
	})
}
