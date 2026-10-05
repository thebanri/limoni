package terminal

import (
	"image"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/graphics"
)

// kittyImages is what kitty has been sent. Each picture is sent once, under
// an id of its own, and shown with placements: when the pictures on screen
// change, the placements are deleted and made again, and only a picture
// kitty does not have yet is sent. A picture moving, or scrolling half out
// of view (a graphics.Clip), costs a placement of a few dozen bytes instead
// of the whole picture again — every picture on screen was sent again before.
// One no longer shown is freed in kitty.
type kittyImages struct {
	ids   map[kittyKey]*kittyEntry
	next  uint32
	stamp uint64
	// gen is graphics.Generation when the pictures were sent; a ForgetImage
	// since means one may have been rewritten in place.
	gen uint64
	// stale is set when the terminal may have none of them (a resume).
	stale bool
}

// kittyKey is a picture as sent: kitty is sent it scaled to its cells.
type kittyKey struct {
	img          image.Image
	cols, rows   uint16
	cellW, cellH uint16
	transparent  bool
}

type kittyEntry struct {
	id    uint32
	stamp uint64
}

// place appends what shows regions: new pictures, then every placement, then
// the freeing of pictures no longer shown.
func (k *kittyImages) place(dst []byte, regions []ImageRegion, cellW, cellH uint16) []byte {
	if gen := graphics.Generation(); k.stale || gen != k.gen {
		dst = k.freeAll(dst)
		k.gen, k.stale = gen, false
	}
	if k.ids == nil {
		k.ids = make(map[kittyKey]*kittyEntry)
	}
	dst = append(dst, graphics.KittyDeletePlacements...)
	k.stamp++
	for _, reg := range regions {
		full, cols, rows := reg.Img, reg.Area.Width, reg.Area.Height
		x, y, w, h := 0, 0, 0, 0
		if c, ok := reg.Img.(*graphics.Clip); ok && c.Rows > 0 && c.Bottom > c.Top {
			full, cols, rows = c.Full, uint16(c.Cols), uint16(c.Rows)
			// The source rectangle is in the pixels of the picture sent,
			// which KittyCanvas may have made smaller than the cells.
			fw, fh := graphics.KittyCanvas(full, cols, rows, cellW, cellH)
			w, y, h = fw, c.Top*fh/c.Rows, (c.Bottom-c.Top)*fh/c.Rows
		}
		key := kittyKey{full, cols, rows, cellW, cellH, reg.Transparent}
		e := k.ids[key]
		if e == nil {
			k.next++
			e = &kittyEntry{id: 0x4c690000 + k.next}
			seq := graphics.EncodeKittyTransmit(full, cols, rows, cellW, cellH, e.id, reg.Transparent)
			if seq == "" {
				continue
			}
			dst = append(dst, seq...)
			k.ids[key] = e
		}
		e.stamp = k.stamp
		z := reg.ZIndex
		if z == 0 {
			z = -1
		}
		dst = buffer.AppendCursor(dst, reg.Area.X, reg.Area.Y)
		dst = graphics.AppendKittyPlace(dst, e.id, reg.Area.Width, reg.Area.Height, x, y, w, h, z)
	}
	for key, e := range k.ids {
		if e.stamp != k.stamp {
			dst = graphics.AppendKittyFree(dst, e.id)
			delete(k.ids, key)
		}
	}
	return dst
}

// freeAll appends the deletion of every picture and forgets them.
func (k *kittyImages) freeAll(dst []byte) []byte {
	clear(k.ids)
	return append(dst, graphics.KittyFreeAll...)
}
