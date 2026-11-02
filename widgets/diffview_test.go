package widgets

import (
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

func drawDiff(t *testing.T, dv DiffView, w uint16) (*buffer.Buffer, []string) {
	t.Helper()
	_, h := dv.SizeHint(cell.NewRect(0, 0, w, 100))
	area := cell.NewRect(0, 0, w, h)
	buf := buffer.NewBuffer(area)
	dv.Draw(cell.NewContext(area, cell.Style{}), buf)
	rows := strings.Split(buf.Snapshot(), "\n")
	for i := range rows {
		rows[i] = strings.TrimRight(rows[i], " ")
	}
	return buf, rows
}

func TestDiffViewShowsTheEdit(t *testing.T) {
	oldSrc := "a\nb\nc\nd\ne\nf\ng\nh\ni\nj\n"
	newSrc := "a\nb\nc\nD\ne\nf\ng\nh\ni\nj\nk\n"
	s := NewDiffViewState(oldSrc, newSrc, nil, 1)
	if a, r := s.Stats(); a != 2 || r != 1 {
		t.Errorf("Stats = +%d -%d, want +2 -1", a, r)
	}
	buf, rows := drawDiff(t, DiffView{State: s}, 12)
	want := []string{
		" 3   c",
		" 4 - d",
		" 4 + D",
		" 5   e",
		"⋯",
		"10   j",
		"11 + k",
	}
	if strings.Join(rows, "\n") != strings.Join(want, "\n") {
		t.Errorf("rows\n%s\nwant\n%s", strings.Join(rows, "\n"), strings.Join(want, "\n"))
	}
	// Removed and added rows carry their backgrounds across the width.
	if bg := buf.Get(11, 1).Style.Bg; bg != cell.NewColorRGB(66, 30, 32) { // row " 4 - d"
		t.Errorf("removed row background %v", bg)
	}
	if bg := buf.Get(11, 2).Style.Bg; bg != cell.NewColorRGB(28, 56, 34) {
		t.Errorf("added row background %v", bg)
	}
	if bg := buf.Get(11, 0).Style.Bg; bg.Type() != cell.ColorDefault {
		t.Errorf("context row background %v", bg)
	}
}

// Applying the script to the old lines gives the new ones, for edits of
// every shape: the diff is a correct one, whichever of the equal-length
// scripts it picks.
func TestDiffOpsRebuildTheNewSource(t *testing.T) {
	cases := [][2]string{
		{"", "x\ny\n"},
		{"x\ny\n", ""},
		{"a\nb\nc\n", "a\nb\nc\n"},
		{"a\nb\nc\n", "c\nb\na\n"},
		{"a\nx\nb\nx\nc\n", "x\na\nb\nc\nx\n"},
		{"1\n2\n3\n4\n5\n6\n", "1\n3\n4\nnew\n6\n7\n"},
	}
	for _, c := range cases {
		a, b := highlightLines(c[0], nil), highlightLines(c[1], nil)
		ops := diffOps(a, b)
		var rebuilt, kept []string
		i, j := 0, 0
		for _, op := range ops {
			switch op {
			case ' ':
				if a[i].text != b[j].text {
					t.Errorf("%q→%q: kept %q against %q", c[0], c[1], a[i].text, b[j].text)
				}
				rebuilt, kept = append(rebuilt, a[i].text), append(kept, a[i].text)
				i++
				j++
			case '-':
				i++
			case '+':
				rebuilt = append(rebuilt, b[j].text)
				j++
			}
		}
		if i != len(a) || j != len(b) {
			t.Errorf("%q→%q: script used %d/%d old and %d/%d new lines", c[0], c[1], i, len(a), j, len(b))
		}
		var want []string
		for _, l := range b {
			want = append(want, l.text)
		}
		if strings.Join(rebuilt, "\n") != strings.Join(want, "\n") {
			t.Errorf("%q→%q rebuilt %q", c[0], c[1], rebuilt)
		}
	}
	// Minimal on a case with one right answer: one line changed among many.
	ops := diffOps(highlightLines("1\n2\n3\n4\n5\n", nil), highlightLines("1\n2\nthree\n4\n5\n", nil))
	if string(ops) != "  -+  " {
		t.Errorf("ops %q", ops)
	}
}

func TestDiffViewWholeFileAndHighlighting(t *testing.T) {
	s := NewDiffViewState("x := 1\n", "x := 1\n// note\n", LanguageGo, -1)
	buf, rows := drawDiff(t, DiffView{State: s, HideLineNumbers: true}, 20)
	if want := []string{"  x := 1", "+ // note"}; strings.Join(rows, "\n") != strings.Join(want, "\n") {
		t.Errorf("rows %q, want %q", rows, want)
	}
	if fg := buf.Get(2, 1).Style.Fg; fg != DefaultCodeTheme[TokenComment].Fg {
		t.Errorf("comment not highlighted: %v", fg)
	}
}

func TestDiffViewDrawDoesNotAllocate(t *testing.T) {
	s := NewDiffViewState(strings.Repeat("func f() {}\n", 50), strings.Repeat("func g() {}\n", 50), LanguageGo, 3)
	area := cell.NewRect(0, 0, 80, 24)
	buf := buffer.NewBuffer(area)
	dv := DiffView{State: s}
	if n := testing.AllocsPerRun(50, func() { dv.Draw(cell.NewContext(area, cell.Style{}), buf) }); n != 0 {
		t.Errorf("Draw allocates %v times", n)
	}
}

func BenchmarkDiffViewDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	old := strings.Repeat("func f(x int) string { return \"hello\" } // comment\n", 40)
	dv := DiffView{State: NewDiffViewState(old, strings.Replace(old, "hello", "bye", 3), LanguageGo, 3)}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dv.Draw(ctx, buf)
	}
}

