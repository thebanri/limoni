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
  <a href="https://golang.org"><img src="https://img.shields.io/badge/go-%3E%3D%201.22-blue?style=flat-square&logo=go" alt="Go Version"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-emerald?style=flat-square" alt="License"></a>
  <a href="#-benchmarks"><img src="https://img.shields.io/badge/allocs-0_B%2Fop-brightgreen?style=flat-square" alt="Zero Allocations"></a>
  <a href="AWESOME.md"><img src="https://img.shields.io/badge/awesome-limoni-gold?style=flat-square&logo=awesomelists" alt="Awesome Limoni"></a>
  <a href="CODE_OF_CONDUCT.md"><img src="https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg?style=flat-square" alt="Code of Conduct"></a>
</p>

<p align="center">
  <strong>Language:</strong>
  <a href="README.md">English</a> •
  <a href="README_TR.md">Türkçe</a>
</p>

<p align="center">
  <a href="#-why-limoni">Why Limoni?</a> •
  <a href="#-key-features">Key Features</a> •
  <a href="#-quick-start">Quick Start</a> •
  <a href="#-documentation">Documentation</a> •
  <a href="#-rich-widget-ecosystem">Widgets</a> •
  <a href="#-benchmarks">Benchmarks</a> •
  <a href="#-examples--showcase-applications">Examples</a> •
  <a href="#-awesome-limoni">Awesome Limoni</a>
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
| **3D & Vector Graphics**| **Built-in 3D (OBJ/STL/PLY) & Shaders** | Third-party / custom | Addons required |
| **Accessibility (A11y)** | **Screen-reader & semantic tree built-in** | Limited / Manual | Experimental |
| **Concurrency Model**  | **Lock-Free Channels / Thread-Safe Buffer Swaps** | Single-threaded TEA loop | Manual thread coordination |

### Key Advantages:
1. **Zero GC Stutter**: Critical rendering loops generate zero heap allocations, eliminating random frame drops during heavy interactions or animations.
2. **True Multithreaded State**: Push state updates from any goroutine safely without bottlenecking the main event loop.
3. **Virtual Viewport Paging**: Render tables and lists with millions of rows without loading invisible cells into memory.
4. **Batteries-Included**: 3D Wireframe/Lambert/Gouraud rendering, rich markdown parser, physics/easing animations, fuzzy search, and command palettes out-of-the-box.

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
	"os"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/runtime"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/layout"
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
		switch msg.Key.Type {
		case backend.KeyEsc:
			return runtime.UpdateResult{Quit: true}
		case backend.KeyRune:
			switch msg.Key.Ch {
			case 'q', 'Q':
				return runtime.UpdateResult{Quit: true}
			case '+', '=':
				m.count++
				return runtime.UpdateResult{Redraw: true}
			case '-', '_':
				m.count--
				return runtime.UpdateResult{Redraw: true}
			}
		}
	}
	return runtime.UpdateResult{}
}

func (m *AppModel) View(frame *terminal.Frame) {
	area := frame.Area()
	chunks := layout.FlexLayout{
		Direction: layout.Vertical,
		Constraints: []layout.Constraint{
			layout.Fixed(3),
			layout.Fill(),
			layout.Fixed(3),
		},
	}.Split(area)

	// Header
	frame.RenderWidget(widgets.Block{
		Title:       " 🍋 Limoni Quickstart ",
		BorderStyle: cell.Style{Fg: cell.NewColorRGB(255, 215, 0)},
		TitleStyle:  cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Modifier: cell.ModifierBold},
	}, chunks[0])

	// Body
	text := fmt.Sprintf("Counter: %d  (Press '+' / '-' to change, 'q' to quit)", m.count)
	p := &widgets.Paragraph{
		Text:  text,
		Style: cell.Style{Fg: cell.NewColorRGB(0, 255, 200), Modifier: cell.ModifierBold},
	}
	frame.RenderWidget(p, chunks[1])

	// Footer
	frame.RenderWidget(widgets.Block{
		Title:       " [+] Increment  [-] Decrement  [Q/Esc] Quit ",
		BorderStyle: cell.Style{Fg: cell.NewColorRGB(100, 110, 120)},
	}, chunks[2])
}

