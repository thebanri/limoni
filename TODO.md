### [Feature/Optimization] Adaptive full-redraw threshold for high-churn frames

#### Context & Problem
Currently, the differential engine computes cell-by-cell diffs and emits cursor positioning sequences (`CUP`) for every dirty region. 
While this is optimal for sparse updates (typing, metric updates, cursor blinks), frames with high churn rates (full-screen 3D mesh rotation, fast scrolling, modal closing over large transparent areas) suffer from:
1. Overhead of calculating jump diffs across thousands of cells.
2. Emitting more bytes via individual cursor jumps than a simple sequential flush would take.
3. Potential artifacts when clearing large areas that simulate transparency.

#### Proposed Solution (Threshold-based Adaptive Redraw)
Implement a threshold check during the diff pass:
1. Count dirty cells: `dirtyRatio = float64(dirtyCells) / float64(totalCells)`.
2. If `dirtyRatio >= 0.50` (or configurable threshold):
   - Skip sparse diffing and cursor jump math.
   - Emit synchronized home + full sequential draw (or `\x1b[H\x1b[2J` clear when appropriate).
   - Stream `back` buffer continuously to stdout with lazy style updates.
   - Synchronize `front` buffer via fast memory copy (`copy(front, back)`).
3. If `dirtyRatio < 0.50`:
   - Proceed with the standard sparse diffing and cursor jumping pipeline.

#### Acceptance Criteria
- [ ] Benchmark memory allocations on 100% dirty frames (must remain 0 B/op).
- [ ] Measure byte throughput difference during full-screen 3D model rotation before vs. after.
- [ ] Verify no visual tearing when switching dynamically between sparse and full-clear modes.



### [Docs & Assets] Re-record demo assets in Ghostty and document font gap issue

#### Context & Problem
In standard terminal emulators, default line-height (cell padding) and certain monospace fonts leave 1-2px hairline gaps between adjacent character cells.
When rendering half-blocks (`▀`, `▄`) or Braille sub-pixel matrices, these gaps break the continuous visual surface and make 3D meshes or images look perforated ("grid gap" artifact).

#### Tasks
- [ ] **Asset Refresh:**
  - Install Ghostty terminal (which renders box-drawing, block elements, and braille glyphs seamlessly without inter-cell padding).
  - Re-record the hero demo GIF, 3D mesh turntable preview, and image rendering comparisons inside Ghostty.
  - Replace assets in `README.md` and the marketing website.

- [ ] **Documentation / FAQ Note:**
  - Add a short section in `README.md` (under "Rendering Quirks & FAQ") explaining why lines/meshes might show hairline gaps in some emulators (e.g. Alacritty, GNOME Terminal, Windows Terminal).
  - Recommend optimal user terminal settings:
    - Set terminal `line-height` / `cell-height` to `1.0` (disable extra vertical line spacing).
    - Recommend patched fonts or terminals that support contiguous block glyphs (Ghostty, Kitty with symbol scaling enabled).
