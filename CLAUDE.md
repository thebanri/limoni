## Git Commit Guidelines
- Never add `Co-Authored-By: Claude...` or `Claude-Session:...` trailers to git commit messages.


# Limoni — working notes for Claude Code

Limoni is a terminal UI engine for Go: a flat 1D cell grid, a double-buffered ANSI
diff, and zero heap allocations on the render hot path. It also ships a software
3D rasteriser, four image protocols, charts, markdown, and an accessibility tree —
none of which the other Go TUI libraries have.

Longer architectural background lives in `.agents/skills/limoni_development/skill.md`
and `docs/`. This file is the short version plus the things that are easy to get
wrong.

Four skills in `.claude/skills/` carry the detail for the areas that bite
hardest, and load themselves when the work touches them:

- `limoni-agent-surface` — the semantic tree, the automation socket,
  `cmd/limoni-mcp` and `uitest`: invariants, the bugs they came from, and how to
  test an agent surface (unit, mutation, PTY, a real agent).
- `limoni-text-rendering` — grapheme clusters, widths, mode 2027, the cluster
  table, and the tools widgets use to measure and cut text.
- `limoni-performance` — where allocations hide in a frame, how to find them
  with a profile, how to compare benchmarks, and the release steps.
- `limoni-zest` — the log viewer: its store, filtering and LogView, and the
  four places it is verified.

---

## Verify before you claim anything works

```bash
go build ./...
go vet ./...
go test ./...
go test -race . ./component ./core/engine ./core/terminal ./testkit ./widgets ./layout ./core/accessibility ./core/driver

# Allocation budgets — these are enforced in CI and must stay at 0
go test ./core/buffer -run '^$' -bench 'BenchmarkDiff_' -benchmem
go test ./widgets    -run '^$' -bench . -benchmem
```

The WebAssembly build is part of the contract, not an afterthought:

```bash
GOOS=js GOARCH=wasm go build -o /tmp/limoni.wasm ./examples/wasm
```

---

## Non-negotiables

**Zero allocations on the hot path.** `Draw` implementations must not allocate.
Write a `-benchmem` benchmark for every new widget and check `0 B/op`. If a widget
genuinely cannot avoid allocating, say so in its doc comment with the measured
number rather than leaving it for someone to discover.

**Comments in English.** Parts of the codebase still carry Turkish comments
(`widgets/`, `core/terminal/`, `graphics/`, `layout/`, `animation/`). New code is
English; converting the rest is welcome. User-facing docs stay bilingual —
`README.md` / `README_TR.md` and `docs/` / `docs/tr/` — that is deliberate.

**Benchmark honesty is a hard rule.** This repository publishes comparisons against
Bubble Tea and Ratatui. Read `docs/benchmark-methodology.md` before touching any
number, and follow it:

- Every comparison carries an explicit version label. "Bubble Tea" alone is not a
  claim, "Bubble Tea v1.3.10" is.
- If the runners measure different pipelines, say so rather than quoting the ratio.
  The v1 Bubble Tea runner measures view-string construction only; the Limoni
  runner measures drawing *plus* the full ANSI diff. That gap is documented, not
  hidden.
- Workloads with no equivalent on the other side (`empty-frame`, `mouse-hit-test`,
  `async-update-burst`) are reported without ratios.
- A number that looks too good is a bug in the harness until proven otherwise. One
  earlier run reported a 4,700× advantage; it was entirely a harness artifact.

**Compiling is not verifying.** Two real bugs shipped in this repo because someone
checked that the code built and stopped there: the WebAssembly demo called
`Program.Run` (which only drives the message loop) instead of `RunTerminal`, so it
rendered nothing at all; and the browser ran in 16 colors because capability
detection reads environment variables that do not exist under `js/wasm`. Both were
found by running the thing, not by building it.

---

## Two application models, both first-class

```go
// Immediate mode — dashboards, 3D viewers, games, animation
limoni.Run(func(f *limoni.Frame, ev *limoni.Event) bool { ... })

// Declarative (Elm architecture) — forms, wizards, CRUD, async work
limoni.RunProgram(ctx, &model{})          // batteries included
limoni.NewProgram(&model{}, opts...)      // when you own the terminal
```

