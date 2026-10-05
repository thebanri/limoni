package main

import (
	"bytes"
	"context"
	"hash/fnv"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	_ "golang.org/x/image/webp"
)

// A cover is kept as artSize² linear-light RGB, cropped square. The
// renderer resamples it to the disc's size whenever that changes.
const artSize = 256

type art struct {
	px    []rgb // artSize*artSize, row by row
	tint  rgb   // the cover's colour, for the panel behind it
	made  bool  // generated, not read from a picture
	label string
}

// loadArt finds a track's cover: a picture in the file, then one beside
// it, then one made from the album's name.
func loadArt(t *track) *art {
	if t.hasPic {
		if img := embeddedPicture(t.path); img != nil {
			return fromImage(img)
		}
	}
	if !t.demo {
		if img := folderPicture(filepath.Dir(t.path)); img != nil {
			return fromImage(img)
		}
	}
	return generatedArt(t.artist + "\x00" + t.album)
}

func embeddedPicture(path string) image.Image {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	b, err := exec.CommandContext(ctx, "ffmpeg", "-v", "error", "-nostdin", "-i", path,
		"-map", "0:v:0", "-frames:v", "1", "-f", "image2pipe", "-c:v", "png", "-").Output()
	if err != nil || len(b) == 0 {
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return nil
	}
	return img
}

// folderPicture prefers the names people give covers, then any picture.
func folderPicture(dir string) image.Image {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	best, bestRank := "", 99
	for _, e := range entries {
		name := strings.ToLower(e.Name())
		ext := filepath.Ext(name)
		if e.IsDir() || (ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" && ext != ".gif") {
			continue
		}
		rank := 10
		for i, want := range []string{"cover", "folder", "front", "album", "art"} {
			if strings.Contains(name, want) {
				rank = i
				break
			}
		}
		if rank < bestRank {
			best, bestRank = e.Name(), rank
		}
	}
	if best == "" {
		return nil
	}
	f, err := os.Open(filepath.Join(dir, best))
	if err != nil {
		return nil
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil
	}
	return img
}

func decodeImage(r io.Reader) image.Image {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil
	}
	return img
}

