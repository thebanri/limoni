package cell

import "sync"

// A hyperlink is a property of a cell, not of a span of text: the diff writes
// cells in whatever order it finds them, so the URL has to travel with each
// one. Storing the string itself would put a pointer in every Cell and end
// the flat-grid, 16-byte-cell design, so a URL is interned once and the cell
// keeps a 16-bit handle — which fits in the two bytes Style was padding with
// anyway, and costs no memory at all.
//
// The table is append-only and shared by every buffer, because a front and a
// back buffer compare styles directly: a handle has to mean the same URL in
// both. It is the same arrangement as the grapheme cluster table.

// LinkID identifies a hyperlink target. The zero value means no link, which
// is what nearly every cell holds.
type LinkID uint16

// maxLinks is what a 16-bit handle allows, minus the zero value. A screen has
// far fewer distinct URLs than that; the limit exists so that a stream of
// unique URLs cannot grow the process without bound. Past it, text is drawn
// without its link rather than with the wrong one.
const maxLinks = 1<<16 - 1

type linkTable struct {
	mu    sync.RWMutex
	byURL map[string]LinkID
	url   []string
}

var links = linkTable{byURL: make(map[string]LinkID)}

// Link returns the handle for a URL, interning it on first sight. Looking up
// a URL that has been seen before does not allocate, so a widget may call it
// on the draw path.
//
// An empty URL, or one past the table's capacity, returns 0: no link.
func Link(url string) LinkID {
	if url == "" {
		return 0
	}

	links.mu.RLock()
	id, ok := links.byURL[url]
	links.mu.RUnlock()
	if ok {
		return id
	}

	links.mu.Lock()
	defer links.mu.Unlock()
	if id, ok := links.byURL[url]; ok {
		return id
	}
	if len(links.url) >= maxLinks {
		return 0
	}
	// Cloned so the table never pins the caller's backing array, which may be
	// a whole document the URL was sliced out of.
	u := string([]byte(url))
	id = LinkID(len(links.url) + 1)
	links.byURL[u] = id
	links.url = append(links.url, u)
	return id
}

// LinkURL returns the URL a handle stands for, or "" for the zero handle.
func LinkURL(id LinkID) string {
	if id == 0 {
		return ""
	}
	links.mu.RLock()
	defer links.mu.RUnlock()
	if i := int(id) - 1; i < len(links.url) {
		return links.url[i]
	}
	return ""
}

// WithLink makes the text drawn in this style a hyperlink to url. The link is
// written as OSC 8, which terminals show as a clickable region; terminals
// without hyperlink support are never sent it (see CapabilityProfile).
//
//	f.SetString(x, y, "the changelog", limoni.NewStyle().WithLink(changelogURL).Underline())
//
// An empty url clears the link.
func (s Style) WithLink(url string) Style {
	s.Link = Link(url)
	return s
}

// LinkURL returns the URL this style links to, or "" if it does not link.
func (s Style) LinkURL() string { return LinkURL(s.Link) }
