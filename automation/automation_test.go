package automation

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

// recorder stands in for a running application: it captures the events the
// server synthesises so a test can assert on what the app would have received.
type recorder struct {
	mu     sync.Mutex
	events []driver.Event
}

func (r *recorder) inject(ev driver.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, ev)
}

func (r *recorder) snapshot() []driver.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]driver.Event(nil), r.events...)
}

func sampleTree() []accessibility.AccessibilityNode {
	return []accessibility.AccessibilityNode{
		{
			ID:     "submit",
			Role:   accessibility.RoleButton,
			Label:  "Submit",
			Bounds: cell.NewRect(10, 4, 8, 3),
		},
		{
			ID:       "items",
			Role:     accessibility.RoleList,
			Label:    "List",
			Value:    "beta",
			Position: 2,
			SetSize:  3,
			Bounds:   cell.NewRect(0, 0, 20, 10),
			Children: []accessibility.AccessibilityNode{
				{ID: "row-1", Role: accessibility.RoleListItem, Label: "alpha", Bounds: cell.NewRect(0, 0, 20, 1)},
				{ID: "row-2", Role: accessibility.RoleListItem, Label: "beta", Bounds: cell.NewRect(0, 1, 20, 1)},
			},
		},
		{ID: "cancel", Role: accessibility.RoleButton, Label: "Cancel", Bounds: cell.NewRect(20, 4, 8, 3)},
	}
}

func startServer(t *testing.T) (*Server, *Client, *recorder) {
	t.Helper()
	rec := &recorder{}
	socket := shortSocketPath(t)

	server, err := Listen(socket, WithPolicy(Policy{AllowInput: true, ExposeScreen: true, AllowUnverifiedPeers: !peerVerificationSupported}))
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	t.Cleanup(func() { _ = server.Close() })

	server.Publish(Snapshot{
		Tree:     sampleTree(),
		Screen:   "hello\nworld",
		Focused:  "submit",
		Width:    80,
		Height:   24,
		Injector: rec.inject,
	})

	client, err := Dial(socket)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	return server, client, rec
}

func TestHelloReportsVersionAndSize(t *testing.T) {
	_, client, _ := startServer(t)
	version, width, height, err := client.Hello()
	if err != nil {
		t.Fatalf("Hello: %v", err)
	}
	if version != Version {
		t.Errorf("version = %d, want %d", version, Version)
	}
	if width != 80 || height != 24 {
		t.Errorf("size = %dx%d, want 80x24", width, height)
	}
}

func TestTreeAndScreenRoundTrip(t *testing.T) {
	_, client, _ := startServer(t)

	nodes, err := client.Tree()
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}
	if len(nodes) != 3 {
		t.Fatalf("tree has %d roots, want 3", len(nodes))
	}
	if nodes[1].Position != 2 || nodes[1].SetSize != 3 {
		t.Errorf("position survived as %d/%d, want 2/3", nodes[1].Position, nodes[1].SetSize)
	}
	if len(nodes[1].Children) != 2 {
		t.Errorf("children did not survive the wire: %d", len(nodes[1].Children))
	}

	screen, err := client.Screen()
	if err != nil {
		t.Fatalf("Screen: %v", err)
	}
	if screen != "hello\nworld" {
		t.Errorf("screen = %q", screen)
	}

	focused, err := client.Focused()
	if err != nil {
		t.Fatalf("Focused: %v", err)
	}
	if focused != "submit" {
		t.Errorf("focused = %q, want submit", focused)
	}
}

