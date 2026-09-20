package uitest

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/thebanri/limoni/automation"
	"github.com/thebanri/limoni/core/accessibility"
)

// Locator finds widgets in the semantic tree. It is a query, not a handle:
// every action and assertion resolves it again against the current frame.
//
// Locators are values. Refining one — WithValue, Nth, Within — returns a new
// locator and leaves the original unchanged.
type Locator struct {
	page   *Page
	sel    automation.Selector
	parent *Locator
}

// GetByRole finds widgets with a role — "button", "input", "list",
// "list-item", "checkbox", "table", "dialog" — and, if label is not empty,
// exactly that label.
func (p *Page) GetByRole(role, label string) Locator {
	return Locator{page: p, sel: automation.Selector{Role: role, Label: label}}
}

// GetByLabel finds widgets whose label is exactly label, whatever their role.
func (p *Page) GetByLabel(label string) Locator {
	return Locator{page: p, sel: automation.Selector{Label: label}}
}

// GetByText finds widgets whose label contains text, for labels that carry
// live data such as a count or a status.
func (p *Page) GetByText(text string) Locator {
	return Locator{page: p, sel: automation.Selector{LabelContains: text}}
}

// GetByID finds the widget with this ID — the identifier it was rendered with,
// and the most stable way to address it.
func (p *Page) GetByID(id string) Locator {
	return Locator{page: p, sel: automation.Selector{ID: id}}
}

// WithValue narrows the locator to widgets whose value is exactly value: the
// text of an input, the selected item of a list.
func (l Locator) WithValue(value string) Locator {
	l.sel.Value = value
	return l
}

// Nth picks one match when several qualify, 0-based; negative counts from the
// end. Without it, acting on an ambiguous locator fails the test, because
// silently taking the first match is how a test starts passing for the wrong
// reason.
func (l Locator) Nth(index int) Locator {
	l.sel.Nth = &index
	return l
}

// Within looks for matches only inside the one widget parent resolves to —
// the rows of one particular list, the buttons of one dialog.
func (l Locator) Within(parent Locator) Locator {
	l.parent = &parent
	return l
}

// String describes the locator the way a failure message should.
func (l Locator) String() string {
	s := l.sel.String()
	if l.parent != nil {
		s += " within " + l.parent.String()
	}
	return s
}

// resolve returns every match in the current frame. With Nth set it returns
// the chosen match alone.
func (l Locator) resolve(tree []accessibility.AccessibilityNode) ([]accessibility.AccessibilityNode, error) {
	scope := tree
	if l.parent != nil {
		parents, err := l.parent.resolve(tree)
		switch {
		case err != nil:
			return nil, err
		case len(parents) == 0:
			// No parent, so nothing inside it: zero matches, which is what
			// Not().ToBeVisible should see once a dialog has closed.
			return nil, nil
		case len(parents) > 1:
			return nil, fmt.Errorf("%s matches %d widgets; narrow it, or use Nth to choose one", *l.parent, len(parents))
		}
		scope = parents[0].Children
	}
	matches := automation.Resolve(scope, withoutNth(l.sel))
	if l.sel.Nth == nil {
		return matches, nil
	}
	index := *l.sel.Nth
	if index < 0 {
		index += len(matches)
	}
	if index < 0 || index >= len(matches) {
		return nil, nil
	}
	return matches[index : index+1], nil
}

func (l Locator) one(tree []accessibility.AccessibilityNode) (accessibility.AccessibilityNode, error) {
	matches, err := l.resolve(tree)
	if err != nil {
		return accessibility.AccessibilityNode{}, err
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return accessibility.AccessibilityNode{}, fmt.Errorf("nothing matches %s", l)
	default:
		return accessibility.AccessibilityNode{}, fmt.Errorf("%s matches %d widgets; narrow it, or use Nth to choose one", l, len(matches))
	}
}

func withoutNth(sel automation.Selector) automation.Selector {
	sel.Nth = nil
	return sel
}

