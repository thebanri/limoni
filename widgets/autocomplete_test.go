package widgets

import (
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

var gitCommands = []string{"git status", "git checkout", "git commit", "git cherry-pick", "go test"}

func typeInto(s *AutocompleteState, text string) {
	for _, r := range text {
		ev := driver.KeyEvent{Type: driver.KeyRune, Ch: r}
		if r == ' ' {
			ev = driver.KeyEvent{Type: driver.KeySpace}
		}
		s.HandleKey(ev, gitCommands)
	}
}

func matchTexts(s *AutocompleteState) []string {
	var out []string
	for _, i := range s.Matches() {
		out = append(out, gitCommands[i])
	}
	return out
}

func TestAutocompleteNarrowsAsYouType(t *testing.T) {
	s := &AutocompleteState{}
	typeInto(s, "gc")
	if !s.Open {
		t.Fatal("list closed with matches")
	}
	got := matchTexts(s)
	if len(got) != 3 || got[0] != "git checkout" && got[0] != "git commit" && got[0] != "git cherry-pick" {
		t.Errorf("gc matched %q", got)
	}
	for _, m := range got {
		if m == "go test" || m == "git status" {
			t.Errorf("gc matched %q", m)
		}
	}
	typeInto(s, "o")
	// "co" is consecutive in "commit", so it ranks above "checkout".
	if got := matchTexts(s); len(got) != 2 || got[0] != "git commit" || got[1] != "git checkout" {
		t.Errorf("gco matched %q, want git commit, then git checkout", got)
	}
	typeInto(s, "zzz")
	if s.Open || len(s.Matches()) != 0 {
		t.Errorf("nothing matches, list open=%v matches=%v", s.Open, s.Matches())
	}
}

func TestAutocompleteAcceptAndNavigate(t *testing.T) {
	s := &AutocompleteState{}
	typeInto(s, "gco")
	s.HandleKey(driver.KeyEvent{Type: driver.KeyArrowDown}, gitCommands)
	if !s.HandleKey(driver.KeyEvent{Type: driver.KeyTab}, gitCommands) {
		t.Fatal("Tab not handled")
	}
	if got := s.Input.Value(); got != "git checkout" {
		t.Errorf("accepted %q, want the second match, git checkout", got)
	}
	if s.Open || s.Input.Cursor != len([]rune("git checkout")) {
		t.Errorf("after accepting: open=%v cursor=%d", s.Open, s.Input.Cursor)
	}
	// With the list closed, Enter is the application's (submit).
	if s.HandleKey(driver.KeyEvent{Type: driver.KeyEnter}, gitCommands) {
		t.Error("Enter handled with the list closed")
	}

	// ↑ wraps; Esc closes; ↓ reopens without moving.
	s = &AutocompleteState{}
	typeInto(s, "gc")
	s.HandleKey(driver.KeyEvent{Type: driver.KeyArrowUp}, gitCommands)
	if s.Selected != len(s.Matches())-1 {
		t.Errorf("↑ from the top selected %d", s.Selected)
	}
	s.HandleKey(driver.KeyEvent{Type: driver.KeyEsc}, gitCommands)
	if s.Open {
		t.Error("Esc left the list open")
	}
	s.HandleKey(driver.KeyEvent{Type: driver.KeyArrowDown}, gitCommands)
	if !s.Open || s.Selected != len(s.Matches())-1 {
		t.Errorf("↓ reopened=%v selected=%d", s.Open, s.Selected)
	}
}

func TestAutocompleteDraw(t *testing.T) {
	s := &AutocompleteState{}
	typeInto(s, "gco")
	s.HandleKey(driver.KeyEvent{Type: driver.KeyArrowDown}, gitCommands)
	area := cell.NewRect(0, 0, 16, 4)
	buf := buffer.NewBuffer(area)
	Autocomplete{ID: "cmd", Suggestions: gitCommands, State: s}.Draw(cell.NewContext(area, cell.Style{}), buf)
	lines := strings.Split(buf.Snapshot(), "\n")
	if !strings.HasPrefix(lines[0], "gco") || !strings.HasPrefix(lines[1], " git commit") || !strings.HasPrefix(lines[2], " git checkout") {
		t.Errorf("drawn:\n%s", buf.Snapshot())
	}
	if buf.CellAt(3, 2).Style.Modifier&cell.ModifierReverse == 0 {
		t.Error("the selected suggestion is not highlighted")
	}
	if strings.TrimSpace(lines[3]) != "" {
		t.Errorf("a third row for two matches: %q", lines[3])
	}
}

func TestAutocompleteDoesNotAllocate(t *testing.T) {
	buf, ctx := prepareBenchmarkEnv()
	s := &AutocompleteState{}
	typeInto(s, "g")
	a := Autocomplete{ID: "cmd", Suggestions: gitCommands, State: s}
	a.Draw(ctx, buf)
	if n := testing.AllocsPerRun(50, func() { a.Draw(ctx, buf) }); n != 0 {
		t.Errorf("%.0f allocs per Draw", n)
	}
}

func BenchmarkAutocompleteDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	s := &AutocompleteState{}
	typeInto(s, "g")
	a := Autocomplete{ID: "cmd", Suggestions: gitCommands, State: s}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		a.Draw(ctx, buf)
	}
}
