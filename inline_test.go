package limoni

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/widgets"
)

// Inline mode has to stay out of the alternate screen and address the frame
// relatively, or it paints over the user's scrollback. Asserting on the bytes
// the driver actually wrote is the only way to know.
func TestInlineModeEmitsRelativeFrames(t *testing.T) {
	io := driver.NewMemoryTerminalIO(nil, 60, 20)
	backend := driver.NewPortableBackend(io)
	backend.SetInline(5)
	if err := backend.Setup(); err != nil {
		t.Fatalf("setup: %v", err)
	}
	term, err := terminal.New(backend)
	if err != nil {
		t.Fatalf("terminal: %v", err)
	}
	term.SetInline(5)

	var stop atomic.Bool
	var frames atomic.Int32
	app := testApp(term, appConfig{catchCtrlC: true, inlineHeight: 5})
	done := make(chan error, 1)
	go func() {
		done <- app.Run(context.Background(), func(f *Frame, ev *Event) bool {
			if frames.Add(1) > 1 && stop.Load() {
				return false
			}
			f.RenderWidget(widgets.NewLabel("inline frame"), NewRect(0, 0, 20, 1))
			return true
		})
	}()

	// Let the first frame render, then ask the loop to finish.
	stop.Store(true)
	app.Wakeup()
	if err := <-done; err != nil {
		t.Fatalf("run loop: %v", err)
	}
	_ = term.Close()

	out := string(string(io.Output()))

	if strings.Contains(out, "\x1b[?1049h") {
		t.Error("inline mode switched to the alternate screen")
	}
	if strings.Contains(out, "\x1b[?1049l") {
		t.Error("inline mode left the alternate screen it never entered")
	}
	// A full-screen clear in inline mode wipes the user's scrollback.
	if strings.Contains(out, "\x1b[2J") {
		t.Errorf("inline mode cleared the whole screen: %q", out)
	}
	// Absolute addressing would land on whatever row the terminal has scrolled
	// to, not the application's band.
	if strings.Contains(out, "\x1b[1;1H") {
		t.Errorf("inline frame used absolute cursor addressing: %q", out)
	}
	// Space is reserved by printing rows and stepping back up.
	if !strings.Contains(out, "\x1b[5A") {
		t.Errorf("inline setup did not reserve and rewind 5 rows: %q", out)
	}
	if !strings.Contains(out, "inline frame") {
		t.Errorf("frame content missing from output: %q", out)
	}
	// On exit the cursor is parked below the band so a prompt lands after it.
	if !strings.Contains(out, "\x1b[5B") {
		t.Errorf("inline teardown did not move past the frame: %q", out)
	}
	if frames.Load() == 0 {
		t.Error("application never rendered")
	}
}

// Full-screen mode must keep using the alternate screen.
func TestFullScreenModeStillUsesTheAlternateScreen(t *testing.T) {
	io := driver.NewMemoryTerminalIO(nil, 60, 20)
	backend := driver.NewPortableBackend(io)
	if err := backend.Setup(); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if !strings.Contains(string(io.Output()), "\x1b[?1049h") {
		t.Errorf("full-screen setup did not enter the alternate screen: %q", string(io.Output()))
	}
	_ = backend.Close()
	if !strings.Contains(string(io.Output()), "\x1b[?1049l") {
		t.Errorf("full-screen teardown did not leave the alternate screen: %q", string(io.Output()))
	}
}
