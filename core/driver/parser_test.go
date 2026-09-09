package driver

import "testing"

func TestParseShiftTab(t *testing.T) {
	event, consumed := ParseEvent([]byte("\x1b[Z"))
	if consumed != 3 {
		t.Fatalf("consumed = %d; want 3", consumed)
	}
	if event.Type != EventKey || event.Key.Type != KeyTab || !event.Key.Shift {
		t.Fatalf("event = %+v; want Shift+Tab", event)
	}
}

func TestParseSGRMouse(t *testing.T) {
	// 1. Left Click Press
	ev, consumed := ParseEvent([]byte("\x1b[<0;10;20M"))
	if consumed != 11 || ev.Type != EventMouse || ev.Mouse.Button != MouseLeft || ev.Mouse.Drag || ev.Mouse.X != 9 || ev.Mouse.Y != 19 {
		t.Errorf("Left click press failed: %+v (consumed %d)", ev, consumed)
	}

	// 2. Pure Hover / Pointer Motion (no buttons down)
	ev, consumed = ParseEvent([]byte("\x1b[<35;15;25M"))
	if consumed != 12 || ev.Type != EventMouse || ev.Mouse.Button != MouseNone || ev.Mouse.Drag {
		t.Errorf("Hover motion failed: %+v (consumed %d), expected Button=MouseNone and Drag=false", ev, consumed)
	}

	// 3. Drag with Left Button
	ev, consumed = ParseEvent([]byte("\x1b[<32;15;25M"))
	if consumed != 12 || ev.Type != EventMouse || ev.Mouse.Button != MouseLeft || !ev.Mouse.Drag {
		t.Errorf("Left drag failed: %+v (consumed %d), expected Button=MouseLeft and Drag=true", ev, consumed)
	}

	// 4. Drag with Right Button
	ev, consumed = ParseEvent([]byte("\x1b[<34;15;25M"))
	if consumed != 12 || ev.Type != EventMouse || ev.Mouse.Button != MouseRight || !ev.Mouse.Drag {
		t.Errorf("Right drag failed: %+v (consumed %d), expected Button=MouseRight and Drag=true", ev, consumed)
	}

	// 5. Button Release
	ev, consumed = ParseEvent([]byte("\x1b[<0;10;20m"))
	if consumed != 11 || ev.Type != EventMouse || ev.Mouse.Button != MouseRelease || ev.Mouse.Drag {
		t.Errorf("Release failed: %+v (consumed %d)", ev, consumed)
	}

	// 6. Scroll Up & Down
	ev, _ = ParseEvent([]byte("\x1b[<64;10;20M"))
	if ev.Type != EventMouse || ev.Mouse.Button != MouseScrollUp {
		t.Errorf("Scroll up failed: %+v", ev)
	}
	ev, _ = ParseEvent([]byte("\x1b[<65;10;20M"))
	if ev.Type != EventMouse || ev.Mouse.Button != MouseScrollDown {
		t.Errorf("Scroll down failed: %+v", ev)
	}
}
