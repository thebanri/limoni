# 🎬 Animation and Physics Engine

Limoni includes the `animation` package for fluid 60 FPS UI transitions, spring-based physics, and smooth color interpolations.

---

## 1. Numeric and Color Interpolation

### `animation.Float`
A continuous numeric animation variable that smoothly approaches a target value using configurable easing functions:

```go
anim := animation.NewFloat(0.0)
anim.SetTarget(100.0, 500*time.Millisecond, animation.EaseOutCubic)

// On each render frame:
anim.Update(deltaTime)
currentVal := anim.Value()
```

### `animation.Color`
Provides perceptually uniform color interpolation between two RGB color points (Linear RGB / HSV Lerp):

```go
import "github.com/thebanri/limoni"

colAnim := animation.NewColor(limoni.RGB(30, 144, 255))
colAnim.SetTarget(limoni.RGB(255, 69, 0), 300*time.Millisecond, animation.EaseInOutQuad)
```

---

## 2. Easing Functions

Limoni includes standard easing curves:
- `animation.Linear`
- `animation.EaseInQuad` / `animation.EaseOutQuad` / `animation.EaseInOutQuad`
- `animation.EaseInCubic` / `animation.EaseOutCubic` / `animation.EaseInOutCubic`
- `animation.EaseOutBounce` (Realistic bounce effect)
- `animation.EaseOutElastic` (Spring / elastic overshoot effect)
