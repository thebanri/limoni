package buffer

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"

	"github.com/thebanri/limoni/core/cell"
)

// Written as escapes, not literal characters: an editor that normalises
// Unicode would silently turn the decomposed "e" + U+0301 into the single code
// point U+00E9, and the combining-accent tests would stop testing anything.
const (
	flagTR   = "\U0001F1F9\U0001F1F7"                       // two regional indicators
	eAcute   = "e\u0301"                                    // e + combining acute
	family   = "\U0001F468\u200D\U0001F469\u200D\U0001F467" // five code points joined by ZWJ
	thumbsUp = "\U0001F44D\U0001F3FD"                       // emoji + skin tone modifier
	heartEmo = "\u2764\uFE0F"                               // heavy black heart + VS16
	conjunct = "\u0915\u094D\u0937"                         // Devanagari KSSA conjunct
)

func TestStringWidthMeasuresClusters(t *testing.T) {
	for _, tc := range []struct {
		name  string
		text  string
		width int
	}{
		{"ascii", "hello", 5},
		{"cjk", "日本", 4},
		{"combining accent", eAcute, 1},
		{"flag", flagTR, 2},
		{"zwj family", family, 2},
		{"skin tone", thumbsUp, 2},
		{"heart with VS16", heartEmo, 2},
		{"heart without VS16", "\u2764", 1},
		{"indic conjunct", conjunct, 1},
		{"mixed", "a" + flagTR + eAcute + "日", 1 + 2 + 1 + 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := cell.StringWidth(tc.text); got != tc.width {
				t.Errorf("StringWidth(%q) = %d, want %d", tc.text, got, tc.width)
			}
		})
	}
}

func TestClustersOccupyOneCell(t *testing.T) {
	buf := NewBuffer(cell.NewRect(0, 0, 20, 1))
	written := buf.SetString(0, 0, flagTR+eAcute+"x"+family+"!", cell.Style{})
	if written != 2+1+1+2+1 {
		t.Fatalf("wrote %d columns, want 7", written)
	}

	want := []struct {
		x    uint16
		text string
	}{
		{0, flagTR}, {2, eAcute}, {3, "x"}, {4, family}, {6, "!"},
	}
	for _, w := range want {
		got := cell.ClusterText(buf.Get(w.x, 0).Content)
		if got != w.text {
			t.Errorf("cell %d = %q, want %q", w.x, got, w.text)
		}
	}
	for _, x := range []uint16{1, 5} {
		if buf.Get(x, 0).Content != cell.RuneContinuation {
			t.Errorf("cell %d is not the continuation of a wide cluster", x)
		}
	}

	snap := buf.Snapshot()
	if !strings.HasPrefix(snap, flagTR+" "+eAcute+"x"+family+" !") {
		t.Errorf("snapshot = %q", snap)
	}
}

// The accent used to be dropped: the writer skipped zero-width runes.
func TestCombiningAccentIsKept(t *testing.T) {
	buf := NewBuffer(cell.NewRect(0, 0, 10, 1))
	buf.SetString(0, 0, eAcute, cell.Style{})
	if got := cell.ClusterText(buf.Get(0, 0).Content); got != eAcute {
		t.Fatalf("cell holds %q, the accent was lost", got)
	}
}

