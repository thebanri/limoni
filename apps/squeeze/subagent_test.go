package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/thebanri/limoni/core/driver"
)

// hookFile leaves a hook's input as subCmd does and hands its name over.
func hookFile(t *testing.T, r *racer, json string) {
	t.Helper()
	f, err := os.CreateTemp(r.subsDir, "h.")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(json); err != nil {
		t.Fatal(err)
	}
	f.Close()
	r.subEvent(filepath.Base(f.Name()))
	if _, err := os.Stat(f.Name()); err == nil {
		t.Error("the hook's file was left behind")
	}
}

// Claude Code's hooks, in the order it runs them (captured from 2.1.325):
// a subagent waited for, and one sent to the background.
func TestClaudesSubagentsAreFollowed(t *testing.T) {
	r := &racer{subsDir: t.TempDir()}
	hookFile(t, r, `{"hook_event_name":"PreToolUse","tool_name":"Agent","tool_use_id":"t1",
		"tool_input":{"description":"List files","prompt":"Run ls","subagent_type":"general-purpose"}}`)
	hookFile(t, r, `{"hook_event_name":"SubagentStart","agent_id":"a1","agent_type":"general-purpose",
		"transcript_path":"/x/s1.jsonl"}`)
	if s := r.subagents(); len(s) != 1 || !s[0].running() || s[0].id != "a1" || s[0].transcript != "/x/s1/subagents/agent-a1.jsonl" {
		t.Fatalf("after the start: %+v", s)
	}
	hookFile(t, r, `{"hook_event_name":"SubagentStop","agent_id":"a1","last_assistant_message":"go.mod main.go"}`)
	hookFile(t, r, `{"hook_event_name":"PostToolUse","tool_use_id":"t1",
		"tool_response":{"status":"completed","agentId":"a1","content":[{"type":"text","text":"noise"}]}}`)
	s := r.subagents()
	if len(s) != 1 || s[0].running() || s[0].result != "go.mod main.go" || s[0].desc != "List files" || s[0].prompt != "Run ls" {
		t.Fatalf("a finished subagent: %+v", s)
	}

	// In the background: PostToolUse before SubagentStart, and a helper of
	// Claude Code's own stopping in between, untyped and never started.
	hookFile(t, r, `{"hook_event_name":"PreToolUse","tool_use_id":"t2","tool_input":{"description":"Wait","subagent_type":"Explore"}}`)
	hookFile(t, r, `{"hook_event_name":"PostToolUse","tool_use_id":"t2","tool_response":{"isAsync":true,"agentId":"a2"}}`)
	hookFile(t, r, `{"hook_event_name":"SubagentStart","agent_id":"a2","agent_type":"Explore","transcript_path":"/x/s1.jsonl"}`)
	hookFile(t, r, `{"hook_event_name":"SubagentStop","agent_id":"a9","agent_type":""}`)
	hookFile(t, r, `{"hook_event_name":"SubagentStart","agent_id":"a8","agent_type":""}`)
	if s := r.subagents(); len(s) != 2 || s[1].transcript != "/x/s1/subagents/agent-a2.jsonl" {
		t.Fatalf("the background subagent: %+v", s)
	}
	r.asked.Store(true) // the main agent's turn ended
	r.lastOut.Store(time.Now().UnixNano())
	if !r.background() || r.state(time.Now()) != working {
		t.Error("a subagent in the background does not keep the racer working")
	}
	r.promptSent()
	if !r.background() {
		t.Error("a new prompt ended a background subagent")
	}
	hookFile(t, r, `{"hook_event_name":"SubagentStop","agent_id":"a2","last_assistant_message":"done"}`)
	if r.background() || r.state(time.Now()) != waiting {
		t.Error("the racer works on after its background subagent stopped")
	}
}

