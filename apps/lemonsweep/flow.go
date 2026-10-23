package main

// The way to the player round the walls: a breadth-first count of steps
// from the player's tile over the whole map, made again only when the
// player moves to another tile. An enemy with nothing in between walks
// straight at the player; one with a wall in the way follows the count
// downhill, so the wall is walked round, not pushed against.
//
// A boss is wider than a tile and sticks in a gap a small enemy walks
// through, so it has a count of its own, made only over roomy tiles.

const unreached = 255

// flowFrom counts the steps to every floor tile from the player's: over
// all floor for flow[0], over roomy floor for flow[1].
func (g *game) flowFrom() {
	px, py := int(g.pos.x), int(g.pos.y)
	at := py*mapW + px
	if at == g.flowAt && g.flowArena == cur {
		return
	}
	g.flowAt, g.flowArena = at, cur
	for big := range g.flow {
		f := &g.flow[big]
		for i := range f {
			f[i] = unreached
		}
		if tile(px, py) != '.' {
			continue
		}
		var queue [mapW * mapH]int16
		head, tail := 0, 0
		f[at] = 0
		queue[tail] = int16(at)
		tail++
		if big == 1 && !roomy[at] {
			// The player is in a gap a boss does not fit: the roomy floor
			// beside it is as near as a boss gets.
			for dy := -2; dy <= 2; dy++ {
				for dx := -2; dx <= 2; dx++ {
					x, y := px+dx, py+dy
					if x >= 0 && y >= 0 && x < mapW && y < mapH && roomy[y*mapW+x] && f[y*mapW+x] == unreached {
						f[y*mapW+x] = 0
						queue[tail] = int16(y*mapW + x)
						tail++
					}
				}
			}
		}
		for head < tail {
			c := int(queue[head])
			head++
			x, y := c%mapW, c/mapW
			next := f[c] + 1
			if next == unreached {
				continue
			}
			for _, d := range steps8 {
				nx, ny := x+d[0], y+d[1]
				if !passable(x, y, nx, ny, big == 1) {
					continue
				}
				if n := ny*mapW + nx; f[n] == unreached {
					f[n] = next
					queue[tail] = int16(n)
					tail++
				}
			}
		}
	}
}

var steps8 = [8][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}}

// passable reports whether a step from tile (x, y) to its neighbour
// (nx, ny) is open: a diagonal one only if it cuts no corner, and for a big
// body only onto a roomy tile.
func passable(x, y, nx, ny int, big bool) bool {
	if tile(nx, ny) != '.' || big && !roomy[ny*mapW+nx] {
		return false
	}
	return nx == x || ny == y || tile(nx, y) == '.' && tile(x, ny) == '.'
}

// direct reports whether the way from p to the player is as short as the
// straight one: no wall between that a step round would have to go round.
// It costs a look-up, where a line of sight costs a walk along the line.
func (g *game) direct(p vec, big int) bool {
	x, y := int(p.x), int(p.y)
	if x < 0 || y < 0 || x >= mapW || y >= mapH {
		return true
	}
	f := g.flow[big][y*mapW+x]
	if big == 1 {
		// A straight line as short as the count can still clip a wall's
		// end with a wide body: a boss follows its count all the way.
		return f == 0
	}
	return f == unreached || int(f) <= max(abs(x-int(g.pos.x)), abs(y-int(g.pos.y)))
}

// flowDir is the way from p to the player by the count: towards the middle
// of the neighbouring tile nearest them, or along n if p is off the count.
func (g *game) flowDir(p, n vec, big int) vec {
	x, y := int(p.x), int(p.y)
	if x < 0 || y < 0 || x >= mapW || y >= mapH {
		return n
	}
	f := &g.flow[big]
	best, bx, by := f[y*mapW+x], -1, -1
	for _, d := range steps8 {
		nx, ny := x+d[0], y+d[1]
		if !passable(x, y, nx, ny, big == 1) {
			continue
		}
		if c := f[ny*mapW+nx]; c < best {
			best, bx, by = c, nx, ny
		}
	}
	if bx < 0 {
		return n
	}
	to := vec{float64(bx) + 0.5 - p.x, float64(by) + 0.5 - p.y}
	if l := to.len(); l > 0.001 {
		return to.scale(1 / l)
	}
	return n
}

// roomy marks the floor tiles whose eight neighbours are floor too: room
// for a boss to stand in. useArena fills it.
var roomy [mapW * mapH]bool

func markRoomy() {
	for y := 0; y < mapH; y++ {
		for x := 0; x < mapW; x++ {
			ok := true
			for dy := -1; dy <= 1 && ok; dy++ {
				for dx := -1; dx <= 1 && ok; dx++ {
					ok = tile(x+dx, y+dy) == '.'
				}
			}
			roomy[y*mapW+x] = ok
		}
	}
}
