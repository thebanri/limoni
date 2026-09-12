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
| Bubble Tea v2 | **v2.0.9** | ✅ **Measured** via `uv.TerminalScreen`. First baseline in [§2.4](#24-bubble-tea-v2--ultraviolet); one machine, one run, not yet a published result. |
| Ratatui | **0.30.2** | ✅ **Measured** on a rebuilt runner. First baseline in [§2.5](#25-ratatui); one machine, one run, not yet a published result. |

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
| Bubble Tea v2 | `benchmarks/runners/bubbletea-v2` | separate Go module (needs Go ≥ 1.25) |

> ⚠️ **These runners do not all measure the same pipeline.** The v1 Bubble Tea
> runner measures view-string construction only; the Limoni runner measures
> drawing *plus* the full ANSI diff. Any Limoni-vs-Bubble-Tea-v1 ratio therefore
> compares different amounts of work and should not be quoted as a like-for-like
> speedup. §2.4 explains the fix in progress.

```bash
go run ./benchmarks/runners/limoni -output benchmark-results/limoni.json

cd benchmarks/runners/bubbletea && go mod download && \
  go run . -output ../../../benchmark-results/bubbletea.json

cd benchmarks/runners/ratatui && \
  cargo run --release -- ../../../benchmark-results/ratatui.json
```

### 2.3 Comparison dashboard

```bash
go run ./benchmarks/runners/dashboard \
  -output benchmark-results/dashboard.html \
  benchmark-results/limoni.json \
  benchmark-results/bubbletea.json \
  benchmark-results/ratatui.json
```

### 2.4 Bubble Tea v2 / Ultraviolet

`benchmarks/runners/bubbletea-v2` measures Bubble Tea v2 (v2.0.9) through
`uv.TerminalScreen` — the layer Bubble Tea v2 actually drives. It exists because
of a fairness problem in the v1 runner:

> **The v1 Bubble Tea runner and the Limoni runner do not measure the same
> pipeline.** The v1 runner measures `Model.Update` + `Model.View` — building the
> output string — and never invokes Bubble Tea's renderer, so diffing and ANSI
> encoding are excluded. The Limoni runner measures widgets drawing into a cell
> buffer **and** the full `buffer.Diff` that emits the escape sequences.
> Any Limoni-vs-v1 ratio therefore compares different amounts of work.

The v2 runner closes that gap: cells in, diffed ANSI bytes out, colour profile
pinned to TrueColor on both sides, same workload manifest, same machine, same Go
toolchain (1.25.1).

#### Comparable workloads

Both engines compose a frame and let their own diff decide what to emit.

| Workload | Limoni | Bubble Tea v2 | ratio | Limoni bytes/frame | v2 bytes/frame |
| :--- | ---: | ---: | ---: | ---: | ---: |
| `full-redraw-120x40` | 148.0 µs | 702.1 µs | 4.7× | 4897 | 6329 |
| `table-10000` | 202.4 µs | 538.5 µs | 2.7× | 298 | 1136 |
| `virtual-1000000` | 124.4 µs | 635.2 µs | 5.1× | 0 | 1888 |
| `hundred-layers` | 220.9 µs | 439.8 µs | 2.0× | 124 | 11743 |
| `native-image-capability` | 10.4 µs | 94.8 µs | 9.1× | 0 | 0 |
| `single-cell-update` | 11.5 µs | 150.5 µs | 13.0× | 7 | 3 |
| `resize` | 15.8 µs | 142.4 µs | 9.0× | 2735 | 8 |
| `text-heavy-120x40` | 18.2 µs | 540.3 µs | 29.7× | 0 | 0 |
| `unicode-emoji` | 9.4 µs | 220.0 µs | 23.3× | 0 | 0 |

#### Workloads that are *not* comparable

These are reported for completeness and should never be quoted as ratios.

| Workload | Limoni | Bubble Tea v2 | Why it is not comparable |
| :--- | ---: | ---: | :--- |
| `empty-frame` | 0.0 µs | 138.0 µs | Limoni measures a clean-frame fast path. v2 has no equivalent because a Bubble Tea program does not render an unchanged frame at all — the cost shown is `TerminalScreen.Render`'s unconditional full-screen copy, which a real app would never pay here. |
| `mouse-hit-test` | 0.8 µs | 132.8 µs | Limoni has spatial hit-testing; Ultraviolet has none. The v2 figure is a one-line repaint, not the same algorithm. |
| `async-update-burst` | 0.6 µs | 142.3 µs | Limoni measures Elm-runtime message dispatch. The v2 figure is a one-line repaint; the runtimes are not exercised the same way. |

#### What these numbers do and do not show

- **They are one machine, one run.** No repetitions, no confidence intervals.
  Treat them as a first baseline, not a published result.
- **`TerminalScreen.Render` copies the whole screen every call** — an
  unconditional 4,800-cell scan at 120×40 — where Limoni's diff short-circuits
  on a clean buffer. That is a real architectural difference and it accounts for
  most of the fixed ~140 µs floor in the v2 column.
- **But per-frame cost matters less for v2 than the ratios suggest.** A Bubble
  Tea program renders when its model changes, not on a timer, so an idle
  application does not pay that cost repeatedly. Limoni's immediate mode with
  `WithFPS` does render continuously, which is why its fast path exists.
- **Byte counts are the more interesting column.** Limoni emits fewer bytes per
  frame on `full-redraw` (4,897 vs 6,329), `table-10000` (298 vs 1,136) and
  dramatically fewer on `hundred-layers` (124 vs 11,743). Emitted bytes, not CPU
  time, is what determines responsiveness over SSH.
- **Allocation counts are not directly comparable** and are omitted above:
  Limoni's runner reuses buffers it owns, and the v2 runner allocates a `Cell`
  per glyph because that is Ultraviolet's API shape.

#### Reproducing

```bash
cd benchmarks/runners/bubbletea-v2
go run . -output ../../../benchmark-results/bubbletea-v2.json
```

Requires Go ≥ 1.25 (Bubble Tea v2's own floor) and a reachable module proxy for
the `charm.land` vanity path.

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

One machine, one run, 1,000 frames each, AMD Ryzen class CPU, rustc 1.98.1.

| Workload | Limoni | Ratatui 0.30.2 | ratio | Limoni bytes/frame | Ratatui bytes/frame |
| :--- | ---: | ---: | ---: | ---: | ---: |
| `full-redraw-120x40` | 121.0 µs | 175.1 µs | 1.4× | 4897 | 5828 |
| `single-cell-update` | 9.9 µs | 15.9 µs | 1.6× | 7 | 32 |
| `text-heavy-120x40` | 14.2 µs | 65.2 µs | 4.6× | 0 | 25 |
| `unicode-emoji` | 7.3 µs | 28.7 µs | 3.9× | 0 | 25 |
| `virtual-1000000` | 103.6 µs | 141.8 µs | 1.4× | 0 | 25 |
| `hundred-layers` | 164.0 µs | **103.9 µs** | **0.6×** | 124 | 911 |

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
| `table-10000` | 163.0 µs | 2741.6 µs | Structural, not like-for-like. Limoni's `Table` holds its rows and redraws the visible window; Ratatui's `Table` takes ownership of a row iterator, so an application rebuilds all 10,000 rows every frame. That is idiomatic Ratatui, not a harness artifact — but it is a different amount of work. |
| `empty-frame` | 0.02 µs | 15.7 µs | Limoni has a clean-frame fast path keyed off a dirty flag; Ratatui rescans the buffer unconditionally. |
| `mouse-hit-test` | 0.8 µs | 20.2 µs | Limoni has spatial hit-testing; Ratatui has none. The Ratatui figure is a one-block repaint, not the same algorithm. |
| `async-update-burst` | 0.1 µs | 15.8 µs | Measures Limoni's Elm-runtime dispatch. Ratatui has no runtime to exercise. |
| `native-image-capability` | 9.6 µs | 20.2 µs | Limoni ships graphics protocols in-tree; Ratatui requires a third-party crate. |

#### 0.29 → 0.30 on the same harness

Holding the runner constant and changing only the Ratatui version, 0.30 is
faster nearly everywhere — `empty-frame` −40%, `single-cell-update` −39%,
`async-update-burst` −39%, `mouse-hit-test` −27%, `text-heavy` −14% — with byte
counts identical on every workload but one.

> ⚠️ **`resize` regressed by 1807%** (57.5 µs → 1095.9 µs) and its output grew
> from 306 to 1856 bytes per frame, with allocations going from 1 to 13.5 per
> frame. **This is not published as a Ratatui result.** Per the rule in
> `CLAUDE.md`, a number this extreme is a harness bug until proven otherwise;
> the scene alternates viewport size every frame, which is pathological, and
> 0.30's backend `clear_region()` requirement plausibly changed what a resize
> emits. It needs a root-cause pass before it means anything.

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
- **Repetitions.** CI runs `-count=3`. Single-run numbers are not published.

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

1. **Add a Bubble Tea v2 runner.** Create `benchmarks/runners/bubbletea-v2` as a
   separate module depending on `charm.land/bubbletea/v2`. v2 requires Go ≥ 1.25
   and its API differs from v1 (`View()` returns a `tea.View` struct;
   `tea.KeyMsg` splits into `tea.KeyPressMsg`/`tea.KeyReleaseMsg`), so this is a
   port rather than a version bump. Keep the v1 runner for historical continuity
   and report both columns.
2. ~~**Bump the Ratatui runner to 0.30.**~~ Done — see [§2.5](#25-ratatui). The
   breaking changes turned out not to reach this runner: it drives
   `CrosstermBackend` rather than a custom backend, so the new associated
   `Error` type and `clear_region()` requirement do not apply, and it uses no
   `Flex`. The runner compiled against 0.30.2 unchanged. What did need work was
   the harness, which was measuring fabricated byte counts and two scenes that
   did not match their Limoni counterparts.
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
