package widgets

import (
	"fmt"
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

type dummyListProvider struct {
	total int
}

func (p *dummyListProvider) Len() int {
	return p.total
}

func (p *dummyListProvider) ItemAt(index int) string {
	return fmt.Sprintf("Item %d", index)
}

func TestVirtualListRendering(t *testing.T) {
	provider := &dummyListProvider{total: 100}
	state := NewListState()
	state.Selected = 5

	list := List{
		Provider:  provider,
		State:     state,
		Scrollbar: true,
	}

	area := cell.NewRect(0, 0, 15, 10)
	buf := buffer.NewBuffer(area)
	ctx := cell.NewContext(area, cell.Style{})

	list.Draw(ctx, buf)

	// Since height is 10 and selected is 5, Offset should be 0.
	// Column 14 (rightmost) should contain the scrollbar.
	// Visible items should render on columns 0 to 13.

	// Check item rendering
	for y := 0; y < 10; y++ {
		// Visible row should contain "Item Y"
		expectedText := fmt.Sprintf("Item %d", y)
		actualText := ""
		for x := 0; x < 14; x++ {
			char := buf.Get(uint16(x), uint16(y)).Content
			if char != 0 {
				actualText += string(char)
			}
		}
		actualText = strings.TrimSpace(actualText)
		if actualText != expectedText {
			t.Errorf("Row %d: expected text %q, got %q", y, expectedText, actualText)
		}
	}

	// Check scrollbar track and thumb rendering
	scrollbarCol := uint16(14)
	trackCount := 0
	thumbCount := 0
	for y := 0; y < 10; y++ {
		char := buf.Get(scrollbarCol, uint16(y)).Content
		if char == '░' {
			trackCount++
		} else if char == '█' {
			thumbCount++
		} else {
			t.Errorf("Row %d: unexpected scrollbar char %q", y, string(char))
		}
	}

	if thumbCount == 0 {
		t.Error("Scrollbar did not render any thumb cells ('█')")
	}
	if trackCount == 0 {
		t.Error("Scrollbar did not render any track cells ('░')")
	}
}
