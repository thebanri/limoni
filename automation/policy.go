package automation

import "github.com/thebanri/limoni/core/accessibility"

// Policy decides what the server lets leave the process.
//
// Every field defaults to the closed position. The server is a gateway between
// a running application and whatever connects to its socket, and the safe
// default for a gateway is to expose structure — roles, labels, positions,
// bounds — and nothing a user typed or anything the application did not mark as
// safe to show.
type Policy struct {
	// ExposeInputValues sends the value of input nodes (text fields, text areas).
	// Off by default because a text field is where a user types things that
	// matter: an email, a search, a card number that the application did not
	// think to mark Secret.
	//
	// Nodes marked StateSensitive are never exposed, whatever this says.
	ExposeInputValues bool

	// ExposeScreen allows the snapshot op, which returns the rendered grid.
	// Off by default because the grid contains every character on screen, and
	// the application cannot mark an arbitrary paragraph as secret.
	ExposeScreen bool

	// AllowInput permits key, text and click synthesis. Off by default: with it
	// off, a client can observe the application but cannot act on it.
	AllowInput bool

	// AllowUnverifiedPeers accepts connections whose owning user the kernel
	// cannot report. Linux, macOS and FreeBSD report it, and there a connection
	// from any other user is always refused. Elsewhere — Windows among them —
	// the socket's file permissions are the only protection, so the server
	// refuses every connection unless this is set.
	AllowUnverifiedPeers bool
}

// redactTree applies the shared redaction rule with this policy's setting.
func (p Policy) redactTree(nodes []accessibility.Node) []accessibility.Node {
	return accessibility.Redact(nodes, p.ExposeInputValues)
}
