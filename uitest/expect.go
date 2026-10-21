package uitest

import (
	"fmt"
	"strings"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/cell"
)

// Assertion checks a locator against the application, retrying until the
// check holds or the page's timeout passes. Create one with Page.Expect.
type Assertion struct {
	locator Locator
	negate  bool
}

// Expect starts an assertion about a locator.
//
//	page.Expect(page.GetByID("status")).ToHaveLabel("Saved")
//	page.Expect(page.GetByRole("dialog", "")).Not().ToBeVisible()
func (p *Page) Expect(l Locator) Assertion {
	return Assertion{locator: l}
}

// Not inverts the assertion that follows: it passes once the check stops
// holding. Waiting for a spinner to disappear reads
// Expect(spinner).Not().ToBeVisible().
func (a Assertion) Not() Assertion {
	a.negate = !a.negate
	return a
}

// ToBeVisible passes when the locator matches at least one widget. With Not,
// when it matches none.
func (a Assertion) ToBeVisible() {
	a.locator.page.t.Helper()
	a.run("be visible", func(matches []accessibility.Node) (bool, string) {
		return len(matches) > 0, fmt.Sprintf("%d match(es)", len(matches))
	})
}

// ToHaveCount passes when the locator matches exactly n widgets.
func (a Assertion) ToHaveCount(n int) {
	a.locator.page.t.Helper()
	a.run(fmt.Sprintf("have %d match(es)", n), func(matches []accessibility.Node) (bool, string) {
		return len(matches) == n, fmt.Sprintf("%d match(es)", len(matches))
	})
}

// ToHaveValue passes when the one matching widget's value is exactly value.
func (a Assertion) ToHaveValue(value string) {
	a.locator.page.t.Helper()
	a.one(fmt.Sprintf("have value %q", value), func(n accessibility.Node) (bool, string) {
		return n.Value == value, fmt.Sprintf("value %q", n.Value)
	})
}

// ToHaveLabel passes when the one matching widget's label is exactly label.
func (a Assertion) ToHaveLabel(label string) {
	a.locator.page.t.Helper()
	a.one(fmt.Sprintf("have label %q", label), func(n accessibility.Node) (bool, string) {
		return n.Label == label, fmt.Sprintf("label %q", n.Label)
	})
}

// ToContainLabel passes when the one matching widget's label contains text.
func (a Assertion) ToContainLabel(text string) {
	a.locator.page.t.Helper()
	a.one(fmt.Sprintf("have a label containing %q", text), func(n accessibility.Node) (bool, string) {
		return strings.Contains(n.Label, text), fmt.Sprintf("label %q", n.Label)
	})
}

// ToContainValue passes when the one matching widget's value contains text.
// A paragraph's value is its text, so this is how to check what one says
// without spelling out all of it.
func (a Assertion) ToContainValue(text string) {
	a.locator.page.t.Helper()
	a.one(fmt.Sprintf("have a value containing %q", text), func(n accessibility.Node) (bool, string) {
		return strings.Contains(n.Value, text), fmt.Sprintf("value %q", n.Value)
	})
}

// ToBeFocused passes when the one matching widget has keyboard focus.
func (a Assertion) ToBeFocused() {
	a.locator.page.t.Helper()
	a.state("be focused", accessibility.StateFocused)
}

// ToBeSelected passes when the one matching widget is selected.
func (a Assertion) ToBeSelected() {
	a.locator.page.t.Helper()
	a.state("be selected", accessibility.StateSelected)
}

// ToBeChecked passes when the one matching widget is checked.
func (a Assertion) ToBeChecked() {
	a.locator.page.t.Helper()
	a.state("be checked", accessibility.StateChecked)
}

// ToBeDisabled passes when the one matching widget is disabled.
func (a Assertion) ToBeDisabled() {
	a.locator.page.t.Helper()
	a.state("be disabled", accessibility.StateDisabled)
}

