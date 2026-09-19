package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/thebanri/limoni/automation"
	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

const secretToken = "hunter2-do-not-leak"

// fakeApp stands in for a running Limoni application behind a real automation
// server: a name field, a task list, an Add button, two Remove buttons with
// the same label, and a secret token field. Input changes the tree the way an
// application would — asynchronously, on the next frame.
type fakeApp struct {
	t      *testing.T
	server *automation.Server

	queue chan driver.Event

	mu      sync.Mutex
	name    string
	tasks   []string
	focus   string
	events  []driver.Event
	delayed bool // publish a "done" label a while after the first frame
	agree   bool // a checkbox that a click toggles
}

func startFakeApp(t *testing.T, socket string, policy automation.Policy) *fakeApp {
	t.Helper()
	// Only Linux, macOS and FreeBSD can report the connecting user; elsewhere
	// the server refuses every peer unless told otherwise.
	switch runtime.GOOS {
	case "linux", "darwin", "freebsd":
	default:
		policy.AllowUnverifiedPeers = true
	}
	server, err := automation.Listen(socket, automation.WithPolicy(policy))
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	a := &fakeApp{t: t, server: server, tasks: []string{"write tests"}, focus: "name", queue: make(chan driver.Event, 256)}
	stop := make(chan struct{})
	t.Cleanup(func() {
		close(stop)
		_ = server.Close()
	})
	go a.loop(stop)
	a.publish()
	return a
}

func (a *fakeApp) publish() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.publishLocked()
}

func (a *fakeApp) publishLocked() {
	focused := func(id string) accessibility.NodeState {
		if a.focus == id {
			return accessibility.StateFocused
		}
		return 0
	}
	items := make([]accessibility.AccessibilityNode, 0, len(a.tasks))
	for i, task := range a.tasks {
		items = append(items, accessibility.AccessibilityNode{
			ID: fmt.Sprintf("task-%d", i), Role: accessibility.RoleListItem, Label: task,
			Bounds: cell.NewRect(0, uint16(2+i), 30, 1),
		})
	}
	tree := []accessibility.AccessibilityNode{
		{ID: "name", Role: accessibility.RoleInput, Label: "Task name", Value: a.name, State: focused("name"), Bounds: cell.NewRect(0, 0, 30, 1)},
		{ID: "tasks", Role: accessibility.RoleList, Label: "Tasks", SetSize: len(a.tasks), Position: 1, Bounds: cell.NewRect(0, 2, 30, 10), Children: items},
		{ID: "add", Role: accessibility.RoleButton, Label: "Add", Bounds: cell.NewRect(40, 0, 7, 1)},
		{ID: "remove-1", Role: accessibility.RoleButton, Label: "Remove", Bounds: cell.NewRect(40, 2, 8, 1)},
		{ID: "remove-2", Role: accessibility.RoleButton, Label: "Remove", Bounds: cell.NewRect(40, 3, 8, 1)},
		{ID: "token", Role: accessibility.RoleInput, Label: "API token", Value: secretToken, State: accessibility.StateSensitive | focused("token"), Bounds: cell.NewRect(0, 14, 30, 1)},
	}
	agreeState := accessibility.NodeState(0)
	if a.agree {
		agreeState = accessibility.StateChecked
	}
	tree = append(tree, accessibility.AccessibilityNode{ID: "agree", Role: accessibility.RoleCheckbox, Label: "Agree", State: agreeState, Bounds: cell.NewRect(0, 18, 10, 1)})
	if a.delayed {
		tree = append(tree, accessibility.AccessibilityNode{ID: "sync", Role: accessibility.RoleProgress, Label: "Sync done", Bounds: cell.NewRect(0, 16, 20, 1)})
	}
	a.server.Publish(automation.Snapshot{
		Tree: tree, Screen: "screen text", Focused: a.focus, Width: 80, Height: 24,
		Injector: a.inject,
	})
}

// inject queues input for the application's loop. The socket acknowledges it
// before anything is redrawn, as with a real application.
func (a *fakeApp) inject(ev driver.Event) {
	a.mu.Lock()
	a.events = append(a.events, ev)
	a.mu.Unlock()
	a.queue <- ev
}

