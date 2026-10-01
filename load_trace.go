package main

import (
	"log"
	"time"
)

// loadTraceStart はプログラム開始時刻。Web版ではゲーム本体(wasm)の
// ダウンロードとコンパイルが終わって、Goが動き始めた時点になる。
var loadTraceStart = time.Now()

// loadTrace は読み込みの節目(タイトル表示、各段階の完了、BGMの準備、
// 暗転中の待ち時間など)を、起動からの経過秒つきでログに出す。Web版は
// ブラウザのコンソール、ネイティブ版は標準エラーに出る。改善の前後比較や、
// 実機で遅い場所を探すときに使う。
func loadTrace(format string, args ...any) {
	elapsed := time.Since(loadTraceStart).Seconds()
	log.Printf("[load %6.2fs] "+format, append([]any{elapsed}, args...)...)
}

// mib はバイト数をMiB単位の小数にする（ログ表示用）。
func mib(n int64) float64 {
	return float64(n) / (1 << 20)
}
