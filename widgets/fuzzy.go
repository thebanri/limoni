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

// FuzzyFilterBy filters arbitrary values using their searchable text.
// It is shared by command palettes, tables and other data widgets.
func FuzzyFilterBy[T any](query string, items []T, text func(T) string) []T {
	if query == "" {
		result := make([]T, len(items))
		copy(result, items)
		return result
	}
	results := make([]struct {
		item  T
		score int
		index int
	}, 0, len(items))
	for index, item := range items {
		score, matched := FuzzyMatch(query, text(item))
		if matched {
			results = append(results, struct {
				item  T
				score int
				index int
			}{item: item, score: score, index: index})
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].score == results[j].score {
			return results[i].index < results[j].index
		}
		return results[i].score > results[j].score
	})
	filtered := make([]T, len(results))
	for i, result := range results {
		filtered[i] = result.item
	}
	return filtered
}

// FuzzyFilterByFields filters arbitrary values against multiple searchable fields,
// keeping the best field score for each item.
func FuzzyFilterByFields[T any](query string, items []T, fields func(T) []string) []T {
	results := make([]struct {
		item  T
		score int
		index int
	}, 0, len(items))
	for index, item := range items {
		bestScore := 0
		matched := false
		for _, field := range fields(item) {
			score, ok := FuzzyMatch(query, field)
			if ok && score > bestScore {
				bestScore, matched = score, true
			}
		}
		if matched {
			results = append(results, struct {
				item  T
				score int
				index int
			}{item: item, score: bestScore, index: index})
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].score == results[j].score {
			return results[i].index < results[j].index
		}
		return results[i].score > results[j].score
	})
	filtered := make([]T, len(results))
	for i, result := range results {
		filtered[i] = result.item
	}
	return filtered
}

// FuzzyFilterByStable filters arbitrary values without changing their input order.
// This is useful when another ordering, such as table sorting, already exists.
func FuzzyFilterByStable[T any](query string, items []T, text func(T) string) []T {
	if query == "" {
		result := make([]T, len(items))
		copy(result, items)
		return result
	}
	filtered := make([]T, 0, len(items))
	for _, item := range items {
		if _, matched := FuzzyMatch(query, text(item)); matched {
			filtered = append(filtered, item)
		}
	}
	return filtered
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

	return FuzzyFilterByFields(query, items, func(item CommandItem) []string {
		return []string{item.Label, item.Category, item.Detail}
	})
}
