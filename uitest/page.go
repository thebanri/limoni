// Package uitest writes tests for Limoni applications the way Playwright
// writes them for web pages: find widgets by what they are, act on them, and
// assert with checks that wait for the application to catch up.
//
//	func TestAddTask(t *testing.T) {
//		page := uitest.Run(t, 80, 24, app.Draw)
//
//		page.GetByRole("input", "New task").Type("Tag v1.0")
//		page.GetByRole("button", "Add task").Click()
//
//		page.Expect(page.GetByRole("list-item", "Tag v1.0")).ToBeVisible()
//		page.Expect(page.GetByID("status")).ToContainLabel("Added")
//	}
//
// A locator never mentions a coordinate. It is a query against the semantic
// tree — the one a screen reader reads — resolved again every time it is used,
// so a test survives relayout, resizing and restyling, and a locator created
// before a widget exists works once it appears.
//
// # Waiting instead of sleeping
//
// Actions wait until their locator matches exactly one widget. Assertions retry
// until they hold. Both give up after the page's timeout, five seconds by
// default, and fail the test with the tree as it last was. There is no call to
// sleep in a test written with this package, because a fixed sleep is what
// makes terminal UI tests flaky: too short on a loaded CI machine, wasted time
// everywhere else.
//
// # Three ways to reach an application
//
//   - Run drives an immediate-mode draw function in process, with no terminal
//     and no build tag.
//   - Program drives a declarative Model in process, through its real message
//     loop, so commands and their results behave as they do in production.
//   - Connect drives a running binary through its automation socket, built with
//     -tags limoni_debug and started with limoni.WithAutomation.
//
// The same locators and assertions work against all three. The in-process
// pages see the application's full tree; a connected page sees what the
// application's automation policy lets out, secrets always redacted.
package uitest

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/thebanri/limoni/automation"
	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/engine"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/testkit"
)

// DefaultTimeout is how long actions and assertions wait unless the page was
// created WithTimeout.
const DefaultTimeout = 5 * time.Second

// Option configures a Page.
type Option func(*Page)

// WithTimeout sets how long actions wait for their locator and assertions
// retry before failing the test.
func WithTimeout(d time.Duration) Option {
	return func(p *Page) {
		if d > 0 {
			p.timeout = d
		}
	}
}

// WithSlowMo pauses for d after every action, so a person can follow a test
// driving a visible application — for a demo, or for working out why a test
// does what it does. Assertions are not slowed; they already wait for as long
// as they need to.
func WithSlowMo(d time.Duration) Option {
	return func(p *Page) {
		if d > 0 {
			p.slowMo = d
		}
	}
}

// Page is one application under test.
type Page struct {
	t       testing.TB
	app     app
	timeout time.Duration
	slowMo  time.Duration
}

// acted records an action in the test log — shown with go test -v, and above
// the failure message when a later step fails, so the failure comes with the
// steps that led to it — then applies the slow-motion pause.
func (p *Page) acted(format string, args ...any) {
	p.t.Helper()
	p.t.Logf("uitest: "+format, args...)
	if p.slowMo > 0 {
		time.Sleep(p.slowMo)
	}
}

// app is what a page drives: an in-process application or a remote one.
type app interface {
	// tree returns the semantic tree of the latest frame, first rendering a
	// new frame if the application has one pending.
	tree() ([]accessibility.AccessibilityNode, error)
	click(node accessibility.AccessibilityNode, sel automation.Selector) error
	key(ev driver.KeyEvent, name string) error
	typeText(text string) error
	screen() (string, error)
}

// errExited is returned once an in-process application has quit.
var errExited = errors.New("uitest: the application has exited")

func newPage(t testing.TB, a app, opts []Option) *Page {
	p := &Page{t: t, app: a, timeout: DefaultTimeout}
	for _, opt := range opts {
		if opt != nil {
			opt(p)
		}
	}
	return p
}

// Run starts an immediate-mode application in process, the function you would
// pass to limoni.Run, on a width×height screen. It draws the first frame
// before returning.
//
// Input is delivered the way limoni.Run delivers it: a click is routed to the
// click handlers the previous frame registered, then the function is called
// with the event. While an action or assertion waits, the function is also
// called with a nil event, as it is on a Wakeup, so state changed by a
// background goroutine shows up.
func Run(t testing.TB, width, height uint16, draw func(*terminal.Frame, *driver.Event) bool, opts ...Option) *Page {
	t.Helper()
	a := &runApp{term: testkit.NewTerminal(width, height), draw: draw, running: true}
	a.frame(nil)
	return newPage(t, a, opts)
}

