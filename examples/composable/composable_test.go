package main

import (
	"fmt"
	"testing"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/testkit"
	"github.com/thebanri/limoni/widgets"
)

func TestComposableSnapshot(t *testing.T) {
	term := testkit.NewTerminal(84, 24)

	listState := limoni.NewListState()
	inputState := limoni.NewTextInputState()
	inputState.SetValue("Limoni Composable Lego UI")

	list := limoni.NewList(
		"Unified Component",
		"Pad, Border, Align",
		"VStack & HStack",
		"ZStack (Depth)",
		"Overlay (Compositor)",
		"Transform (Uppercase/Mask)",
		"Half-Block Borders",
	).WithState(listState).WithHighlightSymbol("👉 ")

	table := limoni.NewTable().
		WithHeaders("METRIC", "STATUS", "ALLOCS").
		WithRow("VStack / HStack", "OPTIMAL", "0 B/op").
		WithRow("ZStack Depth", "OPTIMAL", "0 B/op").
		WithRow("Overlay Compositor", "OPTIMAL", "0 B/op").
		WithRow("Transform Pipeline", "OPTIMAL", "0 B/op")

	input := limoni.NewTextInput("demo_input").WithState(inputState)

	listColumn := limoni.Overlay(
		limoni.Border(limoni.AsComponent(list), widgets.SymbolsSingle, limoni.Fg(limoni.Hex("#FFCC00"))),
		limoni.FixedSize(16, 1,
			limoni.Label(" 🚀 PARITY BADGE ", limoni.Bold().WithFg(limoni.Hex("#000000")).WithBg(limoni.Hex("#00FFAA"))),
		),
		4, 0,
	)

	view := limoni.VStack(
		limoni.FixedSize(0, 3, limoni.Border(
			limoni.Pad(
				limoni.HStack(
					limoni.Label("🍋 LIMONI LEGO ARCHITECTURE", limoni.Bold()),
					limoni.Uppercase(limoni.Label("zero-alloc v1.0")),
					limoni.Label("[ ESC: Exit ]"),
				).WithJustify(limoni.JustifySpaceBetween).WithAlignItems(limoni.AlignItemsCenter),
				0, 1, 0, 1,
			),
			widgets.SymbolsRounded,
			limoni.Fg(limoni.Hex("#00FFAA")),
		)),
		limoni.Flex(1, limoni.HStack(
			limoni.Flex(1, listColumn),
			limoni.Flex(2, limoni.VStack(
				limoni.Flex(1, limoni.Border(limoni.AsComponent(table), widgets.SymbolsOuterHalfBlock, limoni.Fg(limoni.Hex("#3399FF")))),
				limoni.FixedSize(0, 3, limoni.HStack(
					limoni.Flex(1, limoni.Border(limoni.AsComponent(input), widgets.SymbolsRounded, limoni.Fg(limoni.Hex("#FF5599")))),
					limoni.Flex(1, limoni.Border(limoni.Pad(limoni.Mask(limoni.Label("secretpassword"), '*'), 0, 1, 0, 1), widgets.SymbolsInnerHalfBlock, limoni.Fg(limoni.Hex("#FFAA88")))),
				)),
			)),
		)),
		limoni.FixedSize(0, 3, limoni.Border(
			limoni.Pad(
				limoni.HStack(
					limoni.Label("0 B/op Hot-Path Zero Allocation"),
					limoni.Label("Lipgloss & Glyph Parity"),
				).WithJustify(limoni.JustifySpaceBetween),
				0, 1, 0, 1,
			),
			widgets.SymbolsSingle,
			limoni.Fg(limoni.Hex("#666666")),
		)),
	)

	term.Draw(func(f *limoni.Frame) {
		f.RenderComponent(view, f.Area())
	})

	fmt.Println("\n=== VISUAL SNAPSHOT OUTPUT (84x24) ===")
	fmt.Println(term.Snapshot())
	fmt.Println("======================================")
}
