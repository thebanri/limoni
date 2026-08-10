---
name: limoni_development
description: Guidelines and principles for developing the Limoni TUI library in Go and teaching Go to the user.
---

# Limoni TUI Development, Architecture, and Handover Guide

This skill file defines the vision, architectural principles, current development status, solved TUI limitations, and future roadmap of the **Limoni** project. It is intended to serve as a complete system handbook for AI agents taking over the project after a break or in a new chat session.

---

## 1. Project Vision and Motivation

Limoni is a next-generation **TUI (Terminal User Interface) engine** that combines the rendering speed and immediate-mode rendering capabilities of Rust's **Ratatui** ecosystem with Go's native concurrency primitives (**goroutines and channels**).

### Limitations We Aim to Overcome in Competing Libraries

1. **Automatic Mouse Click Routing (Mouse Event Hit-Testing)**:
   - *Problem*: In Ratatui and Bubble Tea, developers must manually calculate coordinates to determine which widget received a mouse click.
   - *Solution*: Limoni registers interactive regions during rendering through `Frame.RegisterClickHandler(area, callback)`. `Terminal.RouteMouseEvent(ev)` automatically routes clicks to the correct callback.
2. **An Interactivity Bridge Without Circular Package Dependencies**:
   - *Problem*: If the low-level `cell` package depends on higher-level `terminal` or `backend` event types, circular package dependencies are created.
   - *Solution*: `cell.Context` contains a `RegisterClick func(area Rect, handler func())` function field. `Frame.RenderWidget` connects this bridge to its own router when rendering begins. This keeps the `widgets` package independent.
3. **Layout Negotiation**:
   - *Problem*: In Ratatui, a widget cannot communicate its size requirements to the layout system.
   - *Solution*: The `Widget` interface includes a `SizeHint` API. Components such as `Block`, `Paragraph`, and `List` can negotiate their border, title, and content requirements with the flexible `layout` engine.
4. **Cascading Styles**:
   - *Problem*: In Ratatui, styles must be passed manually to every child component.
   - *Solution*: Parent styles are automatically inherited through `cell.Context`. With `Style.Merge(other)`, a child component overrides only the properties it needs to change.

---

## 2. Current Architecture and Package Organization

The project is organized as modular packages within a single Go module (`github.com/thebanri/limoni`):

```
limoni/
├── go.mod
├── .agents/skills/limoni_development/skill.md  # This handbook
├── core/
│   ├── cell/
│   │   ├── cell.go       # Cell (16 bytes), Style (12 bytes), Color (uint32)
│   │   ├── rect.go       # Screen-area geometry (Rect)
│   │   └── context.go    # Stack-allocated cascading Context and Merge
│   ├── buffer/
│   │   ├── buffer.go     # Flat 1D cell matrix (Buffer), Resize
│   │   └── diff.go       # Zero-allocation double-buffered diff algorithm
│   ├── backend/
│   │   ├── types.go      # Flat event types (Event, KeyEvent, etc.)
│   │   ├── termios_linux # Pure Go raw-mode control through Unix ioctl calls
│   │   ├── parser.go     # Keyboard/mouse/focus/hover ANSI sequence parser
│   │   └── backend.go    # Asynchronous event loop with SIGWINCH and 25 ms ESC timeout
│   └── terminal/
│       ├── frame.go      # Frame drawing context and click-handler registration
│       └── terminal.go   # Terminal manager, draw loop, and mouse router
├── layout/
│   └── layout.go         # Flexbox layout engine (Fixed, Percentage, Ratio, Min, Max, Fill)
├── widgets/
│   ├── widget.go         # Core Widget interface (Draw and SizeHint)
│   ├── block.go          # Bordered, titled, padded Block container
│   ├── paragraph.go      # Multiline word-wrapping text widget
│   └── list.go           # Selectable, automatically scrolling interactive list
└── examples/
    └── demo/main.go      # Demo with mouse-click list selection and hover-to-exit support
```

---

## 3. Performance Guarantees and Standards

The following technical decisions and performance requirements **MUST** be preserved in all newly developed modules:

- **Memory Alignment**:
  - `Style` is fixed at **12 bytes** (`Fg`: 4, `Bg`: 4, `Modifier`: 2 + 2 bytes of padding).
  - `Cell` is fixed at **16 bytes** (`Content`: 4 + `Style`: 12). Cache-friendly alignment must be preserved.
