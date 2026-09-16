package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

var (
	assetBytesCache   = map[string][]byte{}
	assetBytesCacheMu sync.Mutex
)

// loadAssetBytesCached はloadAssetBytesの結果をパスごとにキャッシュする。
// Web版は1回のfetchにネットワーク往復が丸ごとかかるため、同じ画像を
// 複数箇所から読み込んでも2回目以降はキャッシュから即座に返す。
func loadAssetBytesCached(path string) ([]byte, error) {
	assetBytesCacheMu.Lock()
	if data, ok := assetBytesCache[path]; ok {
		assetBytesCacheMu.Unlock()
		return data, nil
	}
	assetBytesCacheMu.Unlock()

	data, err := loadAssetBytes(path)
	if err != nil {
		return nil, err
	}

	assetBytesCacheMu.Lock()
	assetBytesCache[path] = data
	assetBytesCacheMu.Unlock()
	return data, nil
}

// prefetchAssetBytes は起動時に必要なアセットのバイト列をまとめて並列に
// 先読みし、キャッシュへ詰めておく。Web版は1件ずつ順番にfetchすると
// ネットワーク往復時間がファイル数だけ積み重なって起動が遅くなるため、
// goroutineで同時にfetchを走らせて待ち時間を重ね合わせる。
// 実際の画像デコード・エラー処理は呼び出し元のloadAssetImageに任せる
// (ここで失敗したパスはキャッシュに入らず、後段で通常どおりリトライ
// されてエラーメッセージが出る)。
func prefetchAssetBytes(paths []string) {
	seen := make(map[string]bool, len(paths))
	var wg sync.WaitGroup
	for _, p := range paths {
		if seen[p] {
			continue
		}
		seen[p] = true

		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			data, err := loadAssetBytes(path)
			if err != nil {
				return
			}
			assetBytesCacheMu.Lock()
			assetBytesCache[path] = data
			assetBytesCacheMu.Unlock()
		}(p)
	}
	wg.Wait()
}

func loadAssetImage(path string) (*ebiten.Image, error) {
	data, err := loadAssetBytesCached(path)
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
	data, err := loadAssetBytesCached(path)
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