func main() {
	b := backend.NewBackend(os.Stdin, os.Stdout)
	if err := b.Setup(); err != nil {
		fmt.Fprintf(os.Stderr, "Setup failed: %v\n", err)
		os.Exit(1)
	}
	defer b.Close()

	term, err := terminal.New(b)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Terminal failed: %v\n", err)
		os.Exit(1)
	}

	app := runtime.New(
		runtime.WithModel(&AppModel{}),
		runtime.WithFPS(60),
	)

	if err := app.RunTerminal(context.Background(), term, b); err != nil {
		panic(err)
	}
}
```

---

## 📚 Documentation

Detailed guides and API references are available in the [`docs/`](./docs) directory and on our [**Interactive Documentation Website**](./docs/site):

| Guide | Description |
| :--- | :--- |
| **[⚡ Getting Started](./docs/getting-started.md)** | Step-by-step introduction, installation, and first interactive app. |
| **[🏛️ Architecture & Zero-Alloc Deep Dive](./docs/architecture.md)** | Memory layout, 1D contiguous grid, and cache locality. |
| **[⚙️ Core API Reference](./docs/core-api.md)** | `cell`, `buffer`, `terminal`, `backend`, and `runtime` packages. |
| **[📐 Flexbox Layout Engine](./docs/layout-guide.md)** | Multi-column, multi-row, percentage, ratio, and constraint layouts. |
| **[🧩 Widget Reference & Guide](./docs/widgets-reference.md)** | Full reference for all display, input, and modal widgets. |
| **[🎨 2D/3D Graphics & Canvas](./docs/graphics-and-canvas.md)** | Braille canvas, 3D Mesh loaders, Lambert/Gouraud shaders, and image protocols. |
| **[🎬 Animation & Physics](./docs/animation-and-physics.md)** | Interpolation, spring physics, and smooth easing curves. |
| **[♿ Accessibility & Theming](./docs/accessibility-and-theming.md)** | Screen-readers, High-Contrast mode, and `NO_COLOR` standard. |
| **[🌐 Drivers & WebAssembly](./docs/drivers-and-platforms.md)** | Cross-platform details: Linux, macOS, Windows VT100, WASM, and SSH. |
| **[📂 Examples Directory Guide](./docs/examples.md)** | Feature map and run instructions for all 12 example applications. |

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

## 📂 Examples & Showcase Applications

Explore runnable demo applications inside the [`examples/`](./examples) directory. See the full [**Examples Directory Guide (docs/examples.md)**](./docs/examples.md) for details on all available apps.

| Example | Description | Run Command |
| :--- | :--- | :--- |
| **[`3d_viewer`](./examples/3d_viewer)** | Professional 3D model viewer supporting `.obj`, `.stl`, `.ply`, texture mapping & Lambertian/Gouraud shaders. | `go run ./examples/3d_viewer` |
| **[`todo`](./examples/todo)** | Full-featured TEA Todo app with tags, priorities, filters, fuzzy search, and progress bars. | `go run ./examples/todo` |
| **[`dashboard`](./examples/dashboard)** | DevOps monitoring dashboard with CPU/Memory sparklines, live process table & streaming logs. | `go run ./examples/dashboard` |
| **[`table_virtual`](./examples/table_virtual)** | 1,000,000 row virtual table showcasing `0 B/op` zero-allocation 120 FPS streaming. | `go run ./examples/table_virtual` |
| **[`colors_and_styles`](./examples/colors_and_styles)** | 24-bit TrueColor gradients, 256-color ANSI palettes, text modifiers & A11y themes. | `go run ./examples/colors_and_styles` |
| **[`ssh_server`](./examples/ssh_server)** | Remote terminal server streaming interactive 60 FPS Limoni sessions over network/SSH sockets. | `go run ./examples/ssh_server` |
| **[`custom_widget`](./examples/custom_widget)** | Developer guide for implementing custom `widgets.Widget` components (Analog Meter / Gauge). | `go run ./examples/custom_widget` |
| **[`demo`](./examples/demo)** | Comprehensive showcase with 3D graphics, matrix rain, tabs, and command palettes. | `go run ./examples/demo` |
| **[`wasm`](./examples/wasm)** | In-browser WebAssembly demo running on xterm.js. | `go run ./examples/wasm` |
| **[`animation`](./examples/animation)** | Physics-based animations, color transitions, and easing curves. | `go run ./examples/animation` |
| **[`forms`](./examples/forms)** | Text inputs, text areas, radios, checkboxes, and sliders. | `go run ./examples/forms` |
| **[`layer_demo`](./examples/layer_demo)** | Layered modals, popups, and focus isolation. | `go run ./examples/layer_demo` |

---

## 🌟 Awesome Limoni

Check out our curated list of real-world apps, tools, and third-party widgets in [**AWESOME.md**](./AWESOME.md).

> Built something cool with Limoni? Open a Pull Request and add your project to [AWESOME.md](./AWESOME.md)!

---

## 🤝 Contributing

Contributions, issues, and feature requests are welcome!
Please make sure to review our [**Code of Conduct**](./CODE_OF_CONDUCT.md) before participating.

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

---

## 🛡️ Security Policy

Please read our [**Security Policy**](./SECURITY.md) to report vulnerabilities responsibly.

---

## 📜 Code of Conduct

This project adheres to the [**Contributor Covenant v2.1**](./CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

---

## 📄 License

Distributed under the **MIT License**. See `LICENSE` for more information.

<p align="center">
  Made with 🍋 by <a href="https://github.com/thebanri">thebanri</a> and contributors.
</p>
