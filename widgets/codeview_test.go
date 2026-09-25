package widgets

import (
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

// kinds renders a line's spans as "text:kind" pairs, whitespace left out.
func kinds(l codeLine) string {
	names := [...]string{"plain", "kw", "type", "str", "num", "comment", "func", "punct"}
	var out []string
	for _, sp := range l.spans {
		text := l.text[sp.start:sp.end]
		if strings.TrimSpace(text) == "" {
			continue
		}
		out = append(out, strings.TrimSpace(text)+":"+names[sp.kind])
	}
	return strings.Join(out, " ")
}

func TestLexGo(t *testing.T) {
	src := "func main() {\n" +
		"\tx := 42 // answer\n" +
		"\ts := \"a \\\"quoted\\\" word\"\n" +
		"\t/* a comment\n" +
		"\t   over lines */ return nil\n" +
		"\tr := `raw\n" +
		"still raw` + len(s)\n" +
		"}\n"
	s := NewCodeViewState(src, LanguageGo)
	want := []string{
		"func:kw main:func ():punct {:punct", // neighbouring punctuation is one span
		"x:plain :=:punct 42:num // answer:comment",
		`s:plain :=:punct "a \"quoted\" word":str`,
		"/* a comment:comment",
		"over lines */:comment return:kw nil:type",
		"r:plain :=:punct `raw:str",
		"still raw`:str +:punct len:type (:punct s:plain ):punct",
		"}:punct",
	}
	if s.Lines() != len(want) {
		t.Fatalf("%d lines, want %d (a trailing newline is not a line)", s.Lines(), len(want))
	}
	for i, w := range want {
		if got := kinds(s.lines[i]); got != w {
			t.Errorf("line %d:\n got %s\nwant %s", i+1, got, w)
		}
	}
	// The tab became four spaces.
	if !strings.HasPrefix(s.lines[1].text, "    x") {
		t.Errorf("tab not expanded: %q", s.lines[1].text)
	}
}

func TestLexPythonTripleQuotes(t *testing.T) {
	s := NewCodeViewState("def f():\n    '''doc\n    more'''\n    return True # yes\n", LanguagePython)
	for i, w := range []string{
		"def:kw f:func ()::punct",
		"'''doc:str",
		"more''':str",
		"return:kw True:type # yes:comment",
	} {
		if got := kinds(s.lines[i]); got != w {
			t.Errorf("line %d:\n got %s\nwant %s", i+1, got, w)
		}
	}
}

func TestLexRustNestedBlockComments(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "same line",
			src:  `/* outer /* inner /* deeper */ inner */ still comment */ let visible = true`,
			want: []string{"/* outer /* inner /* deeper */ inner */ still comment */:comment let:kw visible:plain =:punct true:type"},
		},
		{
			name: "across lines",
			src:  "/* outer\n/* inner */\nstill comment */ let visible = true",
			want: []string{"/* outer:comment", "/* inner */:comment", "still comment */:comment let:kw visible:plain =:punct true:type"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := NewCodeViewState(tc.src, LanguageRust)
			if s.Lines() != len(tc.want) {
				t.Fatalf("got %d lines, want %d", s.Lines(), len(tc.want))
			}
			for i, want := range tc.want {
				if got := kinds(s.lines[i]); got != want {
					t.Errorf("line %d:\n got %s\nwant %s", i+1, got, want)
				}
			}
		})
	}
}

func TestLanguageForFile(t *testing.T) {
	for name, want := range map[string]*Language{"main.go": LanguageGo, "a.TSX": LanguageJavaScript, "x.rs": LanguageRust, "ci.yml": LanguageYAML, "README": nil} {
		if got := LanguageForFile(name); got != want {
			t.Errorf("%s: %v", name, got)
		}
	}
}

func TestCodeViewDraw(t *testing.T) {
	s := NewCodeViewState("package main\n\nfunc 👋() {}\n", LanguageGo)
	area := cell.NewRect(0, 0, 14, 4)
	buf := buffer.NewBuffer(area)
	CodeView{State: s}.Draw(cell.NewContext(area, cell.Style{}), buf)
	lines := strings.Split(buf.Snapshot(), "\n")
	for i, w := range []string{"1 package main", "2", "3 func 👋 () {}", ""} {
		if got := strings.TrimRight(lines[i], " "); got != w {
			t.Errorf("row %d: %q, want %q", i, got, w)
		}
	}
	if got := buf.CellAt(2, 0).Style.Fg; got != DefaultCodeTheme[TokenKeyword].Fg {
		t.Errorf("'package' drawn in %v, not the keyword colour", got)
	}
	if got := buf.CellAt(0, 0).Style.Fg; got == DefaultCodeTheme[TokenKeyword].Fg {
		t.Error("the gutter took the keyword colour")
	}

	// Scrolled one column into the emoji: its visible half stays blank and
	// what follows keeps its column.
	s.HOffset = 6
	CodeView{State: s, HideLineNumbers: true}.Draw(cell.NewContext(area, cell.Style{}), buf)
	if got := strings.TrimRight(strings.Split(buf.Snapshot(), "\n")[2], " "); got != " () {}" {
		t.Errorf("scrolled row %q", got)
	}
}

func TestCodeViewCursorScrolls(t *testing.T) {
	var src strings.Builder
	for i := 0; i < 50; i++ {
		src.WriteString("x := 1\n")
	}
	s := NewCodeViewState(src.String(), LanguageGo)
	s.Cursor = 0
	area := cell.NewRect(0, 0, 20, 10)
	buf := buffer.NewBuffer(area)
	cv := CodeView{State: s}
	cv.Draw(cell.NewContext(area, cell.Style{}), buf)
	s.HandleKey(driver.KeyEvent{Type: driver.KeyPageDown})
	s.HandleKey(driver.KeyEvent{Type: driver.KeyPageDown})
	cv.Draw(cell.NewContext(area, cell.Style{}), buf)
	if s.Cursor != 18 || s.Offset != 9 {
		t.Errorf("after two pages: cursor %d offset %d, want 18 and 9", s.Cursor, s.Offset)
	}
	s.HandleKey(driver.KeyEvent{Type: driver.KeyEnd})
	cv.Draw(cell.NewContext(area, cell.Style{}), buf)
	if s.Cursor != 49 || s.Offset != 40 {
		t.Errorf("End: cursor %d offset %d", s.Cursor, s.Offset)
	}
}

func TestCodeViewDoesNotAllocate(t *testing.T) {
	buf, ctx := prepareBenchmarkEnv()
	cv := CodeView{State: NewCodeViewState(strings.Repeat("func f(x int) string { return \"héllo\" } // 👋\n", 40), LanguageGo)}
	cv.Draw(ctx, buf)
	if n := testing.AllocsPerRun(50, func() { cv.Draw(ctx, buf) }); n != 0 {
		t.Errorf("%.0f allocs per Draw", n)
	}
}

func BenchmarkCodeViewDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	cv := CodeView{State: NewCodeViewState(strings.Repeat("func f(x int) string { return \"hello\" } // comment\n", 40), LanguageGo)}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cv.Draw(ctx, buf)
	}
}
