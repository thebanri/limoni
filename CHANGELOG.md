# Changelog

All notable changes to Limoni are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/) with the pre-1.0 rules described in
[docs/stability.md](docs/stability.md): a minor bump (`v0.x.0`) may break the API,
a patch bump (`v0.x.y`) does not.

## [Unreleased]

## [v0.3.0] — 2026-09-19

### Breaking
- `widgets.ListState` is no longer comparable with `==`: it now owns the buffer
  that holds the list's semantic row nodes. Compare the fields you care about
  (`Selected`, `Offset`) instead.
- The module requires **Go 1.25** (the floor `golang.org/x/sys` sets). It had
  declared `go 1.26.5`, which made it uninstallable on older toolchains.

### Added
- **Declarative runtime in the root package**: `limoni.Model`, `Msg`, `Cmd`,
  `UpdateResult`, `NewProgram`, `RunProgram` and the runtime message types, so a
  declarative application no longer imports `core/engine`.
- **Agent surface**:
  - `WithAutomation` serves the semantic tree over a Unix socket, behind the
    `limoni_debug` build tag and a closed-by-default `AutomationPolicy`. It verifies
    the connecting user with the kernel.
  - `cmd/limoni-mcp`: a Model Context Protocol bridge so AI agents such as
    Claude Code can drive a running Limoni app by selector.
  - `uitest`: Playwright-style locators and waiting assertions, which run in
    process (`Run`, `Program`) or against a running binary (`Connect`). Includes
    `WithSlowMo` and an action log.
  - Lists expose their visible rows as `list-item` nodes.
- **Session recording** (`session`): record a declarative app at `Update` and
  replay it frame by frame as a regression test. `tools/limonivet` flags
  non-deterministic `Update`/`View` code.
- **Grapheme clusters**: one UAX #29 cluster per cell (Unicode 17.0, all 766
  official break tests pass), mode 2027, and cursor re-anchoring for terminals
  without it.
- **Inline mode**: `WithInline(height)` renders in a band of the normal screen
  buffer and keeps scrollback intact.
- **Widgets**: `Viewport`, `Scrollbar`, `Tabs`, `Spinner`. `Block.MergeBorders`
  joins adjacent borders into `┬ ┼ ├ ┤ ┴`.
- **WebAssembly playground** at <https://thebanri.github.io/limoni/>.
- A Bubble Tea v2 / Ultraviolet benchmark runner with a documented baseline.

### Changed
- The diff emits `ECH`/`EL` for blank runs and `REP` for repeated glyphs. A
  full-screen redraw went from 4,897 bytes to 377. `REP` is capability-gated.
- `cell.RuneWidth` answers from a lookup table, which made the draw path roughly
  2–3× faster.
- Benchmark tables were corrected to figures that reproduce. See
  [docs/benchmark-methodology.md](docs/benchmark-methodology.md).

### Fixed
- An idle application sent an empty synchronized-update pair every frame
  (960 bytes/s at 60 FPS). A frame that changes nothing now writes nothing.
- `limoni new` generated a `go.mod` requiring Go 1.26.5 and code against
  `core/engine`. It now targets Go 1.25 and the root package, and a test builds
  the generated project.
- The WebAssembly demo called `Program.Run` and rendered nothing; it also ran
  in 16 colours because capability detection read absent environment variables.
- Tab did nothing at the top of a `limoni.Run` callback because the focus list
  was cleared before the draw.
- `automation.Server.Close` waited for connected clients, so the app hung on exit.
- The virtual viewport cache was not invalidated when the query changed.
- The dashboard example now builds on FreeBSD, OpenBSD and NetBSD.

### Security
- Secret `TextInput` values never reach the cell buffer, the semantic tree or
  the automation socket, and selectors resolve against the redacted tree.

## [v0.2.7] — 2026-09-12
### Fixed
- Demo: Esc no longer opens the exit dialog, modal layers are isolated, and
  arrow navigation works in the 3D view.

## [v0.2.6] — 2026-09-12
### Fixed
- Demo and `ascii3d`: embed the 3D mascot and images instead of reading a large
  web asset path, and add a level-of-detail safeguard.

## [v0.2.5] — 2026-09-12
### Added
- Kitty keyboard protocol (`CSI u`), multiline `TextInput` with selection, and
  `CONTRIBUTING.md`.

## [v0.2.4] — 2026-09-10
### Added
- `widgets.Label`.
### Fixed
- Modal and text background bleed in the 3D viewer, and z-buffered textures.

## [v0.2.3] — 2026-09-10
### Added
- A 240 FPS mode, the `-fps` flag and dynamic frame-rate controls.

## [v0.2.2] — 2026-09-10
### Added
- Adaptive flush: frames with high churn switch from sparse diffing to a
  synchronized full-stream redraw.
### Changed
- Half-block rendering standardised on `▄` (U+2584) to avoid gaps from line height.

## [v0.2.1] — 2026-09-10
### Added
- `limoni.Wakeup` for asynchronous render triggers.

## [v0.2.0] — 2026-09-10
### Breaking
- Packages reorganised: `core/runtime` → `core/engine`, `core/backend` → `core/driver`.
- Licence changed from MIT to Apache 2.0.
### Added
- The `component` package: `VStack`, `HStack`, `ZStack`, `Border`, `Flex`,
  `Spacer`, `Overlay`, `Transform` and more, with a zero-allocation stack solver.

## [v0.1.9] — 2026-09-09
### Fixed
- Emoji widths, border tearing when dragging a modal, cancellation precedence in
  the runtime, dual button focus in `Dialog`.

## v0.1.0 – v0.1.8
See the [GitHub releases](https://github.com/thebanri/limoni/releases).

[Unreleased]: https://github.com/thebanri/limoni/compare/v0.3.0...HEAD
[v0.3.0]: https://github.com/thebanri/limoni/compare/v0.2.7...v0.3.0
[v0.2.7]: https://github.com/thebanri/limoni/compare/v0.2.6...v0.2.7
[v0.2.6]: https://github.com/thebanri/limoni/compare/v0.2.5...v0.2.6
[v0.2.5]: https://github.com/thebanri/limoni/compare/v0.2.4...v0.2.5
[v0.2.4]: https://github.com/thebanri/limoni/compare/v0.2.3...v0.2.4
[v0.2.3]: https://github.com/thebanri/limoni/compare/v0.2.2...v0.2.3
[v0.2.2]: https://github.com/thebanri/limoni/compare/v0.2.1...v0.2.2
[v0.2.1]: https://github.com/thebanri/limoni/compare/v0.2.0...v0.2.1
[v0.2.0]: https://github.com/thebanri/limoni/compare/v0.1.9...v0.2.0
[v0.1.9]: https://github.com/thebanri/limoni/compare/v0.1.8...v0.1.9