// loop handles one event per frame, in order, each frame a little later than
// the last — so typed text arrives as a run of redraws, the way it does in a
// real application.
func (a *fakeApp) loop(stop chan struct{}) {
	for {
		select {
		case <-stop:
			return
		case ev := <-a.queue:
			time.Sleep(8 * time.Millisecond)
			if ev.Type == driver.EventKey && ev.Key.Type == driver.KeyEsc {
				_ = a.server.Close() // Esc quits, as in a real application
				return
			}
			a.mu.Lock()
			switch {
			case ev.Type == driver.EventKey && ev.Key.Type == driver.KeyRune && a.focus == "name":
				a.name += string(ev.Key.Ch)
			case ev.Type == driver.EventMouse && ev.Mouse.Y == 18:
				a.agree = !a.agree
			case ev.Type == driver.EventMouse && ev.Mouse.X >= 40 && ev.Mouse.Y == 0:
				a.tasks = append(a.tasks, a.name)
				a.name = ""
			}
			a.publishLocked()
			a.mu.Unlock()
		}
	}
}

// mcpClient talks to the bridge over pipes, exactly as an MCP host does over
// the process's stdin and stdout.
type mcpClient struct {
	t      *testing.T
	in     *io.PipeWriter
	out    *bufio.Reader
	nextID int
	done   chan error
}

func startBridge(t *testing.T, socket string) *mcpClient {
	t.Helper()
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	c := &mcpClient{t: t, in: inW, out: bufio.NewReader(outR), done: make(chan error, 1)}
	go func() {
		err := run([]string{"-socket", socket, "-settle", "400ms"}, inR, outW, io.Discard)
		_ = outW.Close()
		c.done <- err
	}()
	t.Cleanup(func() {
		_ = inW.Close()
		select {
		case <-c.done:
		case <-time.After(5 * time.Second):
			t.Error("bridge did not exit after stdin closed")
		}
	})
	return c
}

func (c *mcpClient) send(v any) {
	c.t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		c.t.Fatal(err)
	}
	if _, err := c.in.Write(append(data, '\n')); err != nil {
		c.t.Fatalf("write to bridge: %v", err)
	}
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error"`
}

func (c *mcpClient) read() response {
	c.t.Helper()
	type result struct {
		line []byte
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		line, err := c.out.ReadBytes('\n')
		ch <- result{line, err}
	}()
	select {
	case r := <-ch:
		if r.err != nil {
			c.t.Fatalf("read from bridge: %v", r.err)
		}
		if strings.Count(string(r.line), "\n") != 1 {
			c.t.Fatalf("message is not exactly one line: %q", r.line)
		}
		var resp response
		if err := json.Unmarshal(r.line, &resp); err != nil {
			c.t.Fatalf("decode %q: %v", r.line, err)
		}
		if resp.JSONRPC != "2.0" {
			c.t.Fatalf("jsonrpc = %q in %q", resp.JSONRPC, r.line)
		}
		return resp
	case <-time.After(10 * time.Second):
		c.t.Fatal("timed out waiting for the bridge to answer")
		return response{}
	}
}

func (c *mcpClient) request(method string, params any) response {
	c.t.Helper()
	c.nextID++
	id := c.nextID
	c.send(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
	resp := c.read()
	if string(resp.ID) != fmt.Sprint(id) {
		c.t.Fatalf("response id = %s, want %d", resp.ID, id)
	}
	return resp
}

type callResult struct {
	Content []textContent `json:"content"`
	IsError bool          `json:"isError"`
}

func (c *mcpClient) call(name string, args map[string]any) (string, bool) {
	c.t.Helper()
	resp := c.request("tools/call", map[string]any{"name": name, "arguments": args})
	if resp.Error != nil {
		c.t.Fatalf("tools/call %s: JSON-RPC error %d %s", name, resp.Error.Code, resp.Error.Message)
	}
	var result callResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		c.t.Fatalf("decode tool result: %v", err)
	}
	if len(result.Content) != 1 || result.Content[0].Type != "text" {
		c.t.Fatalf("content = %+v, want one text block", result.Content)
	}
	return result.Content[0].Text, result.IsError
}

func (c *mcpClient) mustCall(name string, args map[string]any) string {
	c.t.Helper()
	text, isError := c.call(name, args)
	if isError {
		c.t.Fatalf("%s failed: %s", name, text)
	}
	return text
}

