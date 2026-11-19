package main

import (
	"strings"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// rowText is row y of a buffer as a string, without its trailing blanks.
func rowText(b *buffer.Buffer, y int) string {
	w := int(b.Area.Width)
	var s strings.Builder
	for _, c := range b.Content[y*w : (y+1)*w] {
		s.WriteRune(c.Content)
	}
	return strings.TrimRight(s.String(), " ")
}

func newBuf(w, h int) *buffer.Buffer {
	return buffer.NewBuffer(cell.NewRect(0, 0, uint16(w), uint16(h)))
}
