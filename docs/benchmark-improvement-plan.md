# Limoni `benchmark-results` Directory — Implementation Plan

> This document establishes the guidelines for making benchmark artifacts in `/home/thebanri/Projects/Limoni/benchmark-results` dependable, reproducible, and verifiable by automated workflows and coding agents.

## 1. Scope and Current Status

This plan covers the results directory, artifact generation pipeline, dashboard accuracy, and CI regression checks. Production code optimizations should only be performed once result artifacts are trustworthy and reproducible.

Existing files:

- `README.txt`
- `go-benchmark.txt`
- `limoni.json`
- `bubbletea.json`
- `ratatui.json`
- `dashboard.html`
- `limoni-baseline.json`

### Critical Issue

The `/home/thebanri/Projects/Limoni/benchmark-results/limoni-baseline.json` file must not be 0 bytes or invalid JSON. A valid baseline or fallback must be in place so CI regression checks do not fail prematurely.

## 2. Interpretation of Current Benchmark Results

### Strong Limoni Workloads

- `text-heavy-120x40`: ~16.6 µs p50
- `unicode-emoji`: ~7.3 µs p50
- `table-10000`: ~90 µs p50
- `async-update-burst`: ~190 ns p50
- `mouse-hit-test` and `hundred-layers`

### Priority Workloads

| Workload | Limoni p50 | Priority | Notes |
|---|---:|---:|---|
| `empty-frame` | 2.67 µs | P1 | Verify whether Bubble Tea's 50 ns measures equivalent scope |
| `single-cell-update` | 8.45 µs | P1 | Ensure dirty-region optimization is visible |
| `full-redraw-120x40` | 29.64 µs | P2 | Profile buffer fill, diffing, and ANSI encoding separately |
| `virtual-1000000` | ~34.46 µs | P1 | Investigate performance relative to baseline |
| `resize` | 21.67 µs | P2 | Distinguish message-only handling from full reflow/redraw |

### Comparison Caveats

Even when all three runners report the same `manifest_hash`, identically named workloads may execute different code paths:

- Limoni benchmarks the real contiguous buffer, widget layout, and differential engine.
- Bubble Tea benchmarks model updates and string generation.
- Ratatui executes framework-specific buffer and table primitives.
- If a runner reports `0` allocations due to lack of allocation profiling, it must not be conflated with zero-allocation memory guarantees.
- `native-image-capability`, `mouse-hit-test`, `resize`, and `async-update-burst` should be reported with implementation-specific annotations.

The dashboard should present workload-specific comparison statuses rather than an unconditional `VALID COMPARISON` banner.

## 3. Phase P0 — Artifact Hygiene and Baseline Standards

**Goal:** Ensure the results directory contains only valid, documented, and reproducible data.

### Target Files

- `/home/thebanri/Projects/Limoni/benchmark-results/limoni-baseline.json`
- `/home/thebanri/Projects/Limoni/benchmark-results/README.txt`
- `/home/thebanri/Projects/Limoni/.github/workflows/benchmarks.yml`

### Tasks

- [ ] Ensure `limoni-baseline.json` is non-empty and contains valid JSON.
- [ ] Align baseline schema with current runner JSON output.
- [ ] Include commit hash, manifest hash, timestamp, OS, architecture, CPU, runtime/compiler, runner version, warmup count, and iterations in metadata.
- [ ] Update `README.txt` describing the role of each artifact.
- [ ] Reject missing or corrupted JSON files during CI validation.

### Acceptance Criteria

- All files in `benchmark-results/*.json` are valid JSON.
- No empty or unparseable baseline files exist.
- Baselines and current results adhere to identical schemas.
- Every artifact reflects its generating commit, manifest, and host environment.

## 4. Phase P0 — JSON Schema and Directory Validation

**Goal:** Prevent malformed or mismatched benchmark reports from reaching the dashboard.

### Mandatory Fields and Checks

- `implementation`, `environment`, `valid`, and `workloads` must be present.
- `manifest_hash`, `git_commit`, and `runner_version` must be defined.
- Workload count must equal 12.
- Workload spec definitions must be identical across all runners.
- `Frames >= Iterations`.
- `P50 <= P95 <= P99`.
- Summary metrics must be non-negative.

### Tasks

