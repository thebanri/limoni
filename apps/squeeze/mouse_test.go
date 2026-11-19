package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/vt"
	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/testkit"
)

// Claude Code titles its window "✳ …"; ✳ is E2 9C B3, and x/vt takes the
// 0x9C for ST, printing the rest of the title onto the screen, where the
// agent believes the cells are blank and types over them.
func TestAUTF8TitleStaysOffTheScreen(t *testing.T) {
	e := vt.NewSafeEmulator(30, 2)
	var r racer
	_, _ = e.Write(r.scan([]byte("\x1b]0;✳ Greeting exchange\x07ab")))
	got := ""
	for x := range 30 {
		if c := e.CellAt(x, 0); c != nil {
			got += c.Content
		}
	}
	if strings.TrimSpace(got) != "ab" {
		t.Errorf("screen = %q, want only ab", got)
	}
}

// Output that only sets modes, as agy sends every few seconds while it
// waits, is not the agent working.
func TestModesAloneAreNotWork(t *testing.T) {
	defer func(q time.Duration) { quiet = q }(quiet)
	quiet = time.Second
	r := fake(t, `printf 'hi'; while :; do printf '\033[?2004h\033[>4;2m'; sleep .2; done`)
	time.Sleep(1500 * time.Millisecond)
	if got := r.state(time.Now()); got != waiting {
		t.Errorf("state = %v, want waiting", got)
	}
}

func TestMouseModesAreFollowed(t *testing.T) {
	var r racer
	for _, c := range []struct {
		in   string
		want bool
	}{
		{"\x1b[?1000;1006h", true},
		{"\x1b[?1006l", true}, // the encoding alone
		{"\x1b[?1003h\x1b[?1000l", true},
		{"\x1b[?1003l", false},
		{"\x1b[1000h", false}, // not a DEC mode
	} {
		r.scan([]byte(c.in))
		if got := r.mouse.Load() != 0; got != c.want {
			t.Errorf("after %q mouse = %v, want %v", c.in, got, c.want)
		}
	}
}

// A click and the wheel reach an agent that asked for the mouse as SGR
// reports, at its own coordinates.
func TestMouseReachesTheAgent(t *testing.T) {
	e := vt.NewSafeEmulator(20, 5)
	_, _ = e.Write([]byte("\x1b[?1000h\x1b[?1006h"))
	got := make(chan string, 1)
	go func() {
		b := make([]byte, 64)
		n, _ := e.Read(b)
		got <- string(b[:n])
	}()
	e.SendMouse(toEmulatorMouse(driver.MouseEvent{Button: driver.MouseLeft}, 2, 1))
	if s := <-got; s != "\x1b[<0;3;2M" {
		t.Errorf("click = %q", s)
	}
	go func() {
		b := make([]byte, 64)
		n, _ := e.Read(b)
		got <- string(b[:n])
	}()
	e.SendMouse(toEmulatorMouse(driver.MouseEvent{Button: driver.MouseScrollUp}, 0, 0))
	if s := <-got; s != "\x1b[<64;1;1M" {
		t.Errorf("wheel = %q", s)
	}
}

// On the race screen a click on the list picks a racer and takes the keys
// back, a click on the pane types into it, and the wheel over an agent that
// did not ask for the mouse scrolls back through what it printed.
func TestTheMouseOnTheRaceScreen(t *testing.T) {
	u := newUI(t.TempDir(), "", "task", nil)
	u.paneW, u.paneH = 40, 6
	a := fake(t, `seq 1 30`)
	b := fake(t, `true`)
	u.racers = []*racer{a, b}
	time.Sleep(500 * time.Millisecond)
	u.notify, u.wake = func(string, string) {}, func() {}
	frameOf(t, u) // clicks land on what the last frame drew

	u.mouse(driver.MouseEvent{Button: driver.MouseLeft, X: sidebar + 5, Y: 3})
	if !u.attached || u.sel != 0 {
		t.Fatalf("a click on the pane: attached=%v sel=%d", u.attached, u.sel)
	}
	u.mouse(driver.MouseEvent{Button: driver.MouseLeft, X: 3, Y: 1 + cardH + 1})
	if u.attached || u.sel != 1 {
		t.Fatalf("a click on the second racer: attached=%v sel=%d", u.attached, u.sel)
	}
	u.mouse(driver.MouseEvent{Button: driver.MouseLeft, X: 3, Y: 1})
	u.mouse(driver.MouseEvent{Button: driver.MouseScrollUp, X: sidebar + 5, Y: 3})
	if a.scroll != 3 {
		t.Errorf("scroll = %d after the wheel, want 3", a.scroll)
	}
	top := a.emu.ScrollbackCellAt(0, a.emu.ScrollbackLen()-3)
	if top == nil || top.Content == "" {
		t.Error("nothing in the scrollback to show")
	}
	u.mouse(driver.MouseEvent{Button: driver.MouseScrollDown, X: sidebar + 5, Y: 3})
	if a.scroll != 0 {
		t.Errorf("scroll = %d after scrolling back down", a.scroll)
	}
}