func TestFindMatchesByMeaning(t *testing.T) {
	_, client, _ := startServer(t)

	buttons, err := client.Find(Selector{Role: "button"})
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(buttons) != 2 {
		t.Fatalf("found %d buttons, want 2", len(buttons))
	}

	// Nested nodes are reachable, which is what makes a list item addressable.
	items, err := client.Find(Selector{Role: "listitem"})
	if err != nil {
		t.Fatalf("Find listitem: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("found %d list items, want 2", len(items))
	}

	byLabel, err := client.Find(Selector{LabelContains: "ubmi"})
	if err != nil {
		t.Fatalf("Find label: %v", err)
	}
	if len(byLabel) != 1 || byLabel[0].ID != "submit" {
		t.Errorf("label match = %+v", byLabel)
	}
}

// An ambiguous selector must fail rather than silently pick one: a test that
// picks the first match passes for the wrong reason once a second button
// appears.
func TestAmbiguousSelectorIsRejected(t *testing.T) {
	_, client, rec := startServer(t)

	err := client.Click(Selector{Role: "button"})
	if err == nil {
		t.Fatal("clicking an ambiguous selector succeeded")
	}
	if got := rec.snapshot(); len(got) != 0 {
		t.Errorf("ambiguous click still injected %d events", len(got))
	}

	nth := 1
	if err := client.Click(Selector{Role: "button", Nth: &nth}); err != nil {
		t.Fatalf("Click with nth: %v", err)
	}
}

func TestMissingSelectorIsRejected(t *testing.T) {
	_, client, _ := startServer(t)
	if err := client.Click(Selector{Label: "Nope"}); err == nil {
		t.Fatal("clicking a non-existent node succeeded")
	}
}

func TestClickTargetsNodeCentre(t *testing.T) {
	_, client, rec := startServer(t)

	if err := client.Click(Selector{ID: "submit"}); err != nil {
		t.Fatalf("Click: %v", err)
	}

	// A press and a release, as a terminal reports a click.
	events := rec.snapshot()
	if len(events) != 2 {
		t.Fatalf("injected %d events, want a press and a release", len(events))
	}
	for i, want := range []driver.MouseButton{driver.MouseLeft, driver.MouseRelease} {
		ev := events[i]
		if ev.Type != driver.EventMouse || ev.Mouse.Button != want {
			t.Fatalf("event %d is %+v, want button %v", i, ev, want)
		}
		// Submit is at 10,4 sized 8x3, so the centre is 14,5.
		if ev.Mouse.X != 14 || ev.Mouse.Y != 5 {
			t.Errorf("event %d at %d,%d, want 14,5", i, ev.Mouse.X, ev.Mouse.Y)
		}
	}
}

func TestKeyAndTextInjection(t *testing.T) {
	_, client, rec := startServer(t)

	if err := client.Key("enter"); err != nil {
		t.Fatalf("Key: %v", err)
	}
	if err := client.Key("a"); err != nil {
		t.Fatalf("Key rune: %v", err)
	}
	if err := client.Type("hi"); err != nil {
		t.Fatalf("Type: %v", err)
	}
	if err := client.Key("nonsense"); err == nil {
		t.Fatal("unknown key name was accepted")
	}

	events := rec.snapshot()
	if len(events) != 4 {
		t.Fatalf("injected %d events, want 4", len(events))
	}
	if events[0].Key.Type != driver.KeyEnter {
		t.Errorf("first key = %v, want enter", events[0].Key.Type)
	}
	if events[1].Key.Type != driver.KeyRune || events[1].Key.Ch != 'a' {
		t.Errorf("second key = %+v, want rune a", events[1].Key)
	}
	if events[2].Key.Ch != 'h' || events[3].Key.Ch != 'i' {
		t.Errorf("typed %q%q, want hi", events[2].Key.Ch, events[3].Key.Ch)
	}
}

func TestUnknownOpIsRejected(t *testing.T) {
	_, client, _ := startServer(t)
	if _, err := client.Do(Request{Op: "teleport"}); err == nil {
		t.Fatal("unknown op was accepted")
	}
}

// A selector survives the widget moving, which is the entire point: a
// coordinate-based test breaks here, a semantic one does not.
func TestSelectorSurvivesRelayout(t *testing.T) {
	server, client, rec := startServer(t)

	moved := sampleTree()
	moved[0].Bounds = cell.NewRect(40, 18, 10, 3)
	server.Publish(Snapshot{Tree: moved, Width: 80, Height: 24, Injector: rec.inject})

	if err := client.Click(Selector{Label: "Submit"}); err != nil {
		t.Fatalf("Click after relayout: %v", err)
	}
	events := rec.snapshot()
	ev := events[len(events)-1]
	if ev.Mouse.X != 45 || ev.Mouse.Y != 19 {
		t.Errorf("clicked %d,%d, want the new centre 45,19", ev.Mouse.X, ev.Mouse.Y)
	}
}

func TestWaitForPollsUntilPresent(t *testing.T) {
	server, client, rec := startServer(t)

	server.Publish(Snapshot{Tree: nil, Width: 80, Height: 24, Injector: rec.inject})

	go func() {
		time.Sleep(30 * time.Millisecond)
		server.Publish(Snapshot{Tree: sampleTree(), Width: 80, Height: 24, Injector: rec.inject})
	}()

	node, err := client.WaitFor(Selector{ID: "submit"}, 2*time.Second)
	if err != nil {
		t.Fatalf("WaitFor: %v", err)
	}
	if node.Label != "Submit" {
		t.Errorf("waited for %q", node.Label)
	}
}

func TestWaitForTimesOut(t *testing.T) {
	_, client, _ := startServer(t)
	if _, err := client.WaitFor(Selector{ID: "never"}, 50*time.Millisecond); err == nil {
		t.Fatal("WaitFor returned success for a node that never appears")
	}
}

func TestInjectionRefusedWithoutInjector(t *testing.T) {
	server, client, _ := startServer(t)
	server.Publish(Snapshot{Tree: sampleTree(), Width: 80, Height: 24})

	if err := client.Key("enter"); err == nil {
		t.Fatal("key injection succeeded with no injector wired")
	}
	if err := client.Click(Selector{ID: "submit"}); err == nil {
		t.Fatal("click succeeded with no injector wired")
	}
}

func TestRoleSelectorToleratesSpelling(t *testing.T) {
	_, client, _ := startServer(t)
	for _, spelling := range []string{"list-item", "listitem", "ListItem", "LIST_ITEM"} {
		nodes, err := client.Find(Selector{Role: spelling})
		if err != nil {
			t.Fatalf("Find(%q): %v", spelling, err)
		}
		if len(nodes) != 2 {
			t.Errorf("Find(%q) matched %d nodes, want 2", spelling, len(nodes))
		}
	}
}

// shortSocketPath returns a socket path well under the sockaddr_un limit.
// t.TempDir() embeds the test name and a long random suffix, which on macOS
// pushes the path past 104 bytes and makes bind fail with a bare EINVAL.
func shortSocketPath(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "lmn")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return filepath.Join(dir, "a.sock")
}

