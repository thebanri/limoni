package graphics

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"reflect"
	"sync"
)

var (
	flattenedImageCache = make(map[flattenedImageKey]derivedImage)
	flattenedCacheMu    sync.RWMutex
	opacityImageCache   = make(map[opacityImageKey]derivedImage)
	opacityCacheMu      sync.RWMutex
)

// derivedImage is a cached result and the image it was made from. Keeping
// the source is what makes a pointer a safe key: while the entry exists the
// source cannot be collected, so its address cannot be reused by a different
// picture that would then be answered with this one's pixels.
type derivedImage struct{ src, dst image.Image }

// knownOpaque reports, without reading a pixel, whether img has no
// transparency: the types image/jpeg and grey-scale decoders produce.
func knownOpaque(img image.Image) bool {
	switch img.(type) {
	case *image.YCbCr, *image.Gray, *image.Gray16, *image.CMYK:
		return true
	}
	return false
}

type flattenedImageKey struct {
	pointer       uintptr
	r, g, b       uint8
	width, height int
}

// FlattenImage composites transparent pixels over an opaque background.
func FlattenImage(src image.Image, background color.Color) image.Image {
	r, g, b, _ := background.RGBA()
	return FlattenImageRGB(src, uint8(r>>8), uint8(g>>8), uint8(b>>8))
}

// FlattenImageRGB is FlattenImage over the background r, g, b. It takes no
// color.Color, whose boxing would cost a draw that calls it every frame an
// allocation even when the result is cached.
func FlattenImageRGB(src image.Image, r, g, b uint8) image.Image {
	if src == nil {
		return nil
	}
	// Nothing to composite: a photo is returned as it is, not copied.
	if knownOpaque(src) {
		return src
	}
	bounds := src.Bounds()
	// Images such as image.Uniform report effectively unbounded bounds, and
	// allocating a buffer that size overflows or panics. Returning the source
	// unflattened is the safe answer for them.
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 || width > 1<<20 || height > 1<<20 {
		return src
	}
	key := flattenedImageKey{}
	cacheable := false
	value := reflect.ValueOf(src)
	if value.Kind() == reflect.Pointer {
		key = flattenedImageKey{pointer: value.Pointer(), r: r, g: g, b: b, width: width, height: height}
		cacheable = true
		flattenedCacheMu.RLock()
		if cached, ok := flattenedImageCache[key]; ok && cached.src == src {
			flattenedCacheMu.RUnlock()
			return cached.dst
		}
		flattenedCacheMu.RUnlock()
	}
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.RGBA{R: r, G: g, B: b, A: 255}}, image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), src, bounds.Min, draw.Over)
	if cacheable {
		flattenedCacheMu.Lock()
		if len(flattenedImageCache) > 256 {
			clear(flattenedImageCache)
		}
		flattenedImageCache[key] = derivedImage{src, dst}
		flattenedCacheMu.Unlock()
	}
	return dst
}

type opacityImageKey struct {
	pointer       uintptr
	opacity       uint64
	width, height int
}

// ApplyOpacity multiplies the image's alpha channel by the given opacity factor (0.0 to 1.0).
// Pointer-backed images are cached because this function is called from Image.Draw,
// which runs on every frame. Keeping the transformed image stable also prevents native
// terminal protocols from re-uploading the same avatar on every frame.
func ApplyOpacity(src image.Image, opacity float64) image.Image {
	if src == nil || opacity >= 1.0 {
		return src
	}
	bounds := src.Bounds()
	if opacity <= 0.0 {
		return image.NewRGBA(bounds)
	}

	value := reflect.ValueOf(src)
	cacheable := value.Kind() == reflect.Pointer
	var key opacityImageKey
	if cacheable {
		key = opacityImageKey{
			pointer: value.Pointer(),
			opacity: math.Float64bits(opacity),
			width:   bounds.Dx(),
			height:  bounds.Dy(),
		}
		opacityCacheMu.RLock()
		if cached, ok := opacityImageCache[key]; ok && cached.src == src {
			opacityCacheMu.RUnlock()
			return cached.dst
		}
		opacityCacheMu.RUnlock()
	}

	clamp := func(v float64) uint8 {
		if v <= 0 {
			return 0
		}
		if v >= 255 {
			return 255
		}
		return uint8(v)
	}

	dst := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := src.At(x, y).RGBA()
			if a == 0 {
				continue
			}
			newA := uint16(float64(a) * opacity)
			factor := float64(newA) / float64(a)
			nr := clamp((float64(r) * factor) / 257.0)
			ng := clamp((float64(g) * factor) / 257.0)
			nb := clamp((float64(b) * factor) / 257.0)
			na := clamp(float64(newA) / 257.0)
			dst.Set(x, y, color.RGBA{R: nr, G: ng, B: nb, A: na})
		}
	}
	if cacheable {
		opacityCacheMu.Lock()
		if len(opacityImageCache) > 256 {
			clear(opacityImageCache)
		}
		opacityImageCache[key] = derivedImage{src, dst}
		opacityCacheMu.Unlock()
	}
	return dst
}

// forgetDerived drops what FlattenImage and ApplyOpacity made from img.
func forgetDerived(img image.Image) {
	flattenedCacheMu.Lock()
	for key, entry := range flattenedImageCache {
		if entry.src == img {
			delete(flattenedImageCache, key)
		}
	}
	flattenedCacheMu.Unlock()
	opacityCacheMu.Lock()
	for key, entry := range opacityImageCache {
		if entry.src == img {
			delete(opacityImageCache, key)
		}
	}
	opacityCacheMu.Unlock()
}
