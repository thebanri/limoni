# Cross-implementation benchmark runners

Each runner writes a `benchmarks.DashboardReport` compatible JSON document:

```text
{ "implementation": "...", "workloads": [{ "spec": {...}, "summary": {...} }] }
```

The Bubble Tea runner has its own `go.mod`/`go.sum` because it intentionally
uses the external Charm dependency tree. The Rust runner requires the Rust
toolchain and downloads Ratatui/Serde from crates.io. The GitHub Actions
benchmark workflow installs both toolchains and validates all three
implementation names before generating the dashboard.

The workloads intentionally use the same dimensions and text. They measure an
in-memory render/update path, not terminal I/O, so results are comparable only
within the same CI host and build configuration.

Run locally:

```bash
go run ./benchmarks/runners/limoni -output benchmark-results/limoni.json
go run ./benchmarks/runners/bubbletea -output benchmark-results/bubbletea.json
cargo run --manifest-path benchmarks/runners/ratatui/Cargo.toml -- benchmark-results/ratatui.json
go run ./benchmarks/runners/dashboard -output benchmark-results/dashboard.html \
  benchmark-results/limoni.json benchmark-results/bubbletea.json benchmark-results/ratatui.json
```