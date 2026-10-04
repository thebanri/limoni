package uitest

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/thebanri/limoni/automation"
	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/engine"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/widgets"
)

// label is a line of text with a semantic node, standing in for the status
// lines and buttons real applications build from widgets.Accessible.
type label struct {
	widgets.Accessible
}

func (l label) Draw(ctx cell.Context, buf *buffer.Buffer) {
	buf.SetStringWithin(ctx.Area.X, ctx.Area.Y, l.Label, ctx.Style, ctx.Area.Width)
}

func (l label) SizeHint(cell.Rect) (uint16, uint16) { return uint16(len(l.Label)), 1 }

// todoApp is an immediate-mode application: a name field, an Add button, a
// task list, a status line, and Esc to quit.
type todoApp struct {
	name   *widgets.TextInputState
	list   *widgets.ListState
	tasks  []string
	status string

	// background is set by another goroutine, as a network response would be.
	background atomic.Value
}

func newTodoApp() *todoApp {
	a := &todoApp{name: widgets.NewTextInputState(), list: widgets.NewListState(), status: "ready"}
	a.background.Store("")
	return a
}

func (a *todoApp) add() {
	if v := a.name.Value(); v != "" {
		a.tasks = append(a.tasks, v)
		a.name.SetValue("")
		a.status = fmt.Sprintf("%d task(s)", len(a.tasks))
	}
}

func (a *todoApp) draw(f *terminal.Frame, ev *driver.Event) bool {
	if ev != nil && ev.Type == driver.EventKey {
		switch {
		case ev.Key.Type == driver.KeyEsc:
			return false
		case ev.Key.Type == driver.KeyTab:
			f.FocusManager.Next()
		case f.FocusManager.Focused() == "name" && ev.Key.Type == driver.KeyEnter:
			a.add()
		case f.FocusManager.Focused() == "name":
			a.name.HandleKey(ev.Key)
		case f.FocusManager.Focused() == "tasks" && ev.Key.Type == driver.KeyArrowDown:
			a.list.Next()
		}
	}
	f.RenderWidget(&widgets.TextInput{ID: "name", State: a.name, Placeholder: "Task name", Focused: f.FocusManager.IsFocused("name")}, cell.NewRect(0, 0, 30, 1))
	addArea := cell.NewRect(32, 0, 9, 1)
	f.RenderWidget(label{widgets.Accessible{ID: "add", Role: accessibility.RoleButton, Label: "Add"}}, addArea)
	f.RegisterClickHandler(addArea, func(driver.MouseEvent) { a.add() })
	if len(a.tasks) > 0 {
		f.RenderWidget(&widgets.List{ID: "tasks", Label: "Tasks", Items: a.tasks, State: a.list}, cell.NewRect(0, 2, 30, uint16(len(a.tasks))))
	}
	f.RenderWidget(label{widgets.Accessible{ID: "status", Label: a.status}}, cell.NewRect(0, 20, 40, 1))
	if bg := a.background.Load().(string); bg != "" {
		f.RenderWidget(label{widgets.Accessible{ID: "notice", Label: bg}}, cell.NewRect(0, 22, 40, 1))
	}
	return true
}

func TestRunDrivesAnImmediateModeApplication(t *testing.T) {
	app := newTodoApp()
	page := Run(t, 60, 24, app.draw)

	page.GetByRole("input", "Task name").Type("write docs")
	page.Expect(page.GetByID("name")).ToHaveValue("write docs")
	page.GetByRole("button", "Add").Click()

	page.GetByID("name").Type("ship")
	page.GetByID("name").Press("enter")

	rows := page.GetByRole("list-item", "").Within(page.GetByRole("list", "Tasks"))
	page.Expect(rows).ToHaveCount(2)
	page.Expect(page.GetByRole("list-item", "ship")).ToBeVisible()
	page.Expect(page.GetByID("status")).ToHaveLabel("2 task(s)")
	page.Expect(page.GetByID("name")).ToHaveValue("")

	rows.Nth(-1).Click()
	page.Expect(page.GetByRole("list", "Tasks")).ToHaveValue("ship")
	page.Expect(page.GetByRole("list", "Tasks")).ToBeFocused()
	page.Expect(rows.Nth(1)).ToBeSelected()
	page.Expect(rows.Nth(0)).Not().ToBeSelected()

	page.Press("tab")
	page.Expect(page.GetByID("name")).ToBeFocused()

	page.Press("esc")
	if !page.Exited() {
		t.Error("the application did not exit on esc")
	}
}

// State changed by another goroutine appears on a later frame. Assertions
// wait for it; nothing in the test sleeps.
func TestAssertionsWaitForBackgroundChanges(t *testing.T) {
	app := newTodoApp()
	page := Run(t, 60, 24, app.draw)
	go func() {
		time.Sleep(80 * time.Millisecond)
		app.background.Store("sync finished")
	}()
	page.Expect(page.GetByText("finished")).ToBeVisible()
	page.Expect(page.GetByID("notice")).ToContainLabel("sync")

	go func() {
		time.Sleep(80 * time.Millisecond)
		app.background.Store("")
	}()
	page.Expect(page.GetByID("notice")).Not().ToBeVisible()
}

