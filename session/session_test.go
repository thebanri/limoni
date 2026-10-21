package session

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/engine"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/widgets"
)

// loginModel is a small but realistic declarative application: a text field,
// a secret field, a list, focus moved by Tab, and application messages.
type loginModel struct {
	items    []string
	list     *widgets.ListState
	user     *widgets.TextInputState
	password *widgets.TextInputState
	focus    int // 0 user, 1 password, 2 list
	status   string

	// Knobs for the tests, set through the constructor.
	bugAtStatus  bool // renders a wrong status once a status message arrives
	panicOnKey   rune
	readsTheTime bool
}

// statusMsg is an application message, the kind a command produces.
type statusMsg struct{ Text string }

// loginMsg carries a secret the application needs, but that must not reach a
// recording.
type loginMsg struct {
	User     string
	Password string
}

var focusOrder = []string{"user", "password", "items"}

func newLogin() *loginModel {
	list := widgets.NewListState()
	list.Selected = 0
	return &loginModel{
		items:    []string{"alpha", "beta", "gamma"},
		list:     list,
		user:     widgets.NewTextInputState(),
		password: widgets.NewTextInputState(),
	}
}

func (m *loginModel) Init() []engine.Cmd { return nil }

func (m *loginModel) Update(msg engine.Msg) engine.UpdateResult {
	switch msg := msg.(type) {
	case engine.KeyPressMsg:
		if m.panicOnKey != 0 && msg.Key.Type == driver.KeyRune && msg.Key.Ch == m.panicOnKey {
			panic("boom on " + string(msg.Key.Ch))
		}
		switch msg.Key.Type {
		case driver.KeyTab:
			m.focus = (m.focus + 1) % len(focusOrder)
		case driver.KeyArrowDown:
			m.list.Selected = (m.list.Selected + 1) % len(m.items)
		case driver.KeyRune:
			switch m.focus {
			case 0:
				m.user.Text = append(m.user.Text, msg.Key.Ch)
				m.user.Cursor = len(m.user.Text)
			case 1:
				m.password.Text = append(m.password.Text, msg.Key.Ch)
				m.password.Cursor = len(m.password.Text)
			}
		}
	case statusMsg:
		m.status = msg.Text
	case loginMsg:
		m.status = "login " + msg.User
	}
	return engine.UpdateResult{Redraw: true}
}

func (m *loginModel) View(f *terminal.Frame) {
	if f.FocusManager != nil {
		f.FocusManager.SetFocused(focusOrder[m.focus])
	}
	f.RenderWidget(&widgets.TextInput{ID: "user", State: m.user, Placeholder: "User"}, cell.NewRect(0, 0, 30, 1))
	f.RenderWidget(&widgets.TextInput{ID: "password", State: m.password, Placeholder: "Password", Secret: true}, cell.NewRect(0, 1, 30, 1))
	f.RenderWidget(&widgets.List{ID: "items", Items: m.items, State: m.list}, cell.NewRect(0, 2, 30, 4))
	status := m.status
	if m.bugAtStatus && status != "" {
		status = "WRONG " + status
	}
	if m.readsTheTime {
		status = time.Now().Format(time.RFC3339Nano) // limonivet:allow the test proves replay catches this
	}
	f.RenderWidget(&widgets.Paragraph{ID: "status", Text: status}, cell.NewRect(0, 6, 30, 1))
}

// stepObserver forwards to the recorder and signals once a message has been
// handed to Update. Message runs under the model lock, so a Draw issued after
// the signal waits for Update to finish, and each frame reflects exactly the
// messages sent before it.
type stepObserver struct {
	*Recorder
	stepped chan uint64
}

func (o stepObserver) Message(step uint64, msg engine.Msg) {
	o.Recorder.Message(step, msg)
	o.stepped <- step
}

// liveSession drives a real engine.Program with a recorder attached.
type liveSession struct {
	t       *testing.T
	program *engine.Program
	term    *terminal.Terminal
	obs     stepObserver
	cancel  context.CancelFunc
	done    chan error
	path    string
}

