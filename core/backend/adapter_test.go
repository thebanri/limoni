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

func TestPlatformCapabilityMatrix(t *testing.T) {
	linuxCaps := GetPlatformCapabilities("linux")
	if !linuxCaps.HasRawMode || !linuxCaps.HasIoctlResize || linuxCaps.OS != "linux" {
		t.Errorf("linux capabilities = %+v", linuxCaps)
	}

	windowsCaps := GetPlatformCapabilities("windows")
	if !windowsCaps.HasRawMode || windowsCaps.HasIoctlResize || windowsCaps.OS != "windows" {
		t.Errorf("windows capabilities = %+v", windowsCaps)
	}

	unknownCaps := GetPlatformCapabilities("unknown")
	if unknownCaps.HasRawMode || unknownCaps.HasIoctlResize {
		t.Errorf("unknown capabilities = %+v", unknownCaps)
	}
}
