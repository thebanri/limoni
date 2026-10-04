package widgets

import (
	"image"
	"image/color"
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/graphics"
)

func treeFixture() []TreeNode {
	return []TreeNode{
		{ID: "src", Label: "src", Expanded: true, Children: []TreeNode{
			{ID: "a.go", Label: "a.go"},
			{ID: "lib", Label: "lib", Children: []TreeNode{{ID: "b.go", Label: "b.go"}}},
		}},
		{ID: "go.mod", Label: "go.mod"},
	}
}

// The keyboard walks the visible items, opens with → and Enter, closes with
// ←, and ← on a child goes to its parent.
func TestTreeViewKeyboard(t *testing.T) {
	roots := treeFixture()
	s := NewTreeViewState()
	key := func(k driver.KeyType) bool { return s.HandleKey(driver.KeyEvent{Type: k}, roots) }

	key(driver.KeyArrowUp) // nothing selected: the first item
	steps := []struct {
		key  driver.KeyType
		want string
	}{
		{driver.KeyArrowDown, "a.go"},
		{driver.KeyArrowDown, "lib"},
		{driver.KeyArrowRight, "lib"},  // opens lib
		{driver.KeyArrowRight, "b.go"}, // already open: into it
		{driver.KeyArrowLeft, "lib"},   // a child: to its parent
		{driver.KeyArrowLeft, "lib"},   // closes lib
		{driver.KeyEnd, "go.mod"},
		{driver.KeyHome, "src"},
		{driver.KeyPageDown, "go.mod"},
		{driver.KeyPageUp, "src"},
	}
	for i, st := range steps {
		key(st.key)
		if s.SelectedID != st.want {
			t.Fatalf("step %d: selected %q, want %q", i, s.SelectedID, st.want)
		}
	}
	if s.IsExpanded("lib", false) {
		t.Error("← on an open lib did not close it")
	}
	if !key(driver.KeyEnter) || s.IsExpanded("src", true) {
		t.Error("Enter on src did not close it")
	}
	if key(driver.KeyArrowDown); s.SelectedID != "go.mod" {
		t.Errorf("with src closed, ↓ went to %q", s.SelectedID)
	}
	if FindNode(roots, "b.go") == nil || FindNode(roots, "b.go").Label != "b.go" || FindNode(roots, "nope") != nil {
		t.Error("FindNode")
	}
}

func TestListStateNextPrevious(t *testing.T) {
	s := NewListState()
	s.Selected = 0
	s.Next()
	s.Next()
	s.Previous()
	if s.Selected != 1 {
		t.Fatalf("selected %d", s.Selected)
	}
	s.Previous()
	s.Previous()
	if s.Selected != 0 {
		t.Errorf("Previous went below 0: %d", s.Selected)
	}
	l := NewList("a").WithID("l").WithItems("x", "y").WithState(s).WithHighlightSymbol("> ").WithScrollbar(true).
		WithStyle(cell.Style{}).WithSelectedStyle(cell.Style{Modifier: cell.ModifierBold}).WithFocusedStyle(cell.Style{})
	if l.ID != "l" || len(l.Items) != 2 || l.HighlightSymbol != "> " || l.State != s {
		t.Errorf("builders did not set their fields: %+v", l)
	}
}

// The wheel scrolls rows, and with Shift it scrolls columns; columns do not
// scroll before the first.
func TestTableWheelAndHorizontalScroll(t *testing.T) {
	ts := NewTableState()
	ts.handleScroll(driver.MouseEvent{Button: driver.MouseScrollDown}, 100, 10)
	if ts.Offset != 3 {
		t.Errorf("wheel down scrolled to %d, want 3", ts.Offset)
	}
	ts.handleScroll(driver.MouseEvent{Button: driver.MouseScrollUp}, 100, 10)
	if ts.Offset != 0 {
		t.Errorf("wheel up scrolled to %d, want 0", ts.Offset)
	}
	ts.handleScroll(driver.MouseEvent{Button: driver.MouseScrollDown, Shift: true}, 100, 10)
	if ts.HorizontalOffset != 2 {
		t.Errorf("shift+wheel down: horizontal offset %d, want 2", ts.HorizontalOffset)
	}
	ts.handleScroll(driver.MouseEvent{Button: driver.MouseScrollUp, Shift: true}, 100, 10)
	ts.ScrollHorizontal(-5)
	if ts.HorizontalOffset != 0 {
		t.Errorf("horizontal offset went to %d", ts.HorizontalOffset)
	}
}

