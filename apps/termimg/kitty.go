//go:build unix

package main

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif" // decoders for f=100 and iTerm2
	_ "image/jpeg"
	_ "image/png"
	"io"
	"maps"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/thebanri/limoni/graphics"
)

// maxData caps the bytes of one picture, compressed or not, from any source.
const maxData = 256 << 20

// kitty is what is kept of kitty's graphics protocol between commands:
// pictures sent with an id, the pictures shown on screen of each, and a
// transmission arriving in chunks. The protocol is spoken in full for the
// programs that draw with it (kitten icat, chafa, yazi, mpv --vo=kitty,
// image.nvim): sending a picture directly, from a file, a temporary file or
// shared memory; PNG, RGB or RGBA, compressed or not, in chunks; image
// numbers; showing it at the cursor, stretched over given cells, a part of
// it, offset within the cell, relative to another placement or through
// Unicode placeholders (placeholder.go); animation; deleting by id, number,
// position or z-index; and the query programs probe the terminal with.
//
// shortcut: pictures under the text (z<0) are shown over it: a window
// cannot be under the terminal's text.
type kitty struct {
	images  map[uint32]*anim
	numbers map[uint32]uint32 // an image number (I) → the id it was given
	shown   []kittyShown
	virtual []*kittyVirtual
	first   map[string]string // the keys of a transmission in chunks
	chunks  []byte            // its payload so far
}

type kittyShown struct {
	image, placement uint32
	z                int
	pic              *picture
	parent           *picture // a relative placement's (P, Q)
}

// apc takes kitty graphics commands, ESC _ G keys ; payload ESC \.
func (s *session) apc(body []byte) bool {
	if len(body) == 0 || body[0] != 'G' || s.kout != nil {
		return false // with kout, the terminal itself does kitty's
	}
	keys, payload := kittyKeys(body[1:])
	k := &s.kitty
	if k.first != nil { // a later chunk carries only m (and q)
		if len(k.chunks)+len(payload) > maxData {
			k.first, k.chunks = nil, nil
			return true
		}
		k.chunks = append(k.chunks, payload...)
		if keys["m"] == "1" {
			return true
		}
		keys, payload = k.first, k.chunks
		k.first, k.chunks = nil, nil
	} else if keys["m"] == "1" && (keys["t"] == "" || keys["t"] == "d") {
		// Only a direct transmission comes in chunks; with a file or shared
		// memory kitty ignores m, and mpv sets it on every frame.
		k.first, k.chunks = keys, append([]byte(nil), payload...)
		return true
	}

	action := keys["a"]
	transmit := action == "" || action == "t" || action == "T"
	id := kittyNum(keys, "i")
	if num := kittyNum(keys, "I"); id == 0 && num != 0 {
		// A number instead of an id: a new picture gets an id of the
		// terminal's choosing, said in the reply; anything else means the
		// newest picture with that number.
		if transmit {
			id = k.freeID()
			if k.numbers == nil {
				k.numbers = map[uint32]uint32{}
			}
			k.numbers[num] = id
		} else {
			id = k.numbers[num]
		}
		keys["i"] = strconv.FormatUint(uint64(id), 10)
	}
	switch a := k.images[id]; {
	case action == "q":
		_, err := s.kittyDecode(keys, payload)
		s.kittyReply(keys, err)
	case transmit:
		img, err := s.kittyDecode(keys, payload)
		if err == nil {
			a = still(img)
			if id != 0 {
				if k.images == nil {
					k.images = map[uint32]*anim{}
				}
				k.images[id] = a
			}
			if action == "T" {
				err = s.kittyPut(keys, a)
			}
		}
		s.kittyReply(keys, err)
	case action == "d":
		s.kittyDelete(keys)
	case a == nil:
		s.kittyReply(keys, errors.New("ENOENT:no such image"))
	case action == "p":
		s.kittyReply(keys, s.kittyPut(keys, a))
	case action == "f":
		img, err := s.kittyDecode(keys, payload)
		if err == nil {
			err = kittyFrame(a, keys, img)
		}
		s.kittyReply(keys, err)
	case action == "a":
		kittyControl(a, keys)
	case action == "c":
		s.kittyReply(keys, kittyCompose(a, keys))
	}
	return true
}

// freeID is an id for a picture sent with a number only, from the top of
// the range, where programs that pick ids themselves rarely go.
func (k *kitty) freeID() uint32 {
	id := uint32(math.MaxUint32)
	for k.images[id] != nil {
		id--
	}
	return id
}

