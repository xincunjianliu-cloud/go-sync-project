//go:build !js

package main

import "os"

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
)

func readRuntimeFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func writeRuntimeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

func saveRuntimePNG(path string, pixels []byte, width, height int, done func(error)) {
	img := &image.NRGBA{
		Pix:    pixels,
		Stride: 4 * width,
		Rect:   image.Rect(0, 0, width, height),
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		done(fmt.Errorf("PNG encode: %w", err))
		return
	}
	done(writeRuntimeFile(path, buf.Bytes()))
}

func runtimeFileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
