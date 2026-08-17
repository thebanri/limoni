# 🎬 Animasyon ve Fizik Motoru (Animation & Physics)

Limoni, 60 FPS akıcı arayüz geçişleri, yay tabanlı fizik animasyonları ve renk geçişleri için `animation` paketini içerir.

---

## 1. Sayısal ve Renk Enterpolasyonu

### `animation.Float`
Hedef değere yumuşak easing eğrileriyle yaklaşan sayısal animasyon değişkenidir.

```go
anim := animation.NewFloat(0.0)
anim.SetTarget(100.0, 500*time.Millisecond, animation.EaseOutCubic)

// Her render karesinde:
anim.Update(deltaTime)
currentVal := anim.Value()
```

### `animation.Color`
İki RGB renk arasında algısal olarak pürüzsüz geçiş (Linear RGB / HSV Lerp) sağlar.

```go
colAnim := animation.NewColor(cell.NewColorRGB(30, 144, 255))
colAnim.SetTarget(cell.NewColorRGB(255, 69, 0), 300*time.Millisecond, animation.EaseInOutQuad)
```

---

## 2. Easing Eğrileri

Limoni aşağıdaki standart easing eğrilerini içerir:
- `animation.Linear`
- `animation.EaseInQuad` / `animation.EaseOutQuad` / `animation.EaseInOutQuad`
- `animation.EaseInCubic` / `animation.EaseOutCubic` / `animation.EaseInOutCubic`
- `animation.EaseOutBounce` (Sekme efekti)
- `animation.EaseOutElastic` (Yay efekti)