// kittyKeys splits "k=v,k=v;payload".
func kittyKeys(b []byte) (map[string]string, []byte) {
	ctrl, payload, _ := bytes.Cut(b, []byte(";"))
	keys := map[string]string{}
	for _, kv := range strings.Split(string(ctrl), ",") {
		if k, v, ok := strings.Cut(kv, "="); ok {
			keys[k] = v
		}
	}
	return keys, payload
}

func kittyNum(keys map[string]string, k string) uint32 {
	n, _ := strconv.ParseUint(keys[k], 10, 32)
	return uint32(n)
}

// kittyInt reads the keys that may be negative: H, V, z.
func kittyInt(keys map[string]string, k string) int {
	n, _ := strconv.Atoi(keys[k])
	return n
}

// kittyReply answers a command that named an image, as kitty does: OK, or
// the error, unless q asks for quiet (1: no OK, 2: nothing).
func (s *session) kittyReply(keys map[string]string, err error) {
	id := kittyNum(keys, "i")
	if id == 0 || keys["q"] == "2" || err == nil && keys["q"] == "1" {
		return
	}
	msg := "OK"
	if err != nil {
		msg = err.Error()
		if !strings.Contains(msg, ":") {
			msg = "EINVAL:" + msg
		}
	}
	r := "\x1b_Gi=" + strconv.FormatUint(uint64(id), 10)
	if n := kittyNum(keys, "I"); n != 0 {
		r += ",I=" + strconv.FormatUint(uint64(n), 10)
	}
	if p := kittyNum(keys, "p"); p != 0 {
		r += ",p=" + strconv.FormatUint(uint64(p), 10)
	}
	s.reply(r + ";" + msg + "\x1b\\")
}

// kittyDecode makes a picture of a transmission: payload is base64, of the
// pixels or of where to read them.
func (s *session) kittyDecode(keys map[string]string, payload []byte) (image.Image, error) {
	data, err := base64.StdEncoding.DecodeString(string(payload))
	if err != nil {
		data, err = base64.RawStdEncoding.DecodeString(strings.TrimRight(string(payload), "="))
		if err != nil {
			return nil, errors.New("EINVAL:bad base64")
		}
	}
	if m := keys["t"]; m != "" && m != "d" {
		if data, err = kittyRead(m, string(data), kittyNum(keys, "O"), kittyNum(keys, "S")); err != nil {
			return nil, err
		}
	}
	if keys["o"] == "z" {
		zr, err := zlib.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, errors.New("EINVAL:bad zlib data")
		}
		if data, err = io.ReadAll(io.LimitReader(zr, maxData)); err != nil {
			return nil, errors.New("EINVAL:bad zlib data")
		}
	}
	switch f := keys["f"]; f {
	case "100":
		img, _, err := decodeFile(data)
		if err != nil {
			return nil, errors.New("EINVAL:bad PNG")
		}
		return img, nil
	case "", "32", "24":
		bpp := 4
		if f == "24" {
			bpp = 3
		}
		w, h := int(kittyNum(keys, "s")), int(kittyNum(keys, "v"))
		if w <= 0 || h <= 0 || w > maxSide || h > maxSide || len(data) < w*h*bpp {
			return nil, errors.New("EINVAL:size does not match the data")
		}
		img := image.NewNRGBA(image.Rect(0, 0, w, h))
		for i := range w * h {
			copy(img.Pix[4*i:4*i+3], data[bpp*i:bpp*i+3])
			img.Pix[4*i+3] = 0xff
			if bpp == 4 {
				img.Pix[4*i+3] = data[bpp*i+3]
			}
		}
		return img, nil
	}
	return nil, errors.New("EINVAL:unknown format")
}

// kittyRead reads a picture sent by name: a file (t=f), a temporary file
// to delete after (t=t), or POSIX shared memory (t=s), from offset off,
// size bytes of it if size is not 0. The rules are kitty's, since the
// output may come from another machine: only regular files are read, and a
// temporary file is deleted only in a temporary directory and with
// tty-graphics-protocol in its name.
func kittyRead(medium, name string, off, size uint32) ([]byte, error) {
	path := name
	switch medium {
	case "s":
		// A POSIX name is "/name": one slash, at the start.
		name = strings.TrimPrefix(name, "/")
		if name == "" || strings.ContainsRune(name, '/') {
			return nil, errors.New("EINVAL:bad shared memory name")
		}
		path = "/dev/shm/" + name
	case "f", "t":
		if !filepath.IsAbs(path) {
			return nil, errors.New("EINVAL:path is not absolute")
		}
	default:
		return nil, errors.New("EINVAL:unknown transmission medium")
	}
	fi, err := os.Lstat(path)
	if err != nil || !fi.Mode().IsRegular() {
		return nil, errors.New("EBADF:not a regular file")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.New("EBADF:cannot open")
	}
	defer f.Close()
	if _, err := f.Seek(int64(off), io.SeekStart); err != nil {
		return nil, errors.New("EINVAL:bad offset")
	}
	n := int64(maxData)
	if size != 0 {
		n = min(n, int64(size))
	}
	data, err := io.ReadAll(io.LimitReader(f, n))
	if err != nil {
		return nil, errors.New("EBADF:cannot read")
	}
	dir := filepath.Dir(path)
	switch {
	case medium == "s":
		os.Remove(path) // shm_unlink, as kitty does
	case medium == "t" && strings.Contains(filepath.Base(path), "tty-graphics-protocol") &&
		(dir == filepath.Clean(os.TempDir()) || dir == "/tmp" || dir == "/dev/shm"):
		os.Remove(path)
	}
	return data, nil
}

