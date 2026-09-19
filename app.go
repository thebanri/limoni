package limoni

import (
	"errors"
	"time"
)

var wakeupChan = make(chan struct{}, 1)

// Wakeup signals the render loop to re-render a frame immediately without waiting for terminal input.
// Safe to call concurrently from any goroutine (tickers, background workers, etc.).
func Wakeup() {
	select {
	case wakeupChan <- struct{}{}:
	default:
		// Sinyal kanalda bekliyorsa fazladan yığılma yapmaması için atla
	}
}

// AppOption configures the application lifecycle in Run.
type AppOption func(*appConfig)

type appConfig struct {
	catchCtrlC       bool
	fps              int
	automationPath   string
	automationPolicy AutomationPolicy
	inlineHeight     uint16
	title            string
	hasTitle         bool
}

// AutomationPolicy decides what an application's automation socket lets out.
// Every field defaults to closed: with the zero value a client can see roles,
// labels, positions and bounds, and nothing else.
//
// It mirrors automation.Policy field for field. It is declared here, in the
// root package, so that a build without the limoni_debug tag does not import
// the automation package at all.
type AutomationPolicy struct {
	// ExposeInputValues sends the values of text fields. Fields marked Secret
	// are never sent, whatever this says.
	ExposeInputValues bool
	// ExposeScreen allows the snapshot of the rendered grid.
	ExposeScreen bool
	// AllowInput permits key, text and click synthesis.
	AllowInput bool
	// AllowUnverifiedPeers accepts connections on platforms that cannot report
	// the connecting user. Without it, those platforms refuse every connection.
	AllowUnverifiedPeers bool
}

// WithTitle sets the terminal window title (OSC 2) when Run starts.
// Control characters in the title are stripped; see Terminal.SetTitle.
func WithTitle(title string) AppOption {
	return func(c *appConfig) {
		c.title = title
		c.hasTitle = true
	}
}

// WithInline renders the application in place, in a band of the given height,
// instead of taking over the screen.
//
// This is the mode `gum`, CI progress renderers and shell prompts use: the
// scrollback above stays intact, and what the application drew is still on
// screen after it exits. Full-screen mode is the default.
func WithInline(height uint16) AppOption {
	return func(c *appConfig) {
		c.inlineHeight = height
	}
}

// WithAutomation serves the application's semantic tree on a Unix socket, so
// tests and agents can drive it by selector instead of by screen coordinate.
// See the automation package for the protocol and its security caveats.
//
// It is off unless you call this. The socket accepts commands that synthesise
// input into the running application, so treat enabling it the way you would
// treat enabling a debug console: a development and CI facility, not something
// to ship on by default.
func WithAutomation(socketPath string, policy AutomationPolicy) AppOption {
	return func(c *appConfig) {
		c.automationPath = socketPath
		c.automationPolicy = policy
	}
}

// WithFPS configures a continuous animation frame rate (e.g. 60, 120, 240 FPS).
// When configured, the render loop continuously invokes the draw function at the target rate.
func WithFPS(fps int) AppOption {
	return func(c *appConfig) {
		if fps > 0 {
			c.fps = fps
		}
	}
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

	term, err := NewInline(cfg.inlineHeight)
	if err != nil {
		return err
	}
	defer term.Close()
	if cfg.hasTitle {
		term.SetTitle(cfg.title)
	}
	return runLoop(term, appFn, cfg)
}

// ErrAutomationNotCompiled is returned by Run when WithAutomation is used in a
// binary built without the limoni_debug tag.
//
// This is the point of the tag. A release binary does not merely have
// automation switched off; the socket server, the policy and the input
// injector are not in it, so no configuration mistake can switch them on.
var ErrAutomationNotCompiled = errors.New("limoni: automation is not compiled into this binary; build with -tags limoni_debug")

// gateway carries application state out of the process and synthetic input
// back in. See app_gateway_debug.go; release builds have no implementation.
type gateway interface {
	publish(f *Frame)
	events() <-chan Event
	close() error
}

// runLoop is Run's body with the terminal supplied, so the loop — including
// the automation wiring — can be exercised against a headless terminal instead
// of only against a tty.
func runLoop(term *Terminal, appFn func(f *Frame, ev *Event) bool, cfg appConfig) error {
	var err error
	term.StartEventLoop()

	// The gateway is everything that carries application state out of the
	// process — the automation socket today. Its implementation only exists in
	// builds tagged limoni_debug; a release build gets a stub that refuses to
	// start one, so the code is not merely disabled but absent from the binary.
	gw, err := openGateway(cfg)
	if err != nil {
		return err
	}
	var injected <-chan Event
	publish := func(*Frame) {}
	if gw != nil {
		defer gw.close()
		injected = gw.events()
		publish = gw.publish
	}

	running := true
	// Initial render
	err = term.Draw(func(f *Frame) {
		running = appFn(f, nil)
		publish(f)
	})
	if err != nil || !running {
		return err
	}

	var tickerChan <-chan time.Time
	if cfg.fps > 0 {
		ticker := time.NewTicker(time.Second / time.Duration(cfg.fps))
		defer ticker.Stop()
		tickerChan = ticker.C
	}

	// handle draws one frame for an event, or for nil on a timer or wakeup.
	handle := func(ev *Event) error {
		if ev != nil && ev.Type == EventMouse {
			// Route mouse event through terminal hit-test router
			term.RouteMouseEvent(ev.Mouse)
		}
		return term.Draw(func(f *Frame) {
			running = appFn(f, ev)
			publish(f)
		})
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
			if err := handle(&ev); err != nil {
				return err
			}

		case ev := <-injected:
			// Synthetic input from the automation socket. Ctrl+C is not given
			// the quit shortcut here: a remote client should not be able to
			// terminate the application by accident.
			if err := handle(&ev); err != nil {
				return err
			}

		case <-wakeupChan:
			// Arka plandaki goroutine'den Wakeup() çağrıldığında tetiklenir
			if err := handle(nil); err != nil {
				return err
			}

		case <-tickerChan:
			// WithFPS ayarlandığında hedef kare hızında tetiklenir
			if err := handle(nil); err != nil {
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
