package main

import (
	"bytes"
	"io"

	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
)

// decodeMP3ToPCMGo はGo製デコーダでmp3を44.1kHz・16bitステレオのPCMへ変換する。
// ネイティブ版の標準経路であり、Web版ではブラウザのデコーダが使えない場合の
// フォールバック(wasm上では非常に遅い。pcm_decode_js.go参照)。
func decodeMP3ToPCMGo(data []byte) ([]byte, error) {
	d, err := mp3.DecodeWithSampleRate(sampleRate, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(d)
}