type runApp struct {
	mu      sync.Mutex
	term    *testkit.Terminal
	draw    func(*terminal.Frame, *driver.Event) bool
	running bool
}

func (a *runApp) frame(ev *driver.Event) {
	if !a.running {
		return
	}
	a.term.Draw(func(f *terminal.Frame) {
		a.running = a.draw(f, ev)
	})
}

func (a *runApp) tree() ([]accessibility.AccessibilityNode, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.running {
		return nil, errExited
	}
	a.frame(nil)
	return a.term.AccessibilityTree(), nil
}

func (a *runApp) click(node accessibility.AccessibilityNode, _ automation.Selector) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.running {
		return errExited
	}
	ev := driver.Event{Type: driver.EventMouse, Mouse: centre(node)}
	a.term.Mouse(ev.Mouse)
	a.frame(&ev)
	return nil
}

func (a *runApp) key(key driver.KeyEvent, _ string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.running {
		return errExited
	}
	ev := driver.Event{Type: driver.EventKey, Key: key}
	a.frame(&ev)
	return nil
}

func (a *runApp) typeText(text string) error {
	for _, r := range text {
		if err := a.key(driver.KeyEvent{Type: driver.KeyRune, Ch: r}, ""); err != nil {
			return err
		}
	}
	return nil
}

func (a *runApp) screen() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.term.Snapshot(), nil
}

// Program starts a declarative application in process: model's Init runs, its
// commands are scheduled, and every input goes through the real message loop.
// The program is stopped when the test ends.
//
// Because the loop runs on its own goroutine, an input's effect is not visible
// the moment the action returns. Assertions wait for it; that is the point of
// them.
func Program(t testing.TB, width, height uint16, model engine.Model, opts ...Option) *Page {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	program := engine.New(engine.WithModel(model))
	a := &programApp{term: testkit.NewTerminal(width, height), program: program, ctx: ctx, done: make(chan error, 1)}
	go func() { a.done <- program.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		program.Stop()
		select {
		case <-a.done:
		case <-time.After(5 * time.Second):
			t.Error("uitest: the program did not stop within 5s of the test ending")
		}
	})
	return newPage(t, a, opts)
}

type programApp struct {
	mu      sync.Mutex
	term    *testkit.Terminal
	program *engine.Program
	ctx     context.Context
	done    chan error

	exited bool
}

func (a *programApp) checkRunning() error {
	if a.exited {
		return errExited
	}
	select {
	case err := <-a.done:
		a.exited = true
		a.done <- err // leave it for the cleanup
		return errExited
	default:
		return nil
	}
}

func (a *programApp) tree() ([]accessibility.AccessibilityNode, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.checkRunning(); err != nil {
		return nil, err
	}
	a.term.Draw(a.program.View)
	return a.term.AccessibilityTree(), nil
}

func (a *programApp) send(ev driver.Event) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.checkRunning(); err != nil {
		return err
	}
	return a.program.SendDriver(a.ctx, ev)
}

func (a *programApp) click(node accessibility.AccessibilityNode, _ automation.Selector) error {
	return a.send(driver.Event{Type: driver.EventMouse, Mouse: centre(node)})
}

func (a *programApp) key(key driver.KeyEvent, _ string) error {
	return a.send(driver.Event{Type: driver.EventKey, Key: key})
}

func (a *programApp) typeText(text string) error {
	for _, r := range text {
		if err := a.key(driver.KeyEvent{Type: driver.KeyRune, Ch: r}, ""); err != nil {
			return err
		}
	}
	return nil
}

func (a *programApp) screen() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.term.Draw(a.program.View)
	return a.term.Snapshot(), nil
}

