package animation

import (
	"testing"
)

func TestEasingBoundaries(t *testing.T) {
	funcs := []struct {
		name string
		f    EasingFunc
	}{
		{"Linear", Linear},
		{"EaseInQuad", EaseInQuad},
		{"EaseOutQuad", EaseOutQuad},
		{"EaseInOutQuad", EaseInOutQuad},
		{"EaseInCubic", EaseInCubic},
		{"EaseOutCubic", EaseOutCubic},
		{"EaseInOutCubic", EaseInOutCubic},
		{"EaseInSine", EaseInSine},
		{"EaseOutSine", EaseOutSine},
		{"EaseInOutSine", EaseInOutSine},
		{"EaseInExpo", EaseInExpo},
		{"EaseOutExpo", EaseOutExpo},
		{"EaseInOutExpo", EaseInOutExpo},
		{"EaseOutBounce", EaseOutBounce},
		{"EaseInBounce", EaseInBounce},
		{"EaseInOutBounce", EaseInOutBounce},
	}

	for _, tc := range funcs {
		t.Run(tc.name, func(t *testing.T) {
			// Başlangıç noktası (0.0) doğrulaması
			val0 := tc.f(0.0)
			if val0 < -0.0001 || val0 > 0.0001 {
				t.Errorf("%s(0.0) = %v; 0.0 bekleniyordu", tc.name, val0)
			}

			// Bitiş noktası (1.0) doğrulaması
			val1 := tc.f(1.0)
			if val1 < 0.9999 || val1 > 1.0001 {
				t.Errorf("%s(1.0) = %v; 1.0 bekleniyordu", tc.name, val1)
			}
		})
	}
}
