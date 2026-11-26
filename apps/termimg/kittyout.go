//go:build unix

package main

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"image"
	"image/draw"
	"strconv"
)

// kittyOut shows the sixel and iTerm2 pictures termimg tracks with the real
// terminal's own kitty graphics protocol, for terminals that have it but
// neither of the others (kwi, Ghostty): no overlay window, so it works on
// Wayland too. The program's own kitty commands then pass through untouched.
//
// The terminal moves a placement with its text as it scrolls, keeps it in
// its scrollback and drops it with the screen it is on, so a picture is only
// sent again when the shadow screen says something the terminal cannot know:
// text written over its last rows (sixel's rule: the text wins), the
// picture written over in full, or a new animation frame.
type kittyOut struct {
	next   uint32
	placed map[*picture]*kittyPlaced
}

type kittyPlaced struct {
	id   uint32
	img  image.Image // the frame sent
	full int         // its rows when sent whole
	rows int         // the rows placed
}

// kittyOutBase starts termimg's image ids, far above the small ones
// programs choose for themselves.
const kittyOutBase = 0x7e000000

// kittyQuery asks whether the terminal has kitty's graphics protocol: a 1×1
// picture it checks but never shows. The answer is kittyAnswer's.
const kittyQuery = "\x1b_Gi=31,s=1,v=1,a=q,t=d,f=24;AAAA\x1b\\"

// kittyAnswer takes the answer to kittyQuery out of b, and reports whether
// it was OK.
func kittyAnswer(b []byte) (bool, []byte) {
	const head = "\x1b_Gi=31;"
	i := bytes.Index(b, []byte(head))
	if i < 0 {
		return false, b
	}
	j := bytes.Index(b[i:], []byte("\x1b\\"))
	if j < 0 {
		return false, b
	}
	ok := string(b[i+len(head):i+j]) == "OK"
	return ok, append(b[:i:i], b[i+j+2:]...)
}

// showKitty brings the terminal's placements in line with the pictures on
// the shadow screen, and puts the cursor back where the shadow screen has it.
func (s *session) showKitty() {
	k := s.kout
	alt := s.emu.IsAltScreen()
	at := s.emu.CursorPosition()
	moved := false
	for _, pic := range s.pics.byID {
		if pic.alt != alt {
			continue // the terminal keeps it with the other screen
		}
		p := k.placed[pic]
		switch {
		case p == nil:
			k.next++
			p = &kittyPlaced{id: kittyOutBase + k.next, full: pic.rows}
			k.placed[pic] = p
		case p.img == pic.img && p.rows == pic.rows:
			continue // where the terminal has it already
		}
		if pic.top < 0 {
			// shortcut: cut or changed while partly scrolled away, it stays as
			// it was; a placement cannot start above the screen.
			continue
		}
		if p.img != pic.img {
			s.buf = kittyTransmit(s.buf, p.id, pic.img)
			p.img = pic.img
		}
		p.rows = pic.rows
		s.buf = cup(s.buf, pic.col, pic.top)
		s.buf = kittyPlace(s.buf, p.id, pic.cols, pic.rows, pic.img.Bounds().Dy()*pic.rows/p.full)
		moved = true
	}
	for pic, p := range k.placed {
		if s.pics.idOf(pic) != 0 {
			continue
		}
		delete(k.placed, pic)
		if !pic.inScrollback(s.pics.gone) {
			// Written over or cleared away: in a sixel terminal the text took
			// its cells.
			s.buf = append(s.buf, "\x1b_Ga=d,d=I,q=2,i="...)
			s.buf = strconv.AppendUint(s.buf, uint64(p.id), 10)
			s.buf = append(s.buf, "\x1b\\"...)
		}
	}
	if moved {
		s.buf = cup(s.buf, at.X, at.Y)
	}
}

// inScrollback says whether pic scrolled off the top of the screen, where
// the terminal keeps it by itself.
func (pic *picture) inScrollback(gone []*picture) bool {
	for _, q := range gone {
		if q == pic {
			return true
		}
	}
	return false
}

// kittyTransmit sends img as RGBA, compressed, in chunks, under id; sent
// again under the same id it replaces the picture.
func kittyTransmit(b []byte, id uint32, img image.Image) []byte {
	r := img.Bounds()
	n, ok := img.(*image.NRGBA)
	if !ok || n.Stride != 4*r.Dx() {
		n = image.NewNRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
		draw.Draw(n, n.Bounds(), img, r.Min, draw.Src)
	}
	var z bytes.Buffer
	w := zlib.NewWriter(&z)
	_, _ = w.Write(n.Pix[:4*r.Dx()*r.Dy()])
	_ = w.Close()
	data := base64.StdEncoding.EncodeToString(z.Bytes())

	b = append(b, "\x1b_Ga=t,f=32,o=z,q=2,i="...)
	b = strconv.AppendUint(b, uint64(id), 10)
	b = append(b, ",s="...)
	b = strconv.AppendInt(b, int64(r.Dx()), 10)
	b = append(b, ",v="...)
	b = strconv.AppendInt(b, int64(r.Dy()), 10)
	const chunk = 4096
	for first := true; first || len(data) > 0; first = false {
		part := data[:min(chunk, len(data))]
		data = data[len(part):]
		if !first {
			b = append(b, "\x1b_Gq=2"...)
		}
		if len(data) > 0 {
			b = append(b, ",m=1;"...)
		} else {
			b = append(b, ",m=0;"...)
		}
		b = append(b, part...)
		b = append(b, "\x1b\\"...)
	}
	return b
}

// kittyPlace shows picture id at the cursor over cols×rows cells, its top h
// pixel rows only, without moving the cursor. Placed again it moves.
func kittyPlace(b []byte, id uint32, cols, rows, h int) []byte {
	b = append(b, "\x1b_Ga=p,p=1,q=2,C=1,i="...)
	b = strconv.AppendUint(b, uint64(id), 10)
	b = append(b, ",c="...)
	b = strconv.AppendInt(b, int64(cols), 10)
	b = append(b, ",r="...)
	b = strconv.AppendInt(b, int64(rows), 10)
	b = append(b, ",h="...)
	b = strconv.AppendInt(b, int64(max(h, 1)), 10)
	return append(b, "\x1b\\"...)
}
