package graphics

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"strconv"
	"strings"
	"testing"
)

// kittyPicture decodes the picture an EncodeKitty sequence carries.
func kittyPicture(t *testing.T, seq string) image.Image {
	t.Helper()
	var b64 strings.Builder
	for _, chunk := range strings.Split(seq, "\x1b_G")[1:] {
		_, data, _ := strings.Cut(strings.TrimSuffix(chunk, "\x1b\\"), ";")
		b64.WriteString(data)
	}
	raw, err := base64.StdEncoding.DecodeString(b64.String())
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	return img
}

// A picture smaller than its cells is sent at its own resolution, in the
// cells' shape, and kitty scales it: scaling it up first made a turning
// record's every frame a larger PNG and a slower one. A larger picture is
// still brought down to the cells.
func TestKittyIsSentSmallPicturesAtTheirOwnSize(t *testing.T) {
	for _, c := range []struct {
		name                     string
		w, h                     int
		cols, rows, cellW, cellH uint16
		wantW, wantH             int
	}{
		{"small, same shape", 20, 20, 4, 2, 10, 20, 20, 20},
		{"small, taller cells", 20, 20, 3, 2, 9, 19, 20, 29},
		{"small, wider cells", 20, 20, 4, 1, 10, 20, 40, 20},
		{"larger than the cells", 200, 200, 4, 2, 10, 20, 40, 40},
		{"exactly the cells", 40, 40, 4, 2, 10, 20, 40, 40},
	} {
		src := image.NewRGBA(image.Rect(0, 0, c.w, c.h))
		for i := range src.Pix {
			src.Pix[i] = 255
		}
		w, h := KittyCanvas(src, c.cols, c.rows, c.cellW, c.cellH)
		if w != c.wantW || h != c.wantH {
			t.Errorf("%s: canvas %dx%d, want %dx%d", c.name, w, h, c.wantW, c.wantH)
			continue
		}
		seq := EncodeKitty(src, c.cols, c.rows, c.cellW, c.cellH, 1, 0, false)
		if !strings.Contains(seq, ",s="+strconv.Itoa(w)+",v="+strconv.Itoa(h)+",") {
			t.Errorf("%s: header does not say %dx%d: %.90q", c.name, w, h, seq)
		}
		if b := kittyPicture(t, seq).Bounds(); b.Dx() != w || b.Dy() != h {
			t.Errorf("%s: sent %v, header says %dx%d", c.name, b, w, h)
		}
		if b := kittyPicture(t, EncodeKittyTransmit(src, c.cols, c.rows, c.cellW, c.cellH, 1, false)).Bounds(); b.Dx() != w || b.Dy() != h {
			t.Errorf("%s: transmitted %v, want %dx%d", c.name, b, w, h)
		}
	}
}

// A record turning in kitty: a 240-pixel picture in 24×12 cells of 10×20.
func BenchmarkEncodeKittyRecord(b *testing.B) {
	img := image.NewRGBA(image.Rect(0, 0, 240, 240))
	for y := 0; y < 240; y++ {
		for x := 0; x < 240; x++ {
			i := img.PixOffset(x, y)
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = uint8(x), uint8(y), uint8(x^y), 255
		}
	}
	b.ReportAllocs()
	for b.Loop() {
		_ = EncodeKitty(img, 24, 12, 10, 20, 1, 0, false)
	}
}
