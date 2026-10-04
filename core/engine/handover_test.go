package engine

import (
	"context"
	"errors"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
)

type editedMsg struct{ err error }

// handoverModel asks for an editor on "edit" and records what comes back.
type handoverModel struct {
	cmd  *exec.Cmd
	mu   sync.Mutex
	seen []Msg
	got  chan error
}

func (m *handoverModel) Init() []Cmd { return nil }
func (m *handoverModel) Update(msg Msg) UpdateResult {
	m.mu.Lock()
	m.seen = append(m.seen, msg)
	m.mu.Unlock()
	switch msg := msg.(type) {
	case string:
		if msg == "edit" {
			return UpdateResult{Commands: []Cmd{ExecCmd(m.cmd, func(err error) Msg { return editedMsg{err} })}}
		}
	case editedMsg:
		m.got <- msg.err
	}
	return UpdateResult{}
}
func (m *handoverModel) View(*terminal.Frame) {}

func (m *handoverModel) sawHandover() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, msg := range m.seen {
		if _, ok := msg.(handoverMsg); ok {
			return true
		}
	}
	return false
}

func waitErr(t *testing.T, ch <-chan error) error {
	t.Helper()
	select {
	case err := <-ch:
		return err
	case <-time.After(3 * time.Second):
		t.Fatal("the model heard nothing back")
		return nil
	}
}

// Whoever owns the terminal receives the request, runs the program, and the
// outcome reaches Update as the message done made. The request itself never
// reaches Update.
func TestExecCmdRunsThroughTheTerminalOwner(t *testing.T) {
	model := &handoverModel{cmd: exec.Command("false"), got: make(chan error, 1)}
	p := New(WithModel(model))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = p.Run(ctx) }()

	if err := p.Send(ctx, "edit"); err != nil {
		t.Fatal(err)
	}
	var h Handover
	select {
	case h = <-p.Handovers():
	case <-time.After(3 * time.Second):
		t.Fatal("no handover requested")
	}
	if h.Suspend || h.Run == nil {
		t.Fatalf("handover = %+v, want a program to run", h)
	}
	h.Done(h.Run())
	var exitErr *exec.ExitError
	if err := waitErr(t, model.got); !errors.As(err, &exitErr) {
		t.Fatalf("done got %v, want the program's exit status", err)
	}
	if model.sawHandover() {
		t.Error("Update received the handover request")
	}
}

// A second request while one is still waiting fails at once rather than
// blocking Update until the first is taken.
func TestExecCmdWhileOneIsPendingIsBusy(t *testing.T) {
	model := &handoverModel{cmd: exec.Command("true"), got: make(chan error, 2)}
	p := New(WithModel(model))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = p.Run(ctx) }()

	_ = p.Send(ctx, "edit")
	_ = p.Send(ctx, "edit")
	if err := waitErr(t, model.got); !errors.Is(err, ErrHandoverBusy) {
		t.Fatalf("second request got %v, want ErrHandoverBusy", err)
	}
}

// RunTerminal takes the request on its own loop. A backend with no terminal
// to give away reports so to the model, without starting the program.
func TestExecCmdWithoutATerminalReportsUnsupported(t *testing.T) {
	t.Setenv("LIMONI_PROBE", "0")
	b := driver.NewPortableBackend(driver.NewMemoryTerminalIO(nil, 20, 5))
	term, err := terminal.New(b)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("true")
	model := &handoverModel{cmd: cmd, got: make(chan error, 1)}
	p := New(WithModel(model))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- p.RunTerminal(ctx, term, b) }()
	defer func() { cancel(); <-done }()

	_ = p.Send(ctx, "edit")
	if err := waitErr(t, model.got); !errors.Is(err, driver.ErrReleaseUnsupported) {
		t.Fatalf("done got %v, want ErrReleaseUnsupported", err)
	}
	if cmd.Process != nil {
		t.Error("the program was started with no terminal to run in")
	}
}