func startRecording(t *testing.T, model engine.Model, policy Policy) *liveSession {
	t.Helper()
	path := filepath.Join(t.TempDir(), "run.limoni")
	rec, err := Create(path, model, 40, 10, policy)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	obs := stepObserver{Recorder: rec, stepped: make(chan uint64, 64)}
	program := engine.New(engine.WithModel(model), engine.WithObserver(obs), engine.WithPanicHandler(func(any) {}))

	backend := driver.NewPortableBackend(driver.NewMemoryTerminalIO(nil, 40, 10))
	term, err := terminal.New(backend)
	if err != nil {
		t.Fatalf("terminal: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	s := &liveSession{t: t, program: program, term: term, obs: obs, cancel: cancel, done: make(chan error, 1), path: path}
	go func() { s.done <- program.Run(ctx) }()
	s.draw()
	return s
}

func (s *liveSession) send(msg engine.Msg) {
	s.t.Helper()
	if err := s.program.Send(context.Background(), msg); err != nil {
		s.t.Fatalf("send: %v", err)
	}
	select {
	case <-s.obs.stepped:
	case <-time.After(3 * time.Second):
		s.t.Fatal("message was never handled")
	}
	s.draw()
}

func (s *liveSession) key(k driver.KeyEvent) { s.send(engine.KeyPressMsg{Key: k}) }

func (s *liveSession) typeText(text string) {
	for _, r := range text {
		s.key(driver.KeyEvent{Type: driver.KeyRune, Ch: r})
	}
}

func (s *liveSession) draw() {
	s.t.Helper()
	if err := s.program.Draw(s.term); err != nil {
		s.t.Fatalf("draw: %v", err)
	}
}

func (s *liveSession) finish() []byte {
	s.t.Helper()
	s.cancel()
	<-s.done
	if err := s.obs.Recorder.Close(); err != nil {
		s.t.Fatalf("Close: %v", err)
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		s.t.Fatalf("read: %v", err)
	}
	return data
}

var (
	tab  = driver.KeyEvent{Type: driver.KeyTab}
	down = driver.KeyEvent{Type: driver.KeyArrowDown}
)

func init() {
	Register[statusMsg]()
	RegisterRedacted(func(m loginMsg) loginMsg { m.Password = ""; return m })
}

func constructor(configure func(*loginModel)) func() engine.Model {
	return func() engine.Model {
		m := newLogin()
		if configure != nil {
			configure(m)
		}
		return m
	}
}

func TestRecordedSessionReplaysAndVerifies(t *testing.T) {
	s := startRecording(t, newLogin(), Policy{})
	s.typeText("ada")
	s.key(tab)
	s.typeText("hunter2")
	s.key(tab)
	s.key(down)
	s.send(statusMsg{Text: "synced"})
	data := s.finish()

	report, err := ReplayBytes(data, constructor(nil), ReplayOptions{})
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}
	if !report.Complete {
		t.Error("a closed recording was reported incomplete")
	}
	if !report.Verified() {
		t.Fatalf("replay of an unchanged model did not verify: %v", report.Divergence)
	}
	if report.Steps != 14 { // 3 + tab + 7 + tab + down + status
		t.Errorf("replayed %d steps, want 14", report.Steps)
	}
	if report.Frames == 0 {
		t.Error("no frames were verified")
	}
	if report.RedactedSteps != 10 {
		t.Errorf("redacted %d steps, want the 10 typed characters", report.RedactedSteps)
	}
}

func TestReplayNamesTheStepWhereBehaviourChanged(t *testing.T) {
	s := startRecording(t, newLogin(), Policy{})
	s.key(down)
	s.key(down)
	s.send(statusMsg{Text: "synced"}) // step 3
	s.key(down)
	data := s.finish()

	report, err := ReplayBytes(data, constructor(func(m *loginModel) { m.bugAtStatus = true }), ReplayOptions{})
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}
	if report.Divergence == nil {
		t.Fatal("a changed model replayed as verified")
	}
	if report.Divergence.Step != 3 {
		t.Errorf("diverged at step %d, want 3, where the status message arrived", report.Divergence.Step)
	}
	if !strings.Contains(report.Divergence.Got, "WRONG") {
		t.Errorf("divergence does not show the replayed tree: %s", report.Divergence)
	}
	if report.Divergence.AfterRedactedInput {
		t.Error("no text was typed, yet the divergence is blamed on redaction")
	}
}

