// Package engine provides the optional Init/Update/View application runtime
// (The Elm Architecture) that powers limoni.NewProgram.
package engine

import "github.com/thebanri/limoni/widgets"

// Msg is an application message delivered to Model.Update.
type Msg any

// UpdateResult describes the work requested after handling a message.
type UpdateResult struct {
	Commands []Cmd
	Redraw   bool
	Quit     bool
	// Commit prints each widget, as tall as it needs, above an inline
	// Program's band and into the terminal's scrollback, in order, with the
	// next frame; see terminal.Terminal.Commit. The widgets are drawn later,
	// on the goroutine that draws: give each its own value, not one the model
	// keeps changing. On a full-screen terminal they are dropped.
	Commit []widgets.Widget
}
