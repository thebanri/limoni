# 🌐 Cross-Platform and Network Drivers

Limoni abstracts the driver layer (`core/driver`) to achieve complete operating system and hardware independence.

---

## 1. Supported Platforms

| Platform | Driver File | Mechanism |
| :--- | :--- | :--- |
| **Linux & BSD** | `core/driver/termios_linux.go` | `ioctl` TCGETS/TCSETS, Epoll / Non-blocking TTY I/O |
| **macOS (Darwin)** | `core/driver/termios_darwin.go` | Darwin `termios` CGO-free Syscall & Kqueue |
| **Windows** | `core/driver/backend_windows.go` | Windows Console Virtual Terminal Sequences (`ENABLE_VIRTUAL_TERMINAL_PROCESSING`) |
| **WebAssembly** | `core/driver/backend_wasm.go` | `syscall/js` with browser xterm.js bridge |
| **Remote Network / SSH** | `core/driver/ssh.go` | Direct isolated ANSI diff stream over `net.Conn` or `crypto/ssh.Session` |

---

## 2. Running in the Browser with WebAssembly (WASM)

Limoni applications can be directly compiled to WebAssembly and run in any web browser:

```bash
GOOS=js GOARCH=wasm go build -o limoni.wasm ./examples/wasm
```

In your HTML page:
```html
<div id="terminal"></div>
<script src="wasm_exec.js"></script>
<script>
    const go = new Go();
    WebAssembly.instantiateStreaming(fetch("limoni.wasm"), go.importObject).then((result) => {
        go.run(result.instance);
    });
</script>
```

---

## 3. SSH / Remote Terminal Server

Using `driver.NewSSHDriver` (or `driver.NewSSHBackend`), a single Go binary can serve interactive TUI sessions to hundreds of concurrent users:

```bash
go run ./examples/ssh_server
```

To connect:
```bash
nc localhost 2222
```

---

## 4. Terminal Capability Handshake

Environment variables are a poor guide to what a terminal can do. Inside tmux
`TERM` is `screen` or `tmux-256color`, over SSH it is whatever the client sent,
and many emulators set nothing that identifies them. So when a backend sets up
the terminal, it also asks the terminal directly:

| Query | What it answers |
| :--- | :--- |
| `CSI > 0 q` (XTVERSION) | the terminal's name and version, e.g. `kitty(0.48.2)`, `tmux 3.5a` |
| `CSI ? 2026 $ p` (DECRQM) | whether synchronized output is supported |
| `CSI ? 2027 $ p` (DECRQM) | whether grapheme-cluster mode is supported and on |
| `CSI ? u` | whether the Kitty keyboard protocol is available |
| a space, `CSI 1 b`, `CSI 6 n` | **measured**: whether REP really repeats a glyph |
| a ZWJ family emoji, `CSI 6 n` | **measured**: how many columns the terminal draws a cluster |
| `CSI c` (DA1) | sent last, as a sentinel: every terminal answers it, in order |

The two measurements write a few cells, ask where the cursor went, then erase
them and restore the cursor before anything is drawn. They matter because a
name does not settle the question. On the same machine, kitty 0.48.2 and
Konsole 26.08.1 report mode 2027 as unsupported or don't answer it, yet both
draw the family emoji two columns wide. Alacritty draws it six columns wide.
Limoni skips the per-cluster cursor re-anchoring on the first two and keeps it
on Alacritty.

The replies arrive as input. The backend's event loop takes them out of the
stream, so an application never sees them as key presses, and folds them into
a `driver.TerminalReport`. `Terminal.Draw` checks for new answers on every
frame; checking costs one atomic load. If answers arrive after the first
frame, which is common over SSH, and they change the profile, the screen is
repainted in full. `Terminal.SetCapabilities` still has the final word, and
`Backend.Close` waits up to 150 ms for replies still in flight, so they do not
end up printed in your shell.

To see what your terminal reports, and what Limoni makes of it:

```bash
go run github.com/thebanri/limoni/cmd/limoni@latest doctor
```

Include that output in rendering bug reports. Escape hatches:
`LIMONI_PROBE=0` sends no queries, `LIMONI_REP=0|1` forces REP off or on, and
`LIMONI_NO_SYNC=1` disables synchronized output.

---

## 5. Suspending with Ctrl+Z

`limoni.WithSuspend()` makes Ctrl+Z behave the way it does in `vim` or `less`:
the application hands the terminal back to the shell and stops; `fg` brings it
back and the screen is repainted.

```go
limoni.Run(draw, limoni.WithSuspend())
```

The order is what matters. Limoni leaves the alternate screen and raw mode
*before* raising `SIGTSTP`, or the shell inherits a terminal with no echo and
the application's screen still on it. On resume it re-enters raw mode, sends
the setup sequence, asks the terminal again what it supports (it may be a
different terminal), and forces a full repaint, because the shell has written
over the screen in the meantime.

`Terminal.Suspend()` does the same for an application that would rather bind
its own key. Both return `driver.ErrSuspendUnsupported` where there is no shell
to return to — a remote or in-memory backend, the browser, Windows — and with
`WithSuspend` the key is then delivered to the application as usual.

---

## 6. The window title

`limoni.WithTitle("zest — app.log")` sets the terminal's window title with
OSC 2 while the application runs, and puts the previous one back on the way
out. `Terminal.SetTitle`, `SaveTitle` and `RestoreTitle` are there for an
application that wants to change the title as its state changes — a file name,
a progress figure.

Control characters are stripped from the title before it is written, so a
title built from a file name or a log line cannot smuggle an escape sequence
through. Saving and restoring uses XTWINOPS (`CSI 22;2t` / `CSI 23;2t`);
terminals that do not implement it ignore both, and the title then simply
stays as the application set it.

---

## 7. Hyperlinks (OSC 8)

```go
f.Buffer.SetString(2, 1, "the changelog", limoni.Hyperlink(url).Underline())
```

`limoni.Hyperlink(url)` — or `Style.WithLink(url)` on a style you already
have — makes the text a link. `widgets.Markdown` uses it for `[text](url)`,
so a markdown pane renders real clickable links.

A link belongs to the *cell*, not to a span of text, because the diff writes
cells in whatever order it finds them. Storing the URL in each cell would put
a pointer in every one of them and end the flat 16-byte-cell design, so URLs
are interned and the cell keeps a 16-bit handle — in the two bytes `Style`
was padding with anyway. The handle is also the `id=` parameter of OSC 8,
which is what lets a terminal treat a link split across rows as one link when
the pointer hovers it.

**Terminals that cannot show links are never sent the sequence.** OSC 8 is
supposed to be ignored where it is unknown, but not every terminal obeys
that, and a URL printed into the middle of a frame is worse than no link. So
it is capability-gated like REP: on for kitty, WezTerm, foot, Ghostty,
iTerm2, Konsole, Contour, the VTE terminals, Windows Terminal and Alacritty,
off elsewhere, and `LIMONI_HYPERLINKS=1` or `=0` overrides either way.
`limoni doctor` prints the decision.

Widgets can read the same capability from `ctx.Hyperlinks`, which is how
Markdown decides whether to print the address as well: with links, the label
alone is drawn; without them, `text (https://…)`, so a reader who cannot
click still sees where it points.
