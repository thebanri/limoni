package terminal_test

import (
	"bytes"
	"testing"

	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
)

func newTitleTerm(t *testing.T) (*terminal.Terminal, *driver.MemoryTerminalIO) {
	t.Helper()
	io := driver.NewMemoryTerminalIO(nil, 40, 10)
	backend := driver.NewPortableBackend(io)
	if err := backend.Setup(); err != nil {
		t.Fatalf("setup: %v", err)
	}
	term, err := terminal.New(backend)
	if err != nil {
		t.Fatalf("terminal: %v", err)
	}
	return term, io
}

func TestSetTitleWritesOSC2(t *testing.T) {
	term, io := newTitleTerm(t)
	before := len(io.Output())
	term.SetTitle("Limoni")
	got := io.Output()[before:]
	want := []byte("\x1b]2;Limoni\x07")
	if !bytes.Equal(got, want) {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestSetTitleStripsESCAndBEL(t *testing.T) {
	term, io := newTitleTerm(t)
	before := len(io.Output())
	term.SetTitle("hi\x1b]99;pwn\x07there")
	got := io.Output()[before:]
	want := []byte("\x1b]2;hi]99;pwnthere\x07")
	if !bytes.Equal(got, want) {
		t.Fatalf("got %q want %q", got, want)
	}
	if bytes.Count(got, []byte{0x1b}) != 1 {
		t.Fatalf("injected ESC survived: %q", got)
	}
	if bytes.Count(got, []byte{0x07}) != 1 {
		t.Fatalf("injected BEL survived: %q", got)
	}
}

func TestSetTitleEmptyStillWritesOSC2(t *testing.T) {
	term, io := newTitleTerm(t)
	before := len(io.Output())
	term.SetTitle("")
	got := io.Output()[before:]
	want := []byte("\x1b]2;\x07")
	if !bytes.Equal(got, want) {
		t.Fatalf("got %q want %q", got, want)
	}
}
