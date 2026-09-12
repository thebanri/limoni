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

Declarative mode is context-aware. Immediate mode is not yet — see open work.

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
| `benchmarks` | Harness plus the cross-framework runners |

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

**`settings.json` once contained a live API key** and is gitignored for that reason.
Do not re-add it.

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

1. **Instance isolation and `RunWithContext`.** `limoni.Wakeup` writes to a
   package-level channel, so two Limoni applications cannot run in one process —
   yet `examples/ssh_server` exists and multi-session SSH is a stated target. Make
   the wakeup channel instance-bound and give immediate mode a context-aware entry
   point. Keep `limoni.Run` as a default-instance wrapper for compatibility.

2. **Terminal capability handshake.** `DetectCapabilities` only reads `TERM`,
   `COLORTERM` and `TERM_PROGRAM`, which is wrong inside tmux, over SSH with an
   unhelpful `TERM`, and in emulators that do not advertise themselves. Add a
   short, timeout-guarded probe at startup: DA1, XTVERSION, DECRQM for modes 2026
   and 2027, and the Kitty keyboard query — falling back to the current guess.
   `Terminal.SetCapabilities` already exists as the manual override.

3. **Grapheme clusters.** `cell.RuneWidth` / `StringWidth` walk runes with
   hand-written East Asian ranges. ZWJ emoji, regional-indicator flags, skin-tone
   modifiers and VS16 all measure wrong, which corrupts layout in any application
   rendering user content. Needs UAX #29 segmentation plus mode 2027 negotiation,
   and a decision about how `Cell` stores a multi-rune cluster.

4. **Diff bandwidth.** The diff emits cursor jumps and runs. Ultraviolet also uses
   `ECH`/`REP`/`ICH`/`DCH` and scroll-region optimisation, which is where its SSH
   bandwidth story comes from. Emitted bytes, not CPU time, govern responsiveness
   over a network link.

5. **Missing terminal integration.** No OSC 8 hyperlinks, no OSC 9/777
   notifications, no mouse shape, no window title, no suspend/resume, no inline
   (non-altscreen) render mode. Inline mode in particular is what tools like `gum`
   and CI progress renderers are built on.

6. **Remaining widget gaps.** FilePicker, Gauge/LineGauge, StatusBar, SplitPane,
   syntax-highlighted code view, log view, big text, calendar, autocomplete.

7. **Canvas markers.** Ratatui 0.30 added quadrant (2×2) and sextant (2×3) markers
   alongside Braille (2×4). Sextants help where Braille fonts are missing.

8. **Automatic border merging** between adjacent `Block`s. Ratatui 0.30 does this;
   a cell grid can do it far more easily than a string-based renderer, so it is a
   concrete demonstration of the architecture's advantage.
