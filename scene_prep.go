package main

import (
	"image"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// 画面切り替えの流れ（すべての遷移で共通）:
//
//  1. 暗転を始めると同時に、次の画面に必要なもの(scenePrep)の準備を始める。
//  2. 暗転しきったら、準備が終わるまで暗いまま待つ。待ちが
//     loadingIndicatorDelayTicksより長引いたときだけLoading表示を出す
//     （一瞬で終わる待ちでLoadingがちらつかないように）。
//  3. 準備ができたら次の画面を組み立て、BGMのデコードを待って画面を明ける。
//
// 準備が最初から済んでいれば、ただのフェード切り替えと同じ速さで終わる。
// さらに「もうすぐ使う」と分かった時点(ドアに近づいた、ボスとの会話が
// 始まった)で同じ準備を先に始めておくので、多くの場合は待ちが発生しない。

// scenePrep は次の画面を見せる前にそろえておくもの。
type scenePrep struct {
	// tier はこの段階(assetTier)までの画像の読み込み完了を待つ。
	tier assetTier
	// mapPath はフィールド画面へ切り替えるときの行き先のマップ。そのマップの
	// タイルセット画像を先にデコードしてtilesetImageCacheへ入れておく。
	mapPath string
}

// loadingIndicatorDelayTicks は暗転したまま待つ時間がこれを超えたら
// Loading表示を出すまでのフレーム数（0.25秒）。
const loadingIndicatorDelayTicks = 15

// asyncImageResult はバックグラウンドでデコードした画像を、メインゴルーチンの
// pumpAsyncImagesへ渡すためのメッセージ。
type asyncImageResult struct {
	path    string
	img     image.Image
	err     error
	assign  func(*ebiten.Image)
	onError func()
}

// decodeImageAsync はpathの画像の取得とデコードをバックグラウンドで行い、
// 終わったらメインゴルーチンでassignを呼ぶ(失敗したらonError)。同じpathを
// デコード中なら何もしない。ebiten.Imageの生成(GPUテクスチャ確保)は
// メインゴルーチンでしか行えないため、pumpAsyncImagesで行う。
func (g *Game) decodeImageAsync(path string, prio int, assign func(*ebiten.Image), onError func()) {
	if g.asyncImagePending[path] {
		return
	}
	if g.asyncImagePending == nil {
		g.asyncImagePending = map[string]bool{}
		g.asyncImageDone = make(chan asyncImageResult, 16)
	}
	g.asyncImagePending[path] = true
	assetStore.request(path, prio)
	done := g.asyncImageDone
	go func() {
		img, err := decodeAssetImage(path)
		done <- asyncImageResult{path: path, img: img, err: err, assign: assign, onError: onError}
	}()
}

// asyncPumpFrameBudget は1フレームのうちpumpAsyncImagesがGPUテクスチャ生成に
// 使ってよい時間。
const asyncPumpFrameBudget = 4 * time.Millisecond

// pumpAsyncImages はUpdate()から毎フレーム呼ばれ、decodeImageAsyncの結果を
// 反映する。
func (g *Game) pumpAsyncImages() {
	if g.asyncImageDone == nil {
		return
	}
	start := time.Now()
	for time.Since(start) < asyncPumpFrameBudget {
		select {
		case r := <-g.asyncImageDone:
			delete(g.asyncImagePending, r.path)
			if r.err != nil {
				if r.onError != nil {
					r.onError()
				}
				continue
			}
			if r.assign != nil {
				r.assign(ebiten.NewImageFromImage(r.img))
			}
		default:
			return
		}
	}
}

// startScenePrep はprepの準備を始める(すぐに始められるものだけ)。
func (g *Game) startScenePrep(prep scenePrep) {
	g.scenePrepReady(prep)
}

// scenePrepReady はprepの準備が終わっているかを返す。まだ始めていない準備が
// あれば(段階の読み込みが終わってマップが分かった時点など)ここで始める。
func (g *Game) scenePrepReady(prep scenePrep) bool {
	if !g.assetTierReady(prep.tier) {
		return false
	}
	if prep.mapPath == "" {
		return true
	}
	// マップはフィールド段階の読み込みで全部パース済みになるので、
	// キャッシュだけを見る(メインゴルーチンで取得を待たない)。無ければ
	// NewRoomSceneがその場で読み込み、エラーを出す。
	tmap, ok := peekTiledMap(prep.mapPath)
	if !ok {
		return true
	}
	p, ok := mapTilesetImagePath(tmap)
	if !ok {
		return true
	}
	g.prepareTileset(p, prioUrgent)
	return !g.asyncImagePending[p]
}

// prepareTileset はタイルセット画像がまだキャッシュに無ければ、バックグラウンドで
// デコードしてキャッシュへ入れる。失敗した場合はキャッシュに入らず、
// NewRoomSceneがその場で読み込み直してエラーを出す。デコード中に、より高い
// 優先度で頼み直すと取得の順番が繰り上がる。
func (g *Game) prepareTileset(path string, prio int) {
	if path == "" {
		return
	}
	if _, ok := tilesetImageCache[path]; ok {
		return
	}
	if g.asyncImagePending[path] {
		assetStore.request(path, prio)
		return
	}
	g.decodeImageAsync(path, prio, func(img *ebiten.Image) {
		if _, ok := tilesetImageCache[path]; !ok {
			tilesetImageCache[path] = img
		}
	}, nil)
}

// ChangeSceneToMap はマップ(mapPath)のフィールド画面へ切り替える
// (ドア移動・ロード・ニューゲーム)。フィールド段階の読み込みと、そのマップの
// タイルセット画像がまだなら、暗転中にそろえてから組み立てる。
func (g *Game) ChangeSceneToMap(mapPath string, build func() Scene, durationSeconds float64) {
	g.ChangeScenePrepared(scenePrep{tier: assetTierField, mapPath: mapPath}, build, durationSeconds)
}

// prepareMap はマップに行きそうだと分かった時点(ドアに近づいたなど)で呼び、
// そのマップのタイルセットとBGMを先に準備しておく。実際に移動するときには
// 準備が済んでいるので、暗転が延びない。
func (g *Game) prepareMap(mapPath string) {
	tmap, ok := peekTiledMap(mapPath)
	if !ok {
		return
	}
	if p, ok := mapTilesetImagePath(tmap); ok {
		g.prepareTileset(p, prioSoon)
	}
	if bgm, ok := mapBGMPath(tmap); ok {
		g.Audio.Prewarm(prioSoon, bgm)
	}
}
