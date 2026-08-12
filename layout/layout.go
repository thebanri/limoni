package layout

import (
	"sync"

	"github.com/thebanri/limoni/core/cell"
)

type ConstraintKey struct {
	Type  ConstraintType
	Value uint16
}

type layoutKey struct {
	direction   Direction
	gap         uint16
	area        cell.Rect
	constraints [16]ConstraintKey
	fitSizes    [16]uint16
	numC        int
	numF        int
}

var splitCache = struct {
	sync.RWMutex
	m map[layoutKey][]cell.Rect
}{
	m: make(map[layoutKey][]cell.Rect),
}

// Direction, yerleşim elemanlarının dizilim yönünü belirten türdür.
type Direction uint8

const (
	// Horizontal, elemanları soldan sağa yatay olarak dizer.
	Horizontal Direction = iota
	// Vertical, elemanları yukarıdan aşağıya dikey olarak dizer.
	Vertical
)

// ConstraintType, esnek yerleşim kısıtlamasının matematiksel türünü belirtir.
type ConstraintType uint8

const (
	// ConstraintFixed, sabit bir karakter hücresi genişliği/yüksekliği atar.
	ConstraintFixed ConstraintType = iota
	// ConstraintPercentage, üst alanın yüzdesel oranında alan atar (0-100).
	ConstraintPercentage
	// ConstraintRatio, kalan alanı ağırlıklı oranlara göre paylaştırır (örn. 1:2:3).
	ConstraintRatio
	// ConstraintMin, en az belirtilen boyutta alan atar.
	ConstraintMin
	// ConstraintMax, en fazla belirtilen boyutta alan atar.
	ConstraintMax
	// ConstraintFill, geriye kalan tüm boşluğu kaplayan kısıtlama.
	ConstraintFill
	// ConstraintFit, içerik boyutuna göre alan ayırır.
	ConstraintFit
)

// Constraint, tek bir esnek yerleşim kısıtlaması hücresidir.
// Tür (Type) ve ilişkili Değeri (Value) barındırır.
type Constraint struct {
	Type  ConstraintType
	Value uint16
}

// Fixed, sabit boyutta (karakter hücresi cinsinden) bir kısıtlama oluşturur.
// Örneğin: Fixed(10) -> tam olarak 10 satır veya sütunluk alan ayırır.
func Fixed(val uint16) Constraint {
	return Constraint{Type: ConstraintFixed, Value: val}
}

// Percentage, toplam kullanılabilir alanın yüzdesi kadar bir kısıtlama oluşturur (0-100).
// Örneğin: Percentage(30) -> alanın %30'unu ayırır.
func Percentage(val uint16) Constraint {
	if val > 100 {
		val = 100
	}
	return Constraint{Type: ConstraintPercentage, Value: val}
}

// Ratio, geriye kalan boş alanın ağırlıklı oranlarına göre dağıtılmasını sağlar.
// Örneğin: Ratio(2) ve Ratio(1) -> kalan alan 2/3 ve 1/3 oranlarında paylaştırılır.
func Ratio(val uint16) Constraint {
	if val == 0 {
		val = 1
	}
	return Constraint{Type: ConstraintRatio, Value: val}
}

// Min, en az belirtilen boyutta alan tahsis edilmesini garanti eder.
func Min(val uint16) Constraint {
	return Constraint{Type: ConstraintMin, Value: val}
}

// Max, en fazla belirtilen boyutta alan tahsis edilmesini sınırlar.
func Max(val uint16) Constraint {
	return Constraint{Type: ConstraintMax, Value: val}
}

// Fill, geriye kalan tüm boşluğu kaplayan bir kısıtlama oluşturur. (Ratio(1) ile eşdeğerdir).
func Fill() Constraint {
	return Constraint{Type: ConstraintFill}
}

// FitContent, içerik boyutuna (SizeHint) göre kısıtlama oluşturur.
func FitContent() Constraint {
	return Constraint{Type: ConstraintFit}
}

