//go:build !js

package main

import (
	"embed"
	"fmt"
)

// ネイティブ版(デスクトップでの動作確認用exeなど)は単一の実行ファイルで
// 完結させたいので、従来どおりassetsをバイナリに埋め込む。
//
//go:embed all:assets
var embeddedAssets embed.FS

func loadAssetBytes(path string) ([]byte, error) {
	data, err := embeddedAssets.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("asset読み込み失敗 %s: %w", path, err)
	}
	return data, nil
}
