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

### `Paragraph`
Text rendering widget with automatic word wrapping, alignment, and style inheritance.

```go
p := limoni.NewParagraph("Limoni delivers 60+ FPS zero-allocation rendering.").
    WithWrap(true).
    WithStyle(limoni.Fg(limoni.RGB(220, 225, 235)).Bold()).
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

### `ProgressBar` & `LineGauge`
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

## 5. Modals, Dialogs & Overlays

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

## 6. 3D Graphics & Canvas

### `Canvas`
Vector drawing canvas with $2 \times 4$ sub-pixel Braille dots:

```go
cv := widgets.NewCanvas(width, height)
cv.DrawLine(0, 0, 100, 50, cell.Style{Fg: cell.NewColorRGB(0, 255, 180)})
cv.DrawCircle(50, 25, 20, cell.Style{Fg: cell.NewColorRGB(255, 200, 0)})
f.RenderWidget(cv, area)
```

---

### `Viewer3D`
Hardware-independent software 3D rasterizer with mesh loading (STL, OBJ, PLY) and real-time lighting shaders:

```go
v3d := widgets.NewViewer3D()
v3d.LoadMesh(loadedMesh)
v3d.SetShadingMode(widgets.ShadingGouraud) // or ShadingLambert, ShadingWireframe
v3d.SetRotation(rotX, rotY, rotZ)
f.RenderWidget(v3d, area)
```
