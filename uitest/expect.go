package uitest

import (
	"fmt"
	"strings"

	"github.com/thebanri/limoni/core/accessibility"
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
	a.run("be visible", func(matches []accessibility.AccessibilityNode) (bool, string) {
		return len(matches) > 0, fmt.Sprintf("%d match(es)", len(matches))
	})
}

// ToHaveCount passes when the locator matches exactly n widgets.
func (a Assertion) ToHaveCount(n int) {
	a.locator.page.t.Helper()
	a.run(fmt.Sprintf("have %d match(es)", n), func(matches []accessibility.AccessibilityNode) (bool, string) {
		return len(matches) == n, fmt.Sprintf("%d match(es)", len(matches))
	})
}

// ToHaveValue passes when the one matching widget's value is exactly value.
func (a Assertion) ToHaveValue(value string) {
	a.locator.page.t.Helper()
	a.one(fmt.Sprintf("have value %q", value), func(n accessibility.AccessibilityNode) (bool, string) {
		return n.Value == value, fmt.Sprintf("value %q", n.Value)
	})
}

// ToHaveLabel passes when the one matching widget's label is exactly label.
func (a Assertion) ToHaveLabel(label string) {
	a.locator.page.t.Helper()
	a.one(fmt.Sprintf("have label %q", label), func(n accessibility.AccessibilityNode) (bool, string) {
		return n.Label == label, fmt.Sprintf("label %q", n.Label)
	})
}

// ToContainLabel passes when the one matching widget's label contains text.
func (a Assertion) ToContainLabel(text string) {
	a.locator.page.t.Helper()
	a.one(fmt.Sprintf("have a label containing %q", text), func(n accessibility.AccessibilityNode) (bool, string) {
		return strings.Contains(n.Label, text), fmt.Sprintf("label %q", n.Label)
	})
}

// ToContainValue passes when the one matching widget's value contains text.
// A paragraph's value is its text, so this is how to check what one says
// without spelling out all of it.
func (a Assertion) ToContainValue(text string) {
	a.locator.page.t.Helper()
	a.one(fmt.Sprintf("have a value containing %q", text), func(n accessibility.AccessibilityNode) (bool, string) {
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

func (a Assertion) state(what string, flag accessibility.NodeState) {
	a.locator.page.t.Helper()
	a.one(what, func(n accessibility.AccessibilityNode) (bool, string) {
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
func (a Assertion) one(what string, check func(accessibility.AccessibilityNode) (bool, string)) {
	a.locator.page.t.Helper()
	a.check(what, func(tree []accessibility.AccessibilityNode) (bool, string, error) {
		node, err := a.locator.one(tree)
		if err != nil {
			return false, "", err
		}
		ok, got := check(node)
		return ok, got, nil
	})
}

func (a Assertion) run(what string, check func([]accessibility.AccessibilityNode) (bool, string)) {
	a.locator.page.t.Helper()
	a.check(what, func(tree []accessibility.AccessibilityNode) (bool, string, error) {
		matches, err := a.locator.resolve(tree)
		if err != nil {
			return false, "", err
		}
		ok, got := check(matches)
		return ok, got, nil
	})
}

func (a Assertion) check(what string, check func([]accessibility.AccessibilityNode) (bool, string, error)) {
	page := a.locator.page
	page.t.Helper()
	expectation := "to " + what
	if a.negate {
		expectation = "not to " + what
	}
	_, err := page.eventually(func(tree []accessibility.AccessibilityNode) error {
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