func TestReplayReproducesACrash(t *testing.T) {
	crashing := constructor(func(m *loginModel) { m.panicOnKey = 'q' })
	s := startRecording(t, crashing(), Policy{})
	s.key(tab)
	s.key(tab) // focus on the list, so the key is not typed into a field
	s.key(driver.KeyEvent{Type: driver.KeyRune, Ch: 'q', Ctrl: true})
	data := s.finish()

	report, err := ReplayBytes(data, crashing, ReplayOptions{})
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}
	if !report.PanicRecorded || report.PanicStep != 3 {
		t.Fatalf("recorded panic = %v at step %d, want step 3", report.PanicRecorded, report.PanicStep)
	}
	if !report.PanicReproduced {
		t.Error("the crash did not reproduce on replay")
	}
	if !report.Verified() {
		t.Errorf("a reproduced crash should verify: %v", report.Divergence)
	}

	fixed, err := ReplayBytes(data, constructor(nil), ReplayOptions{})
	if err != nil {
		t.Fatalf("Replay against the fixed model: %v", err)
	}
	if fixed.PanicReproduced || fixed.Verified() {
		t.Error("replaying against the fixed model still claims the crash reproduces")
	}
}

func TestReplayDetectsANondeterministicView(t *testing.T) {
	clocky := constructor(func(m *loginModel) { m.readsTheTime = true })
	s := startRecording(t, clocky(), Policy{})
	s.key(down)
	data := s.finish()
	time.Sleep(2 * time.Millisecond)

	report, err := ReplayBytes(data, clocky, ReplayOptions{})
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}
	if report.Divergence == nil {
		t.Fatal("a View that reads the clock replayed as verified")
	}
}

// With the default policy nothing a user typed reaches the file, and a message
// registered with a redactor is written without its secret.
func TestDefaultPolicyWritesNoTypedText(t *testing.T) {
	s := startRecording(t, newLogin(), Policy{})
	s.typeText("ada-lovelace")
	s.key(tab)
	s.typeText("hunter2")
	s.send(loginMsg{User: "ada", Password: "correct-horse"})
	data := s.finish()

	for _, secret := range []string{"lovelace", "hunter2", "correct-horse"} {
		if bytes.Contains(data, []byte(secret)) {
			t.Errorf("%q was written to the recording", secret)
		}
	}
	info, err := os.Stat(s.path)
	if err != nil {
		t.Fatal(err)
	}
	if runtimeSupportsModeBits() && info.Mode().Perm() != 0o600 {
		t.Errorf("recording permissions = %v, want 0600", info.Mode().Perm())
	}
}

