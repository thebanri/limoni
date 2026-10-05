package terminal

import (
	"image"
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/graphics"
	"github.com/thebanri/limoni/widgets"
)

func kittyTerminal(t *testing.T) (*Terminal, *driver.MemoryTerminalIO) {
	t.Helper()
	t.Setenv("LIMONI_PROBE", "0")
	io := driver.NewMemoryTerminalIO(nil, 40, 6)
	term, err := New(driver.NewPortableBackend(io))
	if err != nil {
		t.Fatal(err)
	}
	caps := term.Capabilities()
	caps.GraphicsProto = graphics.ProtocolKitty
	term.SetCapabilities(caps)
	return term, io
}

func picture(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+2], img.Pix[i+3] = 200, 40, 255
	}
	return img
}

// A picture scrolling through a document is sent to kitty once. Each step
// after that moves its placement — a source rectangle while it is half out
// of view — and sends no pixels. Every picture on screen used to be sent
// again whenever any of them moved.
func TestKittyPicturesAreSentOnce(t *testing.T) {
	term, io := kittyTerminal(t)
	img := picture(100, 80) // 10 columns, 4 rows
	offset := 0
	md := &widgets.Markdown{
		Content:      "top\n![](p.png)\nbottom\n1\n2\n3",
		Images:       func(string) image.Image { return img },
		ScrollOffset: &offset,
	}
	frame := func() string {
		before := len(io.Output())
		if err := term.Draw(func(f *Frame) { f.RenderWidget(md, cell.NewRect(0, 0, 40, 6)) }); err != nil {
			t.Fatal(err)
		}
		return string(io.Output()[before:])
	}

	first := frame()
	if n := strings.Count(first, "a=t,"); n != 1 {
		t.Fatalf("first frame sent %d pictures, want 1", n)
	}
	if !strings.Contains(first, "\x1b_Ga=p,") {
		t.Fatal("first frame placed nothing")
	}

	offset = 1 // "top" leaves the view; the picture is still whole
	if out := frame(); strings.Contains(out, "a=t,") || strings.Contains(out, ",y=") {
		t.Fatalf("moving a whole picture sent it again or clipped it: %q", out)
	}
	for step := 2; step <= 3; step++ { // 3 is as far as it scrolls
		offset = step // the picture's top rows leave the view
		out := frame()
		if strings.Contains(out, "a=t,") {
			t.Fatalf("scroll step %d sent the picture again (%d bytes)", step, len(out))
		}
		if !strings.Contains(out, ",y=") {
			t.Fatalf("scroll step %d placed no part of the picture: %q", step, out)
		}
	}
	offset = 0
	if out := frame(); strings.Contains(out, "a=t,") || strings.Contains(out, ",y=") {
		t.Fatalf("scrolled back, the picture was sent again or still clipped: %q", out)
	}

	md.Content = "no pictures now"
	if out := frame(); !strings.Contains(out, "d=I") && !strings.Contains(out, "d=A") {
		t.Fatalf("a picture no longer shown was not freed: %q", out)
	}
}

// After a resume the terminal may be a different one: pictures are sent
// again rather than placed from what the old one had.
func TestKittyPicturesAreSentAgainAfterAFullRedraw(t *testing.T) {
	term, io := kittyTerminal(t)
	img := picture(60, 40)
	md := &widgets.Markdown{Content: "![](p.png)", Images: func(string) image.Image { return img }}
	draw := func() string {
		before := len(io.Output())
		_ = term.Draw(func(f *Frame) { f.RenderWidget(md, cell.NewRect(0, 0, 40, 6)) })
		return string(io.Output()[before:])
	}
	draw()
	term.ForceFullRedraw()
	if out := draw(); strings.Count(out, "a=t,") != 1 {
		t.Fatalf("after a full redraw the picture was sent %d times, want 1", strings.Count(out, "a=t,"))
	}
}

// A part of a picture smaller than its cells is cut from the picture as
// kitty was sent it, which KittyCanvas left at its own size: the source
// rectangle is in those pixels, not in the cells'.
func TestKittyClipsASmallPictureInItsOwnPixels(t *testing.T) {
	full := picture(50, 40)                          // laid out in 10×4 cells of 10×20: 100×80
	part := full.SubImage(image.Rect(0, 20, 50, 40)) // its lower half
	clip := &graphics.Clip{Image: part, Full: full, Cols: 10, Rows: 4, Top: 2, Bottom: 4}
	var k kittyImages
	out := string(k.place(nil, []ImageRegion{{Area: cell.NewRect(0, 0, 10, 2), Img: clip}}, 10, 20))
	if !strings.Contains(out, ",s=50,v=40") {
		t.Fatalf("the picture was not sent at its own size: %.120q", out)
	}
	if !strings.Contains(out, ",x=0,y=20,w=50,h=20") {
		t.Fatalf("the lower half is not cut in the picture's pixels: %q", out[strings.Index(out, "a=p"):])
	}
}