func socketPath(t *testing.T) string {
	t.Helper()
	// Short on purpose: t.TempDir() can exceed the 104-byte sockaddr_un limit
	// on macOS.
	dir, err := os.MkdirTemp("", "lmcp")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return filepath.Join(dir, "a.sock")
}

var inputPolicy = automation.Policy{AllowInput: true}

func TestInitializeNegotiatesTheProtocolVersion(t *testing.T) {
	c := startBridge(t, socketPath(t))
	for _, tc := range []struct{ requested, want string }{
		{"2025-06-18", "2025-06-18"},
		{"2024-11-05", "2024-11-05"},
		{"1999-01-01", supportedVersions[0]},
	} {
		resp := c.request("initialize", map[string]any{
			"protocolVersion": tc.requested, "capabilities": map[string]any{},
			"clientInfo": map[string]any{"name": "test", "version": "1"},
		})
		var result struct {
			ProtocolVersion string `json:"protocolVersion"`
			Capabilities    struct {
				Tools *struct{} `json:"tools"`
			} `json:"capabilities"`
			ServerInfo struct {
				Name string `json:"name"`
			} `json:"serverInfo"`
			Instructions string `json:"instructions"`
		}
		if err := json.Unmarshal(resp.Result, &result); err != nil {
			t.Fatal(err)
		}
		if result.ProtocolVersion != tc.want {
			t.Errorf("requested %s, got %s, want %s", tc.requested, result.ProtocolVersion, tc.want)
		}
		if result.Capabilities.Tools == nil || result.ServerInfo.Name != "limoni" || result.Instructions == "" {
			t.Errorf("initialize result incomplete: %s", resp.Result)
		}
	}
}

func TestToolsAreListedWithSchemasAndHonestHints(t *testing.T) {
	c := startBridge(t, socketPath(t))
	resp := c.request("tools/list", map[string]any{})
	var result struct {
		Tools []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			InputSchema struct {
				Type       string                     `json:"type"`
				Properties map[string]json.RawMessage `json:"properties"`
			} `json:"inputSchema"`
			Annotations struct {
				ReadOnly    bool `json:"readOnlyHint"`
				Destructive bool `json:"destructiveHint"`
			} `json:"annotations"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatal(err)
	}
	readOnly := map[string]bool{}
	for _, tool := range result.Tools {
		if tool.InputSchema.Type != "object" || tool.InputSchema.Properties == nil || tool.Description == "" {
			t.Errorf("%s: incomplete descriptor", tool.Name)
		}
		if tool.Annotations.ReadOnly == tool.Annotations.Destructive {
			t.Errorf("%s: readOnly and destructive hints agree", tool.Name)
		}
		readOnly[tool.Name] = tool.Annotations.ReadOnly
	}
	want := map[string]bool{
		"status": true, "tree": true, "find": true, "screen": true, "wait_for": true,
		"click": false, "press_key": false, "type_text": false,
	}
	if len(readOnly) != len(want) {
		t.Errorf("tools = %v, want %v", readOnly, want)
	}
	for name, ro := range want {
		if got, ok := readOnly[name]; !ok || got != ro {
			t.Errorf("%s: listed=%t readOnly=%t, want readOnly=%t", name, ok, got, ro)
		}
	}
}

// The whole point: an agent fills in a field and presses a button by meaning,
// and sees the result of its own action in the same call.
func TestAnAgentCanFillAFormAndSeeTheResult(t *testing.T) {
	socket := socketPath(t)
	app := startFakeApp(t, socket, automation.Policy{AllowInput: true, ExposeInputValues: true})
	c := startBridge(t, socket)

	tree := c.mustCall("tree", nil)
	for _, want := range []string{`input#name "Task name"`, `button#add "Add"`, `list-item#task-0 "write tests"`} {
		if !strings.Contains(tree, want) {
			t.Errorf("tree lacks %s:\n%s", want, tree)
		}
	}

	typed := c.mustCall("type_text", map[string]any{"text": "ship it"})
	if !strings.Contains(typed, `value="ship it"`) {
		t.Errorf("type_text did not return the redrawn tree:\n%s", typed)
	}

	clicked := c.mustCall("click", map[string]any{"role": "button", "label": "Add"})
	if !strings.Contains(clicked, `clicked button#add "Add"; the application redrew`) {
		t.Errorf("click summary wrong:\n%s", clicked)
	}
	if !strings.Contains(clicked, `list-item#task-1 "ship it"`) {
		t.Errorf("click did not return the tree after the redraw:\n%s", clicked)
	}

	app.mu.Lock()
	defer app.mu.Unlock()
	if got := len(app.events); got != len("ship it")+1 {
		t.Errorf("application received %d events, want %d", got, len("ship it")+1)
	}
}

