package animation

import (
	"time"

	"github.com/thebanri/limoni/core/cell"
)

// Color, zaman tabanlı olarak bir cell.Color değerini (TrueColor/RGB) hedeflenen renge doğru
// ivmelenme eğrisi kullanarak pürüzsüzce geçiren (fade/blend) animasyon yöneticisidir.
type Color struct {
	startCol  cell.Color
	endCol    cell.Color
	current   cell.Color
	startTime time.Time
	duration  time.Duration
	easing    EasingFunc
	animating bool
}

// NewColor, başlangıç rengiyle yeni bir Color animasyon nesnesi oluşturur.
func NewColor(initial cell.Color) *Color {
	return &Color{
		startCol:  initial,
		endCol:    initial,
		current:   initial,
		animating: false,
	}
}

// AnimateTo, belirtilen hedef renge doğru yeni bir renk geçişi başlatır.
// Eğer duration sıfır veya sıfırdan küçükse, hedef renge anında geçiş yapılır.
func (c *Color) AnimateTo(target cell.Color, duration time.Duration, easing EasingFunc) {
	if easing == nil {
		easing = Linear
	}
	c.startCol = c.current
	c.endCol = target
	c.duration = duration
	c.easing = easing
	c.startTime = time.Now()

	if duration <= 0 {
		c.current = target
		c.animating = false
	} else {
		c.animating = true
	}
}

// Update, animasyonun durumunu verilen zamana göre günceller.
// Animasyon devam ediyorsa true, bitmişse veya hiç başlamamışsa false döner.
func (c *Color) Update(now time.Time) bool {
	if !c.animating {
		return false
	}

	elapsed := now.Sub(c.startTime)
	if elapsed >= c.duration {
		c.current = c.endCol
		c.animating = false
		return false
	}

	// Normalize edilmiş zaman (0.0 - 1.0)
	t := float64(elapsed) / float64(c.duration)
	// İvmelenme katsayısı
	progress := c.easing(t)

	// Eğer her iki renk de RGB ise TrueColor kanal interpolasyonu uygula
	if c.startCol.Type() == cell.ColorRGB && c.endCol.Type() == cell.ColorRGB {
		sr, sg, sb := c.startCol.RGB()
		er, eg, eb := c.endCol.RGB()

		r := uint8(float64(sr) + (float64(er)-float64(sr))*progress)
		g := uint8(float64(sg) + (float64(eg)-float64(sg))*progress)
		b := uint8(float64(sb) + (float64(eb)-float64(sb))*progress)

		c.current = cell.NewColorRGB(r, g, b)
	} else {
		// RGB dışındaki renk türleri (Default veya ANSI) için orta noktada doğrudan geçiş yap (step function)
		if progress >= 0.5 {
			c.current = c.endCol
		} else {
			c.current = c.startCol
		}
	}

	return true
}

// Value, güncel rengi döndürür.
func (c *Color) Value() cell.Color {
	return c.current
}

// SetColor, animasyonu sonlandırıp rengi doğrudan belirtilen renge eşitler.
func (c *Color) SetColor(col cell.Color) {
	c.startCol = col
	c.endCol = col
	c.current = col
	c.animating = false
}

// Stop, animasyonu olduğu yerde durdurur. Renk güncel geçiş durumunda kalır.
func (c *Color) Stop() {
	c.animating = false
}

// IsAnimating, animasyonun çalışıp çalışmadığını belirtir.
func (c *Color) IsAnimating() bool {
	return c.animating
}
