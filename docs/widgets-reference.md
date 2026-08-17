# 🧩 Rich Widgets Reference & API Guide

Limoni includes a comprehensive, batteries-included suite of high-performance widgets designed for modern terminal interfaces. All widgets implement the `widgets.Widget` interface with `Draw` and `SizeHint`.

---

## 1. Container & Structural Widgets

### `widgets.Block`
The foundational container providing rounded/double borders, titles, padding, and nested inner layout negotiation.

```go
block := widgets.Block{
    Title:          " 📦 SERVER TELEMETRY ",
    TitleAlignment: widgets.AlignCenter,
    Borders:        widgets.BorderAll,
    BorderSymbols:  widgets.SymbolsRounded,
    BorderStyle:    cell.Style{Fg: cell.NewColorRGB(0, 210, 255)},
    TitleStyle:     cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Modifier: cell.ModifierBold},
    PaddingLeft:    1,
    PaddingRight:   1,
}
frame.RenderWidget(block, area)
innerArea := block.Inner(area)
```

---

## 2. Data Presentation & Telemetry

### `widgets.Table`
Full-featured table with column constraints, headers, rows, multi-selection, and automatic scrolling.

```go
table := &widgets.Table{
    Header: &widgets.TableRow{
        Cells: []widgets.TableCell{
            {Text: "PID", Style: cell.Style{Fg: cell.NewColorRGB(0, 255, 180), Modifier: cell.ModifierBold}},
            {Text: "PROCESS", Style: cell.Style{Fg: cell.NewColorRGB(0, 255, 180), Modifier: cell.ModifierBold}},
            {Text: "CPU %", Style: cell.Style{Fg: cell.NewColorRGB(0, 255, 180), Modifier: cell.ModifierBold}},
        },
    },
    Rows: []widgets.TableRow{
        {Cells: []widgets.TableCell{{Text: "1024"}, {Text: "nginx"}, {Text: "4.2%"}}},
        {Cells: []widgets.TableCell{{Text: "2048"}, {Text: "postgres"}, {Text: "12.8%"}}},
    },
    Constraints: []widgets.TableConstraint{
        {Type: widgets.ConstraintFixed, Value: 8},
        {Type: widgets.ConstraintFill},
        {Type: widgets.ConstraintFixed, Value: 10},
    },
    DrawGrid: true,
}
frame.RenderWidget(table, area)
```

### `widgets.VirtualDataView`
High-performance virtualized table capable of rendering **1,000,000+ rows** with zero heap allocations during scrolling.

```go
vView := widgets.VirtualDataView{
    ID:            "log_stream",
    State:         state,
    Source:        logSource, // Implements widgets.VirtualDataSource
    Prefetch:      20,
    Offset:        &offset,
    Style:         cell.Style{Fg: cell.NewColorRGB(190, 195, 205)},
    SelectedStyle: cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: cell.NewColorRGB(0, 80, 130), Modifier: cell.ModifierBold},
}
frame.RenderWidget(vView, area)
```

### `widgets.Canvas`
Sub-pixel vector drawing canvas utilizing $2 \times 4$ Braille dot matrices.

```go
cv := widgets.NewCanvas(width, height)
cv.DrawLine(0, 0, 100, 50, cell.Style{Fg: cell.NewColorRGB(0, 255, 180)})
cv.DrawCircle(50, 25, 20, cell.Style{Fg: cell.NewColorRGB(255, 200, 0)})
frame.RenderWidget(cv, area)
```

### `widgets.Viewer3D`
3D software rasterizer with depth buffer, Gouraud/Lambert shading, and Euler orbit controls.

```go
v3d := widgets.NewViewer3D()
v3d.LoadMesh(myMesh)
v3d.SetShadingMode(widgets.ShadingGouraud)
v3d.SetRotation(rx, ry, rz)
frame.RenderWidget(v3d, area)
```

---

## 3. Form & User Input Controls

### `widgets.TextInput`
Single-line text input with cursor positioning, selection, and password masking.

```go
state := widgets.NewTextInputState()
state.SetValue("my-service-cluster")

input := widgets.TextInput{
    ID:           "cluster_name",
    State:        state,
    Placeholder:  "Enter cluster name...",
    FocusedStyle: cell.Style{Fg: cell.NewColorRGB(0, 255, 200), Modifier: cell.ModifierBold},
}
frame.RenderWidget(input, area)
```

### `widgets.Select`
Dropdown select menu supporting mouse interaction, custom open triggers, and keyboard navigation.

```go
sel := widgets.NewSelect([]string{"US-East-1", "EU-Central-1", "AP-Southeast-1"})
sel.SetSelected(1)
frame.RenderWidget(sel, area)
```

### `widgets.Slider`
Interactive numerical slider with mouse dragging and keyboard adjustment.

