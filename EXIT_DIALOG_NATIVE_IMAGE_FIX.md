# Exit Dialog, Native Profile Image, and Terminal Compatibility (Kitty & Alacritty)

This document details the root causes, architectural solutions, and permanent fixes for visual artifacts that previously occurred when an exit dialog hovered over a native profile image.

## Symptoms and Issues Encountered

1. **Transparent Dialog**: When the dialog opened over the profile image, the background cell color did not occlude the native GPU image layer; the profile image bled through the dialog.
2. **Ghost Lines & Artifacts**: Gradient bars or table lines from prior tabs (Settings, Graphics, Home) or dragged dialog borders remained stamped on top of the native profile image.
3. **Black Staircase Artifacts**: Dragging or resizing the dialog across an image left rectangular black staircase clipping artifacts.
4. **Alacritty Screen Flicker & Tearing**: Opening, closing, or animating the dialog under Alacritty caused full-screen clearing (`\x1b[2J`) and flickering.

---

## Root Causes

1. **Layer Stacking and Z-Index**:
   - In the Kitty Graphics protocol, images are placed at `z < 0` beneath the text grid. A standard cell background (`Style.Bg`) alone cannot occlude hardware-accelerated GPU image overlays.
2. **Cursor Tracking & Diff Skipping (`diff.go`)**:
   - Skipping image cells (`cell.RuneImage`) incremented virtual cursor coordinates (`cursorX++`), while the physical terminal cursor remained parked on the left margin. Subsequent characters were emitted at offset positions over the image.
3. **Stale Cell Retention**:
   - When a dialog moved off an image, former dialog characters were not cleared from the terminal hardware buffer, remaining visible over the GPU image layer.
4. **Unnecessary Full Clears (`\x1b[2J`)**:
   - Invoking `ForceFullRedraw()` triggered full terminal screen clears on non-graphics emulators like Alacritty, creating visible screen tearing.

---

## Definitive Architectural Solution

### 1. Z-Index Layering Architecture
```text
Profile image          -3 (Bottom layer via Frame.imageClosure)
Dialog opaque backdrop -2 (shadowBackdrop: animatedArea.Width+2, Height+1)
Dialog shadow          ASCII buffer
Dialog border/text     ASCII buffer (z = 0, crisp top-layer title, message, and buttons)
```

- Implementation in `examples/demo/main.go`:
```go
if animatedArea.Width > 0 && animatedArea.Height > 0 {
    shadowBackdrop := cell.NewRect(
        animatedArea.X,
        animatedArea.Y,
        animatedArea.Width+2,
        animatedArea.Height+1,
    )
    f.RenderWidget(widgets.Block{
        Style:  cell.Style{Bg: cell.NewColorRGB(18, 20, 24)},
        Opaque: true,
    }, shadowBackdrop)

    exitDialog := widgets.Dialog{ ... }
    f.RenderWidget(exitDialog, animatedArea)
}
```

### 2. Dynamic ECMA-48 ECH Erasing & Hardware Cursor Invalidation
- File: `core/buffer/diff.go`
- Invalidate physical cursor tracking when scanning image cells (`cursorX = 9999, cursorY = 9999`).
- When cells previously occupied by text or dialogs are exposed (`needsErase = true`), reset styles with `\x1b[0m` and emit `\x1b[<length>X` (ECH - Erase Characters) to clear character cells in terminal memory immediately.

### 3. Alacritty (ProtocolHalfBlock) Isolation
- Files: `widgets/block.go`, `core/terminal/terminal.go`
- `Block.Opaque` only registers native image overrides when `proto != graphics.ProtocolHalfBlock`.
- In Alacritty, all rendering routes directly through the cell matrix at 60 FPS with zero latency, eliminating full-screen clears and tearing.

### 4. Clean Tab Switching Transitions
- Files: `examples/demo/main.go`, `examples/demo/helpers.go`
- On explicit tab switch events (mouse click, Enter/Space, Shift+Tab, or Command Palette), trigger a single synchronized refresh to ensure no stale text remains in the terminal hardware buffer.

---

## Verification Checklist

- [x] Switching tabs leaves zero residual artifact lines over native images.
- [x] Exit dialog completely occludes profile images without transparency artifacts.
- [x] Dragging dialogs produces zero ghost lines and zero black staircase clipping.
- [x] Alacritty exhibits zero flicker or full-screen clearing.
- [x] `go test ./...` and `go test -race ./...` pass with zero failures.

Core Principle:

> Confine dialog animation to the dialog's visual bounds; decouple the native image's geometry and transformations from the dialog's animation state.