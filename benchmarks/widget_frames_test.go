package benchmarks

import (
	"strings"
	"testing"

	"github.com/thebanri/limoni/component"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/widgets"
)

// The widget benchmarks call Draw without a frame, so they never see the
// handlers a widget registers. These are drawn through a real Terminal, with
// the state that keeps their handlers between frames, open where they open.
// Each one allocated before its handlers moved into its state: Slider 3,
// Table 4 (20 sorted), Dialog 18, Viewport 15, ColorPicker 20, …
func TestInteractiveWidgetsDrawWithoutAllocating(t *testing.T) {
	sortedState := widgets.NewTableState()
	sortedState.SortColumn = 0
	sorted := widgets.NewTable().WithID("sorted").WithHeaders("Name", "Age").
		WithRow("Cem", "41").WithRow("Ada", "36").WithState(sortedState)
	sorted.SortEnabled = true

	openSelect := widgets.NewSelectState()
	openSelect.Open = true
	openPopup := widgets.NewPopupState()
	openPopup.Open()
	palette := widgets.NewCommandPaletteState()
	palette.AllItems = []widgets.CommandItem{{Label: "Open file"}, {Label: "Save"}}
	palette.Open()
	toasts := widgets.NewToastManager(widgets.ToastTopRight)
	toasts.Info("Saved", "3 files")
	offset := 2

	cases := []struct {
		name string
		w    widgets.Widget
	}{
		{"Slider", &widgets.Slider{ID: "s", State: widgets.NewSliderState(5), Min: 0, Max: 10, OnChange: func(int) {}}},
		{"Table", widgets.NewTable().WithID("t").WithHeaders("a", "b").WithRow("1", "2").WithState(widgets.NewTableState())},
		{"Table sorted", sorted},
		{"Dialog", &widgets.Dialog{ID: "d", Title: "Quit?", Message: "Unsaved changes will be lost.",
			Buttons: []widgets.DialogButton{{Text: "OK", Handler: func() {}}, {Text: "Cancel"}}, State: &widgets.DialogState{}}},
		{"Viewport", &widgets.Viewport{ID: "v", State: widgets.NewViewportState(), Child: widgets.NewParagraph("x\ny\nz"), ContentHeight: 30}},
		{"TreeView", &widgets.TreeView{ID: "tv", State: widgets.NewTreeViewState(), Roots: []widgets.TreeNode{
			{ID: "a", Label: "a", Expanded: true, Children: []widgets.TreeNode{{ID: "b", Label: "b"}}}}}},
		{"Select open", &widgets.Select{ID: "sel", Options: []string{"x", "y"}, State: openSelect}},
		{"Popup open", &widgets.Popup{ID: "pp", Label: "Menu", State: openPopup, Items: []widgets.PopupItem{{Text: "One"}, {Text: "Two"}}}},
		{"ColorPicker", &widgets.ColorPicker{ID: "cp", State: widgets.NewColorPickerState(10, 20, 30)}},
		{"CommandPalette open", &widgets.CommandPalette{ID: "cmd", State: palette}},
		{"Toast", toasts},
		{"Markdown scrolling", &widgets.Markdown{ID: "md", Content: "# Title\n\n" + strings.Repeat("line\n", 40), ScrollOffset: &offset}},
		{"OnClick", component.OnClick(component.Text("press"), func(driver.MouseEvent) {})},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			term := newBenchTerminal(t)
			draw := func(f *terminal.Frame) { f.RenderWidget(c.w, cell.NewRect(0, 0, 48, 14)) }
			_ = term.Draw(draw)
			if n := testing.AllocsPerRun(20, func() { _ = term.Draw(draw) }); n != 0 {
				t.Errorf("%v allocations a frame", n)
			}
		})
	}
}
