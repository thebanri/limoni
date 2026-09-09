# 🚀 Getting Started with Limoni

Limoni is an ultra-high-performance, zero-allocation, 60+ FPS Terminal User Interface (TUI) library for Go. It brings modern web and game engine rendering principles to the terminal: 24-bit TrueColor, double-buffered differential rendering, CSS Flexbox and Grid layouts, native image protocols (Kitty, Sixel, iTerm2), a software 3D rasterizer, and a rich, batteries-included widget catalog.

---

## 📦 Installation

Add Limoni to your Go module (Go 1.22+ required):

```bash
go get github.com/thebanri/limoni
```

---

## ⚡ 1-Minute Quick Start: Single Import API

With Limoni's unified root package (`import "github.com/thebanri/limoni"`), you don't need to juggle multiple subpackages. You can build and run applications with zero boilerplate.

### Approach 1: High-Speed Interactive App (`limoni.Run`)

The simplest way to create an interactive terminal app with automatic raw mode, alt-screen, mouse tracking, and event handling:

```go
package main

import (
	"fmt"
	"github.com/thebanri/limoni"
)

func main() {
	count := 0

	limoni.Run(func(f *limoni.Frame, ev *limoni.Event) bool {
		// 1. Handle user inputs
		if ev != nil && ev.Type == limoni.EventKey {
			switch ev.Key.Ch {
			case 'q', 'Q':
				return false // Exit application
			case '+', '=':
				count++
			case '-', '_':
				count--
			case 'r', 'R':
				count = 0
			}
			if ev.Key.Type == limoni.KeyEsc {
				return false
			}
		}

		// 2. Split screen layout into header, body, and footer
		rows := limoni.SplitVertical(f.Area(), limoni.Fixed(3), limoni.Fill(), limoni.Fixed(3))

		// 3. Render Header
		header := limoni.NewBlock().
			WithTitle(" 🍋 LIMONI COUNTER ").
			WithTitleAlign(limoni.AlignCenter).
			WithBorderStyle(limoni.Fg(limoni.RGB(255, 215, 0)))
		f.RenderWidget(header, rows[0])

		// 4. Render Body Card
		color := limoni.RGB(80, 220, 140)
		if count < 0 {
			color = limoni.RGB(255, 80, 80)
		}
		body := limoni.NewBlock().
			Rounded().
			WithTitle(" STATE ").
			WithPadding(1, 2, 1, 2).
			WithChild(limoni.NewParagraph(fmt.Sprintf("Current Value: %d", count)).
				WithStyle(limoni.Fg(color).Bold()))
		f.RenderWidget(body, rows[1])

		// 5. Render Footer Shortcuts
		footer := limoni.NewBlock().
			WithTitle(" [+] Increment  [-] Decrement  [R] Reset  [Q/Esc] Quit ").
			WithBorderStyle(limoni.Fg(limoni.RGB(100, 110, 130)))
		f.RenderWidget(footer, rows[2])

		return true // Continue running
	})
}
```

Run it directly:
```bash
go run main.go
```

---

### Approach 2: One-Liner Static / Dashboard Display (`limoni.Start`)

If you just want to render a dashboard or snapshot without managing an event loop:

```go
package main

import "github.com/thebanri/limoni"

func main() {
	limoni.Start(func(f *limoni.Frame) {
		cols := limoni.SplitHorizontal(f.Area(), limoni.Percentage(30), limoni.Percentage(70))

		sidebar := limoni.NewBlock().
			WithTitle(" Navigation ").
			Rounded().
			WithChild(limoni.NewList("Dashboard", "Telemetry", "Settings").
				WithHighlightSymbol("👉 "))
		f.RenderWidget(sidebar, cols[0])

		content := limoni.NewBlock().
			WithTitle(" System Overview ").
			Rounded().
			WithChild(limoni.NewParagraph("Welcome to Limoni! High performance TUI in Go.").
				WithStyle(limoni.Fg(limoni.Hex("#00FFAA")).Bold()))
		f.RenderWidget(content, cols[1])
	})
}
```

---

### Approach 3: The Elm Architecture (TEA)

For large, complex applications requiring structured state management, commands, async workers, and sub-models, Limoni provides a first-class TEA runtime:

```go
package main

import (
	"context"
	"fmt"

	"github.com/thebanri/limoni"
)

type Model struct {
	count int
}

func (m Model) Init() []limoni.Cmd {
	return nil
}

func (m Model) Update(msg limoni.Msg) limoni.UpdateResult {
	switch msg := msg.(type) {
	case limoni.KeyMsg:
		switch msg.Key.Ch {
		case 'q', 'Q':
			return limoni.UpdateResult{Quit: true}
		case '+', '=':
			m.count++
			return limoni.UpdateResult{Redraw: true}
		case '-', '_':
			m.count--
			return limoni.UpdateResult{Redraw: true}
		}
	}
	return limoni.UpdateResult{}
}

func (m Model) View(f *limoni.Frame) {
	card := limoni.NewBlock().
		WithTitle(" TEA Architecture ").
		Rounded().
		WithChild(limoni.NewParagraph(fmt.Sprintf("Counter: %d (Press +/- or Q)", m.count)).
			WithStyle(limoni.Fg(limoni.Hex("#00E5FF")).Bold()))
	f.RenderWidget(card, f.Area())
}

func main() {
	program := limoni.NewProgram(
		limoni.WithModel(Model{count: 0}),
		limoni.WithAltScreen(),
		limoni.WithFPS(60),
	)
	if err := program.Run(context.Background()); err != nil {
		fmt.Printf("Application exited with error: %v\n", err)
	}
}
```

---

## 🎨 Fluent Styling & Color Helpers

Limoni provides intuitive helpers for colors and modifiers:

```go
// 24-bit TrueColor RGB & Hex
gold  := limoni.RGB(255, 215, 0)
neon  := limoni.Hex("#00FFAA")
ansi  := limoni.ANSI(196)

// Fluent Styles
style := limoni.NewStyle().
	WithFg(neon).
	WithBg(limoni.RGB(20, 24, 32)).
	Bold().
	Underline()

// Direct helpers
fgOnly := limoni.Fg(gold).Bold()
```

---

## 📐 Layout Ergonomics

Limoni supports CSS Flexbox and Grid layouts:

```go
// Fast vertical partition
rows := limoni.SplitVertical(area, limoni.Fixed(3), limoni.Fill(), limoni.Fixed(1))

// Fast horizontal partition
cols := limoni.SplitHorizontal(area, limoni.Percentage(25), limoni.Percentage(75))

// Weighted ratios
sections := limoni.SplitVertical(area, limoni.Ratio(2), limoni.Ratio(1)) // 2/3 and 1/3
```

---

## 📚 Next Steps

- [Core Engine Architecture & Zero-Alloc Diff](./architecture.md)
- [Complete Widgets Reference](./widgets-reference.md)
- [Flexbox & Grid Layout Guide](./layout-guide.md)
- [3D Graphics & Native Image Protocols](./graphics-and-canvas.md)
- [Full Application Gallery & Examples](./examples.md)
