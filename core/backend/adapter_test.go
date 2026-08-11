package backend

import "testing"

func TestMemoryPTYAdapterLifecycleAndResize(t *testing.T) {
	adapter := NewMemoryPTYAdapter(80, 24)
	if err := adapter.Start(); err != nil {
		t.Fatal(err)
	}
	if err := adapter.Resize(120, 40); err != nil {
		t.Fatal(err)
	}
	if width, height, _ := adapter.Size(); width != 120 || height != 40 {
		t.Fatalf("size=%dx%d", width, height)
	}
	if err := adapter.Stop(); err != nil {
		t.Fatal(err)
	}
}