// applyOps checks that ops rebuilds b from a and returns the number of
// changed rows.
func applyOps(t *testing.T, a, b []codeLine, ops []byte) int {
	t.Helper()
	i, j, changes := 0, 0, 0
	for _, op := range ops {
		switch op {
		case ' ':
			if i >= len(a) || j >= len(b) || a[i].text != b[j].text {
				t.Fatalf("bad keep at %d/%d", i, j)
			}
			i++
			j++
		case '-':
			i++
			changes++
		case '+':
			j++
			changes++
		}
	}
	if i != len(a) || j != len(b) {
		t.Fatalf("script used %d/%d old and %d/%d new lines", i, len(a), j, len(b))
	}
	return changes
}

func linesOf(s []string) []codeLine {
	l := make([]codeLine, len(s))
	for i, x := range s {
		l[i].text = x
	}
	return l
}

func bruteLCS(a, b []codeLine) int {
	t := make([][]int, len(a)+1)
	for i := range t {
		t[i] = make([]int, len(b)+1)
	}
	for i := len(a) - 1; i >= 0; i-- {
		for j := len(b) - 1; j >= 0; j-- {
			if a[i].text == b[j].text {
				t[i][j] = t[i+1][j+1] + 1
			} else {
				t[i][j] = max(t[i+1][j], t[i][j+1])
			}
		}
	}
	return t[0][0]
}

// Random edits of small files: the script is valid, minimal (against a
// brute-force LCS) and puts removals before additions in every change.
func TestDiffOpsAreMinimal(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	for n := 0; n < 3000; n++ {
		alpha := 2 + rng.Intn(6)
		mk := func(k int) []string {
			s := make([]string, k)
			for i := range s {
				s[i] = string(rune('a' + rng.Intn(alpha)))
			}
			return s
		}
		a := linesOf(mk(rng.Intn(25)))
		b := linesOf(mk(rng.Intn(25)))
		ops := diffOps(a, b)
		got := applyOps(t, a, b, ops)
		if want := len(a) + len(b) - 2*bruteLCS(a, b); got != want {
			t.Fatalf("%d changes, minimum %d: %q → %q ops %q", got, want, a, b, ops)
		}
		for i := 1; i < len(ops); i++ {
			if ops[i-1] == '+' && ops[i] == '-' {
				t.Fatalf("addition before removal in %q", ops)
			}
		}
	}
}

// A large file with scattered edits gets a minimal script, quickly.
func TestDiffOpsLargeFile(t *testing.T) {
	var as, bs []string
	edits := 0
	for i := 0; i < 20000; i++ {
		line := "line " + strconv.Itoa(i)
		as = append(as, line)
		switch {
		case i%500 == 100:
			bs = append(bs, line, "inserted "+strconv.Itoa(i))
			edits++
		case i%500 == 300:
			edits++ // deleted
		case i%500 == 400:
			bs = append(bs, "changed "+strconv.Itoa(i))
			edits += 2
		default:
			bs = append(bs, line)
		}
	}
	a, b := linesOf(as), linesOf(bs)
	start := time.Now()
	ops := diffOps(a, b)
	t.Logf("20000 lines, %d changes: %v", edits, time.Since(start))
	if got := applyOps(t, a, b, ops); got != edits {
		t.Errorf("%d changes, want %d", got, edits)
	}
	// A rewrite of every 20th line: D is 2000, still minimal and fast.
	var cs []string
	for i, l := range as {
		if i%20 == 0 {
			l = "new " + l
		}
		cs = append(cs, l)
	}
	start = time.Now()
	ops = diffOps(a, linesOf(cs))
	t.Logf("rewrite of every 20th line: %v", time.Since(start))
	if got := applyOps(t, a, linesOf(cs), ops); got != 2000 {
		t.Errorf("%d changes, want 2000", got)
	}
}

// Unrelated files hit the work budget and fall back to removed-then-added,
// still a valid script.
func TestDiffOpsBudgetFallback(t *testing.T) {
	var as, bs []string
	for i := 0; i < 60000; i++ {
		as = append(as, "a"+strconv.Itoa(i))
		bs = append(bs, "b"+strconv.Itoa(i))
	}
	a, b := linesOf(as), linesOf(bs)
	start := time.Now()
	ops := diffOps(a, b)
	t.Logf("unrelated 60000 lines: %v", time.Since(start))
	applyOps(t, a, b, ops)
}
