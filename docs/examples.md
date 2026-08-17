# 🍋 Limoni Examples Guide

Limoni includes a comprehensive collection of production-ready showcase and reference applications under the `examples/` directory. Each example demonstrates real-world architectural patterns, widget compositions, rendering performance, and event handling.

---

## 📂 Complete Examples Matrix

| Directory | Name | Highlights & APIs | Run Command |
| :--- | :--- | :--- | :--- |
| **[`examples/3d_viewer`](../examples/3d_viewer)** | **3D Software Rasterizer** | 3D mesh loading (`.obj`, `.stl`, `.ply`), Lambertian / Gouraud depth-buffer shading, Euler rotations, and freeform mouse orbit. | `go run ./examples/3d_viewer` |
| **[`examples/paint`](../examples/paint)** | **Point Paint Studio** | $2 \times 4$ Braille sub-pixel canvas, 10 core HIG colors, custom Hex picker modal, live drag-and-hold shape preview, line/circle/rect/eraser, and 25-step undo. | `go run ./examples/paint` |
| **[`examples/dashboard`](../examples/dashboard)** | **System Telemetry & DevOps** | Multi-engine chart visualizer (Braille line with moving averages, full-height equalizer bars, 2x2 sparkline grid, area fill), live Linux telemetry (`/proc`), and interactive process table. | `go run ./examples/dashboard` |
| **[`examples/table_virtual`](../examples/table_virtual)** | **1M Row Virtual Table** | 1,000,000 log stream records, zero heap allocation viewport virtualization, sub-millisecond scrolling, and live metadata inspector. | `go run ./examples/table_virtual` |
| **[`examples/todo`](../examples/todo)** | **TEA Todo Manager** | The Elm Architecture (`Init`, `Update`, `View`), fuzzy search, status toggles, priority filtering, and keyboard navigation. | `go run ./examples/todo` |
| **[`examples/demo`](../examples/demo)** | **Complete Showcase** | Multi-tab flagship demo featuring 3D models, system gauges, Matrix rain, interactive forms, and command palettes. | `go run ./examples/demo` |
| **[`examples/forms`](../examples/forms)** | **Form Controls & Inputs** | `TextInput`, `TextArea`, `Checkbox`, `RadioGroup`, `Select`, and `Slider` with validation and focus management. | `go run ./examples/forms` |
| **[`examples/layer_demo`](../examples/layer_demo)** | **Layer System & Modals** | Z-indexed rendering layers, overlapping modals (`LayerModal`), tooltips (`LayerTooltip`), and click-outside dismissal. | `go run ./examples/layer_demo` |
| **[`examples/animation`](../examples/animation)** | **Animations & Physics** | Spring physics, color interpolation, and easing curves (`animation.Float`, `animation.Color`) rendered at 60 FPS. | `go run ./examples/animation` |
| **[`examples/custom_widget`](../examples/custom_widget)** | **Custom Widget Creation** | Creating custom widgets with the `Widget` interface (`Draw` and `SizeHint`), featuring an interactive analog speedometer. | `go run ./examples/custom_widget` |
| **[`examples/ssh_server`](../examples/ssh_server)** | **Multi-User SSH Server** | Headless network TUI server rendering isolated 60 FPS ANSI diff streams over SSH connections. | `go run ./examples/ssh_server` |
| **[`examples/wasm`](../examples/wasm)** | **WebAssembly Browser App** | Running Limoni applications inside web browsers compiled to WebAssembly with `xterm.js`. | `go run ./examples/wasm` |

---

## 🔍 Detailed Application Breakdown

### 1. 3D Software Rasterizer (`examples/3d_viewer`)
- **Supported Formats**: `.obj`, `.stl`, `.ply`
- **Shading Modes**:
  - `[4]` Wireframe (Vector Bresenham wireframe)
  - `[5]` Solid Prismatic
  - `[6]` Lambertian Diffuse (Directional lighting & surface normals)
  - `[7]` Gouraud Shading (Barycentric RGB vertex interpolation with Depth Buffer)
  - `[8]` Texture Mapping (PNG diffuse map projection)
- **Controls**: `1-3` (Switch Models), `WASD` / Mouse Drag (Orbit), `+/-` / Mouse Wheel (Zoom), `Space` (Auto-rotate).

---

### 2. Point Paint Studio (`examples/paint`)
- **Sub-Pixel Resolution**: $2 \times 4 = 8$ sub-pixel Braille dots per cell (thousands of virtual pixels).
- **Features**:
  - **10 Core Swatches**: White, Red, Orange, Yellow, Green, Cyan, Blue, Purple, Pink, Dark Gray (`1-9, 0`).
  - **Custom Color Modal (`C`)**: Hex color input with live swatch validation + 24-color rainbow matrix.
  - **Live Shape Preview**: Hold and drag lines, circles, and rectangles with live rubber-band ghost preview.
  - **Tools**: `[B]` Brush, `[E]` Eraser, `[L]` Line, `[O]` Circle, `[R]` Rect, `[Z]` Undo (25 steps), `[K]` Clear.
  - **Brush Sizing**: `[` and `]` to resize brush thickness.

---

### 3. System Telemetry & DevOps Dashboard (`examples/dashboard`)
- **4 Switchable Visualization Engines (`m` key)**:
  1. **Line Chart**: High-resolution Braille curves with rolling moving average dashed line and left Y-axis ticks (`100┼`, `75┼`, `50┼`, `25┼`, `0┴`).
  2. **Bar Spectrum**: Side-by-side full-height equalizer spectrum bars with vertical RGB gradient interpolation and floating peak caps.
  3. **Quad Sparkline Grid**: 2x2 live telemetry cards (CPU, RAM, Disk I/O, Network Rx) with `Min`, `Max`, `Avg`, and `P95` metrics.
  4. **Area Fill**: Sub-pixel shaded area graph.
- **System Telemetry**: Real Linux telemetry sampled directly from `/proc/stat`, `/proc/meminfo`, `/proc/net/dev`, and `syscall.Statfs`.
- **Searchable Process Manager**: Filter by name/PID with column sorting (`Left/Right`) and multi-selection (`Space`).

---

### 4. 1,000,000 Row Virtual Table (`examples/table_virtual`)
- **Data Capacity**: 1,000,000 structured enterprise log records.
- **Performance**: Zero-allocation viewport slicing at 60 FPS. Only the visible ~25 rows are rendered in memory.
- **Controls**: `Up/Down` (Select row), `PgUp/PgDn` (25-row jump), `Home/End` (Jump to top/bottom).
- **Right Inspector**: Detailed log inspection with latency, cluster node, timestamp, and performance counters.

---

### 5. Custom Widget Creation Guide (`examples/custom_widget`)
- Reference implementation demonstrating how to build a custom widget by implementing `widgets.Widget`:
  ```go
  type Widget interface {
      Draw(ctx cell.Context, buf *buffer.Buffer)
      SizeHint(maxArea cell.Rect) (uint16, uint16)
  }
  ```
- Implements an interactive **Analog Speedometer** with dynamic dial needle, Braille arc rendering, and mouse dragging.
