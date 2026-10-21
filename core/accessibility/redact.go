package accessibility

// Redact returns a copy of a tree with the values that must not leave the
// process removed. It is the single implementation of that rule, shared by
// everything that carries a tree outward — the automation socket and session
// recordings — so the two cannot drift apart.
//
// Sensitive nodes always lose their value and description, whatever the
// caller asks for: a widget that marks itself sensitive is supposed to leave
// Value empty, and clearing it again here means one that forgets does not
// leak. Input nodes lose their value unless exposeInputValues is set, because
// a text field is where users type things an application did not think to
// mark secret.
//
// The input tree is not modified. It belongs to the frame that produced it.
func Redact(nodes []Node, exposeInputValues bool) []Node {
	if nodes == nil {
		return nil
	}
	out := make([]Node, len(nodes))
	for i, node := range nodes {
		if node.State&StateSensitive != 0 {
			node.Value = ""
			node.Description = ""
		} else if node.Role == RoleInput && !exposeInputValues {
			node.Value = ""
		}
		node.Children = Redact(node.Children, exposeInputValues)
		out[i] = node
	}
	return out
}