// wait resolves the locator to exactly one widget, retrying until it does or
// the page's timeout passes.
func (l Locator) wait() (accessibility.AccessibilityNode, error) {
	var lastErr error
	var lastTree []accessibility.AccessibilityNode
	deadline := time.Now().Add(l.page.timeout)
	for {
		tree, err := l.page.app.tree()
		if errors.Is(err, errExited) {
			return accessibility.AccessibilityNode{}, err
		}
		if err == nil {
			lastTree = tree
			node, resolveErr := l.one(tree)
			if resolveErr == nil {
				return node, nil
			}
			lastErr = resolveErr
		} else {
			lastErr = err
		}
		if !time.Now().Before(deadline) {
			return accessibility.AccessibilityNode{}, fmt.Errorf("%v after %s%s", lastErr, l.page.timeout, treeDump(lastTree))
		}
		time.Sleep(pollInterval)
	}
}

// pollInterval is how often a waiting action or assertion looks again. An
// in-process application also renders a frame each time.
const pollInterval = 10 * time.Millisecond

// Click waits for the locator to match one widget and clicks its centre.
func (l Locator) Click() {
	l.page.t.Helper()
	node, err := l.wait()
	if err != nil {
		l.page.t.Fatalf("uitest: click %s: %v", l, err)
	}
	if node.Bounds.Width == 0 || node.Bounds.Height == 0 {
		l.page.t.Fatalf("uitest: click %s: the widget has no area on screen", l)
	}
	if err := l.page.app.click(node, l.selectorFor(node)); err != nil {
		l.page.t.Fatalf("uitest: click %s: %v", l, err)
	}
	l.page.acted("click %s", l)
}

// Check makes a checkbox checked. It clicks only if the checkbox is not checked
// already, then waits until it is, so repeating it is harmless. Click toggles:
// an agent or a test that runs a step twice with Click undoes it.
func (l Locator) Check() {
	l.page.t.Helper()
	l.ensure("check", accessibility.StateChecked, true)
}

// Uncheck makes a checkbox unchecked, clicking only if it is checked.
func (l Locator) Uncheck() {
	l.page.t.Helper()
	l.ensure("uncheck", accessibility.StateChecked, false)
}

// Select makes the widget selected — a list item, a table row, a tab, a tree
// item or a radio button — clicking only if it is not selected already, then
// waiting until it is.
func (l Locator) Select() {
	l.page.t.Helper()
	l.ensure("select", accessibility.StateSelected, true)
}

// ensure clicks the widget unless its state already has flag set (or clear,
// for want false), and then waits for the click to have that effect.
func (l Locator) ensure(verb string, flag accessibility.NodeState, want bool) {
	l.page.t.Helper()
	node, err := l.wait()
	if err != nil {
		l.page.t.Fatalf("uitest: %s %s: %v", verb, l, err)
	}
	if (node.State&flag != 0) == want {
		l.page.acted("%s %s: already done", verb, l)
		return
	}
	if node.Bounds.Width == 0 || node.Bounds.Height == 0 {
		l.page.t.Fatalf("uitest: %s %s: the widget has no area on screen", verb, l)
	}
	if err := l.page.app.click(node, l.selectorFor(node)); err != nil {
		l.page.t.Fatalf("uitest: %s %s: %v", verb, l, err)
	}
	_, err = l.page.eventually(func(tree []accessibility.AccessibilityNode) error {
		now, err := l.one(tree)
		if err != nil {
			return err
		}
		if (now.State&flag != 0) != want {
			return fmt.Errorf("clicked, but its state is still %v", now.State.StateNames())
		}
		return nil
	})
	if err != nil {
		l.page.t.Fatalf("uitest: %s %s: %v", verb, l, err)
	}
	l.page.acted("%s %s", verb, l)
}