- **Zero Heap Allocations During Drawing**:
  - The draw and diff loops must not perform dynamic memory allocations.
  - The `writeBuf` byte slice inside `Terminal` is reused. Use `strconv.AppendInt` for ANSI encoding.
  - Updating a 120x40 screen (4,800 cells) takes **18.3 μs** (**54,000+ FPS** on a single core).
- **Stack-Allocated Context**:
  - During drawing, `cell.Context` is passed to child widgets by value. This prevents heap escape.

---

## 4. Completed Phases (Phases 1–27)

- **Phase 1: Core Buffer and Diff Engine [COMPLETED]**
  - Matrix alignment, double-buffer diffing, and a zero-allocation ANSI encoder were completed.
- **Phase 2: Backend and OS Terminal Control [COMPLETED]**
  - Linux ioctl termios control without CGO and an event loop with a 25 ms ESC filter were implemented.
- **Phase 3: Flexible Layout Engine [COMPLETED]**
  - A Flexbox layout engine based on ratio, percentage, and fixed-split calculations was implemented.
- **Phase 4: Terminal, Frame, and Block Widget [COMPLETED]**
  - The drawing Frame, Block border styles (rounded, single, etc.), and mouse-click routing were implemented.
- **Phase 5: Rich Widget Set (Paragraph and List) [COMPLETED]**
  - Interactive `Paragraph` (automatic word wrapping) and `List` (selection state and automatic viewport scrolling) widgets were completed.
- **Phase 6: Braille Canvas and Vector Drawing (High-Resolution Graphics) [COMPLETED]**
  - A Braille Canvas widget providing 2x4-pixel resolution per cell and vector drawing functions (lines, circles, rectangles, and Bézier curves) were completed.
- **Phase 7: Media and Image Display Layer (`graphics`) [COMPLETED]**
  - Native image encoders for the Kitty Graphics Protocol, Sixel, iTerm2, and Ghostty were completed for rendering real PNG/JPG images in the terminal. Base64 chunking was added for Kitty's 4,096-byte individual data-packet limit.
  - Universal **Half-Block (`▄`, U+2584)** 1x2-resolution cell-buffer rendering was added for terminals without native protocol support, such as Alacritty. The `Image` widget was completed.
  - A bug that prevented container widgets such as `Block` from forwarding `RegisterClick` and `RegisterImage` callback bridges to child widgets was fixed, stabilizing image rendering and click support inside containers.

- **Phase 8: Animation Engine [COMPLETED]**
  - The `animation` package was implemented with time-based interpolation, smooth transitions, and 16 standard easing functions (Linear, Quad, Cubic, Sine, Expo, Bounce).
  - `Float` animation structures for numeric values and `Color` animation structures for 24-bit TrueColor RGB transitions were added.
  - An animated demo application (`examples/animation`) was added to demonstrate FPS monitoring and 30 FPS drawing with `time.Ticker` without blocking the event loop.

- **Phase 9: Rich Forms and Input Controls [COMPLETED]**
  - The `FocusManager`, which manages dynamic tab-based focus traversal according to draw order, was implemented.
  - Single-line interactive `TextInput` (with state and cursor management), boolean `Checkbox`, and single-selection `RadioButton` components were added.
  - Unit tests (`widgets/form_test.go`) and an integrated form demonstration with dynamic color-theme switching were added to the main demo.

- **Phase 10: Layered Rendering, Modal Windows, and Popups [COMPLETED]**
  - Support for multiple overlapping layers was added through the `Layer` structure (`ID`, `Type`, `Area`, `ZIndex`, `ClickOutside`).
  - `Frame.BeginLayer(id)` / `Frame.EndLayer()` lets widgets identify their layer during drawing.
  - Dynamic layer creation and removal are supported through `Frame.RegisterLayer()` and `Frame.RemoveLayer()`.
  - `RouteMouseEvent()` routes events by z-index: the topmost layer receives the event first, and click-outside handlers are triggered when appropriate.
  - The `Popup` widget supports an opening button, dropdown list, hover selection, disabled items, and border rendering.
  - `FocusManager` provides modal focus trapping: focus and click registrations outside the modal are automatically blocked.
  - `ClickRegion` now contains a `LayerID` field to track the layer associated with each click region.
  - Backward compatibility is preserved: the existing `RegisterModal()` API remains available and updates both `ActiveModal` and the `Layers` list.
  - The incorrect `RegisterImage` callback signature (two parameters instead of three) was fixed.
  - The `layer_demo` example demonstrates multi-layer rendering, popup menus, and modal interaction.
  - Eight unit tests were added, including `TestLayerSystemBasic`, `TestMultiLayerZOrdering`, `TestRemoveLayer`, `TestTopLayer`, and `TestResetClearsLayers`.

