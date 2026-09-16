//go:build js

package main

import (
	"fmt"
	"syscall/js"
)

// web版はwasm本体にassetsを埋め込まず、index.htmlと同じ場所に置いた
// assetsフォルダをブラウザのfetchで都度取得する。こうすることで
// wasm本体のサイズが縮み、コードだけの変更時にassetsがブラウザキャッシュ
// から再利用され、再ダウンロードされなくなる。
// net/httpパッケージ経由だとcrypto/tls等の未使用コードまで含まれて
// バイナリが太るため、syscall/jsで直接fetch()を呼ぶ。
func loadAssetBytes(path string) ([]byte, error) {
	done := make(chan struct{})
	var result []byte
	var fetchErr error

	then := js.FuncOf(func(this js.Value, args []js.Value) any {
		resp := args[0]
		if !resp.Get("ok").Bool() {
			fetchErr = fmt.Errorf("asset読み込み失敗 %s: status %d", path, resp.Get("status").Int())
			close(done)
			return nil
		}
		resp.Call("arrayBuffer").Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
			buf := args[0]
			u8 := js.Global().Get("Uint8Array").New(buf)
			data := make([]byte, u8.Get("length").Int())
			js.CopyBytesToGo(data, u8)
			result = data
			close(done)
			return nil
		}))
		return nil
	})
	defer then.Release()

	catch := js.FuncOf(func(this js.Value, args []js.Value) any {
		fetchErr = fmt.Errorf("asset読み込み失敗 %s: %s", path, args[0].Call("toString").String())
		close(done)
		return nil
	})
	defer catch.Release()

	js.Global().Call("fetch", path).Call("then", then).Call("catch", catch)
	<-done

	if fetchErr != nil {
		return nil, fetchErr
	}
	return result, nil
}
