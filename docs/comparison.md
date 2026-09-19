# Why Limoni?

| Feature / Goal | 🍋 Limoni (Go) | 🫧 Bubble Tea **v1** + Lip Gloss v1 (Go) | 🌈 Bubble Tea **v2** + Ultraviolet (Go) | 🐀 Ratatui **0.30** (Rust) |
| :--- | :--- | :--- | :--- | :--- |
| **Language & Tooling** | **Go (Native)** | Go (Native) | Go (Native) | Rust (Native) |
| **Render Architecture** | **Flat 1D Grid + Adaptive ANSI Diff** | String concatenation / TEA | Cell buffer + ncurses-style diff | Immediate Mode Double Buffer |
| **Hot-Path Allocations**| **`0 B/op` (Zero Alloc)** | High heap allocation overhead | Reduced; not a zero-alloc design goal — Ultraviolet allocates a `Cell` per glyph | Stack / RAII |
| **Layout Paradigm** | **Declarative Flexbox & Stack Solver** | String slicing (`JoinHorizontal/Vertical`) | Cassowary constraint solver | Constraint solver |
| **Mouse Interaction** | **Spatial Hit-Testing & Z-Index Routing** | None (manual coordinate math) | SGR mouse events; no built-in hit-testing | Manual coordinates |
| **Double Buffering & Diff** | **Sub-microsecond dirty-cell diff + Adaptive flush** | None (entire strings dumped to stdout) | Cell diff + `ECH`/`REP`/`ICH`/`DCH` + scroll optimization | Double-buffered diff |
| **Grapheme Clusters** | **UAX #29 clusters, Unicode 17.0, all 766 official break tests pass** + Mode 2027 request; cursor re-anchored after each cluster for terminals without it | `uniseg` | `uniseg` + Mode 2027 negotiation | `unicode-width` |
| **Capability Detection** | Environment variables only | Environment / terminfo | Runtime queries (no terminfo) | terminfo / crossterm |
| **Large Datasets / Tables**| **Virtual paging (1M rows, ~2.6 ms/frame under continuous scroll)** | High GC load on scroll | Improved vs v1 | Rebuilds every row each frame — `Table` owns its row iterator |
| **3D & Vector Graphics**| **Built-in 3D (OBJ/STL/PLY/GLB) & Gouraud Shaders** | Third-party / custom | Third-party / custom | Addons required |
| **Accessibility (A11y)** | **Screen-reader & semantic tree built-in** | Limited / Manual | Limited / Manual | Experimental |
| **External Dependencies** | **2 (`golang.org/x/sys`, `golang.org/x/crypto`)** | ~15 transitive modules | ~15 transitive modules | crates.io graph |
| **Concurrency Model**  | **Synchronized Model Lifecycle & Event Loops** | Single-threaded TEA loop | Single-threaded TEA loop | Manual thread coordination |

> **On the Bubble Tea v2 column:** the entries are taken from upstream documentation, not from Limoni's own measurements. Charm rebuilt its renderer on [Ultraviolet](https://github.com/charmbracelet/ultraviolet), a cell-based diffing layer, so the architectural gap Limoni originally opened against **v1** does not carry over to **v2** unchanged.
>
> Ultraviolet *is* now measured here, and Limoni is 1.9×–20× faster on the comparable render workloads while emitting far fewer bytes per frame ([§2.4](benchmark-methodology.md#24-ultraviolet)). That is **not** a Bubble Tea v2 result: a v2 program also pays for its runtime, message dispatch and view construction, none of which this measures. No runner in this repository links Bubble Tea v2, so treat any performance claim against v2 itself as unproven.

## 🍋 Limoni Composable (Lego UI) vs. 🎀 Charm Lip Gloss **v1**

While **Lip Gloss v1** popularized styling in Go, its string-concatenation architecture imposed structural limits on interactive, high-frequency applications. The comparison below is against **v1 specifically**:

> ⚠️ **Lip Gloss v2 changes this picture.** v2 is built on [Ultraviolet](https://github.com/charmbracelet/ultraviolet)'s cell buffer rather than raw string concatenation, so the "Data Primitive", "Rendering Pipeline" and "Screen Clipping" rows below no longer describe the current Charm stack. Limoni's remaining structural advantages over v2 are hit-testing, virtual paging, built-in 3D, and the dependency footprint — not string-vs-cell architecture.

| Capability | 🍋 Limoni Composable (`component`) | 🎀 Charm Lip Gloss **v1** |
| :--- | :--- | :--- |
| **Data Primitive** | **16-byte cache-aligned `Cell` struct matrix** | Raw ANSI-escaped strings (`string`) |
| **Hot-Path Allocations** | **`0 B/op` (0 allocs/op)** on layout & render | High allocation rate (~100s of KBs to MBs/sec) |
| **Layout Model** | **True Flexbox & Grid constraint solver** | String slicing (`JoinHorizontal`, `JoinVertical`) |
| **Size Constraints** | **Proportional `Flex`, `Ratio`, `Min`, `Max`** | Fixed manual character widths only |
| **Mouse Hit-Testing** | **Automatic spatial bounds & z-index routing** | None (requires manual coordinate mapping) |
| **Screen Clipping** | **Sub-cell rectangular spatial clipping** | String chopping (causes broken ANSI codes) |
| **Z-Index & Overlays** | **Hardware-like layer stack & modal trapping** | Line-by-line string splicing (`PlaceOverlay`) |
| **Rendering Pipeline** | **Double-buffered ANSI diffing (`~14 µs` sparse, `~50 µs` full-screen)** | Full terminal string dump (causes screen flicker) |
| **Migration Bridge** | **`compat/bubbletea` fluent style builder** | Native Charm ecosystem standard |

### Why Zero-Allocation Architecture Matters:
1. **Eliminating Garbage Collector Stutter**: Lip Gloss v1 computes layouts by allocating intermediate heap strings for every border, padding byte, and horizontal slice. In animated 60 FPS applications, this generates massive heap churn that triggers periodic Go GC pauses (frame stutter). Limoni's component modifiers wrap children on the call stack and write directly into a reusable flat 1D buffer—generating **zero heap allocations (`0 B/op`)**.
2. **Native Interactivity & Hit-Testing**: Because Lipgloss outputs only a flat text string, it cannot determine which component received a mouse click. Limoni components automatically register their physical terminal boundaries (`cell.Rect`), dispatching click, hover, drag, and scroll events directly to callbacks with z-index ordering.

## Key Advantages:
1. **Zero GC Stutter**: Critical rendering loops generate zero heap allocations, eliminating random frame drops during heavy interactions or animations.
2. **True Multithreaded State**: Push state updates from any goroutine safely without bottlenecking the main event loop.
3. **Virtual Viewport Paging**: Render tables and lists with millions of rows without loading invisible cells into memory.
4. **Batteries-Included**: 3D Wireframe/Lambert/Gouraud rendering, rich markdown parser, physics/easing animations, fuzzy search, and command palettes out-of-the-box.
