//go:build !limoni_debug

package limoni

import (
	"context"
	"errors"
	"testing"

	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
)

// In a release build, asking for automation must fail loudly rather than
// quietly running without it — a test that relies on the socket should not
// pass by never connecting.
func TestReleaseBuildRefusesAutomation(t *testing.T) {
	backend := driver.NewPortableBackend(driver.NewMemoryTerminalIO(nil, 40, 10))
	if err := backend.Setup(); err != nil {
		t.Fatalf("setup: %v", err)
	}
	term, err := terminal.New(backend)
	if err != nil {
		t.Fatalf("terminal: %v", err)
	}
	defer term.Close()

	err = testApp(term, appConfig{automationPath: "/tmp/never.sock"}).Run(context.Background(), func(*Frame, *Event) bool { return false })
	if !errors.Is(err, ErrAutomationNotCompiled) {
		t.Fatalf("runLoop = %v, want ErrAutomationNotCompiled", err)
	}
}
