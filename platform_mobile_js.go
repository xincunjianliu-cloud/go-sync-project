//go:build js

package main

import "syscall/js"

func detectMobileMode() bool {
	mm := js.Global().Call("matchMedia", "(pointer: coarse)")
	return mm.Get("matches").Bool()
}
