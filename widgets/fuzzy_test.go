package widgets

import "testing"

func TestFuzzyMatch_EmptyQuery(t *testing.T) {
	score, matched := FuzzyMatch("", "anything")
	if !matched {
		t.Fatal("empty query should always match")
	}
	if score != 0 {
		t.Fatalf("empty query should score 0, got %d", score)
	}
}

func TestFuzzyMatch_ExactPrefix(t *testing.T) {
	score, matched := FuzzyMatch("Gra", "Grafik Sekmesine Git")
	if !matched {
		t.Fatal("should match")
	}
	if score <= 0 {
		t.Fatalf("score should be positive, got %d", score)
	}
}

func TestFuzzyMatch_Subsequence(t *testing.T) {
	_, matched := FuzzyMatch("gsg", "Grafik Sekmesine Git")
	if !matched {
		t.Fatal("should match subsequence g-s-g")
	}
}

func TestFuzzyMatch_NoMatch(t *testing.T) {
	_, matched := FuzzyMatch("xyz", "Grafik")
	if matched {
		t.Fatal("should not match")
	}
}

func TestFuzzyMatch_CaseInsensitive(t *testing.T) {
	_, matched := FuzzyMatch("GRAFIK", "grafik sekmesi")
	if !matched {
		t.Fatal("should match case-insensitively")
	}
}

func TestFuzzyMatch_ConsecutiveBonus(t *testing.T) {
	scoreConsec, _ := FuzzyMatch("gra", "Grafik")
	scoreSparse, _ := FuzzyMatch("gfk", "Grafik")
	if scoreConsec <= scoreSparse {
		t.Fatalf("consecutive match should score higher: consec=%d sparse=%d", scoreConsec, scoreSparse)
	}
}

func TestFuzzyFilter_EmptyQuery(t *testing.T) {
	items := []CommandItem{
		{Label: "Beta", Category: "B"},
		{Label: "Alpha", Category: "A"},
	}
	result := FuzzyFilter("", items)
	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}
	// Should be sorted by category first
	if result[0].Label != "Alpha" {
		t.Fatalf("expected Alpha first (category A), got %s", result[0].Label)
	}
}

func TestFuzzyFilter_FiltersCorrectly(t *testing.T) {
	items := []CommandItem{
		{Label: "Grafik Sekmesine Git", Category: "Navigasyon"},
		{Label: "Form Sekmesine Git", Category: "Navigasyon"},
		{Label: "Yardım Panelini Aç", Category: "Görünüm"},
	}
	result := FuzzyFilter("grafik", items)
	if len(result) == 0 {
		t.Fatal("should find at least one result")
	}
	if result[0].Label != "Grafik Sekmesine Git" {
		t.Fatalf("first result should be Grafik, got %s", result[0].Label)
	}
}

func TestFuzzyFilterByStablePreservesOrder(t *testing.T) {
	items := []string{"105.4 MB go", "256.1 MB go", "75.4 MB go"}
	result := FuzzyFilterByStable("go", items, func(item string) string { return item })
	for i := range items {
		if result[i] != items[i] {
			t.Fatalf("stable result[%d] = %q; want %q", i, result[i], items[i])
		}
	}
}

func TestFuzzyFilterByTableRows(t *testing.T) {
	rows := []TableRow{NewRow("limoni_demo", "Running"), NewRow("vscode", "Idle")}
	result := FuzzyFilterBy("vsc", rows, func(row TableRow) string { return row.SearchText() })
	if len(result) != 1 || result[0].Cells[0].Text != "vscode" {
		t.Fatalf("filtered rows = %+v; want vscode row", result)
	}
}

func TestFuzzyFilter_NoResults(t *testing.T) {
	items := []CommandItem{
		{Label: "Grafik", Category: "A"},
	}
	result := FuzzyFilter("zzzzz", items)
	if len(result) != 0 {
		t.Fatalf("expected 0 results, got %d", len(result))
	}
}
