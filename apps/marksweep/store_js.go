//go:build js

package main

import "syscall/js"

const storageKey = "marksweep"

type browserStore struct{}

func newStore() store { return browserStore{} }

// localStorage throws in some private windows and when storage is blocked;
// syscall/js turns that into a panic, which here means "nothing stored".
func (browserStore) load() (r records) {
	defer func() {
		if recover() != nil {
			r = records{volume: -1}
		}
	}()
	v := js.Global().Get("localStorage").Call("getItem", storageKey)
	if v.Type() != js.TypeString {
		return records{volume: -1}
	}
	return decodeRecords([]byte(v.String()))
}

func (browserStore) save(r records) {
	defer func() { _ = recover() }()
	js.Global().Get("localStorage").Call("setItem", storageKey, string(encodeRecords(r)))
}
