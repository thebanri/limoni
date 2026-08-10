package runtime

import "github.com/thebanri/limoni/core/terminal"

// Model is the application contract used by Program.
type Model interface {
	Init() []Cmd
	Update(Msg) UpdateResult
	View(*terminal.Frame)
}
