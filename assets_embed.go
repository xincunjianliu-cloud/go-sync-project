package main

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed all:assets
var embeddedAssets embed.FS

func loadAssetBytes(path string) ([]byte, error) {
	data, err := embeddedAssets.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("asset読み込み失敗 %s: %w", path, err)
	}
	return data, nil
}

func loadAssetImage(path string) (*ebiten.Image, error) {
	data, err := loadAssetBytes(path)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("画像デコード失敗 %s: %w", path, err)
	}
	return ebiten.NewImageFromImage(img), nil
}

func loadAssetReader(path string) (*bytes.Reader, error) {
	data, err := loadAssetBytes(path)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

func loadRuntimeImage(path string) (*ebiten.Image, error) {
	data, err := readRuntimeFile(path)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("画像デコード失敗 %s: %w", path, err)
	}
	return ebiten.NewImageFromImage(img), nil
}
