package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/thebanri/limoni/core/cell"
)

func TestSplitInputKeepsBackWhatTheWrapperHandles(t *testing.T) {
	in := []byte("ls\x1b[5;2~\x1b[A\x1b[I\x1b[6;2~x\x1b[O")
	var kinds []inputKind
	pass, _ := splitInput(in, nil, false, func(ev inputEvent) { kinds = append(kinds, ev.kind) })
	if want := []byte("ls\x1b[Ax"); !bytes.Equal(pass, want) {
		t.Fatalf("passed %q, want %q", pass, want)
	}
	want := []inputKind{inputOther, inputScrollUp, inputOther, inputOther, inputFocusIn, inputScrollDown, inputOther, inputFocusOut}
	if !reflect.DeepEqual(kinds, want) {
		t.Fatalf("events %v, want %v", kinds, want)
	}
}

func TestTerminalColorAnswersAreParsedAndRemoved(t *testing.T) {
	in := []byte("a\x1b]10;rgb:ffff/8080/0000\x1b\\b\x1b]11;rgb:12/34/56\x07\x1b[?62;22cc")
	fg, rest := takeColor(in, "10")
	bg, rest := takeColor(rest, "11")
	if fg != cell.NewColorRGB(255, 128, 0) {
		t.Errorf("fg = %06x", uint32(fg))
	}
	if bg != cell.NewColorRGB(0x12, 0x34, 0x56) {
		t.Errorf("bg = %06x", uint32(bg))
	}
	end := da1End(rest)
	if end < 0 || rest[end] != 'c' || !bytes.HasPrefix(rest, []byte("ab\x1b[?62;22")) {
		t.Fatalf("DA1 not found in %q (end %d)", rest, end)
	}
	if da1End([]byte("\x1b[?62;2")) >= 0 {
		t.Fatal("a cut-off DA1 answer was taken as whole")
	}
}

// The shell's colours are kept as it asked for them, and a cell it left
// without a background stays without one, which is what lets the scene show.
func TestConvertCell(t *testing.T) {
	c := convertCell(&uv.Cell{Content: "x", Width: 1, Style: uv.Style{
		Fg: ansi.BasicColor(2), Attrs: uv.AttrBold, Underline: uv.UnderlineCurly,
	}})
	want := cell.Cell{Content: 'x', Style: cell.Style{
		Fg: cell.NewColorANSI(2), Modifier: cell.ModifierBold | cell.ModifierUndercurl,
	}}
	if c != want {
		t.Fatalf("got %+v, want %+v", c, want)
	}
	if c := convertCell(&uv.Cell{Content: " ", Width: 1, Style: uv.Style{Bg: ansi.IndexedColor(236)}}); c.Style.Bg != cell.NewColorANSI(236) {
		t.Fatalf("background lost: %+v", c)
	}
	if c := convertCell(&uv.Cell{Width: 0}); c.Content != cell.RuneContinuation {
		t.Fatalf("second column of a wide character: %+v", c)
	}
	if c := convertCell(nil); c != (cell.Cell{Content: ' '}) {
		t.Fatalf("missing cell: %+v", c)
	}
}

// The example art that ships next to the program, and that the guide
// points to, must load and choose as a background.
func TestExampleArtLoads(t *testing.T) {
	files, _ := filepath.Glob("art/*.txt")
	if len(files) == 0 {
		t.Fatal("no example art")
	}
	for _, f := range files {
		if _, err := chooseScene("", "", f, "", false); err != nil {
			t.Errorf("%s: %v", f, err)
		}
	}
	if _, err := chooseScene("", "", "art/missing.txt", "", false); err == nil {
		t.Error("a missing file was accepted")
	}
	if bd, _ := chooseScene("aurora", "", "art/cat.txt", "", true); bd.Interval() != 0 {
		t.Error("-still did not freeze animated art")
	}
}

