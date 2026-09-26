//go:build js

package main

import "syscall/js"

// In a browser the name and the game's own board live in localStorage,
// under one key, so they are there the next time the page is opened, in
// that browser. The zero store keeps nothing.
type store struct {
	key string
}

func newStore() store { return store{key: "lemondrop"} }

// localStorage throws in some private windows and when storage is blocked;
// syscall/js turns that into a panic, which here means "nothing stored".
func (s store) load() (name string, board []scoreEntry) {
	if s.key == "" {
		return "", nil
	}
	defer func() {
		if recover() != nil {
			name, board = "", nil
		}
	}()
	v := js.Global().Get("localStorage").Call("getItem", s.key)
	if v.Type() != js.TypeString {
		return "", nil
	}
	return decodeSaved([]byte(v.String()))
}

func (s store) save(name string, board []scoreEntry) {
	if s.key == "" {
		return
	}
	data := encodeSaved(name, board)
	defer func() { _ = recover() }()
	js.Global().Get("localStorage").Call("setItem", s.key, string(data))
}
