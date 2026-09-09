# Bubble Tea → Limoni Migration Guide

The `compat/bubbletea` package allows you to run existing Bubble Tea models on top of Limoni's high-performance runtime with minimal changes. This guide outlines the incremental migration path: from running via the compatibility adapter to transitioning to Limoni's native API.

## Step 1: Running with the Adapter

The standard Bubble Tea interface is fully preserved:

```go
type Model interface {
	Init() Cmd
	Update(Msg) (Model, Cmd)
	View() string
}
```

The only change is the import path and program instantiation:

```go
// Before:
// import tea "github.com/charmbracelet/bubbletea"
// p := tea.NewProgram(model)
// _, err := p.Run()

// After:
import tea "github.com/thebanri/limoni/compat/bubbletea"

p := tea.NewProgram(model)
err := p.RunTerminal(context.Background())
```

- `Program.RunTerminal(ctx)` → Delegates terminal driver setup, raw mode event loop, and frame rendering to Limoni's engine (requires an active TTY).
- `Program.Run(ctx)` → Executes only the Init/Update loop without binding to a physical terminal; ideal for headless unit tests.

## Step 2: Message Mapping

Limoni messages are automatically translated into their Bubble Tea equivalents:

| Limoni (`core/engine` / `limoni.*`) | Bubble Tea Compatible (`compat/bubbletea`) |
| --- | --- |
| `engine.KeyPressMsg` | `KeyMsg{Type, Runes, Alt, Ctrl, Shift}` |
| `engine.ResizeMsg` | `WindowSizeMsg{Width, Height}` |
| All other messages | Passed through without modification |

Automatically translated keys include: `KeyRunes`, `KeyEnter`, `KeyBackspace`, `KeyTab`, `KeyEsc`, `KeyUp`, `KeyDown`, `KeyLeft`, `KeyRight`, and `Ctrl+C`.
`KeyPgUp`, `KeyPgDown`, `KeyHome`, `KeyEnd`, `KeyDelete`, `KeySpace`, and `KeyCtrlA`...`KeyCtrlZ` constants are defined.
`KeyMsg.String()` produces standard Bubble Tea representations such as `"ctrl+c"`, `"enter"`, and `"up"`.

`Ctrl+C` is delivered as `KeyMsg{Type: KeyCtrlC}`. Calling `Quit()` returns `QuitMsg`, which the adapter maps to `engine.UpdateResult{Quit: true}`.

## Step 3: Lipgloss Styles

`compat/bubbletea` provides a `Style` builder that supports a core subset of Lipgloss's fluent API:

```go
style := tea.NewStyle().
	Foreground(cell.NewColorRGB(100, 200, 255)).
	Bold(true).
	Padding(1, 2)

out := style.Render("Hello")       // ANSI-escaped string
cs := style.ToCellStyle()          // Bridge to native cell.Style
```

`ToCellStyle()` serves as a migration bridge for transferring Lipgloss styles to native Limoni widgets (`Block.BorderStyle`, `Paragraph.Style`, etc.).

## Step 4: Moving to the Native API

The adapter renders string output from `View() string` line-by-line into the cell buffer. While functional, migrating to the native `limoni.Model` interface unlocks full differential buffer rendering, zero-allocation cell styling, and declarative layout (with a single import `"github.com/thebanri/limoni"`):

```go
package main

import (
	"context"
	"github.com/thebanri/limoni"
)

type model struct {
	counter int
}

func (m *model) Init() []limoni.Cmd { return nil }

func (m *model) Update(msg limoni.Msg) limoni.UpdateResult {
	switch msg := msg.(type) {
	case limoni.KeyPressMsg:
		switch msg.Key.Type {
		case limoni.KeyEsc:
			return limoni.Quit()
		case limoni.KeyUp, limoni.KeyRunes:
			if msg.Key.Runes == '+' {
				m.counter++
				return limoni.Redraw()
			}
		case limoni.KeyDown:
			m.counter--
			return limoni.Redraw()
		}
	}
	return limoni.Noop()
}

func (m *model) View(f *limoni.Frame) {
	f.RenderWidget(
		limoni.NewBlock().
			Title(" Limoni Native App ").
			Border(limoni.BorderRounded).
			Style(limoni.Fg(limoni.ColorCyan)),
		f.Area(),
	)
}

func main() {
	_ = limoni.RunProgram(context.Background(), &model{})
}
```

### Feature Equivalents

| Bubble Tea | Limoni Native (`limoni.*`) |
| --- | --- |
| `Init() Cmd` | `Init() []limoni.Cmd` |
| `Update(Msg) (Model, Cmd)` | `Update(limoni.Msg) limoni.UpdateResult` |
| `View() string` | `View(*limoni.Frame)` (zero-alloc contiguous cell buffer) |
| `tea.Batch(a, b)` | `[]limoni.Cmd{a, b}` |
| `tea.Quit` | `limoni.Quit()` or `limoni.UpdateResult{Quit: true}` |
| String concatenation layout | `limoni.SplitVertical` / `limoni.FlexLayout` / `component.VStack` |

To scaffold a new project:

```bash
go run github.com/thebanri/limoni/cmd/limoni@latest new myapp
```

## Known Limitations

- `View() string` does not carry rich per-cell RGB style metadata across string boundaries; text is rendered using default styles. For rich styling, migrate to `View(*terminal.Frame)`.
- Commands in `tea.Batch` execute sequentially. For concurrent command execution, return a slice of `engine.Cmd`.
- Mouse events are not mapped to `KeyMsg`; `engine.MousePressMsg` events are passed through directly to the model.