// Without mouse regions a table registers a click per row, and a click
// selects the row and focuses the table.
func TestTablePerRowClicksWithoutMouseRegions(t *testing.T) {
	state := NewTableState()
	tbl := NewTable().WithID("t").WithHeaders("a").WithRow("1").WithRow("2").WithState(state)
	area := cell.NewRect(0, 0, 10, 5)
	ctx := cell.NewContext(area, cell.Style{})
	var clicks []func()
	focused := ""
	ctx.RegisterClick = func(_ cell.Rect, f func()) { clicks = append(clicks, f) }
	ctx.SetFocus = func(id string) { focused = id }
	tbl.Draw(ctx, buffer.NewBuffer(area))
	if len(clicks) != 2 {
		t.Fatalf("%d row clicks registered, want 2", len(clicks))
	}
	clicks[1]()
	if state.Selected != 1 || focused != "t" {
		t.Errorf("second row click: selected %d, focused %q", state.Selected, focused)
	}
}

func TestGridIntersections(t *testing.T) {
	cases := map[[4]bool]rune{
		{true, true, true, true}:   '┼',
		{false, true, true, true}:  '┬',
		{true, false, true, true}:  '┴',
		{true, true, false, true}:  '├',
		{true, true, true, false}:  '┤',
		{true, true, false, false}: '│',
		{false, false, true, true}: '─',
	}
	for in, want := range cases {
		if got := getIntersectionChar(in[0], in[1], in[2], in[3]); got != want {
			t.Errorf("up %v down %v left %v right %v: %q, want %q", in[0], in[1], in[2], in[3], got, want)
		}
	}
}

func canvasDot(c *Canvas, px, py int) (bool, cell.Style) {
	idx := (py/4)*int(c.width) + px/2
	return c.grid[idx]&brailleOffset[py%4][px%2] != 0, c.styles[idx]
}

// A filled triangle covers its inside and nothing outside it.
func TestDrawFilledTriangle(t *testing.T) {
	c := NewCanvas(10, 5) // 20×20 dots
	style := cell.Style{Fg: cell.NewColorRGB(1, 2, 3)}
	c.DrawFilledTriangle(graphics.Vertex2D{X: 0, Y: 0}, graphics.Vertex2D{X: 19, Y: 0}, graphics.Vertex2D{X: 0, Y: 19}, style)
	if on, st := canvasDot(c, 3, 3); !on || st.Fg != style.Fg {
		t.Error("a point inside the triangle is not set")
	}
	if on, _ := canvasDot(c, 18, 18); on {
		t.Error("a point outside the triangle is set")
	}
	c.DrawFilledTriangle(graphics.Vertex2D{X: 0, Y: 0}, graphics.Vertex2D{X: 5, Y: 5}, graphics.Vertex2D{X: 10, Y: 10}, style) // degenerate: a line
}

// A textured triangle takes each dot's colour from the texture at its UV.
func TestDrawTexturedTriangle(t *testing.T) {
	tex := image.NewRGBA(image.Rect(0, 0, 2, 1))
	tex.Set(0, 0, color.RGBA{R: 255, A: 255}) // left half red
	tex.Set(1, 0, color.RGBA{B: 255, A: 255}) // right half blue
	c := NewCanvas(10, 5)
	p0, p1, p2 := graphics.Vertex2D{X: 0, Y: 0}, graphics.Vertex2D{X: 19, Y: 0}, graphics.Vertex2D{X: 0, Y: 19}
	c.DrawTexturedTriangle(p0, p1, p2, graphics.UV{U: 0, V: 0}, graphics.UV{U: 1, V: 0}, graphics.UV{U: 0, V: 1}, tex)
	if on, st := canvasDot(c, 1, 1); !on || st.Fg != cell.NewColorRGB(255, 0, 0) {
		t.Errorf("near the left corner: set %v colour %v, want red", on, st.Fg)
	}
	if on, st := canvasDot(c, 17, 1); !on || st.Fg != cell.NewColorRGB(0, 0, 255) {
		t.Errorf("near the right corner: set %v colour %v, want blue", on, st.Fg)
	}
	if on, _ := canvasDot(c, 18, 18); on {
		t.Error("outside the triangle is set")
	}
	c.DrawTexturedTriangle(p0, p1, p2, graphics.UV{}, graphics.UV{}, graphics.UV{}, nil) // no texture: nothing
}
