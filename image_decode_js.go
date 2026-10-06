//go:build js

package main

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"sync"
	"syscall/js"
)

var errBrowserImageDecodeUnavailable = errors.New("ブラウザの画像デコードが使えない")

var pngSignature = []byte("\x89PNG\r\n\x1a\n")

// decodeImageData はPNGなどの画像データをデコードする。Web版はブラウザ
// 組み込みのデコーダ(createImageBitmap)を使う。Go製のデコーダをwasm上で
// 動かすと、1スレッドしかないwasmがデコードの間ずっと塞がり、裏で読み込んで
// いる最中のタイトル画面やフィールドがカクつく(mp3で起きていたのと同じ)。
// ブラウザのデコーダはネイティブで動き、多くのブラウザでは別スレッドで動く。
// 使えないブラウザや、ブラウザ側で失敗した場合はGo製のデコーダで読む。
func decodeImageData(data []byte) (image.Image, error) {
	img, err := decodeImageInBrowser(data)
	if err == nil {
		return img, nil
	}
	browserImageFallbackOnce.Do(func() {
		loadTrace("画像をGo製のデコーダで読みます(以後この警告は出しません): %v", err)
	})
	img, _, err = image.Decode(bytes.NewReader(data))
	return img, err
}

var browserImageFallbackOnce sync.Once

func decodeImageInBrowser(data []byte) (img image.Image, err error) {
	// 古いブラウザでオプション指定が例外になる場合などに備える。
	defer func() {
		if r := recover(); r != nil {
			img, err = nil, fmt.Errorf("ブラウザでの画像デコード失敗: %v", r)
		}
	}()

	global := js.Global()
	if !global.Get("createImageBitmap").Truthy() {
		return nil, errBrowserImageDecodeUnavailable
	}

	u8 := global.Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(u8, data)
	blobOpts := map[string]any{}
	if bytes.HasPrefix(data, pngSignature) {
		blobOpts["type"] = "image/png"
	}
	blob := global.Get("Blob").New([]any{u8}, blobOpts)

	// Go製のデコーダと同じ結果になるよう、色空間の変換(PNGのgAMA・ICCの適用)と
	// アルファの乗算をしないで取り出す。
	bmp, err := awaitPromise(global.Call("createImageBitmap", blob, map[string]any{
		"premultiplyAlpha":     "none",
		"colorSpaceConversion": "none",
	}))
	if err != nil {
		return nil, err
	}
	defer bmp.Call("close")
	w, h := bmp.Get("width").Int(), bmp.Get("height").Int()

	var canvas js.Value
	if oc := global.Get("OffscreenCanvas"); oc.Truthy() {
		canvas = oc.New(w, h)
	} else {
		canvas = global.Get("document").Call("createElement", "canvas")
		canvas.Set("width", w)
		canvas.Set("height", h)
	}
	// 読み出し専用なので、GPUではなくCPU側に置かせる(読み出しが速い)。
	ctx := canvas.Call("getContext", "2d", map[string]any{"willReadFrequently": true})
	if !ctx.Truthy() {
		return nil, errBrowserImageDecodeUnavailable
	}
	ctx.Call("drawImage", bmp, 0, 0)
	pix := ctx.Call("getImageData", 0, 0, w, h).Get("data")

	// getImageDataはアルファを乗算していないRGBAを返す。
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	js.CopyBytesToGo(out.Pix, pix)

	// キャンバスのメモリを早めに手放す。
	canvas.Set("width", 0)
	canvas.Set("height", 0)
	return out, nil
}

// awaitPromise はPromiseが決着するまで待ち、結果を返す。
func awaitPromise(p js.Value) (js.Value, error) {
	done := make(chan struct{})
	var result js.Value
	var err error
	onResolve := js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) > 0 {
			result = args[0]
		}
		close(done)
		return nil
	})
	defer onResolve.Release()
	onReject := js.FuncOf(func(this js.Value, args []js.Value) any {
		msg := "unknown error"
		if len(args) > 0 && args[0].Truthy() {
			msg = args[0].Call("toString").String()
		}
		err = fmt.Errorf("ブラウザでの画像デコード失敗: %s", msg)
		close(done)
		return nil
	})
	defer onReject.Release()
	p.Call("then", onResolve, onReject)
	<-done
	return result, err
}