`Model`, `Msg`, `Cmd`, `UpdateResult` and the runtime message types are re-exported
from the root package; the runtime itself lives in `core/engine`. Options that
collide with the immediate-mode `AppOption` names are spelled
`WithProgramFPS` / `WithProgramCatchCtrlC` / `WithoutProgramQuitKeys`.

Both modes are context-aware: `limoni.RunWithContext(ctx, fn)` and
`limoni.NewApp(term, opts...).Run(ctx, fn)` for immediate mode. An `App` owns
its wakeup channel, so several run in one process (one per SSH session in
`examples/ssh_server`); package-level `limoni.Wakeup()` wakes all of them.

Note: `.agents/skills/limoni_development/skill.md` still refers to `runtime.New`.
The package was renamed to `core/engine`; that doc is stale in places.

---

## Package map

| Path | What lives there |
| :--- | :--- |
| `core/cell` | `Cell`, `Style`, `Rect`, `Context`, rune widths |
| `core/buffer` | Flat 1D buffer, the ANSI diff engine, snapshots |
| `core/terminal` | `Terminal`, `Frame`, focus, modals, capability profile |
| `core/driver` | termios, epoll/kqueue, Windows VT, WASM bridge, input parser, clipboard |
| `core/engine` | Elm runtime: messages, commands, cancellation, panic recovery |
| `core/accessibility` | Semantic tree, line navigation, screen-reader mode |
| `component` | Composable view tree (VStack/HStack/Border/Flex…), zero-alloc stack solver |
| `layout` | Flexbox and grid constraint solving |
| `widgets` | The widget catalogue |
| `graphics` | 3D meshes (OBJ/STL/PLY/GLB), shading, image protocol encoders |
| `testkit` | Deterministic in-memory terminal, golden files |
| `uitest` | Playwright-style locators and waiting assertions over the semantic tree |
| `automation` | Semantic tree over a Unix socket (wired into apps only with `-tags limoni_debug`) |
| `cmd/limoni-mcp` | MCP bridge from agents to the automation socket |
| `benchmarks` | Harness plus the cross-framework runners |
| `apps/globe` | A searchable, zoomable ASCII globe — a separate module, so it ships to nobody |
| `apps/castle-lemonstein` | A short raycaster game in half blocks (rats, lemons, Ratatui as the boss), with synthesised sound — its own module, 0 allocs per frame through the diff |
| `apps/lemondrop` | Falling blocks that turn to sand; a one-colour run from wall to wall clears — its own module, 0 allocs per frame through the diff |
| `apps/scoreboard` | Castle Lemonstein's and Lemon Drop's shared leaderboards (`/scores`, `/drop/scores`): Vercel functions over a free Neon Postgres, or the same handler as a long-running server (Render, Railway) — its own module, so pgx reaches no importer |

---

## Traps that have already cost time

**`go.mod` floor.** The module once declared `go 1.26.5`, which forced every user
to download that exact toolchain and made the module uninstallable on anything
older. It is now `go 1.25`, the honest floor — `golang.org/x/sys v0.47.0` itself
requires 1.25. The whole tree compiles under a `go 1.23` directive, so 1.25 is a
dependency constraint, not a language one. Dropping to 1.23 means moving to
x/crypto v0.38.0 + x/sys v0.33.0, which costs security fixes; that is a support
policy decision, not a cleanup.

CI enforces this with `go list -m` under `GOTOOLCHAIN=local` on every module. Note
that three-component directives like `go 1.25.0` are *fine* — that is what the Go
toolchain writes. The failure mode is a directive newer than the toolchain in hand.

**Bubble Tea v2 needs Go ≥ 1.25** and lives at the vanity path
`charm.land/bubbletea/v2`, not `github.com/charmbracelet/bubbletea/v2`.

