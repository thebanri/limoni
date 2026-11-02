package uitest

import (
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/engine"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/widgets"
)

// committer commits each key it is sent, as a chat commits a message.
type committer struct{}

func (committer) Init() []engine.Cmd { return nil }
func (committer) Update(msg engine.Msg) engine.UpdateResult {
	if k, ok := msg.(engine.KeyPressMsg); ok {
		return engine.UpdateResult{Commit: []widgets.Widget{widgets.NewLabel("sent " + string(k.Key.Ch))}}
	}
	return engine.UpdateResult{}
}
func (committer) View(f *terminal.Frame) {
	f.RenderWidget(widgets.NewLabel("> band"), cell.NewRect(0, 0, 20, 1))
}

// Committed rows are what an inline terminal shows above the band, so the
// screen has them there, in order.
func TestScreenShowsCommittedRowsAboveTheFrame(t *testing.T) {
	page := Program(t, 20, 2, committer{})
	page.Press("a")
	page.Press("b")
	page.WaitForScreen("sent b")
	if got := page.Screen(); !strings.HasPrefix(got, "sent a\nsent b\n> band") {
		t.Errorf("screen %q", got)
	}
}