// Claude Code's Stop hook runs with no terminal of its own; what it writes
// to SQUEEZE_TTY reaches the racer as a notification.
func TestClaudesStopHookSignals(t *testing.T) {
	var s struct {
		Hooks map[string][]struct {
			Hooks []struct{ Command string }
		}
	}
	if err := json.Unmarshal([]byte(claudeHooks), &s); err != nil {
		t.Fatal(err)
	}
	hook := filepath.Join(t.TempDir(), "hook.sh")
	if err := os.WriteFile(hook, []byte(s.Hooks["Stop"][0].Hooks[0].Command), 0o644); err != nil {
		t.Fatal(err)
	}
	r := fake(t, "sh "+hook+" </dev/null >/dev/null 2>&1")
	deadline := time.Now().Add(3 * time.Second)
	for r.ask() == "" && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if r.ask() != "Claude has finished" || r.state(time.Now()) != waiting {
		t.Errorf("ask = %q, state = %v", r.ask(), r.state(time.Now()))
	}
}

// The buttons work with the mouse: compare and back, quit and its yes;
// and a message goes after a while, giving the help line back.
func TestButtons(t *testing.T) {
	u := newUI(t.TempDir(), "", "task", nil)
	u.racers = []*racer{fake(t, `true`)}
	u.paneW, u.paneH = 60, 10
	click := func(label string) {
		t.Helper()
		f := frameOf(t, u)
		for _, b := range u.buttons {
			if row := []rune(f[b.y]); b.x+b.w <= len(row) && strings.Contains(string(row[b.x:b.x+b.w]), label) {
				u.mouse(driver.MouseEvent{Button: driver.MouseLeft, X: uint16(b.x + 1), Y: uint16(b.y)})
				return
			}
		}
		t.Fatalf("no %q button on\n%s", label, strings.Join(f, "\n"))
	}
	click("Compare")
	if u.mode != compareMode {
		t.Fatal("compare did not open the compare screen")
	}
	click("Back")
	if u.mode != raceMode {
		t.Fatal("back did not go back")
	}
	u.say("a message")
	if f := frameOf(t, u); !strings.Contains(f[len(f)-1], "a message") {
		t.Error("the message is not shown")
	}
	u.statusAt = time.Now().Add(-time.Minute)
	if f := frameOf(t, u); !strings.Contains(f[len(f)-1], "q quit") {
		t.Errorf("the help line did not come back: %q", f[len(f)-1])
	}
	click("Quit")
	if u.confirm != "quit" {
		t.Fatal("Quit did not ask")
	}
	time.Sleep(modalOpen) // it grows open first
	click("Quit")         // the modal's
	// The agents stop, then the modal shuts, then squeeze quits.
	deadline := time.Now().Add(5 * time.Second)
	for !u.quit && time.Now().Before(deadline) {
		frameOf(t, u)
		time.Sleep(20 * time.Millisecond)
	}
	if !u.quit {
		t.Error("quit, Quit did not quit")
	}
}

// frameOf draws one frame of u and returns its rows.
func frameOf(t *testing.T, u *ui) []string {
	t.Helper()
	term := testkit.NewTerminal(80, 16)
	term.Draw(func(f *limoni.Frame) { u.frame(f, nil) })
	return strings.Split(strings.TrimRight(term.Snapshot(), "\n"), "\n")
}

