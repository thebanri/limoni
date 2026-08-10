package runtime

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/thebanri/limoni/core/terminal"
)

type programOptions struct {
	model        Model
	messageQueue int
	commandQueue int
	onPanic      func(any)
}

// Option configures a Program.
type Option func(*programOptions)

// WithModel sets the application model.
func WithModel(model Model) Option {
	return func(opts *programOptions) { opts.model = model }
}

// WithMessageQueue sets the bounded application message queue capacity.
func WithMessageQueue(capacity int) Option {
	return func(opts *programOptions) {
		if capacity > 0 {
			opts.messageQueue = capacity
		}
	}
}

// WithCommandQueue sets the bounded command result queue capacity.
func WithCommandQueue(capacity int) Option {
	return func(opts *programOptions) {
		if capacity > 0 {
			opts.commandQueue = capacity
		}
	}
}

// WithPanicHandler installs a handler for panics raised by commands or model
// lifecycle methods. The program remains alive after a command panic.
func WithPanicHandler(handler func(any)) Option {
	return func(opts *programOptions) { opts.onPanic = handler }
}

type commandResult struct {
	sequence uint64
	message  Msg
	panicked any
}

// Program runs a Model and its commands.
type Program struct {
	model Model

	messages       chan Msg
	commandResults chan commandResult
	redraw         chan struct{}

	onPanic func(any)

	sequence atomic.Uint64
	workers  sync.WaitGroup
	stopOnce sync.Once
	stop     chan struct{}
}

// New creates an application program.
func New(options ...Option) *Program {
	opts := programOptions{messageQueue: 64, commandQueue: 64}
	for _, option := range options {
		if option != nil {
			option(&opts)
		}
	}
	return &Program{
		model:          opts.model,
		messages:       make(chan Msg, opts.messageQueue),
		commandResults: make(chan commandResult, opts.commandQueue),
		redraw:         make(chan struct{}, 1),
		onPanic:        opts.onPanic,
		stop:           make(chan struct{}),
	}
}

// Redraws returns a coalescing stream of redraw requests.
func (p *Program) Redraws() <-chan struct{} { return p.redraw }

// Send injects a message, respecting both caller and program cancellation.
func (p *Program) Send(ctx context.Context, message Msg) error {
	if p == nil {
		return context.Canceled
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-p.stop:
		return context.Canceled
	case p.messages <- message:
		return nil
	}
}

// RequestRedraw asks the host renderer for a frame. Multiple requests before
// the host consumes one notification collapse into a single notification.
func (p *Program) RequestRedraw() {
	if p == nil {
		return
	}
	select {
	case p.redraw <- struct{}{}:
	default:
	}
}

// Stop requests graceful shutdown. Run waits for command workers to finish or
// observe their cancelled context before returning.
func (p *Program) Stop() { p.stopOnce.Do(func() { close(p.stop) }) }

// Run starts the model, processes messages, and returns when the context,
// model, or Stop requests shutdown.
func (p *Program) Run(ctx context.Context) error {
	if p == nil || p.model == nil {
		return fmt.Errorf("runtime: model is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	commands := p.callInit()
	for _, command := range commands {
		p.schedule(ctx, command)
	}
	p.RequestRedraw()

	pending := make(map[uint64]commandResult)
	var nextSequence uint64
	for {
		select {
		case <-ctx.Done():
			p.Stop()
			p.workers.Wait()
			return ctx.Err()
		case <-p.stop:
			cancel()
			p.workers.Wait()
			return nil
		case message := <-p.messages:
			if p.update(ctx, message) {
				cancel()
				p.Stop()
				p.workers.Wait()
				return nil
			}
		case result := <-p.commandResults:
			pending[result.sequence] = result
			for {
				ready, ok := pending[nextSequence]
				if !ok {
					break
				}
				delete(pending, nextSequence)
				nextSequence++
				if ready.panicked != nil {
					p.reportPanic(ready.panicked)
					continue
				}
				if ready.message != nil && p.update(ctx, ready.message) {
					cancel()
					p.Stop()
					p.workers.Wait()
					return nil
				}
			}
		}
	}
}

func (p *Program) callInit() (commands []Cmd) {
	defer func() {
		if recovered := recover(); recovered != nil {
			p.reportPanic(recovered)
		}
	}()
	return p.model.Init()
}

func (p *Program) update(ctx context.Context, message Msg) (quit bool) {
	var result UpdateResult
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				p.reportPanic(recovered)
				quit = false
			}
		}()
		result = p.model.Update(message)
	}()
	for _, command := range result.Commands {
		p.schedule(ctx, command)
	}
	if result.Redraw {
		p.RequestRedraw()
	}
	return result.Quit
}

func (p *Program) schedule(parent context.Context, command Cmd) {
	if command == nil {
		return
	}
	sequence := p.sequence.Add(1) - 1
	p.workers.Add(1)
	go func() {
		defer p.workers.Done()
		result := commandResult{sequence: sequence}
		commandCtx, cancel := context.WithCancel(parent)
		defer cancel()
		func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					result.panicked = recovered
				}
			}()
			result.message = command(commandCtx)
		}()
		select {
		case p.commandResults <- result:
		case <-p.stop:
		case <-commandCtx.Done():
		}
	}()
}

func (p *Program) reportPanic(value any) {
	if p.onPanic != nil {
		p.onPanic(value)
	}
}

// View calls the model's View method for hosts that own a terminal frame.
func (p *Program) View(frame *terminal.Frame) {
	if p != nil && p.model != nil {
		p.model.View(frame)
	}
}
