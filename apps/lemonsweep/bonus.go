package main

import "math"

// Once every slot is full and every card held is at its last rank, there
// is no card to offer: a level is a random stat instead, taken at once,
// with no card screen.

type bonusID uint8

const (
	bonDamage bonusID = iota
	bonRate
	bonHealth
	bonSpeed
	bonCrit
	bonAmmo
	bonRegen
	bonArmor
	nBonus
)

var bonusName = [nBonus]string{
	"+6% damage", "+5% fire rate", "+15 max health", "+4% speed",
	"+3% critical chance", "+3 rounds", "+0.4 health a second", "-4% damage taken",
}

// bonuses adds the random stats taken so far to s.
func (g *game) bonuses(s *stats) {
	n := func(b bonusID) float64 { return float64(g.bonus[b]) }
	s.dmg *= math.Pow(1.06, n(bonDamage))
	s.fireRate *= math.Pow(1.05, n(bonRate))
	s.maxHP += 15 * n(bonHealth)
	s.speed *= math.Pow(1.04, n(bonSpeed))
	s.crit += 0.03 * n(bonCrit)
	s.mag += 3 * g.bonus[bonAmmo]
	s.regen += 0.4 * n(bonRegen)
	s.taken *= math.Pow(0.96, n(bonArmor))
}

// randomStat is a level with nothing to offer: a stat of chance, said.
func (g *game) randomStat() {
	b := bonusID(g.rnd.intn(int(nBonus)))
	g.bonus[b]++
	oldMax := g.st.maxHP
	g.recalc()
	g.hp = math.Min(g.st.maxHP, g.hp+g.st.maxHP-oldMax)
	g.say("LEVEL UP · " + bonusName[b])
	g.sound(sfxPick)
}
