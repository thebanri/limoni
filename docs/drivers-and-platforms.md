# 🌐 Çapraz Platform ve Ağ Sürücüleri (Drivers & Platforms)

Limoni, işletim sistemi ve donanım bağımsızlığı sağlamak için sürücü katmanını (`core/backend`) soyutlamıştır.

---

## 1. Desteklenen Platformlar

| Platform | Sürücü Dosyası | Mekanizma |
| :--- | :--- | :--- |
| **Linux & BSD** | `core/backend/termios_linux.go` | `ioctl` TCGETS/TCSETS, Epoll / Non-blocking TTY I/O |
| **macOS (Darwin)** | `core/backend/termios_darwin.go` | Darwin `termios` CGO'suz Syscall & Kqueue |
| **Windows** | `core/backend/backend_windows.go` | Windows Console Virtual Terminal Sequences (`ENABLE_VIRTUAL_TERMINAL_PROCESSING`) |
| **WebAssembly** | `core/backend/backend_wasm.go` | `syscall/js` ile tarayıcı xterm.js köprüsü |
| **Uzak Ağ / SSH** | `core/backend/ssh.go` | `net.Conn` veya `crypto/ssh.Session` üzerinde doğrudan izole ANSI diff akışı |

---

## 2. WebAssembly (WASM) ile Tarayıcıda Çalıştırma

Limoni uygulamaları doğrudan WebAssembly olarak derlenip herhangi bir web sayfasında çalıştırılabilir:

```bash
GOOS=js GOARCH=wasm go build -o limoni.wasm ./examples/wasm
```

HTML tarafında:
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

## 3. SSH / Uzak Terminal Sunucusu

`backend.NewSSHBackend` sayesinde tek bir Go ikili dosyası ile yüzlerce eşzamanlı kullanıcıya interaktif TUI oturumları sunulabilir:

```bash
go run ./examples/ssh_server
```
Bağlanmak için:
```bash
nc localhost 2222
```
