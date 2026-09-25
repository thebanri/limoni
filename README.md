<p align="center">
  <img src="assets/logo.png" alt="Limoni Logo" width="160" />
</p>

<h1 align="center">🍋 Limoni</h1>

<p align="center">
  <strong>A terminal UI engine for Go that tests can click, AI agents can drive,<br>and the garbage collector never sees.</strong>
</p>

<p align="center">
  <a href="https://github.com/thebanri/limoni/actions"><img src="https://img.shields.io/github/actions/workflow/status/thebanri/limoni/ci.yml?branch=main&style=flat-square&logo=github" alt="Build Status"></a>
  <a href="https://pkg.go.dev/github.com/thebanri/limoni"><img src="https://img.shields.io/badge/go.dev-reference-007d9c?style=flat-square&logo=go&logoColor=white" alt="Go Reference"></a>
  <a href="https://golang.org"><img src="https://img.shields.io/badge/go-%3E%3D%201.25-blue?style=flat-square&logo=go" alt="Go Version"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-Apache_2.0-blue?style=flat-square" alt="License"></a>
  <a href="docs/benchmarks.md"><img src="https://img.shields.io/badge/draw%20path-0_B%2Fop-brightgreen?style=flat-square" alt="Zero Allocations"></a>
</p>

<p align="center">
  <a href="README.md">English</a> • <a href="README_TR.md">Türkçe</a>
</p>

<p align="center">
  <strong><a href="https://thebanri.github.io/limoni/">▶ Try it in your browser</a></strong> — the same engine compiled to WebAssembly, no install.
</p>

<p align="center">
  <img src="assets/3d.gif" alt="Limoni rendering a shaded 3D model in the terminal" width="100%" />
</p>

---

## Quick start

```bash
go get github.com/thebanri/limoni
```

```go
package main

import "github.com/thebanri/limoni"

func main() {
	limoni.Run(func(f *limoni.Frame, ev *limoni.Event) bool {
		if ev != nil && ev.Type == limoni.EventKey && ev.Key.Type == limoni.KeyEsc {
			return false // quit
		}
		f.RenderComponent(limoni.Border(
			limoni.Center(limoni.Label("Hello from Limoni 🍋  (Esc quits)", limoni.Bold())),
			limoni.SymbolsRounded,
			limoni.Fg(limoni.Hex("#FFCC00")),
		), f.Area())
		return true
	})
}
```

Or generate a project that runs straight away, with a test already written:

```bash
go run github.com/thebanri/limoni/cmd/limoni@latest new myapp    # -template counter|dashboard|form|ssh
cd myapp && go mod tidy && go run .
go test ./...                                                     # a uitest test comes with every template
```

