package widgets

import (
	"image"
	"image/draw"
	"strings"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/graphics"
)

// markdownImage is a picture standing on a line of its own.
type markdownImage struct {
	alt, src string
}

// markdownPlacement is where a picture went in the laid-out rows: rows
// rows from row, cols columns from col.
type markdownPlacement struct {
	src       string
	img       image.Image
	row, rows int
	col, cols int
}

// markdownPicture is what is kept of one source image while the content
// stays: the widget that draws it, and the slices of it shown while it is
// partly scrolled out of view.
//
// The slices are kept, not made again, for two reasons. A new image value
// on every scroll step is a new encode and a new upload for kitty, Sixel and
// iTerm2, whose escape sequences are cached by image value. And the caches
// key on the value: keeping each slice alive is what stops a later one from
// being allocated at the same address and answered with the old pixels.
type markdownPicture struct {
	img    image.Image
	view   Image
	slices map[[2]int]image.Image
}

// defaultMaxImageRows caps a picture's height when MaxImageRows is zero.
const defaultMaxImageRows = 20

// picture is the image Images has for src, or nil.
func (m *Markdown) picture(src string) image.Image {
	if m.Images == nil {
		return nil
	}
	return m.Images(src)
}

// picturesChanged reports whether Images answers differently, for a picture
// in the text, from when the rows were laid out: one has arrived, or been
// replaced, and the layout must follow. It runs on every draw, so it reads
// only what the last layout kept.
func (m *Markdown) picturesChanged() bool {
	if m.Images == nil {
		return false
	}
	shown := 0
	for _, line := range m.cachedLines {
		if line.image == nil {
			continue
		}
		img := m.Images(line.image.src)
		var laid image.Image
		if shown < len(m.placements) && m.placements[shown].src == line.image.src {
			laid = m.placements[shown].img
			shown++
		}
		if img != laid {
			// A tracking pixel is laid out as nothing; it has not changed.
			if laid == nil && img != nil {
				if _, _, show := m.pictureSize(img, 1<<15); !show {
					continue
				}
			}
			return true
		}
	}
	return false
}

// pictureSize is how many columns and rows a picture takes at most width
// columns. Cells are taken to be twice as tall as wide, and a picture is
// given a column for every ten pixels, the size a terminal cell usually is,
// so that a small picture is not blown up. show is false for an image
// nobody can see: one or two pixels across, the tracking pixel of a feed.
func (m *Markdown) pictureSize(img image.Image, width int) (cols, rows int, show bool) {
	b := img.Bounds()
	pw, ph := b.Dx(), b.Dy()
	if pw <= 2 || ph <= 2 || width <= 0 {
		return 0, 0, false
	}
	maxRows := m.MaxImageRows
	if maxRows <= 0 {
		maxRows = defaultMaxImageRows
	}
	cols = min(width, max(1, (pw+5)/10))
	rows = max(1, (cols*ph+pw)/(2*pw))
	if rows > maxRows {
		rows = maxRows
		cols = min(width, max(1, (2*rows*pw+ph/2)/ph))
	}
	return cols, rows, true
}

// slice is the part of the picture shown when only rows v0 to v1 of its
// rows rows are in view.
func (p *markdownPicture) slice(v0, v1, rows int) image.Image {
	if v0 == 0 && v1 == rows {
		return p.img
	}
	key := [2]int{v0, v1}
	if s, ok := p.slices[key]; ok {
		return s
	}
	b := p.img.Bounds()
	y0 := b.Min.Y + v0*b.Dy()/rows
	y1 := max(b.Min.Y+v1*b.Dy()/rows, y0+1)
	r := image.Rect(b.Min.X, y0, b.Max.X, y1)
	var s image.Image
	if sub, ok := p.img.(interface {
		SubImage(image.Rectangle) image.Image
	}); ok {
		s = sub.SubImage(r)
	} else {
		dst := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
		draw.Draw(dst, dst.Bounds(), p.img, r.Min, draw.Src)
		s = dst
	}
	if p.slices == nil {
		p.slices = make(map[[2]int]image.Image)
	}
	p.slices[key] = s
	return s
}

// forgetPictures drops what was kept for the pictures of the last content,
// and what the image caches hold for them: the terminal will not be shown
// them again from here.
func (m *Markdown) forgetPictures() {
	for src, p := range m.pictures {
		for _, s := range p.slices {
			graphics.ForgetImage(s)
		}
		graphics.ForgetImage(p.img)
		delete(m.pictures, src)
	}
	m.placements = m.placements[:0]
}

// drawPictures draws the pictures in view, the visible part of each.
func (m *Markdown) drawPictures(ctx cell.Context, buf *buffer.Buffer, offset int, baseStyle cell.Style) {
	height := int(ctx.Area.Height)
	for i := range m.placements {
		pl := &m.placements[i]
		top := pl.row - offset
		v0, v1 := max(0, -top), min(pl.rows, height-top)
		if v1 <= v0 {
			continue
		}
		p := m.pictures[pl.src]
		if p == nil || p.img != pl.img {
			if p != nil {
				for _, s := range p.slices {
					graphics.ForgetImage(s)
				}
			}
			if m.pictures == nil {
				m.pictures = make(map[string]*markdownPicture)
			}
			p = &markdownPicture{img: pl.img}
			m.pictures[pl.src] = p
		}
		p.view.Img = p.slice(v0, v1, pl.rows)
		child := ctx
		child.Style = baseStyle
		child.Area = cell.NewRect(ctx.Area.X+uint16(pl.col), ctx.Area.Y+uint16(top+v0), uint16(pl.cols), uint16(v1-v0))
		p.view.Draw(child, buf)
	}
}

// MarkdownImageSources lists the addresses of the images a Markdown
// document refers to, in order and without repeats, for an application to
// fetch and hand back through Markdown.Images.
func MarkdownImageSources(content string) []string {
	var sources []string
	seen := map[string]bool{}
	for rest := content; ; {
		i := strings.Index(rest, "![")
		if i < 0 {
			return sources
		}
		runes := []rune(rest[i+1:])
		_, src, next, ok := parseMarkdownLink(runes, 0)
		if !ok {
			rest = rest[i+2:]
			continue
		}
		if !seen[src] {
			seen[src] = true
			sources = append(sources, src)
		}
		rest = string(runes[next:])
	}
}