**Ultraviolet's layers matter.** Drive `uv.TerminalScreen`, not `uv.TerminalRenderer`.
The screen owns the dirty tracking; the renderer does not reset it, so a harness
that bypasses the screen re-scans every line every frame. Also: `Render()` composes
and diffs into an internal buffer — `Flush()` is what emits bytes. And pin the
colour profile explicitly, or Ultraviolet sniffs the environment and downsamples
while Limoni diffs in truecolor, making byte counts incomparable.

**Never commit build artifacts.** A 5.9 MB `widgets.test`, a 3.1 MB `limoni.wasm`
and a 4.8 MB runner binary all made it in at various points. CI now rejects tracked
files that are ELF/Mach-O/PE executables, whatever they are named.

**`go get` bumps the `go` directive silently.** Adding `golang.org/x/tools@v0.50.0`
to `tools/limonivet` rewrote its `go.mod` to `go 1.26.0`, which the Go 1.25 CI
toolchain cannot resolve under `GOTOOLCHAIN=local`. It is pinned to v0.49.0, the
newest release that declares `go 1.25.0`. After any `go get`, read the directive.
Better: verify with a real old toolchain, `GOTOOLCHAIN=go1.25.0 go list -m`, which
downloads it once.

**Automation and session recording must stay out of release binaries.** The
automation gateway lives behind `-tags limoni_debug`; the session recorder is only
linked if an application imports `session`. The root package must import neither in
an untagged build — CI checks `go list -deps` and the symbol table of a built
example. An untagged import of either quietly defeats the whole security model.

**A kitty image that reaches the last row scrolls the screen.** Without
`C=1` kitty moves the cursor below a placed image; at the bottom row that
scrolls the whole screen a line, under a diff that does not know. It showed
as a smeared left pane next to a pixel-mode Viewer3D. iTerm2 gets
`doNotMoveCursor=1`. `TestImagePlacementsKeepTheCursor`.

**Images are cached by value.** `GetCachedEscapeSequence` keys on the
`image.Image`; one whose pixels change in place must be `graphics.ForgetImage`d
and should be a different value per frame, or the terminal is sent the old
picture — or nothing, since the terminal compares images by identity.

**A Program without `WithProgramFPS` must not tick.** `RunTerminal` used to
draw 30 frames a second whatever happened: 0.7% of a core and ~210 wakeups a
second while idle, where `limoni.Run` and ratatui 0.30 use nothing (Bubble Tea
v1.3.10 and v2.0.10 tick at 60 and cost about the same). Measured from a real
pty, idle CPU from the kernel's schedstat. Now it draws on input, redraw
requests and Updates; one that did not ask for a redraw is drawn at once, or at
the end of the current frame under a stream of them. Both loops also draw once
when the capability probe is answered (`Backend.ProbeAnswered`), or late
answers would wait for the next key. `core/engine/idle_test.go`.

**Continuation cells written through SetCell used to blank wide characters.**
Fixed in the buffer (`setContinuation`); Markdown and TextInput's wide mask
were drawing spaces.

**Tests: escape combining characters.** The write path of this tool turned
`\u200D` and `\u0301` in heredocs into the literal characters; build them
with `chr(92)` in Python, then check with the snippet in the text-rendering
skill.

**`settings.json` once contained a live API key** and is gitignored for that reason.
Do not re-add it.

**Nested modules are what keeps the download honest.** `assets/`, `apps/globe`
and `tools/limonivet` each carry their own `go.mod`, and a directory with a
`go.mod` is not part of the module around it — so none of them reach anyone who
imports Limoni. That took the published module from **14.3 MB to 3.7 MB**; the
difference was four README GIFs. CI checks that those `go.mod` files are still
there. Two consequences: nothing in the library may `go:embed` or read from
`assets/`, and a new application belongs in `apps/<name>` with its own module,
not in `cmd/`.

To see what a change would actually ship, build the zip the proxy would build,
with `golang.org/x/mod/zip`, rather than reasoning about it — that is how the
14.3 MB was found in the first place. An untagged nested module still installs:
`go install github.com/thebanri/limoni/apps/globe@latest` resolves to a
pseudo-version of the default branch, so no `apps/globe/v0.1.0` tag is needed.

