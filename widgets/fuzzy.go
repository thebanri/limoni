package widgets

import (
	"sort"
	"strings"
	"unicode"
)

// FuzzyMatch, query'nin tüm karakterlerini sırasıyla target içinde arayarak
// bir eşleşme puanı ve eşleşme durumu döndürür.
// Büyük/küçük harf duyarsız çalışır.
// Ardışık eşleşmeler bonus puan alır (VS Code davranışı).
func FuzzyMatch(query, target string) (score int, matched bool) {
	if query == "" {
		return 0, true
	}

	queryLower := strings.ToLower(query)
	targetLower := strings.ToLower(target)

	queryRunes := []rune(queryLower)
	targetRunes := []rune(targetLower)
	originalRunes := []rune(target)

	qi := 0 // query index
	consecutive := 0
	totalScore := 0

	for ti := 0; ti < len(targetRunes) && qi < len(queryRunes); ti++ {
		if targetRunes[ti] == queryRunes[qi] {
			// Temel eşleşme puanı
			points := 1

			// Ardışık eşleşme bonusu (gittikçe artan)
			consecutive++
			points += consecutive * 2

			// Kelime başlangıcı bonusu (ilk karakter veya öncesinde boşluk/tire/alt çizgi)
			if ti == 0 {
				points += 10
			} else {
				prev := targetRunes[ti-1]
				if prev == ' ' || prev == '-' || prev == '_' || prev == '/' {
					points += 8
				}
				// CamelCase bonusu
				if unicode.IsUpper(originalRunes[ti]) && unicode.IsLower(originalRunes[ti-1]) {
					points += 6
				}
			}

			// Tam eşleşme bonusu (büyük/küçük harf birebir aynı)
			if originalRunes[ti] == []rune(query)[qi] {
				points += 1
			}

			totalScore += points
			qi++
		} else {
			consecutive = 0
		}
	}

	if qi < len(queryRunes) {
		return 0, false
	}

	return totalScore, true
}

// FuzzyResult, bulanık arama sonucunu temsil eder.
type FuzzyResult struct {
	Item  CommandItem
	Score int
}

// FuzzyFilter, verilen query ile tüm öğeleri filtreler ve skora göre azalan sırada döndürür.
func FuzzyFilter(query string, items []CommandItem) []CommandItem {
	if query == "" {
		// Boş sorgu: tüm öğeleri kategoriye göre sıralı döndür
		result := make([]CommandItem, len(items))
		copy(result, items)
		sort.SliceStable(result, func(i, j int) bool {
			if result[i].Category != result[j].Category {
				return result[i].Category < result[j].Category
			}
			return result[i].Label < result[j].Label
		})
		return result
	}

	var results []FuzzyResult

	for _, item := range items {
		// Hem Label hem de Category üzerinde arama yap
		labelScore, labelMatch := FuzzyMatch(query, item.Label)
		catScore, catMatch := FuzzyMatch(query, item.Category)
		detailScore, detailMatch := FuzzyMatch(query, item.Detail)

		bestScore := 0
		anyMatch := false

		if labelMatch && labelScore > bestScore {
			bestScore = labelScore
			anyMatch = true
		}
		if catMatch && catScore > bestScore {
			bestScore = catScore
			anyMatch = true
		}
		if detailMatch && detailScore > bestScore {
			bestScore = detailScore
			anyMatch = true
		}

		if anyMatch {
			results = append(results, FuzzyResult{Item: item, Score: bestScore})
		}
	}

	// Skora göre azalan sırada sırala
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	filtered := make([]CommandItem, len(results))
	for i, r := range results {
		filtered[i] = r.Item
	}

	return filtered
}
