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
		"📐 VStack & HStack (Flexbox)",
		"🥞 ZStack (Depth / Painter's)",
		"🎯 Overlay (Absolute Compositor)",
		"⚡ Transform (Uppercase / Mask)",
		"🔲 Half-Block Border Presets",
		"❓ Conditionals (When, Match)",
		"🎭 Style Cascading (WithStyle)",
		"🖱️ Interactive & Local Hit-Testing",
		"⚡ Zero-Alloc Hot Path (0 B/op)",
	).
		WithState(listState).
		WithHighlightSymbol("👉 ").
		WithSelectedStyle(limoni.Fg(limoni.Hex("#000000")).WithBg(limoni.Hex("#00FFAA")).Bold())

	table := limoni.NewTable().
		WithHeaders("METRIC", "STATUS", "LATENCY", "ALLOCS").
		WithRow("VStack / HStack Draw", "OPTIMAL", "650 ns", "0 B/op (0 allocs)").
		WithRow("ZStack Layering", "OPTIMAL", "437 ns", "0 B/op (0 allocs)").
		WithRow("Overlay Compositor", "OPTIMAL", "368 ns", "0 B/op (0 allocs)").
		WithRow("Transform Pipeline", "OPTIMAL", "623 ns", "0 B/op (0 allocs)").
		WithRow("Flexbox Justify/Align", "OPTIMAL", "767 ns", "0 B/op (0 allocs)").
		WithRow("Style Cascading", "OPTIMAL", "428 ns", "0 B/op (0 allocs)").
		WithRow("Interactive Click Hit", "OPTIMAL", "120 ns", "0 B/op (0 allocs)").
		WithSelectedStyle(limoni.Fg(limoni.ColorWhite).WithBg(limoni.Hex("#224466")))

	input := limoni.NewTextInput("demo_input").
		WithState(inputState).
		WithStyle(limoni.Fg(limoni.Hex("#FFFFFF"))).
		WithFocusedStyle(limoni.Fg(limoni.Hex("#FFFF00")).Bold())

	showModal := false
	clickCount := 0
	currentMode := "monitoring"

	err := limoni.Run(func(f *limoni.Frame, ev *limoni.Event) bool {
		if ev != nil && ev.Type == limoni.EventKey {
			switch ev.Key.Type {
			case limoni.KeyEsc:
				if showModal {
					showModal = false
				} else {
					return false
				}
			case limoni.KeyRune:
				if ev.Key.Ch == '?' || ev.Key.Ch == 'h' {
					showModal = !showModal
				} else if ev.Key.Ch == 'm' {
					if currentMode == "monitoring" {
						currentMode = "debug"
					} else {
						currentMode = "monitoring"
					}
				} else {
					inputState.HandleKey(ev.Key)
				}
			case limoni.KeyUp:
				listState.Previous()
			case limoni.KeyDown:
				listState.Next()
			default:
				inputState.HandleKey(ev.Key)
			}
		}

		// Mode badge via Match pattern matching:
		modeBadge := limoni.Match(currentMode, map[string]limoni.Component{
			"monitoring": limoni.WithForeground(limoni.Hex("#00FFAA"), limoni.Label("🟢 MONITORING")),
			"debug":      limoni.WithForeground(limoni.Hex("#FFCC00"), limoni.Label("🟡 DEBUG TRACE")),
		}, limoni.Label("⚪ UNKNOWN"))

		// Action button with OnClick:
		clickBtn := limoni.OnClick(
			limoni.Border(
				limoni.Pad(
					limoni.Label(fmt.Sprintf("🖱️ Clicks: %d", clickCount), limoni.Bold().WithFg(limoni.Hex("#FF77AA"))),
					0, 1, 0, 1,
				),
				widgets.SymbolsRounded,
				limoni.Fg(limoni.Hex("#FF77AA")),
			),
			func(m limoni.MouseEvent) {
				clickCount++
			},
		)

		// Left column with an Overlay badge:
		listColumn := limoni.Overlay(
			limoni.Border(
				limoni.AsComponent(list),
				widgets.SymbolsSingle,
				limoni.Fg(limoni.Hex("#FFCC00")),
			),
			limoni.FixedSize(16, 1,
				limoni.Label(" 🚀 PARITY BADGE ", limoni.Bold().WithFg(limoni.Hex("#000000")).WithBg(limoni.Hex("#00FFAA"))),
			),
			4, 0,
		)

		// -------------------------------------------------------------
		// Base Dashboard View (VStack + HStack):
		// -------------------------------------------------------------
		baseDashboard := limoni.VStack(
			// Top Header (Fixed Height 3): Justified title and status badge
			limoni.FixedSize(0, 3, limoni.Border(
				limoni.Pad(
					limoni.HStack(
						limoni.Label("🍋 LIMONI LEGO ARCHITECTURE", limoni.Bold().WithFg(limoni.Hex("#00FFAA"))),
						limoni.Uppercase(limoni.Label("zero-alloc v1.0", limoni.Fg(limoni.Hex("#88CCFF")))),
						modeBadge,
						clickBtn,
					).WithJustify(limoni.JustifySpaceBetween).WithAlignItems(limoni.AlignItemsCenter),
					0, 1, 0, 1,
				),
				widgets.SymbolsRounded,
				limoni.Fg(limoni.Hex("#00FFAA")),
			)),

			// Main Body: HStack with 1:2 flex ratio
			limoni.Flex(1, limoni.HStack(
				// Left Column: List wrapped in a single border + Overlay badge
				limoni.Flex(1, listColumn),

				// Right Column: VStack with Table (OuterHalfBlock) and Inputs (Rounded + Mask)
				limoni.Flex(2, limoni.VStack(
					limoni.Flex(1, limoni.Border(
						limoni.AsComponent(table),
						widgets.SymbolsOuterHalfBlock,
						limoni.Fg(limoni.Hex("#3399FF")),
					)),
					limoni.FixedSize(0, 3, limoni.HStack(
						limoni.Flex(2, limoni.Border(
							limoni.AsComponent(input),
							widgets.SymbolsRounded,
							limoni.Fg(limoni.Hex("#FF5599")),
						)),
						limoni.Flex(1, limoni.Border(
							limoni.Pad(
								limoni.Mask(limoni.Label("secretpassword", limoni.Fg(limoni.Hex("#FFAA88"))), '*'),
								0, 1, 0, 1,
							),
							widgets.SymbolsInnerHalfBlock,
							limoni.Fg(limoni.Hex("#FFAA88")),
						)),
					)),
				)),
			)),

			// Footer: SpaceBetween distribution with cascaded styles
			limoni.FixedSize(0, 3, limoni.Border(
				limoni.Pad(
					limoni.HStack(
						limoni.WithForeground(limoni.Hex("#888888"), limoni.Label("ESC: Exit | ?: Toggle Modal | m: Toggle Mode | ↑/↓: List")),
						limoni.WithForeground(limoni.Hex("#00FFAA"), limoni.Label("0 B/op Zero Alloc")),
					).WithJustify(limoni.JustifySpaceBetween).WithAlignItems(limoni.AlignItemsCenter),
					0, 1, 0, 1,
				),
				widgets.SymbolsSingle,
				limoni.Fg(limoni.Hex("#666666")),
			)),
		)

		// -------------------------------------------------------------
		// Modal Overlay via ZStack + When (Painter's Algorithm):
		// -------------------------------------------------------------
		modalLayer := limoni.When(showModal,
			limoni.Center(
				limoni.FixedSize(54, 11,
					limoni.WithBackground(limoni.Hex("#111827"),
						limoni.Border(
							limoni.Pad(
								limoni.VStack(
									limoni.Center(limoni.Label("✨ ARCHITECTURAL EXTENSIONS ✨", limoni.Bold().WithFg(limoni.Hex("#00FFAA")))),
									limoni.Label("• ZStack: Depth-axis layering with zero offscreen buffers", limoni.Fg(limoni.Hex("#FFFFFF"))),
									limoni.Label("• Flexbox: JustifyContent & AlignItems on Stacks", limoni.Fg(limoni.Hex("#E5E7EB"))),
									limoni.Label("• Conditionals: Declarative When & Match", limoni.Fg(limoni.Hex("#D1D5DB"))),
									limoni.Label("• Style Cascading: WithStyle, WithForeground", limoni.Fg(limoni.Hex("#9CA3AF"))),
									limoni.Label("• Interactive: OnClick & local hit-testing", limoni.Fg(limoni.Hex("#6EE7B7"))),
									limoni.Center(limoni.Label("[Press Esc or ? to Close]", limoni.Fg(limoni.Hex("#F59E0B")).Bold())),
								).WithJustify(limoni.JustifySpaceAround),
								1, 2, 1, 2,
							),
							widgets.SymbolsDouble,
							limoni.Fg(limoni.Hex("#F59E0B")),
						),
					),
				),
			),
		)

		// ZStack combines base dashboard and modal layer:
		rootView := limoni.ZStack(
			baseDashboard,
			modalLayer,
		)

		f.RenderComponent(rootView, f.Area())
		return true
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
