package main

import (
	"bytes"
	"encoding/base64"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/vt"
	"github.com/thebanri/limoni/backdrop"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
)

func TestMouseReportsAreTakenOnlyWhenTheWrapperHasTheMouse(t *testing.T) {
	in := []byte("a\x1b[<0;10;5M\x1b[<32;12;6M\x1b[<0;12;6m\x1b[<64;1;1Mb")
	var got []mouseEvent
	pass, _ := splitInput(in, nil, true, func(ev inputEvent) {
		if ev.kind == inputMouse {
			got = append(got, ev.mouse)
		}
	})
	if string(pass) != "ab" {
		t.Fatalf("passed %q", pass)
	}
	want := []mouseEvent{{0, 9, 4, false}, {32, 11, 5, false}, {0, 11, 5, true}, {64, 0, 0, false}}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("event %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	if !got[1].motion() || !got[3].wheel() || got[3].which() != 0 {
		t.Error("motion or wheel misread")
	}
	// A program in the shell that has the mouse gets the reports untouched.
	if pass, _ := splitInput(in, nil, false, func(inputEvent) {}); !bytes.Equal(pass, in) {
		t.Fatalf("reports not passed through: %q", pass)
	}
}

// testSession is a session on an emulated 30×6 screen with some output,
// writing what it sends the terminal to a file the test reads.
func testSession(t *testing.T, output string) (*session, *os.File) {
	t.Helper()
	// Never the real clipboard: no display, so copies go out as OSC 52,
	// which the tests read. TestCopyGoesThroughTheSystemClipboard sets up a
	// fake clipboard tool of its own.
	t.Setenv("WAYLAND_DISPLAY", "")
	t.Setenv("DISPLAY", "")
	// No clipboard tool either: these tests are about the text a selection
	// takes, and they read the OSC 52 the wrapper falls back to. A tool in
	// PATH — clip, on Windows — would take that path instead and copy nothing
	// they can see.
	t.Setenv("PATH", t.TempDir())
	out, err := os.CreateTemp(t.TempDir(), "out")
	if err != nil {
		t.Fatal(err)
	}
	// Windows cannot delete a file that is still open, and TempDir's cleanup
	// removes the directory: close it before that runs, as cleanups are LIFO.
	t.Cleanup(func() { _ = out.Close() })
	s := &session{
		w: 30, h: 6, out: out,
		emu:        vt.NewEmulator(30, 6),
		childModes: map[ansi.DECMode]bool{},
		outerMouse: map[ansi.DECMode]bool{},
		opts:       options{selection: true},
	}
	area := cell.NewRect(0, 0, 30, 6)
	s.app, s.bg, s.front = buffer.NewBuffer(area), buffer.NewBuffer(area), buffer.NewBuffer(area)
	s.back = buffer.NewEmptyBuffer()
	s.scene = backdrop.Still(backdrop.Aurora(), 0)
	s.sceneFrame = 0 // the test paints s.bg itself
	_, _ = s.emu.Write([]byte(output))
	s.copyScreen()
	return s, out
}

