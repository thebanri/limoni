package animation

import (
	"math"
)

// EasingFunc, normalize edilmiş zaman/ilerleme (0.0 - 1.0) parametresi alan
// ve normalize edilmiş çıktı değeri (0.0 - 1.0) dönen ivmelenme fonksiyonudur.
type EasingFunc func(t float64) float64

// Linear, doğrusal (sabit hızlı) geçiş sağlar.
func Linear(t float64) float64 {
	return t
}

// EaseInQuad, yavaş başlar, giderek hızlanır (karesel).
func EaseInQuad(t float64) float64 {
	return t * t
}

// EaseOutQuad, hızlı başlar, sona doğru yavaşlar (karesel).
func EaseOutQuad(t float64) float64 {
	return t * (2 - t)
}

// EaseInOutQuad, yavaş başlar, ortada hızlanır, sonda tekrar yavaşlar (karesel).
func EaseInOutQuad(t float64) float64 {
	if t < 0.5 {
		return 2 * t * t
	}
	return -1 + (4-2*t)*t
}

// EaseInCubic, yavaş başlar, giderek hızlanır (kübik).
func EaseInCubic(t float64) float64 {
	return t * t * t
}

// EaseOutCubic, hızlı başlar, sona doğru yavaşlar (kübik).
func EaseOutCubic(t float64) float64 {
	t2 := t - 1
	return t2*t2*t2 + 1
}

// EaseInOutCubic, yavaş başlar, ortada hızlanır, sonda yavaşlar (kübik).
func EaseInOutCubic(t float64) float64 {
	if t < 0.5 {
		return 4 * t * t * t
	}
	t2 := 2*t - 2
	return 0.5*t2*t2*t2 + 1
}

// EaseInSine, sinüs dalgası eğrisiyle yavaş başlayıp hızlanır.
func EaseInSine(t float64) float64 {
	return 1 - math.Cos((t*math.Pi)/2)
}

// EaseOutSine, sinüs dalgası eğrisiyle hızlı başlayıp yavaşlar.
func EaseOutSine(t float64) float64 {
	return math.Sin((t * math.Pi) / 2)
}

// EaseInOutSine, sinüs dalgası eğrisiyle başlayıp biten pürüzsüz geçiş.
func EaseInOutSine(t float64) float64 {
	return -(math.Cos(math.Pi*t) - 1) / 2
}

// EaseInExpo, üstel (exponential) olarak yavaş başlayıp çok hızlı biter.
func EaseInExpo(t float64) float64 {
	if t == 0 {
		return 0
	}
	return math.Pow(2, 10*(t-1))
}

// EaseOutExpo, üstel olarak çok hızlı başlayıp sona doğru yavaşlar.
func EaseOutExpo(t float64) float64 {
	if t == 1 {
		return 1
	}
	return 1 - math.Pow(2, -10*t)
}

// EaseInOutExpo, yavaş başlayıp ortada çok hızlı geçiş yapan ve sonda yavaşlayan üstel eğri.
func EaseInOutExpo(t float64) float64 {
	if t == 0 {
		return 0
	}
	if t == 1 {
		return 1
	}
	if t < 0.5 {
		return math.Pow(2, 20*t-10) / 2
	}
	return (2 - math.Pow(2, -20*t+10)) / 2
}

// EaseOutBounce, çarpma ve geri zıplama efekti sunar.
func EaseOutBounce(t float64) float64 {
	const n1 = 7.5625
	const d1 = 2.75

	if t < 1/d1 {
		return n1 * t * t
	} else if t < 2/d1 {
		t -= 1.5 / d1
		return n1*t*t + 0.75
	} else if t < 2.5/d1 {
		t -= 2.25 / d1
		return n1*t*t + 0.9375
	} else {
		t -= 2.625 / d1
		return n1*t*t + 0.984375
	}
}

// EaseInBounce, tersten zıplama efekti sunar.
func EaseInBounce(t float64) float64 {
	return 1 - EaseOutBounce(1-t)
}

// EaseInOutBounce, başta ve sonda zıplama efekti sunar.
func EaseInOutBounce(t float64) float64 {
	if t < 0.5 {
		return (1 - EaseOutBounce(1-2*t)) / 2
	}
	return (1 + EaseOutBounce(2*t-1)) / 2
}
