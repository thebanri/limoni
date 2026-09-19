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
