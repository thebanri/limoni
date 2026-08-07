package animation

import (
	"time"
)

// Float, zaman tabanlı olarak bir sayısal değeri (float64) hedeflenen değere doğru
// ivmelenme (easing) eğrisi kullanarak pürüzsüzce yakınsatan animasyon yöneticisidir.
type Float struct {
	startVal  float64
	endVal    float64
	current   float64
	startTime time.Time
	duration  time.Duration
	easing    EasingFunc
	animating bool
}

// NewFloat, belirtilen başlangıç değeriyle yeni bir Float animasyon nesnesi oluşturur.
func NewFloat(initial float64) *Float {
	return &Float{
		startVal:  initial,
		endVal:    initial,
		current:   initial,
		animating: false,
	}
}

// AnimateTo, belirtilen hedef değere doğru yeni bir animasyon başlatır.
// Eğer duration sıfır veya sıfırdan küçükse, hedef değere anında geçiş yapılır.
func (f *Float) AnimateTo(target float64, duration time.Duration, easing EasingFunc) {
	if easing == nil {
		easing = Linear
	}
	f.startVal = f.current
	f.endVal = target
	f.duration = duration
	f.easing = easing
	f.startTime = time.Now()

	if duration <= 0 {
		f.current = target
		f.animating = false
	} else {
		f.animating = true
	}
}

// Update, animasyonun durumunu verilen zamana göre günceller.
// Animasyon hâlâ devam ediyorsa true, bitmişse veya hiç başlamamışsa false döner.
func (f *Float) Update(now time.Time) bool {
	if !f.animating {
		return false
	}

	elapsed := now.Sub(f.startTime)
	if elapsed >= f.duration {
		f.current = f.endVal
		f.animating = false
		return false
	}

	// Normalize edilmiş zaman (0.0 - 1.0)
	t := float64(elapsed) / float64(f.duration)
	// İvmelenme katsayısı
	progress := f.easing(t)
	// Doğrusal interpolasyon (Lerp)
	f.current = f.startVal + (f.endVal-f.startVal)*progress

	return true
}

// Value, güncel değeri döndürür.
func (f *Float) Value() float64 {
	return f.current
}

// SetValue, animasyonu sonlandırıp değeri doğrudan belirtilen sayıya eşitler.
func (f *Float) SetValue(val float64) {
	f.startVal = val
	f.endVal = val
	f.current = val
	f.animating = false
}

// Stop, animasyonu olduğu yerde durdurur. Değer güncel konumunda kalır.
func (f *Float) Stop() {
	f.animating = false
}

// IsAnimating, animasyonun çalışıp çalışmadığını belirtir.
func (f *Float) IsAnimating() bool {
	return f.animating
}
