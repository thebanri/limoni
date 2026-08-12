package accessibility

import (
	"bytes"
	"testing"
)

func TestLineModeAdapterAnnounceAndWrite(t *testing.T) {
	var lm LineModeAdapter
	err := lm.Announce("Hello")
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	nodes := []AccessibilityNode{
		{ID: "btn", Role: RoleButton, Label: "Click"},
	}
	mode := Mode{ScreenReader: true}
	err = lm.WriteTree(&buf, mode, nodes)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("Click")) {
		t.Errorf("expected Click in tree output, got %q", buf.String())
	}
}

func TestNewPlatformScreenReaderAdapter(t *testing.T) {
	linuxAdapter := NewPlatformScreenReaderAdapter("linux")
	if _, ok := linuxAdapter.(LinuxScreenReaderAdapter); !ok {
		t.Errorf("expected LinuxScreenReaderAdapter, got %T", linuxAdapter)
	}

	darwinAdapter := NewPlatformScreenReaderAdapter("darwin")
	if _, ok := darwinAdapter.(MacOSScreenReaderAdapter); !ok {
		t.Errorf("expected MacOSScreenReaderAdapter, got %T", darwinAdapter)
	}

	windowsAdapter := NewPlatformScreenReaderAdapter("windows")
	if _, ok := windowsAdapter.(WindowsScreenReaderAdapter); !ok {
		t.Errorf("expected WindowsScreenReaderAdapter, got %T", windowsAdapter)
	}

	unknownAdapter := NewPlatformScreenReaderAdapter("unknown")
	if _, ok := unknownAdapter.(LineModeAdapter); !ok {
		t.Errorf("expected LineModeAdapter, got %T", unknownAdapter)
	}
}
