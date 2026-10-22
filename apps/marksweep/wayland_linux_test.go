//go:build linux

package main

import (
	"encoding/binary"
	"io"
	"net"
	"os"
	"syscall"
	"testing"
	"time"
)

// wlMsg builds an event as a compositor sends it.
type wlMsg []byte

func ev(object, opcode uint32) *wlMsg {
	m := wlMsg(binary.NativeEndian.AppendUint32(nil, object))
	m = binary.NativeEndian.AppendUint32(m, opcode)
	return &m
}

func (m *wlMsg) u32(v uint32) *wlMsg { *m = binary.NativeEndian.AppendUint32(*m, v); return m }
func (m *wlMsg) arr(b []byte) *wlMsg {
	m.u32(uint32(len(b)))
	*m = append(*m, b...)
	for len(*m)%4 != 0 {
		*m = append(*m, 0)
	}
	return m
}
func (m *wlMsg) str(s string) *wlMsg { return m.arr(append([]byte(s), 0)) }
func (m *wlMsg) bytes() []byte {
	w := binary.NativeEndian.Uint32((*m)[4:])
	binary.NativeEndian.PutUint32((*m)[4:], uint32(len(*m))<<16|w)
	return *m
}

// fakeCompositor is the other end of the socket: it reads requests one by
// one, with the descriptors that came with them.
type fakeCompositor struct {
	t    *testing.T
	conn *net.UnixConn
	in   []byte
	fds  []int
}

type request struct {
	object, opcode uint32
	body           []byte
}

func (s *fakeCompositor) next() request {
	s.t.Helper()
	for len(s.in) < 8 || len(s.in) < int(binary.NativeEndian.Uint32(s.in[4:])>>16) {
		buf, oob := make([]byte, 4096), make([]byte, 64)
		n, oobn, _, _, err := s.conn.ReadMsgUnix(buf, oob)
		if err != nil {
			s.t.Fatal(err)
		}
		if oobn > 0 {
			msgs, _ := syscall.ParseSocketControlMessage(oob[:oobn])
			for _, m := range msgs {
				fds, _ := syscall.ParseUnixRights(&m)
				s.fds = append(s.fds, fds...)
			}
		}
		s.in = append(s.in, buf[:n]...)
	}
	w := binary.NativeEndian.Uint32(s.in[4:])
	r := request{binary.NativeEndian.Uint32(s.in), w & 0xffff, s.in[8 : w>>16]}
	s.in = s.in[w>>16:]
	return r
}

func (s *fakeCompositor) expect(object, opcode uint32) *args {
	s.t.Helper()
	r := s.next()
	if r.object != object || r.opcode != opcode {
		s.t.Fatalf("got object %d opcode %d, want %d %d", r.object, r.opcode, object, opcode)
	}
	return &args{b: r.body}
}

func (s *fakeCompositor) send(m *wlMsg, fds ...int) {
	var oob []byte
	if len(fds) > 0 {
		oob = syscall.UnixRights(fds...)
	}
	if _, _, err := s.conn.WriteMsgUnix(m.bytes(), oob, nil); err != nil {
		s.t.Fatal(err)
	}
}

func socketPair(t *testing.T) (*net.UnixConn, *net.UnixConn) {
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		t.Fatal(err)
	}
	conn := func(fd int) *net.UnixConn {
		f := os.NewFile(uintptr(fd), "wl")
		defer f.Close()
		c, err := net.FileConn(f)
		if err != nil {
			t.Fatal(err)
		}
		return c.(*net.UnixConn)
	}
	return conn(fds[0]), conn(fds[1])
}