// agy's Stop hook, written beside the worktrees, signals the racer too.
func TestAgysStopHookSignals(t *testing.T) {
	dir := t.TempDir()
	if err := writeAgyHooks(dir); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(agyHooksDir(dir), ".agents", "hooks.json"))
	if err != nil {
		t.Fatal(err)
	}
	var s map[string]struct{ Stop []struct{ Command string } }
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatal(err)
	}
	hook := filepath.Join(dir, "hook.sh")
	if err := os.WriteFile(hook, []byte(s["squeeze"].Stop[0].Command), 0o644); err != nil {
		t.Fatal(err)
	}
	r := fake(t, "sh "+hook+" </dev/null >/dev/null 2>&1")
	deadline := time.Now().Add(3 * time.Second)
	for r.ask() == "" && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if r.ask() != "agy has finished" {
		t.Errorf("ask = %q", r.ask())
	}
}

// In an agent that does not take the mouse, a drag selects text, across
// lines and without the blanks that pad them, and letting go copies it;
// a click alone selects nothing.
func TestDraggingSelectsAndCopies(t *testing.T) {
	u := newUI(t.TempDir(), "", "task", nil)
	u.paneW, u.paneH = 40, 6
	u.notify, u.wake = func(string, string) {}, func() {}
	var copied string
	u.copy = func(s string) { copied = s }
	r := fake(t, `printf 'first line\r\nsecond line\r\n'`)
	u.racers = []*racer{r}
	time.Sleep(300 * time.Millisecond)

	at := func(b driver.MouseButton, drag bool, x, y int) driver.MouseEvent {
		return driver.MouseEvent{Button: b, Drag: drag, X: uint16(sidebar + 1 + x), Y: uint16(2 + y)}
	}
	u.mouse(at(driver.MouseLeft, false, 6, 0))
	u.mouse(at(driver.MouseLeft, true, 3, 1))
	u.mouse(at(driver.MouseRelease, false, 3, 1))
	if copied != "line\nseco" {
		t.Errorf("copied %q, want %q", copied, "line\nseco")
	}
	if !u.marked.has(0, 8) || u.marked.has(1, 4) {
		t.Error("the selection is not where the drag went")
	}

	copied = ""
	u.mouse(at(driver.MouseLeft, false, 2, 0))
	u.mouse(at(driver.MouseRelease, false, 2, 0))
	if copied != "" || !u.marked.empty() {
		t.Errorf("a click selected %q", copied)
	}
}

