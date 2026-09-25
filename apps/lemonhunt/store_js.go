//go:build js

package main

import "syscall/js"

// In a browser the name and the leaderboard live in localStorage, under one
// key, so they are there the next time the page is opened — in that browser.
const storageKey = "lemonhunt"

type browserStore struct{}

func newStore() store { return browserStore{} }

// localStorage throws in some private windows and when storage is blocked;
// syscall/js turns that into a panic, which here means "nothing stored".
func (browserStore) load() (name string, board []scoreEntry) {
	defer func() {
		if recover() != nil {
			name, board = "", nil
		}
	}()
	v := js.Global().Get("localStorage").Call("getItem", storageKey)
	if v.Type() != js.TypeString {
		return "", nil
	}
	return decodeSaved([]byte(v.String()))
}

func (browserStore) save(name string, board []scoreEntry) {
	defer func() { _ = recover() }()
	js.Global().Get("localStorage").Call("setItem", storageKey, string(encodeSaved(name, board)))
}
