package main

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/graphics"
	"github.com/thebanri/limoni/layout"
	"github.com/thebanri/limoni/widgets"
)

type text struct {
	value string
	style cell.Style
}

func (t text) Draw(ctx cell.Context, buf *buffer.Buffer) {
	buf.SetString(ctx.Area.X, ctx.Area.Y, t.value, ctx.Style.Merge(t.style))
}

func (t text) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	return uint16(len(t.value)), 1
}

// sshSessionWrapper wraps ssh.Channel to implement backend.SSHSessionIO
type sshSessionWrapper struct {
	channel ssh.Channel
	width   uint16
	height  uint16
	mu      sync.Mutex
}

func (s *sshSessionWrapper) Read(p []byte) (int, error) {
	return s.channel.Read(p)
}

func (s *sshSessionWrapper) Write(p []byte) (int, error) {
	return s.channel.Write(p)
}

func (s *sshSessionWrapper) Size() (uint16, uint16, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.width, s.height, nil
}

func (s *sshSessionWrapper) SetSize(w, h uint16) {
	s.mu.Lock()
	s.width = w
	s.height = h
	s.mu.Unlock()
}

func main() {
	// Configure SSH Server
	config := &ssh.ServerConfig{
		NoClientAuth: true, // Allow any username and password for simple demo
	}

	// Generate a temporary host key programmatically
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate host key: %v\n", err)
		os.Exit(1)
	}
	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create signer: %v\n", err)
		os.Exit(1)
	}
	config.AddHostKey(signer)

	// Listen on 127.0.0.1:2222
	listener, err := net.Listen("tcp", "127.0.0.1:2222")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Listen error: %v\n", err)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Println("Limoni SSH Server started: ssh -p 2222 localhost")

	for {
		nConn, err := listener.Accept()
		if err != nil {
			continue
		}

		go handleSSHConnection(nConn, config)
	}
}

