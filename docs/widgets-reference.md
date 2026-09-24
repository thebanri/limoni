# 🧩 Limoni Widgets Reference & API Guide

Limoni includes a comprehensive, batteries-included suite of high-performance widgets designed for modern terminal interfaces. All widgets implement the `widgets.Widget` interface (`Draw(ctx cell.Context, buf *buffer.Buffer)` and `SizeHint(maxArea cell.Rect) (uint16, uint16)`).

You can use either the unified root package (`github.com/thebanri/limoni`) with fluent builder methods or the underlying `github.com/thebanri/limoni/widgets` package.

---

## 1. Structural & Container Widgets

### `Block`
The foundational container providing borders, titles, padding, background fill, and child routing.

#### Fluent Constructor
```go
block := limoni.NewBlock().
    WithTitle(" 📦 SYSTEM TELEMETRY ").
    WithTitleAlign(limoni.AlignCenter).
    Rounded().                                  // Rounded corners (╭ ╮ ╰ ╯)
    WithBorderStyle(limoni.Fg(limoni.Hex("#00FFAA"))).
    WithPadding(1, 2, 1, 2).                    // Top, Right, Bottom, Left
    WithChild(childWidget)

f.RenderWidget(block, area)
inner := block.Inner(area)                      // Exact printable area inside borders and padding
```

#### Border Styles
* `Rounded()`: `╭─╮ │ ╰─╯`
* `Single()`: `┌─┐ │ └─┘`
* `Double()`: `╔═╗ ║ ╚═╝`
* `Thick()`: `┏━┓ ┃ ┗━┛`
* `BlockBorder()`: `███ █ ███`

---

### `Label`
Ultra-lightweight, zero-overhead text widget designed for single or multi-line text strings, modal dialogs, status lines, and inspector panels.

In **v0.2.4+**, `Label` guarantees solid background coverage: when a background color is set on the widget or inherited from its parent `cell.Context`, all cells in the bounding area—including trailing spaces and empty rows—are filled with the background color, completely preventing underlying terminal text from bleeding through. Supports full `SizeHint` and `Measure` layout negotiation.

```go
lbl := widgets.NewLabel("Press [Ctrl+S] to export OBJ mesh").
    WithStyle(cell.Style{
        Fg: cell.ColorYellow,
        Bg: cell.NewColorRGB(30, 32, 48),
    })

f.RenderWidget(lbl, area)

// Or as an inline Lego Composable component:
badge := limoni.HStack(
    limoni.Label("Status: ", limoni.Fg(limoni.ColorGray)),
    limoni.Label("READY", limoni.Fg(limoni.ColorGreen).Bold()),
)
```

---

### `Paragraph`
Text rendering widget with automatic word wrapping, alignment, and style inheritance. In **v0.2.4+**, setting a background color fills the entire bounding box so underlying modal/window content is reliably occluded.

```go
p := limoni.NewParagraph("Limoni delivers 60-240 FPS zero-allocation rendering.").
    WithWrap(true).
    WithStyle(cell.Style{
        Fg: cell.ColorWhite,
        Bg: cell.NewColorRGB(20, 24, 38),
    }).
    WithAlignment(limoni.AlignCenter)

f.RenderWidget(p, area)
```

---

