// Package automation exposes a running Limoni application's semantic tree over
// a local socket, so tests and agents can drive it by meaning rather than by
// screen scraping.
//
// # Why this is different from a PTY wrapper
//
// Tools that automate terminal applications today — termwright, mcp-tui-test —
// wrap the process in a pseudo-terminal and parse the rendered character grid.
// They have no choice: the application underneath has no semantics to offer, so
// a test asserts that some text sits at some coordinate, and breaks when the
// layout moves by one column.
//
// A Limoni application already builds a semantic tree every frame for screen
// readers (see core/accessibility). This serves that same tree, which turns
//
//	assert text "Submit" at 42,7; click 42,7
//
// into
//
//	click(role=button, label="Submit")
//
// The selector survives relayout, resizing and restyling, because it never
// mentions where anything is drawn.
//
// # The protocol
//
// Newline-delimited JSON over a Unix domain socket: one JSON request per line,
// one JSON response per line. No dependencies, and `socat` or `nc` is a usable
// client when debugging.
//
// # Security
//
// This opens a socket into a running process and accepts commands that
// synthesise input. It is off unless the application asks for it, the listener
// is a Unix socket with 0600 permissions, and there is deliberately no TCP
// option: a port would expose application control to anything that can reach
// the host. Treat enabling it in production as equivalent to enabling a debug
// console.
package automation

import (
	"fmt"
	"strings"

	"github.com/thebanri/limoni/core/accessibility"
)

// Protocol version, reported in every Hello. Bumped when the wire format
// changes in a way an older client cannot parse.
const Version = 1

// Op names a request. Unknown ops are rejected rather than ignored, so a
// version mismatch surfaces immediately instead of silently doing nothing.
type Op string

const (
	// OpHello returns the protocol version and viewport size.
	OpHello Op = "hello"
	// OpTree returns the semantic tree of the most recent frame.
	OpTree Op = "tree"
	// OpSnapshot returns the rendered text grid. The escape hatch for
	// assertions the semantic tree cannot express.
	OpSnapshot Op = "snapshot"
	// OpFind resolves a selector and returns the matching nodes.
	OpFind Op = "find"
	// OpClick resolves a selector and clicks the centre of its bounds.
	OpClick Op = "click"
	// OpKey synthesises a key press.
	OpKey Op = "key"
	// OpText synthesises a run of character key presses.
	OpText Op = "text"
	// OpFocused returns the focused widget ID.
	OpFocused Op = "focused"
)

// Selector matches nodes in the semantic tree. Empty fields are ignored, and
// all populated fields must match. Matching is on meaning, never position.
type Selector struct {
	// ID matches the widget's focus ID exactly.
	ID string `json:"id,omitempty"`
	// Role matches the semantic role by name, as Role.String reports it
	// ("button", "list", "table"...).
	Role string `json:"role,omitempty"`
	// Label matches the node's label exactly.
	Label string `json:"label,omitempty"`
	// LabelContains matches a substring of the label, for labels that carry
	// live data.
	LabelContains string `json:"label_contains,omitempty"`
	// Value matches the node's value exactly.
	Value string `json:"value,omitempty"`
	// Nth picks one match when several qualify, 0-based. Negative counts from
	// the end. Without it, an ambiguous selector is an error rather than a
	// coin toss.
	Nth *int `json:"nth,omitempty"`
}

