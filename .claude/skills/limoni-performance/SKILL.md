---
name: limoni-performance
description: Where allocations hide in a Limoni frame, how to find them, how to compare benchmarks honestly, and the release and CI checks that catch what one machine cannot. Load before touching a Draw path, a benchmark, an allocation test, or before cutting a release.
---

# Allocations, measurement, releases

The headline claim is a number — zero heap allocations per frame — so the rule is
simple: measure the program people run, on more than one operating system, and
distrust a number you like.

## Where allocations hide

Every one of these was found in this repository, each after the widget
benchmarks said `0 B/op`:

| Hiding place | What it looked like | Fix |
| :--- | :--- | :--- |
| A closure built in `Draw` | `ctx.RegisterClick(area, func(){...})` per clickable widget per frame | `cell.ClickAction{Focus: id, Toggle: &b}` / `RegisterScroll`: data, copied into a slice the frame reuses |
| The frame wrapping handlers | `RegisterClick` wrapped each `func()` in a `func(ev)` | store the handler as it is |
| A theme closure | `ctx.ThemeStyle = func(role)...` per widget per frame | build once on the frame |
| A captured local escaping | `visibleCount` captured by a mouse closure escaped even when no closure was registered | copy it inside the `if` that registers |
| String building | `prefix + label`, `icon + " "`, `fmt.Sprintf` in a draw | write the parts separately with `SetStringWithin` |
| `sync.Pool` | every GC empties it, so the next frame allocated a scratch and two maps | keep scratch on the widget's state |
| `os.Getenv` on Windows | `graphics.DetectProtocol()` per frame: ~12 env reads, 21 allocations on Windows only | detect once, keep it in `CapabilityProfile` |
| Interface boxing by the caller | `RenderWidget(widgets.Checkbox{...})` copies the struct to the heap | hold widgets by pointer |
| A struct growing past 128 bytes | one `bool` added to the end of `cell.Context` → 144 B/op in *every* widget with a fallback click closure | keep `Context` ≤ 128 bytes; put new fields in an existing padding hole |

**The 128-byte cliff.** A closure captures a variable by value only while it
is small; above 128 bytes the compiler captures by reference, which moves the
variable to the heap. `cell.Context` is passed to every `Draw` and captured by
the fallback click closures, so one extra field at the end of it turned every
widget from `0 B/op` to `144 B/op` — while the code around it was unchanged.
`go build -gcflags=-m` says which it is: `leaking param: ctx` is free,
`moved to heap: ctx` is an allocation per call. `TestContextStaysUnder128Bytes`
guards the size; new fields go in the padding after `Style`.

## Finding them

```go
// Deterministic: force a GC each iteration, or a sync.Pool refill looks flaky.
if a := testing.AllocsPerRun(50, func() { draw(); runtime.GC() }); a != 0 { ... }
```

```bash
go test ./pkg -run '^$' -bench X -benchmem -memprofile mem.out -memprofilerate 1 -o pkg.test
go tool pprof -sample_index=alloc_objects -top -lines pkg.test mem.out   # names the line
```

`AllocsPerRun` returns an average: a reading of `1` means *every* iteration
allocated, not one stray. Treat it as a fact about the program, not as flakiness.

Benchmarks that only call `Draw` cannot see registration. `BenchmarkInteractiveFrame`
(in `benchmarks/`) draws through a real `Terminal` with a theme; keep new
interactive work covered by it, and CI gates every widget benchmark.

## Comparing honestly

```bash
git worktree add --detach /tmp/base <base-commit>
(cd /tmp/base && go test ./core/buffer -run '^$' -bench X -count=5)
go test ./core/buffer -run '^$' -bench X -count=5      # same machine, same sitting
git worktree remove /tmp/base
```

- Compare against the previous commit, never against the README's numbers.
- Check numbers against each other. A load time that contradicts the per-item
  cost, or a line count that contradicts the elapsed time, means the harness is
  wrong — a "30 s" load turned out to be a leftover process from a failed run.
- The order of a condition matters in a per-cell loop:
  `cell.IsCluster(c) && !opts.ClusterWidths` costs nothing;
  the reverse cost 5% on `BenchmarkDiff_FullChanges`.
- Kill stray processes by pid file. `pkill -f <pattern>` matches the shell
  running it and kills your own session.

## What only CI can tell you

Windows is the runner that catches what Linux hides: `os.Getenv` allocating, and
AF_UNIX resetting a connection closed with unread data. Cross-compiling
(`GOOS=windows go vet ./...`) proves it builds, nothing more. Push and read the
job.

## Releasing

1. `go build ./... && go vet ./... && go test ./...`, `-race` on the CLAUDE.md
   list, the two benchmark gates, `GOOS=js GOARCH=wasm` build.
2. `gorelease -base=<previous tag>` **without** `-version`: naming a version
   that does not exist yet makes the Go proxy cache a "not found" for it, and
   `go install pkg@version` then fails for ~18 minutes.
3. In a worktree of `origin/main`: turn `## [Unreleased]` into the version with
   today's date, add the compare link, commit `release: vX.Y.Z`, push, tag, push
   the tag.
4. `gh release create` with notes assembled from that CHANGELOG section.
5. Verify from outside the repository: `go install github.com/thebanri/limoni/cmd/zest@vX.Y.Z`
   in an empty directory. The proxy's `@latest` list lags the tag by a few
   minutes; the version itself is usually available at once.
