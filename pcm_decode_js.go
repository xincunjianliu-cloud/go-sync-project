//go:build js

package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"syscall/js"
	"time"
)

// pcmConvertSlice はFloat32→16bit PCM変換の途中でブラウザへ制御を返すまでに
// 連続して処理する時間の目安。
const pcmConvertSlice = 6 * time.Millisecond

// pcmConvertChunkFrames はJS側のFloat32ArrayからGoへ一度にコピーするフレーム数。
const pcmConvertChunkFrames = 32 * 1024

var noopPromiseCatch = js.FuncOf(func(this js.Value, args []js.Value) any { return nil })

// decodeMP3ToPCM はmp3をブラウザ組み込みのデコーダ(Web Audio APIの
// decodeAudioData)で44.1kHz・16bitステレオのPCMへ変換する。
// Go製のmp3デコーダをwasm上で動かすと1曲(約2.5分)あたり十数秒CPUを
// 占有し、その間ゲーム全体が固まっていた。decodeAudioDataはブラウザの
// 別スレッドでネイティブに動くため、数百ms程度で終わりメインスレッドも塞がない。
func decodeMP3ToPCM(data []byte) ([]byte, error) {
	ctor := js.Global().Get("OfflineAudioContext")
	if !ctor.Truthy() {
		ctor = js.Global().Get("webkitOfflineAudioContext")
	}
	if !ctor.Truthy() {
		return decodeMP3ToPCMGo(data)
	}
	// OfflineAudioContextのサンプルレートに合わせてリサンプリングされる。
	ctx := ctor.New(2, 1, sampleRate)

	u8 := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(u8, data)

	done := make(chan struct{})
	var decoded js.Value
	var decodeErr error
	success := js.FuncOf(func(this js.Value, args []js.Value) any {
		decoded = args[0]
		close(done)
		return nil
	})
	defer success.Release()
	failure := js.FuncOf(func(this js.Value, args []js.Value) any {
		msg := "unknown error"
		if len(args) > 0 && args[0].Truthy() {
			msg = args[0].Call("toString").String()
		}
		decodeErr = fmt.Errorf("decodeAudioData失敗: %s", msg)
		close(done)
		return nil
	})
	defer failure.Release()

	// 古いSafariはコールバック形式しか無いのでコールバックで受け取る。
	// Promiseも返すブラウザでは、失敗時の未処理rejection警告を抑えておく。
	p := ctx.Call("decodeAudioData", u8.Get("buffer"), success, failure)
	if p.Truthy() && p.Get("catch").Truthy() {
		p.Call("catch", noopPromiseCatch)
	}
	<-done
	if decodeErr != nil {
		return nil, decodeErr
	}

	frames := decoded.Get("length").Int()
	left := decoded.Call("getChannelData", 0)
	right := left
	if decoded.Get("numberOfChannels").Int() > 1 {
		right = decoded.Call("getChannelData", 1)
	}

	pcm := make([]byte, frames*4)
	lbuf := make([]byte, pcmConvertChunkFrames*4)
	rbuf := make([]byte, pcmConvertChunkFrames*4)
	u8Ctor := js.Global().Get("Uint8Array")
	sliceStart := time.Now()
	for off := 0; off < frames; off += pcmConvertChunkFrames {
		n := min(pcmConvertChunkFrames, frames-off)
		copyFloat32Chunk(u8Ctor, left, off, n, lbuf)
		copyFloat32Chunk(u8Ctor, right, off, n, rbuf)
		out := pcm[off*4 : (off+n)*4]
		for i := 0; i < n; i++ {
			binary.LittleEndian.PutUint16(out[i*4:], uint16(floatToPCM16(lbuf[i*4:])))
			binary.LittleEndian.PutUint16(out[i*4+2:], uint16(floatToPCM16(rbuf[i*4:])))
		}
		if time.Since(sliceStart) >= pcmConvertSlice {
			yieldToBrowser()
			sliceStart = time.Now()
		}
	}
	return pcm, nil
}

// copyFloat32Chunk はFloat32Arrayのoffからnサンプル分を生のバイト列として
// dstへコピーする(Float32ArrayはGoへ直接コピーできないため、同じ
// メモリ領域を指すUint8Arrayを作ってからCopyBytesToGoする)。
func copyFloat32Chunk(u8Ctor, f32 js.Value, off, n int, dst []byte) {
	sub := f32.Call("subarray", off, off+n)
	view := u8Ctor.New(sub.Get("buffer"), sub.Get("byteOffset"), sub.Get("byteLength"))
	js.CopyBytesToGo(dst[:n*4], view)
}

func floatToPCM16(b []byte) int16 {
	f := math.Float32frombits(binary.LittleEndian.Uint32(b))
	if f >= 1 {
		return math.MaxInt16
	}
	if f <= -1 {
		return math.MinInt16 + 1
	}
	return int16(f * math.MaxInt16)
}
