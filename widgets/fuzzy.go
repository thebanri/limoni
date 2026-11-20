package widgets

import (
	"sort"
	"unicode"
	"unicode/utf8"
)

// FuzzyMatch looks for every character of query, in order, in target, and
// returns a match score and whether it matched.
// It is case-insensitive.
// Consecutive matches earn bonus points (as in VS Code).
// When the query can match in more than one place, the best-scoring
// alignment wins (see fuzzyBest).
func FuzzyMatch(query, target string) (score int, matched bool) {
	if query == "" {
		return 0, true
	}
	_, score, matched = fuzzyBest(query, target)
	return score, matched
}

// fuzzyBest tries each position of target where the first query character
// occurs, matches the rest of query greedily from there, and returns the byte
// offset of the start that scores highest (the earliest one on a tie). It
// does not allocate, so the draw path can use it to highlight the same
// characters that were scored.
// This costs O(len(target) x occurrences of the first query rune). A DP over
// (query index, target index) would also choose the best position for the
// later query characters, but the growing consecutive bonus makes its state
// larger than it is worth here.
func fuzzyBest(query, target string) (start, score int, matched bool) {
	first, _ := utf8.DecodeRuneInString(query)
	first = unicode.ToLower(first)
	for i, r := range target {
		if unicode.ToLower(r) != first {
			continue
		}
		s, ok := fuzzyScoreAt(query, target, i)
		if !ok {
			// A later start sees a suffix of this one, so it cannot match either.
			break
		}
		if !matched || s > score {
			start, score, matched = i, s, true
		}
	}
	return start, score, matched
}

// fuzzyScoreAt scores query against target with the first query character
// matched at byte offset start of target and the rest matched greedily.
func fuzzyScoreAt(query, target string, start int) (score int, matched bool) {
	prev, _ := utf8.DecodeLastRuneInString(target[:start])
	consecutive := 0
	for i, orig := range target[start:] {
		if query == "" {
			break
		}
		ti := start + i
		q, qsize := utf8.DecodeRuneInString(query)
		if unicode.ToLower(orig) != unicode.ToLower(q) {
			consecutive = 0
			prev = orig
			continue
		}
		// Base match score
		points := 1

		// Consecutive match bonus (growing)
		consecutive++
		points += consecutive * 2

		// Word start bonus (first character, or after a space/hyphen/underscore/slash)
		if ti == 0 {
			points += 10
		} else {
			if prev == ' ' || prev == '-' || prev == '_' || prev == '/' {
				points += 8
			}
			// CamelCase bonus
			if unicode.IsUpper(orig) && unicode.IsLower(prev) {
				points += 6
			}
		}

		// Exact match bonus (same case)
		if orig == q {
			points += 1
		}

		score += points
		query = query[qsize:]
		prev = orig
	}
	return score, query == ""
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

// FuzzyResult is a fuzzy search result.
type FuzzyResult struct {
	Item  CommandItem
	Score int
}

// FuzzyFilter filters all items by query and returns them sorted by descending score.
func FuzzyFilter(query string, items []CommandItem) []CommandItem {
	if query == "" {
		// Empty query: return every item, sorted by category
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
