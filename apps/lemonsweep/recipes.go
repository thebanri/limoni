package main

import limoni "github.com/thebanri/limoni"

// The recipe book: cards that are worth more together than apart, from the
// title or the pause. Every recipe is a rule the game really has — a card
// that needs another (augs' needs), stacks one card adds and another reads,
// a blast that catches more of a crowd another card made. In a run the
// cards held are lit, and a card offered that belongs to a recipe already
// begun says so (comboHint).

type recipe struct {
	name  string
	cards []augID // in the order they can be had
	how   string
}

var recipes = [...]recipe{
	{"Fragmentation", []augID{augREP, augFork, augRecursion}, "more shots, each splits on a hit, and the splits split"},
	{"Chain Reaction", []augID{augPanic, augCascade, augNapalm}, "hits explode; blasts throw shrapnel and set fires"},
	{"Pinball", []augID{augScroll, augDiff, augBigO}, "shots bounce and pierce, so they fly long and grow"},
	{"Thermal Shock", []augID{augChill, augIgnite, augThermal}, "freeze them, then burn them: a 400% blast"},
	{"Wildfire", []augID{augIgnite, augFirewall, augWildfire, augOverheat}, "burns hotter, hurt more, and spread when they die"},
	{"Deep Freeze", []augID{augChill, augPermafrost, augShatter, augShards}, "the frozen stay frozen, take more, die in shards"},
	{"Overvoltage", []augID{augStatic, augChain, augConductor}, "the shocked take more, and arcs hit harder and further"},
	{"Superconductor", []augID{augChain, augSuper, augIgnite, augChill}, "every arc carries your burn and your chill"},
	{"Storm Front", []augID{augStatic, augThunder, augStorm}, "bolts from above, on foes that take more from them"},
	{"Pandemic", []augID{augLeak, augPlague, augCorrode, augDangling}, "poison spreads, eats armour and takes % of health"},
	{"Crowd Control", []augID{augCompaction, augPanic, augFumes}, "pull them into a pile, then blast and gas the pile"},
	{"Firewalker", []augID{augHotReload, augBurnIn, augAfterburn}, "move fast and dash often: fire behind you everywhere"},
	{"Hit and Run", []augID{augDoubleBuf, augCtxSwitch, augDefer, augRam}, "dash often: it refills the heap, hits, and arms shots"},
	{"Bottomless Heap", []augID{augHeap, augZeroAlloc, augRefCount}, "free shots and refunded rounds: the heap rarely runs dry"},
	{"Bloodletting", []augID{augEscape, augVampire, augLoop}, "fast crits, and every crit heals"},
	{"Headroom", []augID{augHeadroom, augWatchdog, augMutex, augCryo}, "stay above 80% health and keep its bonus"},
	{"Boss Hunter", []augID{augGenGC, augBatch, augHotPath}, "slugs and bonus damage for wardens and bosses"},
	{"Polyglot", []augID{augPolyglot, augIgnite, augChill, augStatic, augLeak}, "+15% damage for every element carried"},
}

// has is how many of recipe r's cards the build has.
func (g *game) has(r *recipe) int {
	n := 0
	for _, a := range r.cards {
		if g.rank[a] > 0 {
			n++
		}
	}
	return n
}

// comboHint is the recipe card a would take furthest, if the build holds
// another of its cards: the one most begun.
func (g *game) comboHint(a augID) (*recipe, bool) {
	var best *recipe
	most := 0
	for i := range recipes {
		r := &recipes[i]
		in := false
		for _, b := range r.cards {
			in = in || b == a
		}
		if !in {
			continue
		}
		n := g.has(r)
		if g.rank[a] > 0 {
			n-- // the card itself does not count as another
		}
		if n > most {
			best, most = r, n
		}
	}
	return best, best != nil
}

func (g *game) openBook() {
	g.setFrom, g.phase, g.bookTop = g.phase, phRecipes, 0
	g.sound(sfxMenu)
}

func (g *game) scrollBook(d int) {
	g.bookTop = min(max(g.bookTop+d, 0), len(recipes)-1)
}

func (g *game) bookKey(k limoni.KeyEvent, isRune bool, ch rune) {
	switch {
	case isRune && ch == 'w' || k.Type == limoni.KeyUp:
		g.scrollBook(-1)
	case isRune && ch == 's' || k.Type == limoni.KeyDown:
		g.scrollBook(1)
	case k.Type == limoni.KeyEsc || k.Type == limoni.KeyEnter || k.Type == limoni.KeyBackspace:
		g.closeSettings() // back to whichever opened it, the same way
	}
}

// drawBook is the recipe book: a recipe's name, its cards in their
// rarity's colours (lit when held, in a run), and what they do together.
func (g *game) drawBook(c canvas) {
	veil(c, 0.6, hex(0x03040a))
	w, h := min(c.w-4, 76), c.h-4
	x, y := (c.w-w)/2, 2
	card(c, x, y, w, h, hex(cardBorder))
	bg := hex(cardBg)
	c.big(x+4, 2*(y+1)+1, "RECIPES", 1, hudWarn)
	c.text(x+4+7*4+2, y+3, "cards that work together", style(hudDim, bg))
	inRun := !g.onTitle()
	yy, end := y+6, y+h-3
	shown := 0
	for i := g.bookTop; i < len(recipes); i++ {
		r := &recipes[i]
		if yy+3 > end {
			break
		}
		n := g.has(r)
		head := hudWarn
		if inRun && n == 0 {
			head = hudWarn.mix(hudDim, 0.5)
		}
		hx := c.text(x+3, yy, r.name, style(head, bg))
		if inRun && n > 0 {
			hx = c.text(hx+1, yy, "· ", style(hudDim, bg))
			hx = c.number(hx, yy, n, style(hudGood, bg))
			c.text(hx, yy, "/", style(hudDim, bg))
			c.number(hx+1, yy, len(r.cards), style(hudDim, bg))
		}
		yy++
		cx := x + 5
		for j, a := range r.cards {
			name := augs[a].name
			if j > 0 {
				cx = c.text(cx, yy, " + ", style(hudDim, bg))
			}
			if cx+width(name) > x+w-3 {
				yy, cx = yy+1, x+7
			}
			col := hex(rarityCol[augs[a].rar])
			if inRun && g.rank[a] == 0 {
				col = col.mix(bg, 0.55)
			}
			cx = c.text(cx, yy, name, style(col, bg))
		}
		yy++
		c.text(x+5, yy, clip(r.how, w-8), style(hudText.mix(hudDim, 0.4), bg))
		yy += 2
		shown++
	}
	up, down := g.bookTop > 0, g.bookTop+shown < len(recipes)
	hint := bookHints[0]
	switch {
	case up && down:
		hint = bookHints[3]
	case down:
		hint = bookHints[1]
	case up:
		hint = bookHints[2]
	}
	c.centered(y+h-2, hint, style(hex(cardBorder).mix(hudDim, 0.6), bg))
}

var bookHints = [...]string{
	"W/S or wheel scroll · Esc back",
	"▼ more · W/S or wheel scroll · Esc back",
	"▲ more · W/S or wheel scroll · Esc back",
	"▲ ▼ more · W/S or wheel scroll · Esc back",
}

// clip cuts s to n characters, as a string of its own bytes: no copy.
func clip(s string, n int) string {
	for i := range s {
		if n == 0 {
			return s[:i]
		}
		n--
	}
	return s
}