// With RecordText set, ordinary typing is kept, but the secret field is still
// redacted — including characters typed in the window after Tab moved focus
// and before any frame showed where it went.
func TestRecordTextStillProtectsTheSecretField(t *testing.T) {
	path := filepath.Join(t.TempDir(), "race.limoni")
	rec, err := Create(path, newLogin(), 40, 10, Policy{RecordText: true})
	if err != nil {
		t.Fatal(err)
	}

	// The recorder is driven directly here, to control frame timing exactly.
	frame := func(focusedID string) {
		rec.Frame(0, []accessibility.Node{
			{ID: "user", Role: accessibility.RoleInput, State: stateIf(focusedID == "user")},
			{ID: "password", Role: accessibility.RoleInput, State: accessibility.StateSensitive | stateIf(focusedID == "password")},
		})
	}
	step := uint64(0)
	typeRune := func(r rune) {
		step++
		rec.Message(step, engine.KeyPressMsg{Key: driver.KeyEvent{Type: driver.KeyRune, Ch: r}})
	}

	frame("user")
	for _, r := range "visible" {
		typeRune(r)
	}
	step++
	rec.Message(step, engine.KeyPressMsg{Key: tab}) // focus moves; no frame yet
	for _, r := range "SECRET" {                    // typed before the next frame
		typeRune(r)
	}
	frame("password")
	for _, r := range "MORE" {
		typeRune(r)
	}
	// A command result that could have moved focus, then more typing before
	// any frame: still redacted.
	step++
	rec.Message(step, statusMsg{Text: "focus-password"})
	frame("user")
	step++
	rec.Message(step, statusMsg{Text: "moved"})
	for _, r := range "AFTER" {
		typeRune(r)
	}
	if err := rec.Close(); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)

	text := typedText(t, data)
	if !strings.HasPrefix(text, "visible") {
		t.Errorf("ordinary typing was not kept under RecordText: %q", text)
	}
	for _, leaked := range []string{"S", "E", "C", "R", "T", "M", "O", "A", "F"} {
		if strings.Contains(text, leaked) {
			t.Errorf("a character typed into or towards the secret field was recorded: %q", text)
			break
		}
	}
}

func stateIf(focused bool) accessibility.NodeState {
	if focused {
		return accessibility.StateFocused
	}
	return 0
}

// typedText extracts the characters of recorded key presses, in order.
func typedText(t *testing.T, data []byte) string {
	t.Helper()
	var b strings.Builder
	for _, line := range bytes.Split(data, []byte("\n"))[1:] {
		var rec record
		if json.Unmarshal(line, &rec) != nil || rec.Kind != kindMessage || !strings.HasSuffix(rec.Type, "engine.KeyPressMsg") {
			continue
		}
		var msg engine.KeyPressMsg
		if err := json.Unmarshal(rec.Data, &msg); err != nil {
			t.Fatal(err)
		}
		if msg.Key.Type == driver.KeyRune {
			b.WriteRune(msg.Key.Ch)
		}
	}
	return b.String()
}

func TestTamperedRecordingIsRejected(t *testing.T) {
	s := startRecording(t, newLogin(), Policy{})
	s.key(down)
	s.key(down)
	data := s.finish()

	tampered := bytes.Replace(data, []byte(`"step":2`), []byte(`"step":9`), 1)
	if bytes.Equal(tampered, data) {
		t.Fatal("test found nothing to tamper with")
	}
	_, err := ReplayBytes(tampered, constructor(nil), ReplayOptions{})
	if err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("tampered recording = %v, want a checksum error", err)
	}
}

func TestIncompleteRecordingNeedsExplicitPermission(t *testing.T) {
	s := startRecording(t, newLogin(), Policy{})
	s.key(down)
	s.key(down)
	data := s.finish()

	// Drop the trailer and leave half a line behind, as a crash mid-write would.
	lines := bytes.Split(bytes.TrimRight(data, "\n"), []byte("\n"))
	crashed := append(bytes.Join(lines[:len(lines)-1], []byte("\n")), []byte("\n{\"kind\":\"msg\",\"st")...)

	if _, err := ReplayBytes(crashed, constructor(nil), ReplayOptions{}); err == nil {
		t.Fatal("an incomplete recording replayed without AllowIncomplete")
	}
	report, err := ReplayBytes(crashed, constructor(nil), ReplayOptions{AllowIncomplete: true})
	if err != nil {
		t.Fatalf("Replay with AllowIncomplete: %v", err)
	}
	if report.Complete {
		t.Error("an incomplete recording was reported complete")
	}
	if !report.Verified() {
		t.Errorf("the intact part did not verify: %v", report.Divergence)
	}
}

