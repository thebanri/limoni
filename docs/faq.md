# Rendering Quirks & FAQ

## 1. Why do lines, 3D meshes, or images show hairline gaps in some terminals?
In standard terminal emulators, default monospace font line-height (cell padding) often adds 1–2px of empty vertical space between adjacent character rows. When rendering contiguous sub-pixel Braille matrices or half-blocks, this leading gap can cause surfaces to appear perforated ("grid gap" artifact).

### How Limoni Solves This: The Lower Half-Block (`▄`) Baseline Standard
Traditional TUI frameworks frequently use the Upper Half Block (`▀`, `U+2580`). Because typography engines anchor font glyphs to the **baseline** (bottom of the character cell), any extra line-height creates an uncolored gap at the *top* of the cell, physically detaching `▀` from the row above it.

Limoni standardizes on the **Lower Half Block (`▄`, `U+2584`)**:
- **Upper Pixel:** Rendered via the cell background (`Cell.Bg`).
- **Lower Pixel:** Rendered via the cell foreground (`Cell.Fg`).
- **Glyph:** Set to `▄`.

Because background colors always stretch to fill 100% of the character cell, and `▄` rests directly on the baseline, half-block graphics connect seamlessly without inter-cell cracks even on terminals with loose vertical spacing.

## 2. Recommended Terminal Settings for Visual Perfection
To experience Limoni's 3D software rasterization, charts, and Braille vector graphics at maximum fidelity:

* **Set Line Height to 1.0:** In your terminal's configuration, ensure `line-height` / `cell-height` is set to `1.0` (or `100%` / 0px vertical line padding).
* **Recommended Modern Terminals:**
  - **[Ghostty](https://ghostty.org):** Native GPU renderer with pixel-perfect contiguous box-drawing, Braille, and block element rendering out-of-the-box.
  - **[Kitty](https://sw.kovidgoyal.net/kitty/):** Ultra-fast OpenGL engine with native graphics protocols (`kitty` protocol) and gapless glyph rendering.
  - **[WezTerm](https://wezfurlong.org/wezterm/):** Exceptional font fallback and contiguous box glyph handling.
  - **[Alacritty](https://alacritty.org):** Ensure `font.offset.y: 0` and standard line spacing in `alacritty.toml`.
* **Recommended Monospace Fonts:** [JetBrains Mono](https://www.jetbrains.com/lp/mono/), [Fira Code](https://github.com/tonsky/FiraCode), or any patched [Nerd Font](https://www.nerdfonts.com/).

## 3. How does Limoni maintain 60+ FPS during rapid full-screen animations?
Limoni features a threshold-based **Adaptive Flush Engine**:
* **Sparse Diffing (`dirtyRatio < 0.45`):** For typing, metric tickers, and cursor blinks, computes minimal dirty cell regions and emits precise cursor jumps (`CUP`), completing in **`~14.2 µs`** with zero heap allocations.
* **Full-Stream Redraw (`dirtyRatio >= 0.45`):** When rotating 3D meshes, scrolling large tables, or fading tabs, switching to jump diffing would produce thousands of disjoint escape sequences. Limoni automatically switches to synchronized home (`\x1b[H`) full-stream streaming wrapped in DEC synchronized update mode (`\x1b[?2026h`), completely eliminating visual tearing and flicker while preserving **`0 B/op`** zero-allocation efficiency.

## 4. Emoji, flags and accented letters
Limoni stores one **grapheme cluster** per cell — what a reader sees as one character, however many code points it takes. `🇹🇷` (two regional indicators), `👨‍👩‍👧` (five code points joined by ZWJ), `👍🏽` (emoji + skin tone) and `é` written as `e` + U+0301 each occupy one cell, two columns wide for the emoji. Walking runes used to draw the flag as two letters, measure the family as six columns and drop the combining accent.

* **Rules:** segmentation follows UAX #29 for Unicode 17.0 and passes all 766 cases of the official `GraphemeBreakTest.txt`. A cluster's width is its widest code point (East Asian Width, emoji presentation), with VS16 forcing two columns and VS15 one.
* **Storage:** a cell still holds one `rune`. A multi-code-point cluster is interned once in a shared table and the cell stores a handle to it, so `Cell` stays 16 bytes and single code points — nearly all text — never touch the table. The table is capped at about a million distinct clusters; past that, new clusters degrade to their first code point instead of growing memory.
* **Terminals:** Limoni requests mode 2027 (`CSI ? 2027 h`), which Ghostty, WezTerm, foot and Contour implement. Terminals without it advance the cursor per code point and may draw a family emoji six columns wide. To stop that from shifting the rest of the row, the diff re-anchors the cursor right after every cluster. The cluster itself can still look wrong on such a terminal, but nothing after it moves.
* **Opting out:** `LIMONI_GRAPHEME=0` (or `cell.SetGraphemeClusters(false)`) restores one code point per cell and skips the mode 2027 request.
* **Not converted yet:** text drawn through `Buffer.SetString` and measured with `cell.StringWidth` is cluster-aware. Widgets that cut or place text by rune count — `TextInput`, `TextArea`, table and toast truncation, among others — can still split a cluster where they truncate.
