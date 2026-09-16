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

func runtimeFileExists(path string) bool {
	v := js.Global().Get("localStorage").Call("getItem", path)
	return !v.IsNull()
}