// FlexLayout, belirtilen kısıtlamalar ve yön doğrultusunda terminal alanını bölen esnek kutu yapısıdır.
type FlexLayout struct {
	// Direction, elemanların yatay mı dikey mi dizileceğini belirtir.
	Direction Direction
	// Gap, bölünen alanlar arasında bırakılacak boşluk mesafesidir (hücre sayısı).
	Gap uint16
	// Constraints, alanın nasıl bölünmesi gerektiğini tanımlayan kısıtlamalar listesidir.
	Constraints []Constraint
}

// NewFlexLayout yeni bir esnek yerleşim (FlexLayout) motoru oluşturup döndürür.
func NewFlexLayout(dir Direction, gap uint16, constraints ...Constraint) FlexLayout {
	return FlexLayout{
		Direction:   dir,
		Gap:         gap,
		Constraints: constraints,
	}
}

// Split, parametre olarak verilen ana Rect alanını, kısıtlamalara göre alt alanlara böler.
// Yuvarlama hatalarını ve boşluk (gap) hesaplarını sıfır heap tahsisatı ile yönetir.
func (fl FlexLayout) Split(area cell.Rect, fitSizes ...uint16) []cell.Rect {
	if len(fl.Constraints) == 0 {
		return nil
	}

	// Build key
	var key layoutKey
	key.direction = fl.Direction
	key.gap = fl.Gap
	key.area = area
	key.numC = len(fl.Constraints)
	if key.numC > 16 {
		key.numC = 16
	}
	for i := 0; i < key.numC; i++ {
		key.constraints[i] = ConstraintKey{
			Type:  fl.Constraints[i].Type,
			Value: fl.Constraints[i].Value,
		}
	}
	key.numF = len(fitSizes)
	if key.numF > 16 {
		key.numF = 16
	}
	for i := 0; i < key.numF; i++ {
		key.fitSizes[i] = fitSizes[i]
	}

	splitCache.RLock()
	if val, ok := splitCache.m[key]; ok {
		splitCache.RUnlock()
		return val
	}
	splitCache.RUnlock()

	// Bölme yönündeki toplam boyutu belirle (Genişlik veya Yükseklik)
	var totalSize uint16
	if fl.Direction == Horizontal {
		totalSize = area.Width
	} else {
		totalSize = area.Height
	}

	// Eğer alan boyutu sıfırsa sıfır boyutlu dikdörtgenler dön
	if totalSize == 0 {
		return make([]cell.Rect, len(fl.Constraints))
	}

	// Boşluk (gap) hesabı
	numGaps := len(fl.Constraints) - 1
	totalGap := uint16(0)
	if numGaps > 0 {
		totalGap = uint16(numGaps) * fl.Gap
	}

	// Kullanılabilir net alan hesabı
	var usableSize uint16
	if totalSize > totalGap {
		usableSize = totalSize - totalGap
	}

	sizes := make([]uint16, len(fl.Constraints))

	// 1. Aşama: Sabit ve yüzdesel oranlı (flexible olmayan) kısıtlamaları hesapla
	var fixedTotal uint16
	fitIdx := 0

	for i, c := range fl.Constraints {
		switch c.Type {
		case ConstraintFixed:
			sizes[i] = c.Value
			fixedTotal += c.Value
		case ConstraintPercentage:
			sz := (uint32(usableSize) * uint32(c.Value)) / 100
			sizes[i] = uint16(sz)
			fixedTotal += uint16(sz)
		case ConstraintMin:
			sizes[i] = c.Value
			fixedTotal += c.Value
		case ConstraintFit:
			sz := uint16(0)
			if fitIdx < len(fitSizes) {
				sz = fitSizes[fitIdx]
				fitIdx++
			}
			sizes[i] = sz
			fixedTotal += sz
		}
	}

	// Eğer sabit kısıtlamaların toplamı kullanılabilir alanı aşarsa, bunları oranlayarak küçült
	if fixedTotal > usableSize && fixedTotal > 0 {
		var scaledTotal uint16
		for i, c := range fl.Constraints {
			if c.Type != ConstraintRatio && c.Type != ConstraintFill && c.Type != ConstraintMin && c.Type != ConstraintMax {
				sz := (uint32(sizes[i]) * uint32(usableSize)) / uint32(fixedTotal)
				sizes[i] = uint16(sz)
				scaledTotal += uint16(sz)
			} else if c.Type == ConstraintMin {
				// Min kısıtlamaları da sabit boyutlu gibi küçülür
				sz := (uint32(sizes[i]) * uint32(usableSize)) / uint32(fixedTotal)
				sizes[i] = uint16(sz)
				scaledTotal += uint16(sz)
			}
		}
		// Yuvarlama kaynaklı kalan farkları dağıt
		diff := usableSize - scaledTotal
		for i := 0; i < len(sizes) && diff > 0; i++ {
			c := fl.Constraints[i]
			if c.Type != ConstraintRatio && c.Type != ConstraintFill && c.Type != ConstraintMax {
				sizes[i]++
				diff--
			}
		}
		fixedTotal = usableSize
	}

	// Geriye kalan boş alan
	remaining := usableSize - fixedTotal

	// 2. Aşama: Geriye kalan alanı oransal (Ratio, Fill, Min ve Max) kısıtlamalara dağıt
	if remaining > 0 {
		// Ağırlıklı büyüme yapacak elemanları belirle
		var activeMask uint32
		for i := 0; i < len(fl.Constraints) && i < 32; i++ {
			c := fl.Constraints[i]
			if c.Type == ConstraintRatio || c.Type == ConstraintFill || c.Type == ConstraintMin || c.Type == ConstraintMax {
				activeMask |= (1 << i)
			}
		}

		for activeMask > 0 && remaining > 0 {
			var totalWeight uint32
			for i := 0; i < len(fl.Constraints) && i < 32; i++ {
				if (activeMask & (1 << i)) != 0 {
					c := fl.Constraints[i]
					switch c.Type {
					case ConstraintRatio:
						totalWeight += uint32(c.Value)
					case ConstraintFill, ConstraintMin, ConstraintMax:
						totalWeight += 1
					}
				}
			}

			if totalWeight == 0 {
				break
			}

			var added [32]uint16
			var distributed uint16
			var cappedThisIteration bool

			for i := 0; i < len(fl.Constraints) && i < 32; i++ {
				if (activeMask & (1 << i)) != 0 {
					c := fl.Constraints[i]
					weight := uint32(1)
					if c.Type == ConstraintRatio {
						weight = uint32(c.Value)
					}
					sz := uint16((uint32(remaining) * weight) / totalWeight)

					// Max kısıtlaması aşım kontrolü
					if c.Type == ConstraintMax {
						currentTotal := sizes[i] + sz
						if currentTotal > c.Value {
							sz = c.Value - sizes[i] // Sadece limite kadar büyüt
							activeMask &^= (1 << i) // Artık bu eleman daha fazla büyüyemez
							cappedThisIteration = true
						}
					}

					added[i] = sz
					distributed += sz
				}
			}

			// Kalan yuvarlama farkını en son aktif elemana ekle (eğer bu iterasyonda hiç capped eleman yoksa)
			if !cappedThisIteration && remaining > distributed {
				diff := remaining - distributed
				for i := 0; i < len(fl.Constraints) && i < 32 && diff > 0; i++ {
					if (activeMask & (1 << i)) != 0 {
						added[i]++
						distributed++
						diff--
					}
				}
			}

			// Boyutları güncelle
			for i := 0; i < len(fl.Constraints) && i < 32; i++ {
				sizes[i] += added[i]
			}
			remaining -= distributed

			// Eğer bu iterasyonda hiçbir eleman limite takılmadıysa, tüm kalan alan dağıtılmıştır
			if !cappedThisIteration {
				break
			}
		}
	}

	// Sınır alanlarını hesapla ve dilim olarak döndür
	res := make([]cell.Rect, len(fl.Constraints))
	currX := area.X
	currY := area.Y

	for i, sz := range sizes {
		if fl.Direction == Horizontal {
			res[i] = cell.Rect{
				X:      currX,
				Y:      currY,
				Width:  sz,
				Height: area.Height,
			}
			currX += sz + fl.Gap
		} else {
			res[i] = cell.Rect{
				X:      currX,
				Y:      currY,
				Width:  area.Width,
				Height: sz,
			}
			currY += sz + fl.Gap
		}
	}

	splitCache.Lock()
	splitCache.m[key] = res
	splitCache.Unlock()

	return res
}