func TestInputWithNoVisibleEffectSaysSo(t *testing.T) {
	socket := socketPath(t)
	app := startFakeApp(t, socket, inputPolicy)
	c := startBridge(t, socket)
	app.mu.Lock()
	app.focus = "" // keys go nowhere
	app.mu.Unlock()
	app.publish()

	text := c.mustCall("press_key", map[string]any{"key": "x", "ctrl": true})
	if !strings.Contains(text, "pressed ctrl+x") || !strings.Contains(text, "did not change") {
		t.Errorf("unchanged tree not reported:\n%s", text)
	}
}

// With the default policy an input's value is withheld, so typing looks like
// it did nothing. The agent has to be told why, or it types the text again.
func TestTypingIntoAHiddenValueExplainsThePolicy(t *testing.T) {
	socket := socketPath(t)
	startFakeApp(t, socket, inputPolicy)
	c := startBridge(t, socket)

	text := c.mustCall("type_text", map[string]any{"text": "hello"})
	if !strings.Contains(text, "ExposeInputValues is off") {
		t.Errorf("hidden input values not explained:\n%s", text)
	}
}

func TestTypingIntoASecretFieldExplainsWhyNothingShows(t *testing.T) {
	socket := socketPath(t)
	app := startFakeApp(t, socket, automation.Policy{AllowInput: true, ExposeInputValues: true})
	c := startBridge(t, socket)
	app.mu.Lock()
	app.focus = "token"
	app.mu.Unlock()
	app.publish()

	text := c.mustCall("type_text", map[string]any{"text": "hello"})
	if !strings.Contains(text, "focused field is secret") {
		t.Errorf("secret field not explained:\n%s", text)
	}
}

func TestSecretsNeverReachTheAgent(t *testing.T) {
	socket := socketPath(t)
	startFakeApp(t, socket, automation.Policy{AllowInput: true, ExposeInputValues: true, ExposeScreen: true})
	c := startBridge(t, socket)

	outputs := []string{
		c.mustCall("tree", nil),
		c.mustCall("tree", map[string]any{"format": "json"}),
		c.mustCall("find", map[string]any{"id": "token"}),
		c.mustCall("type_text", map[string]any{"text": "a"}),
	}
	// Probing for the value must not work as an oracle either.
	probe := c.mustCall("find", map[string]any{"value": secretToken})
	if !strings.Contains(probe, "nothing matches") {
		t.Errorf("selecting by the secret value matched something:\n%s", probe)
	}
	for _, out := range outputs {
		if strings.Contains(out, secretToken) {
			t.Fatalf("secret leaked:\n%s", out)
		}
	}
	if !strings.Contains(outputs[0], "state=sensitive") {
		t.Errorf("token field not marked sensitive:\n%s", outputs[0])
	}
}

func TestAmbiguousAndMisspelledSelectorsAreExplained(t *testing.T) {
	socket := socketPath(t)
	startFakeApp(t, socket, inputPolicy)
	c := startBridge(t, socket)

	text, isError := c.call("click", map[string]any{"label": "Remove"})
	if !isError || !strings.Contains(text, "matches 2 nodes") || !strings.Contains(text, "nth") {
		t.Errorf("ambiguous click: isError=%t %q", isError, text)
	}

	text, isError = c.call("click", map[string]any{"name": "Add"})
	if !isError || !strings.Contains(text, `unknown field "name"`) {
		t.Errorf("misspelled field: isError=%t %q", isError, text)
	}

	text, isError = c.call("click", map[string]any{})
	if !isError || !strings.Contains(text, "needs a selector") {
		t.Errorf("empty selector: isError=%t %q", isError, text)
	}

	text = c.mustCall("click", map[string]any{"label": "Remove", "nth": -1})
	if !strings.Contains(text, "button#remove-2") {
		t.Errorf("nth=-1 did not pick the last match:\n%s", text)
	}
}

