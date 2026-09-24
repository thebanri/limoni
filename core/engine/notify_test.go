package engine

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
)

type notifyModel struct {
	mu   sync.Mutex
	seen []Msg
}

func (m *notifyModel) Init() []Cmd { return []Cmd{NotifyCmd("Build", "done")} }
func (m *notifyModel) Update(msg Msg) UpdateResult {
	m.mu.Lock()
	m.seen = append(m.seen, msg)
	m.mu.Unlock()
	return UpdateResult{}
}
func (m *notifyModel) View(*terminal.Frame) {}

// NotifyCmd reaches the terminal through RunTerminal's loop, the one that
// draws, and never reaches Update.
func TestNotifyCmdWritesThroughRunTerminal(t *testing.T) {
	t.Setenv("LIMONI_PROBE", "0")
	io := driver.NewMemoryTerminalIO(nil, 20, 5)
	b := driver.NewPortableBackend(io)
	term, err := terminal.New(b)
	if err != nil {
		t.Fatal(err)
	}
	caps := term.Capabilities()
	caps.Notify = terminal.NotifyOSC9
	term.SetCapabilities(caps)

	model := &notifyModel{}
	p := New(WithModel(model))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- p.RunTerminal(ctx, term, b) }()

	deadline := time.Now().Add(2 * time.Second)
	for !strings.Contains(string(io.Output()), "\x1b]9;Build: done\a") {
		if time.Now().After(deadline) {
			cancel()
			t.Fatalf("no notification written; output %q", io.Output())
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	<-done

	model.mu.Lock()
	defer model.mu.Unlock()
	for _, m := range model.seen {
		if _, ok := m.(notifyMsg); ok {
			t.Error("Update received the notification request")
		}
	}
}