func waitFor(t *testing.T, c *wlClient, want ...wlEvent) {
	t.Helper()
	for i, w := range want {
		select {
		case got := <-c.events:
			if got != w {
				t.Fatalf("event %d: %+v, want %+v", i, got, w)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("event %d (%+v) never came", i, w)
		}
	}
}

// The client binds what the lock needs, opens a transparent full-screen
// window locked to the pointer, and turns the compositor's events into
// the keys and motion the game reads — against a compositor scripted from
// the protocols' XML.
func TestTheWindowTakesTheMouseAndTheKeys(t *testing.T) {
	server, client := socketPair(t)
	defer server.Close()
	server.SetDeadline(time.Now().Add(5 * time.Second))
	s := &fakeCompositor{t: t, conn: server}
	c := newWLClient(client)
	ready := make(chan error, 1)
	go func() { ready <- c.setUp() }()

	registry := s.expect(1, 1).u32() // get_registry
	done := s.expect(1, 0).u32()     // sync
	for i, name := range append(wlNeeds, "wl_output") {
		s.send(ev(registry, 0).u32(uint32(40 + i)).str(name).u32(9))
	}
	s.send(ev(done, 0).u32(0))
	if err := <-ready; err != nil {
		t.Fatal(err)
	}
	ids := map[string]uint32{}
	for i, name := range wlNeeds {
		a := s.expect(registry, 0)
		if g, iface := a.u32(), a.str(); g != uint32(40+i) || iface != name {
			t.Fatalf("bound %d %q, want %d %q", g, iface, 40+i, name)
		}
		a.u32()
		ids[name] = a.u32()
	}
	go c.run(nil)

	s.send(ev(ids["wl_seat"], 0).u32(3)) // capabilities: pointer and keyboard
	ptr := s.expect(ids["wl_seat"], 0).u32()
	rel := s.expect(ids["zwp_relative_pointer_manager_v1"], 1).u32()
	kbd := s.expect(ids["wl_seat"], 1).u32()

	c.show()
	surface := s.expect(ids["wl_compositor"], 0).u32()
	xdgSurface := s.expect(ids["xdg_wm_base"], 2).u32()
	toplevel := s.expect(xdgSurface, 1).u32()
	s.expect(toplevel, 2) // title
	s.expect(toplevel, 3) // app id
	s.expect(toplevel, 11)
	a := s.expect(ids["zwp_pointer_constraints_v1"], 1)
	lock := a.u32()
	if sf, p, region, life := a.u32(), a.u32(), a.u32(), a.u32(); sf != surface || p != ptr || region != 0 || life != 2 {
		t.Fatalf("lock_pointer(%d, %d, %d, %d)", sf, p, region, life)
	}
	s.expect(surface, 6) // the first commit, bufferless

	s.send(ev(toplevel, 0).u32(640).u32(480).arr(nil))
	s.send(ev(xdgSurface, 0).u32(77))
	if serial := s.expect(xdgSurface, 4).u32(); serial != 77 {
		t.Fatalf("acked %d", serial)
	}
	a = s.expect(ids["wl_shm"], 0)
	pool := a.u32()
	if size := a.u32(); size != 640*480*4 || len(s.fds) != 1 {
		t.Fatalf("pool of %d bytes with %d descriptors", size, len(s.fds))
	}
	if fi, err := os.NewFile(uintptr(s.fds[0]), "pool").Stat(); err != nil || fi.Size() != 640*480*4 {
		t.Fatalf("the pool's file: %v %v", fi, err)
	}
	a = s.expect(pool, 0)
	buffer := a.u32()
	if off, w, h, stride, format := a.u32(), a.u32(), a.u32(), a.u32(), a.u32(); off != 0 || w != 640 || h != 480 || stride != 2560 || format != 0 {
		t.Fatalf("create_buffer %d %d %d %d %d", off, w, h, stride, format)
	}
	if b := s.expect(surface, 1).u32(); b != buffer {
		t.Fatalf("attached %d", b)
	}
	s.expect(surface, 2)
	s.expect(surface, 6)

	keymap, err := os.CreateTemp(t.TempDir(), "keymap")
	if err != nil {
		t.Fatal(err)
	}
	s.send(ev(kbd, 0).u32(1).u32(0), int(keymap.Fd())) // keymap: its descriptor is closed
	s.send(ev(ptr, 0).u32(5).u32(surface).u32(0).u32(0))
	if a := s.expect(ptr, 0); a.u32() != 5 || a.u32() != 0 {
		t.Fatal("the pointer was not hidden")
	}
	held := binary.NativeEndian.AppendUint32(nil, 17) // W, held as the window got the keyboard
	s.send(ev(kbd, 1).u32(6).u32(surface).arr(held))
	s.send(ev(kbd, 3).u32(7).u32(0).u32(17).u32(0))
	s.send(ev(ptr, 3).u32(8).u32(0).u32(btnLeft).u32(1))
	minus2 := int32(-2 * 256)
	s.send(ev(rel, 0).u32(0).u32(0).u32(999 * 256).u32(0).u32(3*256 + 128).u32(uint32(minus2)))
	s.send(ev(lock, 0))
	s.send(ev(kbd, 2).u32(9).u32(surface + 100)) // a leave from a window of before
	s.send(ev(kbd, 2).u32(9).u32(surface))
	waitFor(t, c, wlEvent{kind: wlEnter}, wlEvent{wlKey, 17, true}, wlEvent{wlKey, 17, false},
		wlEvent{wlButton, btnLeft, true}, wlEvent{kind: wlLeave})
	select {
	case e := <-c.events:
		t.Fatalf("one event too many: %+v", e)
	case <-time.After(50 * time.Millisecond):
	}
	if !c.locked.Load() {
		t.Fatal("not locked")
	}
	if dx, dy := c.motion(); dx != 3.5 || dy != -2 {
		t.Fatalf("motion %v,%v, want the unaccelerated 3.5,-2", dx, dy)
	}
	c.mu.Lock()
	left := len(c.fds)
	c.mu.Unlock()
	if left != 0 {
		t.Fatalf("%d descriptors kept", left)
	}

	c.hide()
	for _, o := range []uint32{lock, toplevel, xdgSurface, surface, buffer} {
		s.expect(o, 0)
	}
	s.expect(pool, 1)
	if c.locked.Load() {
		t.Fatal("still locked after the window went")
	}
	server.Close()
	select {
	case e := <-c.events:
		if e.kind != wlLeave {
			t.Fatalf("%+v at the end", e)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the end of the connection was not a leave")
	}
}

// A message whose size cannot be ends the connection, not a read past
// the buffer.
func TestTheWindowRefusesABrokenMessage(t *testing.T) {
	server, client := socketPair(t)
	defer server.Close()
	c := newWLClient(client)
	done := make(chan error, 1)
	go func() { done <- c.run(nil) }()
	server.Write(binary.NativeEndian.AppendUint32(binary.NativeEndian.AppendUint32(nil, 1), 4<<16))
	if err := <-done; err == nil || err == io.EOF {
		t.Fatalf("a message shorter than its header: %v", err)
	}
}

// The keys come from where they sit on the keyboard, and the game reads
// them as the terminal's: a key held through a card's pick and let go
// after it is let go, and the window losing the keyboard lets go of all.
func TestKeysFromTheWindowAreTheKeysHeld(t *testing.T) {
	g := newGame(1)
	g.reset()
	p := &pointer{shown: true}
	g.phase = phCards
	g.nCards = 3
	p.event(g, wlEvent{kind: wlEnter})
	p.event(g, wlEvent{wlKey, 17, true}) // W
	p.event(g, wlEvent{wlKey, 2, true})  // 1: the first card
	if g.phase != phPlay || !g.held[actFwd] {
		t.Fatalf("after the pick: phase %v, W held %v", g.phase, g.held[actFwd])
	}
	p.event(g, wlEvent{wlKey, 17, false})
	if g.held[actFwd] {
		t.Fatal("W let go after the pick is still held")
	}
	p.event(g, wlEvent{wlKey, 30, true}) // A
	p.event(g, wlEvent{kind: wlLeave})
	if g.held[actLeft] || g.phase != phPause {
		t.Fatalf("after the window lost the keyboard: A held %v, phase %v", g.held[actLeft], g.phase)
	}
}