---

## State

`main` is green: CI, benchmarks, platform smoke and the Pages playground all pass.
The browser playground is live at <https://thebanri.github.io/limoni/> and is
rebuilt from `main` by `.github/workflows/pages.yml`.

Recently landed: the declarative runtime re-exported from the root package;
`Viewport`, `Scrollbar`, `Tabs` and `Spinner`; the WebAssembly playground; a real
Bubble Tea v2 benchmark runner with a documented baseline.

---

## Open work, roughly in priority order

1. **Instance isolation — done.** `limoni.App` carries its own wakeup channel
   and terminal; `Run` and `RunWithContext` build one on stdio. Verified with
   two concurrent `ssh -tt` sessions against `examples/ssh_server`: both
   animate, quitting one leaves the other running.
   `TestAppsInOneProcessAreIsolated` alternates wakeups between two apps; with
   a shared channel it fails, because the goroutine that waited first takes
   every wakeup. A version that woke only one app first passed with the bug.

2. **Terminal capability handshake — done, keep it honest.** Setup sends
   `driver.ProbeQueries` (XTVERSION, DECRQM 2026/2027, Kitty keyboard query, two
   cursor-position *measurements* for REP and cluster width, DA1 last as the
   sentinel). The event loops fold replies into `driver.TerminalReport` and
   never forward them; `Terminal.Draw` applies them via
   `CapabilityProfile.WithReport` and repaints fully if they land after a
   frame. Measured on this machine: kitty 0.48.2 and Konsole 26.08.1 draw a
   ZWJ family 2 columns wide without mode 2027; Alacritty draws it 6. That is
   why measurement beats the name table. `limoni doctor` shows the whole
   decision; verify changes in real terminals with it (via `script -q -c` to
   capture), not only with the in-memory tests. The kitty keyboard answer now
   pushes level 1 (`CSI > 1 u`, popped by `RestoreModes` on Close and Suspend):
   verified in kitty 0.48.2 that Ctrl+I, Esc and Alt+key arrive distinct.
   Level 1 leaves Enter, Tab and Backspace legacy by design, so Shift+Enter is
   still plain Enter; going further (level 8) needs flag 4 and IME text, and
   breaks AltGr layouts if done carelessly. Not yet used: the DA1 sixel bit.
   `WithKeyReleases` / `SetKeyReleases` push flags 3 instead (event types):
   presses arrive as before, repeats and releases as `CSI …;mods:2|3 u` or
   `CSI 1;mods:3 A`, parsed into `KeyEvent.Repeat`/`Release`. Opt-in only — to
   an application that did not ask, a release would read as a second press.

   To drive a real kitty from a test run: `kitty -o allow_remote_control=yes
   --listen-on unix:/tmp/x.sock app`, then `kitty @ --to unix:/tmp/x.sock
   send-key shift+enter` / `get-text`. Keep the socket path short (108 bytes),
   and use `script -f` — without it the capture is empty until exit.

3. **Agent-facing semantics.** `cmd/limoni-mcp` serves the automation socket
   as MCP tools, and a headless Claude Code run completed
   `examples/agent_checklist` with it. `uitest` is the Playwright-style test
   API over the same tree (in-process `Run`/`Program`, remote `Connect`).
   Lists expose visible rows as children, from a buffer in `ListState` so the
   draw path stays allocation-free — which is why `Frame.AccessibilityTree`
   deep-copies and `f.Accessibility` must not be kept past a frame. Table rows,
   TreeView items and Tabs are children too, and `Check`/`Uncheck`/`Select`
   (MCP: `click` with `ensure`) are idempotent. Still missing: custom widgets
   embedding `widgets.Accessible` are not focusable, so Tab skips them. Test
   the bridge against a real app in a PTY as well as with `go test` — the Tab
   bug below was invisible to unit tests. Custom widgets embedding
   `widgets.Accessible` are focusable now: `Accessible.WantsFocus` and a hook in
   `Frame.RenderWidget` (`TestTabReachesAccessibleWidgets`).

   Fixed on the way, both worth remembering: `automation.Server.Close` waited
   for connected clients, so an app hung on exit while anything was attached;
   and `FocusManager.Clear` ran before the draw function, so `Next()` on Tab —
   handled at the top of a `limoni.Run` callback — found an empty list and did
   nothing. Navigation now falls back to the previous frame's registrations.

