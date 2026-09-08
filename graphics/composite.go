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
	flattenedImageCache = make(map[flattenedImageKey]image.Image)
	flattenedCacheMu    sync.RWMutex
	opacityImageCache   = make(map[opacityImageKey]image.Image)
	opacityCacheMu      sync.RWMutex
)

type flattenedImageKey struct {
	pointer       uintptr
	r, g, b       uint8
	width, height int
}

// FlattenImage composites transparent pixels over an opaque background.
func FlattenImage(src image.Image, background color.Color) image.Image {
	if src == nil {
		return nil
	}
	br, bg, bb, _ := background.RGBA()
	bounds := src.Bounds()
	// image.Uniform gibi görüntüler pratikte sınırsız bounds döndürebilir.
	// Bu durumda bounds boyutuyla tampon ayırmak taşma/panik üretir; bu tür
	// kaynaklar için düzleştirme yerine kaynağı korumak güvenlidir.
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 || width > 1<<20 || height > 1<<20 {
		return src
	}
	key := flattenedImageKey{}
	cacheable := false
	value := reflect.ValueOf(src)
	if value.Kind() == reflect.Pointer {
		key = flattenedImageKey{pointer: value.Pointer(), r: uint8(br >> 8), g: uint8(bg >> 8), b: uint8(bb >> 8), width: width, height: height}
		cacheable = true
		flattenedCacheMu.RLock()
		if cached, ok := flattenedImageCache[key]; ok {
			flattenedCacheMu.RUnlock()
			return cached
		}
		flattenedCacheMu.RUnlock()
	}
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: background}, image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), src, bounds.Min, draw.Over)
	if cacheable {
		flattenedCacheMu.Lock()
		if len(flattenedImageCache) > 256 {
			clear(flattenedImageCache)
		}
		flattenedImageCache[key] = dst
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
		if cached, ok := opacityImageCache[key]; ok {
			opacityCacheMu.RUnlock()
			return cached
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
		opacityImageCache[key] = dst
		opacityCacheMu.Unlock()
	}
	return dst
}
