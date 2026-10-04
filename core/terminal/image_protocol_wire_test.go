package terminal_test

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/graphics"
	"github.com/thebanri/limoni/widgets"
)

// The protocol the terminal settled on has to reach the widget: an Image
// used to ask the environment itself, so a Sixel terminal found by its DA1
// answer still got half blocks. The environment here says half blocks; the
// terminal says Sixel; the wire must carry a Sixel picture.
func TestImageWidgetUsesTheTerminalsProtocol(t *testing.T) {
	t.Setenv("LIMONI_PROBE", "0")
	t.Setenv("LIMONI_GRAPHICS", "halfblock")
	pic := image.NewRGBA(image.Rect(0, 0, 8, 16))
	for i := range pic.Pix {
		pic.Pix[i] = 0xC0
	}
	pic.Set(0, 0, color.RGBA{R: 255, A: 255})

	draw := func(proto graphics.Protocol) string {
		io := driver.NewMemoryTerminalIO(nil, 20, 6)
		b := driver.NewPortableBackend(io)
		if err := b.Setup(); err != nil {
			t.Fatal(err)
		}
		term, err := terminal.New(b)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = term.Close() }()
		caps := term.Capabilities()
		caps.GraphicsProto = proto
		term.SetCapabilities(caps)
		before := len(io.Output())
		if err := term.Draw(func(f *terminal.Frame) {
			f.RenderWidget(&widgets.Image{Img: pic}, cell.NewRect(0, 0, 4, 4))
		}); err != nil {
			t.Fatal(err)
		}
		return string(io.Output()[before:])
	}

	if got := draw(graphics.ProtocolSixel); !strings.Contains(got, "\x1bP") {
		t.Errorf("a Sixel terminal was sent no DCS picture: %q", got)
	}
	if got := draw(graphics.ProtocolHalfBlock); strings.Contains(got, "\x1bP") {
		t.Errorf("a half-block terminal was sent a Sixel picture: %q", got)
	}
}
