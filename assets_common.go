package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
)

// loadAssetBytesCached はpathのバイト列を返す。取得はassetStore経由で行い、
// まだ届いていなければ最優先に繰り上げて待つ。先読み済み(prefetch/request)
// なら手元のバイト列をそのまま返す。
func loadAssetBytesCached(path string) ([]byte, error) {
	return assetStore.get(path)
}

// prefetchAssetBytes はpathsをまとめて最優先で取得し、全部届くまで待つ。
// Web版は1件ずつ順番に取得するとネットワーク往復がファイル数だけ積み重なる
// ため、同時に走らせて待ち時間を重ね合わせる。実際のデコード・エラー処理は
// 呼び出し元に任せる。
func prefetchAssetBytes(paths []string) {
	requestAssets(paths, prioUrgent)
	assetStore.wait(paths)
}

// requestAssets はpathsの取得を優先度prioで依頼し、待たずに戻る。
func requestAssets(paths []string, prio int) {
	for _, p := range paths {
		assetStore.request(p, prio)
	}
}

func loadAssetImage(path string) (*ebiten.Image, error) {
	img, err := decodeAssetImage(path)
	if err != nil {
		return nil, err
	}
	return ebiten.NewImageFromImage(img), nil
}

// decodeAssetImage はloadAssetImageと違い、image.Decodeまでで止めて
// ebiten.Imageは生成しない。ebiten.NewImageFromImage等のグラフィックス系
// APIはメインゴルーチン以外からの呼び出しが保証されていないため、
// バックグラウンドgoroutineでの先読みにはこちらを使う。
// デコードが終わったら元のPNGのバイト列は手放す（画像は呼び出し側が
// ebiten.Imageとして保持するので、同じPNGを再デコードすることはない）。
func decodeAssetImage(path string) (image.Image, error) {
	data, err := loadAssetBytesCached(path)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	assetStore.release(path)
	if err != nil {
		return nil, fmt.Errorf("画像デコード失敗 %s: %w", path, err)
	}
	return img, nil
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