- **Phase 11: Interactive and Flexible-Cell Table Component [COMPLETED]**
  - `widgets/table.go`, table state management, the constraint solver (`SolveWidths`), and the `Table` widget were implemented.
  - Cell-level clipping, grid lines, zebra striping, and dynamic column layout were added.
  - Unit tests (`widgets/table_test.go`) and an interactive process-table view in the main demo were completed successfully.
  - Hardware z-index support for the Kitty Graphics Protocol and cell-based Half-Block layering through `ForceHalfBlock` were added.

- **Phase 12: Advanced Layered Rendering and Animated Transitions [COMPLETED]**
  - Dialog and modal opening/closing animations were implemented using the `ScaleRect` and `SlideUpRect` helper matrix formulas.
  - A z-index-prioritized modal stack (`TopmostModal` and sandboxing) was integrated into `frame.go`.
  - Glowing, thick, dashed focus rings (`DrawFocusRing`) were added around focused widgets.
  - Unit tests in `modal_transition_test.go` and animated transitions/focus rings in the main demo were added.

- **Phase 13: Interactive TUI Playground and Dynamic Layout Controls [COMPLETED]**
  - An interactive Playground tab accessible from the left menu was added to the main demo.
  - A live control panel was created to handle keyboard arrow keys, `+`/`-` ratio adjustments, and mouse-click events.

- **Phase 14: CSS Grid Layout, Markdown Renderer, Retro Dither Transitions, and Vector Color Blending [COMPLETED]**
  - The 2D CSS Grid engine in `layout/grid.go` supports `Columns`, `Rows`, `Gap`, and cell merging through `.Span(rowSpan, colSpan)`.
  - The `widgets/markdown.go` component parses `#` headings, `**bold**`, `*italic*`, `- list`, and inline code blocks, with UTF-8-rune-aware word wrapping.
  - Smooth retro dither-fade transitions between screen tabs were implemented with a Bayer 4x4 matrix.
  - An avatar filter applying an anti-aliased circular mask to hardware-rendered images was added.
  - Sub-pixel RGB color blending for overlapping lines and graphics was integrated into the Braille Vector Canvas `Set` method.

- **Phase 15: Terminal Particle Rain (Matrix Rain) and High-Resolution Sparkline Renderer [COMPLETED]**
  - `widgets/sparkline.go` renders multiline area charts using eight vertical block characters with vertical fitting and normalization.
  - A Matrix Rain Canvas simulation was implemented to animate vertically falling particles.
  - A "Resize Ghosting Fix" buffer-reset engine was integrated to prevent character remnants after terminal resizing.

- **Phase 16: Interactive Mouse-Dragged Dialogs and Shortcut Help Panel [COMPLETED]**
  - TUI window dragging was implemented by tracking `RegisterClickHandler` and `MouseEvent.Drag` events, allowing dialogs to be freely dragged by holding the left mouse button on their title bars.
  - A Shortcut Help Panel modal opens when `?` is pressed and includes CSS Grid, Markdown help content, and a circular profile avatar.

- **Phase 17: Advanced Performance Profiling and Zero-Allocation Optimizations [COMPLETED]**
  - `widgets/markdown.go` was converted to use pointer receivers and AST/layout caching was integrated.
  - Markdown rendering performance improved by **11.3x**, and heap allocations during the draw loop were reduced to **zero (0 B/op, 0 allocs/op)**.

- **Phase 18: TUI Layout Inspector and Debugging Layer (Layout Inspector / Debug HUD) [COMPLETED]**
  - The `DebugRegions` structure automatically records the type, dimensions, and z-index of every drawn widget.
  - A Debug HUD layer triggered by `Ctrl+D` was integrated with pixel-perfect **z-order clipping**, allowing upper layers to obscure the outlines of lower layers.
  - A globally prioritized keyboard-routing system (Keyboard Focus Fix) was implemented to prevent keys from being swallowed in focused tables and buttons.

