# Limoni Platform Testing

This document describes how to validate the parts of Limoni that cannot be
trusted from a Linux cross-compile alone.

## Local baseline

Run from the repository root:

```bash
go test ./...
go vet ./...
go test -race ./core/runtime ./core/terminal ./testkit ./widgets ./layout ./core/accessibility ./core/backend ./benchmarks
go test ./benchmarks -run '^$' -bench . -benchmem
```

Cross-compilation checks that do not execute foreign binaries:

```bash
GOOS=darwin GOARCH=amd64 go test -c -o /tmp/limoni-terminal-darwin.test ./core/terminal
GOOS=windows GOARCH=amd64 go test -c -o /tmp/limoni-terminal-windows.test ./core/terminal
GOOS=freebsd GOARCH=amd64 go test -c -o /tmp/limoni-terminal-freebsd.test ./core/terminal
```

## Native platform smoke

The GitHub Actions workflow runs native tests on Linux, macOS, and Windows.
Native tests are different from `GOOS=... go test -c`: the binary is executed
on the target operating system.

Run the following on a real target host or native CI runner:

```bash
go test ./...
go vet ./...
go build ./examples/demo
go run ./examples/demo
```

Resize the terminal while the demo is running and verify resize events, mouse
routing, paste, focus, and modal isolation.

Verified Platform Smoke artifacts (2026-08-11) are stored under
`platform-results/`:

```text
platform-results/linux.env    Linux X64, go1.26.5 linux/amd64
platform-results/macos.env    macOS ARM64, go1.26.5 darwin/arm64
platform-results/windows.env  Windows X64, go1.26.5 windows/amd64
```

All three native CI jobs completed successfully. These artifacts prove the
workflow's native test/build/vet/compile path; they do not by themselves prove
OS-native screen-reader protocol behavior or a platform-specific raw-mode
feature that is not exercised by the workflow.

## Linux TTY and PTY

For raw-mode testing, use a real TTY rather than only an IDE terminal:

```bash
script -qfec 'go run ./examples/demo' /tmp/limoni-pty.log
```

`/tmp/limoni-pty.log` must be inspected for `COMMAND_EXIT_CODE="0"`, alternate
screen setup/restore sequences, synchronized update pairs, and the absence of
panic/fatal output. A successful xterm-kitty PTY run verifies the Linux TTY
lifecycle and line-mode emission. It does not prove macOS VoiceOver, Windows
Narrator, or BSD raw-mode support.

Verified Linux evidence (2026-08-11):

```text
COMMAND="go run ./examples/demo --screen-reader"
TERM="xterm-kitty"
TTY="/dev/pts/0"
COLUMNS="185" LINES="40"
COMMAND_EXIT_CODE="0"
```

The run emitted the `[limoni screen-reader]` semantic line-mode marker and
completed with the normal Limoni exit message. SIGWINCH/resize was verified in
the same Linux TTY test environment.

Direct stdout redirection is not a valid raw-mode test:

```bash
go run ./examples/demo > output.log
```

The demo intentionally requires an active TTY because raw-mode setup and window
size detection use terminal ioctl calls. Such redirection may fail with
`inappropriate ioctl for device`; use `script`, a real terminal, or a PTY
runner instead.

For a process-backed stream, use `backend.ExecPTYAdapter`. Native PTY resize
requires a platform implementation; the portable adapter deliberately returns
an explicit error instead of pretending to resize a real PTY.

## Screen readers

The core package provides a platform-independent `ScreenReaderAdapter` and a
line-mode writer:

```go
adapter := accessibility.LineModeAdapter{}
err := adapter.WriteTree(os.Stdout, accessibility.Mode{ScreenReader: true}, nodes)
```

OS-native adapters belong outside the core package:

- Linux: AT-SPI/desktop-session adapter
- macOS: VoiceOver integration
- Windows: Narrator/UI Automation or ConPTY bridge

These adapters must be tested on their native operating system and should not
be marked complete from a Linux compile alone.

## Ratatui/Bubble Tea comparison

Use the same `benchmarks.WorkloadSpec` and export one JSON report per
implementation. At minimum, keep these fields equivalent:

- terminal width and height
- exact text/row content
- Unicode and color mode
- output sink
- full redraw versus diff mode
- build mode and host CPU

The repository includes minimal runners under `benchmarks/runners/`:

- `limoni` — Go/Limoni runner
- `bubbletea` — Go/Bubble Tea runner
- `ratatui` — Rust/Ratatui runner
- `dashboard` — combines runner JSON files into HTML

The benchmark workflow installs the Rust toolchain, downloads the nested Go
module dependencies, runs all three runners, validates their implementation
names, and uploads the JSON/HTML artifacts. Local environments without Cargo
or network access can still run the Limoni runner and dashboard; the external
runner steps remain CI-only in that case.

The Limoni report/dashboard helpers write the common JSON/HTML shape. External
runners should upload their reports as CI artifacts and must not be compared
with different workloads or sinks.
