package engine

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
)

type quietModel struct{}

func (quietModel) Init() []Cmd             { return nil }
func (quietModel) Update(Msg) UpdateResult { return UpdateResult{} }
func (quietModel) View(f *terminal.Frame)  {}

// WithoutMouse leaves the mouse to the terminal: RunTerminal turns mouse
// reporting off on a terminal that was set up with it on, as limoni.New does.
func TestWithoutMouseTurnsReportingOff(t *testing.T) {
	t.Setenv("LIMONI_PROBE", "0")
	io := driver.NewMemoryTerminalIO(nil, 20, 5)
	b := driver.NewPortableBackend(io)
	if err := b.Setup(); err != nil {
		t.Fatal(err)
	}
	term, err := terminal.New(b)
	if err != nil {
		t.Fatal(err)
	}
	p := New(WithModel(quietModel{}), WithoutMouse())
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- p.RunTerminal(ctx, term, b) }()
	deadline := time.Now().Add(2 * time.Second)
	for !strings.Contains(string(io.Output()), "\x1b[?1006l\x1b[?1003l") {
		if time.Now().After(deadline) {
			cancel()
			t.Fatalf("mouse reporting was not turned off: %q", io.Output())
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	<-done
	if b.MouseEnabled() {
		t.Fatal("the backend still takes the mouse")
	}
}
