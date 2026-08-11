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

The Limoni report/dashboard helpers write the common JSON/HTML shape. External
Ratatui and Bubble Tea runners should upload their reports as CI artifacts and
must not be compared with different workloads or sinks.
