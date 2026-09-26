//go:build js

package main

import (
	"strconv"
	"strings"
	"syscall/js"
)

// In a browser the best score lives in localStorage, so it is there the next
// time the page is opened, in that browser. The zero store keeps nothing.
type store struct {
	key string
}

func newStore() store { return store{key: "lemondrop-best"} }

// localStorage throws in some private windows and when storage is blocked;
// syscall/js turns that into a panic, which here means "nothing stored".
func (s store) load() (best int) {
	if s.key == "" {
		return 0
	}
	defer func() {
		if recover() != nil {
			best = 0
		}
	}()
	v := js.Global().Get("localStorage").Call("getItem", s.key)
	if v.Type() != js.TypeString {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(v.String()))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func (s store) save(best int) {
	if s.key == "" {
		return
	}
	defer func() { _ = recover() }()
	js.Global().Get("localStorage").Call("setItem", s.key, strconv.Itoa(best))
}
