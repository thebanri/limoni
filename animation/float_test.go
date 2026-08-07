package animation

import (
	"testing"
	"time"
)

func TestFloatAnimation(t *testing.T) {
	f := NewFloat(10.0)
	if f.Value() != 10.0 {
		t.Errorf("NewFloat(10.0) = %v; 10.0 bekleniyordu", f.Value())
	}
	if f.IsAnimating() {
		t.Error("Yeni oluşturulan float animasyon durumunda olmamalıdır")
	}

	// 100ms süren doğrusal animasyon başlat
	f.AnimateTo(20.0, 100*time.Millisecond, Linear)
	if !f.IsAnimating() {
		t.Error("AnimateTo çağrısından sonra animasyon başlamış olmalıydı")
	}

	// Test için başlangıç zamanını kontrol edilebilir kıl
	now := time.Now()
	f.startTime = now

	// 50ms sonra (yarı yolda) güncelle
	f.Update(now.Add(50 * time.Millisecond))
	if f.Value() != 15.0 {
		t.Errorf("50ms sonra değer %v; 15.0 bekleniyordu", f.Value())
	}
	if !f.IsAnimating() {
		t.Error("Yarı yolda animasyon hâlâ devam etmeliydi")
	}

	// 100ms sonra (tam zamanında veya sonrasında) güncelle
	stillAnimating := f.Update(now.Add(100 * time.Millisecond))
	if stillAnimating {
		t.Error("Süre bittiğinde Update false dönmeliydi")
	}
	if f.Value() != 20.0 {
		t.Errorf("Süre sonunda değer %v; 20.0 bekleniyordu", f.Value())
	}
	if f.IsAnimating() {
		t.Error("Süre sonunda animasyon durumu durdurulmuş olmalıydı")
	}
}

func TestFloatZeroDuration(t *testing.T) {
	f := NewFloat(10.0)
	f.AnimateTo(30.0, 0, Linear)
	if f.Value() != 30.0 {
		t.Errorf("Sıfır süreli animasyon sonrası değer %v; 30.0 bekleniyordu", f.Value())
	}
	if f.IsAnimating() {
		t.Error("Sıfır süreli animasyon aktif olmamalıdır")
	}
}
