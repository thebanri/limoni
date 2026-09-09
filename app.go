package limoni

// AppOption configures the application lifecycle in Run.
type AppOption func(*appConfig)

type appConfig struct {
	catchCtrlC bool
}

// WithCatchCtrlC configures whether Ctrl+C is forwarded to the application
// as a normal key event instead of automatically terminating the process.
func WithCatchCtrlC(catch bool) AppOption {
	return func(c *appConfig) {
		c.catchCtrlC = catch
	}
}

// WithoutDefaultQuitKeys disables automatic termination on Ctrl+C.
// When enabled, Ctrl+C is forwarded to the application's event handler.
func WithoutDefaultQuitKeys() AppOption {
	return WithCatchCtrlC(true)
}

// Run starts an event-driven Limoni application loop.
// appFn is invoked with the active Frame and the triggering Event.
// Returning false from appFn cleanly exits the application.
// On the first invocation, ev is nil for the initial render.
// By default, Ctrl+C automatically terminates the application gracefully,
// unless WithCatchCtrlC(true) or WithoutDefaultQuitKeys() is supplied.
func Run(appFn func(f *Frame, ev *Event) bool, opts ...AppOption) error {
	var cfg appConfig
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	term, err := New()
	if err != nil {
		return err
	}
	defer term.Close()
	term.StartEventLoop()

	running := true
	// Initial render
	err = term.Draw(func(f *Frame) {
		running = appFn(f, nil)
	})
	if err != nil || !running {
		return err
	}

	events := term.Events()
	for running {
		select {
		case ev, ok := <-events:
			if !ok {
				return nil
			}
			// Automatic graceful exit on Ctrl+C unless explicitly caught
			if !cfg.catchCtrlC && ev.Type == EventKey && ev.Key.Ctrl && (ev.Key.Ch == 'c' || ev.Key.Ch == 'C') {
				return nil
			}
			// Route mouse event through terminal hit-test router
			if ev.Type == EventMouse {
				term.RouteMouseEvent(ev.Mouse)
			}
			err = term.Draw(func(f *Frame) {
				running = appFn(f, &ev)
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Start displays a static or single-state frame and cleanly exits when 'q', 'ESC', or Ctrl+C is pressed.
func Start(drawFn func(f *Frame)) error {
	return Run(func(f *Frame, ev *Event) bool {
		drawFn(f)
		if ev != nil && ev.Type == EventKey {
			if ev.Key.Type == KeyEsc || ev.Key.Ch == 'q' || ev.Key.Ch == 'Q' {
				return false
			}
		}
		return true
	})
}
