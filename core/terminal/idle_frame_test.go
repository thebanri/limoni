package terminal_test

import (
	"testing"

	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/widgets"
)

// A frame that changes nothing must write nothing. The runtime redraws on
// every tick, so an empty "?2026h ?2026l" pair per frame was a constant
// 960 bytes a second from an idle application at 60 FPS — traffic an SSH
// session pays for and a wakeup for the terminal every 16ms.
func TestUnchangedFrameWritesNothing(t *testing.T) {
	io := driver.NewMemoryTerminalIO(nil, 40, 10)
	backend := driver.NewPortableBackend(io)
	if err := backend.Setup(); err != nil {
		t.Fatalf("setup: %v", err)
	}
	term, err := terminal.New(backend)
	if err != nil {
		t.Fatalf("terminal: %v", err)
	}
	if !term.Capabilities().SyncOutput {
		t.Fatal("test assumes synchronized output is enabled by default")
	}

	draw := func(f *terminal.Frame) {
		f.RenderWidget(widgets.NewLabel("steady"), cell.Rect{Width: 20, Height: 1})
	}
	if err := term.Draw(draw); err != nil {
		t.Fatal(err)
	}
	before := len(io.Output())
	for i := 0; i < 3; i++ {
		if err := term.Draw(draw); err != nil {
			t.Fatal(err)
		}
	}
	if extra := io.Output()[before:]; len(extra) != 0 {
		t.Fatalf("unchanged frames wrote %d bytes: %q", len(extra), extra)
	}
}
