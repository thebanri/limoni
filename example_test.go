package limoni_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/testkit"
)

// printScreen prints a testkit snapshot with each row's trailing blanks cut,
// so the examples' expected output survives editors that strip whitespace.
func printScreen(term *testkit.Terminal) {
	for _, row := range strings.Split(term.Snapshot(), "\n") {
		fmt.Println(strings.TrimRight(row, " "))
	}
}

// The smallest immediate-mode application: draw on every event, return false
// to quit. Run takes over the terminal until then.
func Example() {
	limoni.Run(func(f *limoni.Frame, ev *limoni.Event) bool {
		if ev != nil && ev.Type == limoni.EventKey && ev.Key.Type == limoni.KeyEsc {
			return false
		}
		f.RenderComponent(limoni.Center(limoni.Label("Hello from Limoni (Esc quits)")), f.Area())
		return true
	})
}

// counter is a declarative (Elm architecture) model: Update changes state
// and asks for a redraw, View draws it.
type counter struct{ n int }

func (c *counter) Init() []limoni.Cmd { return nil }

func (c *counter) Update(msg limoni.Msg) limoni.UpdateResult {
	key, ok := msg.(limoni.KeyPressMsg)
	if !ok {
		return limoni.UpdateResult{}
	}
	switch {
	case key.Key.Type == limoni.KeyEsc:
		return limoni.UpdateResult{Quit: true}
	case key.Key.Type == limoni.KeyRune && key.Key.Ch == '+':
		c.n++
		return limoni.UpdateResult{Redraw: true}
	}
	return limoni.UpdateResult{}
}

func (c *counter) View(f *limoni.Frame) {
	f.RenderComponent(limoni.Label(fmt.Sprintf("count: %d  (+ adds, Esc quits)", c.n)), f.Area())
}

// RunProgram drives a Model on the real terminal until Update returns Quit
// or the context is cancelled.
func ExampleRunProgram() {
	if err := limoni.RunProgram(context.Background(), &counter{}); err != nil {
		fmt.Println(err)
	}
}

// Components compose without allocating: a border around a centred label.
// testkit draws into memory, which is also how to test a view.
func ExampleBorder() {
	term := testkit.NewTerminal(24, 5)
	term.Draw(func(f *limoni.Frame) {
		f.RenderComponent(limoni.Border(
			limoni.Center(limoni.Label("Hello, Limoni")),
			limoni.SymbolsRounded,
			limoni.NewStyle(),
		), f.Area())
	})
	printScreen(term)
	// Output:
	// ╭──────────────────────╮
	// │                      │
	// │    Hello, Limoni     │
	// │                      │
	// ╰──────────────────────╯
}

// HStack lays children out side by side; Flex shares the space left over by
// weight.
func ExampleHStack() {
	term := testkit.NewTerminal(30, 3)
	term.Draw(func(f *limoni.Frame) {
		f.RenderComponent(limoni.HStack(
			limoni.Flex(1, limoni.Border(limoni.Label("left"), limoni.SymbolsSingle, limoni.NewStyle())),
			limoni.Flex(2, limoni.Border(limoni.Label("right: weight 2"), limoni.SymbolsSingle, limoni.NewStyle())),
		), f.Area())
	})
	printScreen(term)
	// Output:
	// ┌────────┐┌──────────────────┐
	// │left    ││right: weight 2   │
	// └────────┘└──────────────────┘
}

// A table with a header row. Rows can also come from a TableDataSource, which
// is how a million-row table touches only the rows on screen.
func ExampleNewTable() {
	table := limoni.NewTable().
		WithHeaders("Service", "Status").
		WithRow("api", "up").
		WithRow("worker", "degraded")
	term := testkit.NewTerminal(26, 6)
	term.Render(table, term.Area())
	printScreen(term)
	// Output:
	// Service     │Status
	// ────────────┼────────────
	// api         │up
	// worker      │degraded
}

// MessageFromEvent is the conversion a Program applies to terminal input. A
// pointer moving with no button held is motion, not a press.
func ExampleMessageFromEvent() {
	hover := limoni.Event{Type: limoni.EventMouse, Mouse: limoni.MouseEvent{X: 10, Y: 2, Button: limoni.MouseNone}}
	click := limoni.Event{Type: limoni.EventMouse, Mouse: limoni.MouseEvent{X: 10, Y: 2, Button: limoni.MouseLeft}}
	wheel := limoni.Event{Type: limoni.EventMouse, Mouse: limoni.MouseEvent{X: 4, Y: 7, Button: limoni.MouseScrollDown}}
	for _, ev := range []limoni.Event{hover, click, wheel} {
		switch msg := limoni.MessageFromEvent(ev).(type) {
		case limoni.MouseMotionMsg:
			fmt.Println("motion at", msg.Position.X, msg.Position.Y)
		case limoni.MousePressMsg:
			fmt.Println("press at", msg.Position.X, msg.Position.Y)
		case limoni.MouseWheelMsg:
			fmt.Println("wheel", msg.DeltaY, "at", msg.Position.X, msg.Position.Y)
		}
	}
	// Output:
	// motion at 10 2
	// press at 10 2
	// wheel -1 at 4 7
}