func TestDiffEmitsWholeClusters(t *testing.T) {
	area := cell.NewRect(0, 0, 40, 10)
	front, back := NewBuffer(area), NewBuffer(area)
	front.SetString(0, 3, flagTR+" "+eAcute+" "+family, cell.Style{})

	out, err := DiffWithOptions(front, back, nil, DiffOptions{EraseChar: true, RepeatChar: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, cluster := range []string{flagTR, eAcute, family} {
		if n := bytes.Count(out, []byte(cluster)); n != 1 {
			t.Errorf("%q emitted %d times, want once: %q", cluster, n, out)
		}
	}
	for i := range front.Content {
		if front.Content[i] != back.Content[i] {
			t.Fatalf("back buffer diverged at %d", i)
		}
	}
}

// REP repeats the preceding graphic character, and for a multi-code-point
// cluster a terminal may repeat only its last code point. Clusters are never
// repeated; they are written out.
func TestClustersAreNeverRepeated(t *testing.T) {
	area := cell.NewRect(0, 0, 40, 1)
	front, back := NewBuffer(area), NewBuffer(area)
	front.SetString(0, 0, strings.Repeat(eAcute, 20), cell.Style{})

	for _, path := range []struct {
		name string
		diff func() []byte
	}{
		{"full", func() []byte {
			out, _ := DiffWithOptions(front, NewBuffer(area), nil, DiffOptions{RepeatChar: true})
			return out
		}},
		{"inline", func() []byte {
			out, _ := DiffInline(front, NewBuffer(area), nil, DiffOptions{RepeatChar: true})
			return out
		}},
	} {
		out := path.diff()
		if n := bytes.Count(out, []byte(eAcute)); n != 20 {
			t.Errorf("%s: cluster written %d times, want 20: %q", path.name, n, out)
		}
		if bytes.Contains(out, []byte("b")) {
			t.Errorf("%s: REP used for a cluster: %q", path.name, out)
		}
	}
	_ = back
}

func TestCodePointModeRestoresPerRuneWidths(t *testing.T) {
	cell.SetGraphemeClusters(false)
	defer cell.SetGraphemeClusters(true)

	if got := cell.StringWidth(flagTR); got != 4 {
		t.Errorf("code-point mode flag width = %d, want 4 (two wide regional indicators)", got)
	}
	buf := NewBuffer(cell.NewRect(0, 0, 10, 1))
	buf.SetString(0, 0, flagTR, cell.Style{})
	if cell.IsCluster(buf.Get(0, 0).Content) {
		t.Error("code-point mode stored a cluster handle")
	}
}

func TestClusterWritesDoNotAllocate(t *testing.T) {
	buf := NewBuffer(cell.NewRect(0, 0, 80, 1))
	text := "status " + flagTR + " " + eAcute + " " + family + " " + thumbsUp + " 日本"
	buf.SetString(0, 0, text, cell.Style{}) // interns the clusters once
	if got := testing.AllocsPerRun(200, func() {
		buf.SetString(0, 0, text, cell.Style{})
	}); got != 0 {
		t.Errorf("writing already-seen clusters allocated %v times per run", got)
	}
}

func TestClusterInterningIsSafeConcurrently(t *testing.T) {
	var wg sync.WaitGroup
	handles := make([]rune, 32)
	for i := range handles {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			handles[i] = cell.ClusterContent(family, 2)
		}(i)
	}
	wg.Wait()
	for _, h := range handles {
		if h != handles[0] {
			t.Fatal("the same cluster was interned under two handles")
		}
	}
}

func BenchmarkSetStringASCII(b *testing.B) {
	buf := NewBuffer(cell.NewRect(0, 0, 120, 1))
	text := strings.Repeat("The quick brown fox ", 6)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.SetString(0, 0, text, cell.Style{})
	}
}

func BenchmarkSetStringClusters(b *testing.B) {
	buf := NewBuffer(cell.NewRect(0, 0, 120, 1))
	text := strings.Repeat(flagTR+" "+eAcute+" "+family+" ", 8)
	buf.SetString(0, 0, text, cell.Style{})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.SetString(0, 0, text, cell.Style{})
	}
}