- **Phase 19: Interactive Mouse-Based Window Resizing [COMPLETED]**
  - A purple `◢` resize handle was drawn in the lower-right corner of the Shortcut Help window, with click/drag tracking.
  - The window dynamically resizes while being dragged (minimum: 40x10, maximum: 100x30). Internal Flex layout areas resize the Markdown content and profile avatar proportionally to the window size.

- **Phase 20: Widgets with Animated Transition Effects (Animated Widget Transducers) [COMPLETED]**
  - `widgets.Transducer` and dither-based modal/widget transitions were completed.
  - Tab transitions render directly from a clean frame to prevent the previous frame's text/canvas from being carried over piece by piece; modal and widget animations continue independently.
  - In addition to disabling the transition flag, `SetTransitionActive(false)` clears the transition progress and `transitionOldBuf`, preventing a closed transition from carrying an old image into the next modal/frame.
  - The Debug HUD is rendered after the dither transition so that debug boundaries and labels are not faded by the transition effect.
  - Full-frame transitions are used instead of a temporary body buffer, preventing widget text from mixing with stale cells and avoiding coordinate shifts.
  - Rows containing text or borders can be changed atomically by the terminal dither engine, preventing characters from being split between the old and new frames.
  - Opening a modal cancels any active terminal frame transition, preventing `transitionOldBuf` from being rendered as a second panel over the dialog. The modal runs its own scale animation independently.
  - Background widgets blocked by the modal sandbox are not added to `DebugRegions`; the Debug HUD therefore does not redraw invisible panels over the modal.

- **Phase 21: 3D Vector Graphics Engine [COMPLETED]**
  - Perspective projection (`Project`) and axis-rotation (`RotateX`/`RotateY`/`RotateZ`) functions were added.
  - An automatically rotating 3D cube controlled by left-mouse dragging was integrated on the Braille Canvas.

- **Phase 22: Command Palette and Keybindings [COMPLETED]**
  - The `CommandPalette` widget (`widgets/command_palette.go`) was implemented. It opens with `Ctrl+P`, performs fuzzy searches across tabs and actions, and executes the selected command.
  - `CommandItem` (`Label`, `Detail`, `Category`, `Handler`) and `CommandPaletteState` (`IsOpen`, `Query`, `Selected`, `ScrollOffset`, `MaxVisible`) were implemented.
  - The fuzzy search engine (`widgets/fuzzy.go`) includes VS Code-style scoring with consecutive-match, word-start, and CamelCase bonuses.
  - The declarative `KeybindingManager` (`widgets/keybinding.go`) was implemented with `Register`, `Handle`, and `ToCommandItems` APIs for event-loop integration.
  - `formatKeybinding` converts shortcuts into readable text such as `Ctrl+P`, `Shift+Tab`, and `↑`.
  - The feature was integrated into `examples/demo/main.go`: `Ctrl+P` toggles the palette, all keys are routed to the palette while it is open, and `Enter` executes the selected command.
  - Selecting a 3D model or rendering style from the command palette automatically switches to the `Graphics` tab so the change is immediately visible.
  - The `CommandPalette.DebugArea()` and `Frame.RenderWidget` debug-area-provider bridge reports the actual centered panel bounds instead of incorrectly reporting the full terminal area.
  - Unit tests were added: `widgets/command_palette_test.go`, `widgets/keybinding_test.go`, and `widgets/fuzzy_test.go`.

- **Phase 23: Advanced Table Cell Spanning and Column Resizing [COMPLETED]**
  - Column dragging through `TableState.ResizeColumn` preserves the total table width.
  - Temporary slice allocation during column resizing was removed; the minimum column width remains two cells.
  - `ColSpan` and `RowSpan` cells are rendered using a cell-ownership matrix.
  - Clipping for wide characters and emojis is based on terminal-cell width.
  - Tests for spanning, resizing, and wide characters were added to `widgets/table_test.go`.

- **Phase 24: Form Components and UI Box Model [COMPLETED]**
  - `Select` / dropdown support with keyboard navigation, mouse selection, and hover state was added.
  - The `Slider` component was completed with keyboard, mouse, drag, and capture support.
  - The `ProgressBar` component was added; the demo uses a 0→100→0 easing animation.
  - CSS-like `Margin`, `Padding`, and `Insets` APIs were added to the `Block` widget.
  - A standalone `examples/forms` example was created.
  - Click and hover events were separated; `MouseRelease` no longer triggers toggle events a second time.