### `Markdown`
Interactive Markdown viewer supporting headers (`#`, `##`), bullet lists (`-`, `*`), horizontal rules (`---`), bold (`**`), italic (`*`), and inline code (`` ` ``). Features mouse wheel and drag scrolling.

```go
md := limoni.NewMarkdown(markdownText).
    WithID("doc_viewer").
    WithStyle(limoni.Fg(limoni.RGB(220, 225, 235))).
    WithFocusedStyle(limoni.Fg(limoni.Hex("#00E5FF"))).
    WithScrollOffset(&scrollOffset)

f.RenderWidget(md, area)
```

> [!NOTE]
> `Markdown` automatically inherits the container or theme background color, preventing transparent text cut-outs across terminals with translucent profiles.

---

### `RichText`, `Text` & `Span`
Fine-grained multi-styled formatted text composition:

```go
text := limoni.NewText(
    limoni.NewLine(
        limoni.NewSpan("Server Status: ").Bold(),
        limoni.NewSpan("ONLINE").WithStyle(limoni.Fg(limoni.RGB(80, 220, 140)).Bold()),
    ),
    limoni.NewLine(
        limoni.NewSpan("Latency: 14ms").Dim(),
    ),
)
f.RenderWidget(text, area)
```

---

## 2. Collections & Data Presentation

### `Table`
Structured tabular presentation with column constraints, row selection, zebra striping, and grid lines.

```go
table := limoni.NewTable().
    WithHeaders("PID", "NAME", "CPU %", "MEMORY").
    WithRow("1024", "nginx", "4.2%", "42 MB").
    WithRow("2048", "postgres", "12.8%", "256 MB").
    WithConstraints(
        limoni.Fixed(8),
        limoni.Fill(),
        limoni.Fixed(10),
        limoni.Fixed(12),
    ).
    WithGrid(true).
    WithSelected(selectedIndex)

f.RenderWidget(table, area)
```

---

### `VirtualDataView`
Extreme-scale virtualized table capable of smoothly scrolling over **1,000,000+ rows** with zero heap allocations during scrolling:

```go
view := widgets.VirtualDataView{
    ID:            "infinite_log_view",
    Source:        logDataSource, // Implements widgets.VirtualDataSource
    Prefetch:      20,
    Offset:        &scrollOffset,
    Style:         cell.Style{Fg: cell.NewColorRGB(190, 195, 205)},
    SelectedStyle: cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: cell.NewColorRGB(0, 80, 130)},
}
f.RenderWidget(view, area)
```

---

### `List`
Interactive scrollable list with custom highlight symbols and item selection:

```go
list := limoni.NewList("Dashboard", "Analytics", "Settings", "Logs").
    WithHighlightSymbol("👉 ").
    WithSelected(selectedIndex).
    WithSelectedStyle(limoni.Fg(limoni.Hex("#00FFAA")).Bold())

f.RenderWidget(list, area)
```

---

### `TreeView`
Collapsible hierarchical tree with nested nodes, icon symbols, and expand/collapse state:

```go
tree := widgets.NewTreeView(rootNode).
    WithIcons("📁 ", "📄 ").
    WithSelectedStyle(limoni.Fg(limoni.Hex("#00FFAA")).Bold())

f.RenderWidget(tree, area)
```

---

## 3. Telemetry & Charts

### `ProgressBar`
Visual progress indicators with percentage labels:

```go
bar := widgets.ProgressBar{
    Value:       76.5,
    Min:         0,
    Max:         100,
    ShowPercent: true,
    FilledStyle: cell.Style{Fg: cell.NewColorRGB(80, 220, 140)},
    EmptyStyle:  cell.Style{Fg: cell.NewColorRGB(60, 65, 75)},
}
f.RenderWidget(bar, area)
```

---

### `Gauge` & `LineGauge`
`Gauge` fills its whole area in proportion to `Ratio`, moving its edge in eighths of a cell, with a centred label that turns reverse where it crosses the fill. `LineGauge` is one row: a label, then a line drawn heavy up to the ratio and light after it.

```go
f.RenderWidget(widgets.Gauge{Ratio: 0.42, GaugeStyle: cell.Style{Fg: cell.NewColorRGB(80, 220, 140)}}, area)
f.RenderWidget(widgets.LineGauge{Ratio: 0.42, Label: "Upload"}, row) // Upload ━━━━━━━━──────────
```

---

### `BarChart`, `LineChart`, `PieChart` & `Sparkline`
High-resolution ASCII and TrueColor charts:

```go
// Bar Chart
barchart := widgets.BarChart{
    Data:   []float64{12, 34, 56, 89, 45},
    Labels: []string{"Mon", "Tue", "Wed", "Thu", "Fri"},
    Color:  cell.NewColorRGB(0, 210, 255),
}
f.RenderWidget(barchart, area)