// legacyScreen interprets diff output the way a terminal without mode 2027
// does: every code point advances the cursor by its own width, so a family
// emoji moves it six columns and a VS16 heart one. It records the column each
// ASCII letter lands in, which is all the resync tests need.
func legacyScreen(t *testing.T, out []byte) map[byte]int {
	t.Helper()
	legacyWidth := func(r rune) int {
		switch {
		case r == 0x200D || r == 0xFE0F || r == 0x0301 || r == 0x094D:
			return 0
		case r >= 0x1F000:
			return 2
		}
		return 1
	}
	landed := map[byte]int{}
	col := 0
	s := string(out)
	for i := 0; i < len(s); {
		switch {
		case s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[':
			j := i + 2
			for j < len(s) && (s[j] < 0x40 || s[j] > 0x7E) {
				j++
			}
			params, final := s[i+2:j], s[j]
			switch final {
			case 'H':
				col = 0
				if k := strings.IndexByte(params, ';'); k >= 0 {
					col = atoi(params[k+1:]) - 1
				}
			case 'G':
				col = atoi(params) - 1
			}
			i = j + 1
		case s[i] == '\r':
			col, i = 0, i+1
		case s[i] == '\n':
			i++
		default:
			r, size := utf8.DecodeRuneInString(s[i:])
			if r >= 'A' && r <= 'Z' {
				landed[byte(r)] = col
			}
			col += legacyWidth(r)
			i += size
		}
	}
	return landed
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}

// A terminal that does not render clusters as units disagrees with the buffer
// about how wide a cluster is. Without a resync after each cluster, every
// cell after it on the row lands in the wrong column. The diff re-anchors the
// cursor after a cluster, so the disagreement is confined to the cluster.
func TestDiffResyncsTheCursorAfterAClusterOnLegacyTerminals(t *testing.T) {
	area := cell.NewRect(0, 0, 30, 2)
	front := NewBuffer(area)
	front.SetString(0, 0, family+"A"+heartEmo+"B"+flagTR+"C", cell.Style{})
	front.SetString(0, 1, eAcute+"D"+conjunct+"E", cell.Style{})
	want := map[byte]int{'A': 2, 'B': 5, 'C': 8, 'D': 1, 'E': 3}

	for _, path := range []struct {
		name string
		diff func() ([]byte, error)
	}{
		{"sparse", func() ([]byte, error) { return diffSparse(front, NewBuffer(area), nil, DiffOptions{}) }},
		{"stream", func() ([]byte, error) { return diffFullStream(front, NewBuffer(area), nil, DiffOptions{}) }},
		{"inline", func() ([]byte, error) { return DiffInline(front, NewBuffer(area), nil, DiffOptions{}) }},
	} {
		t.Run(path.name, func(t *testing.T) {
			out, err := path.diff()
			if err != nil {
				t.Fatal(err)
			}
			got := legacyScreen(t, out)
			for letter, col := range want {
				if got[letter] != col {
					t.Errorf("%c landed in column %d, want %d: %q", letter, got[letter], col, out)
				}
			}
		})
	}
}

// A terminal that confirmed mode 2027 advances by the cluster's width, so the
// re-anchoring above is wasted bytes there: the encoder trusts the cursor
// instead, and every cell still lands where the buffer put it.
func TestDiffTrustsTheCursorWhenTheTerminalMeasuresClusters(t *testing.T) {
	area := cell.NewRect(0, 0, 30, 2)
	front := NewBuffer(area)
	front.SetString(0, 0, family+"A"+heartEmo+"B"+flagTR+"C", cell.Style{})
	front.SetString(0, 1, eAcute+"D"+conjunct+"E", cell.Style{})
	want := map[byte]int{'A': 2, 'B': 5, 'C': 8, 'D': 1, 'E': 3}
	opts := DiffOptions{ClusterWidths: true}

	for _, path := range []struct {
		name string
		diff func(DiffOptions) ([]byte, error)
	}{
		{"sparse", func(o DiffOptions) ([]byte, error) { return diffSparse(front, NewBuffer(area), nil, o) }},
		{"stream", func(o DiffOptions) ([]byte, error) { return diffFullStream(front, NewBuffer(area), nil, o) }},
		{"inline", func(o DiffOptions) ([]byte, error) { return DiffInline(front, NewBuffer(area), nil, o) }},
	} {
		t.Run(path.name, func(t *testing.T) {
			out, err := path.diff(opts)
			if err != nil {
				t.Fatal(err)
			}
			got := clusterScreen(t, out)
			for letter, col := range want {
				if got[letter] != col {
					t.Errorf("%c landed in column %d, want %d: %q", letter, got[letter], col, out)
				}
			}
			legacy, err := path.diff(DiffOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if len(out) >= len(legacy) {
				t.Errorf("trusting the cursor saved nothing: %d bytes vs %d", len(out), len(legacy))
			}
		})
	}
}

// clusterScreen replays out on a terminal with mode 2027: the cursor advances
// by each grapheme cluster's width, as cell.StringWidth measures it.
func clusterScreen(t *testing.T, out []byte) map[byte]int {
	t.Helper()
	landed := map[byte]int{}
	col := 0
	s := string(out)
	for i := 0; i < len(s); {
		switch {
		case s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[':
			j := i + 2
			for j < len(s) && (s[j] < 0x40 || s[j] > 0x7E) {
				j++
			}
			params, final := s[i+2:j], s[j]
			switch final {
			case 'H':
				col = 0
				if k := strings.IndexByte(params, ';'); k >= 0 {
					col = atoi(params[k+1:]) - 1
				}
			case 'G':
				col = atoi(params) - 1
			}
			i = j + 1
		case s[i] == '\r':
			col, i = 0, i+1
		case s[i] == '\n':
			i++
		default:
			cluster, width, _ := cell.NextCluster(s[i:])
			if c := cluster[0]; len(cluster) == 1 && c >= 'A' && c <= 'Z' {
				landed[c] = col
			}
			col += width
			i += len(cluster)
		}
	}
	return landed
}