// ToHaveStyle passes when every on-screen cell of the one matching widget is
// drawn with want: its foreground and background where want sets them, and at
// least its modifiers. With Not, when some cell is not. The tree only says a
// widget has the focus; this checks that the focus can be seen:
//
//	page.Expect(page.GetByRole("list-item", "Two")).ToHaveStyle(cell.Style{Modifier: cell.ModifierReverse})
//
// Styles are read from the frame in process (Run and Program); the automation
// socket sends text only, so a page from Connect fails the test.
func (a Assertion) ToHaveStyle(want cell.Style) {
	page := a.locator.page
	page.t.Helper()
	styles, ok := page.app.(cellStyles)
	if !ok {
		page.t.Fatalf("uitest: %s: styles can only be checked in process (Run or Program); the automation socket sends text", a.locator)
	}
	a.one("have style "+describeStyle(want), func(n accessibility.Node) (bool, string) {
		cells := 0
		for y := n.Bounds.Y; y < n.Bounds.Y+n.Bounds.Height; y++ {
			for x := n.Bounds.X; x < n.Bounds.X+n.Bounds.Width; x++ {
				got, ok := styles.styleAt(x, y)
				if !ok {
					continue
				}
				cells++
				if (want.Fg.Type() != cell.ColorDefault && got.Fg != want.Fg) ||
					(want.Bg.Type() != cell.ColorDefault && got.Bg != want.Bg) ||
					got.Modifier&want.Modifier != want.Modifier {
					return false, fmt.Sprintf("cell %d,%d drawn %s", x, y, describeStyle(got))
				}
			}
		}
		if cells == 0 {
			return false, "no cells on screen"
		}
		return true, fmt.Sprintf("all %d cell(s) drawn with it", cells)
	})
}

var modifierNames = [...]string{"bold", "dim", "italic", "underline", "blink", "reverse", "hidden", "strikethrough", "double-underline", "undercurl"}

func describeStyle(s cell.Style) string {
	var parts []string
	if s.Fg.Type() != cell.ColorDefault {
		parts = append(parts, "fg "+describeColor(s.Fg))
	}
	if s.Bg.Type() != cell.ColorDefault {
		parts = append(parts, "bg "+describeColor(s.Bg))
	}
	for i, name := range modifierNames {
		if s.Modifier&(1<<i) != 0 {
			parts = append(parts, name)
		}
	}
	if len(parts) == 0 {
		return "plain"
	}
	return strings.Join(parts, " ")
}

func describeColor(c cell.Color) string {
	if c.Type() == cell.ColorANSI {
		return fmt.Sprintf("ansi %d", c.ANSI())
	}
	r, g, b := c.RGB()
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

func (a Assertion) state(what string, flag accessibility.NodeState) {
	a.locator.page.t.Helper()
	a.one(what, func(n accessibility.Node) (bool, string) {
		states := n.State.StateNames()
		if len(states) == 0 {
			return n.State&flag != 0, "no states"
		}
		return n.State&flag != 0, "states " + strings.Join(states, ",")
	})
}

// one runs a check that needs exactly one matching widget. An ambiguous or
// missing match is a failure either way, negated or not: "the button is not
// focused" says nothing true when there is no such button.
func (a Assertion) one(what string, check func(accessibility.Node) (bool, string)) {
	a.locator.page.t.Helper()
	a.check(what, func(tree []accessibility.Node) (bool, string, error) {
		node, err := a.locator.one(tree)
		if err != nil {
			return false, "", err
		}
		ok, got := check(node)
		return ok, got, nil
	})
}

func (a Assertion) run(what string, check func([]accessibility.Node) (bool, string)) {
	a.locator.page.t.Helper()
	a.check(what, func(tree []accessibility.Node) (bool, string, error) {
		matches, err := a.locator.resolve(tree)
		if err != nil {
			return false, "", err
		}
		ok, got := check(matches)
		return ok, got, nil
	})
}

func (a Assertion) check(what string, check func([]accessibility.Node) (bool, string, error)) {
	page := a.locator.page
	page.t.Helper()
	expectation := "to " + what
	if a.negate {
		expectation = "not to " + what
	}
	_, err := page.eventually(func(tree []accessibility.Node) error {
		ok, got, err := check(tree)
		if err != nil {
			return err
		}
		if ok == a.negate {
			return fmt.Errorf("got %s", got)
		}
		return nil
	})
	if err != nil {
		page.t.Fatalf("uitest: expected %s %s: %v", a.locator, expectation, err)
	}
}
