package cell

import (
	"sync"
	"unicode/utf8"

	"github.com/thebanri/limoni/core/grapheme"
)

// A Cell holds one rune, but a character on screen can be several: "é" as e
// plus a combining accent, a flag made of two regional indicators, a family
// emoji joined from five code points. Such a cluster is stored as a handle —
// a Content value above the Unicode range — that indexes a shared table of
// cluster text. Single code points, which is nearly all text, are stored as
// themselves and never touch the table, so Cell stays 16 bytes and the common
// path is unchanged.
//
// The table is append-only and shared by every buffer, because a front and a
// back buffer compare Content values directly: a handle has to mean the same
// cluster in both.

// RuneClusterBase is the first cluster handle. Every Content value at or above
// it refers to a multi-code-point grapheme cluster rather than a rune.
const RuneClusterBase rune = utf8.MaxRune + 1

// maxClusters bounds the table. An application displaying user content sees a
// bounded set of distinct emoji sequences and accented letters in practice;
// the cap exists so an adversarial stream of unique clusters cannot grow the
// process without limit. Past it, a cluster degrades to its first code point.
var maxClusters = 1 << 20

type clusterTable struct {
	mu     sync.RWMutex
	byText map[string]rune
	text   []string
	width  []uint8
	// alone is, for each cluster, the code point to draw in its place on
	// a terminal that draws code points one by one, or 0 when the cluster
	// is safe there as it is (ClusterFallback).
	alone []rune
}

var clusters = clusterTable{byText: make(map[string]rune)}

// SetGraphemeClusters switches between grapheme-cluster segmentation (the
// default) and one code point per cell. See grapheme.SetClusters.
func SetGraphemeClusters(enabled bool) { grapheme.SetClusters(enabled) }

// GraphemeClusters reports whether grapheme-cluster segmentation is on.
func GraphemeClusters() bool { return grapheme.Clusters() }

// IsCluster reports whether a Content value is a cluster handle.
func IsCluster(r rune) bool { return r >= RuneClusterBase }

// NextCluster returns the first grapheme cluster of s, its width in cells, and
// the rest of s. In code-point mode it returns the first code point.
func NextCluster(s string) (cluster string, width int, rest string) {
	if !grapheme.Clusters() {
		if s == "" {
			return "", 0, ""
		}
		r, size := utf8.DecodeRuneInString(s)
		return s[:size], RuneWidth(r), s[size:]
	}
	return grapheme.Next(s)
}

// ClusterContent returns the Content value for a cluster: the code point
// itself when the cluster is one, or an interned handle otherwise.
//
// Looking up a cluster that has been seen before does not allocate; the first
// sight of a new cluster copies its text once.
func ClusterContent(cluster string, width int) rune {
	r, size := utf8.DecodeRuneInString(cluster)
	if size == len(cluster) {
		return r
	}

	clusters.mu.RLock()
	handle, ok := clusters.byText[cluster]
	clusters.mu.RUnlock()
	if ok {
		return handle
	}

	clusters.mu.Lock()
	defer clusters.mu.Unlock()
	if handle, ok := clusters.byText[cluster]; ok {
		return handle
	}
	if len(clusters.text) >= maxClusters {
		return r
	}
	// Cloned so the table never pins the caller's backing array, which may be
	// a large document the cluster was sliced from.
	text := string([]byte(cluster))
	handle = RuneClusterBase + rune(len(clusters.text))
	clusters.byText[text] = handle
	clusters.text = append(clusters.text, text)
	clusters.width = append(clusters.width, uint8(width))
	clusters.alone = append(clusters.alone, fallbackFor(text, width))
	return handle
}

// fallbackFor is what ClusterFallback answers for a cluster of width
// columns: its first code point when a terminal that adds up code points
// would draw the joined emoji wider than that, 0 when it would not.
//
// Only joined emoji are replaced — a ZWJ sequence or a skin tone — which
// such a terminal draws as several pictures side by side. A flag, a keycap,
// an accent or VS16 is drawn as it is: they are either measured alike or
// drawn narrower, which the diff's re-anchoring covers.
func fallbackFor(text string, width int) rune {
	sum, joined := 0, false
	for _, r := range text {
		sum += grapheme.RuneWidth(r)
		joined = joined || r == 0x200D || (r >= 0x1F3FB && r <= 0x1F3FF)
	}
	if !joined || sum <= width {
		return 0
	}
	first, _ := utf8.DecodeRuneInString(text)
	return first
}

// ClusterFallback is the code point to draw for cluster r on a terminal
// that draws a cluster code point by one rather than as one character, or 0
// when it draws the cluster in its width all the same.
//
// Such a terminal draws an emoji with a skin tone as the emoji and a
// coloured square, four columns where the buffer gave two: the square
// spills over the next cells, and stays there when they do not change.
// Found as a square after a playlist's 💪🏼 in Alacritty 0.16.
// The first code point keeps the picture, without the tone or the joined
// family; the diff pads it with blanks to the cluster's width.
func ClusterFallback(r rune) rune {
	if !IsCluster(r) {
		return 0
	}
	clusters.mu.RLock()
	defer clusters.mu.RUnlock()
	if i := int(r - RuneClusterBase); i < len(clusters.alone) {
		return clusters.alone[i]
	}
	return 0
}

// AppendDegraded appends cluster r as a terminal without grapheme clusters
// should be sent it: its fallback code point padded with blanks to its
// width, or the cluster itself when it needs none.
func AppendDegraded(out []byte, r rune) []byte {
	fb := ClusterFallback(r)
	if fb == 0 {
		return AppendContent(out, r)
	}
	out = utf8.AppendRune(out, fb)
	for pad := RuneWidth(r) - grapheme.RuneWidth(fb); pad > 0; pad-- {
		out = append(out, ' ')
	}
	return out
}

// ClusterText returns the text a Content value stands for.
func ClusterText(r rune) string {
	if !IsCluster(r) {
		return string(r)
	}
	clusters.mu.RLock()
	defer clusters.mu.RUnlock()
	if i := int(r - RuneClusterBase); i < len(clusters.text) {
		return clusters.text[i]
	}
	return ""
}

// AppendContent appends the UTF-8 text of a Content value to out: the rune,
// or the whole cluster for a handle. It does not allocate beyond growing out.
//
// Kept small enough to inline: the diff calls it once per emitted cell, and an
// out-of-line call per cell cost a full-screen diff a quarter of its speed.
func AppendContent(out []byte, r rune) []byte {
	if uint32(r) < utf8.RuneSelf {
		return append(out, byte(r))
	}
	return appendContentSlow(out, r)
}

func appendContentSlow(out []byte, r rune) []byte {
	if !IsCluster(r) {
		return utf8.AppendRune(out, r)
	}
	clusters.mu.RLock()
	defer clusters.mu.RUnlock()
	if i := int(r - RuneClusterBase); i < len(clusters.text) {
		return append(out, clusters.text[i]...)
	}
	return out
}

func clusterWidth(r rune) int {
	clusters.mu.RLock()
	defer clusters.mu.RUnlock()
	if i := int(r - RuneClusterBase); i < len(clusters.width) {
		return int(clusters.width[i])
	}
	return 0
}