// opencode's plugin signals the racer from opencode's events, and ignores a
// subagent's session.
func TestOpencodesPluginSignals(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("no node")
	}
	dir := t.TempDir()
	if err := writeOpencodePlugin(dir); err != nil {
		t.Fatal(err)
	}
	// opencode loads the plugin with Bun, which reads its import syntax in a
	// .js file; Node before 22.7 (CI's) takes the file for CommonJS unless told.
	if err := os.WriteFile(filepath.Join(opencodeDir(dir), "package.json"), []byte(`{"type":"module"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	drive := filepath.Join(dir, "drive.mjs")
	script := `import { Squeeze } from "` + filepath.Join(opencodeDir(dir), "squeeze.js") + `"
const h = await Squeeze()
const ev = (type, properties) => h.event({ event: { type, properties } })
await ev("session.status", { sessionID: "a", status: { type: "busy" } })
await ev("session.created", { sessionID: "b", info: { parentID: "a" } })
await ev("session.idle", { sessionID: "b" })
await new Promise((r) => setTimeout(r, 600))
await ev("session.idle", { sessionID: "a" })
await new Promise((r) => setTimeout(r, 600))
`
	if err := os.WriteFile(drive, []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	r := fake(t, node+" "+drive+" </dev/null >/dev/null 2>&1; sleep 30")
	time.Sleep(400 * time.Millisecond)
	if a := r.ask(); a != "" {
		t.Errorf("a subagent going idle asked %q", a)
	}
	// Node's start-up alone can take seconds on a loaded CI runner.
	deadline := time.Now().Add(20 * time.Second)
	for r.ask() == "" && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if r.ask() != "opencode has finished" {
		t.Errorf("ask = %q", r.ask())
	}
}

// A click on a web address in the output opens it, a wrapped one whole and
// without the full stop after it; a drag over one only selects.
func TestAClickOpensALink(t *testing.T) {
	u := newUI(t.TempDir(), "", "task", nil)
	u.paneW, u.paneH = 30, 6
	u.notify, u.wake, u.copy = func(string, string) {}, func() {}, func(string) {}
	var opened []string
	u.open = func(s string) { opened = append(opened, s) }
	// 30 columns: the second address runs over the row's end and on, indented.
	r := fake(t, `printf 'see https://go.dev/doc/.\r\n  x https://example.com/aaaa/b\r\n  bbb/cccc ok\r\n'; sleep 5`)
	r.resize(30, 6)
	u.racers = []*racer{r}
	time.Sleep(300 * time.Millisecond)
	at := func(b driver.MouseButton, drag bool, x, y int) {
		u.mouse(driver.MouseEvent{Button: b, Drag: drag, X: uint16(sidebar + 1 + x), Y: uint16(2 + y)})
	}
	click := func(x, y int) { at(driver.MouseLeft, false, x, y); at(driver.MouseRelease, false, x, y) }
	click(8, 0)
	click(4, 2) // on the wrapped part
	click(1, 0) // on "see"
	at(driver.MouseLeft, false, 6, 0)
	at(driver.MouseLeft, true, 12, 0)
	at(driver.MouseRelease, false, 12, 0)
	want := []string{"https://go.dev/doc/", "https://example.com/aaaa/bbbb/cccc"}
	if strings.Join(opened, " ") != strings.Join(want, " ") {
		t.Errorf("opened %q, want %q", opened, want)
	}
}

// In a race, "+ new job / agents" opens the setup screen on the task shown:
// kept, the agents picked join its job; another, they start a job of their
// own. A racer added to a job goes after that job's racers.
func TestNewJobsAndAgentsInARace(t *testing.T) {
	root, base := newRepo(t)
	sh := agentDef{Name: "sh", Bin: "sh", Args: func(string, string) []string { return []string{"-c", "sleep 5"} }}
	u := newUI(root, base, "task", []agentDef{sh})
	u.notify, u.wake = func(string, string) {}, func() {}
	u.others = u.agents
	frameOf(t, u) // starts the race
	defer func() {
		for _, r := range u.racers {
			r.stop()
		}
	}()
	click := func(label string) {
		t.Helper()
		f := frameOf(t, u)
		for _, b := range u.buttons {
			if row := []rune(f[b.y]); b.x+b.w <= len(row) && strings.Contains(string(row[b.x:b.x+b.w]), label) {
				u.mouse(driver.MouseEvent{Button: driver.MouseLeft, X: uint16(b.x + 1), Y: uint16(b.y)})
				return
			}
		}
		t.Fatalf("no %q button on\n%s", label, strings.Join(f, "\n"))
	}
	click("New job")
	if u.mode != setupMode || u.input.Value() != "task" {
		t.Fatalf("mode %v, task %q", u.mode, u.input.Value())
	}
	click("+ add")
	click("Start")
	u.newJob()
	u.input.SetValue("other")
	u.key(limoni.KeyEvent{Type: limoni.KeyF1})
	u.key(limoni.KeyEvent{Type: limoni.KeyEnter})
	u.sel = 0
	u.newJob()
	u.key(limoni.KeyEvent{Type: limoni.KeyF1})
	u.key(limoni.KeyEvent{Type: limoni.KeyEnter})

	var got []string
	for _, r := range u.racers {
		got = append(got, fmt.Sprintf("%s:%d:%s", r.name, r.job, r.task))
	}
	if want := "sh:0:task sh-2:0:task sh-4:0:task sh-3:1:other"; strings.Join(got, " ") != want {
		t.Errorf("racers %s, want %s", strings.Join(got, " "), want)
	}
	if u.mode != raceMode || u.racers[u.sel].name != "sh-4" {
		t.Errorf("mode %v, shown %s", u.mode, u.racers[u.sel].name)
	}
	if _, err := os.Stat(filepath.Join(worktreesDir(root), "sh-3")); err != nil {
		t.Error("no worktree for the new job's racer:", err)
	}
	u.sel = 3
	if f := strings.Join(frameOf(t, u), "\n"); !strings.Contains(f, "#2 other") {
		t.Errorf("the cards do not name the jobs:\n%s", f)
	}
}
