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
| Bubble Tea v2 | **v2.0.9** | 🚧 **Runner exists, results withheld.** See [§2.4](#24-the-v2-runner-and-why-its-numbers-are-not-published). |
| Ratatui | **0.29** | ⚠️ **Stale.** Ratatui 0.30 restructured into `ratatui-core`/`ratatui-widgets` and enabled the layout cache by default. |

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
| Bubble Tea v2 | `benchmarks/runners/bubbletea-v2` | separate Go module (results withheld — see §2.4) |

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

### 2.4 The v2 runner, and why its numbers are not published

`benchmarks/runners/bubbletea-v2` builds and runs against real Bubble Tea v2 and
Ultraviolet. It exists because of a fairness problem in the v1 runner:

> **The v1 Bubble Tea runner and the Limoni runner do not measure the same
> pipeline.** The v1 runner measures `Model.Update` + `Model.View` — building the
> output string — and never invokes Bubble Tea's renderer, so diffing and ANSI
> encoding are excluded. The Limoni runner measures widgets drawing into a cell
> buffer **and** the full `buffer.Diff` that emits the escape sequences.
> Comparing the two compares different amounts of work.

The v2 runner drives Ultraviolet's `TerminalRenderer` directly, so the measured
pipeline is cells in, diffed ANSI bytes out — the same shape as Limoni's.

Its results are nevertheless **not published**, because two harness questions
are unresolved:

1. **Touch tracking.** Ultraviolet's renderer early-returns when no line is
   marked touched, and those marks are reset by `TerminalScreen` — a layer this
   harness bypasses. Resetting them by hand takes an unchanged frame from
   ~195 µs to ~42 ns, the same order as Limoni's clean-frame fast path. The
   first version of this runner did not reset them, and therefore overstated
   v2's cost by roughly four orders of magnitude on the sparse workloads.
2. **`Clear()` semantics.** The harness clears the buffer every frame, mirroring
   the Limoni runner. Limoni's `Buffer.Clear` short-circuits via a clean flag;
   Ultraviolet's does not, so clearing re-dirties every line and defeats the
   early return. Whether that is a real architectural difference or an artifact
   of driving `TerminalRenderer` instead of `TerminalScreen` is not established
   — a Bubble Tea v2 application never clears a `RenderBuffer` by hand.

Resolving this most likely means rewriting the harness against
`uv.TerminalScreen`, the layer Bubble Tea v2 actually drives. Until that is
done, no number from this runner belongs in the README or in any comparison.

This is written down rather than quietly fixed because the failure mode is
instructive: the first run of this harness produced a 4,700× advantage for
Limoni on the empty-frame workload. That number was an artifact of the harness,
not a property of either library.

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
2. **Bump the Ratatui runner to 0.30.** Note the breaking changes: backends now
   require an associated `Error` type and a `clear_region()` method, and
   `Flex::SpaceAround` changed meaning (the old behaviour is now
   `Flex::SpaceEvenly`). The layout cache is enabled by default in 0.30 and will
   shift results.
3. **Add an emitted-bytes-per-frame column.** This is the metric that predicts
   SSH responsiveness and is currently the largest gap between what the suite
   measures and what users feel.
4. **Publish the hardware profile** (CPU model, Go version, OS, terminal) beside
   each result set in `benchmark-results/`, so runs from different machines are
   never silently compared.

Until items 1 and 2 land, every cross-framework claim in the README must carry an
explicit version label.
