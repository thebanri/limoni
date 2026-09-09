# ⚙️ Core Engine Architecture & API Reference

Limoni is built upon a modular, layered architecture designed from the ground up for **zero heap allocations** on the rendering hot path, hardware cursor synchronization, and deterministic event processing.

---

## 1. Unified Root Facade (`package limoni`)

For 95% of applications, you only need to import the root package:

```go
import "github.com/thebanri/limoni"
```

The root package re-exports:
- **Core Types**: `Frame`, `Terminal`, `Rect`, `Style`, `Color`, `Event`, `KeyEvent`, `MouseEvent`.
- **High-Level Runners**: `limoni.Start()`, `limoni.Run()`, `limoni.NewProgram()`.
- **Layout Splitters**: `limoni.SplitVertical()`, `limoni.SplitHorizontal()`, `limoni.Fixed()`, `limoni.Percentage()`, `limoni.Fill()`, `limoni.Ratio()`.
- **Fluent Builders**: `limoni.NewBlock()`, `limoni.NewParagraph()`, `limoni.NewTable()`, `limoni.NewList()`, `limoni.NewTextInput()`, `limoni.NewMarkdown()`.
- **Color & Style Helpers**: `limoni.RGB()`, `limoni.Hex()`, `limoni.ANSI()`, `limoni.Fg()`, `limoni.Bg()`, `limoni.Bold()`, `limoni.Italic()`.

---

## 2. `core/cell` — Atomic Cells & Geometric Math

Defines the fundamental building block of the terminal grid: character cells, TrueColor RGB, text formatting modifiers, and bounding rectangles.

### `cell.Color`
Represents 24-bit TrueColor RGB, 8-bit ANSI, or default terminal colors packed into a single 32-bit word:

```go
colRGB  := cell.NewColorRGB(0, 255, 180)
colANSI := cell.NewColorANSI(196)
colDef  := cell.NewColorDefault()
```

### `cell.Style`
12-byte packed struct containing foreground, background, and bitmask modifiers:

```go
style := cell.Style{
    Fg: cell.NewColorRGB(0, 210, 255),
    Bg: cell.NewColorRGB(18, 20, 24),
    Modifier: cell.ModifierBold | cell.ModifierUnderline,
}

// Merging styles (cascading overrides):
merged := baseStyle.Merge(overrideStyle)
```

### `cell.Rect`
2D bounding box math with zero heap allocations:

```go
area := cell.NewRect(0, 0, 80, 24)
isInside := area.Contains(10, 5)
intersection := area.Intersection(otherArea)
```

### `cell.Context`
Ephemeral draw context passed down to widgets:
- `ctx.Area`: Active bounding rectangle.
- `ctx.Style`: Inherited style from parent container or theme.
- `ctx.ThemeStyle(role)`: Resolves semantic theme colors (`"surface"`, `"border"`, `"text"`).
- `ctx.RegisterClick(area, handler)`: Registers clickable zones.
- `ctx.RegisterMouse(area, handler)`: Registers mouse hover, drag, and scroll zones.
- `ctx.RegisterFocus(id)`: Registers focusable element boundaries.

---

## 3. `core/buffer` — 1D Memory Matrix & ANSI Diff Engine

High-performance contiguous cell buffer with differential ANSI rendering.

### Highlights
- **1D Flat Memory Array**: `Buffer.Content` is stored as a contiguous slice of `cell.Cell`, maximizing CPU cache locality and eliminating pointer indirection.
- **Differential Rendering (`buffer.Diff`)**: Compares `front` and `back` buffers, calculating the absolute minimal sequence of ANSI escape codes required to update the hardware screen.
- **Unicode East Asian Width (`W`/`F`) Accuracy**: Strict table-driven width calculations ensure that single-width symbols (`✓`, `⚠`) occupy 1 column, while emojis (`🔴`, `🚀`, `☕`) occupy 2 columns without cursor desynchronization.
- **Continuation Cell Protection**: When wide characters are partially occluded or restored (e.g. dragging a modal window), continuation cells (`RuneContinuation`) automatically trigger wide-rune invalidation, preventing ghost borders and visual shredding.

```go
buf := buffer.NewBuffer(area)
buf.SetCellDirect(x, y, cell.Cell{Content: 'A', Style: style})
buf.SetString(x, y, "🚀 Hello Limoni", style)

// Generate ANSI delta stream
writeBuf, err := buffer.Diff(frontBuf, backBuf, writeBuf[:0], true, true)
```

---

## 4. `core/terminal` — Engine, Layers, Modals & Focus

Owns the terminal lifecycle, double buffers, frame generation, and input routing.

### Key Capabilities
- **60+ FPS Rendering Pipeline**: Renders frames to an in-memory buffer, diffs against the previous frame, and flushes output via buffered I/O.
- **Layer Stacking**: `f.BeginLayer("overlay")` and `f.EndLayer()` allow non-destructive overlays and floating panels.
- **Modal Stack Isolation**: `f.RegisterModal("dialog", area, onDismiss)` sandboxes events, automatically blocking underlying widgets from receiving mouse clicks or keyboard events while a modal is active.
- **Focus Scoping**: `f.BeginFocusScope("modal_id")` restricts `Tab` / `Shift+Tab` keyboard navigation strictly to interactive elements inside the modal.

---

## 5. `core/engine` — The Elm Architecture (TEA)

Provides a predictable, functional state management loop:

```
[Init] ──> Model + Cmd
             │
             ▼
[Message] ──> [Update] ──> Model + Cmd
                            │
                            ▼
                         [View] ──> Frame Render
```

### Safety & Concurrency Guarantees
- **Deterministic Command Ordering**: Commands are executed concurrently on worker goroutines, but their results are buffered and delivered to `Update` in strict dispatch sequence order.
- **Strict Cancellation Precedence**: When context cancellation (`ctx.Done()`) or program shutdown occurs, pending command results and queued messages are immediately discarded, preventing race conditions or late mutations after exit.
- **Safe Panic Recovery**: `WithPanicHandler` catches panics in user commands or models, preventing process termination and allowing telemetry logging.

---

## 6. `core/driver` — Cross-Platform VT & Raw Terminal Engine

Communicates directly with the operating system terminal driver:

- **Linux / macOS**: Configures `termios` for raw mode, enables alternate screen buffer (`\x1b[?1049h`), mouse tracking (`\x1b[?1006h`), and bracketed paste.
- **Windows**: Uses native Win32 Console API (`GetConsoleMode`, `SetConsoleMode`) with `ENABLE_VIRTUAL_TERMINAL_PROCESSING` and native event loop input decoding.
- **Signal Handling**: Listens for `SIGWINCH` on Unix and console resize events on Windows to trigger instantaneous window reflows.
