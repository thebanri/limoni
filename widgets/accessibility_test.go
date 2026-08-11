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

func TestWidgetAccessibilityProviders(t *testing.T) {
	// Checkbox
	checked := true
	cb := widgets.Checkbox{ID: "cb", Checked: &checked, Label: "Agree"}
	node := cb.AccessibilityNode(cell.NewRect(0, 0, 10, 1), true)
	if node.Role != accessibility.RoleCheckbox || node.Label != "Agree" || node.Value != "true" || (node.State&accessibility.StateFocused) == 0 {
		t.Errorf("Checkbox node = %+v", node)
	}

	// RadioButton
	selected := "opt1"
	rb := widgets.RadioButton{ID: "rb", Selected: &selected, Value: "opt1", Label: "Option 1"}
	rbNode := rb.AccessibilityNode(cell.NewRect(0, 0, 10, 1), false)
	if rbNode.Role != accessibility.RoleRadioButton || rbNode.Label != "Option 1" || rbNode.Value != "true" || (rbNode.State&accessibility.StateSelected) == 0 {
		t.Errorf("RadioButton node = %+v", rbNode)
	}

	// TextInput
	tiState := widgets.NewTextInputState()
	tiState.Text = []rune("Hello")
	ti := widgets.TextInput{ID: "ti", State: tiState, Placeholder: "Name"}
	tiNode := ti.AccessibilityNode(cell.NewRect(0, 0, 10, 1), true)
	if tiNode.Role != accessibility.RoleInput || tiNode.Label != "Name" || tiNode.Value != "Hello" {
		t.Errorf("TextInput node = %+v", tiNode)
	}

	// Slider
	slState := widgets.NewSliderState(5)
	sl := widgets.Slider{ID: "sl", State: slState, Min: 0, Max: 10}
	slNode := sl.AccessibilityNode(cell.NewRect(0, 0, 10, 1), false)
	if slNode.Role != accessibility.RoleSlider || slNode.Value != "5" {
		t.Errorf("Slider node = %+v", slNode)
	}

	// ProgressBar
	pb := widgets.ProgressBar{ID: "pb", Min: 0, Max: 100, Value: 75}
	pbNode := pb.AccessibilityNode(cell.NewRect(0, 0, 10, 1), false)
	if pbNode.Role != accessibility.RoleProgress || pbNode.Value != "75%" {
		t.Errorf("ProgressBar node = %+v", pbNode)
	}
}
