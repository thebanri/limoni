//go:build linux

package overlay

// history is every row of cells Follow has seen at a known place, by its
// row counted from the top of the bottom screen (rows above it negative):
// when a view has nothing in common with the last one followed (a picture
// as tall as the screen scrolled through), the text that comes into view
// was seen before, when it was output, and says where the view is.
// Only the middle line of a row is kept, maxHistory rows of it.
type history map[int]histRow

type histRow struct {
	line []byte
	skip [][2]int
}

const maxHistory = 1000

// mid is the middle sample line of row r of s, and its skipped ranges.
func mid(s *sample, r int) ([]byte, [][2]int) {
	n := len(s.rows[r]) / len(sampleLines)
	from, to := n*(len(sampleLines)/2), n*(len(sampleLines)/2+1)
	var skip [][2]int
	for _, x := range s.skip[r] {
		if x[1] > from && x[0] < to {
			skip = append(skip, [2]int{max(x[0], from) - from, min(x[1], to) - from})
		}
	}
	return s.rows[r][from:to], skip
}

// shift moves every row up by n: output scrolled the screen n lines.
func (h *history) shift(n int) {
	if n == 0 || len(*h) == 0 {
		return
	}
	moved := make(history, len(*h))
	for r, row := range *h {
		if r-n > -maxHistory {
			moved[r-n] = row
		}
	}
	*h = moved
}

// remember keeps the rows of s, a view k rows up from the bottom, but for
// the row of cells the cursor may blink in.
func (h *history) remember(s *sample, k, cursor int) {
	if *h == nil {
		*h = history{}
	}
	for r := range s.rows {
		if r == cursor || r-k <= -maxHistory {
			continue
		}
		line, skip := mid(s, r)
		(*h)[r-k] = histRow{append([]byte(nil), line...), skip}
	}
}

// recall finds the view b among the rows seen before: the k that puts
// every row of b on the one remembered there, with two rows that are not
// blank among them, and no other k that does.
func (h history) recall(b *sample) (int, bool) {
	first := -1
	for r := range b.rows {
		line, skip := mid(b, r)
		if _, info := sameRow(line, line, skip, nil); info {
			first = r
			break
		}
	}
	if first < 0 {
		return 0, false
	}
	line, skip := mid(b, first)
	found, at := 0, 0
	for br, row := range h {
		k := first - br
		if k < 0 {
			continue
		}
		if same, _ := sameRow(line, row.line, skip, row.skip); !same {
			continue
		}
		fits, informative := true, 0
		for r := range b.rows {
			row, ok := h[r-k]
			if !ok {
				continue
			}
			l, s := mid(b, r)
			var info bool
			if fits, info = sameRow(l, row.line, s, row.skip); !fits {
				break
			}
			if info {
				informative++
			}
		}
		if fits && informative >= 2 {
			found++
			at = k
		}
	}
	return at, found == 1
}