// kittyPut shows a's current frame at the cursor: the part x, y, w, h of it
// if given, over c×r cells if given (stretched, as kitty does when both
// are), else at its own size; X, Y pixels into the cell. With P (and Q) it
// goes H, V cells from that placement instead, the cursor left alone; with
// U=1 it waits for placeholders. A placement id shown again replaces the old
// one.
func (s *session) kittyPut(keys map[string]string, a *anim) error {
	src := a.frames[a.cur]
	b := src.Bounds()
	part := b
	x, y := int(kittyNum(keys, "x")), int(kittyNum(keys, "y"))
	w, h := int(kittyNum(keys, "w")), int(kittyNum(keys, "h"))
	if x > 0 || y > 0 || w > 0 || h > 0 {
		if w == 0 {
			w = b.Dx() - x
		}
		if h == 0 {
			h = b.Dy() - y
		}
		if part = image.Rect(x, y, x+w, y+h).Add(b.Min).Intersect(b); part.Empty() {
			return nil
		}
	}
	cols, rows := int(kittyNum(keys, "c")), int(kittyNum(keys, "r"))
	pw, ph := part.Dx(), part.Dy()
	switch {
	case cols > 0 && rows > 0:
		pw, ph = cols*s.cellW, rows*s.cellH
	case cols > 0:
		pw, ph = cols*s.cellW, part.Dy()*cols*s.cellW/part.Dx()
	case rows > 0:
		pw, ph = part.Dx()*rows*s.cellH/part.Dy(), rows*s.cellH
	}
	pw, ph = max(pw, 1), max(ph, 1)
	if cols == 0 {
		cols = (pw + s.cellW - 1) / s.cellW
	}
	if rows == 0 {
		rows = (ph + s.cellH - 1) / s.cellH
	}
	id, placement := kittyNum(keys, "i"), kittyNum(keys, "p")
	if keys["U"] == "1" {
		s.kitty.virtual = slices.DeleteFunc(s.kitty.virtual, func(v *kittyVirtual) bool { return v.image == id && v.placement == placement })
		s.kitty.virtual = append(s.kitty.virtual, &kittyVirtual{image: id, placement: placement, anim: a, part: part, cols: cols, rows: rows})
		s.dirty = true
		return nil
	}
	render := func(f image.Image) image.Image {
		if part != f.Bounds() {
			f = subImage(f, part)
		}
		if pw != part.Dx() || ph != part.Dy() {
			f = graphics.ResizeImage(f, pw, ph)
		}
		return f
	}

	var parent *picture
	if p := kittyNum(keys, "P"); p != 0 {
		s.pics.update() // where the parent is now
		q := kittyNum(keys, "Q")
		for _, k := range s.kitty.shown {
			if k.image == p && k.placement == q && s.pics.idOf(k.pic) != 0 {
				parent = k.pic
			}
		}
		if parent == nil {
			return errors.New("ENOPARENT:no such parent placement")
		}
	}
	// The same placement shown again replaces the old one; either way the
	// pictures already gone are forgotten, or a program that never deletes
	// (mpv shows every frame anew) would keep them all.
	s.kittyRemove(func(k kittyShown) bool { return placement != 0 && k.image == id && k.placement == placement })
	var pic *picture
	switch {
	case parent != nil:
		if col := parent.col + kittyInt(keys, "H"); col >= 0 && col < s.cols {
			pic = s.place(render(src), col, parent.top+kittyInt(keys, "V"), cols, rows, true)
		}
	case keys["C"] == "1":
		pic = s.put(render(src), cols, rows, moveNone, true)
	default:
		pic = s.put(render(src), cols, rows, moveAfter, true)
	}
	if pic != nil {
		pic.anim, pic.src, pic.render = a, src, render
		pic.ox, pic.oy = int(kittyNum(keys, "X")), int(kittyNum(keys, "Y"))
		s.kitty.shown = append(s.kitty.shown, kittyShown{id, placement, kittyInt(keys, "z"), pic, parent})
	}
	return nil
}

