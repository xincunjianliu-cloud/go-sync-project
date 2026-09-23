//go:build js

package main

import "time"

// yieldToBrowser はバックグラウンドgoroutineの重い処理(画像・mp3のデコード)の
// 合間に呼び、ブラウザのイベントループへ一旦制御を返す。
// WebAssembly版のGoはシングルスレッドで、goroutineはメインの描画ループと同じ
// スレッドを奪い合う。途中で手放さないと、デコードが終わるまで
// requestAnimationFrameが発火できず、Loading表示ごと画面が固まってしまう。
// time.Sleepはjs/wasmではsetTimeout経由で実装されているため、
// これで確実にイベントループへ戻れる。
func yieldToBrowser() {
	time.Sleep(time.Millisecond)
}
