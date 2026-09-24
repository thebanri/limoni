package widgets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

// pickerTree makes:
//
//	root/
//	  .hidden
//	  b.md        (1536 bytes)
//	  A.go        (12 bytes)
//	  zeta/
//	    inner.txt
//	  Alpha/
func pickerTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(name string, size int) {
		if err := os.WriteFile(filepath.Join(root, name), make([]byte, size), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".hidden", 1)
	write("b.md", 1536)
	write("A.go", 12)
	for _, d := range []string{"zeta", "Alpha"} {
		if err := os.Mkdir(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join("zeta", "inner.txt"), 3)
	return root
}

func entryNames(s *FilePickerState) string {
	var names []string
	for _, e := range s.Entries {
		names = append(names, e.Name)
	}
	return strings.Join(names, " ")
}

func TestFilePickerListing(t *testing.T) {
	root := pickerTree(t)
	s := NewFilePickerState(root)
	if s.Err != nil {
		t.Fatal(s.Err)
	}
	// ".." first, then directories, then files, case-insensitively by name.
	if got := entryNames(s); got != ".. Alpha zeta A.go b.md" {
		t.Errorf("entries %q", got)
	}
	s.HandleKey(driver.KeyEvent{Type: driver.KeyRune, Ch: 'h', Ctrl: true})
	if got := entryNames(s); got != ".. Alpha zeta .hidden A.go b.md" {
		t.Errorf("with hidden files %q", got)
	}

	s = &FilePickerState{Extensions: []string{".GO"}}
	s.Load(root)
	if got := entryNames(s); got != ".. Alpha zeta A.go" {
		t.Errorf("only .go files %q", got)
	}
	s = &FilePickerState{DirsOnly: true}
	s.Load(root)
	if got := entryNames(s); got != ".. Alpha zeta" {
		t.Errorf("directories only %q", got)
	}

	if err := s.Load(filepath.Join(root, "missing")); err == nil || s.Err == nil || s.Dir != root {
		t.Errorf("a missing directory: err=%v, and the old listing should stay (dir %q)", err, s.Dir)
	}
}

func TestFilePickerNavigation(t *testing.T) {
	root := pickerTree(t)
	s := NewFilePickerState(root)
	key := func(k driver.KeyType) { s.HandleKey(driver.KeyEvent{Type: k}) }

	key(driver.KeyArrowDown)
	key(driver.KeyArrowDown) // zeta
	key(driver.KeyEnter)
	if s.Dir != filepath.Join(root, "zeta") || entryNames(s) != ".. inner.txt" {
		t.Fatalf("entered %q: %q", s.Dir, entryNames(s))
	}
	key(driver.KeyArrowDown)
	key(driver.KeyEnter)
	if want := filepath.Join(root, "zeta", "inner.txt"); s.Chosen != want {
		t.Errorf("chosen %q, want %q", s.Chosen, want)
	}
	// Going up puts the cursor back on the directory just left.
	key(driver.KeyBackspace)
	if e, _ := s.SelectedEntry(); s.Dir != root || e.Name != "zeta" {
		t.Errorf("after going up: dir %q, selected %q", s.Dir, e.Name)
	}
	key(driver.KeyEnd)
	if e, _ := s.SelectedEntry(); e.Name != "b.md" {
		t.Errorf("End selected %q", e.Name)
	}
	if s.HandleKey(driver.KeyEvent{Type: driver.KeyArrowRight}) {
		t.Error("→ on a file did something")
	}
}

func TestFilePickerDraw(t *testing.T) {
	root := pickerTree(t)
	s := NewFilePickerState(root)
	s.Selected = 4 // b.md
	area := cell.NewRect(0, 0, 20, 4)
	buf := buffer.NewBuffer(area)
	FilePicker{State: s}.Draw(cell.NewContext(area, cell.Style{}), buf)
	lines := strings.Split(buf.Snapshot(), "\n")

	// The path is cut from the left, keeping its end.
	if !strings.HasPrefix(lines[0], "…") || !strings.HasSuffix(root, strings.TrimPrefix(strings.TrimRight(lines[0], " "), "…")) {
		t.Errorf("header %q for %q", lines[0], root)
	}
	// Three rows for five entries, scrolled so the selection shows.
	if lines[1] != "zeta/               " || lines[2] != "A.go             12B" || lines[3] != "b.md            1.5K" {
		t.Errorf("rows:\n%s", strings.Join(lines[1:], "\n"))
	}
	if buf.CellAt(0, 3).Style.Modifier&cell.ModifierReverse == 0 {
		t.Error("selected row not highlighted")
	}
}

func TestFilePickerClick(t *testing.T) {
	root := pickerTree(t)
	s := NewFilePickerState(root)
	var handler func(driver.MouseEvent)
	area := cell.NewRect(0, 0, 20, 8)
	ctx := cell.NewContext(area, cell.Style{})
	ctx.RegisterMouse = func(_ cell.Rect, h func(driver.MouseEvent)) { handler = h }
	FilePicker{State: s}.Draw(ctx, buffer.NewBuffer(area))

	handler(driver.MouseEvent{Button: driver.MouseLeft, Y: 3}) // header, "..", Alpha, zeta
	if e, _ := s.SelectedEntry(); e.Name != "zeta" {
		t.Fatalf("click selected %q", e.Name)
	}
	handler(driver.MouseEvent{Button: driver.MouseLeft, Y: 3})
	if s.Dir != filepath.Join(root, "zeta") {
		t.Errorf("a second click did not open zeta: %q", s.Dir)
	}
}

func TestAppendHumanSize(t *testing.T) {
	for n, want := range map[int64]string{0: "0B", 999: "999B", 1000: "1.0K", 1536: "1.5K", 20 << 10: "20K", 3 << 30: "3.0G", 1 << 62: "4.0E"} {
		if got := string(appendHumanSize(nil, n)); got != want {
			t.Errorf("%d: %q, want %q", n, got, want)
		}
	}
}

func TestFilePickerDoesNotAllocate(t *testing.T) {
	buf, ctx := prepareBenchmarkEnv()
	ctx.RegisterMouse = func(cell.Rect, func(driver.MouseEvent)) {}
	fp := FilePicker{State: NewFilePickerState(pickerTree(t))}
	fp.Draw(ctx, buf)
	if n := testing.AllocsPerRun(50, func() { fp.Draw(ctx, buf) }); n != 0 {
		t.Errorf("%.0f allocs per Draw", n)
	}
}

func BenchmarkFilePickerDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	fp := FilePicker{State: NewFilePickerState(".")}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		fp.Draw(ctx, buf)
	}
}
