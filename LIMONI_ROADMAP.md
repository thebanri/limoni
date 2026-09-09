# Limoni — Strategic Roadmap & Architecture Plan

## Implementation Status — 2026-08-17 (v0.1.0 Ready)

All roadmap phases and core systems engineering targets are fully implemented and verified:

- **TestKit & Benchmarks:** `testkit` provides fixed-size terminal emulation, text/style snapshots, golden file matching, resize, mouse click/drag, focus, event propagation, hover, and accessibility tree assertions. Benchmarked against Ratatui and Bubble Tea across 12 standardized cross-implementation workloads.
- **Zero-Allocation Hot Path:** Achieved `0 B/op, 0 allocs/op` on render hot paths: `EmptyFrame`, `TextHeavyFrame`, `MouseHitTest`, `HundredLayers`, and `TenThousandRowTable`.
- **Runtime Engine:** `core/engine` provides `Msg`, `Cmd`, `Model`, cancellation, deterministic command ordering, panic recovery, redraw coalescing, and graceful shutdown.
- **Typed Input & Interaction:** `core/engine/input.go` translates raw driver events into typed messages; `core/terminal` provides event regions with metadata, disabled zones, capture/target/bubble propagation, hover tracking, and double-click detection.
- **Layout Negotiation:** `layout/measure.go` provides min/ideal/max sizing, overflow policies, measure/arrange passes, responsive breakpoints, and cross-axis alignment.
- **Virtual Data Runtime:** `widgets/virtual_data.go` and `widgets/virtual_data_view.go` provide stable row IDs, asynchronous data providers, viewport prefetching, and support for 1,000,000+ virtual rows.
- **Accessibility:** `core/accessibility` provides semantic tree construction, role/state modeling, high-contrast mode, NO_COLOR compliance, ASCII mode, reduced-motion handling, and line-oriented screen reader output.
- **Cross-Platform Native Drivers:** Linux, macOS (Darwin `TIOCGETA`/`TIOCSETA`), BSD (`TIOCGETA`), Windows (ConPTY / VT100 Virtual Terminal), WebAssembly (`js/wasm` xterm.js bridge), and Remote SSH PTY drivers are complete.
- **3D Graphics & Shaders:** OBJ/STL/PLY mesh rasterizer, surface normal computation, Lambertian directional lighting, and Gouraud barycentric color interpolation (`widgets/vector_depth.go`, `graphics/shading.go`) are implemented.

The widget showcase in `examples/demo` is fully functional; modern engine, accessibility, layout, and benchmark APIs are thoroughly covered by unit and race tests.

> This document serves as the master architectural reference for Limoni.
> The goal is not to merely clone Ratatui or Bubble Tea, but to take their strongest concepts and unite them with Limoni's differential renderer, interaction system, layout negotiation, graphics pipeline, and Go concurrency model into a unified, zero-allocation runtime.

## 1. Product Vision

Limoni is positioned as:

> **An event-aware, layout-negotiating, low-allocation terminal UI runtime for Go.**

Limoni is more than "a Ratatui port in Go." Its defining strength is the seamless cohesion of four tightly integrated layers:

```text
Application Runtime
    Init / Update / Cmd / Msg
            ↓
Interaction Runtime
    input / keybindings / focus / mouse / modal
            ↓
UI Engine
    layout / size negotiation / theme / accessibility
            ↓
Renderer
    cell buffer / diff / ANSI / native graphics / layers
```

## 2. Best-of-Breed Architectural Insights

### From Ratatui

- Immediate-mode cell-based rendering
- Double-buffered differential ANSI emission
- Constraint-driven layout system
- Clear separation between widget descriptors and mutable state
- Deterministic headless UI testing (analogous to `TestBackend`)
- Multi-driver abstraction and capability detection
- Modular package separation: core primitives, widgets, drivers
- Unicode cell-width and fullwidth character accuracy

Capabilities intentionally left to end-user application code in Ratatui are handled out of the box in Limoni:

- Built-in mouse hit-testing
- Event propagation and bubbling
- Focus management and modal sandbox isolation
- Keybinding precedence hierarchies
- Two-pass widget size negotiation
- Semantic accessibility trees
- Built-in Elm Architecture application runtime

### From Bubble Tea

- The Elm Architecture (`Init / Update / View`) mental model
- Asynchronous task isolation via `Cmd / Msg`
- Decoupling timers, HTTP requests, file I/O, and subprocesses from the render loop
- Injectable I/O streams for headless testing
- Terminal lifecycle and raw mode management
- Context cancellation and graceful shutdown
- Ergonomic, production-grade runtime options

Capabilities that Bubble Tea delegates to third-party packages or string concatenation are natively built into Limoni:

- Direct contiguous cell buffer rendering (avoiding string allocations)
- Automatic spatial mouse routing
- Z-index layer compositing
- Native terminal image protocols (Kitty, Sixel, iTerm2)
- 2D Braille Canvas & 3D software rendering
- Two-pass layout negotiation
- Inherited semantic theming
- Accessible semantic node trees

## 3. Current Architecture Status

### Foundational Strengths

