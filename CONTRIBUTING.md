# Contributing to Limoni

Thank you for your interest in contributing to **Limoni**! Limoni is a high-performance, next-generation TUI (Terminal User Interface) engine for Go that combines the speed and immediate-mode architecture of Rust's Ratatui with Go's native concurrency primitives.

This guide outlines our development workflow, architectural constraints, model size and memory boundaries, and testing procedures.

---

## Table of Contents

1. [Code of Conduct & Philosophy](#code-of-conduct--philosophy)
2. [Getting Started & Development Setup](#getting-started--development-setup)
3. [Contribution Workflow (Git & PRs)](#contribution-workflow-git--prs)
4. [Model Size & Memory Invariant Limits](#model-size--memory-invariant-limits)
   - [3D Graphics Models (OBJ, STL, PLY)](#1-3d-graphics-models-obj-stl-ply)
   - [Core Memory Model & Struct Size Limits](#2-core-memory-model--struct-size-limits)
   - [Repository Asset Limits](#3-repository-asset-limits)
5. [Performance Standards & Zero-Allocation Invariants](#performance-standards--zero-allocation-invariants)
6. [Testing & Quality Verification](#testing--quality-verification)
   - [Running Unit Tests](#running-unit-tests)
   - [Running Static Analysis](#running-static-analysis)
   - [Running Data Race Detection](#running-data-race-detection)
   - [Verifying Zero-Allocation Hot-Path Budgets](#verifying-zero-allocation-hot-path-budgets)
   - [Building All Examples and WASM](#building-all-examples-and-wasm)
   - [TestKit & Deterministic Snapshots](#testkit--deterministic-snapshots)

---

## Code of Conduct & Philosophy

Limoni is built around several non-negotiable architectural tenets:

- **Zero Dynamic Heap Allocations During Drawing**: The draw loop and diff engines must not allocate on the heap.
- **Pure Go & Cross-Platform**: No CGO dependencies. Native OS termios/ioctl for Linux/macOS, ConPTY for Windows, and browser WASM support.
- **Decoupled Architecture**: Widgets must never create circular dependencies back into driver or backend packages.
- **Predictable, Deterministic Behavior**: State, mouse hit-testing, and rendering must be testable through deterministic in-memory harnesses (`testkit`).

---

## Getting Started & Development Setup

### Prerequisites

- **Go**: Version `1.22` or later.
- **OS**: Linux, macOS, or Windows.
- **Terminal Emulator**: A terminal supporting TrueColor (24-bit ANSI) is recommended (Ghostty, Kitty, WezTerm, Alacritty, iTerm2, Windows Terminal).

### Clone and Verify Setup

```bash
git clone https://github.com/thebanri/limoni.git
cd limoni

# Verify the test suite
go test ./...
go vet ./...
```

---

## Contribution Workflow (Git & PRs)

1. **Fork the Repository**: Create a personal fork on GitHub.
2. **Create a Feature Branch**:
   ```bash
   git checkout -b feat/my-new-feature
   # or: fix/issue-description, perf/alloc-reduction, docs/guide-update
   ```
3. **Write Clean, Idiomatic Go**:
   - Follow standard Go naming conventions and formatting (`gofmt` / `goimports`).
   - Keep comments clear and preserve existing API documentation.
   - Implement fluent builder patterns (`With...`) where appropriate for widget configuration.
4. **Commit with Conventional Commits**:
   - Format: `<type>(<scope>): <short summary>`
   - Examples:
     - `feat(driver): add Kitty keyboard protocol CSI u parsing`
     - `fix(textinput): resolve horizontal scroll clipping on wide runes`
     - `perf(buffer): eliminate slice allocation during diff scan`
     - `docs: update CONTRIBUTING.md with model size limits`
5. **Run the Full Test Suite** (see [Testing & Quality Verification](#testing--quality-verification)).
6. **Open a Pull Request**: Provide a clear description of the change, benchmarks (if modifying hot paths), and link any related issues.

---

## Model Size & Memory Invariant Limits

When contributing components, 3D graphics, or state models, you must respect the following boundaries:

### 1. 3D Graphics Models (OBJ, STL, PLY)

Limoni features native 3D vector graphics and software z-buffer rasterization (`graphics/` and `widgets/vector_depth.go`):

| Parameter | Limit / Recommendation | Rationale |
| :--- | :--- | :--- |
| **Parser Hard Limit** | **10,000,000 vertices / faces** | Hard cap in PLY/OBJ/STL parsers to prevent memory exhaustion (OOM) and malicious payload DOS attacks. |
| **Real-Time Render Budget** | **Up to 5,000 – 50,000 triangles** | Recommended for smooth **60 FPS to 240 FPS** terminal rendering on a single CPU core without GPU acceleration. |
| **Coordinate Normalization** | Must call `Model3D.Normalize(size)` | Arbitrary scale models must be normalized to viewport coordinates (typically `1.0` to `2.0` units). |
| **Depth Buffer Precision** | 32-bit float Z-buffer | Interpolated depth testing ensures correct occlusion independent of face draw order. |

### 2. Core Memory Model & Struct Size Limits

Limoni's rendering speed is powered by strict CPU cache alignment. **Do not modify the memory layout of core structs without architectural review:**

- **`cell.Style` is strictly 12 bytes**:
  ```go
  type Style struct {
      Fg       Color    // 4 bytes (uint32)
      Bg       Color    // 4 bytes (uint32)
      Modifier Modifier // 2 bytes (uint16)
      _        uint16   // 2 bytes explicit padding
  }
  ```
- **`cell.Cell` is strictly 16 bytes**:
  ```go
  type Cell struct {
      Content rune  // 4 bytes (int32)
      Style   Style // 12 bytes
  }
  ```
  *Why*: Exactly four cells fit into a single 64-byte L1 CPU cache line. Adding fields to `Cell` degrades matrix traversal throughput across thousands of cells.
- **Stack-Allocated `cell.Context`**:
  `cell.Context` must be passed by value on the stack to nested widgets. Do not store pointers to `cell.Context` that trigger heap escape.
- **State Models (Elm / Runtime Architecture)**:
  In `core/engine` (`Model`, `Update`, `View`) and `compat/bubbletea`, state structs should be kept lightweight and avoid retaining unbounded slice allocations across frame ticks.

### 3. Repository Asset Limits

- **No Large Binary Models in Git**: Sample 3D models committed to the repository (under `examples/`) must be low-poly demonstrations (e.g. `cube.obj`, `deniz_topu.obj`), ideally under **500 KB**.
- **Media Assets**: GIFs and screenshots in `assets/` must be optimized and compressed (avoid multi-megabyte uncompressed screen captures).

---

## Performance Standards & Zero-Allocation Invariants

All core rendering and layout code must adhere to these performance guarantees:

- **Draw Loop Zero Allocations**: `Widget.Draw(ctx, buf)` must achieve `0 B/op` and `0 allocs/op`.
  - Reuse internal buffers or slices across frames.
  - Avoid creating closures inside per-cell or per-row rendering loops.
  - Use `strconv.AppendInt` / byte buffers rather than `fmt.Sprintf` in hot paths.
- **Diff Engine Zero Allocations**: Double-buffered diffing between `frontBuf` and `backBuf` must not allocate dynamic memory.
- **Empty Frame Short-Circuit**: If no buffer modifications occur between frames, `Terminal.Draw` must complete in `< 100 ns` (current baseline: ~11 ns).

---

## Testing & Quality Verification

Before submitting any Pull Request, ensure that all automated checks pass locally.

### Running Unit Tests

Run all unit tests across the entire repository:

```bash
go test ./...
```

To run tests without the Go test cache:

```bash
go test -count=1 ./...
```

### Running Static Analysis

```bash
go vet ./...
```

### Running Data Race Detection

Limoni is heavily concurrent (event loops, tickers, render frames). The race detector must pass without any warnings:

```bash
go test -race . ./component ./core/engine ./core/terminal ./testkit ./widgets ./layout ./core/accessibility ./core/driver ./compat/bubbletea
```

### Verifying Zero-Allocation Hot-Path Budgets

Our CI enforces strict allocation budgets via benchmarks. You can run the exact verification gate locally:

```bash
set -euo pipefail
go test ./core/buffer -run '^$' -bench 'BenchmarkDiff_' -benchmem
go test ./widgets -run '^$' -bench 'BenchmarkBlockDraw|BenchmarkParagraphDraw|BenchmarkTableDraw|BenchmarkTableVisibleRows' -benchmem
go test ./benchmarks -run '^$' -bench 'BenchmarkEmptyFrame|BenchmarkMouseHitTest|BenchmarkHundredLayers|BenchmarkTenThousandRowTable' -benchmem
```

*Budget requirements*:
- `BenchmarkEmptyFrame`: 0 allocs/op, latency `< 100 ns/op`.
- `BenchmarkMouseHitTest`: 0 allocs/op.
- `BenchmarkHundredLayers`: 0 allocs/op.
- `BenchmarkBlockDraw` & `BenchmarkParagraphDraw`: 0 allocs/op.
- `BenchmarkTenThousandRowTable`: `<= 16 allocs/op`.

### Building All Examples and WASM

Ensure all example applications and the WebAssembly target build without errors:

```bash
mkdir -p /tmp/limoni-test-build
go build -o /tmp/limoni-test-build/limoni-cli ./cmd/limoni
go build -o /tmp/limoni-test-build/simple ./examples/simple
go build -o /tmp/limoni-test-build/showcase ./examples/showcase
go build -o /tmp/limoni-test-build/demo ./examples/demo
go build -o /tmp/limoni-test-build/anim ./examples/animation
go build -o /tmp/limoni-test-build/ascii3d ./examples/ascii3d
go build -o /tmp/limoni-test-build/charts ./examples/charts
go build -o /tmp/limoni-test-build/forms ./examples/forms
go build -o /tmp/limoni-test-build/layer_demo ./examples/layer_demo
go build -o /tmp/limoni-test-build/3d_viewer ./examples/3d_viewer
go build -o /tmp/limoni-test-build/paint ./examples/paint
go build -o /tmp/limoni-test-build/toast ./examples/toast
go build -o /tmp/limoni-test-build/todo ./examples/todo
go build -o /tmp/limoni-test-build/treeview ./examples/treeview
go build -o /tmp/limoni-test-build/dashboard ./examples/dashboard
go build -o /tmp/limoni-test-build/table_virtual ./examples/table_virtual
go build -o /tmp/limoni-test-build/ssh_server ./examples/ssh_server
go build -o /tmp/limoni-test-build/custom_widget ./examples/custom_widget
go build -o /tmp/limoni-test-build/composable ./examples/composable
GOOS=js GOARCH=wasm go build -o /tmp/limoni-test-build/limoni.wasm ./examples/wasm
rm -rf /tmp/limoni-test-build
```

### TestKit & Deterministic Snapshots

When writing tests for new widgets or layouts, use Limoni's deterministic `testkit`:

```go
package mywidget_test

import (
    "testing"
    "github.com/thebanri/limoni/testkit"
    "github.com/thebanri/limoni/widgets"
)

func TestMyWidgetSnapshot(t *testing.T) {
    term := testkit.NewTerminal(40, 10)
    widget := widgets.NewBlock().WithTitle("Test")
    
    term.DrawWidget(widget)
    term.AssertContains(t, "Test")
    term.AssertGolden(t, "testdata/my_widget.golden")
}
```

---

## Questions & Discussions

- If you encounter bugs or want to propose an enhancement, please open a GitHub Issue.
- Join the discussions and help make terminal user interfaces in Go faster, lighter, and more beautiful!