func srgbToLinear(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

func linearToSRGB(v float64) float64 {
	v = max(0, min(1, v))
	if v <= 0.0031308 {
		return v * 12.92
	}
	return 1.055*math.Pow(v, 1/2.4) - 0.055
}

// fromImage crops the middle square and averages it down to artSize².
func fromImage(img image.Image) *art {
	b := img.Bounds()
	side := min(b.Dx(), b.Dy())
	x0 := b.Min.X + (b.Dx()-side)/2
	y0 := b.Min.Y + (b.Dy()-side)/2
	a := &art{px: make([]rgb, artSize*artSize)}
	cnt := make([]float64, artSize*artSize)
	for y := 0; y < side; y++ {
		ty := y * artSize / side
		for x := 0; x < side; x++ {
			tx := x * artSize / side
			r, g, bl, _ := img.At(x0+x, y0+y).RGBA()
			i := ty*artSize + tx
			a.px[i].r += srgbToLinear(float64(r) / 65535)
			a.px[i].g += srgbToLinear(float64(g) / 65535)
			a.px[i].b += srgbToLinear(float64(bl) / 65535)
			cnt[i]++
		}
	}
	// A cover smaller than artSize leaves gaps; fill each from the source
	// pixel it would have been, nearest-neighbour.
	for i := range a.px {
		if cnt[i] > 0 {
			a.px[i] = a.px[i].scale(1 / cnt[i])
			continue
		}
		x, y := i%artSize, i/artSize
		r, g, bl, _ := img.At(x0+x*side/artSize, y0+y*side/artSize).RGBA()
		a.px[i] = rgb{srgbToLinear(float64(r) / 65535), srgbToLinear(float64(g) / 65535), srgbToLinear(float64(bl) / 65535)}
	}
	a.tint = tintOf(a.px)
	return a
}

// tintOf is the cover's most colourful average: saturated pixels count more
// than grey ones, so a mostly white cover with a red title reads as red.
func tintOf(px []rgb) rgb {
	var sum rgb
	var w float64
	for _, c := range px {
		hi := max(c.r, c.g, c.b)
		lo := min(c.r, c.g, c.b)
		k := 0.02 + (hi-lo)*(hi-lo)*4
		sum = sum.add(c.scale(k))
		w += k
	}
	return sum.scale(1 / w)
}

// generatedArt paints a cover for an album that has none: marbled paint,
// folded over itself, in three colours picked from the name. The same name
// always gets the same picture.
func generatedArt(name string) *art {
	h := fnv.New64a()
	h.Write([]byte(name))
	seed := h.Sum64()
	hue := float64(seed%360) / 360
	dark := hsl(hue, 0.35, 0.07)
	mid := hsl(hue+0.06, 0.28, 0.42)
	light := hsl(hue-0.04, 0.15, 0.92)
	accentC := hsl(hue+0.5, 0.55, 0.55)
	a := &art{px: make([]rgb, artSize*artSize), made: true}
	s := float64(seed>>20&0xffff) / 997
	for y := 0; y < artSize; y++ {
		for x := 0; x < artSize; x++ {
			u := float64(x)/artSize*3 + s
			v := float64(y)/artSize*3 - s
			// Domain warping: the noise is sampled where other noise
			// points, twice, which makes the flowing, folded shapes.
			qx := fbm(u, v, seed)
			qy := fbm(u+5.2, v+1.3, seed)
			rx := fbm(u+4*qx+1.7, v+4*qy+9.2, seed)
			ry := fbm(u+4*qx+8.3, v+4*qy+2.8, seed)
			f := fbm(u+4*rx, v+4*ry, seed)
			f = smoothstep(0.25, 0.75, f)
			var c rgb
			if f < 0.5 {
				c = dark.mix(mid, f*2)
			} else {
				c = mid.mix(light, (f-0.5)*2)
			}
			// A thread of colour where the folds are tightest.
			if e := math.Abs(rx - ry); e < 0.04 {
				c = c.mix(accentC, (0.04-e)/0.04*0.5)
			}
			a.px[y*artSize+x] = rgb{srgbToLinear(c.r), srgbToLinear(c.g), srgbToLinear(c.b)}
		}
	}
	a.tint = tintOf(a.px)
	return a
}

func smoothstep(e0, e1, x float64) float64 {
	t := max(0, min(1, (x-e0)/(e1-e0)))
	return t * t * (3 - 2*t)
}

func hash2(x, y int, seed uint64) float64 {
	h := uint64(x)*0x9E3779B97F4A7C15 ^ uint64(y)*0xC2B2AE3D27D4EB4F ^ seed
	h ^= h >> 31
	h *= 0xBF58476D1CE4E5B9
	h ^= h >> 29
	return float64(h>>11) / (1 << 53)
}

func valueNoise(x, y float64, seed uint64) float64 {
	xi, yi := int(math.Floor(x)), int(math.Floor(y))
	fx, fy := x-math.Floor(x), y-math.Floor(y)
	fx = fx * fx * (3 - 2*fx)
	fy = fy * fy * (3 - 2*fy)
	a := hash2(xi, yi, seed)
	b := hash2(xi+1, yi, seed)
	c := hash2(xi, yi+1, seed)
	d := hash2(xi+1, yi+1, seed)
	return a + (b-a)*fx + (c-a)*fy + (a-b-c+d)*fx*fy
}

func fbm(x, y float64, seed uint64) float64 {
	sum, amp := 0.0, 0.5
	for range 5 {
		sum += amp * valueNoise(x, y, seed)
		// Rotate between octaves so the grid does not show.
		x, y = 1.6*x+1.2*y+17, -1.2*x+1.6*y+31
		amp *= 0.5
	}
	return sum / 0.97
}

// hsl gives an sRGB colour (not linear) from hue, saturation and lightness.
func hsl(h, s, l float64) rgb {
	h -= math.Floor(h)
	f := func(n float64) float64 {
		k := math.Mod(n+h*12, 12)
		a := s * min(l, 1-l)
		return l - a*max(-1, min(k-3, 9-k, 1))
	}
	return rgb{f(0), f(8), f(4)}
}
