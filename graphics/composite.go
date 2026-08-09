package graphics

import (
	"image"
	"image/color"
	"image/draw"
	"reflect"
	"sync"
)

var flattenedImageCache sync.Map

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
		if cached, ok := flattenedImageCache.Load(key); ok {
			return cached.(image.Image)
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: background}, image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), src, bounds.Min, draw.Over)
	if cacheable {
		flattenedImageCache.Store(key, dst)
	}
	return dst
}