- [ ] Implement JSON schema or Go validation routines.
- [ ] Verify workload name, dimensions, row counts, Unicode flags, full_draw, mouse, async_burst, output mode, color mode, and iterations.
- [ ] Fail CI if runner manifest hashes do not match.
- [ ] Flag CPU, OS, architecture, runtime, or build mode discrepancies during baseline comparisons.
- [ ] Emit structured validation errors identifying the offending file and workload.

### Acceptance Criteria

- Missing metadata causes immediate CI failure.
- Differing workload counts or orders fail validation.
- Mismatched spec fields generate actionable error messages.
- Invalid JSON is rejected before dashboard generation begins.

## 5. Phase P1 — Dashboard Comparison Accuracy

### Recommended Status Values

- `VALID`: Equivalent workload and comparable measurement scope.
- `WARNING`: Implementation-specific workload or divergent execution path.
- `INVALID`: Mismatched workload specification.

### Tasks

- [ ] Replace global `VALID COMPARISON` banner with granular per-workload badges.
- [ ] Tag `mouse-hit-test`, `hundred-layers`, `resize`, `async-update-burst`, and `native-image-capability` as implementation-specific.
- [ ] Record measurement scope per workload in each runner.
- [ ] Display `N/A` or `not_measured` instead of `0` when allocations are not profiled.
- [ ] Display manifest hash, commit, CPU, runtime, runner version, warmup, and iterations in the dashboard header.
- [ ] Validate consistency between JSON data and HTML dashboard output.
- [ ] Display warning explanations inline on the corresponding workload row.

Recommended scope descriptors: `render-only`, `buffer-write`, `diff-only`, `output-encode`, `model-update`, `layout`, `input-routing`, `runtime`, `allocation`.

### Acceptance Criteria

- Only strictly equivalent workloads receive a `VALID` badge.
- Implementation-specific workloads are grouped and annotated.
- Unmeasured metrics are clearly distinguished from zero-overhead runs.
- Dashboard metadata matches source JSON files.

## 6. Phase P1 — Baseline and Regression Analysis

### Calculated Metrics

- `current_p50_ns`, `baseline_p50_ns`, `p50_delta_ns`, `p50_delta_percent`
- `current_p95_ns`, `baseline_p95_ns`, `p95_delta_ns`, `p95_delta_percent`
- `current_alloc_bytes`, `baseline_alloc_bytes`, `allocation_delta_percent`

### Tasks

- [ ] Formalize the baseline data schema.
- [ ] Compare current results against baseline only when manifest hash and environment match.
- [ ] Mark comparisons as `not comparable` if CPU or build mode differs.
- [ ] Calculate p50, p95, p99, and memory allocation deltas.
- [ ] Emit an explicit error if baseline data is missing.
- [ ] Render regression metrics and status indicators in the dashboard.

Initial Regression Thresholds:

- p50 regression > 5%: warning
- p95 regression > 10%: failure
- p99 regression > 15%: warning
- allocation/frame regression > 10%: failure
- manifest mismatch, missing baseline, or invalid report: failure

## 7. Phase P2 — Separating `go-benchmark.txt` Results

The repository maintains two benchmark reporting mechanisms:

1. `/home/thebanri/Projects/Limoni/benchmark-results/go-benchmark.txt` (Standard Go `testing.B`)
2. `limoni.json`, `bubbletea.json`, `ratatui.json` (Structured cross-framework runners)

Go benchmark output contains `BenchmarkEmptyFrame`, `BenchmarkTextHeavyFrame`, `BenchmarkTenThousandRowTable`, `BenchmarkMouseHitTest`, `BenchmarkHundredLayers`, and `BenchmarkAsyncUpdateBurst`.

### Tasks

- [ ] Generate structured JSON or parsed output for `go-benchmark.txt`.
- [ ] Clearly separate native Go microbenchmarks from cross-implementation suites.
- [ ] Report `ns/op`, `B/op`, and `allocs/op` as discrete metrics.
- [ ] Include CPU and Go toolchain version in native benchmark summaries.
- [ ] Display native Go benchmarks and cross-framework comparisons in dedicated sections.
- [ ] Ensure text parsing failures do not silently succeed in CI.

### Acceptance Criteria

- Go benchmark output can be consumed programmatically without manual text parsing.
- Native Go microbenchmarks and cross-framework comparisons remain unmixed.
- Each result identifies its source benchmark harness.

## 8. Phase P2 — Measurement Scope and Runner Parity

### Tasks