- **Phase 25: Advanced 3D File Import [COMPLETED / EXTENSIBLE]**
  - A dependency-free Wavefront OBJ parser was added with support for vertices, polygon faces, texture/normal indices, and negative indices.
  - `Model3D.Normalize` scales file-based models for the existing perspective renderer.
  - OBJ files can be loaded into the demo with `LIMONI_OBJ=/path/model.obj go run ./examples/demo`.
  - Example models include `examples/demo/cube.obj` and the eight-part `examples/demo/deniz_topu.obj`.
  - A Canvas depth buffer and z-interpolated filled-triangle rasterizer were added, allowing OBJ faces to use depth testing independent of draw order.
  - OBJ `mtllib`, `usemtl`, `.mtl`, and `Kd` diffuse-color support was added; the demo uses material colors in filled-color mode.
  - OBJ `vt` and face UV-index support was added; textured demo rendering uses model UV coordinates.
  - An ASCII and binary STL loader was added; the demo selects OBJ or STL based on the `LIMONI_MODEL` extension.
  - An ASCII PLY loader was also added; it can be loaded with `LIMONI_MODEL=/path/model.ply`.
  - Future extensions may include a glTF/GLB loader, advanced texture/material features, and large-model optimizations.

- **Phase 26: Dashboard Table [COMPLETED]**
  - Column sorting with `▲/▼` header indicators, including numeric and text sorting.
  - Generic `FuzzyFilterBy`, `FuzzyFilterByFields`, and order-preserving `FuzzyFilterByStable`.
  - Fuzzy filtering through `Table.FilterQuery` and a demo search field.
  - Multi-select with `ToggleRow`, `IsRowSelected`, `ClearSelectedRows`, and `Space`-based selection.
  - The `Table.CellStyle(row, column, value)` callback and demo CPU/status color rules were completed.
  - The demo process table refreshes real PID, name, CPU delta, RSS memory, and status data from Linux `/proc` approximately every 500 ms.
  - Visible vertical-row rendering and a fixed header were completed; sticky-column horizontal navigation remains for a later phase.

---

## 5. Future Roadmap

1. **Phase 27: Rich Text and Centralized Theme System [COMPLETED / EXTENSIBLE]**
   - A `Span` / `Line` / `Text` rich-text widget was added.
   - Semantic token infrastructure with `Theme`, `ThemeColors`, `DarkTheme`, and `LightTheme` was added.
   - `Frame.SetTheme` / `Context.ThemeStyle` propagates the theme from the Frame to nested child widgets.
   - `Block` automatically uses the `surface` and `border` tokens when no custom style is provided; the main demo was migrated to semantic color tokens.
   - Accessibility validation was added through `HighContrastTheme`, `ContrastRatio`, and `Theme.ValidateContrast`.
   - Rich text supports cell-width-aware wrapping and left/center/right alignment.
   - Span semantic `Role` and `OnClick` callback support were added.
   - The phase is complete; possible future extensions include text selection, hyperlink semantics, and semantic-role integration for more widgets.
2. **Phase 28: Event Propagation, Focus Scope, and Horizontal Layout [COMPLETED]**
   - Provider-based virtual rows were added through `TableDataSource` / `RowCount` / `RowAt`.
   - Mouse-wheel vertical scrolling, `Shift+wheel` horizontal offset, and `StickyColumns` rendering were added.
   - Focus scope/group APIs (`BeginFocusScope`, `EndFocusScope`, and scoped Tab/Shift+Tab) were added.
   - Help and exit modals are isolated from background widgets through focus scopes.
   - Capture/target/bubble event propagation APIs (`RegisterEventHandler`, `EventContext`, `StopPropagation`, `PreventDefault`) and their tests were completed.
   - A deterministic text snapshot API (`Buffer.Snapshot`) and a visible-row table benchmark were added.
   - Horizontal grid/header intersections clipping, horizontal scroll offset column resize drag handle coordinates, and a zero-allocation responsive box model solver supporting `ConstraintMin` and `ConstraintMax` were completed.
3. **Phase 29: Terminal Capabilities and Developer Tooling [ACTIVE]**
   - Capability profiles for TrueColor, 256 colors, mouse, paste, and graphics.
   - A frame profiler, widget render-time measurements, allocation benchmarks, and a widget showcase.

## 6. Current File Structure

