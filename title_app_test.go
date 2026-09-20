package limoni

import (
	"bytes"
	"context"
	"testing"

	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
)

// WithTitle must reach every entry point, not only the package-level Run:
// NewApp(...).Run and RunWithContext go through App.Run, which is where the
// title is applied. The saved title is popped on the way out, so quitting does
// not leave the application's name on the user's terminal.
func TestWithTitleSetsAndRestoresTheTitle(t *testing.T) {
	t.Setenv("LIMONI_PROBE", "0")
	io := driver.NewMemoryTerminalIO(nil, 40, 10)
	b := driver.NewPortableBackend(io)
	if err := b.Setup(); err != nil {
		t.Fatal(err)
	}
	term, err := terminal.New(b)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = term.Close() }()

	var cfg appConfig
	WithTitle("zest — app.log")(&cfg)
	app := testApp(term, cfg)
	if err := app.Run(context.Background(), func(*Frame, *Event) bool { return false }); err != nil {
		t.Fatalf("run: %v", err)
	}

	out := io.Output()
	save, set, restore := []byte("\x1b[22;2t"), []byte("\x1b]2;zest — app.log\x07"), []byte("\x1b[23;2t")
	i, j, k := bytes.Index(out, save), bytes.Index(out, set), bytes.Index(out, restore)
	if i < 0 || j < 0 || k < 0 {
		t.Fatalf("save=%d set=%d restore=%d in %q", i, j, k, out)
	}
	if !(i < j && j < k) {
		t.Errorf("order should be save, set, restore: %d, %d, %d", i, j, k)
	}
}