```go
slider := widgets.NewSlider(0, 100, 75)
slider.SetStep(5)
frame.RenderWidget(slider, area)
```

---

## 4. Modals, Layers & Navigation

### `widgets.Tabs`
Multi-tab navigation header for switching views.

```go
tabs := widgets.NewTabs([]string{"Dashboard", "Logs", "Metrics", "Settings"})
tabs.Select(0)
frame.RenderWidget(tabs, area)
```

### `widgets.CommandPalette`
Fuzzy-search command launcher triggered via keyboard shortcuts (`Ctrl+P`).

```go
palette := widgets.NewCommandPalette()
palette.Register("Reload Services", "System", func() { ... })
palette.Register("Export Logs", "Telemetry", func() { ... })
frame.RenderWidget(palette, area)
```

---

## 5. Next-Generation Productivity & Explorer Widgets

### `widgets.TreeView`
Collapsible hierarchical tree with folder icons, branch guidelines (`│  ├─  └─`), keyboard navigation (Arrow keys, Enter, Space), and mouse click toggles.

```go
state := widgets.NewTreeViewState()

tree := widgets.TreeView{
    ID: "project_tree",
    Roots: []widgets.TreeNode{
        {
            ID: "src", Label: "src", Icon: "📁", Expanded: true,
            Children: []widgets.TreeNode{
                {ID: "main.go", Label: "main.go", Icon: "📄"},
                {ID: "config.json", Label: "config.json", Icon: "⚙️"},
            },
        },
    },
    State:      state,
    ShowGuides: true,
}
frame.RenderWidget(tree, area)
```

### `widgets.ColorPicker`
Interactive color picker with preset swatches, RGB sliders, hex input, and live color previews.

```go
state := widgets.NewColorPickerState(0, 200, 255)

picker := widgets.ColorPicker{
    ID:          "theme_picker",
    State:       state,
    ShowPreview: true,
}
frame.RenderWidget(picker, area)
```

### `widgets.ToastManager`
Notification overlay supporting Info, Success, Warning, and Error toasts with auto-dismiss timers and drop shadows.

```go
toastMgr := widgets.NewToastManager(widgets.ToastTopRight)
toastMgr.Success("Database Connected", "Latency: 2ms")

// In render pass:
toastMgr.Update(time.Now())
toastMgr.Draw(ctx, buf)
```

---

## 6. Rich Data Visualization Charts

### `widgets.BarChart`
Vertical and horizontal bar charts with automatic scaling, custom gaps, and value labels.

```go
chart := widgets.BarChart{
    Data: []widgets.BarData{
        {Label: "Q1", Value: 25, Color: cell.NewColorRGB(0, 255, 128)},
        {Label: "Q2", Value: 68, Color: cell.NewColorRGB(0, 200, 255)},
        {Label: "Q3", Value: 95, Color: cell.NewColorRGB(255, 100, 50)},
    },
    Direction:  widgets.BarVertical,
    BarWidth:   4,
    BarGap:     2,
    ShowValues: true,
}
frame.RenderWidget(chart, area)
```

### `widgets.LineChart`
Multi-series line chart with smooth Braille subpixel rendering, labeled axes, and legends.

```go
lineChart := widgets.LineChart{
    Datasets: []widgets.LineDataset{
        {
            Name:  "Network In",
            Data:  []float64{10, 25, 40, 65, 80, 95},
            Color: cell.NewColorRGB(46, 204, 113),
        },
    },
    ShowAxes:   true,
    ShowLegend: true,
    XLabels:    []string{"00:00", "04:00", "08:00", "12:00"},
}
frame.RenderWidget(lineChart, area)
```

### `widgets.PieChart`
Subpixel Braille-rendered pie and donut charts with percentage calculations and color legends.

```go
pie := widgets.PieChart{
    Data: []widgets.PieSlice{
        {Label: "Go", Value: 50, Color: cell.NewColorRGB(0, 200, 255)},
        {Label: "Rust", Value: 30, Color: cell.NewColorRGB(255, 100, 50)},
        {Label: "TS", Value: 20, Color: cell.NewColorRGB(50, 150, 255)},
    },
    DonutHoleRatio:  0.4,
    ShowLegend:      true,
    ShowPercentages: true,
}
frame.RenderWidget(pie, area)
```

---

## 7. Developer Tooling & Live Inspector

### `widgets.DevTools` (`F12`)
In-terminal live debugging dashboard displaying live FPS, frame render latency (µs/ms), heap memory, GC stats, goroutines, and bounding box inspection.

```go
devState := widgets.NewDevToolsState()

// In event handler:
if ev.Type == backend.KeyF12 {
    devState.Toggle()
}

// In draw loop:
devState.RecordFrame(time.Since(frameStart))
if devState.Enabled {
    widgets.DevTools{State: devState}.Draw(ctx, buf)
}
```