// Connect drives a running application through its automation socket. The
// application must be built with -tags limoni_debug and started with
// limoni.WithAutomation; its policy decides what the page can see and do.
// The connection is closed when the test ends.
func Connect(t testing.TB, socketPath string, opts ...Option) *Page {
	t.Helper()
	client, err := automation.Dial(socketPath)
	if err != nil {
		t.Fatalf("uitest: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return newPage(t, &remoteApp{client: client}, opts)
}

type remoteApp struct {
	client *automation.Client
}

func (a *remoteApp) tree() ([]accessibility.AccessibilityNode, error) { return a.client.Tree() }

// click sends the selector rather than the resolved node's coordinates, so the
// application resolves it against the frame it is actually showing.
func (a *remoteApp) click(_ accessibility.AccessibilityNode, sel automation.Selector) error {
	return a.client.Click(sel)
}

func (a *remoteApp) key(key driver.KeyEvent, name string) error {
	_, err := a.client.Do(automation.Request{Op: automation.OpKey, Key: name, Ctrl: key.Ctrl, Alt: key.Alt, Shift: key.Shift})
	return err
}

func (a *remoteApp) typeText(text string) error { return a.client.Type(text) }

func (a *remoteApp) screen() (string, error) { return a.client.Screen() }

func centre(node accessibility.AccessibilityNode) driver.MouseEvent {
	return driver.MouseEvent{
		Button: driver.MouseLeft,
		X:      node.Bounds.X + node.Bounds.Width/2,
		Y:      node.Bounds.Y + node.Bounds.Height/2,
	}
}

// Tree returns the semantic tree of the current frame.
func (p *Page) Tree() []accessibility.AccessibilityNode {
	p.t.Helper()
	nodes, err := p.app.tree()
	if err != nil {
		p.t.Fatalf("uitest: %v", err)
	}
	return nodes
}

// Screen returns the rendered text grid, for what the tree cannot express.
func (p *Page) Screen() string {
	p.t.Helper()
	text, err := p.app.screen()
	if err != nil {
		p.t.Fatalf("uitest: %v", err)
	}
	return text
}

// Press sends one key press to whatever has focus. key is a single character
// or a key name — "enter", "tab", "esc", "up", "f5" — optionally prefixed with
// modifiers: "ctrl+s", "shift+tab", "alt+enter".
func (p *Page) Press(key string) {
	p.t.Helper()
	ev, name, err := parseChord(key)
	if err != nil {
		p.t.Fatalf("uitest: %v", err)
	}
	if err := p.app.key(ev, name); err != nil {
		p.t.Fatalf("uitest: press %s: %v", key, err)
	}
	p.acted("press %s", key)
}

// Type sends text to whatever has focus, one key press per character.
func (p *Page) Type(text string) {
	p.t.Helper()
	if err := p.app.typeText(text); err != nil {
		p.t.Fatalf("uitest: type %q: %v", text, err)
	}
	// The text itself is not logged: it may be a password.
	p.acted("type %d character(s)", len([]rune(text)))
}

// Exited reports whether an in-process application has quit — its draw
// function returned false, or its program stopped.
func (p *Page) Exited() bool {
	_, err := p.app.tree()
	return errors.Is(err, errExited)
}

// ExpectExit waits for the application to quit and fails the test if it has
// not within the page's timeout. Prefer it to Exited after the key that
// quits: a declarative program quits through its message loop, a moment
// after the key is delivered.
func (p *Page) ExpectExit() {
	p.t.Helper()
	deadline := time.Now().Add(p.timeout)
	for !p.Exited() {
		if time.Now().After(deadline) {
			p.t.Fatalf("uitest: the application did not exit within %s", p.timeout)
		}
		time.Sleep(pollInterval)
	}
}

func parseChord(chord string) (driver.KeyEvent, string, error) {
	var ctrl, alt, shift bool
	name := chord
	// A lone "+" is the plus key, not an empty chord.
	for len(name) > 1 {
		i := strings.IndexByte(name, '+')
		if i <= 0 || i == len(name)-1 {
			break
		}
		switch strings.ToLower(name[:i]) {
		case "ctrl", "control":
			ctrl = true
		case "alt", "option":
			alt = true
		case "shift":
			shift = true
		default:
			return driver.KeyEvent{}, "", fmt.Errorf("unknown modifier %q in %q", name[:i], chord)
		}
		name = name[i+1:]
	}
	ev, err := automation.ParseKey(name)
	if err != nil {
		return driver.KeyEvent{}, "", err
	}
	ev.Ctrl, ev.Alt, ev.Shift = ctrl, alt, shift
	return ev, name, nil
}
