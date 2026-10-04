package widgets

import "github.com/thebanri/limoni/core/cell"

// ListProvider provides items lazily for large scrollable lists.
type ListProvider interface {
	Len() int
	ItemAt(index int) string
}

// ListStyler is a ListProvider that styles its own rows — read items dimmed,
// favourites in bold. StyleAt's style is merged over the list's, and the
// SelectedStyle over that on the selected row. It is asked on every draw, so
// it must not allocate.
type ListStyler interface {
	StyleAt(index int) cell.Style
}
