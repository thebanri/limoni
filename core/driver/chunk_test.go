package driver

import "testing"

// xterm.js hands over one key per chunk, so the ESC a stream reader would
// hold back for its timeout is, at the end of a chunk, the Esc key.
func TestParseChunk(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []KeyEvent
	}{
		{"lone ESC is the Esc key", "\x1b", []KeyEvent{{Type: KeyEsc}}},
		{"arrow is not Esc", "\x1b[A", []KeyEvent{{Type: KeyArrowUp}}},
		{"Alt+x is not Esc", "\x1bx", []KeyEvent{{Type: KeyRune, Ch: 'x', Alt: true}}},
		{"keys then Esc", "ab\x1b", []KeyEvent{{Type: KeyRune, Ch: 'a'}, {Type: KeyRune, Ch: 'b'}, {Type: KeyEsc}}},
		{"half a sequence is dropped, not Esc", "\x1b[", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []KeyEvent
			parseChunk([]byte(tc.in), func(ev Event) {
				if ev.Type != EventKey {
					t.Fatalf("event type %v, want a key", ev.Type)
				}
				got = append(got, ev.Key)
			})
			if len(got) != len(tc.want) {
				t.Fatalf("got %d keys %+v, want %+v", len(got), got, tc.want)
			}
			for i := range got {
				if got[i].Type != tc.want[i].Type || got[i].Ch != tc.want[i].Ch || got[i].Alt != tc.want[i].Alt {
					t.Errorf("key %d = %+v, want %+v", i, got[i], tc.want[i])
				}
			}
		})
	}
}