// kittyFrame adds a frame to an animation (a=f), or changes frame r: the
// data goes at x, y over a copy of frame c, or over the colour Y (RGBA,
// transparent by default), blended unless X=1. z is its gap.
func kittyFrame(a *anim, keys map[string]string, data image.Image) error {
	root := a.frames[0].Bounds()
	edit, base := int(kittyNum(keys, "r")), int(kittyNum(keys, "c"))
	var canvas *image.RGBA
	switch {
	case edit > len(a.frames) || base > len(a.frames):
		return errors.New("ENOENT:no such frame")
	case edit > 0:
		canvas = cloneRGBA(a.frames[edit-1])
	case (len(a.frames)+1)*4*root.Dx()*root.Dy() > maxData:
		return errors.New("ENOSPC:too many frames")
	case base > 0:
		canvas = cloneRGBA(a.frames[base-1])
	default:
		canvas = image.NewRGBA(root)
		if c := kittyNum(keys, "Y"); c != 0 {
			bg := color.NRGBA{uint8(c >> 24), uint8(c >> 16), uint8(c >> 8), uint8(c)}
			draw.Draw(canvas, root, image.NewUniform(bg), image.Point{}, draw.Src)
		}
	}
	op := draw.Over
	if keys["X"] == "1" {
		op = draw.Src
	}
	db := data.Bounds()
	at := image.Pt(int(kittyNum(keys, "x")), int(kittyNum(keys, "y"))).Add(root.Min)
	draw.Draw(canvas, db.Sub(db.Min).Add(at), data, db.Min, op)
	gap, set := kittyGap(keys)
	if edit > 0 {
		a.frames[edit-1] = canvas
		if set {
			a.gaps[edit-1] = gap
		}
		return nil
	}
	if !set {
		gap = defaultGap
	}
	a.frames, a.gaps = append(a.frames, canvas), append(a.gaps, gap)
	return nil
}

// kittyControl handles a=a: s stops (1) or runs (2, 3) the animation, v sets
// its loops (1 forever, n: n-1 more), c the frame shown, z the gap of frame r.
func kittyControl(a *anim, keys map[string]string) {
	if r := int(kittyNum(keys, "r")); r > 0 && r <= len(a.gaps) {
		if gap, set := kittyGap(keys); set {
			a.gaps[r-1] = gap
		}
	}
	if c := int(kittyNum(keys, "c")); c > 0 && c <= len(a.frames) {
		a.cur, a.due = c-1, time.Now().Add(a.gaps[c-1])
	}
	switch v := int(kittyNum(keys, "v")); {
	case v == 1:
		a.loops = -1
	case v > 1:
		a.loops = v - 1
	}
	switch keys["s"] {
	case "1":
		a.run = false
	case "2", "3":
		a.start(time.Now())
	}
}

// kittyCompose handles a=c: the w×h part of frame r at X, Y drawn onto
// frame c at x, y, blended unless C=1.
func kittyCompose(a *anim, keys map[string]string) error {
	from, to := int(kittyNum(keys, "r")), int(kittyNum(keys, "c"))
	if from < 1 || to < 1 || from > len(a.frames) || to > len(a.frames) {
		return errors.New("ENOENT:no such frame")
	}
	src := a.frames[from-1]
	b := src.Bounds()
	w, h := int(kittyNum(keys, "w")), int(kittyNum(keys, "h"))
	if w == 0 {
		w = b.Dx()
	}
	if h == 0 {
		h = b.Dy()
	}
	op := draw.Over
	if keys["C"] == "1" {
		op = draw.Src
	}
	x, y := int(kittyNum(keys, "x")), int(kittyNum(keys, "y"))
	canvas := cloneRGBA(a.frames[to-1])
	sp := image.Pt(int(kittyNum(keys, "X")), int(kittyNum(keys, "Y"))).Add(b.Min)
	draw.Draw(canvas, image.Rect(x, y, x+w, y+h).Add(b.Min), src, sp, op)
	a.frames[to-1] = canvas
	return nil
}

// kittyGap reads z, a frame's gap in milliseconds: 0 changes nothing, a
// negative one skips the frame.
func kittyGap(keys map[string]string) (time.Duration, bool) {
	switch z := kittyInt(keys, "z"); {
	case z == 0:
		return 0, false
	case z < 0:
		return -1, true
	default:
		return time.Duration(z) * time.Millisecond, true
	}
}

