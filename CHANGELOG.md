# Changelog

All notable changes to Limoni are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/) with the pre-1.0 rules described in
[docs/stability.md](docs/stability.md): a minor bump (`v0.x.0`) may break the API,
a patch bump (`v0.x.y`) does not.

## [Unreleased]

### Added
- OSC 8 hyperlinks: `limoni.Hyperlink(url)`, `Style.WithLink(url)`, and
  `widgets.Markdown` rendering `[text](url)` as a clickable link. The URL is
  interned and the cell keeps a 16-bit handle in the padding `Style` already
  had, so a link costs no memory and the draw path stays allocation-free.
  Capability-gated: terminals that cannot show links are not sent the
  sequence and are given the address as text instead. `LIMONI_HYPERLINKS`
  overrides, `limoni doctor` shows the decision.
- `cell.Context.Hyperlinks`, so a widget can render link markup to suit the
  terminal it is drawing into.
- `examples/hyperlinks`.

## [v0.7.0] — 2026-09-20

### Added
- `limoni.WithSuspend()` and `Terminal.Suspend()`: Ctrl+Z hands the terminal
  back to the shell and stops the application, as in vim or less; `fg` resumes
  it, re-runs the capability handshake and repaints. Unsupported backends
  (remote, browser, Windows) say so and deliver the key instead.
