package terminal_test

import (
	"image"
	"image/color"
	"regexp"
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

// A Sixel picture is cells on the terminal's side. It went out before the
// diff, whose ECH for the cells turning into RuneImage then erased it; and
// text over it was covered whenever the picture was sent again. Now the
// cells go first, the picture after them, and the text over it after that.
func TestSixelPictureGoesOutAfterTheCells(t *testing.T) {
	t.Setenv("LIMONI_PROBE", "0")
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
	caps.GraphicsProto = graphics.ProtocolSixel
	term.SetCapabilities(caps)

	area := cell.NewRect(2, 1, 8, 3)
	draw := func(pic *image.RGBA) string {
		before := len(io.Output())
		if err := term.Draw(func(f *terminal.Frame) {
			for y := area.Y; y < area.Y+area.Height; y++ {
				for x := area.X; x < area.X+area.Width; x++ {
					f.Buffer.SetCell(x, y, cell.Cell{Content: cell.RuneImage})
				}
			}
			f.Buffer.SetString(3, 2, "hi", cell.Style{})
			f.RegisterImageRegion(terminal.ImageRegion{Area: area, Img: pic})
		}); err != nil {
			t.Fatal(err)
		}
		return string(io.Output()[before:])
	}
	pic := image.NewRGBA(image.Rect(0, 0, 80, 60))
	for i := range pic.Pix {
		pic.Pix[i] = 0x80
	}
	for _, name := range []string{"first frame", "a new picture"} {
		if name != "first frame" {
			pic = image.NewRGBA(pic.Rect) // another value: the terminal is sent it again
		}
		out := draw(pic)
		start := strings.Index(out, "\x1bP")
		if start < 0 {
			t.Fatalf("%s: no picture sent: %q", name, out)
		}
		end := start + strings.Index(out[start:], "\x1b\\")
		if ech := regexp.MustCompile(`\x1b\[\d+X`).FindAllStringIndex(out, -1); len(ech) > 0 && ech[len(ech)-1][0] > start {
			t.Errorf("%s: cells were erased after the picture: %q", name, out)
		}
		if !strings.Contains(out[end:], "hi") {
			t.Errorf("%s: the text over the picture was not sent after it: %q", name, out[end:])
		}
	}
	if out := draw(pic); strings.Contains(out, "\x1bP") {
		t.Errorf("an unchanged picture was sent again: %q", out)
	}
}

// A dialog over a Sixel picture sends no solid picture of its own: the picture is sent, then
// the dialog's cells over it. Windows Terminal drew the solid one over the dialog's text.
func TestSixelDialogSendsNoBackdropPicture(t *testing.T) {
	t.Setenv("LIMONI_PROBE", "0")
	io := driver.NewMemoryTerminalIO(nil, 40, 12)
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
	caps.GraphicsProto = graphics.ProtocolSixel
	term.SetCapabilities(caps)

	pic := image.NewRGBA(image.Rect(0, 0, 80, 60))
	for i := range pic.Pix {
		pic.Pix[i] = 0x80
	}
	for _, withPicture := range []bool{false, true} {
		before := len(io.Output())
		if err := term.Draw(func(f *terminal.Frame) {
			if withPicture {
				f.RenderWidget(&widgets.Image{Img: pic}, cell.NewRect(1, 1, 20, 8))
			}
			f.RegisterLayer("dialog", terminal.LayerModal, cell.NewRect(10, 3, 20, 6), 1000, nil)
			f.BeginLayer("dialog")
			f.RenderWidget(widgets.Block{Borders: widgets.BorderAll, Opaque: true, Style: cell.Style{Bg: cell.NewColorRGB(20, 20, 30)}}, cell.NewRect(10, 3, 20, 6))
			f.Buffer.SetString(12, 5, "dialog", cell.Style{})
			f.EndLayer()
		}); err != nil {
			t.Fatal(err)
		}
		out := string(io.Output()[before:])
		want := 0
		if withPicture {
			want = 1
		}
		if n := strings.Count(out, "\x1bP"); n != want {
			t.Fatalf("picture %v: %d Sixel pictures sent, want %d", withPicture, n, want)
		}
		if withPicture && strings.LastIndex(out, "dialog") < strings.Index(out, "\x1bP") {
			t.Fatal("the dialog's text was not sent again after the picture")
		}
	}
}