func sent(t *testing.T, f *os.File) string {
	t.Helper()
	b, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func clipboard(t *testing.T, out string) string {
	t.Helper()
	i := strings.LastIndex(out, "\x1b]52;c;")
	if i < 0 {
		t.Fatalf("nothing copied: %q", out)
	}
	rest := out[i+len("\x1b]52;c;"):]
	b, err := base64.StdEncoding.DecodeString(rest[:strings.IndexByte(rest, '\a')])
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Dragging over the shell's output copies its text — and only its text,
// whatever the scene has drawn in the blank cells around it.
func TestDragCopiesTheShellsTextAndNotTheBackground(t *testing.T) {
	s, out := testSession(t, "$ ls -la\r\ntotal 42\r\ndrwxr-xr-x  src")
	for i := range s.bg.Content {
		s.bg.Content[i] = cell.Cell{Content: '✦', Style: cell.Style{Fg: cell.NewColorRGB(200, 200, 255)}}
	}
	s.mouse(mouseEvent{button: 0, x: 2, y: 0})
	s.mouse(mouseEvent{button: 32, x: 5, y: 2})
	s.mouse(mouseEvent{button: 0, x: 5, y: 2, release: true})
	// Both ends are in: the cell under the pointer when the button went
	// up is selected, as in a terminal's own selection.
	if got, want := clipboard(t, sent(t, out)), "ls -la\ntotal 42\ndrwxr-"; got != want {
		t.Fatalf("copied %q, want %q", got, want)
	}

	// The highlight shows what was copied: selected blank cells are blank,
	// not the scene's stars.
	copy(s.front.Content, s.app.Content)
	terminal.ComposeBackdrop(s.front.Content, s.bg.Content)
	s.highlight()
	if c := s.front.Content[0*30+20]; c.Content != ' ' || c.Style.Bg != selectionColor {
		t.Errorf("selected blank cell: %+v", c)
	}
	if c := s.front.Content[5*30+20]; c.Content != '✦' {
		t.Errorf("unselected blank cell lost the scene: %+v", c)
	}

	// Typing drops the selection.
	s.handleInput(inputMsg{events: []inputEvent{{kind: inputOther}}})
	if s.sel.active {
		t.Error("selection kept after typing")
	}
}

func TestDoubleClickTakesAWordAndTripleClickALine(t *testing.T) {
	s, out := testSession(t, "open ~/src/app/main.go now")
	click := func() {
		s.mouse(mouseEvent{button: 0, x: 9, y: 0})
		s.mouse(mouseEvent{button: 0, x: 9, y: 0, release: true})
	}
	click()
	if strings.Contains(sent(t, out), "\x1b]52") {
		t.Fatal("a single click copied something")
	}
	click()
	if got := clipboard(t, sent(t, out)); got != "~/src/app/main.go" {
		t.Fatalf("double click copied %q", got)
	}
	click()
	if got := clipboard(t, sent(t, out)); got != "open ~/src/app/main.go now" {
		t.Fatalf("triple click copied %q", got)
	}
}

// The terminal's mouse goes to whoever should have it: the wrapper while the
// shell's program has not asked, the program while it has, and back.
func TestTheMouseIsHandedToProgramsThatAskForIt(t *testing.T) {
	s, out := testSession(t, "")
	s.syncMouse()
	if !s.ownMouse.Load() || string(s.pending) != "\x1b[?1002h\x1b[?1006h" {
		t.Fatalf("wrapper's modes: own=%v %q", s.ownMouse.Load(), s.pending)
	}
	s.pending = s.pending[:0]
	// A program asks for the mouse (vim with mouse=a, say, without SGR).
	// The change is written to the terminal at once, before the next report.
	s.setChildMode(ansi.ModeMouseNormal, true)
	if s.ownMouse.Load() {
		t.Fatal("the wrapper kept the mouse from a program that asked for it")
	}
	if p := sent(t, out); !strings.Contains(p, "\x1b[?1000h") || !strings.Contains(p, "\x1b[?1002l") || !strings.Contains(p, "\x1b[?1006l") {
		t.Fatalf("terminal not given the program's modes: %q", p)
	}
	before := len(sent(t, out))
	s.setChildMode(ansi.ModeMouseNormal, false)
	if p := sent(t, out)[before:]; !s.ownMouse.Load() || !strings.Contains(p, "\x1b[?1002h") || !strings.Contains(p, "\x1b[?1000l") {
		t.Fatalf("mouse not taken back: own=%v %q", s.ownMouse.Load(), p)
	}

	// With -select=false the terminal keeps its own selection.
	s2, _ := testSession(t, "")
	s2.opts.selection = false
	s2.syncMouse()
	if s2.ownMouse.Load() || len(s2.pending) != 0 {
		t.Fatalf("-select=false still took the mouse: %q", s2.pending)
	}
}

// Locally, a copy goes through the system's clipboard tool — every
// terminal, any length — onto the clipboard and the primary selection; over
// SSH it goes to the terminal as OSC 52.
func TestCopyGoesThroughTheSystemClipboard(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake clipboard tool is a POSIX shell script; clip itself is exercised on Windows by hand")
	}
	dir := t.TempDir()
	log := dir + "/calls"
	tool := "#!/bin/sh\nprintf '%s|' \"$*\" >> '" + log + "'\n/bin/cat >> '" + log + "'\necho >> '" + log + "'\n"
	if err := os.WriteFile(dir+"/wl-copy", []byte(tool), 0o755); err != nil {
		t.Fatal(err)
	}
	s, out := testSession(t, "hello there")
	t.Setenv("PATH", dir)
	t.Setenv("WAYLAND_DISPLAY", "wayland-test")
	t.Setenv("SSH_TTY", "")
	t.Setenv("SSH_CONNECTION", "")
	s.mouse(mouseEvent{button: 0, x: 0, y: 0})
	s.mouse(mouseEvent{button: 32, x: 4, y: 0})
	s.mouse(mouseEvent{button: 0, x: 4, y: 0, release: true})
	calls, _ := os.ReadFile(log)
	if string(calls) != "|hello\n--primary|hello\n" {
		t.Fatalf("clipboard tool called as %q", calls)
	}
	if strings.Contains(sent(t, out), "\x1b]52") {
		t.Fatal("OSC 52 sent as well")
	}

	// Over SSH the tool on this machine is not the user's clipboard.
	t.Setenv("SSH_TTY", "/dev/pts/9")
	s.mouse(mouseEvent{button: 0, x: 6, y: 0})
	s.mouse(mouseEvent{button: 32, x: 10, y: 0})
	s.mouse(mouseEvent{button: 0, x: 10, y: 0, release: true})
	if got := clipboard(t, sent(t, out)); got != "there" {
		t.Fatalf("over SSH copied %q", got)
	}
}

// A socket path longer than the kernel allows made the wrapper give up its
// socket without a word, so settings changes stopped reaching it.
func TestControlSocketFitsWhateverTheRuntimeDirectory(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", "/"+strings.Repeat("deep/", 30))
	if p := controlDir() + "/4194304.sock"; len(p) > 104 {
		t.Fatalf("socket path %d bytes: %s", len(p), p)
	}
	t.Setenv("XDG_RUNTIME_DIR", "/run/user/1000")
	if got, want := controlDir(), filepath.Join("/run/user/1000", "limoni-backdrop"); got != want {
		t.Fatalf("a short runtime directory was not used: %s", got)
	}
}