// counter is a declarative model: '+' increments, a command reports "saved"
// after the count reaches two, and 'q' quits.
type counter struct {
	count int
	saved bool
}

type savedMsg struct{}

func (m *counter) Init() []engine.Cmd { return nil }

func (m *counter) Update(msg engine.Msg) engine.UpdateResult {
	switch msg := msg.(type) {
	case engine.KeyPressMsg:
		switch msg.Key.Ch {
		case '+':
			m.count++
			if m.count == 2 {
				return engine.UpdateResult{Redraw: true, Commands: []engine.Cmd{func(context.Context) engine.Msg {
					time.Sleep(30 * time.Millisecond)
					return savedMsg{}
				}}}
			}
			return engine.UpdateResult{Redraw: true}
		case 'q':
			return engine.UpdateResult{Quit: true}
		}
	case savedMsg:
		m.saved = true
		return engine.UpdateResult{Redraw: true}
	}
	return engine.UpdateResult{}
}

func (m *counter) View(f *terminal.Frame) {
	f.RenderWidget(label{widgets.Accessible{ID: "count", Label: fmt.Sprintf("count %d", m.count)}}, cell.NewRect(0, 0, 20, 1))
	if m.saved {
		f.RenderWidget(label{widgets.Accessible{ID: "saved", Label: "saved"}}, cell.NewRect(0, 2, 20, 1))
	}
}

func TestProgramDrivesADeclarativeModelThroughItsMessageLoop(t *testing.T) {
	page := Program(t, 40, 10, &counter{})
	page.Press("+")
	page.Expect(page.GetByID("count")).ToHaveLabel("count 1")
	page.Press("+")
	page.Expect(page.GetByID("saved")).ToBeVisible()
	page.Press("q")
	deadline := time.Now().Add(2 * time.Second)
	for !page.Exited() {
		if time.Now().After(deadline) {
			t.Fatal("the program did not quit on q")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestConnectDrivesARunningApplicationOverItsSocket(t *testing.T) {
	socket := shortSocket(t)
	policy := automation.Policy{AllowInput: true, ExposeInputValues: true}
	switch runtime.GOOS {
	case "linux", "darwin", "freebsd":
	default:
		policy.AllowUnverifiedPeers = true
	}
	server, err := automation.Listen(socket, automation.WithPolicy(policy))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })

	var mu sync.Mutex
	clicks := 0
	var publish func()
	inject := func(ev driver.Event) {
		if ev.Type == driver.EventMouse {
			go func() {
				time.Sleep(20 * time.Millisecond) // the next frame
				mu.Lock()
				clicks++
				mu.Unlock()
				publish()
			}()
		}
	}
	publish = func() {
		mu.Lock()
		n := clicks
		mu.Unlock()
		server.Publish(automation.Snapshot{
			Tree: []accessibility.AccessibilityNode{
				{ID: "go", Role: accessibility.RoleButton, Label: "Go", Bounds: cell.NewRect(0, 0, 4, 1)},
				{ID: "clicks", Label: fmt.Sprintf("clicked %d", n), Bounds: cell.NewRect(0, 1, 10, 1)},
			},
			Width: 20, Height: 5, Injector: inject,
		})
	}
	publish()

	page := Connect(t, socket)
	page.GetByRole("button", "Go").Click()
	page.Expect(page.GetByID("clicks")).ToHaveLabel("clicked 1")
}

// fakeT records a failure instead of failing the real test, so the failure
// messages themselves can be checked. Fatalf ends the goroutine, as the real
// one does.
type fakeT struct {
	testing.TB
	mu      sync.Mutex
	message string
}

func (f *fakeT) Helper() {}

func (f *fakeT) Fatalf(format string, args ...any) {
	f.mu.Lock()
	f.message = fmt.Sprintf(format, args...)
	f.mu.Unlock()
	runtime.Goexit()
}

func failure(t *testing.T, body func(page *Page)) string {
	t.Helper()
	fake := &fakeT{TB: t}
	app := newTodoApp()
	page := Run(fake, 60, 24, app.draw, WithTimeout(60*time.Millisecond))
	done := make(chan struct{})
	go func() {
		defer close(done)
		body(page)
	}()
	<-done
	fake.mu.Lock()
	defer fake.mu.Unlock()
	return fake.message
}

func TestFailuresSayWhatWasExpectedAndShowTheFrame(t *testing.T) {
	msg := failure(t, func(page *Page) {
		page.Expect(page.GetByID("status")).ToHaveLabel("done")
	})
	for _, want := range []string{`expected id="status" to have label "done"`, `got label "ready"`, "after 60ms", "last frame:", `button#add "Add"`} {
		if !strings.Contains(msg, want) {
			t.Errorf("failure message lacks %q:\n%s", want, msg)
		}
	}

	msg = failure(t, func(page *Page) {
		page.GetByRole("button", "Delete").Click()
	})
	if !strings.Contains(msg, `click role="button" label="Delete": nothing matches`) {
		t.Errorf("missing widget:\n%s", msg)
	}

	msg = failure(t, func(page *Page) {
		page.GetByText("a").Click() // "Task name" and "ready"
	})
	if !strings.Contains(msg, "matches 2 widgets; narrow it, or use Nth") {
		t.Errorf("ambiguous locator:\n%s", msg)
	}

	msg = failure(t, func(page *Page) {
		page.GetByRole("button", "Add").Type("x")
	})
	if !strings.Contains(msg, "did not take focus when clicked") {
		t.Errorf("typing into something unfocusable:\n%s", msg)
	}

	if msg := failure(t, func(page *Page) { page.Press("hyper+x") }); !strings.Contains(msg, `unknown modifier "hyper"`) {
		t.Errorf("bad chord:\n%s", msg)
	}

	// A passing body leaves no message.
	if msg := failure(t, func(page *Page) { page.Expect(page.GetByID("notice")).Not().ToBeVisible() }); msg != "" {
		t.Errorf("a passing assertion failed: %s", msg)
	}
}

func TestParseChord(t *testing.T) {
	for _, tc := range []struct {
		chord            string
		typ              driver.KeyType
		ch               rune
		ctrl, alt, shift bool
	}{
		{"enter", driver.KeyEnter, 0, false, false, false},
		{"ctrl+s", driver.KeyRune, 's', true, false, false},
		{"shift+tab", driver.KeyTab, 0, false, false, true},
		{"ctrl+alt+delete", driver.KeyDelete, 0, true, true, false},
		{"+", driver.KeyRune, '+', false, false, false},
		{"ctrl++", driver.KeyRune, '+', true, false, false},
	} {
		ev, _, err := parseChord(tc.chord)
		if err != nil {
			t.Errorf("%s: %v", tc.chord, err)
			continue
		}
		if ev.Type != tc.typ || ev.Ch != tc.ch || ev.Ctrl != tc.ctrl || ev.Alt != tc.alt || ev.Shift != tc.shift {
			t.Errorf("%s = %+v", tc.chord, ev)
		}
	}
}

func shortSocket(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "lui")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return filepath.Join(dir, "a.sock")
}

