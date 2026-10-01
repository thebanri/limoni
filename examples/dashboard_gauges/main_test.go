package main

import (
	"strings"
	"testing"

	"github.com/thebanri/limoni/uitest"
)

func hasRuneIn(s string, lo, hi rune) bool {
	for _, r := range s {
		if r >= lo && r <= hi {
			return true
		}
	}
	return false
}

func TestDashboardDrawsEveryWidget(t *testing.T) {
	page := uitest.Run(t, 100, 30, newDashboard().draw)
	screen := page.Screen()
	for _, want := range []string{"BigText", "CPU", "Memory", "api", "wasm", "sextant marker", "Space pause"} {
		if !strings.Contains(screen, want) {
			t.Errorf("%q missing from the screen:\n%s", want, screen)
		}
	}
	if !strings.ContainsAny(screen, "━") {
		t.Errorf("no LineGauge line on the screen:\n%s", screen)
	}
	if !hasRuneIn(screen, 0x1FB00, 0x1FB3B) {
		t.Errorf("the chart is not drawn in sextants:\n%s", screen)
	}

	page.Press("m")
	screen = page.Screen()
	if !strings.Contains(screen, "braille marker") || !hasRuneIn(screen, 0x2801, 0x28FF) {
		t.Errorf("m did not switch the chart to Braille:\n%s", screen)
	}

	page.Press("space")
	if screen := page.Screen(); !strings.Contains(screen, "paused") {
		t.Errorf("Space did not pause:\n%s", screen)
	}
}
