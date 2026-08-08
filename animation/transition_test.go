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

	// Test progress = 0.5: glyph içeren satırlar bütün olarak geçer
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

	// Dört satırın iki tanesi eski, iki tanesi yeni frame'de kalmalı.
	if countA != 8 || countB != 8 {
		t.Errorf("Expected 8 'A's and 8 'B's at progress 0.5, got %d 'A's and %d 'B's", countA, countB)
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
