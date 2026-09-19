package uitest

import (
	"testing"

	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/widgets"
)

// Table rows, tabs and tree items used to be flat: an agent could see "a
// table" but not find the row reading "beta" and click it. Now each is a node
// with a label and bounds, addressed like a list item.
func TestRowsTabsAndTreeItemsAreAddressable(t *testing.T) {
	tableState := widgets.NewTableState()
	tabsState := &widgets.TabsState{}
	treeState := widgets.NewTreeViewState()
	selectedTab := 0
	root := []widgets.TreeNode{
		{ID: "src", Label: "src", Children: []widgets.TreeNode{{ID: "main", Label: "main.go"}}},
		{ID: "readme", Label: "README.md"},
	}

	page := Run(t, 60, 20, func(f *terminal.Frame, ev *driver.Event) bool {
		f.RenderWidget(&widgets.Tabs{ID: "tabs", Titles: []string{"Files", "Jobs", "Logs"}, Selected: selectedTab,
			OnSelect: func(i int) { selectedTab = i }, State: tabsState}, cell.NewRect(0, 0, 40, 1))
		f.RenderWidget(&widgets.Table{ID: "jobs", State: tableState,
			Rows:        []widgets.TableRow{widgets.NewRow("alpha", "ok"), widgets.NewRow("beta", "failed"), widgets.NewRow("gamma", "ok")},
			Constraints: []widgets.TableConstraint{{Type: widgets.ConstraintFixed, Value: 10}, {Type: widgets.ConstraintFixed, Value: 10}},
		}, cell.NewRect(0, 2, 30, 4))
		f.RenderWidget(&widgets.TreeView{ID: "tree", Roots: root, State: treeState}, cell.NewRect(0, 8, 30, 5))
		return true
	})

	rows := page.GetByRole("row", "").Within(page.GetByID("jobs"))
	page.Expect(rows).ToHaveCount(3)
	page.GetByRole("row", "beta").Click()
	page.Expect(page.GetByRole("row", "beta")).ToBeSelected()
	page.Expect(page.GetByRole("cell", "failed").Within(page.GetByRole("row", "beta"))).ToBeVisible()

	page.Expect(page.GetByRole("tab", "Files")).ToBeSelected()
	page.GetByRole("tab", "Logs").Click()
	page.Expect(page.GetByRole("tab", "Logs")).ToBeSelected()

	page.Expect(page.GetByRole("tree-item", "main.go")).Not().ToBeVisible()
	page.GetByRole("tree-item", "src").Click()
	page.Expect(page.GetByRole("tree-item", "src")).ToBeSelected()
}

// Check, Uncheck and Select are idempotent: running a step twice leaves the
// widget where the step put it, where Click would toggle it back.
func TestEnsureActionsAreIdempotent(t *testing.T) {
	checked := false
	listState := widgets.NewListState()
	clicks := 0
	page := Run(t, 40, 10, func(f *terminal.Frame, ev *driver.Event) bool {
		if ev != nil && ev.Type == driver.EventMouse && ev.Mouse.Button == driver.MouseLeft {
			clicks++
		}
		f.RenderWidget(&widgets.Checkbox{ID: "agree", Checked: &checked, Label: "Agree"}, cell.NewRect(0, 0, 20, 1))
		f.RenderWidget(&widgets.List{ID: "l", Items: []string{"one", "two"}, State: listState}, cell.NewRect(0, 2, 20, 3))
		return true
	})

	box := page.GetByID("agree")
	box.Check()
	box.Check()
	page.Expect(box).ToBeChecked()
	box.Uncheck()
	box.Uncheck()
	page.Expect(box).Not().ToBeChecked()

	page.GetByRole("list-item", "two").Select()
	page.GetByRole("list-item", "two").Select()
	page.Expect(page.GetByRole("list-item", "two")).ToBeSelected()

	if clicks != 3 { // check, uncheck, select; the repeats click nothing
		t.Fatalf("%d clicks reached the app, want 3", clicks)
	}
}