func handleSSHConnection(nConn net.Conn, config *ssh.ServerConfig) {
	_, chans, reqs, err := ssh.NewServerConn(nConn, config)
	if err != nil {
		return
	}

	// Discard global requests
	go ssh.DiscardRequests(reqs)

	for newChan := range chans {
		if newChan.ChannelType() != "session" {
			newChan.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChan.Accept()
		if err != nil {
			return
		}

		go handleSessionChannel(channel, requests)
	}
}

type ptyRequestMsg struct {
	Term     string
	Columns  uint32
	Rows     uint32
	Width    uint32
	Height   uint32
	Modelist string
}

type windowChangeMsg struct {
	Columns uint32
	Rows    uint32
	Width   uint32
	Height  uint32
}

func handleSessionChannel(channel ssh.Channel, requests <-chan *ssh.Request) {
	wrapper := &sshSessionWrapper{
		channel: channel,
		width:   80,
		height:  24,
	}

	var sshBackend *backend.SSHBackend
	var term *terminal.Terminal
	var activeLoopCancel func()
	var mu sync.Mutex

	closeSession := func() {
		mu.Lock()
		defer mu.Unlock()
		if activeLoopCancel != nil {
			activeLoopCancel()
			activeLoopCancel = nil
		}
		if sshBackend != nil {
			sshBackend.Close()
		}
		channel.Close()
	}

	for req := range requests {
		switch req.Type {
		case "pty-req":
			var msg ptyRequestMsg
			if err := ssh.Unmarshal(req.Payload, &msg); err == nil {
				wrapper.SetSize(uint16(msg.Columns), uint16(msg.Rows))
				if sshBackend != nil {
					sshBackend.SetSize(uint16(msg.Columns), uint16(msg.Rows))
				}
			}
			req.Reply(true, nil)

		case "window-change":
			var msg windowChangeMsg
			if err := ssh.Unmarshal(req.Payload, &msg); err == nil {
				wrapper.SetSize(uint16(msg.Columns), uint16(msg.Rows))
				if sshBackend != nil {
					sshBackend.SetSize(uint16(msg.Columns), uint16(msg.Rows))
				}
			}
			req.Reply(true, nil)

		case "shell":
			req.Reply(true, nil)

			mu.Lock()
			sshBackend = backend.NewSSHBackend(wrapper)
			if err := sshBackend.Setup(); err != nil {
				mu.Unlock()
				channel.Close()
				return
			}

			t, err := terminal.New(sshBackend)
			if err != nil {
				mu.Unlock()
				sshBackend.Close()
				channel.Close()
				return
			}
			term = t

			sshBackend.StartEventLoop()

			// Start application loop
			done := make(chan struct{})
			activeLoopCancel = func() {
				select {
				case <-done:
				default:
					close(done)
				}
			}
			mu.Unlock()

			go runTUIApp(term, sshBackend, done, closeSession)
		}
	}
}

func runTUIApp(t *terminal.Terminal, b *backend.SSHBackend, done chan struct{}, closeSession func()) {
	defer closeSession()

	ticker := time.NewTicker(33 * time.Millisecond)
	defer ticker.Stop()

	rotX, rotY := 0.0, 0.0
	canvas := widgets.NewCanvas(40, 15)

	draw := func() {
		t.Draw(func(f *terminal.Frame) {
			area := f.Buffer.Area
			f.SetTheme(widgets.DarkTheme())

			rootLay := layout.NewFlexLayout(
				layout.Vertical,
				0,
				layout.Fixed(3), // Header
				layout.Fill(),   // Body
				layout.Fixed(1), // Footer
			)
			chunks := rootLay.Split(area)

			// Header
			f.RenderWidget(widgets.Block{
				Title:          " LIMONI REMOTE SSH APP ",
				TitleAlignment: widgets.AlignCenter,
				Borders:        widgets.BorderAll,
				BorderSymbols:  widgets.SymbolsRounded,
				BorderStyle:    cell.Style{Fg: cell.NewColorRGB(0, 180, 255)},
				Child:          text{value: " Interactive remote terminal session over SSH ", style: cell.Style{Fg: cell.NewColorRGB(200, 200, 220)}},
			}, chunks[0])

			// Body Split: Left 3D Canvas + Right Welcome Markdown
			bodyLay := layout.NewFlexLayout(
				layout.Horizontal,
				1,
				layout.Fixed(42),
				layout.Fill(),
			)
			bodyChunks := bodyLay.Split(chunks[1])

			// Left 3D Canvas
			canvasW := bodyChunks[0].Width - 2
			canvasH := bodyChunks[0].Height - 2
			canvas.Reset(canvasW, canvasH)

			drawRemoteCube(canvas, canvasW, canvasH, rotX, rotY)

			f.RenderWidget(widgets.Block{
				Title:         " 3D CUBE ",
				Borders:       widgets.BorderAll,
				BorderSymbols: widgets.SymbolsRounded,
				BorderStyle:   cell.Style{Fg: cell.NewColorRGB(0, 180, 255)},
				Child:         canvas,
			}, bodyChunks[0])

			// Right Info Panel
			welcomeText := "# Limoni SSH Server\n\n" +
				"Interactive TUI running over a remote SSH connection.\n" +
				"- Automatically captures terminal `window-change` signals.\n" +
				"- Zero native GUI dependencies with full TrueColor support.\n\n" +
				"**Shortcuts:**\n" +
				"- `Arrow Up/Down/Left/Right`: Rotate cube manually.\n" +
				"- `Esc` or `q`: Disconnect session."

			f.RenderWidget(widgets.Block{
				Title:         " CONNECTION INFO ",
				Borders:       widgets.BorderAll,
				BorderSymbols: widgets.SymbolsRounded,
				BorderStyle:   cell.Style{Fg: cell.NewColorRGB(0, 180, 255)},
				PaddingLeft:   2,
				PaddingRight:  2,
				Child:         &widgets.Markdown{Content: welcomeText, Style: cell.Style{Fg: cell.NewColorRGB(200, 200, 210)}},
			}, bodyChunks[1])

			// Footer
			f.RenderWidget(widgets.Block{
				Borders: widgets.BorderNone,
				Style:   cell.Style{Fg: cell.NewColorRGB(120, 120, 130), Bg: cell.NewColorRGB(20, 20, 25)},
				Child:   text{value: " Arrow Keys: Rotate Cube | Esc/q: Disconnect ", style: cell.Style{Fg: cell.NewColorRGB(130, 130, 130)}},
			}, chunks[2])
		})
	}

	draw()

	for {
		select {
		case <-done:
			return
		case ev, ok := <-b.Events():
			if !ok {
				return
			}
			switch ev.Type {
			case backend.EventKey:
				if ev.Key.Type == backend.KeyEsc || (ev.Key.Type == backend.KeyRune && ev.Key.Ch == 'q') {
					return
				}

				if ev.Key.Type == backend.KeyArrowUp {
					rotX += 10
				}
				if ev.Key.Type == backend.KeyArrowDown {
					rotX -= 10
				}
				if ev.Key.Type == backend.KeyArrowLeft {
					rotY -= 10
				}
				if ev.Key.Type == backend.KeyArrowRight {
					rotY += 10
				}
				draw()

			case backend.EventResize:
				draw()
			}

		case <-ticker.C:
			rotY += 2
			rotX += 1
			draw()
		}
	}
}

func drawRemoteCube(canvas *widgets.Canvas, w, h uint16, rotX, rotY float64) {
	virtualW := int(w) * 2
	virtualH := int(h) * 4

	vertices := []graphics.Vertex3D{
		{X: -1.0, Y: -1.0, Z: -1.0},
		{X: 1.0, Y: -1.0, Z: -1.0},
		{X: 1.0, Y: 1.0, Z: -1.0},
		{X: -1.0, Y: 1.0, Z: -1.0},
		{X: -1.0, Y: -1.0, Z: 1.0},
		{X: 1.0, Y: -1.0, Z: 1.0},
		{X: 1.0, Y: 1.0, Z: 1.0},
		{X: -1.0, Y: 1.0, Z: 1.0},
	}

	faces := [][]int{
		{0, 1, 2, 3}, // Front
		{5, 4, 7, 6}, // Back
		{1, 5, 6, 2}, // Right
		{4, 0, 3, 7}, // Left
		{3, 2, 6, 7}, // Top
		{4, 5, 1, 0}, // Bottom
	}

	projected := make([]struct {
		x, y    float64
		visible bool
	}, len(vertices))

	for i, v := range vertices {
		v = v.RotateY(rotY)
		v = v.RotateX(rotX)
		px, py, visible := graphics.Project(v, float64(virtualW), float64(virtualH), 3.5, float64(virtualH)*0.40)
		projected[i] = struct {
			x, y    float64
			visible bool
		}{x: px, y: py, visible: visible}
	}

	wireStyle := cell.Style{Fg: cell.NewColorRGB(0, 180, 255)}

	for _, face := range faces {
		p0 := projected[face[0]]
		p1 := projected[face[1]]
		p2 := projected[face[2]]
		p3 := projected[face[3]]

		if !p0.visible || !p1.visible || !p2.visible || !p3.visible {
			continue
		}

		// Backface culling
		cross := (p1.x-p0.x)*(p2.y-p0.y) - (p1.y-p0.y)*(p2.x-p0.x)
		if cross >= 0 {
			continue
		}

		canvas.DrawLine(int(p0.x), int(p0.y), int(p1.x), int(p1.y), wireStyle)
		canvas.DrawLine(int(p1.x), int(p1.y), int(p2.x), int(p2.y), wireStyle)
		canvas.DrawLine(int(p2.x), int(p2.y), int(p3.x), int(p3.y), wireStyle)
		canvas.DrawLine(int(p3.x), int(p3.y), int(p0.x), int(p0.y), wireStyle)
	}
}
