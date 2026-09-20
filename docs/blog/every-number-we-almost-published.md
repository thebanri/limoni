# Every number we almost published

*Limoni is a terminal UI engine for Go whose headline claim is a number: zero heap allocations per
frame. This is a list of the times we measured something, got an answer we liked, and were wrong. Each one
was caught before it went into the README, and each left behind a test or a rule.*

---

## 1. 4,700× faster, according to a harness bug

To compare Limoni with Bubble Tea v2 we wrote a runner for
[Ultraviolet](https://github.com/charmbracelet/ultraviolet), the cell renderer Bubble Tea v2 is built on.
The first run said Limoni drew an unchanged frame **4,700 times faster**.

It didn't. Ultraviolet's `TerminalRenderer` returns early when no line is marked as touched, and it's
`TerminalScreen` — the layer above, which we had skipped — that clears those marks. Our harness left
every line marked, so every frame re-scanned the whole screen: 195 µs. Once the harness cleared the marks
the way the real stack does, the frame took 42 ns, the same order as Limoni's own fast path.

The runner was rewritten against `TerminalScreen`, and its numbers were held back until then. The rule
that came out of it is now written into the benchmark methodology: *a number that looks too good is a bug in
the harness until proven otherwise.* The same rule later held back a "1,807% regression" in Ratatui's resize
path, which also turned out to come from the harness: the runner had been talking to a real terminal.

## 2. Zero allocations, in the benchmarks

Limoni's widgets have `-benchmem` benchmarks, and CI failed the build if one of them allocated. All of
them read `0 B/op` except TextInput, at 24 B/op, and it went unnoticed because CI only gated four of the
benchmarks. That was the first thing fixed: CI now gates every widget benchmark.

Then we drew four ordinary widgets — a checkbox, a text input, a list and a block — through a real
`Terminal` with a theme set: **19 allocations and 816 bytes per frame.** The widget benchmarks call `Draw`
directly. A real frame also registers what a click does, and every clickable widget did that with a
closure built during `Draw`, which is a heap allocation. The frame then wrapped each one in a second closure,
and the theme added one more per widget.

The fix was to describe common click behaviour as data instead of code: `cell.ClickAction{Focus: id,
Toggle: &checked}` is copied into a slice the frame reuses, and regions refer to it by index. The frame
is now at zero, and `BenchmarkInteractiveFrame` guards it in CI. The README claim was narrowed to what's
true: widgets that drag or call back into the application, such as Table and Slider, still allocate one closure
each, and the README says so.

The lesson: a benchmark measures its harness. If the harness leaves out the part of the program that
allocates, it will report zero, and it will keep reporting zero.

## 3. Zero allocations, on Linux

A new test checked that a checkbox nested inside a Block draws without allocating. It passed on Linux and macOS.
On the Windows CI runner it said **21 allocations per frame**.

`Terminal.Draw` called `graphics.DetectProtocol()` on every frame to decide how to draw images. That
function reads about a dozen environment variables. On Linux `os.Getenv` is a lookup in a slice the
process already has. On Windows it asks the OS and converts UTF-16, which allocates. We had been running our
allocation checks on one operating system and describing all of them.

The protocol is now detected once, when the terminal is created. The change also made
`SetCapabilities` able to override it, which its documentation had claimed all along.

## 4. A flaky test that wasn't flaky

The same test later failed once during a full `go test ./...` and then passed fifteen times in a row.
It's tempting to call that flaky and re-run CI.

`testing.AllocsPerRun` reports an average, so a reading of 1 means every iteration allocated during that
run, not one stray allocation. Something depended on the machine's state. It turned out to be Table, which
kept its scratch buffers in a `sync.Pool`, and every garbage collection empties a `sync.Pool`. Under load,
GCs ran during the measurement, and the next frame after each one allocated a new scratch and two maps. A real
application does the same after every collection.

A table with state now keeps its scratch there. The test forces a GC on every iteration, so the pool version
fails it every time instead of now and then.

## 5. A million log lines in 30 seconds

[zest](../../cmd/zest), the log viewer we built on Limoni, reported taking **30 seconds** to load a
million generated lines.

The parts didn't add up to that: level detection took 88 ns a line, storing it 156 ns, and generating it
462 ns, so a million lines should take well under a second. The screen we were reading showed 1,010,150
lines, ten thousand more than the million we'd asked for. At the demo's ~80 new lines a second, that meant
the process had been running for two minutes. We were reading a leftover instance from an earlier failed run,
and our wait loop had simply timed out at 30 seconds.

Measured properly, a million lines are generated and stored in 0.62 s, and a real 67 MiB file is on screen in
0.54 s. The numbers contradicted themselves, and that contradiction is how we noticed.

## 6. A 5% regression hidden in a boolean

Teaching the diff engine to skip cursor re-anchoring on terminals that draw emoji sequences correctly added
one check per cell: `!opts.ClusterWidths && cell.IsCluster(c)`. `BenchmarkDiff_FullChanges` got **5%
slower**, and the result reproduced across five runs against the base commit on the same machine.

The option lives in a struct, and reading it on every cell cost more than the check it guarded. Swapping the order to
`cell.IsCluster(c) && !opts.ClusterWidths` means the common case (not a cluster) stops at the first
comparison, as it did before. The medians went back to 58.0 and 58.2 µs against the base's 58.3 and 58.2.
We only noticed because we measured the change against the previous commit, not against the README.

---

## What we do now

- **Compare commits on one machine in one sitting**, not a new run against old README numbers.
- **Treat a surprising number as a harness bug** until something else explains it.
- **Measure the program people run**, not the parts that are easy to benchmark. `BenchmarkInteractiveFrame`
  exists because the widget benchmarks couldn't see closures.
- **Check the numbers against each other.** A load time that doesn't match the per-line costs, or a line
  count that doesn't match the elapsed time, means something is off.
- **Make a test deterministic instead of re-running it.** A test that fails "sometimes" is usually
  telling you something true about the program.

All of this is in the repository: the commits, the tests that fail without their fixes, and the
[benchmark methodology](../benchmark-methodology.md).
