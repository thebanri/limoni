package widgets_test

import (
	"fmt"
	"strings"

	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/testkit"
	"github.com/thebanri/limoni/widgets"
)

// printScreen prints a testkit snapshot with each row's trailing blanks cut.
func printScreen(term *testkit.Terminal) {
	for _, row := range strings.Split(term.Snapshot(), "\n") {
		fmt.Println(strings.TrimRight(row, " "))
	}
}

// A gauge shows its percentage in the middle unless given a label.
func ExampleGauge() {
	term := testkit.NewTerminal(20, 1)
	term.Render(&widgets.Gauge{Ratio: 0.42}, term.Area())
	printScreen(term)
	// Output:
	// ████████42%
}

// A list draws its items; a ListState carries the selection and the scroll
// offset between frames.
func ExampleList() {
	state := widgets.NewListState()
	state.Selected = 1
	list := widgets.NewList("apples", "lemons", "pears")
	list.State = state
	term := testkit.NewTerminal(12, 3)
	term.Render(list, term.Area())
	printScreen(term)
	// Output:
	// apples
	// lemons
	// pears
}

// Tabs draw their titles in a row; Selected picks the active one.
func ExampleTabs() {
	term := testkit.NewTerminal(30, 1)
	term.Render(&widgets.Tabs{Titles: []string{"Logs", "Metrics", "Traces"}, Selected: 1}, term.Area())
	printScreen(term)
	// Output:
	// Logs │ Metrics │ Traces
}

// TextInputState edits text the way a shell line does. Enter alone is left
// to the application (submit); Shift+Enter, which terminals with the kitty
// keyboard protocol report, inserts a newline.
func ExampleTextInputState_HandleKey() {
	state := widgets.NewTextInputState()
	for _, key := range []driver.KeyEvent{
		{Type: driver.KeyRune, Ch: 'h'},
		{Type: driver.KeyRune, Ch: 'i'},
		{Type: driver.KeyEnter, Shift: true},
		{Type: driver.KeyRune, Ch: 'x'},
	} {
		state.HandleKey(key)
	}
	submit := state.HandleKey(driver.KeyEvent{Type: driver.KeyEnter})
	fmt.Printf("%q, Enter handled by the input: %v\n", state.Value(), submit)
	// Output:
	// "hi\nx", Enter handled by the input: false
}

// A sparkline scales its data between the smallest and largest value, in
// eighth blocks: the smallest value is a blank cell.
func ExampleSparkline() {
	term := testkit.NewTerminal(9, 1)
	term.Render(&widgets.Sparkline{Data: []float64{0, 1, 2, 3, 4, 5, 6, 7, 8}}, cell.NewRect(0, 0, 9, 1))
	fmt.Printf("[%s]\n", term.Snapshot())
	// Output:
	// [ ▁▂▃▄▅▆▇█]
}
