package component

import (
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

type spacerComponent struct {
	weight uint16
}

// Spacer returns an expanding, invisible flexible component.
// In an HStack, it expands horizontally pushing adjacent items apart.
// In a VStack, it expands vertically.
// If weight is omitted, it defaults to 1.
//
// Example:
//
//	limoni.HStack(
//	    limoni.Text("Left"),
//	    limoni.Spacer(), // Pushes "Right" all the way to the right
//	    limoni.Text("Right"),
//	)
func Spacer(weight ...uint16) Component {
	w := uint16(1)
	if len(weight) > 0 && weight[0] > 0 {
		w = weight[0]
	}
	return &spacerComponent{weight: w}
}

func (s *spacerComponent) Draw(ctx cell.Context, buf *buffer.Buffer) {
	// Invisible spacer draws nothing
}

func (s *spacerComponent) LayoutInfo(maxArea cell.Rect) LayoutProps {
	return LayoutProps{
		MinWidth:  0,
		MinHeight: 0,
		Flex:      s.weight,
	}
}

func (s *spacerComponent) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	return 0, 0
}

func (s *spacerComponent) HandleEvent(ctx cell.Context, ev *driver.Event) bool {
	return false
}
