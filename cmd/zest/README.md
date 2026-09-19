# 🍋 zest

<p align="center"><img src="../../assets/zest.gif" alt="zest filtering a million-line log" width="100%" /></p>

A terminal log viewer built on [Limoni](../../README.md). It opens a file or reads a pipe, follows it as it
grows, colours lines by level, and filters a million lines without the UI stalling.

```bash
go install github.com/thebanri/limoni/cmd/zest@latest

zest app.log                          # open and follow a file (tail -f)
kubectl logs -f deploy/api | zest     # read a pipe; the keyboard still works
zest -level warn -filter timeout app.log
zest -demo 1000000                    # a generated, still-growing log to try it on
```

| Key | Does |
| :--- | :--- |
| `/` | filter to lines containing text, case-insensitive (Enter keeps it, Esc clears it) |
| `1` … `6` | minimum level: all, debug, info, warn, error, fatal |
| `Enter` | details: the selected JSON line's fields, one per line, time/level/msg first |
| `f` | follow new lines on or off. Scrolling up pauses it; `End` resumes |
| `↑ ↓ j k`, `PgUp PgDn`, `Home g`, `End G` | move |
| `← →` | scroll sideways |
| `?` | help |
| `Esc` | clear the filter and level, staying on the selected line with its context around it; then close details; then quit |
| `q` | quit |

**Levels** are read from JSON (`"level"`, `"lvl"`, `"severity"`), logfmt (`level=warn`) and plain text
(`ERROR`, `[WARN]`, `panic:`…). An indented line with no level of its own, such as a stack frame, takes the level
of the line above it. So filtering to errors keeps the whole trace.

**Rotation:** a followed file that shrinks (copytruncate) is read again from the start.

There's also a [browser version](https://thebanri.github.io/limoni/): the "Logs · zest" scene of the
playground, running the same code as WebAssembly on a 200,000-line demo log.

## How fast

Measured on an AMD Ryzen 5 5600 in kitty 0.48.2, 130×24, with a build of this commit:

| | |
| :--- | :--- |
| A 1,000,000-line, 67 MiB plain log, from `zest file` to all lines on screen | 0.54 s |
| The same log piped in (`cat file \| zest`) | 0.55 s |
| Resident memory with it loaded | 123 MB |
| 1,000,000 generated JSON lines (160 MiB), generated and stored, without the UI | 0.62 s |

Lines are stored once, in fixed 1 MiB chunks, and handed to the view without copying. That's why drawing a
frame of a million-line log allocates nothing (`BenchmarkLogViewDraw`), and why memory is the text plus 16 bytes
per line. Filtering runs in the background in batches of 200,000 lines and redraws as results arrive.

## For agents and tests

The log is a `list` whose visible lines are `list-item`s. The filter is an `input` with the ID `filter`, and the
details pane is `details`. Built with `-tags limoni_debug`, `zest -socket $XDG_RUNTIME_DIR/zest.sock` serves
that tree to [`limoni-mcp`](../limoni-mcp), so an agent can be asked to "find the first connection error after
the deploy". [`ui_test.go`](ui_test.go) drives the whole viewer through the tree in-process.

## Limits

- One input at a time. Several files are not merged by timestamp yet.
- The filter is a plain substring, not a regular expression.
- On Windows, a pipe can't be read, because there is no `/dev/tty` for the keys. Pass a file instead.
