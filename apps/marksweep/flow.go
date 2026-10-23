package main

// The way to the player round the walls: a breadth-first count of steps
// from the player's tile over the whole map, made again only when the
// player moves to another tile. An enemy with nothing in between walks
// straight at the player; one with a wall in the way follows the count
// downhill, so the wall is walked round, not pushed against.

const unreached = 255

// flowFrom counts the steps to every floor tile from the player's.
func (g *game) flowFrom() {
	px, py := int(g.pos.x), int(g.pos.y)
	at := py*mapW + px
	if at == g.flowAt && g.flowArena == cur {
		return
	}
	g.flowAt, g.flowArena = at, cur
	for i := range g.flow {
		g.flow[i] = unreached
	}
	if tile(px, py) != '.' {
		return
	}
	var queue [mapW * mapH]int16
	head, tail := 0, 0
	g.flow[at] = 0
	queue[tail] = int16(at)
	tail++
	for head < tail {
		c := int(queue[head])
		head++
		x, y := c%mapW, c/mapW
		next := g.flow[c] + 1
		if next == unreached {
			continue
		}
		for _, d := range steps8 {
			nx, ny := x+d[0], y+d[1]
			if !g.passable(x, y, nx, ny) {
				continue
			}
			if n := ny*mapW + nx; g.flow[n] == unreached {
				g.flow[n] = next
				queue[tail] = int16(n)
				tail++
			}
		}
	}
}

var steps8 = [8][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}}

// passable reports whether a step from tile (x, y) to its neighbour
// (nx, ny) is open: a diagonal one only if it cuts no corner.
func (g *game) passable(x, y, nx, ny int) bool {
	if tile(nx, ny) != '.' {
		return false
	}
	return nx == x || ny == y || tile(nx, y) == '.' && tile(x, ny) == '.'
}

// direct reports whether the way from p to the player is as short as the
// straight one: no wall between that a step round would have to go round.
// It costs a look-up, where a line of sight costs a walk along the line.
func (g *game) direct(p vec) bool {
	x, y := int(p.x), int(p.y)
	if x < 0 || y < 0 || x >= mapW || y >= mapH {
		return true
	}
	f := g.flow[y*mapW+x]
	return f == unreached || int(f) <= max(abs(x-int(g.pos.x)), abs(y-int(g.pos.y)))
}

// flowDir is the way from p to the player by the count: towards the middle
// of the neighbouring tile nearest them, or along n if p is off the count.
func (g *game) flowDir(p, n vec) vec {
	x, y := int(p.x), int(p.y)
	if x < 0 || y < 0 || x >= mapW || y >= mapH {
		return n
	}
	best, bx, by := g.flow[y*mapW+x], -1, -1
	for _, d := range steps8 {
		nx, ny := x+d[0], y+d[1]
		if !g.passable(x, y, nx, ny) {
			continue
		}
		if f := g.flow[ny*mapW+nx]; f < best {
			best, bx, by = f, nx, ny
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
