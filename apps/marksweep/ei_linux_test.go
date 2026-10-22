//go:build linux

package main

import (
	"encoding/binary"
	"io"
	"math"
	"net"
	"testing"
	"time"
)

// eisMsg builds a message as an EIS implementation sends it.
type eisMsg []byte

func msg(object uint64, opcode uint32) *eisMsg {
	m := eisMsg(binary.NativeEndian.AppendUint64(nil, object))
	m = binary.NativeEndian.AppendUint32(m, 0)
	m = binary.NativeEndian.AppendUint32(m, opcode)
	return &m
}

func (m *eisMsg) u32(v uint32) *eisMsg { *m = binary.NativeEndian.AppendUint32(*m, v); return m }
func (m *eisMsg) u64(v uint64) *eisMsg { *m = binary.NativeEndian.AppendUint64(*m, v); return m }
func (m *eisMsg) f32(v float32) *eisMsg {
	return m.u32(math.Float32bits(v))
}
func (m *eisMsg) str(s string) *eisMsg {
	m.u32(uint32(len(s) + 1))
	*m = append(append(*m, s...), 0)
	for len(*m)%4 != 0 {
		*m = append(*m, 0)
	}
	return m
}
func (m *eisMsg) bytes() []byte {
	binary.NativeEndian.PutUint32((*m)[8:], uint32(len(*m)))
	return *m
}

// readMsg reads one message the client sent: its object, opcode and body.
func readMsg(t *testing.T, r io.Reader) (uint64, uint32, []byte) {
	t.Helper()
	head := make([]byte, 16)
	if _, err := io.ReadFull(r, head); err != nil {
		t.Fatal(err)
	}
	body := make([]byte, binary.NativeEndian.Uint32(head[8:])-16)
	if _, err := io.ReadFull(r, body); err != nil {
		t.Fatal(err)
	}
	return binary.NativeEndian.Uint64(head), binary.NativeEndian.Uint32(head[12:]), body
}

// The client goes through the handshake, binds the seat to the pointer and
// its buttons, answers a ping, and turns motion and buttons into the
// mouse's movement and presses — against a server speaking libei 1.6.0's
// protocol the way the compositor does.
func TestEIClientTakesTheMouse(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	server.SetDeadline(time.Now().Add(5 * time.Second))
	c := newEIClient(client)
	done := make(chan error, 1)
	go func() { done <- c.run() }()

	send := func(m *eisMsg) {
		if _, err := server.Write(m.bytes()); err != nil {
			t.Fatal(err)
		}
	}
	expect := func(object uint64, opcode uint32) []byte {
		t.Helper()
		o, op, body := readMsg(t, server)
		if o != object || op != opcode {
			t.Fatalf("got object %d opcode %d, want %d %d", o, op, object, opcode)
		}
		return body
	}

	send(msg(0, 0).u32(1)) // handshake_version
	if v := binary.NativeEndian.Uint32(expect(0, 0)); v != 1 {
		t.Fatalf("handshake version %d", v)
	}
	if ctx := binary.NativeEndian.Uint32(expect(0, 2)); ctx != 1 {
		t.Fatalf("context type %d, want receiver", ctx)
	}
	expect(0, 3) // name
	for range eiInterfaces {
		expect(0, 4)
	}
	expect(0, 1) // finish

	const conn, seat, device, ptr, btn, ping = 0xff00000000000001, 0xff00000000000002, 0xff00000000000003, 0xff00000000000004, 0xff00000000000005, 0xff00000000000006
	send(msg(0, 2).u32(7).u64(conn).u32(1)) // connection
	send(msg(conn, 1).u64(seat).u32(1))     // seat
	send(msg(seat, 1).str("default"))       // name
	send(msg(seat, 2).u64(1 << 1).str("ei_pointer"))
	send(msg(seat, 2).u64(1 << 3).str("ei_keyboard"))
	send(msg(seat, 2).u64(1 << 4).str("ei_button"))
	send(msg(seat, 3)) // done
	if mask := binary.NativeEndian.Uint64(expect(seat, 1)); mask != 1<<1|1<<4 {
		t.Fatalf("bound %b, want the pointer and the buttons only", mask)
	}
	send(msg(conn, 3).u64(ping).u32(1))
	expect(ping, 0)
	send(msg(seat, 4).u64(device).u32(1))
	send(msg(device, 5).u64(ptr).str("ei_pointer").u32(1))
	send(msg(device, 5).u64(btn).str("ei_button").u32(1))
	send(msg(device, 6))                    // done
	send(msg(device, 7).u32(8))             // resumed
	send(msg(device, 9).u32(9).u32(1))      // start_emulating
	send(msg(ptr, 1).f32(3.5).f32(-1))      // motion_relative
	send(msg(ptr, 1).f32(1.25).f32(2))      // motion_relative
	send(msg(btn, 1).u32(btnLeft).u32(1))   // left down
	send(msg(btn, 1).u32(btnLeft).u32(0))   // left up
	send(msg(btn, 1).u32(btnRight).u32(1))  // right down
	send(msg(device, 11).u32(10).u64(1234)) // frame
	send(msg(conn, 3).u64(ping + 1).u32(1)) // a ping after it all: answered once all is read
	expect(ping+1, 0)

	if dx, dy := c.motion(); dx != 4.75 || dy != 1 {
		t.Fatalf("motion %.2f,%.2f, want 4.75,1", dx, dy)
	}
	want := []pointerEvent{{0, true}, {0, false}, {2, true}}
	for i, w := range want {
		select {
		case got := <-c.press:
			if got != w {
				t.Fatalf("button %d: %+v, want %+v", i, got, w)
			}
		default:
			t.Fatalf("button %d missing", i)
		}
	}
	send(msg(conn, 0).u32(0).u32(0).str("bye")) // disconnected
	if err := <-done; err != io.EOF {
		t.Fatalf("run ended with %v, want EOF", err)
	}
}

// A message whose length cannot be is the end of the connection, not a
// read past the buffer.
func TestEIClientRefusesABrokenMessage(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	c := newEIClient(client)
	done := make(chan error, 1)
	go func() { done <- c.run() }()
	head := msg(0, 0).bytes()
	binary.NativeEndian.PutUint32(head[8:], 4) // a length shorter than the header
	server.Write(head)
	if err := <-done; err == nil {
		t.Fatal("a message shorter than its header was read")
	}
}
