package widgets

import (
	"image"
	"image/color"
	"reflect"
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/graphics"
)

// solid is a w×h picture of one colour.
func solid(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
	}
	return img
}

// photo is a w×h picture as image/jpeg decodes one: opaque, so it is
// drawn as it is, not flattened into a copy.
func photo(w, h int) *image.YCbCr {
	return image.NewYCbCr(image.Rect(0, 0, w, h), image.YCbCrSubsampleRatio444)
}

type registered struct {
	area cell.Rect
	img  image.Image
}

// imageContext is a context for a terminal with kitty graphics, recording
// what is registered with it.
func imageContext(area cell.Rect, got *[]registered) cell.Context {
	ctx := cell.NewContext(area, cell.Style{})
	ctx.ImageProtocol = uint8(graphics.ProtocolKitty)
	ctx.RegisterImage = func(a cell.Rect, img image.Image, _ int, _ bool) bool {
		*got = append(*got, registered{a, img})
		return true
	}
	return ctx
}

func rowsOf(buf *buffer.Buffer) []string {
	rows := strings.Split(buf.Snapshot(), "\n")
	for i := range rows {
		rows[i] = strings.TrimRight(rows[i], " ")
	}
	return rows
}

// A picture on a line of its own is drawn as one, below the text before it
// and above the text after, once the application has it. Until then its alt
// text stands in for it.
func TestMarkdownDrawsAPictureOnceItArrives(t *testing.T) {
	pics := map[string]image.Image{}
	md := &Markdown{
		Content: "before\n![a red square](red.png)\nafter",
		Images:  func(src string) image.Image { return pics[src] },
	}
	area := cell.NewRect(0, 0, 40, 20)
	var got []registered
	buf := buffer.NewBuffer(area)
	md.Draw(imageContext(area, &got), buf)
	if rows := rowsOf(buf); rows[1] != "▣ a red square" || rows[2] != "after" {
		t.Fatalf("before the picture arrived: %q", rows[:3])
	}
	if len(got) != 0 {
		t.Fatalf("registered %d images with nothing to show", len(got))
	}

	pics["red.png"] = photo(100, 50)
	buf = buffer.NewBuffer(area)
	md.Draw(imageContext(area, &got), buf)
	// 100×50 pixels: ten columns, and cells twice as tall as wide make it
	// two and a half rows, rounded to three.
	if len(got) != 1 || got[0].area != cell.NewRect(0, 1, 10, 3) {
		t.Fatalf("registered %+v, want the picture at (0,1) 10×3", got)
	}
	if got[0].img != pics["red.png"] {
		t.Error("a picture wholly in view was not drawn whole")
	}
	if rows := rowsOf(buf); rows[0] != "before" || rows[4] != "after" {
		t.Fatalf("the text around the picture: %q", rows[:5])
	}
	if !strings.Contains(md.AccessibilityNode(area, false).Value, "[image: a red square]") {
		t.Errorf("the semantic tree does not say there is an image: %q", md.AccessibilityNode(area, false).Value)
	}
}

// Scrolled half out of view, a picture shows the part still in view — not
// the whole picture squeezed into fewer rows — and the same part is the same
// image value every time, so the terminal is not sent it again.
func TestMarkdownShowsThePartOfAPictureInView(t *testing.T) {
	img := photo(100, 80) // 10 columns, 4 rows
	offset := 2
	md := &Markdown{
		Content:      "top\n![](blue.png)\nbottom",
		Images:       func(string) image.Image { return img },
		ScrollOffset: &offset,
	}
	area := cell.NewRect(0, 0, 20, 4)
	var got []registered
	md.Draw(imageContext(area, &got), buffer.NewBuffer(area))
	// Row 1 to 4 is the picture; from offset 2, rows 1 and 2 of it are gone.
	if len(got) != 1 || got[0].area != cell.NewRect(0, 0, 10, 3) {
		t.Fatalf("registered %+v, want the bottom three rows at the top", got)
	}
	if b := got[0].img.Bounds(); b != image.Rect(0, 20, 100, 80) {
		t.Fatalf("the part shown is %v, want the bottom three quarters", b)
	}
	first := got[0].img
	got = got[:0]
	md.Draw(imageContext(area, &got), buffer.NewBuffer(area))
	if got[0].img != first {
		t.Error("the same part of the picture was a new image on the next frame")
	}
}

// A one-pixel image is a feed's tracking pixel: nothing to see, so nothing is
// drawn for it — not even its alt text — once it is known to be one.
func TestMarkdownSkipsTrackingPixels(t *testing.T) {
	md := &Markdown{
		Content: "text\n![](https://feeds.example/pixel.gif)\nmore",
		Images:  func(string) image.Image { return solid(1, 1, color.RGBA{}) },
	}
	area := cell.NewRect(0, 0, 20, 4)
	var got []registered
	buf := buffer.NewBuffer(area)
	md.Draw(imageContext(area, &got), buf)
	if rows := rowsOf(buf); rows[0] != "text" || rows[1] != "more" || len(got) != 0 {
		t.Fatalf("rows %q, registered %d", rows, len(got))
	}
}

