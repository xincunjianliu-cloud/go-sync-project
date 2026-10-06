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
// assetManifest はデプロイ時にindex.htmlへ書き込まれる、素材の相対パス →
// 中身のハッシュの一覧(tools/genswmanifest)。ローカルの確認用サーバーでは
// 書き込まれておらず、オブジェクトにならない。
var assetManifest = js.Global().Get("assetManifest")

// assetURL はpathを取得するときのURLを返す。一覧があれば中身のハッシュを
// 付ける(path?v=ハッシュ)。Service Worker(web/sw.js)は、ハッシュが自分の
// 一覧と同じときだけ保存済みのものを返すので、デプロイ直後に古いService
// Workerが残っていても、古い中身の素材が新しいゲーム本体に渡ることはない。
// ブラウザのHTTPキャッシュも、中身が変わったファイルだけ取り直すようになる。
func assetURL(path string) string {
	if assetManifest.Type() != js.TypeObject {
		return path
	}
	if v := assetManifest.Get(path); v.Type() == js.TypeString {
		return path + "?v=" + v.String()
	}
	return path
}

func loadAssetBytes(path string) ([]byte, error) {
	done := make(chan struct{})
	var result []byte
	var fetchErr error

	onResponse := js.FuncOf(func(this js.Value, args []js.Value) any {
		resp := args[0]
		if !resp.Get("ok").Bool() {
			// 待ちの解除はonBodyで行う(ここで解除すると、続くonBodyが
			// 解放済みの関数を呼ぶことになる)。
			fetchErr = fmt.Errorf("asset読み込み失敗 %s: status %d", path, resp.Get("status").Int())
			return nil
		}
		// 本文の受信が途中で失敗した場合も下のcatchに届くよう、Promiseを返す。
		return resp.Call("arrayBuffer")
	})
	defer onResponse.Release()

	onBody := js.FuncOf(func(this js.Value, args []js.Value) any {
		if fetchErr == nil {
			u8 := js.Global().Get("Uint8Array").New(args[0])
			data := make([]byte, u8.Get("length").Int())
			js.CopyBytesToGo(data, u8)
			result = data
		}
		close(done)
		return nil
	})
	defer onBody.Release()

	catch := js.FuncOf(func(this js.Value, args []js.Value) any {
		fetchErr = fmt.Errorf("asset読み込み失敗 %s: %s", path, args[0].Call("toString").String())
		close(done)
		return nil
	})
	defer catch.Release()

	js.Global().Call("fetch", assetURL(path)).Call("then", onResponse).Call("then", onBody).Call("catch", catch)
	<-done

	if fetchErr != nil {
		return nil, fetchErr
	}
	return result, nil
}
