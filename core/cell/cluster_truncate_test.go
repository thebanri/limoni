package cell

import "testing"

func TestTruncateCutsAtClusterBoundaries(t *testing.T) {
	family := "\U0001F468\u200D\U0001F469\u200D\U0001F467"
	for _, tc := range []struct {
		in    string
		max   int
		want  string
		width int
	}{
		{"hello", 3, "hel", 3},
		{"hello", 10, "hello", 5},
		{"日本語", 3, "日", 2}, // the second 日-width character does not fit in one column
		{"日本語", 4, "日本", 4},
		{"a" + family + "b", 2, "a", 1},
		{"a" + family + "b", 3, "a" + family, 3},
		{"e\u0301x", 1, "e\u0301", 1}, // the accent stays with its letter
		{"\U0001F1F9\U0001F1F7!", 1, "", 0},
		{"abc", 0, "", 0},
	} {
		got, w := Truncate(tc.in, tc.max)
		if got != tc.want || w != tc.width {
			t.Errorf("Truncate(%q, %d) = %q, %d; want %q, %d", tc.in, tc.max, got, w, tc.want, tc.width)
		}
	}
	if a := testing.AllocsPerRun(100, func() { Truncate("日本"+family+"text", 5) }); a != 0 {
		t.Errorf("Truncate allocates %.0f times", a)
	}
}