// A subagent whose turn was interrupted before its hooks ran ends with the
// next prompt; a name that is not a file in the racer's directory is not read.
func TestAPromptEndsAnInterruptedSubagent(t *testing.T) {
	r := &racer{subsDir: t.TempDir()}
	hookFile(t, r, `{"hook_event_name":"PreToolUse","tool_use_id":"t1","tool_input":{"description":"x"}}`)
	r.promptSent()
	if r.subagents()[0].running() {
		t.Error("still running after the next prompt")
	}
	outside := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(outside, []byte(`{"hook_event_name":"PreToolUse"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	r.subEvent("../" + filepath.Base(filepath.Dir(outside)) + "/secret")
	if len(r.subagents()) != 1 {
		t.Error("read a file outside the racer's directory")
	}
	if _, err := os.Stat(outside); err != nil {
		t.Error("removed a file outside the racer's directory")
	}
}

// A subagent's row under its racer's card opens what it was asked, what it
// does, from its transcript, and its answer.
func TestASubagentIsShown(t *testing.T) {
	u := newUI(t.TempDir(), "", "task", nil)
	u.notify, u.wake = func(string, string) {}, func() {}
	r := fake(t, `sleep 5`)
	r.subsDir = t.TempDir()
	u.racers = []*racer{r}
	transcript := filepath.Join(t.TempDir(), "s1", "subagents", "agent-a1.jsonl")
	if err := os.MkdirAll(filepath.Dir(transcript), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(transcript, []byte(`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"ls"}}]}}
{"type":"user","message":{"content":[{"type":"tool_result","content":"go.mod\nmain.go"}]}}
{"type":"assistant","message":{"content":[{"type":"text","text":"Two files."}]}}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	hookFile(t, r, `{"hook_event_name":"PreToolUse","tool_use_id":"t1","tool_input":{"description":"List files","prompt":"Run ls","subagent_type":"Explore"}}`)
	hookFile(t, r, `{"hook_event_name":"SubagentStart","agent_id":"a1","agent_type":"Explore","transcript_path":"`+
		strings.TrimSuffix(filepath.Dir(filepath.Dir(transcript)), "")+`.jsonl"}`)

	f := frameOf(t, u)
	y := -1
	for i, line := range f {
		if strings.Contains(line, "Explore: List") {
			y = i
		}
	}
	if y < 0 {
		t.Fatalf("no row for the subagent:\n%s", strings.Join(f, "\n"))
	}
	u.mouse(driver.MouseEvent{Button: driver.MouseLeft, X: 5, Y: uint16(y)})
	if !u.viewing() {
		t.Fatal("a click on the row did not open the subagent")
	}
	hookFile(t, r, `{"hook_event_name":"SubagentStop","agent_id":"a1","last_assistant_message":"Two files: go.mod and main.go."}`)
	shown := strings.Join(frameOf(t, u), "\n")
	for _, want := range []string{"Run ls", "→ Bash: ls", "go.mod", "Two files.", "Answer", "Two files: go.mod and main.go.", "✓ finished"} {
		if !strings.Contains(shown, want) {
			t.Errorf("%q is not shown:\n%s", want, shown)
		}
	}
	u.key(driverKey(driver.KeyEsc))
	if u.viewing() {
		t.Error("Esc did not go back to the agent's screen")
	}
}

func driverKey(k driver.KeyType) driver.KeyEvent { return driver.KeyEvent{Type: k} }

// A Claude Code 2.1 subagent reports through SubagentHandback; its message
// is the answer.
func TestAHandbackIsTheAnswer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent-a1.jsonl")
	if err := os.WriteFile(path, []byte(`{"type":"assistant","message":{"content":[{"type":"text","text":"Done."}]}}
{"type":"assistant","message":{"content":[{"type":"tool_use","name":"SubagentHandback","input":{"message":"main.go is empty."}}]}}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	rows, answer := activity(path)
	if answer != "main.go is empty." || len(rows) != 2 {
		t.Errorf("answer %q, rows %v", answer, rows)
	}
}
