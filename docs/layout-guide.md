# 📐 Layout Engine Guide: Flexbox & CSS Grid

Limoni incorporates a powerful, zero-allocation layout engine inspired by modern CSS Flexbox, CSS Grid, and Ratatui's constraint-based geometry system.

---

## 1. Quick High-Level Splitters

For the majority of common TUI patterns, Limoni provides ergonomic high-level helpers:

```go
import "github.com/thebanri/limoni"

// 1. Vertical Splits (e.g. Header, Main Body, Footer)
rows := limoni.SplitVertical(area,
    limoni.Fixed(3), // 3 lines header
    limoni.Fill(),   // Fills all remaining vertical space
    limoni.Fixed(1), // 1 line status bar
)
headerArea := rows[0]
bodyArea   := rows[1]
footerArea := rows[2]

// 2. Horizontal Splits (e.g. Sidebar and Main Content)
cols := limoni.SplitHorizontal(bodyArea,
    limoni.Percentage(25), // 25% width
    limoni.Percentage(75), // 75% width
)
sidebarArea := cols[0]
contentArea := cols[1]

// 3. Proportional Weighted Splits
cards := limoni.SplitHorizontal(contentArea,
    limoni.Ratio(1),
    limoni.Ratio(2), // Takes twice as much space as Ratio(1)
)
```

---

## 2. Flexbox Engine (`layout.FlexLayout`)

When you need granular control over gaps, margins, and directional flow, use `FlexLayout`:

```go
chunks := layout.NewFlexLayout(
    layout.Vertical, // or layout.Horizontal
    1,               // Gap in cells between elements
    layout.Fixed(4),
    layout.Fill(),
    layout.Fixed(2),
).Split(area)
```

### Constraint Types

| Constraint | Description | Example |
| :--- | :--- | :--- |
| **`Fixed(N)`** | Allocates exactly $N$ terminal cells. | `limoni.Fixed(3)` (Title bar) |
| **`Percentage(P)`** | Allocates $P\%$ of the available dimension (0–100). | `limoni.Percentage(30)` (Sidebar) |
| **`Ratio(R)`** | Distributes remaining free space proportionally according to weights. | `limoni.Ratio(2)`, `limoni.Ratio(1)` ($2/3$ and $1/3$) |
| **`Fill()`** | Fills all remaining space (alias for `Ratio(1)`). | `limoni.Fill()` (Main content canvas) |
| **`Min(N)`** | Guarantees at least $N$ cells. | `limoni.Min(15)` |
| **`Max(N)`** | Caps the dimension at $N$ cells. | `limoni.Max(40)` |
| **`FitContent()`** | Measures the child widget's `SizeHint` and allocates exact needed space. | `limoni.FitContent()` |

---

## 3. CSS Grid Engine (`layout.GridLayout`)

Limoni features a 2D CSS Grid engine allowing multi-row, multi-column grid layouts with precise track sizing:

```go
grid := layout.NewGridLayout(
    area,
    // Columns: 20 cells, fill remaining, 25% of total
    []layout.Constraint{layout.Fixed(20), layout.Fill(), layout.Percentage(25)},
    // Rows: 3 cells, fill remaining, 5 cells
    []layout.Constraint{layout.Fixed(3), layout.Fill(), layout.Fixed(5)},
)

// Access individual grid cells (row, col)
topLeft     := grid.Cell(0, 0)
centerArea  := grid.Cell(1, 1)
bottomRight := grid.Cell(2, 2)

// Span across multiple rows or columns
bannerArea  := grid.Area(0, 0, 1, 3) // Row 0, Col 0, RowSpan 1, ColSpan 3
```

---

## 4. Spacing: Insets, Padding & Margins

Widgets like `Block` support CSS-like insets for padding (inner spacing) and margins (outer spacing):

```go
block := limoni.NewBlock().
    WithTitle(" Card ").
    Rounded().
    WithPadding(1, 2, 1, 2) // Top, Right, Bottom, Left

// Retrieve the inner printable area after borders and padding:
innerArea := block.Inner(outerArea)
```

---

## 5. Responsive Nested Layout Example

Combining horizontal and vertical splits for a full-featured developer dashboard:

```go
func drawDashboard(f *limoni.Frame) {
    screen := f.Area()

    // Root: Top bar (3 lines), Content (Fill), Status bar (1 line)
    mainRows := limoni.SplitVertical(screen, limoni.Fixed(3), limoni.Fill(), limoni.Fixed(1))

    // Header
    f.RenderWidget(limoni.NewBlock().
        WithTitle(" 🚀 CLOUD OPERATIONS CONSOLE ").
        WithTitleAlign(limoni.AlignCenter).
        WithBorderStyle(limoni.Fg(limoni.Hex("#00E5FF"))), mainRows[0])

    // Body: Sidebar (20%), Middle Workspace (55%), Telemetry Panel (25%)
    bodyCols := limoni.SplitHorizontal(mainRows[1],
        limoni.Percentage(20),
        limoni.Percentage(55),
        limoni.Percentage(25),
    )

    f.RenderWidget(limoni.NewBlock().Rounded().WithTitle(" Services "), bodyCols[0])
    f.RenderWidget(limoni.NewBlock().Rounded().WithTitle(" Main Workload "), bodyCols[1])
    f.RenderWidget(limoni.NewBlock().Rounded().WithTitle(" Telemetry "), bodyCols[2])

    // Footer
    f.RenderWidget(limoni.NewParagraph(" [q] Quit  [tab] Focus  [?] Help").
        WithStyle(limoni.Fg(limoni.RGB(140, 150, 165))), mainRows[2])
}
```

