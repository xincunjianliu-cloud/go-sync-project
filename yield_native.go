//go:build !js

package main

// yieldToBrowser はネイティブ版では何もしない。goroutineは別スレッドで
// 並行に動くため、描画ループへ明示的に制御を返す必要がない
// (js版の説明はyield_js.go参照)。
func yieldToBrowser() {}
