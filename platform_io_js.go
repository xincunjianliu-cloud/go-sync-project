//go:build js

package main

import (
	"encoding/base64"
	"fmt"
	"syscall/js"
)

func readRuntimeFile(path string) ([]byte, error) {
	v := js.Global().Get("localStorage").Call("getItem", path)
	if v.IsNull() {
		return nil, fmt.Errorf("%s: not found", path)
	}
	data, err := base64.StdEncoding.DecodeString(v.String())
	if err != nil {
		return nil, fmt.Errorf("%s: decode failed: %w", path, err)
	}
	return data, nil
}

func writeRuntimeFile(path string, data []byte) error {
	encoded := base64.StdEncoding.EncodeToString(data)
	js.Global().Get("localStorage").Call("setItem", path, encoded)
	return nil
}

func saveRuntimePNG(path string, pixels []byte, width, height int, done func(error)) {
	pixels = append([]byte(nil), pixels...)
	go func() {
		global := js.Global()
		canvas := global.Get("document").Call("createElement", "canvas")
		canvas.Set("width", width)
		canvas.Set("height", height)
		ctx := canvas.Call("getContext", "2d")
		if !ctx.Truthy() {
			done(fmt.Errorf("2D canvas context unavailable"))
			return
		}

		imageData := ctx.Call("createImageData", width, height)
		js.CopyBytesToJS(imageData.Get("data"), pixels)
		ctx.Call("putImageData", imageData, 0, 0)

		blobReady := make(chan js.Value, 1)
		callback := js.FuncOf(func(this js.Value, args []js.Value) any {
			if len(args) == 0 {
				blobReady <- js.Null()
			} else {
				blobReady <- args[0]
			}
			return nil
		})
		canvas.Call("toBlob", callback, "image/png")
		blob := <-blobReady
		callback.Release()
		canvas.Set("width", 0)
		canvas.Set("height", 0)
		if !blob.Truthy() {
			done(fmt.Errorf("canvas PNG encoding returned no blob"))
			return
		}
		buffer, err := awaitPromise(blob.Call("arrayBuffer"))
		if err != nil {
			done(fmt.Errorf("read encoded thumbnail: %w", err))
			return
		}
		u8 := global.Get("Uint8Array").New(buffer)
		encoded := make([]byte, u8.Get("length").Int())
		js.CopyBytesToGo(encoded, u8)
		done(writeRuntimeFile(path, encoded))
	}()
}

func runtimeFileExists(path string) bool {
	v := js.Global().Get("localStorage").Call("getItem", path)
	return !v.IsNull()
}