// The macOS CI path that first broke this: 122 bytes, over the sockaddr_un
// limit. Before the guard, bind failed with a bare EINVAL that said nothing
// about length.
func TestOverlongSocketPathIsRejectedClearly(t *testing.T) {
	long := "/var/folders/36/tjdph2t965j8snz9_vkdnw0r0000gn/T/TestAutomationDrivesARunningApplication2442847269/001/a.sock"
	if len(long) <= maxSocketPathLen {
		t.Fatalf("fixture is %d bytes, expected it to exceed %d", len(long), maxSocketPathLen)
	}
	_, err := Listen(long)
	if err == nil {
		t.Fatal("an overlong socket path was accepted")
	}
	if !strings.Contains(err.Error(), "over the") {
		t.Errorf("error does not explain the length limit: %v", err)
	}
}

// An application shutting down must not wait on its clients. A connected
// agent or test holds its connection open indefinitely, and Close used to
// wait for every connection to end on its own — so the application hung on
// exit for as long as anything was attached.
func TestCloseDoesNotWaitForConnectedClients(t *testing.T) {
	server, client, _ := startServer(t)
	if _, _, _, err := client.Hello(); err != nil {
		t.Fatalf("hello: %v", err)
	}

	closed := make(chan error, 1)
	go func() { closed <- server.Close() }()
	select {
	case <-closed:
	case <-time.After(3 * time.Second):
		t.Fatal("Close blocked on a connected client")
	}
	if _, _, _, err := client.Hello(); err == nil {
		t.Error("the client's connection survived Close")
	}
}