- [ ] Add `measurement_scope` fields per workload across all runners.
- [ ] Document Limoni's buffer/diff pipeline, Bubble Tea's view string generation, and Ratatui's terminal buffer loop.
- [ ] Explicitly annotate operations like `table_rows.clone()` included inside Ratatui benchmark loops.
- [ ] Verify whether native image workloads execute real encoding or stub logic.
- [ ] Dissect async workloads into queue dispatch, update processing, and round-trip latency.
- [ ] Separate resize benchmarks into message dispatch, buffer resize, and full reflow+redraw.
- [ ] Mark non-equivalent workloads as `framework-specific`.

### Acceptance Criteria

- Each JSON artifact identifies the exact execution path under measurement.
- Only workloads sharing equivalent scopes are directly compared.
- Divergent scopes are partitioned into separate dashboard views.
- `VALID` status is reserved strictly for defensible head-to-head comparisons.

## 9. Phase P3 — Artifact Archival and Layout

Recommended directory structure:

```text
benchmark-results/
├── README.txt
├── latest/
│   ├── limoni.json
│   ├── bubbletea.json
│   ├── ratatui.json
│   ├── dashboard.html
│   └── go-benchmark.txt
├── baseline/
│   ├── limoni.json
│   └── metadata.json
└── history/<git-commit>/
    ├── limoni.json
    ├── bubbletea.json
    ├── ratatui.json
    ├── dashboard.html
    ├── go-benchmark.txt
    └── metadata.json
```

### Tasks

- [ ] Formalize artifact layout strategy (`latest`, `baseline`, and `history`).
- [ ] Associate each CI run with its commit hash and manifest hash.
- [ ] Store full historical records as downloadable GitHub Actions artifacts.
- [ ] Retain latest summaries and vetted baselines in the repository.
- [ ] Automate baseline updates through dedicated, controlled workflows.
- [ ] Prevent concurrent runs on the same commit from overwriting artifacts.

## 10. CI Implementation Plan

Target workflow: `/home/thebanri/Projects/Limoni/.github/workflows/benchmarks.yml`

### Workflow Steps

- [ ] Clean output directory prior to benchmark execution.
- [ ] Execute all three runners using a shared workload manifest.
- [ ] Run JSON schema and metadata validation.
- [ ] Verify workload spec and hash parity.
- [ ] Generate HTML dashboard.
- [ ] Verify consistency between JSON data and HTML output.
- [ ] Run baseline regression checks.
- [ ] Upload JSON, dashboard, native benchmark, and metadata artifacts.
- [ ] Fail or warn on invalid reports or regressions exceeding thresholds.

## 11. AI / Agent Operational Protocol

1. Read target files and existing tests.
2. Verify existing artifacts in `benchmark-results`.
3. Confirm baseline validity.
4. Execute benchmark and record baseline prior to modifying code.
5. Focus on a single hypothesis or validation target at a time.
6. Make small, reversible changes.
7. Run affected unit tests.
8. Re-run benchmarks with at least `-count=5`.
9. Evaluate changes across p50, p95, p99, memory/frame, allocs/frame, and schema validity.
10. Add regression or validation tests for successful enhancements.
11. Run `go test ./...`, `go vet ./...`, and relevant race detectors.
12. Review modified files.
13. Verify clean git state with `git diff --check` and `git status`.

## 12. Overall Success Metrics

- Zero empty or invalid JSON files in `benchmark-results`.
- Baseline is valid, complete with metadata, and comparable to latest runs.
- All reports share matching manifest hashes or explicit mismatch indicators.
- Dashboard does not display unconditional `VALID COMPARISON` banners.
- Implementation-specific workloads are clearly segregated.
- Unmeasured allocations are never presented as zero allocations.
- Native Go microbenchmarks remain distinct from cross-implementation suites.
- Baseline regressions are automatically flagged.
- Host, OS, architecture, runtime, and build mode divergences are visible.
- Full provenance (commit, host, timestamp) is preserved for all artifacts.

Final verification:

```bash
cd /home/thebanri/Projects/Limoni
go test ./...
go vet ./...
git diff --check
```

## 13. Non-Goals

- Unsubstantiated general API redesigns without benchmark backing.
- Adding unnecessary third-party dependencies.
- Making architectural decisions based on a single p50 measurement.
- Presenting non-equivalent runner workloads as direct head-to-head marketing comparisons.
- Making definitive claims about end-user perception without measuring real terminal I/O.
- Running regression comparisons across incompatible baseline host environments.