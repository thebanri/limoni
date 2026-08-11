package accessibility

import (
	"fmt"
	"io"
	"strings"
)

// String returns the stable, human-readable name used by line-oriented
// accessibility output.
func (r Role) String() string {
	switch r {
	case RoleButton:
		return "button"
	case RoleCheckbox:
		return "checkbox"
	case RoleInput:
		return "input"
	case RoleList:
		return "list"
	case RoleListItem:
		return "list-item"
	case RoleTable:
		return "table"
	case RoleDialog:
		return "dialog"
	case RoleProgress:
		return "progress"
	case RoleImage:
		return "image"
	default:
		return "generic"
	}
}

// WriteLineMode writes the semantic tree directly to an output sink and adds
// a trailing newline when nodes are present. It is suitable for a pipe,
// screen-reader bridge, or log stream.
func (m Mode) WriteLineMode(w io.Writer, nodes []AccessibilityNode) error {
	if w == nil {
		return fmt.Errorf("accessibility: nil line-mode writer")
	}
	text := m.LineMode(nodes)
	if text == "" {
		return nil
	}
	_, err := io.WriteString(w, text+"\n")
	return err
}

// StateNames returns state flags in a stable order so screen-reader output is
// deterministic across runs and independent of bit layout changes.
func (s NodeState) StateNames() []string {
	states := make([]string, 0, 7)
	for _, state := range []struct {
		flag NodeState
		name string
	}{
		{StateFocused, "focused"},
		{StateDisabled, "disabled"},
		{StateSelected, "selected"},
		{StateExpanded, "expanded"},
		{StateChecked, "checked"},
		{StateBusy, "busy"},
		{StateInvalid, "invalid"},
	} {
		if s&state.flag != 0 {
			states = append(states, state.name)
		}
	}
	return states
}

// LineMode serializes a semantic tree as one node per line. The output is
// intended for screen readers, logs, and low-capability terminals; it does
// not depend on terminal colors or cell rendering.
//
// Each line contains the node role and label, followed by optional value,
// description, state, and bounds fields. Child nodes are indented by two
// spaces to preserve hierarchy while remaining easy to consume line by line.
func (m Mode) LineMode(nodes []AccessibilityNode) string {
	var b strings.Builder
	for i, node := range nodes {
		if i > 0 {
			b.WriteByte('\n')
		}
		writeLineNode(&b, node, 0, m.Normalize())
	}
	return b.String()
}

func writeLineNode(b *strings.Builder, node AccessibilityNode, depth int, mode Mode) {
	if depth > 0 {
		b.WriteString(strings.Repeat("  ", depth))
	}
	b.WriteString(node.Role.String())
	if node.ID != "" {
		fmt.Fprintf(b, "#%s", mode.TextFallback(node.ID))
	}
	if node.Label != "" {
		fmt.Fprintf(b, " %q", mode.TextFallback(node.Label))
	}
	if node.Value != "" {
		fmt.Fprintf(b, " value=%q", mode.TextFallback(node.Value))
	}
	if node.Description != "" {
		fmt.Fprintf(b, " description=%q", mode.TextFallback(node.Description))
	}
	if states := node.State.StateNames(); len(states) > 0 {
		b.WriteString(" state=")
		b.WriteString(strings.Join(states, ","))
	}
	fmt.Fprintf(b, " bounds=%d,%d %dx%d", node.Bounds.X, node.Bounds.Y, node.Bounds.Width, node.Bounds.Height)
	for _, child := range node.Children {
		b.WriteByte('\n')
		writeLineNode(b, child, depth+1, mode)
	}
}
