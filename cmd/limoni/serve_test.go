//go:build linux || darwin

package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// testWS is the browser's side of the connection: it masks what it sends,
// as a real client must.
type testWS struct {
	t    *testing.T
	conn net.Conn
	r    *bufio.Reader
	seen bytes.Buffer // everything the program printed so far
}

func dialWS(t *testing.T, srv *httptest.Server, query, origin string) (*testWS, *http.Response) {
	t.Helper()
	conn, err := net.Dial("tcp", srv.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	var key [16]byte
	_, _ = rand.Read(key[:])
	req := "GET /ws?" + query + " HTTP/1.1\r\nHost: " + srv.Listener.Addr().String() +
		"\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Version: 13\r\nSec-WebSocket-Key: " +
		base64.StdEncoding.EncodeToString(key[:]) + "\r\n"
	if origin != "" {
		req += "Origin: " + origin + "\r\n"
	}
	if _, err := io.WriteString(conn, req+"\r\n"); err != nil {
		t.Fatal(err)
	}
	r := bufio.NewReader(conn)
	resp, err := http.ReadResponse(r, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		conn.Close()
		return nil, resp
	}
	ws := &testWS{t: t, conn: conn, r: r}
	t.Cleanup(func() { conn.Close() })
	return ws, resp
}

func (c *testWS) send(opcode byte, payload []byte) {
	c.t.Helper()
	var mask [4]byte
	_, _ = rand.Read(mask[:])
	frame := []byte{0x80 | opcode}
	switch l := len(payload); {
	case l <= 125:
		frame = append(frame, 0x80|byte(l))
	default:
		frame = append(frame, 0x80|126, byte(l>>8), byte(l))
	}
	frame = append(frame, mask[:]...)
	for i, b := range payload {
		frame = append(frame, b^mask[i%4])
	}
	if _, err := c.conn.Write(frame); err != nil {
		c.t.Fatal(err)
	}
}

// readFrame reads one unmasked server frame.
func (c *testWS) readFrame() (byte, []byte, error) {
	var head [2]byte
	if _, err := io.ReadFull(c.r, head[:]); err != nil {
		return 0, nil, err
	}
	length := int(head[1] & 0x7F)
	switch length {
	case 126:
		var ext [2]byte
		if _, err := io.ReadFull(c.r, ext[:]); err != nil {
			return 0, nil, err
		}
		length = int(binary.BigEndian.Uint16(ext[:]))
	case 127:
		var ext [8]byte
		if _, err := io.ReadFull(c.r, ext[:]); err != nil {
			return 0, nil, err
		}
		length = int(binary.BigEndian.Uint64(ext[:]))
	}
	payload := make([]byte, length)
	_, err := io.ReadFull(c.r, payload)
	return head[0] & 0x0F, payload, err
}

// waitFor reads output until it contains want, and fails after a few seconds.
func (c *testWS) waitFor(want string) {
	c.t.Helper()
	_ = c.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for !strings.Contains(c.seen.String(), want) {
		op, payload, err := c.readFrame()
		if err != nil {
			c.t.Fatalf("waiting for %q: %v; the program printed %q", want, err, c.seen.String())
		}
		if op == wsBinary {
			c.seen.Write(payload)
		}
	}
}

// waitClosed reads until the server's close frame.
func (c *testWS) waitClosed() {
	c.t.Helper()
	_ = c.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		op, _, err := c.readFrame()
		if err != nil {
			c.t.Fatalf("waiting for the close: %v", err)
		}
		if op == wsClose {
			return
		}
	}
}

func newTestServer(t *testing.T, maxConns int, command ...string) (*httptest.Server, *serveHandler) {
	t.Helper()
	h := newServeHandler(serveConfig{token: "secret", maxConns: maxConns, command: command})
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv, h
}