// The terminal's answers to the wrapper's colour queries can arrive after
// the wait at start-up, while the window is still opening. They must never
// reach the shell as typing — they did, as stray hex letters on the command
// line — and a colour in one is still put to use.
func TestLateTerminalRepliesNeverReachTheShell(t *testing.T) {
	var colors []inputEvent
	emit := func(ev inputEvent) {
		if ev.kind == inputColor {
			colors = append(colors, ev)
		}
	}
	in := []byte("ls\x1b]10;rgb:dddd/dddd/dddd\x1b\\\x1b]11;rgb:1e1e/1e1e/2e2e\x07\x1b[?62;22cx")
	pass, carry := splitInput(in, nil, false, emit)
	if string(pass) != "lsx" || carry != nil {
		t.Fatalf("passed %q, carry %q", pass, carry)
	}
	if len(colors) != 2 || colors[1].osc != "11" || colors[1].color != cell.NewColorRGB(0x1e, 0x1e, 0x2e) {
		t.Fatalf("colours %+v", colors)
	}

	// Cut across two reads: nothing leaks from either half. A read that
	// ends on "ESC" or "ESC ]" cannot be told from the Escape key or Alt+],
	// which must not wait; the terminal writes a reply whole, so those two
	// cuts are left out.
	whole := []byte("\x1b]11;rgb:1e1e/1e1e/2e2e\x1b\\a")
	for cut := 3; cut < len(whole)-1; cut++ {
		p1, c1 := splitInput(whole[:cut], nil, false, func(inputEvent) {})
		p2, c2 := splitInput(append(c1, whole[cut:]...), p1, false, func(inputEvent) {})
		if string(p2) != "a" || c2 != nil {
			t.Fatalf("cut at %d: passed %q, carry %q", cut, p2, c2)
		}
	}

	// Keys that begin the same way still arrive: Alt+] is ESC ] alone.
	if pass, carry := splitInput([]byte("\x1b]"), nil, false, func(inputEvent) {}); string(pass) != "\x1b]" || carry != nil {
		t.Fatalf("Alt+] became %q, carry %q", pass, carry)
	}
}

func TestNestedFlagsSayWhatToRun(t *testing.T) {
	fs := flag.NewFlagSet("x", flag.ContinueOnError)
	fs.Float64("opacity", 0.45, "")
	fs.Float64("fps", 0, "")
	if err := fs.Parse([]string{"-opacity", "0.3"}); err != nil {
		t.Fatal(err)
	}
	var set []*flag.Flag
	fs.Visit(func(f *flag.Flag) { set = append(set, f) })
	if got := nestedHint(set); !strings.Contains(got, "backdrop-shell opacity 0.3\n") {
		t.Fatalf("-opacity alone: %q", got)
	}
	_ = fs.Parse([]string{"-opacity", "0.3", "-fps", "30"})
	set = set[:0]
	fs.Visit(func(f *flag.Flag) { set = append(set, f) })
	if got := nestedHint(set); !strings.Contains(got, "backdrop-shell enable -fps 30 -opacity 0.3\n") {
		t.Fatalf("two flags: %q", got)
	}
}

func TestLocalScenesAreFoundByName(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	dir := filepath.Join(cfg, "limoni", "scenes")
	write := func(name string, mode os.FileMode) {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("#!/bin/sh\n"), mode); err != nil {
			t.Fatal(err)
		}
	}
	// What counts as a program is the platform's: an extension Windows starts,
	// a permission bit elsewhere. The scene's name is the file's without it.
	prog := func(base string) string {
		if runtime.GOOS == "windows" {
			return base + ".cmd"
		}
		return base
	}
	write(prog("fireflies"), 0o755)
	write(filepath.Join("cosmos", prog("cosmos")), 0o755)
	write(filepath.Join("cosmos", "main.go"), 0o644)
	write("notes.txt", 0o644)
	got := localScenes()
	want := map[string]string{"fireflies": filepath.Join(dir, prog("fireflies")), "cosmos": filepath.Join(dir, "cosmos", prog("cosmos"))}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("localScenes = %v, want %v", got, want)
	}
	if names := strings.Join(sceneNames(), ","); !strings.HasSuffix(names, ",cosmos,fireflies") {
		t.Fatalf("sceneNames = %s", names)
	}
	bd, err := chooseScene("cosmos", "", "", "", false)
	if c, ok := bd.(*cmdScene); err != nil || !ok || c.path != want["cosmos"] {
		t.Fatalf("chooseScene(cosmos) = %#v, %v", bd, err)
	}
	if _, err := chooseScene("notes", "", "", "", false); err == nil || !strings.Contains(err.Error(), "fireflies") {
		t.Fatalf("an unknown scene: %v", err)
	}
}