// cloneRGBA copies a frame to draw on: a frame shown is never changed in
// place, so a changed one is a new value the overlay sends again.
func cloneRGBA(img image.Image) *image.RGBA {
	b := img.Bounds()
	c := image.NewRGBA(b)
	draw.Draw(c, b, img, b.Min, draw.Src)
	return c
}

// kittyDelete handles a=d: everything (d=a, the default); one image's
// pictures (d=i, a placement of it with p; d=n by number); those over the
// cursor (c), a cell (p, q also at z-index z), a column (x) or a row (y);
// those at z-index z (z); those of ids x to y (r); or an animation's frames
// (f). Upper case frees the images left with no picture.
func (s *session) kittyDelete(keys map[string]string) {
	d := keys["d"]
	id, placement := kittyNum(keys, "i"), kittyNum(keys, "p")
	x, y, z := kittyInt(keys, "x"), kittyInt(keys, "y"), kittyInt(keys, "z")
	cur := s.emu.CursorPosition()
	over := func(p *picture, col, row int) bool { // col, row < 0: any
		return (col < 0 || col >= p.col && col < p.col+p.cols) && (row < 0 || row >= p.top && row < p.top+p.rows)
	}
	// byImage is for the deletes that hit virtual placements too.
	var byImage func(img, placement uint32) bool
	var byPlace func(kittyShown) bool
	switch strings.ToLower(d) {
	case "i", "n":
		byImage = func(img, p uint32) bool { return img == id && (placement == 0 || p == placement) }
	case "r":
		byImage = func(img, _ uint32) bool { return img >= uint32(x) && img <= uint32(y) }
	case "c":
		byPlace = func(k kittyShown) bool { return over(k.pic, cur.X, cur.Y) }
	case "p":
		byPlace = func(k kittyShown) bool { return over(k.pic, x-1, y-1) }
	case "q":
		byPlace = func(k kittyShown) bool { return over(k.pic, x-1, y-1) && k.z == z }
	case "x":
		byPlace = func(k kittyShown) bool { return over(k.pic, x-1, -1) }
	case "y":
		byPlace = func(k kittyShown) bool { return over(k.pic, -1, y-1) }
	case "z":
		byPlace = func(k kittyShown) bool { return k.z == z }
	case "f":
		if a := s.kitty.images[id]; a != nil {
			a.frames, a.gaps, a.cur, a.run = a.frames[:1], a.gaps[:1], 0, false
		}
		return
	default:
		byImage = func(uint32, uint32) bool { return true }
	}
	if byPlace == nil {
		byPlace = func(k kittyShown) bool { return byImage(k.image, k.placement) }
		s.kitty.virtual = slices.DeleteFunc(s.kitty.virtual, func(v *kittyVirtual) bool { return byImage(v.image, v.placement) })
		s.dirty = true
	}
	s.pics.update() // where the pictures are now
	gone := s.kittyRemove(byPlace)
	switch d {
	case "A":
		clear(s.kitty.images)
	case "I", "N":
		delete(s.kitty.images, id)
	case "R", "C", "P", "Q", "X", "Y", "Z":
		for _, img := range gone {
			if !slices.ContainsFunc(s.kitty.shown, func(k kittyShown) bool { return k.image == img }) &&
				!slices.ContainsFunc(s.kitty.virtual, func(v *kittyVirtual) bool { return v.image == img }) {
				delete(s.kitty.images, img)
			}
		}
	default:
		return
	}
	maps.DeleteFunc(s.kitty.numbers, func(_, img uint32) bool { return s.kitty.images[img] == nil })
}

// kittyRemove takes away the pictures shown that match, with the pictures
// placed relative to them, forgets those already gone, and returns the
// images of those it took.
func (s *session) kittyRemove(match func(kittyShown) bool) (gone []uint32) {
	for again := true; again; {
		again = false
		kept := s.kitty.shown[:0]
		for _, k := range s.kitty.shown {
			id := s.pics.idOf(k.pic)
			switch {
			case id == 0:
			case match(k) || k.parent != nil && s.pics.idOf(k.parent) == 0:
				s.pics.remove(id)
				gone = append(gone, k.image)
				again = true // its children go on the next pass
			default:
				kept = append(kept, k)
			}
		}
		clear(s.kitty.shown[len(kept):])
		s.kitty.shown = kept
	}
	return gone
}

func subImage(img image.Image, r image.Rectangle) image.Image {
	if si, ok := img.(interface {
		SubImage(image.Rectangle) image.Image
	}); ok {
		return si.SubImage(r)
	}
	return img
}
