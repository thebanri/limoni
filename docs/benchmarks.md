# Benchmarks

> 📐 **Read [`docs/benchmark-methodology.md`](benchmark-methodology.md) first.** It states which framework versions are measured, what the harness does and does not capture, and which comparison claims are currently unproven. The cross-framework runners target **Ratatui 0.30.2**, **Ultraviolet** (the cell renderer under Bubble Tea v2 and Lip Gloss v2) and **Bubble Tea v1.3.10**. There is **no Bubble Tea v2 runner**: nothing here links it, so no claim in this repository is a claim about v2 itself.

Limoni includes a standardized cross-implementation benchmark suite measuring real dirty diffing, partial invalidations, virtual scrolling, and memory allocations under standard virtual terminal conditions (120×40 cells = 4,800 cells).

Run benchmarks locally:
```bash
# Run Buffer Diff benchmarks (measured dirty and clean passes)
go test ./core/buffer -run '^$' -bench . -benchmem

# Run Widget & Layout benchmarks
go test ./benchmarks -run '^$' -bench . -benchmem

# Cross-framework comparison: builds each runner, runs all three three times,
# reports median latency with run-to-run spread, and suppresses every ratio the
# runners mark non-comparable. Needs a Rust toolchain for the Ratatui runner.
./benchmarks/compare.sh

# Generate HTML Comparison Dashboard
go run ./benchmarks/runners/dashboard -output benchmark-results/dashboard.html benchmark-results/limoni.json benchmark-results/bubbletea.json benchmark-results/ratatui.json
```

## Measured results

**AMD Ryzen 5 5600 (6C/12T), Linux 6.17, Go 1.27.1, `-count=3`, median.** Absolute
figures are hardware-dependent; what is meaningful is the ratio between commits on
one machine. Reproduce with the command above.

| Benchmark Operation | Measured Latency | Throughput | Allocations | Description |
| :--- | :--- | :--- | :--- | :--- |
| **`BenchmarkDiff_FullChanges`** | **`~50.5 µs`** | **~19,800 FPS** | **`0 B/op (0 allocs)`** | 100% full-screen cell mutation (4,800 cells) diffed against persistent double-buffer emitting ANSI escape stream |
| **`BenchmarkDiff_PartialChanges`** | **`~23.7 µs`** | **~42,200 FPS** | **`0 B/op (0 allocs)`** | 10% viewport mutation (480 cells across shifting rows) diffed against persistent double-buffer |
| **`BenchmarkDiff_NoChanges`** | **`~1.94 ns`** | **~516,000,000 FPS** | **`0 B/op (0 allocs)`** | Clean frame fast-path bypass when no buffer cells mutated |
| **`BenchmarkTextHeavyFrame`** | **`~31.7 µs`** | **~31,500 FPS** | **`2 B/op (0 allocs)`** | 40-line text dashboard rendering with unicode symbols and word wrapping across 120 columns |
| **`BenchmarkHundredLayers`** | **`~68.2 µs`** | **~14,700 FPS** | **`1 B/op (0 allocs)`** | 100 layered Block widgets evaluation and frame rendering (Ratatui hundred-layers parity) |
| **`BenchmarkTenThousandRowTable`** | **`~68.5 µs`** | **~14,600 FPS** | **`611 B/op (4 allocs)`** | Active selection scrolling through a 10,000-row table rendering visible rows |
| **`BenchmarkOneMillionRowVirtualScroll`**| **`~2.70 ms`** | **~370 FPS** | **`4.9 KB/op (6 allocs)`** | Active virtual scrolling across 1,000,000 rows with viewport boundary pruning |
| **`BenchmarkMouseHitTest`** | **`~63.4 ns`** | **~15,800,000 ops/s**| **`0 B/op (0 allocs)`** | Hierarchical widget tree spatial hit testing across 100 click regions |
| **`BenchmarkAsyncUpdateBurst`** | **`~232 ns`** | **~4,310,000 msg/s** | **`7 B/op (0 allocs)`** | High-throughput Elm runtime async message dispatch |

> [!NOTE]
> **These figures moved twice, in both directions.** An earlier revision of this
> table quoted latencies that did not reproduce on the hardware class it named —
> `BenchmarkHundredLayers` was claimed at 47 µs and measured 159 µs. Correcting
> that exposed where the time was actually going: `cell.RuneWidth` was 39% of a
> layered frame, walking twenty range comparisons per cell written. It now
> answers from a lookup table, which took the draw path down roughly 2–3× across
> the board. So the numbers above are lower than the corrected ones *and* lower
> than the original inflated claims — this time with a profile behind them.
> The zero-allocation guarantees held throughout.

> [!NOTE]
> **The table predates grapheme clusters.** Segmenting text costs something for
> non-ASCII characters; plain ASCII takes a fast path that skips it. Measured
> back to back on the machine above against the commit before the change
> (`-count=3`, medians): `BenchmarkTextHeavyFrame` +5% (30.7 → 32.3 µs — its text
> has three symbols per line), `BenchmarkDiff_FullChanges` +2%,
> `BenchmarkHundredLayers` and `BenchmarkDiff_PartialChanges` unchanged, all
> still at zero allocations. The absolute figures in the table were not
> re-measured, so compare the ratios, not the rows.

> [!NOTE]
> **Transparency & Engineering Integrity Guarantee**:
> We do not use synthetic shortcuts, artificial buffer clears, or zero-offset static loops.
> - **Diff Benchmarks**: Run against a persistent double-buffer where cells genuinely mutate every single frame, forcing the full diff algorithm and ANSI encoder to run end-to-end.
> - **Scroll Benchmarks**: Actively cycle through rows (`Select((i * 7) % N)`), proving zero-overhead virtual window rendering under continuous scrolling.
> - **Hundred Layers**: Genuinely renders 100 overlapping `Block` widgets rather than a synthetic hit-test shortcut.
