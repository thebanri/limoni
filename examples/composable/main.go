package main

import (
	"fmt"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/widgets"
)

func main() {
	listState := limoni.NewListState()
	listState.Select(0)

	inputState := limoni.NewTextInputState()
	inputState.SetValue("Limoni Lego UI")

	menuItems := []string{
		"[01] Unified Component",
		"[02] Pad, Border, Align",
		"[03] VStack & HStack (Flex)",
		"[04] ZStack (Depth-Axis)",
		"[05] Overlay (Compositor)",
		"[06] Transform (Live Text)",
		"[07] Border Presets",
		"[08] Conditionals (When/Match)",
		"[09] Style Cascading",
		"[10] Interactive & Events",
		"[11] Zero-Alloc Benchmarks",
	}

	list := limoni.NewList(menuItems...).
		WithState(listState).
		WithHighlightSymbol("> ").
		WithSelectedStyle(limoni.Fg(limoni.Hex("#000000")).WithBg(limoni.Hex("#00FFAA")).Bold())

	// Performance benchmark table (for item 10)
	table := limoni.NewTable().
		WithHeaders("PRIMITIVE", "STATUS", "LATENCY", "ALLOCS").
		WithRow("VStack / HStack Draw", "PASS", "660 ns", "0 B/op (0 allocs)").
		WithRow("ZStack Layering", "PASS", "431 ns", "0 B/op (0 allocs)").
		WithRow("Overlay Compositor", "PASS", "368 ns", "0 B/op (0 allocs)").
		WithRow("Transform (Uppercase)", "PASS", "1061 ns", "0 B/op (0 allocs)").
		WithRow("Transform (Mask)", "PASS", "623 ns", "0 B/op (0 allocs)").
		WithRow("Spacer Draw", "PASS", "131 ns", "0 B/op (0 allocs)").
		WithRow("Divider Draw", "PASS", "1666 ns", "0 B/op (0 allocs)").
		WithRow("Margin Draw", "PASS", "215 ns", "0 B/op (0 allocs)").
		WithRow("BorderEdges Draw", "PASS", "2300 ns", "0 B/op (0 allocs)").
		WithRow("Constrain Draw", "PASS", "566 ns", "0 B/op (0 allocs)").
		WithRow("Flexbox Justify/Align", "PASS", "771 ns", "0 B/op (0 allocs)").
		WithRow("Style Cascading", "PASS", "435 ns", "0 B/op (0 allocs)").
		WithSelectedStyle(limoni.Fg(limoni.ColorWhite).WithBg(limoni.Hex("#224466"))).
		WithConstraints(
			widgets.TableConstraint{Type: widgets.ConstraintPercentage, Value: 35},
			widgets.TableConstraint{Type: widgets.ConstraintPercentage, Value: 15},
			widgets.TableConstraint{Type: widgets.ConstraintPercentage, Value: 20},
			widgets.TableConstraint{Type: widgets.ConstraintPercentage, Value: 30},
		)

	input := limoni.NewTextInput("demo_input").
		WithState(inputState).
		WithStyle(limoni.Fg(limoni.Hex("#FFFFFF"))).
		WithFocusedStyle(limoni.Fg(limoni.Hex("#FFFF00")).Bold())

	showModal := false
	clickCount := 0
	modes := []string{"MONITORING", "DEBUG", "PRODUCTION"}
	modeIdx := 0

	err := limoni.Run(func(f *limoni.Frame, ev *limoni.Event) bool {
		// Keyboard input routing
		if ev != nil && ev.Type == limoni.EventKey {
			switch ev.Key.Type {
			case limoni.KeyEsc:
				if showModal {
					showModal = false
				} else {
					return false
				}
			case limoni.KeyRune:
				switch ev.Key.Ch {
				case '?', 'h', 'H':
					showModal = !showModal
				case 'm', 'M':
					modeIdx = (modeIdx + 1) % len(modes)
				case 'c', 'C':
					clickCount++
				default:
					inputState.HandleKey(ev.Key)
				}
			case limoni.KeyUp:
				if listState.Selected > 0 {
					listState.Previous()
				}
			case limoni.KeyDown:
				if listState.Selected < len(menuItems)-1 {
					listState.Next()
				}
			default:
				inputState.HandleKey(ev.Key)
			}
		}

		currentMode := modes[modeIdx]

		// Mode badge via Match
		modeBadge := limoni.Match(currentMode, map[string]limoni.Component{
			"MONITORING": limoni.WithForeground(limoni.Hex("#00FFAA"), limoni.Label("[MONITORING]")),
			"DEBUG":      limoni.WithForeground(limoni.Hex("#FFCC00"), limoni.Label("[DEBUG]")),
			"PRODUCTION": limoni.WithForeground(limoni.Hex("#FF5555"), limoni.Label("[PRODUCTION]")),
		}, limoni.Label("[UNKNOWN]"))

		// Clickable button
		clickBtn := limoni.OnClick(
			limoni.Border(
				limoni.Pad(
					limoni.Label(fmt.Sprintf("Clicks: %d", clickCount), limoni.Bold().WithFg(limoni.Hex("#FF77AA"))),
					0, 1, 0, 1,
				),
				widgets.SymbolsRounded,
				limoni.Fg(limoni.Hex("#FF77AA")),
			),
			func(m limoni.MouseEvent) {
				clickCount++
			},
		)

		// -----------------------------------------------------------------
		// Left Column: Menu List + Overlay Badge
		// -----------------------------------------------------------------
		listColumn := limoni.Overlay(
			limoni.Border(
				limoni.Pad(limoni.AsComponent(list), 1, 0, 0, 0),
				widgets.SymbolsRounded,
				limoni.Fg(limoni.Hex("#5588EE")),
			),
			limoni.FixedSize(13, 1,
				limoni.Label(" [MODULES] ", limoni.Bold().WithFg(limoni.Hex("#000000")).WithBg(limoni.Hex("#5588EE"))),
			),
			3, 0,
		)

		// -----------------------------------------------------------------
		// Right Column: Dynamic Detail View by Selected Menu Item
		// -----------------------------------------------------------------
		selectedIndex := listState.Selected
		if selectedIndex < 0 {
			selectedIndex = 0
		}

		var detailContent limoni.Component

		switch selectedIndex {
		case 0: // Unified Component
			detailContent = limoni.VStack(
				limoni.Label("Unified Component Interface", limoni.Bold().WithFg(limoni.Hex("#00FFAA"))),
				limoni.Label("All visual elements implement the minimal Component interface:", limoni.Fg(limoni.Hex("#AAAAAA"))),
				limoni.Label("  - Draw(ctx, buf)      -> Zero-allocation draw directly into cell buffer", limoni.Fg(limoni.Hex("#CCCCCC"))),
				limoni.Label("  - LayoutInfo(maxArea) -> Returns flex weights, min/max dimensions", limoni.Fg(limoni.Hex("#CCCCCC"))),
				limoni.Label("  - SizeHint(maxArea)   -> Returns preferred width and height", limoni.Fg(limoni.Hex("#CCCCCC"))),
				limoni.Label(""),
				limoni.Label("Live Demo: Arbitrary Nesting", limoni.Bold().WithFg(limoni.Hex("#FFCC00"))),
				limoni.Border(
					limoni.Pad(
						limoni.VStack(
							limoni.Label("Outer Container: Border(Cyan) -> Pad()", limoni.Fg(limoni.Hex("#00FFFF"))),
							limoni.Border(
								limoni.Pad(
									limoni.Label("Inner Child: Border(Pink) -> Pad() -> Label", limoni.Fg(limoni.Hex("#FF77AA"))),
									0, 1, 0, 1,
								),
								widgets.SymbolsRounded,
								limoni.Fg(limoni.Hex("#FF77AA")),
							),
						),
						1, 2, 1, 2,
					),
					widgets.SymbolsRounded,
					limoni.Fg(limoni.Hex("#00FFFF")),
				),
			)

		case 1: // Pad, Border, Align
			detailContent = limoni.VStack(
				limoni.Label("Padding, Borders & Alignment Decorators", limoni.Bold().WithFg(limoni.Hex("#00FFAA"))),
				limoni.Label("Decorators wrap any component with zero heap allocation overhead:", limoni.Fg(limoni.Hex("#AAAAAA"))),
				limoni.Label(""),
				limoni.HStack(
					limoni.Flex(1, limoni.Border(
						limoni.Pad(limoni.Center(limoni.Label("Center()\nCentered text")), 1, 1, 1, 1),
						widgets.SymbolsRounded,
						limoni.Fg(limoni.Hex("#00FFAA")),
					)),
					limoni.Flex(1, limoni.Border(
						limoni.AlignComponent(limoni.Label("AlignTopLeft"), limoni.HAlignLeft, limoni.AlignTop),
						widgets.SymbolsDouble,
						limoni.Fg(limoni.Hex("#FFCC00")),
					)),
					limoni.Flex(1, limoni.Border(
						limoni.AlignComponent(limoni.Label("AlignBottomRight"), limoni.HAlignRight, limoni.AlignBottom),
						widgets.SymbolsThick,
						limoni.Fg(limoni.Hex("#FF77AA")),
					)),
				),
			)

		case 2: // VStack & HStack (Flex)
			detailContent = limoni.VStack(
				limoni.Label("Flexbox Stacks (VStack & HStack)", limoni.Bold().WithFg(limoni.Hex("#00FFAA"))),
				limoni.Label("Supports flex ratios, fixed sizing, gap spacing, and alignment.", limoni.Fg(limoni.Hex("#AAAAAA"))),
				limoni.Label(""),
				limoni.Label("Proportional Sizing (Flex 1 : Flex 2 : Flex 1):", limoni.Fg(limoni.Hex("#FFFF00"))),
				limoni.HStack(
					limoni.Flex(1, limoni.Border(limoni.Center(limoni.Label("Flex(1) - 25%")), widgets.SymbolsRounded, limoni.Fg(limoni.Hex("#00FFAA")))),
					limoni.Flex(2, limoni.Border(limoni.Center(limoni.Label("Flex(2) - 50%")), widgets.SymbolsRounded, limoni.Fg(limoni.Hex("#3399FF")))),
					limoni.Flex(1, limoni.Border(limoni.Center(limoni.Label("Flex(1) - 25%")), widgets.SymbolsRounded, limoni.Fg(limoni.Hex("#00FFAA")))),
				),
				limoni.Label(""),
				limoni.Label("Alignment: SpaceBetween, Center, Start, End for flexible layouts.", limoni.Fg(limoni.Hex("#888888"))),
			)

		case 3: // ZStack (Depth)
			detailContent = limoni.VStack(
				limoni.Label("ZStack (Depth-Axis / Painter's Algorithm)", limoni.Bold().WithFg(limoni.Hex("#00FFAA"))),
				limoni.Label("Layers children back-to-front into the same buffer without extra allocations.", limoni.Fg(limoni.Hex("#AAAAAA"))),
				limoni.Label(""),
				limoni.Border(
					limoni.Pad(
						limoni.ZStack(
							limoni.Label(". . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . .\n. Background Layer (Z-Index 0) . . . . . . . . . . . . . . . . . . . . . . . .\n. . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . .\n. . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . .\n. . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . .", limoni.Fg(limoni.Hex("#444444"))),
							limoni.Center(
								limoni.Border(
									limoni.Pad(limoni.Label("Foreground Card (Z-Index 1)", limoni.Bold().WithFg(limoni.Hex("#FFFFFF"))), 0, 2, 0, 2),
									widgets.SymbolsDouble,
									limoni.Fg(limoni.Hex("#FFCC00")),
								),
							),
						),
						1, 1, 1, 1,
					),
					widgets.SymbolsRounded,
					limoni.Fg(limoni.Hex("#666666")),
				),
				limoni.Label("Tip: Press [?] or [h] to open the full-screen modal overlay.", limoni.Fg(limoni.Hex("#FFD700"))),
			)

		case 4: // Overlay
			detailContent = limoni.VStack(
				limoni.Label("Overlay (Absolute Offset Compositor)", limoni.Bold().WithFg(limoni.Hex("#00FFAA"))),
				limoni.Label("Positions an overlay on top of a base at exact (X, Y) coordinates.", limoni.Fg(limoni.Hex("#AAAAAA"))),
				limoni.Label("The '[MODULES]' badge on the left menu is placed using Overlay.", limoni.Fg(limoni.Hex("#AAAAAA"))),
				limoni.Label(""),
				limoni.Overlay(
					limoni.Border(
						limoni.Pad(
							limoni.VStack(
								limoni.Label("Base Card Surface", limoni.Bold().WithFg(limoni.Hex("#FFFFFF"))),
								limoni.Label("An overlay can be pinned to any corner or coordinate."),
								limoni.Label("Mouse events are automatically translated to local space."),
							),
							1, 2, 1, 2,
						),
						widgets.SymbolsRounded,
						limoni.Fg(limoni.Hex("#3399FF")),
					),
					limoni.FixedSize(14, 1,
						limoni.Label(" [NOTIFICATION] ", limoni.Bold().WithFg(limoni.Hex("#000000")).WithBg(limoni.Hex("#FF5599"))),
					),
					3, 0,
				),
			)

		case 5: // Transform
			inputValue := inputState.Value()
			if inputValue == "" {
				inputValue = "Hello World"
			}
			detailContent = limoni.VStack(
				limoni.Label("Transform (In-Place Cell Pipeline)", limoni.Bold().WithFg(limoni.Hex("#00FFAA"))),
				limoni.Label("Post-processes rendered buffer cells directly without string allocations.", limoni.Fg(limoni.Hex("#AAAAAA"))),
				limoni.Label("Type in the bottom input bar to see real-time transformations:", limoni.Fg(limoni.Hex("#FFFF00"))),
				limoni.Label(""),
				limoni.HStack(
					limoni.Flex(1, limoni.Border(
						limoni.VStack(
							limoni.Label("Original:", limoni.Fg(limoni.Hex("#888888"))),
							limoni.Label(inputValue, limoni.Fg(limoni.Hex("#FFFFFF"))),
						),
						widgets.SymbolsRounded,
						limoni.Fg(limoni.Hex("#666666")),
					)),
					limoni.Flex(1, limoni.Border(
						limoni.VStack(
							limoni.Label("Uppercase(text):", limoni.Fg(limoni.Hex("#888888"))),
							limoni.Uppercase(limoni.Label(inputValue, limoni.Bold().WithFg(limoni.Hex("#00FFAA")))),
						),
						widgets.SymbolsRounded,
						limoni.Fg(limoni.Hex("#00FFAA")),
					)),
				),
				limoni.HStack(
					limoni.Flex(1, limoni.Border(
						limoni.VStack(
							limoni.Label("Lowercase(text):", limoni.Fg(limoni.Hex("#888888"))),
							limoni.Lowercase(limoni.Label(inputValue, limoni.Fg(limoni.Hex("#3399FF")))),
						),
						widgets.SymbolsRounded,
						limoni.Fg(limoni.Hex("#3399FF")),
					)),
					limoni.Flex(1, limoni.Border(
						limoni.VStack(
							limoni.Label("Mask(text, '*'):", limoni.Fg(limoni.Hex("#888888"))),
							limoni.Mask(limoni.Label(inputValue, limoni.Fg(limoni.Hex("#FF77AA"))), '*'),
						),
						widgets.SymbolsRounded,
						limoni.Fg(limoni.Hex("#FF77AA")),
					)),
				),
			)

		case 6: // Border Presets Explained
			detailContent = limoni.VStack(
				limoni.Label("Border Styles Comparison (Standard vs Half-Block)", limoni.Bold().WithFg(limoni.Hex("#00FFAA"))),
				limoni.Label("Half-Block borders use Unicode block elements (top/bottom/side half blocks).", limoni.Fg(limoni.Hex("#AAAAAA"))),
				limoni.Label("Some fonts render them chunky or with subpixel gaps. Standard borders use box-drawing glyphs.", limoni.Fg(limoni.Hex("#AAAAAA"))),
				limoni.Label(""),
				limoni.HStack(
					limoni.Flex(1, limoni.Border(
						limoni.Center(limoni.Label("Rounded")),
						widgets.SymbolsRounded,
						limoni.Fg(limoni.Hex("#00FFAA")),
					)),
					limoni.Flex(1, limoni.Border(
						limoni.Center(limoni.Label("Double")),
						widgets.SymbolsDouble,
						limoni.Fg(limoni.Hex("#3399FF")),
					)),
					limoni.Flex(1, limoni.Border(
						limoni.Center(limoni.Label("Thick")),
						widgets.SymbolsThick,
						limoni.Fg(limoni.Hex("#FFCC00")),
					)),
				),
				limoni.HStack(
					limoni.Flex(1, limoni.Border(
						limoni.Center(limoni.Label("Single")),
						widgets.SymbolsSingle,
						limoni.Fg(limoni.Hex("#AAAAAA")),
					)),
					limoni.Flex(1, limoni.Border(
						limoni.Center(limoni.Label("OuterHalfBlock")),
						widgets.SymbolsOuterHalfBlock,
						limoni.Fg(limoni.Hex("#FF77AA")),
					)),
					limoni.Flex(1, limoni.Border(
						limoni.Center(limoni.Label("InnerHalfBlock")),
						widgets.SymbolsInnerHalfBlock,
						limoni.Fg(limoni.Hex("#FFAA88")),
					)),
				),
			)

		case 7: // Conditionals (When/Match)
			detailContent = limoni.VStack(
				limoni.Label("Conditional Rendering (When & Match)", limoni.Bold().WithFg(limoni.Hex("#00FFAA"))),
				limoni.Label("Unmatched branches are skipped with zero cost (no DOM/allocation overhead).", limoni.Fg(limoni.Hex("#AAAAAA"))),
				limoni.Label(""),
				limoni.Label("Press [m] to cycle application mode:", limoni.Fg(limoni.Hex("#FFFF00"))),
				limoni.Border(
					limoni.Pad(
						limoni.VStack(
							limoni.HStack(
								limoni.Label("Current Active Mode: "),
								modeBadge,
							),
							limoni.When(currentMode == "MONITORING",
								limoni.Label("Status: Collecting real-time engine telemetry...", limoni.Fg(limoni.Hex("#00FFAA"))),
							),
							limoni.When(currentMode == "DEBUG",
								limoni.Label("Status: Verbose debug logging active (allocation tracking on).", limoni.Fg(limoni.Hex("#FFCC00"))),
							),
							limoni.When(currentMode == "PRODUCTION",
								limoni.Label("Status: Production mode, maximum throughput enabled.", limoni.Fg(limoni.Hex("#FF5555"))),
							),
						),
						1, 2, 1, 2,
					),
					widgets.SymbolsRounded,
					limoni.Fg(limoni.Hex("#FFFFFF")),
				),
			)

		case 8: // Style Cascading
			detailContent = limoni.VStack(
				limoni.Label("Style Cascading & Inheritance", limoni.Bold().WithFg(limoni.Hex("#00FFAA"))),
				limoni.Label("Styles flow down the hierarchy via cell.Context. Children only override what they set.", limoni.Fg(limoni.Hex("#AAAAAA"))),
				limoni.Label(""),
				limoni.WithBackground(limoni.Hex("#1A202C"),
					limoni.Border(
						limoni.Pad(
							limoni.WithForeground(limoni.Hex("#E2E8F0"),
								limoni.VStack(
									limoni.Label("Parent Container: Dark background + light gray text"),
									limoni.WithForeground(limoni.Hex("#48BB78"), limoni.Label("  -> Line 1: Green foreground (inherits dark background)")),
									limoni.WithForeground(limoni.Hex("#F6E05E"), limoni.Label("  -> Line 2: Yellow foreground")),
									limoni.Label("  -> Line 3: Inherits parent foreground and background"),
								),
							),
							1, 2, 1, 2,
						),
						widgets.SymbolsRounded,
						limoni.Fg(limoni.Hex("#4A5568")),
					),
				),
			)

		case 9: // Interactive & Events
			detailContent = limoni.VStack(
				limoni.Label("Interactive Components & Event Routing", limoni.Bold().WithFg(limoni.Hex("#00FFAA"))),
				limoni.Label("Mouse and key events are routed directly to components with local coordinates.", limoni.Fg(limoni.Hex("#AAAAAA"))),
				limoni.Label(""),
				limoni.Label(fmt.Sprintf("Total Clicks: %d  (Click button below or press [c])", clickCount), limoni.Fg(limoni.Hex("#FFFF00"))),
				limoni.Label(""),
				limoni.HStack(
					clickBtn,
					limoni.OnClick(
						limoni.Border(
							limoni.Pad(limoni.Label("[ Reset Counter ]", limoni.Bold().WithFg(limoni.Hex("#FF5555"))), 0, 1, 0, 1),
							widgets.SymbolsRounded,
							limoni.Fg(limoni.Hex("#FF5555")),
						),
						func(m limoni.MouseEvent) {
							clickCount = 0
						},
					),
				).WithGap(2),
			)

		case 10: // Zero-Alloc Performance
			detailContent = limoni.VStack(
				limoni.Label("Zero-Allocation Performance Benchmarks", limoni.Bold().WithFg(limoni.Hex("#00FFAA"))),
				limoni.Label("All 14 component primitives execute at 0 B/op (zero heap allocations on hot path):", limoni.Fg(limoni.Hex("#AAAAAA"))),
				limoni.Label(""),
				limoni.Flex(1, limoni.Border(
					limoni.AsComponent(table),
					widgets.SymbolsRounded,
					limoni.Fg(limoni.Hex("#00FFAA")),
				)),
			)
		}

		// -------------------------------------------------------------
		// Base Dashboard View (VStack + HStack):
		// -------------------------------------------------------------
		baseDashboard := limoni.VStack(
			// Top Header (Fixed Height 3): Justified title and status badge
			limoni.FixedSize(0, 3, limoni.Border(
				limoni.Pad(
					limoni.HStack(
						limoni.Label("LIMONI COMPOSABLE ARCHITECTURE", limoni.Bold().WithFg(limoni.Hex("#00FFAA"))),
						limoni.Uppercase(limoni.Label("v1.0 - 0 B/op", limoni.Fg(limoni.Hex("#88CCFF")))),
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
				// Left Column: List wrapped in a rounded border + Overlay badge
				limoni.Flex(1, listColumn),

				// Right Column: Dynamic Detail View
				limoni.Flex(2, limoni.Border(
					limoni.Pad(detailContent, 1, 2, 1, 2),
					widgets.SymbolsRounded,
					limoni.Fg(limoni.Hex("#3399FF")),
				)),
			).WithGap(1)),

			// Bottom Input Bar (Fixed Height 3)
			limoni.FixedSize(0, 3, limoni.HStack(
				limoni.Flex(3, limoni.Border(
					limoni.Pad(limoni.AsComponent(input), 0, 1, 0, 1),
					widgets.SymbolsRounded,
					limoni.Fg(limoni.Hex("#FF5599")),
				)),
				limoni.Flex(1, limoni.Border(
					limoni.Pad(
						limoni.Mask(limoni.Label(inputState.Value(), limoni.Fg(limoni.Hex("#FFAA88"))), '*'),
						0, 1, 0, 1,
					),
					widgets.SymbolsRounded,
					limoni.Fg(limoni.Hex("#FFAA88")),
				)),
			).WithGap(1)),

			// Footer: SpaceBetween distribution with shortcuts
			limoni.FixedSize(0, 3, limoni.Border(
				limoni.Pad(
					limoni.HStack(
						limoni.WithForeground(limoni.Hex("#888888"), limoni.Label("Up/Down: Select | c: Click | m: Toggle Mode | ?: Modal | ESC: Exit")),
						limoni.WithForeground(limoni.Hex("#00FFAA"), limoni.Label("0 B/op Zero Alloc")),
					).WithJustify(limoni.JustifySpaceBetween).WithAlignItems(limoni.AlignItemsCenter),
					0, 1, 0, 1,
				),
				widgets.SymbolsSingle,
				limoni.Fg(limoni.Hex("#555555")),
			)),
		)

		// -------------------------------------------------------------
		// Modal Overlay via ZStack + When (Painter's Algorithm):
		// -------------------------------------------------------------
		modalLayer := limoni.When(showModal,
			limoni.Center(
				limoni.FixedSize(62, 14,
					limoni.WithBackground(limoni.Hex("#111827"),
						limoni.Border(
							limoni.Pad(
								limoni.VStack(
									limoni.Center(limoni.Label("== COMPOSABLE ARCHITECTURE ==", limoni.Bold().WithFg(limoni.Hex("#00FFAA")))),
									limoni.Label("- ZStack: Depth-axis layering with zero offscreen buffers", limoni.Fg(limoni.Hex("#FFFFFF"))),
									limoni.Label("- Overlay: Absolute offset child positioning and event routing", limoni.Fg(limoni.Hex("#E5E7EB"))),
									limoni.Label("- Transform: In-place buffer cell manipulation pipeline", limoni.Fg(limoni.Hex("#D1D5DB"))),
									limoni.Label("- Flexbox: JustifyContent & AlignItems on HStack/VStack", limoni.Fg(limoni.Hex("#9CA3AF"))),
									limoni.Label("- Conditionals: Declarative When & Match branching", limoni.Fg(limoni.Hex("#6EE7B7"))),
									limoni.Label("- Style Cascading: Context-based inheritance (WithStyle, WithFg)", limoni.Fg(limoni.Hex("#93C5FD"))),
									limoni.Center(limoni.Label("[ Press Esc or ? to Close ]", limoni.Fg(limoni.Hex("#F59E0B")).Bold())),
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

		// ZStack: Combines base dashboard and modal layer
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
