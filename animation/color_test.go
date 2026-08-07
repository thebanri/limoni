package animation

import (
	"testing"
	"time"

	"github.com/thebanri/limoni/core/cell"
)

func TestColorAnimationRGB(t *testing.T) {
	c1 := cell.NewColorRGB(100, 150, 200)
	c2 := cell.NewColorRGB(200, 100, 50)

	c := NewColor(c1)
	if c.Value() != c1 {
		t.Errorf("NewColor = %v; %v bekleniyordu", c.Value(), c1)
	}

	c.AnimateTo(c2, 100*time.Millisecond, Linear)
	now := time.Now()
	c.startTime = now

	// 50ms sonra (yarı yolda) güncelle
	c.Update(now.Add(50 * time.Millisecond))
	
	r, g, b := c.Value().RGB()
	if r != 150 || g != 125 || b != 125 {
		t.Errorf("50ms sonra RGB: (%d, %d, %d); (150, 125, 125) bekleniyordu", r, g, b)
	}

	// 100ms sonra güncelle
	c.Update(now.Add(100 * time.Millisecond))
	if c.Value() != c2 {
		t.Errorf("Süre sonunda renk %v; %v bekleniyordu", c.Value(), c2)
	}
}