func TestReplayRefusesTheWrongModel(t *testing.T) {
	s := startRecording(t, newLogin(), Policy{})
	s.key(down)
	data := s.finish()

	_, err := ReplayBytes(data, func() engine.Model { return otherModel{} }, ReplayOptions{})
	if err == nil || !strings.Contains(err.Error(), "model") {
		t.Fatalf("replay against a different model type = %v", err)
	}
}

func TestReplayRefusesAnotherRevisionUnlessAllowed(t *testing.T) {
	s := startRecording(t, newLogin(), Policy{})
	s.key(down)
	data := rewriteHeader(t, s.finish(), func(h *Header) { h.Revision = "0000000deadbeef" })

	if _, err := ReplayBytes(data, constructor(nil), ReplayOptions{}); err == nil || !strings.Contains(err.Error(), "revision") {
		t.Fatalf("revision mismatch = %v", err)
	}
	if _, err := ReplayBytes(data, constructor(nil), ReplayOptions{AllowRevisionMismatch: true}); err != nil {
		t.Fatalf("AllowRevisionMismatch: %v", err)
	}
}

// privateMsg is an application message nobody registered.
type privateMsg struct{ Token string }

// An unregistered application message is recorded by name only. Replaying
// fails loudly rather than skipping it and verifying something else.
func TestUnregisteredMessageIsOpaque(t *testing.T) {
	s := startRecording(t, newLogin(), Policy{})
	s.send(privateMsg{Token: "s3cr3t-token"})
	data := s.finish()

	if bytes.Contains(data, []byte("s3cr3t-token")) {
		t.Error("an unregistered message's content was written")
	}
	_, err := ReplayBytes(data, constructor(nil), ReplayOptions{})
	if err == nil || !strings.Contains(err.Error(), "unregistered") {
		t.Fatalf("replaying an opaque message = %v", err)
	}
}

func TestRecordingIsNeverOverwritten(t *testing.T) {
	path := filepath.Join(t.TempDir(), "existing.limoni")
	if err := os.WriteFile(path, []byte("an earlier recording"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Create(path, newLogin(), 40, 10, Policy{}); err == nil {
		t.Fatal("Create overwrote an existing file")
	}
	data, _ := os.ReadFile(path)
	if string(data) != "an earlier recording" {
		t.Error("the existing file was modified")
	}
}

func TestNotARecording(t *testing.T) {
	for _, data := range [][]byte{nil, []byte("hello\n"), []byte(`{"kind":"header","format":"other"}` + "\n")} {
		if _, err := ReplayBytes(data, constructor(nil), ReplayOptions{}); err == nil {
			t.Errorf("accepted %q as a recording", data)
		}
	}
}

func TestRegisterRejectsUnnamedTypes(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("registering an anonymous struct did not panic")
		}
	}()
	Register[struct{ X int }]()
}

type otherModel struct{}

func (otherModel) Init() []engine.Cmd                    { return nil }
func (otherModel) Update(engine.Msg) engine.UpdateResult { return engine.UpdateResult{} }
func (otherModel) View(*terminal.Frame)                  {}

// rewriteHeader edits the header and recomputes the trailer checksum, which is
// how a recording from another build legitimately differs.
func rewriteHeader(t *testing.T, data []byte, edit func(*Header)) []byte {
	t.Helper()
	lines := bytes.Split(bytes.TrimRight(data, "\n"), []byte("\n"))
	var h Header
	if err := json.Unmarshal(lines[0], &h); err != nil {
		t.Fatal(err)
	}
	edit(&h)
	lines[0], _ = json.Marshal(h)

	sum := sha256.New()
	for _, line := range lines[:len(lines)-1] {
		sum.Write(line)
		sum.Write([]byte("\n"))
	}
	var trailer record
	if err := json.Unmarshal(lines[len(lines)-1], &trailer); err != nil {
		t.Fatal(err)
	}
	trailer.SHA256 = hex.EncodeToString(sum.Sum(nil))
	lines[len(lines)-1], _ = json.Marshal(trailer)
	return append(bytes.Join(lines, []byte("\n")), '\n')
}
