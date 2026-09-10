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
2. If `dirtyRatio >= 0.45`:
   - Skip sparse diffing and cursor jump math.
   - Emit synchronized home + full sequential draw (`\x1b[?2026h` + `\x1b[H`).
   - Stream `back` buffer continuously to stdout with lazy style updates.
   - Synchronize `front` buffer via fast memory copy (`copy(front.Content, back.Content)`).
3. If `dirtyRatio < 0.45`:
   - Proceed with the standard sparse diffing and cursor jumping pipeline.

#### Acceptance Criteria
- [x] Benchmark memory allocations on 100% dirty frames (verified 0 B/op).
- [x] Measure byte throughput difference during full-screen 3D model rotation before vs. after.
- [x] Verify no visual tearing when switching dynamically between sparse and full-clear modes.
- [x] Standardize half-block rendering on `▄` (U+2584) Lower Half Block baseline standard.



### [Docs & Assets] Re-record demo assets in Ghostty and document font gap issue

#### Context & Problem
In standard terminal emulators, default line-height (cell padding) and certain monospace fonts leave 1-2px hairline gaps between adjacent character cells.
When rendering half-blocks (`▀`, `▄`) or Braille sub-pixel matrices, these gaps break the continuous visual surface and make 3D meshes or images look perforated ("grid gap" artifact).

#### Tasks
- [x] **Asset Refresh:**
  - Converted recorded showcase demos into ultra-lightweight, 720p palette-optimized GIFs (`assets/3d.gif`, `assets/treeview.gif`, `assets/chart.gif`).
  - Total asset weight reduced from 146 MB to ~8.1 MB across the repository.
  - Linked optimized GIFs directly in `README.md` and `README_TR.md`.

- [x] **Documentation / FAQ Note:**
  - Added dedicated section in `README.md` and `README_TR.md` under "Rendering Quirks & FAQ".
  - Documented the root cause of monospace font hairline gaps in block graphics.
  - Documented Limoni's Lower Half-Block (`▄`, U+2584) baseline standard and how it eliminates vertical gaps.
  - Recommended optimal terminal configurations (`line-height: 1.0`, Ghostty, Kitty, WezTerm, Alacritty, Nerd Fonts).



### [Feature/Runtime] Context-Aware Application Lifecycle & Instance Isolation

#### Context & Problem
Currently, `limoni.Run(appFn, opts...)` uses a package-level `wakeupChan` and runs until `appFn` returns `false` or receives SIGINT/Ctrl+C. In long-running microservices, SSH servers, or embedded daemon threads, the application lifecycle should be controllable via standard Go `context.Context`.

#### Tasks
- [ ] **Context Lifecycle API (`limoni.RunWithContext`):**
  - Accept `ctx context.Context` to enable graceful shutdown when the parent context is cancelled.
  - Convert `wakeupChan` into an instance-bound channel inside `appConfig` or `Terminal`, avoiding global channel collisions across concurrent test runners or multi-session SSH applications.
- [ ] **Idle Throttling / Dynamic Sleep:**
  - Automatically drop terminal render poll rate when no input events arrive and no background ticker is active, saving battery and CPU cycles on mobile/laptop machines.



### [Feature/Graphics] Modern 3D Asset Import: glTF 2.0 (.gltf / .glb)

#### Context & Problem
Limoni currently supports Wavefront `.obj`, STL (`.stl`), and Stanford PLY (`.ply`). The industry-standard 3D format for real-time graphics and web models is glTF / GLB (GL Transmission Format).

#### Tasks
- [ ] **Dependency-Free GLB (Binary glTF) Parser (`graphics/glb.go`):**
  - Parse GLB 12-byte header (magic `0x46546C67`, version 2, length) and JSON/BIN chunks.
  - Extract vertex positions (`POSITION`), normals (`NORMAL`), and texture coordinates (`TEXCOORD_0`).
  - Feed extracted geometry into `graphics.Model3D` and rasterize with Limoni's z-buffered Lambertian/Gouraud shader.
- [ ] **Showcase Demo:**
  - Add runnable sample model (`examples/demo/lemon.glb`) loadable with `go run ./examples/3d_viewer`.