// Pie Chart
pie := widgets.PieChart{
    Slices: []widgets.PieSlice{
        {Label: "CPU", Value: 40, Color: cell.NewColorRGB(255, 90, 90)},
        {Label: "RAM", Value: 35, Color: cell.NewColorRGB(80, 220, 140)},
        {Label: "Disk", Value: 25, Color: cell.NewColorRGB(100, 200, 255)},
    },
    ShowLegend: true,
}
f.RenderWidget(pie, area)
```

---

## 4. Input Controls & Forms

### `Autocomplete`
A text input with a list of suggestions under it that narrows as the user types, matched fuzzily ("gco" finds "git checkout"). Matching runs in `HandleKey`, not in `Draw`.

```go
state := &widgets.AutocompleteState{}
// in the event handler:
state.HandleKey(ev.Key, commands) // ↑/↓ move, Tab or Enter accepts, Esc closes
// in the draw function:
f.RenderWidget(widgets.Autocomplete{ID: "cmd", Suggestions: commands, State: state}, cell.NewRect(x, y, 40, 7))
```

---

### `FilePicker`
Browses a directory: `..`, directories first with a trailing slash, sizes on the right. Enter opens a directory or chooses a file (`State.Chosen`), Backspace goes up, Ctrl+H shows dot files. `Extensions` and `DirsOnly` filter.

```go
picker := widgets.NewFilePickerState(".")
f.RenderWidget(widgets.FilePicker{ID: "files", State: picker}, area)
```

---

### `Calendar`
One month as a grid, 20×8 cells. Arrows move the selection by a day or a week, Page Up/Down by a month; a click selects. `FirstWeekday`, `MonthNames` and `WeekdayNames` localise it; `DayStyle` marks days with events. It never reads the clock: pass `Today`.

```go
f.RenderWidget(widgets.Calendar{State: calState, Today: time.Now(), FirstWeekday: time.Monday}, area)
```

---

### `TextInput`
Single-line editable input with cursor tracking, text selection, and placeholder support:

```go
input := limoni.NewTextInput("api_key_input").
    WithPlaceholder("Enter secret token...").
    WithStyle(limoni.Fg(limoni.RGB(220, 225, 235))).
    WithFocusedStyle(limoni.Fg(limoni.Hex("#00E5FF")).Bold())

f.RenderWidget(input, area)
```

---

### `Checkbox` & `RadioButton`
Toggle and mutually exclusive selection controls:

```go
cb := widgets.Checkbox{
    ID:      "telemetry_cb",
    Label:   "Enable OpenTelemetry exports",
    Checked: isChecked,
    OnToggle: func(val bool) { isChecked = val },
}
f.RenderWidget(cb, area)
```

---

### `Slider`
Continuous or discrete draggable slider:

```go
slider := widgets.Slider{
    ID:          "volume_slider",
    Min:         0,
    Max:         100,
    State:       sliderState,
    FilledStyle: cell.Style{Fg: cell.NewColorRGB(80, 220, 140)},
    ThumbStyle:  cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Modifier: cell.ModifierBold},
}
f.RenderWidget(slider, area)
```

---

## 5. Layout, Text & Chrome

### `SplitPane`
Two widgets side by side (`SplitHorizontal`) or stacked (`SplitVertical`) with a divider the mouse drags and the arrow keys move (`SplitState.HandleKey`). `MinFirst` and `MinSecond` keep each pane usable. Where the terminal can change the mouse pointer, the divider shows a resize arrow.

```go
f.RenderWidget(widgets.SplitPane{First: tree, Second: preview, State: split, MinFirst: 20}, area)
```

---

### `StatusBar`
The last line of a full-screen application: items on the left, centre and right, each an optional key and text. When space runs out the left group stays, the right is cut from its left edge, and the centre is dropped.

```go
f.RenderWidget(widgets.StatusBar{
    Left:  []widgets.StatusItem{{Text: "NORMAL"}},
    Right: []widgets.StatusItem{{Key: "^Q", Text: "quit"}, {Text: "12:04"}},
}, bottomRow)
```

---

### `CodeView`
Source code with line numbers and syntax highlighting for Go, Python, JavaScript/TypeScript, Rust, C-family, shell, JSON and YAML — a small built-in lexer, no dependency. Highlighting happens once in `SetSource`; `Draw` only paints.

```go
code := widgets.NewCodeViewState(src, widgets.LanguageForFile("main.go"))
f.RenderWidget(widgets.CodeView{State: code}, area)
```

---

### `BigText`
Large letters from the public-domain font8x8 bitmap font, at 8×8, 8×4 (half-height) or 4×4 (quadrant) cells a character — for titles, clocks and counters.

```go
f.RenderWidget(widgets.BigText{Text: "12:04", Size: widgets.BigTextHalfHeight, Alignment: widgets.AlignCenter}, area)
```

---

## 6. Modals, Dialogs & Overlays

### `Dialog`
Glassmorphism modal dialog featuring glowing gradient borders, drop shadow, draggable title bar, and keyboard focus routing:

```go
dialog := widgets.Dialog{
    ID:         "exit_dialog",
    Title:      " ⚠️ CONFIRM ACTION ",
    Message:    "Are you sure you want to proceed?",
    SubMessage: "All unsaved changes will be permanently lost.",
    Shadow:     true,
    Buttons: []widgets.DialogButton{
        {
            Text: "Cancel",
            Handler: func() { closeDialog() },
        },
        {
            Text: "Confirm",
            Handler: func() { executeAction() },
        },
    },
    ButtonStyle:        cell.Style{Fg: cell.NewColorRGB(220, 220, 220), Bg: cell.NewColorRGB(45, 45, 45)},
    ButtonFocusedStyle: cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: cell.NewColorRGB(80, 220, 140), Modifier: cell.ModifierBold},
}