**Next:** [Getting started](docs/getting-started.md) · [Widget gallery](docs/widget-gallery.md) · [Examples](#examples)

---

## Why Limoni

### 1. Your TUI has a semantic tree, so tests and agents address widgets by name

Most terminal automation, such as [termwright](https://github.com/fcoury/termwright) or
[mcp-tui-test](https://github.com/GeorgePearse/mcp-tui-test), parses the rendered character
grid, so a test breaks when the layout moves by one column. A Limoni app builds a
semantic tree every frame (the same tree a screen reader uses), and the tools below
work on that tree instead.

**`uitest`: Playwright-style tests.** Checks wait instead of sleeping, and a failure
prints the whole tree of the last frame.

```go
page := uitest.Run(t, 80, 24, app.draw)            // in process: no terminal needed

page.GetByRole("input", "New task").Type("Tag v1.0")
page.GetByRole("button", "Add task").Click()
page.Expect(page.GetByRole("list-item", "").Within(page.GetByRole("list", "Tasks"))).ToHaveCount(3)
page.Expect(page.GetByID("status")).ToContainLabel("Added")
```

**`limoni-mcp`: let an AI agent drive the app.** It gives Claude Code, Cursor or any
MCP client eight tools (`tree`, `click`, `type_text`, `wait_for`…) that work on the
same tree. In one recorded run, Claude Code completed a release checklist in 17 tool
calls. That run included typing a deploy token the agent could not read back.

```bash
go run -tags limoni_debug ./examples/agent_checklist
claude mcp add limoni -- limoni-mcp -socket "$XDG_RUNTIME_DIR/limoni-checklist.sock"
```

The automation socket only exists in `-tags limoni_debug` builds. It is closed by
default, only the same user can connect, and secret fields are never exposed.
→ [Semantic automation, MCP and uitest](docs/automation.md)

### 2. Fast where you can feel it

- **Zero heap allocations per frame** for the renderer, every widget's drawing, and the click handling of the common input widgets, enforced in CI, so animations don't stutter from GC pauses. Widgets with drag or custom handlers (Table, Slider, Dialog, …) still allocate a closure each. [The details](docs/architecture.md#5-allocation-free-interactive-frames).
- **Few bytes per frame.** Blank runs become `ECH`/`EL` and repeats become `REP`: a full-screen redraw is **377 bytes**, and an idle app sends **nothing**. Bytes, not CPU, are what you feel over SSH.
- **Virtual tables and lists.** One million rows scroll at ~2.7 ms a frame, because only visible rows are touched.

| Measured on a Ryzen 5 5600, Go 1.27.1 | Latency | Allocations |
| :--- | ---: | ---: |
| Full-screen diff, 120×40, every cell changed | ~50 µs | 0 |
| 10% of the screen changed | ~24 µs | 0 |
| Nothing changed | ~2 ns | 0 |
| 100 layered blocks, drawn and diffed | ~68 µs | 0 |

Absolute numbers depend on the machine. The comparisons against Ratatui 0.30.2,
Ultraviolet and Bubble Tea v1.3.10, including which ratios are *not* meaningful, are in
[docs/benchmarks.md](docs/benchmarks.md) and the [methodology](docs/benchmark-methodology.md).
One earlier run showed a 4,700× lead; it came from a bug in the benchmark harness,
and the [methodology](docs/benchmark-methodology.md#the-4700-that-was-a-harness-bug) explains how it was caught.

### 3. Batteries that other TUI libraries leave to you

| | |
| :--- | :--- |
| 🕶️ **3D** | A software rasteriser for OBJ/STL/PLY/GLB with Lambert and Gouraud shading, drawn in terminal cells |
| 🖼️ **Images** | Kitty, Sixel, iTerm2 and half-block fallback |
| 📊 **Charts** | Braille line charts, bar charts, pie charts, sparklines |
| 📝 **Markdown** | GFM rendering with a scrollable reader |
| ♿ **Accessibility** | A semantic tree, a screen-reader line mode, `NO_COLOR`, high contrast, reduced motion |
| 🧬 **Unicode** | UAX #29 grapheme clusters (Unicode 17.0, all 766 conformance tests), so flags and emoji families take one cell |
| 🔗 **Hyperlinks** | OSC 8 links, in markdown or any style — and where the terminal cannot show them, the address is printed instead |
| 📃 **Inline mode** | Render in a band of the normal screen, like `gum`, with scrollback intact |
| ⏺️ **Session replay** | Record a session, replay it as a regression test ([docs](docs/session-recording.md)) |
| 🌐 **Everywhere** | Linux, macOS, BSD, Windows, WebAssembly in the browser, SSH sessions |

The only dependencies are `golang.org/x/sys` and `golang.org/x/crypto`.

<table>
  <tr>
    <td><img src="assets/treeview.gif" alt="TreeView with image preview" /></td>
    <td><img src="assets/chart.gif" alt="Charts" /></td>
  </tr>
  <tr>
    <td align="center"><code>go run ./examples/treeview</code></td>
    <td align="center"><code>go run ./examples/charts</code></td>
  </tr>
</table>

### Is Limoni the right choice?

**Pick Limoni if** you are writing a terminal app in Go and any of these matter:

- **You want to test the UI, or let an AI agent use it**, by widget role and label
  rather than by screen coordinates (`uitest`, `limoni-mcp`).
- **It redraws a lot**: dashboards, log viewers, monitoring, games, animation,
  anything over SSH. The draw path makes no garbage and sends few bytes.
- **It needs things other libraries leave to you**: 3D models, images, charts,
  markdown, a million-row table, screen-reader support, WebAssembly in the browser.
- **You want both styles in one library**: immediate mode (`limoni.Run`) for
  dashboards, the Elm architecture (`limoni.RunProgram`) for forms and wizards.
- **You want a small dependency tree**: `golang.org/x/sys` and `golang.org/x/crypto`.

**Pick something else if:**

- You need a **stable 1.0 API** today. Limoni is pre-1.0 (see [Status](#status)).
- You rely on the **Charm ecosystem** (Bubbles, Huh, Glamour, Wish) and its
  community. Bubble Tea is the larger and older project. If you already have a
  Bubble Tea app, `compat/bubbletea` runs your models on Limoni
  ([migration guide](docs/bubbletea-migration.md)), so you can try it without a rewrite.
- You are writing **Rust**: use Ratatui.
- The app is a **one-shot prompt** (a single question, a spinner): a small prompt
  library is less to learn.

A feature-by-feature table against Bubble Tea v1/v2 and Ratatui 0.30 is in
[docs/comparison.md](docs/comparison.md).

---

## Built with Limoni: zest

<p align="center"><img src="assets/zest.gif" alt="zest filtering a million-line log to billing errors, opening a line's details, then clearing the filter to show it in context" width="100%" /></p>

[**zest**](cmd/zest) is a log viewer and Limoni's flagship app. It follows files and pipes, colours by
level, and filters a million lines without stalling. A 67 MiB, 1,000,000-line log is on screen in about half
a second.

```bash
go install github.com/thebanri/limoni/cmd/zest@latest
zest -demo 1000000        # or: zest app.log, kubectl logs -f pod | zest
```

Or [try it in the browser](https://thebanri.github.io/limoni/): the "Logs · zest" scene.

### And: globe

[**globe**](apps/globe) is a world you can turn, search, zoom into and pin — and what a Limoni
application looks like from the outside, since it is a module of its own that depends on a
published Limoni and uses nothing but its public API.

```bash
go install github.com/thebanri/limoni/apps/globe@latest
globe -at Türkiye
```

`go install` puts the binary in `$(go env GOPATH)/bin` — `~/go/bin` unless you changed it —
which a fresh Go install does not add to your `PATH`. If `globe` is "command not found", add
that directory to `PATH` or run `~/go/bin/globe`.

Everything on it is in the semantic tree, so "find Turkey on the world map" is something an agent
can do over MCP: type into `search`, click the row, read `image#globe` back as
`value="39.3°N 34.5°E · zoom 2.6×"`. Nothing of it reaches you when you import Limoni — a
directory with its own `go.mod` is not part of the module around it.

### And a game: Lemon Hunt

<p align="center"><img src="assets/lemonhunt.png" alt="Lemon Hunt: a first-person view down a brick sewer, two rats coming closer, half a lemon floating on the right, and the lemon squirter in hand" width="100%" /></p>

[**Lemon Hunt**](apps/lemonhunt) is a short first-person game. Squirt the rats and find ten lemons in the sewer,
and the gate to the lair lifts on Ratatui, king of the rats, who has a crown, a health bar and cheese to throw.

```bash
go install github.com/thebanri/limoni/apps/lemonhunt@latest
lemonhunt
```

Or [play it in the browser](https://thebanri.github.io/limoni/?app=lemonhunt), sound included.

It is a raycaster drawn two pixels to a cell in half blocks, with lit textures, pixel-art rats, 3D half lemons
ray-traced per pixel, and sound synthesised at start-up. A frame allocates nothing, through the diff too. It
holds keys down with `WithKeyReleases`, so walking is smooth in kitty, Ghostty and WezTerm.

---

## Two ways to write an app

| | Immediate mode | Declarative (Elm architecture) |
| :--- | :--- | :--- |
| **Entry point** | `limoni.Run(func(f, ev) bool)` | `limoni.RunProgram(ctx, model)` |
| **State lives in** | your closure | a `limoni.Model` with `Init` / `Update` / `View` |
| **Best for** | dashboards, 3D, games, animation | forms, wizards, CRUD tools, async work |
| **Runtime gives you** | a redraw on every event | commands, cancellation, deterministic ordering, panic recovery, session recording |

Both use the same renderer and widgets and are available from the root package.
[`examples/counter`](examples/counter) is a complete declarative app in under 80 lines.
Coming from Bubble Tea? See the [migration guide](docs/bubbletea-migration.md).

---

## Widgets

| Category | Widgets |
| :--- | :--- |
| **Layout** | `VStack` / `HStack` / `ZStack`, `Flex`, `Border`, grid layout (`layout.GridLayout`), `Block` (with border merging), `SplitPane` (draggable), `Viewport`, `Dialog`, `Popup`, `StatusBar` |
| **Data** | `Table` (virtual), `List` (virtual), `TreeView`, `FilePicker`, `Calendar`, `Sparkline`, `ProgressBar`, `Gauge`, `LineGauge`, `RichText` |
| **Charts** | `LineChart`, `BarChart`, `PieChart` — Braille, sextant or quadrant markers |
| **Input** | `TextInput`, `Autocomplete`, `TextArea`, `Checkbox`, `RadioGroup`, `Select`, `Slider`, `ColorPicker` |
| **Navigation** | `Tabs`, `Scrollbar`, `CommandPalette`, fuzzy search, keybinding manager |
| **Feedback** | `Spinner`, `Toast`, desktop notifications (OSC 9 / OSC 99) |
| **Graphics** | `Canvas` (Braille, sextant, quadrant, block), 3D meshes as dots or as a picture over kitty/iTerm2/Sixel, `Image` |
| **Text** | `Markdown`, `CodeView` (syntax highlighting), `BigText`, `Label`, `Paragraph` |
| **Tooling** | `DevTools` HUD (`F12`), themes, validation |

→ [Widget gallery](docs/widget-gallery.md) · [Widget reference](docs/widgets-reference.md)

---

## Examples

| Example | What it shows |
| :--- | :--- |
| [`demo`](examples/demo) | The feature trailer: a 3D lemon in ASCII, Braille and half-blocks |
| [`xray`](examples/xray) | A night city under an X-ray: every cell the diff sends lights up, with measured bytes against a full repaint (`-film` for recording) |
| [`showcase`](examples/showcase) | Tabs, forms, matrix rain, 3D, command palette, DevTools (`F12`) |
| [`3d_viewer`](examples/3d_viewer) | OBJ/STL/PLY viewer with shading and orbit controls (`-fps 240`) |
| [`dashboard`](examples/dashboard) | Live CPU and memory sparklines, a process table, streaming logs |
| [`table_virtual`](examples/table_virtual) | A one-million-row table |
| [`agent_checklist`](examples/agent_checklist) | An app built to be driven by an AI agent, and tested with `uitest` |
| [`todo`](examples/todo) | A declarative todo app with tags, filters and fuzzy search |
| [`counter`](examples/counter) | The smallest declarative app |
| [`composable`](examples/composable) | Layout with `VStack`, `HStack`, `Border`, `Flex` |
| [`forms`](examples/forms) · [`layer_demo`](examples/layer_demo) · [`treeview`](examples/treeview) · [`charts`](examples/charts) | Inputs, modals, file tree, charts |
| [`ssh_server`](examples/ssh_server) · [`wasm`](examples/wasm) | Serving over SSH, running in the browser |

Run any of them with `go run ./examples/<name>`, or without cloning:
`go run github.com/thebanri/limoni/examples/3d_viewer@latest`.
All examples: [docs/examples.md](docs/examples.md).

---

## Documentation

| | |
| :--- | :--- |
| [Getting started](docs/getting-started.md) | Install, first app, both application models |
| [Architecture](docs/architecture.md) | The flat cell grid, the diff, why the draw path doesn't allocate |
| [Layout](docs/layout-guide.md) · [Widgets](docs/widgets-reference.md) · [Core API](docs/core-api.md) | Reference |
| [Semantic automation](docs/automation.md) | The automation socket, `limoni-mcp`, `uitest` and the security model |
| [Session recording](docs/session-recording.md) | Record and replay sessions as regression tests |
| [Graphics](docs/graphics-and-canvas.md) · [Animation](docs/animation-and-physics.md) · [Accessibility](docs/accessibility-and-theming.md) | Feature guides |
| [Drivers and platforms](docs/drivers-and-platforms.md) | Unix, Windows, WebAssembly, SSH |
| [How it compares](docs/comparison.md) | Against Bubble Tea v1/v2, Lip Gloss and Ratatui, with the caveats |
| [Benchmarks](docs/benchmarks.md) | Every measured number and how to reproduce it |
| [Rendering FAQ](docs/faq.md) | Hairline gaps, recommended terminals, emoji |
| [Stability](docs/stability.md) · [Changelog](CHANGELOG.md) | What may change before 1.0 |

Turkish documentation: [docs/tr](docs/tr/README.md).

---

## Status

Limoni is **pre-1.0**. Patch releases don't break the API; minor releases may, and
every break is listed in the [changelog](CHANGELOG.md). The core renderer, layout and
widgets are settling; the automation, `uitest` and session packages are new and
experimental. See [docs/stability.md](docs/stability.md) for what has to happen
before v1.0.

## Community and contributing

- **Questions and ideas:** [GitHub Discussions](https://github.com/thebanri/limoni/discussions)
- **Bugs:** [open an issue](https://github.com/thebanri/limoni/issues/new/choose). The template asks for your terminal emulator, since most rendering bugs depend on it.
- **First contribution:** issues labelled [`good first issue`](https://github.com/thebanri/limoni/labels/good%20first%20issue) are scoped to one file and say how to verify the change. Start with [CONTRIBUTING.md](CONTRIBUTING.md).
- **Built something?** Add it to [AWESOME.md](AWESOME.md).

AI assistants (Claude, Gemini) were used during development for scaffolding, tests
and documentation drafts. The architecture was designed, profiled and benchmarked
by the author, and the benchmark harness exists to check claims, from people or tools.

[Security policy](SECURITY.md) · [Code of Conduct](CODE_OF_CONDUCT.md) · Apache License 2.0