// Without an image protocol the picture is drawn in half blocks into the
// cells it was given.
func TestMarkdownPicturesInHalfBlocks(t *testing.T) {
	md := &Markdown{
		Content: "![](red.png)",
		Images:  func(string) image.Image { return solid(40, 40, color.RGBA{255, 0, 0, 255}) },
	}
	area := cell.NewRect(0, 0, 20, 4)
	ctx := cell.NewContext(area, cell.Style{})
	ctx.ImageProtocol = uint8(graphics.ProtocolHalfBlock)
	buf := buffer.NewBuffer(area)
	md.Draw(ctx, buf)
	c := buf.CellAt(1, 0)
	if c.Content != '▄' {
		t.Fatalf("cell (1,0) = %q, want a half block", c.Content)
	}
	if r, g, b := c.Style.Fg.RGB(); r != 255 || g != 0 || b != 0 {
		t.Errorf("the half block is %d,%d,%d, want red", r, g, b)
	}
}

// A linked picture, [![alt](src)](href), and one with a title are pictures
// too; one inside a sentence is its alt text.
func TestMarkdownImageForms(t *testing.T) {
	for _, src := range []string{
		"[![logo](logo.png)](https://example.com)",
		`![logo](logo.png "The logo")`,
	} {
		alt, img, _, ok := blockImage(src)
		if !ok || alt != "logo" || img != "logo.png" {
			t.Errorf("blockImage(%q) = %q, %q, %v", src, alt, img, ok)
		}
	}
	segs := parseInlineStyles("see ![the chart](c.png) here", cell.Style{}, false)
	if got := segText(segs); got != "see ▣ the chart here" {
		t.Errorf("an inline image drew as %q", got)
	}
}

// The application learns what to fetch from the text, each address once.
func TestMarkdownImageSources(t *testing.T) {
	got := MarkdownImageSources("![a](1.png) text ![b](2.png \"t\")\n[![c](1.png)](x) ![not an image")
	if want := []string{"1.png", "2.png"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("sources %q, want %q", got, want)
	}
}

// New content forgets the old pictures: what the image caches hold for them,
// and the parts kept for scrolling. A reader going through a feed would
// otherwise keep every picture it ever showed.
func TestMarkdownForgetsPicturesWithTheirContent(t *testing.T) {
	img := solid(100, 80, color.RGBA{0, 255, 0, 255})
	offset := 1
	md := &Markdown{
		Content:      "![](g.png)",
		Images:       func(string) image.Image { return img },
		ScrollOffset: &offset,
	}
	area := cell.NewRect(0, 0, 20, 3)
	var got []registered
	md.Draw(imageContext(area, &got), buffer.NewBuffer(area))
	if len(md.pictures) != 1 || len(md.pictures["g.png"].slices) != 1 {
		t.Fatalf("pictures kept: %+v", md.pictures)
	}
	md.Content = "no pictures now"
	md.Draw(imageContext(area, &got), buffer.NewBuffer(area))
	if len(md.pictures) != 0 {
		t.Fatalf("the old content's pictures are still kept: %+v", md.pictures)
	}
}

// Drawing a document with a picture in view, frame after frame, allocates
// nothing once the picture is laid out.
func TestMarkdownWithAPictureDrawsWithoutAllocating(t *testing.T) {
	img := solid(100, 80, color.RGBA{0, 0, 255, 255}) // flattened, through the cache
	offset := 1
	md := &Markdown{
		Content:      "top\n![](blue.png)\nbottom",
		Images:       func(string) image.Image { return img },
		ScrollOffset: &offset,
	}
	area := cell.NewRect(0, 0, 20, 4)
	buf := buffer.NewBuffer(area)
	regions := make([]registered, 0, 4)
	ctx := cell.NewContext(area, cell.Style{})
	ctx.ImageProtocol = uint8(graphics.ProtocolKitty)
	ctx.RegisterImage = func(a cell.Rect, img image.Image, _ int, _ bool) bool {
		regions = append(regions[:0], registered{a, img})
		return true
	}
	md.Draw(ctx, buf)
	if n := testing.AllocsPerRun(100, func() { md.Draw(ctx, buf) }); n != 0 {
		t.Fatalf("Draw allocates %.0f times per frame", n)
	}
}

// BenchmarkMarkdownPictureDraw draws a document with a picture half in view
// through kitty graphics, the path a feed reader scrolls through.
func BenchmarkMarkdownPictureDraw(b *testing.B) {
	img := solid(400, 300, color.RGBA{0, 0, 255, 255})
	offset := 3
	md := &Markdown{
		Content:      "# Title\nsome text\n![](p.png)\nmore text after it",
		Images:       func(string) image.Image { return img },
		ScrollOffset: &offset,
	}
	area := cell.NewRect(0, 0, 80, 12)
	buf := buffer.NewBuffer(area)
	var last image.Image
	ctx := cell.NewContext(area, cell.Style{})
	ctx.ImageProtocol = uint8(graphics.ProtocolKitty)
	ctx.RegisterImage = func(_ cell.Rect, img image.Image, _ int, _ bool) bool { last = img; return true }
	md.Draw(ctx, buf)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		md.Draw(ctx, buf)
	}
	_ = last
}