- `limoni.WithTitle()`, `Terminal.SetTitle`, `SaveTitle` and `RestoreTitle`:
  set the terminal's window title with OSC 2, with the previous title put back
  on exit. Control characters are stripped, so a title built from a file name
  cannot inject escapes. Contributed by
  [@team-humaki](https://github.com/team-humaki) (issue #16).

### Changed
- The comments in `graphics/` are in English (issue #10), thanks to
  [@chenzeyan54-commits](https://github.com/chenzeyan54-commits).
- `docs/tr/architecture.md`: the architecture guide in Turkish, contributed by
  [@Voyagerroc-Lab](https://github.com/Voyagerroc-Lab), with the
  allocation-free interactive frames section added.

## [v0.6.0] — 2026-09-20

### Added
- `TextInput` understands the readline keys: Ctrl+A/E move to the start and
  end, Ctrl+U deletes to the start, Ctrl+K to the end, Ctrl+W the previous
  word.
- `limoni new -template counter|dashboard|form|ssh`. Every template comes
  with a `main_test.go` written with `uitest`, and the scaffold's own test
  runs `go mod tidy`, `build`, `vet` and `test` on each of them.
- `widgets.Button` (also `limoni.Button`): a push button with a `button`
  node in the semantic tree. It does not allocate when drawn.
- `uitest`: `ToContainValue`, and `Page.ExpectExit`, which waits for a
  declarative program's Quit to go through its message loop.
- `limoni.LogView` and `limoni.LogViewState` re-exported from the root.
- zest's status line and key hints are in the semantic tree (`status`, `keys`),
  and the hints stay visible while typing a filter.

### Fixed
- `TextInput` and `TextArea` inserted keys held with Ctrl or Alt as text:
  Ctrl+U typed a "u". A headless Claude Code run against zest hit it trying to
  clear a filter.

## [v0.5.0] — 2026-09-20

### Added
- **zest** (`cmd/zest`), a log viewer and Limoni's flagship app. It follows
  files (surviving truncation) and pipes (keys through `/dev/tty`), detects
  levels in JSON, logfmt and plain text, filters in the background, and shows
  pretty-printed JSON details. A 1,000,000-line, 67 MiB log is on screen in
  0.54 s, measured in kitty.
  Clearing a filter keeps the found line selected and centred, so its
  context is right there.
- The browser playground has a "Logs · zest" scene: zest on a 200,000-line
  demo log, running as WebAssembly. The module grows from 1.25 MB to 1.70 MB
  gzipped, mostly `encoding/json` for the details pane.
- `widgets.LogView`: a virtual log pane with follow mode, a line-number
  gutter, level colours, case-insensitive highlighting and sideways scrolling,
  allocation-free (`BenchmarkLogViewDraw`: 100,000 lines, 0 B/op). With
  `LineNumberSource` a filtered view keeps the original line numbers.
- `Context.Describe`: a container that draws a child itself registers the
  child in the semantic tree with it.

### Fixed
- Widgets inside a `Block` (and inside component trees, via `AsComponent`)
  were missing from the semantic tree, so screen readers, tests and agents
  could not see most of a real application's content.
- `Block` built its child's context by copying fields one by one, so every
  field added to `Context` later never reached nested widgets. That included
  the click actions and wheel scrolling from v0.4.0, so nested checkboxes fell
  back to allocating closures and a nested scroll view lost the wheel.

## [v0.4.0] — 2026-09-19

### Added
- **Terminal capability handshake.** Setup asks the terminal for its name
  (XTVERSION), mode 2026/2027 support (DECRQM), Kitty keyboard support and DA1.
  It also *measures* whether REP works and how wide a grapheme cluster is
  drawn, by writing a few cells and asking where the cursor went. Answers
  refine the environment-based guess on the next frame, and force a full
  repaint if they arrive after one. Tested against kitty 0.48.2, Alacritty and
  Konsole 26.08.1. `LIMONI_PROBE=0` turns it off.
- `limoni doctor` prints what the terminal reported and which capabilities
  Limoni will use. The bug report template asks for it.
- `DiffOptions.ClusterWidths`: skip cursor re-anchoring after grapheme clusters
  on terminals measured to draw them as units.
- **`limoni.App`, `NewApp` and `RunWithContext`.** Immediate-mode
  applications can run several to a process (one per SSH session, say), each
  with its own terminal and wakeup, and stop when a context is cancelled.
  `limoni.Wakeup()` now wakes every running app. `examples/ssh_server` uses it
  instead of a hand-written loop.
- **Allocation-free interactive frames.** `cell.ClickAction` with
  `Context.RegisterClickAction`, and `Context.RegisterScroll`, register what a
  click or the mouse wheel does as data instead of as a closure built every
  frame. Checkbox, Radio, TextInput, TextArea, List, Paragraph, RichText,
  Markdown, Progress, Sparkline and Image use them. The frame also stopped
  wrapping every click handler in a second closure, and stopped building a
  theme closure for every widget. `BenchmarkInteractiveFrame` (a checkbox, an
  input, a list and a block through a real Terminal with a theme) went from 19
  allocations and 816 B per frame to zero, 4% faster, and CI gates it.
- **More of the screen is addressable.** Table rows (`row`, with a `cell` per
  column), TreeView items (`tree-item`, with expanded state) and tabs
  (`tab-list`/`tab`, via the new `Tabs.State`) are semantic nodes, so a test or
  an agent finds "the row reading beta" and clicks it.
- `uitest`: `Check`, `Uncheck` and `Select`, which click only when needed and
  wait for the result. `limoni-mcp`'s `click` takes `ensure` for the same.
- `cell.Truncate`: the longest prefix of a string that fits a column count,
  cut at grapheme cluster boundaries, without allocating.
- `CommandPalette.Title` and `CommandPalette.Placeholder`, and the
  `Shading*` constants for `Viewer3D.Shading`.

### Changed
- `Tabs` report the role `tab-list` instead of `list`.
- `CommandPalette` shows English text by default ("⌘ Commands",
  "Search commands..."). It used to show Turkish text to every user.
- `Validator` default messages are English, and `MinLength`/`MaxLength`
  count grapheme clusters, not code points.
- `Viewer3D.Shading` takes `ShadingTexture`, `ShadingFlat`, `ShadingLambert`,
  `ShadingWireframe` and `ShadingGouraud`. The Turkish names it shipped with
  still work.

### Fixed
- Replies to terminal queries were dropped or misread: a Kitty keyboard reply
  (`CSI ? flags u`) was parsed as a key press, and DA1 replies longer than 32
  bytes were cut off, so the rest arrived as keystrokes.
- Parsing a CSI sequence allocated. Keys, mouse reports and replies now parse
  without allocating.
- Widgets measured text in code points and cut it by rune. `TextInput`
  drew 日本 as blanks, split emoji sequences into their parts, and left
  "man ZWJ woman ZWJ" behind after one Backspace. `TextInput` and `TextArea`
  now edit by grapheme cluster. Table, Toast, Select, Popup, TextArea,
  CommandPalette, the charts, TreeView, List, Progress and VirtualDataView
  measure in columns and cut on cluster boundaries.
- TreeView guide lines vanished below an ancestor with later siblings:
  Flatten built each node's ancestry with an `append` that siblings shared, so
  a later sibling overwrote an earlier one's flags. It now uses a bit mask and
  a reused buffer, and no longer allocates per frame.
- A Table allocated its draw scratch and two maps after every garbage
  collection, because it kept them in a `sync.Pool`. A table with `State` now
  keeps them there.
- Table cells cut to fit could produce invalid UTF-8 (`"e\xcc"`) when the
  text contained a combining mark.
- `TextInput` and `CommandPalette` allocated on every frame, and Table
  allocated for every truncated cell. All three are now allocation-free, and
  CI checks every widget benchmark instead of four.

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

[Unreleased]: https://github.com/thebanri/limoni/compare/v0.7.0...HEAD
[v0.7.0]: https://github.com/thebanri/limoni/compare/v0.6.0...v0.7.0
[v0.6.0]: https://github.com/thebanri/limoni/compare/v0.5.0...v0.6.0
[v0.5.0]: https://github.com/thebanri/limoni/compare/v0.4.0...v0.5.0
[v0.4.0]: https://github.com/thebanri/limoni/compare/v0.3.0...v0.4.0
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
