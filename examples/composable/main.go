package main

import (
	"fmt"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/widgets"
)

func main() {
	listState := limoni.NewListState()
	inputState := limoni.NewTextInputState()
	inputState.SetValue("Composable Lego Component Architecture ✨")

	list := limoni.NewList(
		"🧱 Unified Component Interface",
		"🎨 Pad, Border, Align Wrappers",
		"📐 VStack & HStack Primitives",
		"⚡ Zero-Alloc Stack Solver",
		"🔄 Backward Compatibility",
	).
		WithState(listState).
		WithHighlightSymbol("👉 ").
		WithSelectedStyle(limoni.Fg(limoni.Hex("#000000")).WithBg(limoni.Hex("#00FFAA")).Bold())

	table := limoni.NewTable().
		WithHeaders("METRIC", "STATUS", "LATENCY", "ALLOCS").
		WithRow("VStack Draw", "OPTIMAL", "644 ns", "0 B/op (0 allocs)").
		WithRow("Border Decorator", "OPTIMAL", "4.7 µs", "0 B/op (0 allocs)").
		WithRow("Nested Composite", "OPTIMAL", "14.7 µs", "0 B/op (0 allocs)").
		WithRow("Contiguous Buffer", "ACTIVE", "18.3 µs", "0 B/op (0 allocs)").
		WithSelectedStyle(limoni.Fg(limoni.ColorWhite).WithBg(limoni.Hex("#224466")))

	input := limoni.NewTextInput("demo_input").
		WithState(inputState).
		WithStyle(limoni.Fg(limoni.Hex("#FFFFFF"))).
		WithFocusedStyle(limoni.Fg(limoni.Hex("#FFFF00")).Bold())

	err := limoni.Run(func(f *limoni.Frame, ev *limoni.Event) bool {
		if ev != nil && ev.Type == limoni.EventKey {
			switch ev.Key.Type {
			case limoni.KeyEsc:
				return false
			case limoni.KeyUp:
				listState.Previous()
			case limoni.KeyDown:
				listState.Next()
			default:
				inputState.HandleKey(ev.Key)
			}
		}

		// -------------------------------------------------------------
		// Composable Lego Architecture:
		// Entire view is composed declaratively without manual Split math!
		// -------------------------------------------------------------
		view := limoni.VStack(
			// Top Header (Fixed Height 3): Border around Padded Label
			limoni.FixedSize(0, 3, limoni.Border(
				limoni.Center(limoni.Label("🍋 LIMONI COMPOSABLE LEGO ARCHITECTURE 🚀", limoni.Bold().WithFg(limoni.Hex("#00FFAA")))),
				widgets.SymbolsRounded,
				limoni.Fg(limoni.Hex("#00FFAA")),
			)),

			// Main Body: HStack with 1:2 flex ratio
			limoni.Flex(1, limoni.HStack(
				// Left Column: List wrapped in a single border
				limoni.Flex(1, limoni.Border(
					limoni.AsComponent(list),
					widgets.SymbolsSingle,
					limoni.Fg(limoni.Hex("#FFCC00")),
				)),

				// Right Column: VStack with Table (Flex 1) and TextInput (Fixed 3)
				limoni.Flex(2, limoni.VStack(
					limoni.Flex(1, limoni.Border(
						limoni.AsComponent(table),
						widgets.SymbolsDouble,
						limoni.Fg(limoni.Hex("#3399FF")),
					)),
					limoni.FixedSize(0, 3, limoni.Border(
						limoni.AsComponent(input),
						widgets.SymbolsRounded,
						limoni.Fg(limoni.Hex("#FF5599")),
					)),
				)),
			)),

			// Footer (Fixed Height 3): Centered instructions inside border
			limoni.FixedSize(0, 3, limoni.Border(
				limoni.Center(limoni.Label("ESC: Exit | ↑/↓: Navigate List | Type: Edit Input | 100% Zero Heap Allocations", limoni.Fg(limoni.Hex("#888888")))),
				widgets.SymbolsSingle,
				limoni.Fg(limoni.Hex("#666666")),
			)),
		)

		f.RenderComponent(view, f.Area())
		return true
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
