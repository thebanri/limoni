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

func TestParseKittyKeyboardProtocol(t *testing.T) {
	// Shift+Enter (\x1b[13;2u) -> 7 bytes
	ev, consumed := ParseEvent([]byte("\x1b[13;2u"))
	if consumed != 7 || ev.Type != EventKey || ev.Key.Type != KeyEnter || !ev.Key.Shift {
		t.Errorf("Shift+Enter CSI u failed: %+v (consumed %d)", ev, consumed)
	}

	// Alt+Enter (\x1b[13;3u) -> 7 bytes
	ev, consumed = ParseEvent([]byte("\x1b[13;3u"))
	if consumed != 7 || ev.Type != EventKey || ev.Key.Type != KeyEnter || !ev.Key.Alt {
		t.Errorf("Alt+Enter CSI u failed: %+v (consumed %d)", ev, consumed)
	}

	// Ctrl+Enter (\x1b[13;5u) -> 7 bytes
	ev, consumed = ParseEvent([]byte("\x1b[13;5u"))
	if consumed != 7 || ev.Type != EventKey || ev.Key.Type != KeyEnter || !ev.Key.Ctrl {
		t.Errorf("Ctrl+Enter CSI u failed: %+v (consumed %d)", ev, consumed)
	}

	// Shift+Tab (\x1b[9;2u) -> 6 bytes
	ev, consumed = ParseEvent([]byte("\x1b[9;2u"))
	if consumed != 6 || ev.Type != EventKey || ev.Key.Type != KeyTab || !ev.Key.Shift {
		t.Errorf("Shift+Tab CSI u failed: %+v (consumed %d)", ev, consumed)
	}

	// Shift+Space (\x1b[32;2u) -> 7 bytes
	ev, consumed = ParseEvent([]byte("\x1b[32;2u"))
	if consumed != 7 || ev.Type != EventKey || ev.Key.Type != KeySpace || !ev.Key.Shift {
		t.Errorf("Shift+Space CSI u failed: %+v (consumed %d)", ev, consumed)
	}
}

func TestParseModifyOtherKeys(t *testing.T) {
	// Shift+Enter (\x1b[27;2;13~) -> 10 bytes
	ev, consumed := ParseEvent([]byte("\x1b[27;2;13~"))
	if consumed != 10 || ev.Type != EventKey || ev.Key.Type != KeyEnter || !ev.Key.Shift {
		t.Errorf("Shift+Enter modifyOtherKeys failed: %+v (consumed %d)", ev, consumed)
	}

	// Ctrl+Enter (\x1b[27;5;13~) -> 10 bytes
	ev, consumed = ParseEvent([]byte("\x1b[27;5;13~"))
	if consumed != 10 || ev.Type != EventKey || ev.Key.Type != KeyEnter || !ev.Key.Ctrl {
		t.Errorf("Ctrl+Enter modifyOtherKeys failed: %+v (consumed %d)", ev, consumed)
	}

	// Alt+Enter (\x1b\r) -> 2 bytes
	ev, consumed = ParseEvent([]byte("\x1b\r"))
	if consumed != 2 || ev.Type != EventKey || ev.Key.Type != KeyEnter || !ev.Key.Alt {
		t.Errorf("Alt+Enter \\x1b\\r failed: %+v (consumed %d)", ev, consumed)
	}
}