// The program runs in a real terminal of the size the browser asked for,
// reads what the browser types, follows a resize, and the browser hears when
// it ends.
func TestServeRunsTheProgramInATerminal(t *testing.T) {
	srv, _ := newTestServer(t, 2, "sh", "-c",
		`stty size; printf 'ready\n'; read line; echo "got:$line"; read again; stty size`)
	ws, resp := dialWS(t, srv, "token=secret&cols=100&rows=30", "")
	if ws == nil {
		t.Fatalf("upgrade refused: %s", resp.Status)
	}
	ws.waitFor("30 100")
	ws.waitFor("ready")
	ws.send(wsBinary, []byte("lemon\r"))
	ws.waitFor("got:lemon")
	ws.send(wsText, []byte(`{"type":"resize","cols":120,"rows":40}`))
	ws.send(wsBinary, []byte("\r"))
	ws.waitFor("40 120")
	ws.waitClosed()
}

// Without the token nothing is served, and another site's page cannot open
// the WebSocket even with it.
func TestServeRefusesStrangers(t *testing.T) {
	srv, _ := newTestServer(t, 2, "true")
	for _, path := range []string{"/", "/?token=wrong", "/ws?token=wrong"} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("GET %s: %s, want 403", path, resp.Status)
		}
	}
	resp, err := http.Get(srv.URL + "/?token=secret")
	if err != nil {
		t.Fatal(err)
	}
	page, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !bytes.Contains(page, []byte("xterm")) {
		t.Errorf("the page with the token: %s", resp.Status)
	}
	if ws, resp := dialWS(t, srv, "token=secret", "https://evil.example"); ws != nil || resp.StatusCode != http.StatusForbidden {
		t.Errorf("a foreign Origin was let in (status %d)", resp.StatusCode)
	}
	if ws, resp := dialWS(t, srv, "token=secret", "http://"+srv.Listener.Addr().String()); ws == nil {
		t.Errorf("the server's own Origin was refused: %s", resp.Status)
	}
}

// Closing the tab hangs the program up; one that ignores the hangup is
// killed. Either way it does not outlive its browser tab.
func TestServeHangsUpWhenTheTabCloses(t *testing.T) {
	for _, script := range []string{
		`echo started; sleep 30`,
		`trap '' HUP; echo started; sleep 30`,
	} {
		srv, h := newTestServer(t, 2, "sh", "-c", script)
		ws, resp := dialWS(t, srv, "token=secret", "")
		if ws == nil {
			t.Fatalf("upgrade refused: %s", resp.Status)
		}
		ws.waitFor("started")
		ws.conn.Close()
		deadline := time.Now().Add(5 * time.Second)
		for {
			h.mu.Lock()
			n := len(h.sessions)
			h.mu.Unlock()
			if n == 0 {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("%q: the program still runs after the tab closed", script)
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
}

// -max bounds how many programs run at once.
func TestServeLimitsConcurrentPrograms(t *testing.T) {
	srv, _ := newTestServer(t, 1, "sh", "-c", "echo one; sleep 30")
	first, _ := dialWS(t, srv, "token=secret", "")
	if first == nil {
		t.Fatal("the first connection was refused")
	}
	first.waitFor("one")
	if ws, resp := dialWS(t, srv, "token=secret", ""); ws != nil || resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("a second program started past -max 1 (status %d)", resp.StatusCode)
	}
}

// Frames the browser may send in pieces, or as pings, still make messages.
func TestWebSocketFragmentsAndPings(t *testing.T) {
	srv, _ := newTestServer(t, 1, "sh", "-c", `read line; echo "got:$line"`)
	ws, resp := dialWS(t, srv, "token=secret", "")
	if ws == nil {
		t.Fatalf("upgrade refused: %s", resp.Status)
	}
	ws.send(wsPing, []byte("hi"))
	_ = ws.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	if op, payload, err := ws.readFrame(); err != nil || op != wsPong || string(payload) != "hi" {
		t.Fatalf("ping answered with op %#x %q (%v)", op, payload, err)
	}
	// "ab" then "c\r" as one fragmented binary message.
	mask := []byte{1, 2, 3, 4}
	frag := func(first byte, data string) []byte {
		out := []byte{first, 0x80 | byte(len(data))}
		out = append(out, mask...)
		for i := 0; i < len(data); i++ {
			out = append(out, data[i]^mask[i%4])
		}
		return out
	}
	if _, err := ws.conn.Write(append(frag(wsBinary, "ab"), frag(0x80|wsContinuation, "c\r")...)); err != nil {
		t.Fatal(err)
	}
	ws.waitFor("got:abc")
}
