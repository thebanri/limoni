package graphics

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"strconv"
	"sync/atomic"
	"unsafe"
)

// Clip is the part of a picture in view: Full, laid out at Cols×Rows cells,
// of which rows Top to Bottom show — a picture scrolled half off the screen.
//
// As an image.Image it is that part's pixels, so a protocol that is sent
// pixels (half blocks, Sixel, iTerm2) draws it as it draws any picture. Kitty
// is sent Full once and shown the part with a source rectangle, so a picture
// scrolling through a document costs a few bytes a step instead of a new
// picture each time.
type Clip struct {
	image.Image // the part in view
	Full        image.Image
	// Cols and Rows are the cells Full is laid out in; Top and Bottom the
	// rows of it in view, Bottom exclusive.
	Cols, Rows, Top, Bottom int
}

// flattenClip flattens both pictures of a Clip, keeping it a Clip. The new
// Clip is cached as any flattened picture is, so the same part drawn again
// is the same value — to the caches, and to the terminal.
func flattenClip(c *Clip, r, g, b uint8) image.Image {
	bounds := c.Bounds()
	key := flattenedImageKey{pointer: uintptr(unsafe.Pointer(c)), r: r, g: g, b: b, width: bounds.Dx(), height: bounds.Dy()}
	flattenedCacheMu.RLock()
	if cached, ok := flattenedImageCache[key]; ok && cached.src == image.Image(c) {
		flattenedCacheMu.RUnlock()
		return cached.dst
	}
	flattenedCacheMu.RUnlock()
	part, full := FlattenImageRGB(c.Image, r, g, b), FlattenImageRGB(c.Full, r, g, b)
	var dst image.Image = c
	if part != c.Image || full != c.Full {
		dst = &Clip{Image: part, Full: full, Cols: c.Cols, Rows: c.Rows, Top: c.Top, Bottom: c.Bottom}
	}
	flattenedCacheMu.Lock()
	if len(flattenedImageCache) > 256 {
		clear(flattenedImageCache)
	}
	flattenedImageCache[key] = derivedImage{c, dst}
	flattenedCacheMu.Unlock()
	return dst
}

// generation counts ForgetImage calls. A terminal that keeps pictures it
// has sent compares it, and sends them again when it has changed: one of
// them may have been rewritten in place.
var generation atomic.Uint64

// Generation is the number of ForgetImage calls so far.
func Generation() uint64 { return generation.Load() }

// EncodeKittyTransmit sends img to kitty under id, scaled into cols×rows
// cells, without showing it: AppendKittyPlace shows it, as often and as
// partly as needed, without sending it again.
func EncodeKittyTransmit(img image.Image, cols, rows uint16, cellW, cellH uint16, id uint32, transparent bool) string {
	if img == nil || cols == 0 || rows == 0 || cellW == 0 || cellH == 0 {
		return ""
	}
	targetW := int(cols) * int(cellW)
	targetH := int(rows) * int(cellH)
	resized := ResizeImageContain(img, targetW, targetH, transparent)
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, resized); err != nil {
		return ""
	}
	b64Data := base64.StdEncoding.EncodeToString(pngBuf.Bytes())
	return chunkKittyPayload(fmt.Sprintf("q=2,f=100,a=t,t=d,i=%d,s=%d,v=%d", id, targetW, targetH), b64Data)
}

// AppendKittyPlace appends a placement of picture id at the cursor, over
// cols×rows cells. With h > 0 only the source rectangle x, y, w, h (in the
// picture's pixels) is shown. C=1 keeps the cursor where it is: a picture
// that reaches the last row would otherwise scroll the screen.
func AppendKittyPlace(dst []byte, id uint32, cols, rows uint16, x, y, w, h, z int) []byte {
	dst = append(dst, "\x1b_Ga=p,q=2,C=1,i="...)
	dst = strconv.AppendUint(dst, uint64(id), 10)
	dst = append(dst, ",c="...)
	dst = strconv.AppendUint(dst, uint64(cols), 10)
	dst = append(dst, ",r="...)
	dst = strconv.AppendUint(dst, uint64(rows), 10)
	dst = append(dst, ",z="...)
	dst = strconv.AppendInt(dst, int64(z), 10)
	if h > 0 {
		dst = append(dst, ",x="...)
		dst = strconv.AppendInt(dst, int64(x), 10)
		dst = append(dst, ",y="...)
		dst = strconv.AppendInt(dst, int64(y), 10)
		dst = append(dst, ",w="...)
		dst = strconv.AppendInt(dst, int64(w), 10)
		dst = append(dst, ",h="...)
		dst = strconv.AppendInt(dst, int64(h), 10)
	}
	return append(dst, "\x1b\\"...)
}

// AppendKittyFree appends the command that deletes picture id and frees what
// kitty keeps of it.
func AppendKittyFree(dst []byte, id uint32) []byte {
	dst = append(dst, "\x1b_Ga=d,d=I,q=2,i="...)
	dst = strconv.AppendUint(dst, uint64(id), 10)
	return append(dst, "\x1b\\"...)
}

// Kitty commands that delete every placement and keep the pictures, and that
// delete every picture and free them.
const (
	KittyDeletePlacements = "\x1b_Ga=d,d=a,q=2\x1b\\"
	KittyFreeAll          = "\x1b_Ga=d,d=A,q=2\x1b\\"
)