```
limoni/
├── go.mod
├── .agents/skills/limoni_development/skill.md  # This handbook
├── core/
│   ├── cell/
│   │   ├── cell.go       # Cell (16 bytes), Style (12 bytes), Color (uint32)
│   │   ├── rect.go       # Screen-area geometry (Rect)
│   │   └── context.go    # Stack-allocated cascading Context and Merge
│   ├── buffer/
│   │   ├── buffer.go     # Flat 1D cell matrix (Buffer), Resize
│   │   └── diff.go       # Zero-allocation double-buffered diff algorithm
│   ├── backend/
│   │   ├── types.go      # Flat event types (Event, KeyEvent, etc.)
│   │   ├── termios_linux # Pure Go raw-mode control through Unix ioctl calls
│   │   ├── parser.go     # Keyboard/mouse/focus/hover ANSI sequence parser
│   │   └── backend.go    # Asynchronous event loop with SIGWINCH and 25 ms ESC timeout
│   └── terminal/
│       ├── frame.go      # Frame context, click-handler registration, Layer API, DebugArea provider
│       ├── terminal.go   # Terminal manager, draw loop, multi-layer mouse router, Debug HUD
│       ├── focus.go      # FocusManager: Tab/Shift+Tab navigation
│       └── modal.go      # Modal, Layer, and LayerType structures; CenterRect and ContainsRect
├── layout/
│   └── layout.go         # Flexbox layout engine (Fixed, Percentage, Ratio, Min, Max, Fill)
├── widgets/
│   ├── widget.go         # Core Widget interface (Draw and SizeHint)
│   ├── block.go          # Bordered, titled CSS box-model container with Margin/Padding
│   ├── paragraph.go      # Multiline word-wrapping text widget
│   ├── list.go           # Selectable, automatically scrolling interactive list
│   ├── table.go          # Phase 23: Span, RowSpan, column resizing, and wide-cell clipping
│   ├── dialog.go         # 3D-shadowed modal dialog window
│   ├── textinput.go      # Single-line interactive text input
│   ├── checkbox.go       # Checkbox [ ]/[x]
│   ├── radio.go          # Single-selection control ( )/(*)
│   ├── popup.go          # Dropdown widget with hover highlighting
│   ├── select.go         # Select dropdown with keyboard/mouse/hover support
│   ├── textarea.go       # Multiline TextArea and cursor editing
│   ├── validation.go     # Form validator and field-error API
│   ├── slider.go         # Slider with keyboard/mouse/drag support
│   ├── progress.go       # Percentage- and style-aware ProgressBar
│   ├── richtext.go       # Span/Line/Text rich-text renderer
│   └── theme.go          # Semantic Theme and dark/light presets
│   ├── canvas.go         # Braille 2x4 sub-pixel-resolution drawing area
│   ├── vector.go         # Bresenham line, circle, rectangle, and Bézier drawing
│   ├── vector_depth.go   # Z-buffer-enabled filled-triangle rasterizer
│   ├── image.go          # Kitty/Sixel/iTerm2/HalfBlock image rendering
│   ├── command_palette.go # Phase 22: Command Palette (Ctrl+P, fuzzy search, CommandItem)
│   ├── keybinding.go     # Phase 22: Declarative keybinding manager (KeybindingManager)
│   └── fuzzy.go          # Phase 22: VS Code-style fuzzy search engine (FuzzyMatch/FuzzyFilter)
├── animation/
│   ├── float.go          # Time-based float interpolation
│   ├── color.go          # RGB color transition animation
│   └── easing.go         # 15+ easing functions
├── graphics/
│   ├── graphics.go       # Protocol detection and Kitty/Sixel/iTerm2 encoders
│   ├── vector3d.go       # 3D vertices, rotation, and perspective projection
│   ├── obj.go            # Wavefront OBJ parser, material library, and Model3D normalization
│   ├── mtl.go            # Wavefront MTL diffuse-material parser
│   ├── stl.go            # ASCII/binary STL loader
│   └── ply.go            # ASCII PLY loader
└── examples/
    ├── demo/main.go      # Fully interactive demo; table, forms, 3D, and OBJ import
    ├── demo/cube.obj     # OBJ import example
    ├── demo/deniz_topu.obj # Eight-sector beach-ball OBJ example
    ├── animation/main.go # Animation showcase
    ├── forms/main.go     # Select, Slider, ProgressBar, and box-model example
    └── layer_demo/main.go # Phase 10: Layered rendering, modal, and popup demo
```
