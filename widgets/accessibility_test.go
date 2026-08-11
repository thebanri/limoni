package widgets_test

import (
	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/widgets"
	"testing"
)

type accessibleWidget struct{ widgets.Accessible }

func (accessibleWidget) Draw(_ cell.Context, _ *buffer.Buffer)    {}
func (accessibleWidget) SizeHint(area cell.Rect) (uint16, uint16) { return area.Width, area.Height }

func TestAccessibleWidgetProvider(t *testing.T) {
	f := terminal.NewFrame(buffer.NewBuffer(cell.NewRect(0, 0, 10, 2)), terminal.NewFocusManager())
	w := accessibleWidget{Accessible: widgets.Accessible{ID: "save", Role: accessibility.RoleButton, Label: "Save"}}
	f.RenderWidget(w, cell.NewRect(0, 0, 5, 1))
	if nodes := f.AccessibilityTree(); len(nodes) != 1 || nodes[0].Label != "Save" {
		t.Fatalf("nodes = %+v", nodes)
	}
}
