package terminal_test

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
)

// Replies from a terminal that the environment says nothing about — the tmux
// case: TERM=screen, no TERM_PROGRAM — still switch on what it supports.
func TestHandshakeRefinesCapabilities(t *testing.T) {
	t.Setenv("TERM", "screen")
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("COLORTERM", "")
	t.Setenv("LIMONI_REP", "")
	t.Setenv("LIMONI_NO_SYNC", "")
	t.Setenv("LIMONI_PROBE", "")

	term, b := handshakeTerminal(t, "\x1bP>|kitty(0.39.1)\x1b\\\x1b[?2026;2$y\x1b[?2027;1$y\x1b[?1;2c")
	before := term.Capabilities()
	if before.RepeatChar || before.TrueColor || before.ClusterWidths {
		t.Fatalf("environment guess already has the answers: %+v", before)
	}

	waitAnswered(t, b)
	if err := term.Draw(func(*terminal.Frame) {}); err != nil {
		t.Fatal(err)
	}
	after := term.Capabilities()
	if !after.RepeatChar || !after.TrueColor || !after.ClusterWidths || !after.SyncOutput {
		t.Fatalf("handshake not applied: %+v", after)
	}
}

// A terminal that answers DECRQM with "not recognised" loses the guess.
func TestHandshakeTurnsOffWhatTheTerminalLacks(t *testing.T) {
	t.Setenv("LIMONI_NO_SYNC", "")
	t.Setenv("LIMONI_PROBE", "")
	term, b := handshakeTerminal(t, "\x1b[?2026;0$y\x1b[?2027;0$y\x1b[?1;2c")
	if !term.Capabilities().SyncOutput {
		t.Fatal("test assumes the environment guess enables synchronized output")
	}
	waitAnswered(t, b)
	if err := term.Draw(func(*terminal.Frame) {}); err != nil {
		t.Fatal(err)
	}
	if caps := term.Capabilities(); caps.SyncOutput || caps.ClusterWidths {
		t.Fatalf("unsupported modes still enabled: %+v", caps)
	}
}

// SetCapabilities is the application's word and outranks the handshake.
func TestSetCapabilitiesOutranksHandshake(t *testing.T) {
	t.Setenv("LIMONI_PROBE", "")
	term, b := handshakeTerminal(t, "\x1bP>|kitty(0.39.1)\x1b\\\x1b[?1;2c")
	term.SetCapabilities(terminal.CapabilityProfile{})
	waitAnswered(t, b)
	if err := term.Draw(func(*terminal.Frame) {}); err != nil {
		t.Fatal(err)
	}
	if caps := term.Capabilities(); caps.RepeatChar || caps.TrueColor {
		t.Fatalf("handshake overrode SetCapabilities: %+v", caps)
	}
}

func handshakeTerminal(t *testing.T, replies string) (*terminal.Terminal, *driver.Backend) {
	t.Helper()
	b := driver.NewPortableBackend(driver.NewMemoryTerminalIO([]byte(replies), 40, 10))
	if err := b.Setup(); err != nil {
		t.Fatal(err)
	}
	term, err := terminal.New(b)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = term.Close() })
	b.StartEventLoop()
	return term, b
}

func waitAnswered(t *testing.T, b *driver.Backend) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if r, _ := b.TerminalReport(); r.Answered {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("DA1 never arrived")
}

// Measurements beat names: a terminal that calls itself kitty but did not
// move the cursor for REP gets no REP, and one measured drawing the family
// emoji two columns wide needs no cursor re-anchoring even without mode 2027.
func TestMeasurementsBeatNames(t *testing.T) {
	t.Setenv("LIMONI_REP", "")
	t.Setenv("LIMONI_PROBE", "")
	term, b := handshakeTerminal(t, "\x1bP>|kitty(0.39.1)\x1b\\\x1b[?2027;0$y\x1b[1;2R\x1b[1;3R\x1b[?1;2c")
	waitAnswered(t, b)
	if err := term.Draw(func(*terminal.Frame) {}); err != nil {
		t.Fatal(err)
	}
	caps := term.Capabilities()
	if caps.RepeatChar {
		t.Error("REP enabled although the measurement said it does not work")
	}
	if !caps.ClusterWidths {
		t.Error("cluster widths not trusted although measured at two columns")
	}
}

// Replies can arrive after the first frame — over SSH they often do. If they
// change the profile, what is already on screen was encoded for the wrong
// terminal and must be repainted in full, not patched.
func TestLateAnswersRepaintTheScreen(t *testing.T) {
	t.Setenv("LIMONI_REP", "")
	t.Setenv("LIMONI_PROBE", "")
	t.Setenv("TERM", "xterm-256color") // the guess enables REP
	t.Setenv("TERM_PROGRAM", "")

	in := &lateInput{ch: make(chan []byte, 1)}
	io := &lateIO{MemoryTerminalIO: driver.NewMemoryTerminalIO(nil, 40, 4), in: in}
	b := driver.NewPortableBackend(io)
	if err := b.Setup(); err != nil {
		t.Fatal(err)
	}
	term, err := terminal.New(b)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { close(in.ch); _ = term.Close() })
	b.StartEventLoop()

	draw := func(f *terminal.Frame) {
		f.Buffer.SetString(0, 0, "==========", cell.Style{})
	}
	if err := term.Draw(draw); err != nil {
		t.Fatal(err)
	}
	if !term.Capabilities().RepeatChar {
		t.Fatal("test assumes the environment guess enables REP")
	}

	in.ch <- []byte("\x1b[1;2R\x1b[1;7R\x1b[?1;2c") // REP does not work here
	waitAnswered(t, b)
	before := len(io.Output())
	if err := term.Draw(draw); err != nil {
		t.Fatal(err)
	}
	repaint := string(io.Output()[before:])
	if !strings.Contains(repaint, "==========") {
		t.Fatalf("unchanged frame was not repainted without REP: %q", repaint)
	}
	if strings.Contains(repaint, "b") {
		t.Fatalf("repaint still uses REP: %q", repaint)
	}
}

// lateIO delivers input only when the test sends it.
type lateIO struct {
	*driver.MemoryTerminalIO
	in *lateInput
}

type lateInput struct{ ch chan []byte }

func (l *lateIO) Read(p []byte) (int, error) {
	data, ok := <-l.in.ch
	if !ok {
		return 0, io.EOF
	}
	return copy(p, data), nil
}
