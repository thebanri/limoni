package backend

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