4. **Grapheme clusters in widgets.** The engine is done: `core/grapheme`
   segments by UAX #29 (Unicode 17.0, tables generated by `gen.go` from the UCD,
   checked by `GraphemeBreakTest.txt`), `Cell` stores multi-code-point clusters
   as interned handles ≥ `cell.RuneClusterBase`, setup sends mode 2027, and the
   diff re-anchors the cursor after each cluster so terminals without 2027 do
   not shift the row. Widgets are converted: use `cell.StringWidth` for widths,
   `cell.Truncate` to cut, `setEllipsized`/`setClipped` to draw cut text without
   allocating, and `clusterBounds` for cursor movement. Markdown now wraps
   and draws by cluster (`WordWidths`, `appendClusters`); palette highlighting
   already did. Mode 2027 is still *set* unconditionally, but whether the
   terminal honours it (or draws clusters as units anyway) is now probed. To
   regenerate tables for a new Unicode version, download the UCD files listed
   in `gen.go`, run it, and replace the conformance test data.

5. **Terminal integration.** Desktop notifications are done:
   `Terminal.Notify`, `App.Notify`, `NotifyCmd` for Programs — OSC 99 for
   kitty, OSC 9 for iTerm2/WezTerm/Ghostty/foot, nothing for unknown terminals
   (`LIMONI_NOTIFY` overrides). A Program's notification goes through a
   channel to `RunTerminal`'s loop: writing it from Update raced the frame.
   Mouse pointer shapes are done: `ClickAction.Pointer` (a CSS cursor name)
   over an area, applied as the mouse moves with OSC 22 in kitty, foot and
   Ghostty, reset on Close and Suspend. A pointer-only ClickAction is kept out
   of the click regions, or it would swallow the click meant for the widget
   under it — SplitPane's divider found that. Supported terminals were taken
   from their docs, not from memory; check again before adding one.

   The window title (OSC 2) is done: `WithTitle`, pushed and popped
   with XTWINOPS so an app does not leave its name on the user's terminal.

   OSC 8 hyperlinks are done. The link lives in the *cell*, as a 16-bit
   handle into an interned URL table, in the two bytes `Style` was already
   padding with — `Cell` is still 16 bytes. The handle doubles as the OSC 8
   `id=`, which is what makes a link split across rows one link. It is
   capability-gated like REP, and widgets read `ctx.Hyperlinks` to decide
   whether the address still has to be printed as text.

   Two things that cost time and will again: TERM for kitty is
   *`xterm-kitty`* and for Ghostty *`xterm-ghostty`*, so a `HasPrefix`
   table silently finds neither — match anywhere in TERM. And `cell.Context`
   must stay at or under **128 bytes**: past that, a closure capturing it
   captures by reference rather than by value, which moves the whole Context
   to the heap. Adding one `bool` at the end of the struct cost 144 B/op in
   every widget with a fallback click closure; moved into the padding after
   `Style` it costs nothing. `TestContextStaysUnder128Bytes` guards it.

   Suspend/resume is done: `WithSuspend()` / `Terminal.Suspend()`. Restore the
   screen and termios *before* `SIGTSTP` — a test with a real pty checks that
   the shell does not get the terminal back in raw mode, and fails when the
   order is swapped. Verified end to end under an interactive bash: Ctrl+Z
   stops it, the shell works, `fg` repaints, keys still arrive. `stopSelf` is a
   variable so tests can stand in for the signal.