func TestPolicyRefusalsReachTheAgentVerbatim(t *testing.T) {
	socket := socketPath(t)
	startFakeApp(t, socket, automation.Policy{}) // closed by default
	c := startBridge(t, socket)

	status := c.mustCall("status", nil)
	for _, want := range []string{"screen 80x24", "focused name", "input allowed: false", "screen text exposed: false"} {
		if !strings.Contains(status, want) {
			t.Errorf("status lacks %q:\n%s", want, status)
		}
	}
	if text, isError := c.call("screen", nil); !isError || !strings.Contains(text, "Policy.ExposeScreen") {
		t.Errorf("screen: isError=%t %q", isError, text)
	}
	if text, isError := c.call("type_text", map[string]any{"text": "x"}); !isError || !strings.Contains(text, "AllowInput") {
		t.Errorf("type_text: isError=%t %q", isError, text)
	}
}

func TestWaitForSeesAsynchronousChangesAndTimesOut(t *testing.T) {
	socket := socketPath(t)
	app := startFakeApp(t, socket, inputPolicy)
	c := startBridge(t, socket)

	go func() {
		time.Sleep(100 * time.Millisecond)
		app.mu.Lock()
		app.delayed = true
		app.mu.Unlock()
		app.publish()
	}()
	text := c.mustCall("wait_for", map[string]any{"label_contains": "done", "timeout_ms": 3000})
	if !strings.Contains(text, `progress#sync "Sync done"`) {
		t.Errorf("wait_for result:\n%s", text)
	}

	start := time.Now()
	text, isError := c.call("wait_for", map[string]any{"label": "never", "timeout_ms": 100})
	if !isError || !strings.Contains(text, "within 100ms") {
		t.Errorf("timeout: isError=%t %q", isError, text)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("a 100ms wait took %s", elapsed)
	}
}

// A cancelled request gets no response, and does not hold up the ones after it.
func TestCancellationEndsALongWait(t *testing.T) {
	socket := socketPath(t)
	startFakeApp(t, socket, inputPolicy)
	c := startBridge(t, socket)

	c.send(map[string]any{"jsonrpc": "2.0", "id": "long", "method": "tools/call",
		"params": map[string]any{"name": "wait_for", "arguments": map[string]any{"label": "never", "timeout_ms": 60000}}})
	time.Sleep(50 * time.Millisecond)
	c.send(map[string]any{"jsonrpc": "2.0", "method": "notifications/cancelled", "params": map[string]any{"requestId": "long"}})

	start := time.Now()
	resp := c.request("ping", nil)
	if resp.Error != nil || string(resp.Result) != "{}" {
		t.Errorf("ping = %s %v", resp.Result, resp.Error)
	}
	if time.Since(start) > 2*time.Second {
		t.Error("ping was held up by the cancelled wait")
	}
	// If the cancelled call had answered, its response would arrive before this
	// one.
	resp = c.request("ping", nil)
	if string(resp.ID) != fmt.Sprint(c.nextID) {
		t.Errorf("got a response for %s after cancelling it", resp.ID)
	}
}

