# ⚙️ Core Engine API Reference

Limoni's low-level architecture is organized under the `core/` package hierarchy. These modules provide fine-grained, zero-allocation control over hardware terminals, memory buffers, cell geometries, layered rendering, focus management, and event routing.

---

## 1. `core/cell`

Defines the fundamental atomic unit of the terminal screen: character cells, 24-bit TrueColors, text modifiers, cascading context, and bounding geometry.

### Key Types & Constructors

#### `cell.Color`
Represents 24-bit TrueColor RGB, 8-bit ANSI, or the default terminal color packed into a single 32-bit integer.

```go
// 24-bit TrueColor RGB
colRGB := cell.NewColorRGB(255, 128, 0)

// 8-bit ANSI color (0-255)
colANSI := cell.NewColorANSI(196)

// Default terminal color
colDef := cell.NewColorDefault()
```

#### `cell.Modifier`
Bitmask flags for text formatting:

```go
const (
    ModifierBold            Modifier = 1 << 0
    ModifierDim             Modifier = 1 << 1
    ModifierItalic          Modifier = 1 << 2
    ModifierUnderline       Modifier = 1 << 3
    ModifierDoubleUnderline Modifier = 1 << 4
    ModifierUndercurl       Modifier = 1 << 5
    ModifierBlink           Modifier = 1 << 6
    ModifierReverse         Modifier = 1 << 7
    ModifierHidden          Modifier = 1 << 8
    ModifierStrikethrough   Modifier = 1 << 9
)

// Example: Bold with Neon Cyan text
style := cell.Style{
    Fg: cell.NewColorRGB(0, 210, 255),
    Modifier: cell.ModifierBold | cell.ModifierUnderline,
}
```

#### `cell.Rect`
Represents a 2D bounding rectangle in terminal cell coordinates:

```go
type Rect struct {
    X, Y          uint16
    Width, Height uint16
}

area := cell.NewRect(0, 0, 80, 24)
inside := area.Contains(10, 5)
clipped := area.Intersection(cell.NewRect(20, 10, 40, 10))
```

#### `cell.Context`
Stack-allocated drawing context passed down to widgets during `Draw`:
- `ctx.Area`: Active bounding box.
- `ctx.Style`: Cascading parent style.
- `ctx.RegisterClick(area, callback)`: Registers interactive mouse click zones.
- `ctx.RegisterMouse(area, callback)`: Registers drag / hover / wheel handlers.
- `ctx.RegisterFocus(id)`: Registers focusable widget IDs.
- `ctx.IsFocused(id)`: Checks active focus ownership.

---

## 2. `core/buffer`

High-performance 1D contiguous cell memory matrix and zero-allocation ANSI differential renderer.

### Methods

```go
// Create buffer
buf := buffer.NewBuffer(area)

// Clear buffer (fast path skips clean buffers)
buf.Clear()

// Write individual cell
buf.SetCell(x, y, cell.Cell{
    Content: '█',
    Style: cell.Style{Fg: cell.NewColorRGB(0, 255, 128)},
})

// Write UTF-8 string with automatic continuation markers for double-width runes
buf.SetString(x, y, "Hello Limoni!", cell.Style{Modifier: cell.ModifierBold})

// Generate minimal ANSI diff stream between front and back buffers
writeBuf, err := buffer.Diff(frontBuf, backBuf, writeBuf[:0], true, true)
```

---

## 3. `core/terminal`

The terminal orchestration engine managing double buffering, render frames, focus managers, and layered rendering.

### `terminal.Frame` & Layer System

```go
term, err := terminal.New(b)
if err != nil {
    log.Fatal(err)
}

term.Draw(func(f *terminal.Frame) {
    area := f.Buffer.Area
    f.SetTheme(widgets.DarkTheme())

    // 1. Base Layer (Main application layout)
    f.RenderWidget(myDashboardWidget, area)

    // 2. Modal Layer with Dismissal Callback
    if showModal {
        modalArea := terminal.CenterRect(area, 50, 15)
        f.RegisterLayer("settings_modal", terminal.LayerModal, modalArea, 3000, func() {
            showModal = false
        })

        f.BeginLayer("settings_modal")
        f.RenderWidget(settingsDialog, modalArea)
        f.EndLayer()
    }
})
```

---

## 4. `core/backend`

Hardware abstraction layer supporting Linux TTY/PTY (via pure Go termios ioctls), Windows ConPTY, macOS BSD termios, SSH sessions, and WebAssembly.

```go
// Initialize standard TTY backend
b := backend.NewBackend(os.Stdin, os.Stdout)
b.Setup() // Enables Raw Mode, SGR Mouse Reporting, and Alt Buffer
defer b.Close()

// Event Loop
b.StartEventLoop()
for ev := range b.Events() {
    switch ev.Type {
    case backend.EventKey:
        if ev.Key.Type == backend.KeyEsc || (ev.Key.Type == backend.KeyRune && ev.Key.Ch == 'q') {
            return
        }
    case backend.EventMouse:
        // Automatically routed to active widgets and layers
        term.RouteMouseEvent(ev.Mouse)
    case backend.EventResize:
        // Window dimensions changed
    }
}
```

---

## 5. `core/runtime`

The Elm Architecture (TEA) runtime for declarative state management.

```go
type Model interface {
    Init(ctx context.Context) (Model, Cmd)
    Update(msg Msg) (Model, Cmd)
    View(f *terminal.Frame)
}

program := runtime.New(
    runtime.WithModel(initialModel),
    runtime.WithFPS(60),
)
program.RunTerminal(ctx, term, b)
```
