// Package runtime provides an optional Init/Update/View application runtime.
package runtime

// Msg is an application message delivered to Model.Update.
type Msg any

// UpdateResult describes the work requested after handling a message.
type UpdateResult struct {
	Commands []Cmd
	Redraw   bool
	Quit     bool
}