// normalizeRole folds case and separators so a hand-written selector matches
// however the caller spelled it: "list-item", "listitem" and "ListItem" are
// the same role. Selectors get typed by hand at a shell prompt, and failing on
// a hyphen would be a bad first experience.
func normalizeRole(role string) string {
	var b strings.Builder
	b.Grow(len(role))
	for _, r := range strings.ToLower(role) {
		if r == '-' || r == '_' || r == ' ' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// IsEmpty reports whether the selector would match every node.
func (s Selector) IsEmpty() bool {
	return s.ID == "" && s.Role == "" && s.Label == "" && s.LabelContains == "" && s.Value == ""
}

// Matches reports whether a single node satisfies the selector.
func (s Selector) Matches(node accessibility.Node) bool {
	if s.ID != "" && node.ID != s.ID {
		return false
	}
	if s.Role != "" && normalizeRole(node.Role.String()) != normalizeRole(s.Role) {
		return false
	}
	if s.Label != "" && node.Label != s.Label {
		return false
	}
	if s.LabelContains != "" && !strings.Contains(node.Label, s.LabelContains) {
		return false
	}
	if s.Value != "" && node.Value != s.Value {
		return false
	}
	return true
}

// String renders the selector the way an error message should read.
func (s Selector) String() string {
	var parts []string
	for _, p := range []struct{ k, v string }{
		{"id", s.ID}, {"role", s.Role}, {"label", s.Label},
		{"label_contains", s.LabelContains}, {"value", s.Value},
	} {
		if p.v != "" {
			parts = append(parts, fmt.Sprintf("%s=%q", p.k, p.v))
		}
	}
	if s.Nth != nil {
		parts = append(parts, fmt.Sprintf("nth=%d", *s.Nth))
	}
	if len(parts) == 0 {
		return "(any)"
	}
	return strings.Join(parts, " ")
}

// Resolve walks the tree depth-first and returns every node the selector
// matches, in render order.
func Resolve(nodes []accessibility.Node, sel Selector) []accessibility.Node {
	var found []accessibility.Node
	var walk func([]accessibility.Node)
	walk = func(list []accessibility.Node) {
		for _, node := range list {
			if sel.Matches(node) {
				found = append(found, node)
			}
			walk(node.Children)
		}
	}
	walk(nodes)
	return found
}

// ResolveOne returns the single node a selector designates.
//
// An ambiguous selector is an error unless Nth says which match is meant.
// Silently taking the first would make a test pass for the wrong reason the
// moment a second matching widget appears.
func ResolveOne(nodes []accessibility.Node, sel Selector) (accessibility.Node, error) {
	matches := Resolve(nodes, sel)
	if len(matches) == 0 {
		return accessibility.Node{}, fmt.Errorf("automation: no node matches %s", sel)
	}
	if sel.Nth == nil {
		if len(matches) > 1 {
			return accessibility.Node{}, fmt.Errorf(
				"automation: %s matches %d nodes; set nth to choose one", sel, len(matches))
		}
		return matches[0], nil
	}
	index := *sel.Nth
	if index < 0 {
		index += len(matches)
	}
	if index < 0 || index >= len(matches) {
		return accessibility.Node{}, fmt.Errorf(
			"automation: nth=%d out of range, %s matches %d nodes", *sel.Nth, sel, len(matches))
	}
	return matches[index], nil
}

// Request is one line of input on the socket.
type Request struct {
	Op       Op       `json:"op"`
	Selector Selector `json:"selector,omitempty"`
	// Key names a key for OpKey: a single character, or one of "enter", "tab",
	// "esc", "space", "backspace", "up", "down", "left", "right", "home",
	// "end", "pgup", "pgdn", "delete".
	Key string `json:"key,omitempty"`
	// Ctrl, Alt and Shift are modifiers for OpKey.
	Ctrl  bool `json:"ctrl,omitempty"`
	Alt   bool `json:"alt,omitempty"`
	Shift bool `json:"shift,omitempty"`
	// Text is the literal string for OpText.
	Text string `json:"text,omitempty"`
}

// Response is one line of output on the socket. Err is set instead of the
// payload fields when the request failed.
type Response struct {
	OK  bool   `json:"ok"`
	Err string `json:"error,omitempty"`

	Version int    `json:"version,omitempty"`
	Width   uint16 `json:"width,omitempty"`
	Height  uint16 `json:"height,omitempty"`

	// AllowInput, ExposeScreen and ExposeInputValues report the server's policy
	// in the hello response, so a client can tell a refusal apart from a bug.
	AllowInput        bool `json:"allow_input,omitempty"`
	ExposeScreen      bool `json:"expose_screen,omitempty"`
	ExposeInputValues bool `json:"expose_input_values,omitempty"`

	Nodes    []accessibility.Node `json:"nodes,omitempty"`
	Snapshot string               `json:"snapshot,omitempty"`
	Focused  string               `json:"focused,omitempty"`
}
