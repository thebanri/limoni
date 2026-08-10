package backend

import "testing"

func TestMemoryTerminalIO(t *testing.T) {
	io := NewMemoryTerminalIO([]byte("input"), 80, 24)
	buf := make([]byte, 5)
	if _, err := io.Read(buf); err != nil || string(buf) != "input" {
		t.Fatalf("read = %q/%v", buf, err)
	}
	if _, err := io.Write([]byte("output")); err != nil {
		t.Fatal(err)
	}
	if string(io.Output()) != "output" {
		t.Fatalf("output = %q", io.Output())
	}
	if w, h, _ := io.Size(); w != 80 || h != 24 {
		t.Fatalf("size = %dx%d", w, h)
	}
}