- `core/buffer`: Flat 1D cell buffer, front/back diff engine, ANSI output optimizer, snapshots, and microbenchmarks.
- `core/driver`: Raw mode terminal driver, ANSI stream parser, mouse tracking, window resize listener, focus events, and bracketed paste.
- `core/terminal`: `Frame` abstraction, layer/z-index compositing, mouse capture, event propagation, focus scopes, and modal sandboxes.
- `layout`: Flex, grid, fixed, percentage, ratio, and fill constraints.
- `widgets`: Block, text, markdown, list, virtual scrolling list, table, form inputs, popups, sliders, and command palettes.
- `widgets/theme.go`: Semantic color themes, contrast calculation, and WCAG AAA high-contrast modes.
- `graphics`: Kitty, Sixel, iTerm2, HalfBlock fallback, alpha blending, and 2D/3D mesh loaders.
- `animation`: Easing curves, color interpolation, physics transitions, and ordered dithering.
- `examples/demo`: Reference implementation featuring component playground, profiler, keybindings, virtual lists, and graphics.

## 4. Key Architectural Decisions

### 4.1 The Elm Architecture as an Optional, First-Class Engine

The immediate-mode API remains completely valid. Applications can optionally adopt the TEA runtime for predictable state management:

```go
type Msg any

type Cmd func(context.Context) Msg

type Model interface {
    Init() []Cmd
    Update(Msg) UpdateResult
    View(*terminal.Frame)
}
```

Benefits:
- Simple utility scripts can draw directly without boilerplate.
- Complex applications gain structured `Init/Update/View` state management.
- The high-performance cell buffer and differential engine remain identical across both approaches.

### 4.2 Framework-Level Event Routing

Standard Event Dispatch Flow:

```text
Capture Phase
   ↓
Modal / Layer Boundary
   ↓
Target Widget (Spatial Hit Test)
   ↓
Focused Widget
   ↓
Parent Scope
   ↓
Global Application Handlers
```

Supported Event Types:
- Pointer move, enter, and leave
- Mouse press, release, click, and double-click
- Drag start, move, and end
- Mouse wheel scroll
- Key press and release
- Bracketed paste
- Focus and blur
- Window resize

### 4.3 Two-Pass Layout Negotiation

```text
Measure Phase
   ↓
Resolve Constraints
   ↓
Arrange Children
   ↓
Draw Phase
```

Widgets expose layout characteristics via structured measurements:

```go
type Measure struct {
    MinWidth       uint16
    MinHeight      uint16
    IdealWidth     uint16
    IdealHeight    uint16
    MaxWidth       uint16
    MaxHeight      uint16
    ShrinkPriority int
    GrowPriority   int
    Overflow       OverflowPolicy
}
```

### 4.4 Package Modularity: Core Primitives vs. Domain Extras

```text
core/
    cell/
    buffer/
    driver/
    terminal/
    engine/
    accessibility/
layout/
component/
widgets/
graphics/
animation/
testkit/
compat/
    bubbletea/
```

Heavy visual features (3D mesh rasterization, image protocols, complex charts) remain modular and do not burden core runtime primitives.

## 5. Roadmap Phases and Verification

### Status Summary

`[x]` Verified complete and passing CI across Linux, macOS, and Windows.

| Phase | Status | Implemented Scope |
|---|:---:|---|
| Phase 33 — Engine Core | `[x]` | `core/engine`, command scheduler, cancellation, panic recovery, redraw coalescing, terminal event loop. |
| Phase 34 — Typed Input & Events | `[x]` | Key/mouse/wheel/resize/focus/paste message structs and driver stream injection. |
| Phase 35 — Interaction Engine 2.0 | `[x]` | Spatial event regions, capture/target/bubble routing, hover detection, double clicks, modal isolation. |
| Phase 36 — Layout Negotiation | `[x]` | Two-pass measure/arrange, intrinsic content sizing, child aggregation, overflow policies, responsive breakpoints. |
| Phase 37 — Virtualized Data Runtime | `[x]` | Stable row IDs, async providers, viewport prefetching, selection remapping, row recycling, variable height rows. |
| Phase 38 — Theme & Accessibility 2.0 | `[x]` | Frame/TestKit semantic trees, line-mode screen-reader serializers, high-contrast AAA themes, NO_COLOR compliance. |
| Phase 39 — Cross-Platform Drivers | `[x]` | Linux raw-mode TTY, macOS Darwin syscalls, Windows Virtual Terminal, WebAssembly bridge, and SSH PTY streams. |
| Phase 40 — TestKit Harness | `[x]` | Differential cell snapshots, input simulation, focus/layer assertions, golden file comparisons. |
| Phase 41 — Benchmark Laboratory | `[x]` | Automated cross-implementation runner suite (Limoni, Ratatui, Bubble Tea) with JSON reports and HTML dashboards. |
| Phase 42 — Bubble Tea Compatibility | `[x]` | `compat/bubbletea` adapter supporting `tea.Model`, `tea.Cmd`, `tea.Program`, and `lipgloss.Style` transitions. |

## 6. Success Metrics & Verification

### Ergonomics
- Minimal application startup in 30–50 lines of idiomatic Go.
- Zero manual coordinate math required for mouse click routing.
- Modals automatically enforce focus sandboxing and background event suppression.
- Styles and themes cascade predictably down component trees.

### Performance
- Zero heap allocations on static drawing hot paths (`0 B/op, 0 allocs/op`).
- Frame render latency in single-digit microseconds.
- Tables with 1,000,000+ rows query and render only visible viewport slices.
- Emitted ANSI bytes minimized via differential cell hashing.

### Reliability
- Deterministic behavior testing via `testkit.NewTerminal`.
- Automated data race detection (`-race`) and zero-allocation assertions enforced in CI.
- Multi-platform native smoke test matrix running on Linux, macOS, and Windows runners.

### Accessibility
- Interactive widgets generate structured semantic nodes.
- High-contrast and NO_COLOR modes preserve complete UI legibility.
- Reduced-motion settings automatically bypass animations.
- Screen readers receive structured, accessible line-mode output.
