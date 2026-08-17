<p align="center">
  <img src="docs/assets/showcase.png" alt="Limoni Showcase" width="100%" />
</p>

<h1 align="center">🍋 Limoni</h1>

<p align="center">
  <strong>An Ultra-Fast, Zero-Allocation, Thread-Safe Modern TUI Framework for Go.</strong>
</p>

<p align="center">
  <a href="https://github.com/thebanri/limoni/actions"><img src="https://img.shields.io/github/actions/workflow/status/thebanri/limoni/ci.yml?branch=main&style=flat-square&logo=github" alt="Build Status"></a>
  <a href="https://pkg.go.dev/github.com/thebanri/limoni"><img src="https://img.shields.io/badge/go.dev-reference-007d9c?style=flat-square&logo=go&logoColor=white" alt="Go.Dev Reference"></a>
  <a href="https://golang.org"><img src="https://img.shields.io/badge/go-%3E%3D%201.24-blue?style=flat-square&logo=go" alt="Go Version"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-emerald?style=flat-square" alt="License"></a>
  <a href="#-benchmarks"><img src="https://img.shields.io/badge/allocs-0_B%2Fop-brightgreen?style=flat-square" alt="Zero Allocations"></a>
</p>

<p align="center">
  <a href="#-why-limoni">Why Limoni?</a> •
  <a href="#-key-features">Key Features</a> •
  <a href="#-quick-start">Quick Start</a> •
  <a href="#-rich-widget-ecosystem">Widgets</a> •
  <a href="#-architecture">Architecture</a> •
  <a href="#-benchmarks">Benchmarks</a> •
  <a href="#-examples">Examples</a>
</p>

---

## ⚡ Overview

**Limoni** is an enterprise-grade, high-performance Terminal User Interface (TUI) engine for Go. Designed from the ground up for data-intensive dashboards, devtools, and modern terminal applications, Limoni bridges the gap between Go's developer ergonomics and Rust-like raw rendering speed.

By utilizing a **flat 1D cell grid**, **zero-allocation hot-paths**, and a **sub-microsecond differential ANSI engine**, Limoni achieves ultra-smooth 60+ FPS rendering without triggering Go's Garbage Collector.

---

## 💡 Why Limoni?

| Feature / Goal | 🍋 Limoni (Go) | 🫧 Bubble Tea (Go) | 🐀 Ratatui (Rust) |
| :--- | :--- | :--- | :--- |
| **Language & Tooling** | **Go (Native)** | Go (Native) | Rust (Native) |
| **Render Architecture** | **Flat 1D Grid + ANSI Diff Engine** | String concatenation / TEA | Immediate Mode Double Buffer |
| **Hot-Path Allocations**| **`0 B/op` (Zero Alloc)** | High heap allocation overhead | Stack / RAII |
| **Large Datasets / Tables**| **Sub-µs Virtual Paging (Millions of rows)** | High GC load on scroll | High layout cloning overhead |
| **3D & Vector Graphics**| **Built-in 3D (OBJ/STL/PLY) & Canvas** | Third-party / custom | Addons required |
| **Accessibility (A11y)** | **Screen-reader & navigation tree built-in** | Limited / Manual | Experimental |
| **Concurrency Model**  | **Lock-Free Channels / Thread-Safe Buffer Swaps** | Single-threaded TEA loop | Manual thread coordination |

### Key Advantages:
1. **Zero GC Stutter**: Critical rendering loops generate zero heap allocations, eliminating random frame drops during heavy interactions or animations.
2. **True Multithreaded State**: Push state updates from any goroutine safely without bottlenecking the main event loop.
3. **Virtual Viewport Paging**: Render tables and lists with millions of rows without loading invisible cells into memory.
4. **Batteries-Included**: 3D Wireframe rendering, rich markdown parser, physics/easing animations, fuzzy search, and command palettes out-of-the-box.

---

## ✨ Key Features

* 🚀 **Sub-Microsecond ANSI Diffing**: Computes dirty cell regions and emits minimal ANSI escape sequences; short-circuits instantly if nothing changed.
* 📦 **Contiguous 1D Buffer**: Flat memory layout eliminates pointer chasing and maximizes CPU L1/L2 cache locality.
* 🎨 **TrueColor & Fallback Engine**: Full 24-bit RGB TrueColor support with automatic downsampling fallbacks for 256-color and 16-color terminals.
* 📐 **Responsive Flexbox Layouts**: Declarative layout engine supporting proportional splits, minimum/maximum size constraints, and nested alignments.
* 🎬 **Animation & Easing Engine**: Built-in interpolation for float, color, and transitions (Linear, Quad, Cubic, Elastic, Bounce).
* 🕶️ **Native 3D & Vector Graphics**: Render 3D `.obj`, `.stl`, `.ply` meshes directly in terminal cells with camera projection, rotation, and lighting!
* ♿ **Built-in Accessibility**: Accessible navigation tree, line-by-line inspection mode, and semantic annotations for screen-readers.

---

## 🚀 Quick Start

### Installation

```bash
go get github.com/thebanri/limoni
```

### 1. Minimal TEA (The Elm Architecture) Example

