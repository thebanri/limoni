package widgets

// ListProvider provides items lazily for large scrollable lists.
type ListProvider interface {
	Len() int
	ItemAt(index int) string
}
