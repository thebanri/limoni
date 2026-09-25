//go:build js

package main

import (
	"errors"
	"fmt"
	"syscall/js"
)

// In a browser the request goes through fetch rather than net/http, which
// would add most of a megabyte to the module for the same call. The page
// allows it: the server names the playground's origin in its CORS headers.
func httpDo(method, url string, body []byte) ([]byte, error) {
	opts := js.Global().Get("Object").New()
	opts.Set("method", method)
	if body != nil {
		headers := js.Global().Get("Object").New()
		headers.Set("Content-Type", "application/json")
		opts.Set("headers", headers)
		opts.Set("body", string(body))
	}
	done := make(chan struct{})
	var (
		data   []byte
		err    error
		status int
	)
	onResponse := js.FuncOf(func(_ js.Value, a []js.Value) any {
		status = a[0].Get("status").Int()
		return a[0].Call("text")
	})
	onText := js.FuncOf(func(_ js.Value, a []js.Value) any {
		data = []byte(a[0].String())
		close(done)
		return nil
	})
	onError := js.FuncOf(func(_ js.Value, a []js.Value) any {
		err = errors.New(a[0].Call("toString").String())
		close(done)
		return nil
	})
	defer onResponse.Release()
	defer onText.Release()
	defer onError.Release()
	js.Global().Call("fetch", url, opts).Call("then", onResponse).Call("then", onText).Call("catch", onError)
	<-done
	if err == nil && status/100 != 2 {
		err = fmt.Errorf("the scoreboard answered %d", status)
	}
	return data, err
}

// boardURL is the shared leaderboard's address, which the page leaves at
// window.__lemonhunt_scoreboard; the flag has no say in a browser.
func boardURL(string) string {
	v := js.Global().Get("__lemonhunt_scoreboard")
	if v.Type() != js.TypeString {
		return ""
	}
	return v.String()
}