// Every action is logged, so a failure arrives with the steps before it, and
// slow motion paces actions for a person watching.
func TestActionsAreLoggedAndSlowMoPacesThem(t *testing.T) {
	rec := &logT{TB: t}
	app := newTodoApp()
	page := Run(rec, 60, 24, app.draw, WithSlowMo(40*time.Millisecond))

	start := time.Now()
	page.GetByID("name").Type("secret-ish")
	page.GetByRole("button", "Add").Click()
	page.Press("tab")
	elapsed := time.Since(start)

	logs := strings.Join(rec.lines, "\n")
	for _, want := range []string{`uitest: type 10 character(s) into id="name"`, `uitest: click role="button" label="Add"`, "uitest: press tab"} {
		if !strings.Contains(logs, want) {
			t.Errorf("log lacks %q:\n%s", want, logs)
		}
	}
	if strings.Contains(logs, "secret-ish") {
		t.Errorf("typed text was logged:\n%s", logs)
	}
	if elapsed < 3*40*time.Millisecond {
		t.Errorf("three slowed actions took %s, want at least %s", elapsed, 3*40*time.Millisecond)
	}
}

type logT struct {
	testing.TB
	lines []string
}

func (l *logT) Helper() {}

func (l *logT) Logf(format string, args ...any) {
	l.lines = append(l.lines, fmt.Sprintf(format, args...))
}

// editor asks for an editor on 'e' and shows what came back.
type editor struct{ status string }

type editorClosed struct{ err error }

func (m *editor) Init() []engine.Cmd { return nil }

func (m *editor) Update(msg engine.Msg) engine.UpdateResult {
	switch msg := msg.(type) {
	case engine.KeyPressMsg:
		if msg.Key.Ch == 'e' {
			return engine.UpdateResult{Commands: []engine.Cmd{engine.ExecCmd(exec.Command("true"), func(err error) engine.Msg {
				return editorClosed{err}
			})}}
		}
	case editorClosed:
		m.status = fmt.Sprint(msg.err)
		return engine.UpdateResult{Redraw: true}
	}
	return engine.UpdateResult{}
}

func (m *editor) View(f *terminal.Frame) {
	f.RenderWidget(label{widgets.Accessible{ID: "status", Label: m.status}}, cell.NewRect(0, 0, 40, 1))
}

// A test's terminal is in memory, with nothing to hand to an editor: the
// model hears so, as it would in a remote session, rather than never hearing.
func TestProgramAnswersExecCmdWithUnsupported(t *testing.T) {
	page := Program(t, 60, 3, &editor{})
	page.Press("e")
	page.Expect(page.GetByID("status")).ToHaveLabel(driver.ErrReleaseUnsupported.Error())
}
