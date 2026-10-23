package main

import (
	"strings"
	"testing"

	limoni "github.com/thebanri/limoni"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// Every recipe can be made: its cards are different, and each one that is
// offered only after another has that other in the recipe too.
func TestEveryRecipeCanBeMade(t *testing.T) {
	for _, r := range recipes {
		if len(r.cards) < 2 {
			t.Errorf("%s: a recipe of %d cards", r.name, len(r.cards))
		}
		seen := map[augID]bool{}
		for _, a := range r.cards {
			if seen[a] {
				t.Errorf("%s: %s twice", r.name, augs[a].name)
			}
			seen[a] = true
		}
		for _, a := range r.cards {
			if n := augs[a].needs; n != a && !seen[n] {
				t.Errorf("%s: %s needs %s, which is not in it", r.name, augs[a].name, augs[n].name)
			}
		}
	}
}

// An offered card names the recipe it takes furthest, counting only what
// else is held; a card no recipe of the build wants names none.
func TestCardsNameTheirRecipe(t *testing.T) {
	g := playing()
	if _, ok := g.comboHint(augFirewall); ok {
		t.Fatal("a recipe named with nothing of it held")
	}
	pickN(g, augIgnite, 1)
	pickN(g, augWildfire, 1)
	r, ok := g.comboHint(augFirewall)
	if !ok || r.name != "Wildfire" {
		t.Fatalf("Firewall with Ignite and Wildfire held named %v", r)
	}
	if r, _ := g.comboHint(augIgnite); r.name != "Wildfire" {
		t.Fatalf("a held card counted itself: %s", r.name)
	}
}

// The book opens from the title and the pause, scrolls, goes back where
// it came from, and draws without allocating.
func TestTheRecipeBook(t *testing.T) {
	g := newGame(1)
	b := buffer.NewBuffer(cell.Rect{Width: 120, Height: 40})
	g.titlePick(2)
	if g.phase != phRecipes || !g.onTitle() {
		t.Fatalf("RECIPES on the title gave phase %v", g.phase)
	}
	g.render(b)
	snap := snapshot(b)
	for _, s := range []string{recipes[0].name, augs[recipes[0].cards[0]].name, "Esc back", "▼ more"} {
		if !strings.Contains(snap, s) {
			t.Errorf("the book does not show %q", s)
		}
	}
	for range len(recipes) + 3 {
		g.key(limoni.KeyEvent{Type: limoni.KeyDown})
	}
	if g.bookTop != len(recipes)-1 {
		t.Errorf("scrolled to %d of %d", g.bookTop, len(recipes))
	}
	g.render(b)
	if !strings.Contains(snapshot(b), recipes[len(recipes)-1].name) {
		t.Error("the last recipe is not shown at the end")
	}
	if avg := testing.AllocsPerRun(20, func() { g.render(b) }); avg != 0 {
		t.Errorf("%.1f allocations a frame on the book", avg)
	}
	g.key(limoni.KeyEvent{Type: limoni.KeyEsc})
	if g.phase != phTitle {
		t.Fatalf("Esc went to phase %v", g.phase)
	}

	g = playing()
	g.pauseRun()
	g.pausePick(1)
	g.key(limoni.KeyEvent{Type: limoni.KeyEsc})
	if g.phase != phPause {
		t.Fatalf("the book from the pause went back to phase %v", g.phase)
	}
}
