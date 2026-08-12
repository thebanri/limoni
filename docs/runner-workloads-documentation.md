# Runner Workload Execution Documentation

This document describes the exact operations executed by Limoni, Bubble Tea, and Ratatui for key benchmark workloads, guaranteeing fairness and equivalents in comparison.

---

## 1. `empty-frame` (W: 80, H: 24, Iterations: 1000)
- **Limoni**: Bypasses any widgets. Executes `front.Clear()` on every frame, calls `buffer.Diff(front, back, ...)` to calculate changes. Bypasses loop execution and returns immediately due to `front.IsDirty == false` fast-path check.
- **Bubble Tea**: Model `Update` does no-op. `View()` method returns an empty string `""` immediately without terminal cell formatting or double buffering.
- **Ratatui**: Calls `terminal.draw(|_f| {})`. Bypasses drawing widgets but executes the underlying `TestBackend` buffer comparison logic on 1920 cells.

---

## 2. `full-redraw-120x40` (W: 120, H: 40, Iterations: 1000)
- **Limoni**: Loops through 4,800 coordinates, calling `front.SetCell(x, y, ...)` to write characters with alternating foreground/background colors. Calls `buffer.Diff` which outputs style escape codes for every cell.
- **Bubble Tea**: Instantiates a 4,840-byte slice in memory containing `'A'` and newline characters, converting it to a string. Returns the raw string without formatting colors or executing terminal diff blocks.
- **Ratatui**: Repeats character `'A'` to fill the buffer. Paragraph widget renders the text on 4,800 cells with default terminal styles.

---

## 3. `single-cell-update` (W: 80, H: 24, Iterations: 1000)
- **Limoni**: Renders a static string on the first frame. On subsequent frames, updates a single cell at `(0, 0)` alternating between `'X'` and `'Y'`. Calls `buffer.Diff` which scans modified row bounds `[0, 0]`, emitting a single cursor move and character write.
- **Bubble Tea**: Alternates model toggle boolean. `View()` returns `"X"` or `"Y"` string representation.
- **Ratatui**: Alternates model toggle. Paragraph widget renders `"X"` or `"Y"` on Rect `(0, 0, 1, 1)`.

---

## 4. `resize` (W: 80, H: 24, Iterations: 1000)
- **Limoni**: Alternates buffer size between `(80, 24)` and `(100, 34)`. Calls `front.Resize` and `buffer.Diff` which detects size changes, clears the screen with `\x1b[2J`, and updates the backend size.
- **Bubble Tea**: Passes `tea.WindowSizeMsg` to `Update()`, updating layout dimensions variables in the model.
- **Ratatui**: Alternates size between `(80, 24)` and `(100, 34)`. Calls `terminal.resize()` and `terminal.draw()`.

---

## 5. `virtual-1000000` (W: 120, H: 40, Iterations: 1000)
- **Limoni**: Viewport is populated with 40 rows. `VirtualDataView` widget calculates visible range, reads from the optimized viewport cache `VirtualDataState` (preventing RowAt provider hits on static frames), and uses cached formatting text `rowTextCache` to write to the buffer.
- **Bubble Tea**: Loops 40 times to print 40 rows as a single string. Bypasses cell layouts, alignment constraints, or double-buffered calculations.
- **Ratatui**: Pre-builds 40 table rows in setup. In timed loop, table widget processes layout constraints and draws the table on the screen.