f.BeginFocusScope("exit_dialog")
f.RenderWidget(dialog, modalArea)
```

---

### `CommandPalette`
Fuzzy-search command palette (similar to VSCode / Sublime Text `Ctrl+P`):

```go
palette := widgets.CommandPalette{
    Items: []widgets.CommandItem{
        {Label: "Git: Commit", Category: "Version Control", Handler: doCommit},
        {Label: "View: Toggle Fullscreen", Category: "Display", Handler: toggleFull},
    },
}
f.RenderWidget(palette, area)
```

---

## 7. 3D Graphics & Canvas

### `Canvas`
Vector drawing canvas with $2 \times 4$ sub-pixel dots per cell. `Marker` picks how the dots are drawn: Braille (the default), sextants ($2 \times 3$), quadrants ($2 \times 2$), half blocks or full blocks. `LineChart` and `PieChart` take the same `Marker`.

```go
cv := widgets.NewCanvas(width, height)
cv.DrawLine(0, 0, 100, 50, cell.Style{Fg: cell.NewColorRGB(0, 255, 180)})
cv.DrawCircle(50, 25, 20, cell.Style{Fg: cell.NewColorRGB(255, 200, 0)})
f.RenderWidget(cv, area)
```

---

### `Viewer3D`
A software 3D rasteriser: OBJ, STL, PLY and GLB meshes (`graphics.LoadOBJ` and friends) or the built-in primitives, with near-plane clipping, a depth buffer in every mode, and flat, Lambert, Gouraud (smooth vertex normals) or texture shading.

```go
v3d := &widgets.Viewer3D{
    Model:   graphics.NewTorus(1, 0.4, 32, 16),
    Shading: widgets.ShadingGouraud, // ShadingFlat, ShadingLambert, ShadingTexture, ShadingWireframe
    RotX:    35,
    RotY:    angle,
}
f.RenderWidget(v3d, area)
```

It draws Braille dots by default; `Marker: widgets.MarkerSextant` or `MarkerQuadrant` draws solid blocks for fonts without Braille. `Pixels: true` renders the model as a picture and sends it with the terminal's image protocol (kitty, iTerm2, Sixel), falling back to dots where there is none. A still model is encoded once; a moving one costs an encode per frame — about 7 ms and 2.7 MB for a 60×24 area with kitty on the machine this was measured on — so it is opt-in.
