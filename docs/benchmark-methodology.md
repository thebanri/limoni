# Benchmark Methodology

This document states exactly what Limoni's benchmark suite measures, what it does
not measure, and which comparison claims are currently supported by evidence.

Benchmarks published by a project about its own competitors are only as credible
as the methodology behind them. The purpose of this file is to make every number
in the README reproducible and every comparison auditable — including the places
where the comparison is currently out of date.

---

## 1. Current status at a glance

| Target | Version measured | Status |
| :--- | :--- | :--- |
| Limoni | this commit | ✅ Current |
| Bubble Tea | **v1.3.10** | ⚠️ **Stale.** Bubble Tea v2 rebuilt its renderer on [Ultraviolet](https://github.com/charmbracelet/ultraviolet). v1 numbers do not describe v2. |
| Bubble Tea v2 | — | ❌ **Not measured.** No runner links it. The renderer it is built on is measured instead; see the row below and [§2.4](#24-ultraviolet). |
| Ultraviolet | **v0.0.0-20260703014108** | ✅ **Measured** via `uv.TerminalScreen`. [§2.4](#24-ultraviolet); three runs, one machine. |
| Ratatui | **0.30.2** | ✅ **Measured** on a rebuilt runner. [§2.5](#25-ratatui); three runs, one machine. |

**Consequence:** any README statement comparing Limoni's rendering architecture to
"Bubble Tea" or "Lip Gloss" without a version qualifier should be read as
**"Bubble Tea v1 / Lip Gloss v1"**. Claims against v2 are *not* backed by the
numbers in this repository. The README tables are labelled accordingly.

Tracking work to close this gap is listed in [§6](#6-open-work).

---

## 2. What the suite consists of

There are three independent layers. They answer different questions and should
not be conflated.

### 2.1 Go micro-benchmarks (`go test -bench`)

Standard Go benchmarks run by `testing`. These produce the `ns/op`, `B/op` and
`allocs/op` figures in the README table.

```bash
# Buffer diff engine (dirty and clean passes)
go test ./core/buffer -run '^$' -bench . -benchmem

# Widget, layout and frame-level workloads
go test ./benchmarks -run '^$' -bench . -benchmem

# Individual widget draw paths
go test ./widgets -run '^$' -bench . -benchmem
```

These are **intra-project** measurements. They compare Limoni against itself
across commits. They involve no other framework and make no cross-framework
claim.

### 2.2 Cross-framework runners (`benchmarks/runners/`)

Three standalone programs execute the same named workloads and emit JSON:

| Runner | Path | Toolchain |
| :--- | :--- | :--- |
| Limoni | `benchmarks/runners/limoni` | this module |
| Bubble Tea | `benchmarks/runners/bubbletea` | separate Go module |
| Ratatui | `benchmarks/runners/ratatui` | Cargo crate |
| Ultraviolet | `benchmarks/runners/ultraviolet` | separate Go module (needs Go ≥ 1.25) |

> ⚠️ **These runners do not all measure the same pipeline.** The v1 Bubble Tea
> runner measures view-string construction only; the Limoni runner measures
> drawing *plus* the full ANSI diff. Any Limoni-vs-Bubble-Tea-v1 ratio therefore
> compares different amounts of work and should not be quoted as a like-for-like
> speedup. §2.4 explains what was done about it.

To reproduce the published comparison, use the script — it builds each runner,
executes all three three times in sequence, and prints the median p50 per
workload with its run-to-run spread, suppressing every ratio the runners mark
non-comparable:

```bash
./benchmarks/compare.sh
```

Individually:

```bash
go run ./benchmarks/runners/limoni -output benchmark-results/limoni.json

cd benchmarks/runners/bubbletea && go mod download && \
  go run . -output ../../../benchmark-results/bubbletea.json

cd benchmarks/runners/ratatui && \
  cargo run --release -- ../../../benchmark-results/ratatui.json

cd benchmarks/runners/ultraviolet && \
  go run . -output ../../../benchmark-results/ultraviolet.json
```

Each report carries a `comparable` flag per workload. A `false` there means a
ratio against that workload would mislead, and the reason is given in §2.4/§2.5.

### 2.3 Comparison dashboard

```bash
go run ./benchmarks/runners/dashboard \
  -output benchmark-results/dashboard.html \
  benchmark-results/limoni.json \
  benchmark-results/bubbletea.json \
  benchmark-results/ratatui.json
```

### 2.4 Ultraviolet

`benchmarks/runners/ultraviolet` measures `uv.TerminalScreen` from
`github.com/charmbracelet/ultraviolet` — the cell-based diffing layer that
Bubble Tea v2 and Lip Gloss v2 are both built on, and the layer a Bubble Tea v2
program actually drives. It exists because of a fairness problem in the v1
runner:

> **The v1 Bubble Tea runner and the Limoni runner do not measure the same
> pipeline.** The v1 runner measures `Model.Update` + `Model.View` — building the
> output string — and never invokes Bubble Tea's renderer, so diffing and ANSI
> encoding are excluded. The Limoni runner measures widgets drawing into a cell
> buffer **and** the full `buffer.Diff` that emits the escape sequences.
> Any Limoni-vs-v1 ratio therefore compares different amounts of work.

This runner closes that gap: cells in, diffed ANSI bytes out, colour profile
pinned to TrueColor on both sides, same workload manifest, same machine.

#### Why this is not labelled "Bubble Tea v2"

It measures what it links, and **it does not link Bubble Tea**. Nothing in the
runner imports `charm.land/bubbletea/v2`; the requirement that used to sit in
its `go.mod` was unreferenced, and `go mod tidy` removed it. Earlier revisions
of this document labelled these numbers "Bubble Tea v2 (v2.0.9)" — a version
that was never linked into the binary that produced them.

Ultraviolet is a fair proxy for v2's *rendering* cost and an unfair label for
v2 itself: a Bubble Tea program also pays for its runtime, message dispatch and
view construction, none of which is exercised here. So the comparison is
published under the name of the thing being measured. The report's `target`
field carries the exact module version, read from the binary's build info
rather than typed by hand.

Measured here: `github.com/charmbracelet/ultraviolet
v0.0.0-20260703014108-f5a850f9c2b7`.

#### Comparable workloads

Three runs per implementation, 1,000 frames each, run sequentially on one
machine. The figure is the median p50 across the three runs; ± is the full
spread, `(max − min) / median`.

| Workload | Limoni | Ultraviolet | ratio | Limoni bytes/frame | UV bytes/frame |
| :--- | ---: | ---: | ---: | ---: | ---: |
| `hundred-layers` | 164.0 µs ±0% | 320.2 µs ±4% | 2.0× | 124 | 11743 |
| `table-10000` | 162.9 µs ±0% | 337.2 µs ±2% | 2.1× | 298 | 1136 |
| `virtual-1000000` | 111.9 µs ±8% | 389.7 µs ±1% | 3.5× | 0 | 1888 |
| `full-redraw-120x40` | 120.8 µs ±0% | 492.7 µs ±1% | 4.1× | 4897 | 6329 |
| `resize` | 13.9 µs ±0% | 89.7 µs ±4% | 6.4× | 2735 | 8 |
| `single-cell-update` | 9.9 µs ±0% | 90.1 µs ±1% | 9.1× | 7 | 3 |
| `unicode-emoji` | 7.3 µs ±2% | 138.5 µs ±5% | 18.9× | 0 | 25 |
| `text-heavy-120x40` | 14.1 µs ±0% | 285.5 µs ±2% | 20.2× | 0 | 0 |

Byte counts were **bit-identical across all three runs** for every workload and
every implementation, so that column carries no spread.

#### Workloads that are *not* comparable

These are reported for completeness and should never be quoted as ratios.

| Workload | Limoni | Ultraviolet | Why it is not comparable |
| :--- | ---: | ---: | :--- |
| `empty-frame` | 0.0 µs ±50% | 87.7 µs ±1% | Limoni measures a clean-frame fast path. Ultraviolet has no equivalent because a Bubble Tea program does not render an unchanged frame at all — the cost shown is `TerminalScreen.Render`'s unconditional full-screen copy, which a real app would never pay here. The resulting ratio is four digits wide and means nothing. |
| `mouse-hit-test` | 0.8 µs ±1% | 87.3 µs ±4% | Limoni has spatial hit-testing; Ultraviolet has none. The UV figure is a one-line repaint, not the same algorithm. |
| `async-update-burst` | 0.1 µs ±33% | 89.2 µs ±1% | Limoni measures Elm-runtime message dispatch. Ultraviolet has no runtime; the figure is a one-line repaint. |
| `native-image-capability` | 9.6 µs ±1% | 65.4 µs ±1% | A capability comparison, per [§3](#workload-fairness-caveats): Limoni ships graphics protocols in-tree, Ultraviolet does not implement them at all. |

The only two measurements with meaningful run-to-run spread are Limoni's
`empty-frame` and `async-update-burst`, and both are sub-microsecond, where
timer granularity dominates. Neither is quoted as a ratio.

#### What these numbers do and do not show

- **They are one machine, one run.** No repetitions, no confidence intervals.
  Treat them as a baseline, not a published result.
- **Ultraviolet carries a ~88 µs floor on every workload.**
  `TerminalScreen.Render` copies the whole screen every call — an unconditional
  1,920-cell scan at 80×24 — where Limoni's diff short-circuits on a clean
  buffer. That is a real architectural difference and it accounts for most of
  the fixed cost in the UV column.
- **But per-frame cost matters less for a Bubble Tea app than the ratios
  suggest.** Such a program renders when its model changes, not on a timer, so
  an idle application does not pay that cost repeatedly. Limoni's immediate mode
  with `WithFPS` does render continuously, which is why its fast path exists.
- **Byte counts are the more interesting column.** Limoni emits fewer bytes per
  frame on `full-redraw` (4,897 vs 6,329), `table-10000` (298 vs 1,136) and
  dramatically fewer on `hundred-layers` (124 vs 11,743). Emitted bytes, not CPU
  time, is what determines responsiveness over SSH.
- **Allocation counts are not directly comparable** and are omitted above:
  Limoni's runner reuses buffers it owns, and this runner allocates a `Cell`
  per glyph because that is Ultraviolet's API shape.

#### Reproducing

```bash
cd benchmarks/runners/ultraviolet
go run . -output ../../../benchmark-results/ultraviolet.json
```

Requires Go ≥ 1.25.

### 2.5 Ratatui

`benchmarks/runners/ratatui` measures Ratatui 0.30.2 through
`CrosstermBackend` writing into an in-memory sink, behind a `Viewport::Fixed`
so no tty is required. The measured pipeline is cells in, diffed ANSI bytes
out — the same pipeline the Limoni runner measures, since `buffer.Diff` also
syncs its back buffer and both sides therefore diff against the frame they last
emitted.

#### What the previous runner got wrong

The 0.29 runner's numbers should not be compared against anything, including
its own successor. Four defects, in descending order of severity:

1. **`BytesPerFrame` was fabricated.** It summed `cell.symbol().len() + 10`
   over every cell of a `TestBackend` buffer — a constant 21,120 bytes for an
   empty 80×24 frame, which is not a quantity Ratatui ever emits. The column
   was never measuring output.
2. **Two workloads rendered nothing like their Limoni counterparts.**
   `full-redraw-120x40` drew one unchanging glyph, so after the first frame the
   diff had nothing to emit; Limoni's advances the glyph and the colours every
   step. `hundred-layers` drew `Block::default()` — borderless and titleless,
   i.e. no cells at all — at fixed positions, against Limoni's bordered, titled,
   moving blocks.
3. **Fixtures were built inside the timed region.** `table-10000` cloned a
   10,000-element `Vec<Row>` per frame and reported 80,049 allocations per
   frame with a p50 of 2.1 ms that was largely memcpy.
4. **`table-10000` did not scroll,** contrary to [§4](#4-measurement-rules).

#### Comparable workloads

Three runs per implementation, 1,000 frames each, run sequentially on one
machine, rustc 1.98.1. Median p50 across the three runs; ± is the full spread.

| Workload | Limoni | Ratatui 0.30.2 | ratio | Limoni bytes/frame | Ratatui bytes/frame |
| :--- | ---: | ---: | ---: | ---: | ---: |
| `hundred-layers` | 164.0 µs ±1% | **106.4 µs ±1%** | **0.6×** | 124 | 911 |
| `virtual-1000000` | 104.8 µs ±6% | 138.9 µs ±1% | 1.3× | 0 | 25 |
| `full-redraw-120x40` | 120.9 µs ±0% | 172.1 µs ±0% | 1.4× | 4897 | 5828 |
| `single-cell-update` | 9.9 µs ±0% | 15.6 µs ±3% | 1.6× | 7 | 32 |
| `unicode-emoji` | 7.3 µs ±1% | 28.2 µs ±2% | 3.8× | 0 | 25 |
| `text-heavy-120x40` | 14.2 µs ±0% | 63.9 µs ±3% | 4.5× | 0 | 25 |
| `resize` | 13.9 µs ±2% | 97.5 µs ±5% | 7.0× | 2735 | 1856 |

Byte counts were bit-identical across all three runs.

`hundred-layers` is the result the old harness was hiding: with both runners
drawing the same bordered, titled, moving blocks, **Ratatui is faster than
Limoni** on this workload. Limoni emits far fewer bytes for it (124 vs 911),
which is the trade the cell-grid architecture is supposed to make, but the
per-frame CPU cost is not currently in Limoni's favour here.

Ratatui carries a ~16 µs floor on every workload and never emits fewer than 25
bytes a frame: `Terminal::draw` resets the back buffer and the backend writes
cursor-visibility and positioning sequences unconditionally. That floor is
inherent to the engine, not to this harness.

#### Not comparable

| Workload | Limoni | Ratatui 0.30.2 | Why |
| :--- | ---: | ---: | :--- |
| `table-10000` | 162.8 µs ±1% | 2722.4 µs ±4% | Structural, not like-for-like. Limoni's `Table` holds its rows and redraws the visible window; Ratatui's `Table` takes ownership of a row iterator, so an application rebuilds all 10,000 rows every frame. That is idiomatic Ratatui, not a harness artifact — but it is a different amount of work. |
| `empty-frame` | 0.0 µs ±33% | 15.3 µs ±0% | Limoni has a clean-frame fast path keyed off a dirty flag; Ratatui rescans the buffer unconditionally. |
| `mouse-hit-test` | 0.8 µs ±25% | 19.8 µs ±0% | Limoni has spatial hit-testing; Ratatui has none. The Ratatui figure is a one-block repaint, not the same algorithm. |
| `async-update-burst` | 0.1 µs ±25% | 15.3 µs ±0% | Measures Limoni's Elm-runtime dispatch. Ratatui has no runtime to exercise. |
| `native-image-capability` | 9.6 µs ±0% | 19.8 µs ±0% | Limoni ships graphics protocols in-tree; Ratatui requires a third-party crate. |

#### 0.29 → 0.30 on the same harness

Holding the runner constant and changing only the Ratatui version, 0.30 is
faster nearly everywhere — `empty-frame` −40%, `single-cell-update` −39%,
`async-update-burst` −39%, `mouse-hit-test` −27%, `text-heavy` −14% — with byte
counts identical on every workload but `resize`, which grew from 306 to 1856
bytes per frame as 0.30 clears more on a viewport change.

#### A worked example of the harness-bug rule

An intermediate revision of this runner reported `resize` at 1106 µs on 0.30
against 57 µs on 0.29 — a 1807% regression, reproducible to ±3% across three
runs. It was withheld rather than published, on the rule that a number that
extreme is a harness bug until proven otherwise. It was a harness bug.

0.30 added `clear_fixed_viewport`, which calls `backend.size()` on every
viewport clear, and therefore on every resize. Through `CrosstermBackend` that
is an `ioctl` against the controlling terminal — roughly a millisecond when
something answers, and `EAGAIN` on CI, where no `TERM` is set. The runner had
been paying a terminal round-trip per frame and calling it Ratatui's cost. It
was caught by CI failing outright, not by the number looking wrong; the run had
been green locally, which is the failure mode `CLAUDE.md` warns about under
"compiling is not verifying".

With the size query answered from a fixed value — which is what an in-memory
benchmark must do — `resize` on 0.30 is 97.5 µs against 0.29's 57 µs. A 1.7×
increase, attributable to the extra clearing work 0.30 genuinely does, and now
published as a comparable workload. Reproducibility was no defence here: the
wrong number was stable to ±3% precisely because the terminal answered
consistently.

#### Reproducing

```bash
cd benchmarks/runners/ratatui
cargo run --release -- ../../../benchmark-results/ratatui.json
```

The report's `ratatui_version` field is stamped from `Cargo.lock` at build time
by `build.rs`, so the version label cannot drift from what was linked.

---

## 3. Workloads

The cross-framework runners implement these scenarios. A scenario is only
comparable across frameworks when all three runners implement it in the
idiomatic style of their own framework — not when one is hand-optimised.

| Workload | What it exercises |
| :--- | :--- |
| `empty-frame` | Clean-frame fast path: no cells mutated |
| `full-redraw-120x40` | Every cell in a 120×40 viewport mutated |
| `single-cell-update` | Minimal dirty region, worst case for diff overhead |
| `text-heavy-120x40` | 40 lines of wrapped text with styling |
| `unicode-emoji` | Wide characters and emoji in the cell grid |
| `table-10000` | Scrolling selection through a 10,000-row table |
| `virtual-1000000` | Virtual paging across 1,000,000 rows |
| `mouse-hit-test` | Spatial hit-testing across 100 click regions |
| `hundred-layers` | 100 overlapping widgets (Ratatui parity scenario) |
| `resize` | Viewport reallocation and reflow |
| `async-update-burst` | Elm-runtime message dispatch throughput |
| `native-image-capability` | Graphics-protocol encode path |

### Workload fairness caveats

Not every scenario has an equivalent in every framework, and where it does not,
the comparison is structural rather than like-for-like:

- **`mouse-hit-test`** — Limoni has built-in spatial hit-testing; Bubble Tea and
  Ratatui do not. The runners implement the closest equivalent the framework
  offers. This measures *"what does this cost in each framework"*, not
  *"whose implementation of the same algorithm is faster"*.
- **`virtual-1000000`** — depends on whether the framework provides virtual
  paging at all. Where it does not, the runner implements the naive approach a
  user would realistically write.
- **`native-image-capability`** — Limoni ships graphics protocols in-tree;
  the comparison targets require third-party crates/modules.

When reading the dashboard, treat these three as *capability* comparisons and the
rest as *performance* comparisons.

---

## 4. Measurement rules

The harness (`benchmarks/metrics.go`) follows these rules:

- **Warm-up.** `MeasureWorkloadWithWarmup` runs untimed iterations before
  recording, so JIT-free but cache-cold effects are excluded.
- **Persistent double buffer.** Diff benchmarks mutate a buffer that persists
  across iterations. The diff algorithm and the ANSI encoder both run end-to-end
  every iteration; there is no artificial `Clear()` that would let the diff
  short-circuit.
- **Real scrolling.** Table and virtual-scroll benchmarks advance the selection
  (`Select((i * 7) % N)`) rather than re-rendering a static window.
- **Real layering.** `hundred-layers` renders 100 overlapping `Block` widgets.
- **Allocation accounting.** `MeasureWorkloadWithStats` captures heap deltas via
  `runtime.MemStats`; Go benchmarks report `-benchmem` figures directly.
- **Repetitions.** The Go micro-benchmarks run under `-count=3` in CI. The
  cross-framework runners have no such flag — each executes the manifest once —
  so they are invoked three times in sequence and the published figure is the
  median p50 across runs, carrying the full spread. Single-run numbers are not
  published.

### What is *not* measured

These are real costs that the current suite does not capture. They are listed
here so the numbers are not over-read:

- **Terminal write bandwidth over a network.** Every runner writes to an
  in-memory sink, not to a PTY or an SSH channel. Byte counts are recorded, but
  no scenario measures end-to-end latency over a real link. This matters:
  emitted-byte volume, not CPU time, dominates TUI responsiveness over SSH.
- **Startup and teardown cost.** Raw-mode entry, capability detection and
  alternate-screen setup are excluded.
- **Steady-state memory footprint.** Only per-iteration allocation deltas are
  recorded, not resident set size.
- **Perceived smoothness.** No frame-pacing or jitter distribution is reported;
  the suite publishes means, not percentiles. A framework with a lower mean and
  a worse p99 would look better here than it feels.

---

## 5. Reproducing the published numbers

The README table was recorded on an AMD Ryzen / EPYC class CPU at a 120×40
viewport. To reproduce:

```bash
go test ./core/buffer ./benchmarks ./widgets -run '^$' -bench . -benchmem -count=3
```

Absolute figures are hardware- and terminal-dependent. **Ratios between
consecutive commits on the same machine are the meaningful signal; absolute
`ns/op` values copied from this table onto different hardware are not.**

CI enforces allocation budgets on the hot paths rather than latency thresholds,
because latency on shared CI runners is too noisy to gate on. See the
`Verify Zero-Allocation Hot Paths` step in `.github/workflows/ci.yml`.

---

## 6. Open work

To make the cross-framework comparison current again:

1. **Add a Bubble Tea v2 runner — a real one.** What exists today measures
   Ultraviolet, the renderer underneath v2, and is now labelled as such
   ([§2.4](#24-ultraviolet)). A genuine v2 runner would depend on
   `charm.land/bubbletea/v2` and drive `Model`/`Update`/`View` through the
   program loop, so the runtime and view construction are in the measurement
   too. Its API differs from v1 (`View()` returns a `tea.View` struct;
   `tea.KeyMsg` splits into `tea.KeyPressMsg`/`tea.KeyReleaseMsg`), so this is a
   port rather than a version bump. Keep the v1 runner for historical continuity
   and report all three columns.
2. ~~**Bump the Ratatui runner to 0.30.**~~ Done — see [§2.5](#25-ratatui). Most
   of the work was in the harness, which was measuring fabricated byte counts
   and two scenes that did not match their Limoni counterparts. The 0.30 backend
   changes did reach the runner in the end: driving `CrosstermBackend` directly
   compiled and ran locally but failed on CI with `EAGAIN`, because with no
   `TERM` set crossterm queries the terminal for the cursor position and waits
   for a reply that never arrives. It now wraps that in a `FixedBackend`
   implementing `Backend` with 0.30's associated `Error` type and
   `clear_region()`, answering size and cursor queries from fixed values.
   **Still open:** root-cause the `resize` regression flagged in §2.5 before any
   0.30 number for that workload is published.
3. **Add an emitted-bytes-per-frame column.** This is the metric that predicts
   SSH responsiveness and is currently the largest gap between what the suite
   measures and what users feel.
4. **Publish the hardware profile** (CPU model, Go version, OS, terminal) beside
   each result set in `benchmark-results/`, so runs from different machines are
   never silently compared.

Until items 1 and 2 land, every cross-framework claim in the README must carry an
explicit version label.
