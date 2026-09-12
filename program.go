package limoni

import (
	"context"

	"github.com/thebanri/limoni/core/engine"
)

// Limoni ships two application models. Both drive the same renderer, the same
// widgets, and the same zero-allocation buffer pipeline; they differ only in how
// application state is owned.
//
// Immediate mode (Run / Start) keeps state in your own closure and redraws the
// whole frame on every event. It suits render loops that are naturally
// continuous: dashboards, 3D viewers, games, animations.
//
//	limoni.Run(func(f *limoni.Frame, ev *limoni.Event) bool {
//		f.Draw(limoni.Label("hello"))
//		return ev == nil || ev.Key.Type != limoni.KeyEsc
//	})
//
// Declarative mode (NewProgram) follows The Elm Architecture: a Model owns the
// state, Update folds messages into it, and View renders. It suits
// forms, wizards, CRUD tools, and anything with asynchronous commands, because
// the runtime handles message scheduling, command cancellation, deterministic
// ordering, and panic recovery for you.
//
//	type model struct{ count int }
//
//	func (m *model) Init() []limoni.Cmd { return nil }
//	func (m *model) Update(msg limoni.Msg) limoni.UpdateResult {
//		if key, ok := msg.(limoni.KeyPressMsg); ok && key.Key.Ch == 'q' {
//			return limoni.UpdateResult{Quit: true}
//		}
//		return limoni.UpdateResult{Redraw: true}
//	}
//	func (m *model) View(f *limoni.Frame) { /* ... */ }
//
//	prog := limoni.NewProgram(&model{}, limoni.WithAltScreen())
//	err := prog.Run(ctx)
//
// Declarative mode is context-aware today: Program.Run and Program.RunTerminal
// both take a context.Context and shut down cleanly when it is cancelled.
//
// The full runtime lives in core/engine; the aliases below exist so the common
// path never needs a second import.

// Re-exported application runtime types.
type (
	// Model is the Init/Update/View contract implemented by declarative applications.
	Model = engine.Model
	// Msg is an application message delivered to Model.Update.
	Msg = engine.Msg
	// Cmd is a cancellable side effect that eventually produces a Msg.
	Cmd = engine.Cmd
	// UpdateResult describes the work requested after handling a message.
	UpdateResult = engine.UpdateResult
	// Program is the declarative application runtime.
	Program = engine.Program
	// ProgramOption configures a Program created with NewProgram.
	ProgramOption = engine.Option
)

// Re-exported runtime messages delivered to Model.Update.
type (
	KeyPressMsg     = engine.KeyPressMsg
	KeyReleaseMsg   = engine.KeyReleaseMsg
	MousePressMsg   = engine.MousePressMsg
	MouseReleaseMsg = engine.MouseReleaseMsg
	MouseWheelMsg   = engine.MouseWheelMsg
	PasteMsg        = engine.PasteMsg
	ResizeMsg       = engine.ResizeMsg
	FocusMsg        = engine.FocusMsg
	BlurMsg         = engine.BlurMsg
)

// NewProgram creates a declarative (Elm architecture) application runtime for
// model. Run it with Program.Run for full control over the terminal, or
// Program.RunTerminal for the batteries-included path.
func NewProgram(model Model, opts ...ProgramOption) *Program {
	options := make([]ProgramOption, 0, len(opts)+1)
	options = append(options, engine.WithModel(model))
	options = append(options, opts...)
	return engine.New(options...)
}

// RunProgram is the batteries-included entry point for declarative
// applications. It initialises the terminal, runs model until it quits or ctx is
// cancelled, and restores the terminal state on the way out.
//
//	if err := limoni.RunProgram(context.Background(), &myModel{}); err != nil {
//		log.Fatal(err)
//	}
//
// Use NewProgram directly when you need to own the terminal lifecycle, drive a
// custom backend, or render into an SSH session.
func RunProgram(ctx context.Context, model Model, opts ...ProgramOption) error {
	term, err := New()
	if err != nil {
		return err
	}
	defer term.Close()
	return NewProgram(model, opts...).RunTerminal(ctx, term, term.Backend())
}

// MessageFromEvent converts a driver event into the corresponding runtime Msg.
// Useful when feeding a Program from a custom backend or from a test harness.
func MessageFromEvent(event Event) Msg { return engine.MessageFromDriver(event) }

// WithAltScreen renders the Program on the terminal's alternate screen buffer.
func WithAltScreen() ProgramOption { return engine.WithAltScreen() }

// WithMessageQueue sets the Program's inbound message queue capacity.
func WithMessageQueue(capacity int) ProgramOption { return engine.WithMessageQueue(capacity) }

// WithCommandQueue sets the Program's outbound command queue capacity.
func WithCommandQueue(capacity int) ProgramOption { return engine.WithCommandQueue(capacity) }

// WithPanicHandler installs a recovery handler invoked when a Model method or a
// Cmd panics, instead of tearing down the process.
func WithPanicHandler(handler func(any)) ProgramOption { return engine.WithPanicHandler(handler) }

// WithProgramFPS sets a continuous redraw rate for a Program.
//
// It mirrors WithFPS, which configures immediate-mode Run instead. The two
// application models take different option types, so the names cannot be shared.
func WithProgramFPS(fps int) ProgramOption { return engine.WithFPS(fps) }

// WithProgramCatchCtrlC configures whether a Program receives Ctrl+C as a normal
// key message instead of terminating. It mirrors WithCatchCtrlC for Run.
func WithProgramCatchCtrlC(catch bool) ProgramOption { return engine.WithCatchCtrlC(catch) }

// WithoutProgramQuitKeys disables a Program's automatic termination on Ctrl+C.
// It mirrors WithoutDefaultQuitKeys for Run.
func WithoutProgramQuitKeys() ProgramOption { return engine.WithoutDefaultQuitKeys() }
