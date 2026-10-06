//go:build !js

package main

import (
	"bytes"
	"image"
)

// decodeImageData はPNGなどの画像データをデコードする。
func decodeImageData(data []byte) (image.Image, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	return img, err
}
