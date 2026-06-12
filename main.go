// overlap is a when2meet-style scheduling app built with grove (Go + wasm)
// and warchest for the fallible domain logic. Run it with `grove serve`.
package main

import (
	"syscall/js"

	g "github.com/gyoumi/grove"
	"github.com/gyoumi/grove/dom"
	"github.com/gyoumi/overlap/app"
)

const storageKey = "overlap:event"

// localStore persists the event in the browser's localStorage.
type localStore struct{}

func (localStore) Load() string {
	v := js.Global().Get("localStorage").Call("getItem", storageKey)
	if v.IsNull() || v.IsUndefined() {
		return ""
	}
	return v.String()
}

func (localStore) Save(s string) {
	js.Global().Get("localStorage").Call("setItem", storageKey, s)
}

func main() {
	dom.Mount("#root", g.C(app.App, app.Props{Store: localStore{}}))
}