---

## 6. Composable Lego-Like Component Architecture (VStack, HStack & Decorators)

Limoni provides a modern, composable, Lego-style component layer in the root package (`github.com/thebanri/limoni`) and `component` package (`github.com/thebanri/limoni/component`).

Instead of calculating manual slice splits and passing rectangles by hand, you declare your UI as a nested tree of lightweight components. The layout solver executes on the stack with **zero heap allocations (`0 B/op, 0 allocs/op`)** on hot rendering paths.

### Core Primitives & Decorators

| Function | Description | Example |
| :--- | :--- | :--- |
| **`limoni.VStack(children...)`** | Linear top-to-bottom stack layout (`.WithGap`, `.WithJustify`, `.WithAlignItems`) | `limoni.VStack(header, body, footer)` |
| **`limoni.HStack(children...)`** | Linear left-to-right stack layout (`.WithGap`, `.WithJustify`, `.WithAlignItems`) | `limoni.HStack(sidebar, content)` |
| **`limoni.ZStack(children...)`** | Depth-axis container rendering background to foreground with zero offscreen buffers | `limoni.ZStack(background, modal)` |
| **`limoni.When(cond, then, [else])`** | Declarative conditional rendering with zero runtime overhead | `limoni.When(showHelp, helpView)` |
| **`limoni.Match(val, cases, [default])`** | Declarative pattern matching component selector | `limoni.Match(state, cases, fallback)` |
| **`limoni.WithStyle(style, child)`** | Cascades Style into component subtree via Context copy | `limoni.WithStyle(style, content)` |
| **`limoni.WithForeground(fg, child)`** | Cascades foreground color override into component subtree | `limoni.WithForeground(color, content)` |
| **`limoni.WithBackground(bg, child)`** | Cascades background color override into component subtree | `limoni.WithBackground(color, content)` |
| **`limoni.OnClick(child, handler)`** | Wraps component with mouse click handler | `limoni.OnClick(button, handleClick)` |
| **`limoni.Flex(weight, child)`** | Proportional flex expansion inside a stack | `limoni.Flex(2, mainContent)` |
| **`limoni.FixedSize(w, h, child)`** | Enforces fixed dimensions on a component | `limoni.FixedSize(0, 3, header)` |
| **`limoni.Border(child, symbols, style)`** | Wraps any component with border lines | `limoni.Border(child, widgets.SymbolsRounded, style)` |
| **`limoni.Pad(child, t, r, b, l)`** | Adds custom inner padding | `limoni.Pad(child, 1, 2, 1, 2)` |
| **`limoni.PadAll(child, padding)`** | Adds uniform padding on all 4 sides | `limoni.PadAll(child, 1)` |
| **`limoni.Center(child)`** | Centers a component horizontally & vertically | `limoni.Center(limoni.Label("Centered"))` |
| **`limoni.AlignComponent(child, h, v)`** | Aligns component (Left/Center/Right, Top/Middle/Bottom) | `limoni.AlignComponent(child, limoni.AlignRight, limoni.AlignTop)` |
| **`limoni.Label(text, style...)`** | Ultra-lightweight inline text component | `limoni.Label("Status: OK", limoni.Bold())` |
| **`limoni.AsComponent(widget)`** | Adapts any existing `widgets.Widget` into a composable `Component` | `limoni.AsComponent(table)` |

### Declarative Example

```go
view := limoni.VStack(
    // Fixed height header
    limoni.FixedSize(0, 3, limoni.Border(
        limoni.Center(limoni.Label("🚀 CLOUD METRICS DASHBOARD", limoni.Bold().WithFg(limoni.Hex("#00FFAA")))),
        widgets.SymbolsRounded,
        limoni.Fg(limoni.Hex("#00FFAA")),
    )),

    // Flex body split 1:2
    limoni.Flex(1, limoni.HStack(
        limoni.Flex(1, limoni.Border(
            limoni.AsComponent(sidebarList),
            widgets.SymbolsSingle,
            limoni.Fg(limoni.Hex("#FFCC00")),
        )),
        limoni.Flex(2, limoni.Border(
            limoni.AsComponent(mainTable),
            widgets.SymbolsDouble,
            limoni.Fg(limoni.Hex("#3399FF")),
        )),
    )),

    // Fixed height footer
    limoni.FixedSize(0, 3, limoni.Border(
        limoni.Center(limoni.Label("ESC: Exit | 0 B/op Zero Heap Allocations", limoni.Fg(limoni.Hex("#888888")))),
        widgets.SymbolsSingle,
        limoni.Fg(limoni.Hex("#666666")),
    )),
)

// Render directly into the frame area:
f.RenderComponent(view, f.Area())
```