// selectorFor is the selector a remote application resolves a click with. A
// locator scoped with Within has no single-selector equivalent, so it is
// pinned to what it resolved to here.
func (l Locator) selectorFor(node accessibility.AccessibilityNode) automation.Selector {
	if l.parent == nil {
		return l.sel
	}
	if node.ID != "" {
		return automation.Selector{ID: node.ID}
	}
	return automation.Selector{Role: node.Role.String(), Label: node.Label, Value: node.Value}
}

// Type focuses the widget, by clicking it unless it already has focus, and
// types text into it one key press per character. It does not clear what is
// already there, and it does not press enter.
func (l Locator) Type(text string) {
	l.page.t.Helper()
	l.focus("type into")
	if err := l.page.app.typeText(text); err != nil {
		l.page.t.Fatalf("uitest: type into %s: %v", l, err)
	}
	// The text itself is not logged: it may be a password.
	l.page.acted("type %d character(s) into %s", len([]rune(text)), l)
}

// Press focuses the widget, by clicking it unless it already has focus, and
// presses key — see Page.Press for the names.
func (l Locator) Press(key string) {
	l.page.t.Helper()
	ev, name, err := parseChord(key)
	if err != nil {
		l.page.t.Fatalf("uitest: %v", err)
	}
	l.focus("press " + key + " on")
	if err := l.page.app.key(ev, name); err != nil {
		l.page.t.Fatalf("uitest: press %s on %s: %v", key, l, err)
	}
	l.page.acted("press %s on %s", key, l)
}

func (l Locator) focus(action string) {
	l.page.t.Helper()
	node, err := l.wait()
	if err != nil {
		l.page.t.Fatalf("uitest: %s %s: %v", action, l, err)
	}
	if node.State&accessibility.StateFocused != 0 {
		return
	}
	l.Click()
	// The click has to have moved focus before keys are sent, or they land
	// wherever focus was.
	if _, err := l.page.eventually(func(tree []accessibility.AccessibilityNode) error {
		node, err := l.one(tree)
		if err != nil {
			return err
		}
		if node.State&accessibility.StateFocused == 0 {
			return fmt.Errorf("%s did not take focus when clicked", l)
		}
		return nil
	}); err != nil {
		l.page.t.Fatalf("uitest: %s %s: %v", action, l, err)
	}
}

// Count returns how many widgets match right now, without waiting. Prefer
// Expect(...).ToHaveCount in assertions, which does wait.
func (l Locator) Count() int {
	l.page.t.Helper()
	tree := l.page.Tree()
	matches, err := l.resolve(tree)
	if err != nil {
		return 0
	}
	return len(matches)
}

// Node waits for the locator to match one widget and returns its node, for
// checks the assertions do not cover.
func (l Locator) Node() accessibility.AccessibilityNode {
	l.page.t.Helper()
	node, err := l.wait()
	if err != nil {
		l.page.t.Fatalf("uitest: %s: %v", l, err)
	}
	return node
}

// eventually retries check against fresh frames until it passes or the
// page's timeout runs out, returning the last tree it saw.
func (p *Page) eventually(check func([]accessibility.AccessibilityNode) error) ([]accessibility.AccessibilityNode, error) {
	var lastErr error
	var lastTree []accessibility.AccessibilityNode
	deadline := time.Now().Add(p.timeout)
	for {
		tree, err := p.app.tree()
		if errors.Is(err, errExited) {
			return lastTree, err
		}
		if err == nil {
			lastTree = tree
			if lastErr = check(tree); lastErr == nil {
				return tree, nil
			}
		} else {
			lastErr = err
		}
		if !time.Now().Before(deadline) {
			return lastTree, fmt.Errorf("%v after %s%s", lastErr, p.timeout, treeDump(lastTree))
		}
		time.Sleep(pollInterval)
	}
}

func treeDump(tree []accessibility.AccessibilityNode) string {
	if len(tree) == 0 {
		return "\n(the frame has no semantic nodes)"
	}
	var b strings.Builder
	b.WriteString("\nlast frame:\n")
	for _, line := range strings.Split(accessibility.Mode{}.LineMode(tree), "\n") {
		b.WriteString("  ")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}