func TestProtocolErrors(t *testing.T) {
	c := startBridge(t, socketPath(t))

	// Notifications are never answered: the ping's response must come first.
	c.send(map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"})
	if resp := c.request("ping", nil); resp.Error != nil {
		t.Errorf("ping after notification: %v", resp.Error)
	}

	if resp := c.request("resources/list", nil); resp.Error == nil || resp.Error.Code != codeMethodNotFound {
		t.Errorf("unknown method: %+v", resp.Error)
	}
	if resp := c.request("tools/call", map[string]any{"name": "rm_rf"}); resp.Error == nil || resp.Error.Code != codeInvalidParams {
		t.Errorf("unknown tool: %+v", resp.Error)
	}

	if _, err := c.in.Write([]byte("{not json\n")); err != nil {
		t.Fatal(err)
	}
	if resp := c.read(); resp.Error == nil || resp.Error.Code != codeParseError || string(resp.ID) != "null" {
		t.Errorf("parse error: id=%s %+v", resp.ID, resp.Error)
	}
	if _, err := c.in.Write([]byte(`[{"jsonrpc":"2.0","id":9,"method":"ping"}]` + "\n")); err != nil {
		t.Fatal(err)
	}
	if resp := c.read(); resp.Error == nil || resp.Error.Code != codeInvalidRequest {
		t.Errorf("batch: %+v", resp.Error)
	}
}

// The bridge is configured once, but the application comes and goes: it may
// not be running when the agent starts, and it restarts during development.
func TestTheApplicationCanStartLateAndRestart(t *testing.T) {
	socket := socketPath(t)
	c := startBridge(t, socket)

	text, isError := c.call("tree", nil)
	if !isError || !strings.Contains(text, "no Limoni application is listening") || !strings.Contains(text, "limoni_debug") {
		t.Errorf("before start: isError=%t %q", isError, text)
	}

	first := startFakeApp(t, socket, inputPolicy)
	if !strings.Contains(c.mustCall("tree", nil), `button#add "Add"`) {
		t.Error("tree after the application started")
	}

	_ = first.server.Close()
	startFakeApp(t, socket, inputPolicy)
	if !strings.Contains(c.mustCall("tree", nil), `button#add "Add"`) {
		t.Error("read-only call did not recover after the application restarted")
	}
}

func TestRunRequiresASocket(t *testing.T) {
	t.Setenv("LIMONI_AUTOMATION_SOCKET", "")
	var stderr strings.Builder
	err := run(nil, strings.NewReader(""), io.Discard, &stderr)
	if err == nil || !strings.Contains(err.Error(), "no socket") {
		t.Errorf("err = %v", err)
	}
	if !strings.Contains(stderr.String(), "limoni_debug") {
		t.Errorf("usage does not explain how to enable the application side:\n%s", stderr.String())
	}
}

// When the host closes stdin it is done with the server. A tool call still in
// flight must be abandoned, not waited out — a wait_for can run for a minute.
func TestClosingStdinAbandonsCallsInFlight(t *testing.T) {
	socket := socketPath(t)
	startFakeApp(t, socket, inputPolicy)
	c := startBridge(t, socket)

	c.send(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call",
		"params": map[string]any{"name": "wait_for", "arguments": map[string]any{"label": "never", "timeout_ms": 60000}}})
	time.Sleep(50 * time.Millisecond)
	_ = c.in.Close()
	go func() { _, _ = io.Copy(io.Discard, c.out) }()

	select {
	case <-c.done:
		c.done <- nil // for the cleanup that also waits on it
	case <-time.After(3 * time.Second):
		t.Fatal("bridge kept running a call after stdin closed")
	}
}

// Esc or a Quit button ends the application. The key was delivered and did
// what it should; reporting that as a failed call would send the agent looking
// for a problem that is not there.
func TestInputThatEndsTheApplicationIsReportedAsDone(t *testing.T) {
	socket := socketPath(t)
	startFakeApp(t, socket, inputPolicy)
	c := startBridge(t, socket)

	text, isError := c.call("press_key", map[string]any{"key": "esc"})
	if isError || !strings.Contains(text, "pressed esc; the application then closed its automation socket") {
		t.Errorf("isError=%t %q", isError, text)
	}
}

// click with ensure is safe to repeat: the second call finds the checkbox
// already checked and clicks nothing, where a plain click would uncheck it.
func TestClickWithEnsureIsIdempotent(t *testing.T) {
	socket := socketPath(t)
	app := startFakeApp(t, socket, automation.Policy{AllowInput: true})
	c := startBridge(t, socket)

	first := c.mustCall("click", map[string]any{"id": "agree", "ensure": "checked"})
	if !strings.Contains(first, "clicked checkbox#agree") || strings.Contains(first, "still not") {
		t.Errorf("first ensure-click:\n%s", first)
	}
	second := c.mustCall("click", map[string]any{"id": "agree", "ensure": "checked"})
	if !strings.Contains(second, "already checked; nothing was clicked") {
		t.Errorf("second ensure-click clicked again:\n%s", second)
	}
	app.mu.Lock()
	agree, clicks := app.agree, len(app.events)
	app.mu.Unlock()
	if !agree || clicks != 1 {
		t.Fatalf("agree=%v after %d clicks, want checked after 1", agree, clicks)
	}

	if text, isError := c.call("click", map[string]any{"id": "agree", "ensure": "on"}); !isError || !strings.Contains(text, "ensure must be") {
		t.Errorf("bad ensure value accepted: %s", text)
	}
}