6. **Widget gaps — closed.** Gauge, LineGauge, StatusBar, SplitPane (drag via
   a handler built once per state, so no per-frame closure), CodeView (own
   lexer for Go/Python/JS/Rust/C/shell/JSON/YAML), BigText (public-domain
   font8x8, generated into `bigtext_font.go`), Calendar, Autocomplete and
   FilePicker all exist, each at 0 allocs/op with a benchmark. Close the
   matching `help wanted` issues.

   Viewer3D was not a `Widget` (no `SizeHint`) until now, and three of its
   bugs were found by drawing it to a PNG and looking: texture mode skipped
   the depth buffer and ignored the model's UVs, Gouraud used face normals
   (identical to Lambert), and lighting lit the side facing *away* from the
   light — `graphics.CalculateNormal` is inward for Limoni's front-face
   winding. `outwardNormal` negates it; the examples had the same bug. OBJ UVs
   are flipped to image space on load. `Pixels: true` renders to RGBA and uses
   the image protocol (~7 ms, 2.7 MB per moving frame with kitty).

7. **Canvas markers — done.** `Canvas.Marker`: Braille, sextant, quadrant,
   half block, block. Drawing stays at 2×4 dots and the marker is applied in
   `Draw`, so every drawing call works with every marker. The sextant table is
   checked against the Unicode character names, not typed from memory.

8. **Diff bandwidth — scroll regions and ICH/DCH done.** `core/buffer/scroll.go`:
   a row-hash search finds a band that moved and scrolls it with DECSTBM +
   SU/SD; a row with text inserted or deleted shifts with ICH/DCH. A log
   gaining a line: 1,189 → 125 bytes; a typed character: 77 → 11. Verified by
   replaying the output through a small VT model (`vt_test.go`, 400 random
   frames, mutation-checked) and in kitty with `script -f` + `get-text`.
   The published cross-framework runner does not enable them, so no number in
   `docs/benchmarks.md` includes them — rerun per the methodology before
   quoting a new ratio.

   Done in this area: the encoder emits `ECH`/`EL` for blank runs and `REP` for
   repeated glyphs, which took a full-screen redraw from 4,897 bytes to 377 and
   `resize` from 2,735 to 162. `REP` is capability-gated because a terminal
   without it prints the escape — on for recognised terminals and `LIMONI_REP=1`,
   automatic once the capability handshake (item 2) lands.

   Inline mode is done: `limoni.WithInline(height)` renders in a band of the
   normal screen buffer with no alternate screen, addressing every frame
   relative to the cursor because the application's first row moves whenever the
   terminal scrolls. Note `ESC[2J` is suppressed there — a full-screen clear in
   inline mode wipes the user's scrollback.

   Border merging is done: `Block.MergeBorders` unions the box-drawing segments
   already in the cell, so adjacent blocks meet in `┬ ┼ ├ ┤ ┴`. It costs a read
   per border cell (~4%) and stays at zero allocations. Only the light set is
   merged — heavy and double lines have no honest junction with light ones.

9. **Test coverage.** 72% across the library packages, up from 68%.
   awesome-go asks for 80% and will not consider the project before
   2027-01-06 anyway (they require five months of history).

   Raised so far: `core/cell` 39→92%, `core/engine` 55→80%, the root package
   49→73%, `component` 58→71%, `core/terminal` 65→70%. Writing them found a
   real bug (`RichText` read `"2 < 3 and 4 > 1"` as a tag and swallowed the
   comparison) and two behaviours that are correct but surprising enough to
   need saying: `Transform` sweeps blanks too, and `TextFn`'s getter runs more
   than once per frame.

   Still thin, and open as `testing` issues #35-#40 for contributors:
   `cmd/limoni` (36%), `benchmarks` (56%), `testkit` (59%),
   `compat/bubbletea` (60%), `core/accessibility` (65%), `core/driver` (69%),
   `graphics` (69%), `widgets` (70%).

   Two things to know before writing tests here. Assert the answer, not the
   line: a test that only reaches code is worth nothing when the code is
   wrong. And watch the Windows runner — `LastFrameDuration` was asserted
   `> 0`, which is true everywhere except Windows, where a sub-tick frame
   measures exactly zero.
