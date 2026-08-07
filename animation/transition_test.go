package animation

import (
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

func TestApplyDitherFade(t *testing.T) {
	oldBuf := buffer.NewBuffer(cell.NewRect(0, 0, 4, 4))
	newBuf := buffer.NewBuffer(cell.NewRect(0, 0, 4, 4))

	// Fill oldBuf with 'A'
	for i := range oldBuf.Content {
		oldBuf.Content[i] = cell.Cell{Content: 'A'}
	}

	// Fill newBuf with 'B'
	for i := range newBuf.Content {
		newBuf.Content[i] = cell.Cell{Content: 'B'}
	}

	// Test progress = 0.0 (should copy oldBuf entirely, so all cells are 'A')
	testBuf := buffer.NewBuffer(cell.NewRect(0, 0, 4, 4))
	copy(testBuf.Content, newBuf.Content)
	ApplyDitherFade(testBuf, oldBuf, 0.0)

	for _, c := range testBuf.Content {
		if c.Content != 'A' {
			t.Errorf("Expected cell to be 'A' at progress 0.0, got '%c'", c.Content)
		}
	}

	// Test progress = 0.5 (half 'A's, half 'B's based on Bayer threshold)
	testBuf2 := buffer.NewBuffer(cell.NewRect(0, 0, 4, 4))
	copy(testBuf2.Content, newBuf.Content)
	ApplyDitherFade(testBuf2, oldBuf, 0.5)

	countA := 0
	countB := 0
	for _, c := range testBuf2.Content {
		if c.Content == 'A' {
			countA++
		} else if c.Content == 'B' {
			countB++
		}
	}

	// For a 4x4 grid (16 cells) and a 0.5 threshold, exactly 7 cells should be 'A' (threshold > 0.5)
	// and 9 cells should be 'B' (threshold <= 0.5)
	if countA != 7 || countB != 9 {
		t.Errorf("Expected 7 'A's and 9 'B's at progress 0.5, got %d 'A's and %d 'B's", countA, countB)
	}

	// Test progress = 1.0 (should remain newBuf completely, so all 'B')
	testBuf3 := buffer.NewBuffer(cell.NewRect(0, 0, 4, 4))
	copy(testBuf3.Content, newBuf.Content)
	ApplyDitherFade(testBuf3, oldBuf, 1.0)

	for _, c := range testBuf3.Content {
		if c.Content != 'B' {
			t.Errorf("Expected cell to be 'B' at progress 1.0, got '%c'", c.Content)
		}
	}
}