```go
package main

import (
	"context"
	"fmt"

	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/runtime"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/widgets"
)

type AppModel struct {
	count int
}

func (m *AppModel) Init() []runtime.Cmd {
	return nil
}

func (m *AppModel) Update(msg runtime.Msg) runtime.UpdateResult {
	switch msg := msg.(type) {
	case runtime.KeyPressMsg:
		if msg.Key.Ch == '+' {
			m.count++
			return runtime.UpdateResult{Redraw: true}
		} else if msg.Key.Ch == '-' {
			m.count--
			return runtime.UpdateResult{Redraw: true}
		} else if msg.Key.Ch == 'q' {
			return runtime.UpdateResult{Quit: true}
		}
	}
	return runtime.UpdateResult{}
}

func (m *AppModel) View(frame *terminal.Frame) {
	area := frame.Area()
	block := widgets.Block{
		Title:       " 🍋 Limoni Quickstart ",
		BorderStyle: cell.Style{Fg: cell.NewColorRGB(255, 215, 0)},
		TitleStyle:  cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Modifier: cell.ModifierBold},
	}
	frame.RenderWidget(block, area)

	inner := block.Inner(area)
	text := fmt.Sprintf("Counter: %d  (Press '+' / '-' to change, 'q' to quit)", m.count)
	p := &widgets.Paragraph{
		Text:  text,
		Style: cell.Style{Fg: cell.NewColorRGB(200, 240, 255)},
	}
	frame.RenderWidget(p, inner)
}

func main() {
	app := runtime.New(
		runtime.WithModel(&AppModel{}),
		runtime.WithFPS(60),
	)

	if err := app.Run(context.Background()); err != nil {
		panic(err)
	}
}
```

---

## 🧩 Rich Widget Ecosystem

Limoni comes with an extensive suite of production-ready widgets:

| Category | Available Widgets |
| :--- | :--- |
| **Structure & Layout** | `Block`, `Dialog / Modal`, `Popup`, `ResponsiveGrid`, `Flexbox` |
| **Data Display** | `Table (Virtual/Paged)`, `List (Virtual)`, `Sparkline`, `ProgressBar`, `RichText` |
| **Input Controls** | `TextInput`, `TextArea`, `Checkbox`, `RadioGroup`, `Select / Dropdown`, `Slider` |
| **Navigation & Search**| `CommandPalette`, `FuzzySearch (FZF-style)`, `Tabs`, `KeybindingManager` |
| **Graphics & 3D** | `Canvas (Braille / Block)`, `Vector3D Mesh (OBJ/STL/PLY)`, `Lambertian & Gouraud Shaders`, `Image (Kitty/Sixel/iTerm2/HalfBlock)` |
| **Text & Docs** | `Markdown (Full GFM)`, `RichText Highlighting` |

---

## 🏛️ Architecture

```
                      ┌────────────────────────────────────────┐
                      │             User Application           │
                      └───────────────────┬────────────────────┘
                                          │ State & Views
                                          ▼
                      ┌────────────────────────────────────────┐
                      │         Declarative UI / Widgets       │
                      │   (Tables, Modals, 3D Canvas, Layout)  │
                      └───────────────────┬────────────────────┘
                                          │ Draw to Grid
                                          ▼
                      ┌────────────────────────────────────────┐
                      │          Flat 1D Buffer Grid           │
                      │  [Zero Heap Allocation Cell Memory]   │
                      └───────────────────┬────────────────────┘
                                          │
                        ┌─────────────────┴─────────────────┐
                        ▼                                   ▼
             ┌─────────────────────┐             ┌─────────────────────┐
             │ Previous Frame Snap │             │ Current Frame Snap  │
             └──────────┬──────────┘             └──────────┬──────────┘
                        └─────────────────┬─────────────────┘
                                          │ Sub-microsecond Diff
                                          ▼
                      ┌────────────────────────────────────────┐
                      │       ANSI Diff & Optimize Stream      │
                      │  (Minimizes cursor jump & color reset) │
                      └───────────────────┬────────────────────┘
                                          │ Direct Write
                                          ▼
                      ┌────────────────────────────────────────┐
                      │   Terminal TTY / Windows / macOS / SSH │
                      └────────────────────────────────────────┘
```

---

## 📊 Benchmarks

Limoni includes a standardized cross-implementation benchmark suite comparing native Go and Rust workloads under identical virtual terminals.

Run benchmarks locally:
```bash
# Run Go Limoni Benchmark
go test ./benchmarks -run '^$' -bench . -benchmem

# Generate HTML Comparison Dashboard
go run ./benchmarks/runners/dashboard -output benchmark-results/dashboard.html benchmark-results/limoni.json benchmark-results/bubbletea.json benchmark-results/ratatui.json
```

*Results from 120x40 standard viewport tests:*
- **Frame Diffing Speed:** `< 0.85 µs` per full-screen diff.
- **Heap Allocations in Hot Path:** `0 allocs/op (0 B/op)`.
- **Virtual Table Scrolling:** `> 120 FPS` continuous rendering with 1,000,000 rows.

---

## 📂 Examples

Explore runnable demo applications inside the [`examples/`](./examples) directory:

- **[`examples/demo`](./examples/demo)**: Feature-rich interactive showcase featuring 3D mesh rendering (OBJ/STL/PLY with Lambertian/Gouraud shaders), live system monitoring, matrix rain, and interactive tabs.
- **[`examples/wasm`](./examples/wasm)**: WebAssembly in-browser TUI running with xterm.js bridge and real-time 3D animation.
- **[`examples/animation`](./examples/animation)**: Smooth 60 FPS transitions and easing demos.
- **[`examples/forms`](./examples/forms)**: Complete input validation, text areas, radios, and sliders.
- **[`examples/layer_demo`](./examples/layer_demo)**: Floating modals, popups, and layered depth buffers.

To run the full interactive showcase:
```bash
go run ./examples/demo
```

---

## 🤝 Contributing

Contributions, issues, and feature requests are welcome! Feel free to check the [issues page](https://github.com/thebanri/limoni/issues) or submit a pull request.

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

---

## 🛡️ License

Distributed under the **MIT License**. See `LICENSE` for more information.

<p align="center">
  Made with 🍋 by <a href="https://github.com/thebanri">thebanri</a> and contributors.
</p>
