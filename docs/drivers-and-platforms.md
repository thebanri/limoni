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
