//go:build js

package main

import "syscall/js"

// PCのWeb版はウィンドウ(ブラウザ内表示)が初期値。スマホはこの設定に関係なく
// 起動ごとに最初の操作でフルスクリーンにする（initDisplayMode）。
const defaultFullscreen = false

const isWebBuild = true

// fullscreenSupported はこのブラウザでページのフルスクリーン表示が使えるか。
// iPhoneのSafariなどは動画以外のフルスクリーンに対応していない。
func fullscreenSupported() bool {
	doc := js.Global().Get("document")
	return doc.Get("fullscreenEnabled").Truthy() || doc.Get("webkitFullscreenEnabled").Truthy()
}

// runningAsInstalledApp はホーム画面に追加したアプリ(PWA)として起動して
// いるか。iPhoneは独自のnavigator.standaloneで判定する。
func runningAsInstalledApp() bool {
	w := js.Global()
	if w.Get("navigator").Get("standalone").Truthy() {
		return true
	}
	mm := w.Call("matchMedia", "(display-mode: standalone), (display-mode: fullscreen)")
	return mm.Get("matches").Bool()
}
