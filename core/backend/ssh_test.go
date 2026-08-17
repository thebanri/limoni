package backend

import (
	"bytes"
	"testing"
	"time"
)

type mockSSHSession struct {
	in     *bytes.Buffer
	out    *bytes.Buffer
	width  uint16
	height uint16
}

func (m *mockSSHSession) Read(p []byte) (int, error) {
	return m.in.Read(p)
}

func (m *mockSSHSession) Write(p []byte) (int, error) {
	return m.out.Write(p)
}

func (m *mockSSHSession) Size() (uint16, uint16, error) {
	return m.width, m.height, nil
}

func TestSSHBackend(t *testing.T) {
	session := &mockSSHSession{
		in:     bytes.NewBufferString("a"),
		out:    new(bytes.Buffer),
		width:  100,
		height: 30,
	}

	b := NewSSHBackend(session)
	if err := b.Setup(); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	w, h, err := b.Size()
	if err != nil || w != 100 || h != 30 {
		t.Fatalf("unexpected size: (%d, %d), err: %v", w, h, err)
	}

	b.StartEventLoop()

	// Verify events are delivered from SSH input
	select {
	case ev := <-b.Events():
		if ev.Type != EventKey || ev.Key.Ch != 'a' {
			t.Fatalf("expected EventKey 'a', got %+v", ev)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for event from SSH input")
	}

	// Test dynamic resize
	b.SetSize(120, 40)
	select {
	case ev := <-b.Events():
		if ev.Type != EventResize || ev.Resize.Width != 120 || ev.Resize.Height != 40 {
			t.Fatalf("expected Resize (120, 40), got %+v", ev)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for resize event")
	}

	if err := b.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}
