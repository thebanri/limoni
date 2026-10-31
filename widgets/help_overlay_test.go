package widgets

import (
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

// helpBlockingCallback stands in for a shortcut's Handler, which the card never
// runs: it only reads the bindings.
func helpBlockingCallback() {}

func helpTestManager() (*KeybindingManager, *bool) {
	modal := false
	km := NewKeybindingManager()
	km.Register(Keybinding{Key: driver.KeyRune, Ch: 'q', Ctrl: true, Label: "Quit", Category: "Application", Handler: helpBlockingCallback})
	km.Register(Keybinding{Key: driver.KeyRune, Ch: 's', Ctrl: true, Label: "Save file", Category: "File", Handler: helpBlockingCallback})
	km.Register(Keybinding{Key: driver.KeyTab, Label: "Next field", Category: "Navigation", Handler: helpBlockingCallback})
	km.Register(Keybinding{Key: driver.KeyRune, Ch: 'o', Ctrl: true, Label: "Open", Category: "File", Handler: helpBlockingCallback})
	km.Register(Keybinding{Key: driver.KeyEsc, Label: "Close dialog", Category: "Navigation", Scope: "dialog", When: func() bool { return modal }, Handler: helpBlockingCallback})
	km.Register(Keybinding{Key: driver.KeyRune, Ch: 'x', Ctrl: true, Label: "Drop", Category: "File", Scope: "dialog", Handler: helpBlockingCallback})
	// A binding with no label is a shortcut nothing announces, so the card skips it.
	km.Register(Keybinding{Key: driver.KeyRune, Ch: 'z', Ctrl: true, Category: "File", Handler: helpBlockingCallback})
	return km, &modal
}

// helpRows renders the card and returns its visible rows, with trailing spaces
// trimmed.
func helpRows(w Widget, width, height uint16) []string {
	area := cell.NewRect(0, 0, width, height)
	buf := buffer.NewBuffer(area)
	w.Draw(cell.NewContext(area, cell.Style{}), buf)
	rows := make([]string, 0, height)
	for y := uint16(0); y < height; y++ {
		var b strings.Builder
		for x := uint16(0); x < width; x++ {
			b.WriteRune(buf.CellAt(x, y).Rune())
		}
		rows = append(rows, strings.TrimRight(b.String(), " "))
	}
	return rows
}

func helpJoined(rows []string) string {
	return strings.Join(rows, "\n")
}

func TestHelpOverlayGroupsByCategoryInFirstAppearanceOrder(t *testing.T) {
	km, _ := helpTestManager()
	help := NewHelpOverlay(km)

	got := helpJoined(helpRows(help, 34, 10))
	for _, want := range []string{"Application", "Ctrl+Q  Quit", "File", "Ctrl+S  Save file", "Ctrl+O  Open", "Navigation", "Tab  Next field"} {
		if !strings.Contains(got, want) {
			t.Errorf("card is missing %q:\n%s", want, got)
		}
	}
	// The category order is the order the categories first appear in, not the
	// alphabetical one.
	if strings.Index(got, "Application") > strings.Index(got, "Navigation") {
		t.Errorf("categories are not in first-appearance order:\n%s", got)
	}
}

func TestHelpOverlaySkipsInactiveAndOutOfScopeBindings(t *testing.T) {
	km, modal := helpTestManager()

	got := helpJoined(helpRows(NewHelpOverlay(km), 34, 10))
	for _, hidden := range []string{"Close dialog", "Ctrl+X", "Ctrl+Z"} {
		if strings.Contains(got, hidden) {
			t.Errorf("card shows %q, which is out of scope, inactive or unlabelled:\n%s", hidden, got)
		}
	}

	// The same binding appears as soon as its When reports that it is active.
	*modal = true
	scoped := NewHelpOverlay(km).WithScopes("dialog")
	got = helpJoined(helpRows(scoped, 34, 10))
	for _, want := range []string{"Close dialog", "Ctrl+X  Drop"} {
		if !strings.Contains(got, want) {
			t.Errorf("scoped card is missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Ctrl+Q") {
		t.Errorf("scoped card leaks the global bindings:\n%s", got)
	}
}

func TestHelpOverlayFlowsGroupsIntoColumns(t *testing.T) {
	km, _ := helpTestManager()
	help := NewHelpOverlay(km)

	// 34 cells hold one column of width 20, 70 hold three.
	narrow := helpRows(help, 34, 10)
	wide := helpRows(help, 70, 4)
	if !strings.Contains(narrow[0], "Application") || strings.Contains(narrow[0], "Navigation") {
		t.Fatalf("narrow card should hold one column, got %q", narrow[0])
	}
	if !strings.Contains(wide[0], "Application") {
		t.Fatalf("wide card should start with the first group, got %q", wide[0])
	}
	if second := wide[0][20:]; strings.TrimSpace(second) == "" {
		t.Errorf("wide card should use a second column, got %q", wide[0])
	}
}

func TestHelpOverlayScrollsASingleColumn(t *testing.T) {
	km, _ := helpTestManager()
	help := NewHelpOverlay(km)
	help.State.Offset = 0

	rows := helpRows(help, 34, 3)
	if !strings.Contains(helpJoined(rows), "Application") {
		t.Fatalf("first page should start at the first group, got %q", rows)
	}

	help.State.Offset = 4
	rows = helpRows(help, 34, 3)
	scrolled := helpJoined(rows)
	if strings.Contains(scrolled, "Application") || strings.Contains(scrolled, "Quit") {
		t.Errorf("scrolled card still shows the first page:\n%s", scrolled)
	}
	if !strings.Contains(scrolled, "Navigation") {
		t.Errorf("scrolled card should reach the last group:\n%s", scrolled)
	}

	// An offset past the end is clamped, so the last page stays on screen.
	help.State.Offset = 99
	rows = helpRows(help, 34, 3)
	if !strings.Contains(helpJoined(rows), "Next field") {
		t.Errorf("offset past the end should clamp to the last page:\n%s", helpJoined(rows))
	}
	if got, want := help.State.Offset, 4; got != want {
		t.Errorf("offset = %d, want at most %d", got, want)
	}
}

func TestHelpOverlaySizeHint(t *testing.T) {
	km, _ := helpTestManager()
	help := NewHelpOverlay(km)

	w, h := help.SizeHint(cell.NewRect(0, 0, 80, 25))
	// "Ctrl+S"/"Ctrl+O" is 6 wide, "Next field" is 10, plus the 2-cell gap and
	// the 2-cell column padding; three headings and four bindings are 7 rows.
	if w != 20 {
		t.Errorf("width = %d, want 20", w)
	}
	if h != 7 {
		t.Errorf("height = %d, want 7", h)
	}

	// The hint is capped by the space it is given.
	w, h = help.SizeHint(cell.NewRect(0, 0, 8, 3))
	if w > 8 || h > 3 {
		t.Errorf("SizeHint(8x3) = %dx%d, want at most 8x3", w, h)
	}

	// A title widens a card that would otherwise be narrower than its heading.
	title := NewHelpOverlay(km).WithTitle("Keyboard shortcuts for the whole application")
	tw, _ := title.SizeHint(cell.NewRect(0, 0, 80, 25))
	want := uint16(cell.StringWidth("Keyboard shortcuts for the whole application"))
	if tw != want {
		t.Errorf("title width = %d, want %d", tw, want)
	}
}

func TestHelpOverlayAccessibilityNode(t *testing.T) {
	km, _ := helpTestManager()
	help := NewHelpOverlay(km).WithTitle("Keys")

	bounds := cell.NewRect(0, 0, 34, 10)
	node := help.AccessibilityNode(bounds, true)
	if node.Role == 0 || node.Label != "Key bindings" {
		t.Errorf("node = %+v, want a labelled list", node)
	}
	// The root counts the rows it exposes, headings included, the way List does.
	if node.SetSize != 7 || node.SetSize != len(node.Children) {
		t.Errorf("SetSize = %d with %d children, want 7", node.SetSize, len(node.Children))
	}

	var keys, categories int
	for _, child := range node.Children {
		if child.Label == "" {
			t.Errorf("child without a label: %+v", child)
		}
		if child.Value != "" {
			keys++
			if child.Position == 0 || child.SetSize != 4 {
				t.Errorf("binding node lacks its position: %+v", child)
			}
		} else {
			categories++
		}
	}
	if keys != 4 || categories != 3 {
		t.Errorf("node has %d binding rows and %d headings, want 4 and 3", keys, categories)
	}

	// A row is labelled with the action it runs, and carries the key as its
	// value, so a selector finds it by "Save file".
	found := false
	for _, child := range node.Children {
		if child.Label == "Save file" {
			found = true
			if child.Value != "Ctrl+S" {
				t.Errorf("row %q has value %q, want the key Ctrl+S", child.Label, child.Value)
			}
		}
	}
	if !found {
		t.Error(`no row is labelled "Save file"`)
	}
}

func TestHelpOverlayDrawsAnEmptyCardWithoutAManager(t *testing.T) {
	help := &HelpOverlay{}
	rows := helpRows(help, 20, 3)
	for _, row := range rows {
		if row != "" {
			t.Errorf("empty card drew %q", row)
		}
	}
	if got := help.Entries(); len(got) != 0 {
		t.Errorf("Entries() = %v, want none", got)
	}
}

// TestHelpOverlayDrawAllocatesNothing pins the requirement from the issue: the
// card's layout buffers live in HelpState, so a frame after the first one
// allocates nothing.
func TestHelpOverlayDrawAllocatesNothing(t *testing.T) {
	km, _ := helpTestManager()
	help := NewHelpOverlay(km).WithTitle("Keys")
	area := cell.NewRect(0, 0, 80, 25)
	buf := buffer.NewBuffer(area)
	ctx := cell.NewContext(area, cell.Style{})

	// The first frame fills the state's buffers.
	help.Draw(ctx, buf)

	if allocs := testing.AllocsPerRun(50, func() { help.Draw(ctx, buf) }); allocs != 0 {
		t.Errorf("Draw allocates %v times per frame, want 0", allocs)
	}
	if allocs := testing.AllocsPerRun(50, func() { help.AccessibilityNode(area, true) }); allocs != 0 {
		t.Errorf("AccessibilityNode allocates %v times per frame, want 0", allocs)
	}
}

// TestHelpOverlayEntriesDoesNotDisturbTheLayout pins the bug a review found:
// Entries used to compact the buffer Draw indexes into, so a card with a
// filtered-out row printed the wrong labels on the next frame.
func TestHelpOverlayEntriesDoesNotDisturbTheLayout(t *testing.T) {
	km, _ := helpTestManager()

	// A manager whose first labelled binding is filtered out by When opens the
	// window where the visible indices no longer start at zero.
	hidden := false
	km.Register(Keybinding{Key: driver.KeyRune, Ch: 'h', Ctrl: true, Label: "Hidden", Category: "Application", When: func() bool { return hidden }})
	help := NewHelpOverlay(km)

	entries := help.Entries()
	if len(entries) == 0 {
		t.Fatal("Entries returned nothing")
	}
	for _, entry := range entries {
		if entry.Label == "Hidden" {
			t.Fatalf("Entries returned a filtered-out binding: %v", entries)
		}
	}

	got := helpJoined(helpRows(help, 34, 12))
	if strings.Contains(got, "Hidden") {
		t.Errorf("the card drew the filtered-out binding:\n%s", got)
	}
	for _, entry := range entries {
		if !strings.Contains(got, entry.Label) {
			t.Errorf("the card is missing %q after Entries was called:\n%s", entry.Label, got)
		}
	}
	if second := helpJoined(helpRows(help, 34, 12)); second != got {
		t.Errorf("the card changed between two frames:\n%s\n---\n%s", got, second)
	}
}

func TestHelpOverlayDoesNotScrollPastItsContent(t *testing.T) {
	km, _ := helpTestManager()
	help := NewHelpOverlay(km).WithTitle("Keys")
	help.State.Offset = 99

	rows := helpRows(help, 34, 4)
	// The title takes the first row, so three rows of content remain and the
	// card ends on the last binding rather than on a blank line.
	if strings.TrimSpace(rows[0]) != "Keys" {
		t.Errorf("first row = %q, want the title", rows[0])
	}
	if strings.TrimSpace(rows[3]) == "" {
		t.Errorf("the last page ends on a blank row: %q", rows)
	}
	if !strings.Contains(helpJoined(rows), "Next field") {
		t.Errorf("the last page should reach the last binding: %q", rows)
	}
}

func TestHelpOverlayStaysInsideItsArea(t *testing.T) {
	km := NewKeybindingManager()
	km.Register(Keybinding{Key: driver.KeyRune, Ch: 's', Ctrl: true, Label: "A label much wider than the card", Category: "File"})
	help := NewHelpOverlay(km)

	// The buffer is wider than the card's area: anything written past x=10 is
	// the card painting over its neighbour.
	area := cell.NewRect(0, 0, 10, 3)
	buf := buffer.NewBuffer(cell.NewRect(0, 0, 40, 3))
	help.Draw(cell.NewContext(area, cell.Style{}), buf)

	for x := uint16(10); x < 40; x++ {
		for y := uint16(0); y < 3; y++ {
			if got := buf.CellAt(x, y).Rune(); got != 0 && got != ' ' {
				t.Fatalf("cell (%d,%d) = %q, outside the card's area", x, y, got)
			}
		}
	}
}

func TestHelpOverlayNeverLeavesAHeadingAloneInAColumn(t *testing.T) {
	km, _ := helpTestManager()
	help := NewHelpOverlay(km)

	// Height 3 cuts the first page after "Application"/"Ctrl+Q": the "File"
	// heading must move to the next page rather than end this one.
	rows := helpRows(help, 34, 3)
	if isHeading := strings.Contains(rows[2], "File"); isHeading && strings.TrimSpace(rows[2]) == "File" {
		t.Errorf("the page ends on a lone heading: %q", rows)
	}
	if !strings.Contains(rows[0], "Application") {
		t.Errorf("first row = %q, want the first heading", rows[0])
	}
}

// TestHelpOverlayEveryRowReachable pins the blocker a review found: the card
// counted columnHeight*columns cells, but a heading that moves to the next
// column leaves a cell empty, so the last rows were never reachable.
func TestHelpOverlayEveryRowReachable(t *testing.T) {
	km, _ := helpTestManager()
	help := NewHelpOverlay(km)
	for offset := 0; offset < 10; offset++ {
		help.State.Offset = offset
		if strings.Contains(helpJoined(helpRows(help, 60, 2)), "Next field") {
			// The tree comes from the same walk, so the row is in it too.
			found := false
			for _, child := range help.AccessibilityNode(cell.NewRect(0, 0, 60, 2), true).Children {
				if child.Label == "Next field" {
					found = true
				}
			}
			if !found {
				t.Errorf("AccessibilityNode at offset %d is missing %q", offset, "Next field")
			}
			return
		}
	}
	t.Fatal(`"Next field" is never drawn at 60x2`)
}
