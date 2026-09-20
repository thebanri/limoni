---
name: limoni-zest
description: zest, the log viewer built on Limoni — how its store, filtering and LogView fit together, how it is verified (in-process, in kitty, over MCP, in the browser), and what it deliberately does not do. Load before changing cmd/zest, internal/zestapp or widgets/logview.go.
---

# zest: the flagship log viewer

`cmd/zest` is the command, `internal/zestapp` the code (so the WebAssembly
playground can run the same viewer), `widgets.LogView` the pane itself.

## The shape

```
readLines / generateDemo  →  store  →  view (filter)  →  widgets.LogView
        (goroutine)          chunks     idx []int32        draws what fits
```

- **store** appends line text into fixed 1 MiB chunks that are never modified
  afterwards, so `Line(i)` hands out `unsafe.String` over those bytes. Nothing
  is copied per line or per frame; that is what keeps drawing a million-line log
  allocation-free. A line bigger than a chunk gets a chunk of its own. Memory is
  the text plus 16 bytes per line.
- **detectLevel** reads the level from JSON (`"level"`, `"lvl"`, `"severity"`),
  logfmt (`level=warn`) and plain text, looking at the first 200 bytes only. An
  indented line with no level of its own inherits the previous line's, so
  filtering to errors keeps a stack trace whole.
- **view** is the filter. Scanning runs in a goroutine in batches of 200,000
  lines and calls `wake` after each, so a keystroke is never blocked; a filter
  change bumps `gen` and an in-flight scan of the old filter discards its batch.
  Without a filter it is a pass-through and costs nothing.
- **LogView** draws only the rows on screen, follows new lines (`tail -f`), and
  stops following when the reader scrolls *or clicks a line* — a click sets
  `Selected` directly, and without that check the chosen line scrolled away.
  `LineNumberSource` is how a filtered view shows original line numbers.

## Verification, in four places

1. **In process**: `internal/zestapp/ui_test.go` and `follow_test.go` drive the
   whole viewer through its semantic tree with `uitest`.
2. **A real terminal**: run it in kitty with `--listen-on`, then
   `kitty @ get-text` reads the rendered screen back and `kitty @ send-text`
   types. This is how the load times were measured (1,000,000 lines, 67 MiB,
   0.54 s to screen, 123 MB RSS).
3. **Over MCP**: build with `-tags limoni_debug`, run with `-socket`, and drive
   it with the real `limoni-mcp` binary or a headless agent. Drain the PTY on a
   thread or the app blocks writing and looks hung.
4. **In the browser**: `scripts/verify-wasm.mjs` (which Pages runs before
   deploying) opens the "Logs · zest" scene under Node, filters to errors, and
   checks the emitted bytes. Look for fragments, not whole lines: at 100 columns
   the terminal cuts them off.

`ZEST_MEASURE=1 go test ./internal/zestapp -run TestPreloadSpeed -v` prints the
generate-and-store time for a million lines (0.62 s here).

## Deliberate limits

- One input at a time; several files are not merged by timestamp.
- The filter is a substring, not a regular expression.
- No pipe on Windows: keys need `/dev/tty`. Pass a file.
- The demo generator is seeded, so its content is reproducible — that is what
  makes it usable as ground truth for agent runs.

## Assets

`assets/zest.gif` is recorded by `scripts/record_zest.py` (runs zest in a PTY,
types at a human pace, writes an asciicast) and rendered with `agg`. Record with
`LIMONI_REP=0`: agg's terminal emulator does not implement REP.
