// Package engine provides the optional Init/Update/View application runtime
// (The Elm Architecture) that powers limoni.NewProgram.
package engine

// Msg is an application message delivered to Model.Update.
type Msg any

// UpdateResult describes the work requested after handling a message.
type UpdateResult struct {
	Commands []Cmd
	Redraw   bool
	Quit     bool
}
