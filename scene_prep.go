package main

import (
	"image"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// 画面切り替えの流れ（すべての遷移で共通）:
//
//  1. 暗転を始めると同時に、次の画面に必要なもの(scenePrep)の準備を始める。
//  2. 暗転しきったら、準備が終わるまで暗いまま待つ。
//  3. 準備ができたら次の画面を組み立て、BGMのデコードを待って画面を明ける。
//
// Loading表示を出すのは「ロード地点」(scenePrep.loadPoint)だけ:
//   - はじめから／つづきから(セーブデータのロード)
//   - 別のマップ(.tmj)へ移るとき
//
// ロード地点では、そのマップで遊び終えるまでに使うもの(戦闘段階までの画像、
// タイルセット、戦闘背景、マップ・戦闘・勝利・ボス戦のBGM)をすべてそろえてから画面を
// 明ける。準備が済んでいても毎回Loadingを出し、loadPointMinTicksは表示して
// 区切りとして見せる。それ以外の切り替え(同じマップ内の移動、戦闘、メニュー
// など)ではLoadingを出さない。ロード地点でそろえてあるので、ふつうは待ちも
// 起きない。万一待ちが起きたら暗転のまま黙って待ち、ルールの抜けとして
// ログに警告を出す(updateSceneLoading)。
//
// さらに「もうすぐ使う」と分かった時点(ドアに近づいた、ボスとの会話が
// 始まった)で準備を先に始めておくので、ロード地点の待ちも短くなる。

// scenePrep は次の画面を見せる前にそろえておくもの。
type scenePrep struct {
	// tier はこの段階(assetTier)までの画像の読み込み完了を待つ。
	tier assetTier
	// mapPath はフィールド画面へ切り替えるときの行き先のマップ。そのマップの
	// タイルセット画像(複数あれば全部)を先にデコードしてtilesetImageCacheへ入れておく。
	mapPath string
	// loadPoint ならロード地点として扱う。Loadingを出し、mapPathのマップで
	// 使うBGM(loadPointBGM)と、そのマップにいるボスの画像もそろえる。
	loadPoint bool
	// bosses はそろえておくボスの画像(BossImgs等の添字)。ボスの画像は
	// 段階読み込みに入れず、ボスごとに必要になった時点で読み込む。
	bosses []int
}

// loadPointMinTicks はロード地点でLoadingを表示し続ける最短のフレーム数
// (0.6秒)。準備が一瞬で終わっても、毎回同じ間で区切りを見せる。
const loadPointMinTicks = 36

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
				if g.asyncImageFailed == nil {
					g.asyncImageFailed = map[string]bool{}
				}
				g.asyncImageFailed[r.path] = true
				if r.onError != nil {
					r.onError()
				}
				continue
			}
			if r.assign != nil {
				r.assign(newImageTraced(r.path, r.img))
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
// ボスの画像は段階に入っていないので、段階の読み込みを待っている間も
// 並行して進める。
func (g *Game) scenePrepReady(prep scenePrep) bool {
	ready := g.prepareBossImages(prep.bosses, prioUrgent)
	if prep.mapPath == "" {
		return g.assetTierReady(prep.tier) && ready
	}
	// マップはフィールド段階の読み込みで全部パース済みになるので、
	// キャッシュだけを見る(メインゴルーチンで取得を待たない)。無ければ
	// NewRoomSceneがその場で読み込み、エラーを出す。
	tmap, ok := peekTiledMap(prep.mapPath)
	if ok && prep.loadPoint && !g.prepareBossImages(g.mapBossIndices(tmap), prioUrgent) {
		ready = false
	}
	if !g.assetTierReady(prep.tier) {
		return false
	}
	if !ok {
		return ready
	}
	for _, p := range mapTilesetImagePaths(tmap) {
		g.prepareTileset(p, prioUrgent)
		if g.asyncImagePending[p] {
			ready = false
		}
	}
	if prep.loadPoint {
		if !g.prepareBattleBgs(g.mapBattleBgKeys(tmap), prioUrgent) {
			ready = false
		}
		bgm := g.loadPointBGM(tmap)
		g.Audio.PinBGM(bgm)
		if !g.prepareBGM(bgm) {
			ready = false
		}
	}
	return ready
}

// loadPointBGM はロード地点でそろえておく曲を返す。そのマップの曲と、
// そのマップで起きる戦闘(ザコ戦・勝利・まだ倒していないボス戦)の曲。
func (g *Game) loadPointBGM(tmap TiledMap) []string {
	paths := []string{bgmBattleNormal, bgmVictoryIntro, bgmVictoryLoop}
	if p, ok := mapBGMPath(tmap); ok {
		paths = append(paths, p)
	}
	return append(paths, g.mapBossBGMs(tmap)...)
}

// prepareBGM はpathsの曲のうちデコードがまだのものを始め、全部終わって
// (成功・失敗どちらでも)いるかを返す。
func (g *Game) prepareBGM(paths []string) bool {
	if g.Audio == nil {
		return true
	}
	ready := true
	for _, p := range paths {
		if g.Audio.pcmSettled(p) {
			continue
		}
		g.Audio.Prewarm(prioUrgent, p)
		ready = false
	}
	return ready
}

// prepareTileset はタイルセット画像がまだキャッシュに無ければ、バックグラウンドで
// デコードしてキャッシュへ入れる。失敗した場合はキャッシュに入らず(頼み直しも
// しない)、NewRoomSceneがその場で読み込み直してエラーを出す。デコード中に、
// より高い優先度で頼み直すと取得の順番が繰り上がる。
func (g *Game) prepareTileset(path string, prio int) {
	if path == "" {
		return
	}
	if _, ok := tilesetImageCache[path]; ok || g.asyncImageFailed[path] {
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

// ChangeSceneToMap は同じマップの中でフィールド画面を切り替える(ロード地点
// ではないのでLoadingは出さない)。フィールド段階の読み込みと、そのマップの
// タイルセット画像がまだなら、暗転中にそろえてから組み立てる。
func (g *Game) ChangeSceneToMap(mapPath string, build func() Scene, durationSeconds float64) {
	g.ChangeScenePrepared(scenePrep{tier: assetTierField, mapPath: mapPath}, build, durationSeconds)
}

// ChangeSceneAtLoadPoint はロード地点(はじめから・つづきから・別のマップへの
// 移動)としてマップ(mapPath)のフィールド画面へ切り替える。Loadingを出し、
// そのマップで使う画像と曲をすべてそろえてから画面を明ける。
func (g *Game) ChangeSceneAtLoadPoint(mapPath string, build func() Scene, durationSeconds float64) {
	g.ChangeScenePrepared(scenePrep{tier: assetTierBattle, mapPath: mapPath, loadPoint: true}, build, durationSeconds)
}

// prepareMap はマップに行きそうだと分かった時点(ドアに近づいたなど)で呼び、
// そのマップのタイルセット・BGM・ボスの画像・戦闘背景を先に準備しておく
// (ボス戦の曲は取得だけ)。実際に移動するときのロード地点の待ちが短くなる。
func (g *Game) prepareMap(mapPath string) {
	tmap, ok := peekTiledMap(mapPath)
	if !ok {
		return
	}
	for _, p := range mapTilesetImagePaths(tmap) {
		g.prepareTileset(p, prioSoon)
	}
	if bgm, ok := mapBGMPath(tmap); ok {
		g.Audio.Prewarm(prioSoon, bgm)
	}
	g.prepareBossImages(g.mapBossIndices(tmap), prioSoon)
	g.prepareBattleBgs(g.mapBattleBgKeys(tmap), prioSoon)
	g.Audio.PrefetchBGM(g.mapBossBGMs(tmap)...)
}

// ChangeSceneToBossBattle はボス戦(evTypeは"boss_N")へ切り替える。戦闘段階の
// 画像とそのボスの画像がそろってから組み立てる(ふつうはロード地点か会話の
// 開始時点でそろっている)。
func (g *Game) ChangeSceneToBossBattle(evType string, build func() Scene, durationSeconds float64) {
	prep := scenePrep{tier: assetTierBattle}
	if idx, ok := bossIndexFromEnemyType(evType); ok {
		prep.bosses = []int{idx}
	}
	g.ChangeScenePrepared(prep, build, durationSeconds)
}
