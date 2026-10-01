package main

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// 先読み(prefetch)は「もうすぐ使いそう」と分かった時点で、取得だけを
// 低めの優先度で始めておく仕組み。実際に使う瞬間には手元にあるので、
// ネットワーク待ちの暗転や、会話中の一瞬の固まりが起きにくくなる。
// 対象を決めるのはファイル名ではなくゲームの状況(隣のマップ、これから
// 始まる会話)なので、素材を差し替えても作り直しはいらない。

// prefetchAroundMap はマップに入るときに呼ばれ、このマップから行ける
// 隣のマップのBGMと、このマップにいるボスとの戦闘BGMのmp3を先に取得しておく。
// （デコードは実際に流すときに行う。mp3のままならメモリをあまり使わない）
func (g *Game) prefetchAroundMap(mapPath string, tmap TiledMap) {
	if g.Audio == nil {
		return
	}
	var paths []string
	for _, door := range mapConnectionGraph[mapPath] {
		next, err := loadTiledMap(door.ToMap)
		if err != nil {
			continue
		}
		if p, ok := mapBGMPath(next); ok {
			paths = append(paths, p)
		}
	}

	for _, layer := range tmap.Layers {
		if !strings.HasPrefix(layer.Name, "events") {
			continue
		}
		for _, obj := range layer.Objects {
			p := objProps(obj)
			bossID, hasBossID := objPropInt(obj, "bossid")
			if p["type"] != "boss" || !hasBossID || isBossDefeated(g, bossID) {
				continue
			}
			paths = append(paths, bossBattleBGM(fmt.Sprintf("boss_%d", bossID)))
		}
	}
	g.Audio.PrefetchBGM(paths...)
}

// mapBGMPath はマップで流すBGMのパスを返す(NewRoomSceneと同じ決め方)。
func mapBGMPath(tmap TiledMap) (string, bool) {
	if key, ok := tmap.mapBGMKey(); ok {
		if p, found := resolveBGMKey(key); found {
			return p, true
		}
	}
	return bgmField1, true
}

// prefetchDialogueAssets は会話が始まるときに呼ばれ、登場する話者の立ち絵・
// 背景画像を裏でデコードし、会話中に切り替えるBGMを取得しておく。会話の
// 最後にボス戦が始まる場合は、ボス戦の曲のデコードも始めておく(戦闘開始の
// 暗転が延びないように)。立ち絵は会話中に初めて表示する瞬間に必要になる
// (GetCharaImage)ので、先に用意できていればその瞬間の引っかかりがなくなる。
func (g *Game) prefetchDialogueAssets(bd BossDialogue) {
	seen := map[string]bool{}
	var bgm []string
	for _, cmd := range bd.Commands {
		if cmd.Speaker == systemSpeaker {
			switch {
			case strings.HasPrefix(cmd.Text, cmdPlayBGMPrefix):
				if p, ok := resolveBGMKey(strings.TrimPrefix(cmd.Text, cmdPlayBGMPrefix)); ok {
					bgm = append(bgm, p)
				}
			case strings.HasPrefix(cmd.Text, cmdStartBattlePrefix):
				g.Audio.Prewarm(prioSoon, bossBattleBGM(strings.TrimPrefix(cmd.Text, cmdStartBattlePrefix)))
			}
			continue
		}
		if cmd.Speaker == "" || seen[cmd.Speaker] {
			continue
		}
		seen[cmd.Speaker] = true
		g.prepareSpeaker(cmd.Speaker)
	}
	if key := bd.Background; key != "" && g.bgImgs[key] == nil && !g.bgImgMissing[key] {
		g.decodeImageAsync(backgroundImagePath(key), prioSoon, func(img *ebiten.Image) {
			if _, ok := g.bgImgs[key]; !ok {
				g.bgImgs[key] = img
			}
		}, func() { g.bgImgMissing[key] = true })
	}
	g.Audio.PrefetchBGM(bgm...)
}

// prepareSpeaker は話者の立ち絵がまだ無ければ裏でデコードしてCharaImgsへ
// 入れる。GetCharaImageと同じく表情差分シートを優先し、シートが無いと
// 分かったときだけ単一の立ち絵を用意する。ファイルが無ければ無いことを
// 記録する(lookupCharaAssetと同じ扱い)。
func (g *Game) prepareSpeaker(speaker string) {
	sheetPath, basePath := g.charaAssetPaths(speaker)
	sheetKey := speaker + "#sheet"
	prepareBase := func() {
		if !g.needsCharaAsset(speaker) {
			return
		}
		g.decodeImageAsync(basePath, prioSoon, func(img *ebiten.Image) {
			if _, ok := g.CharaImgs[speaker]; !ok {
				g.CharaImgs[speaker] = img
			}
		}, func() { g.charaImgMissing[speaker] = true })
	}
	if _, ok := g.CharaImgs[sheetKey]; ok {
		return
	}
	if g.charaImgMissing[sheetKey] {
		prepareBase()
		return
	}
	g.decodeImageAsync(sheetPath, prioSoon, func(img *ebiten.Image) {
		if _, ok := g.CharaImgs[sheetKey]; !ok {
			g.CharaImgs[sheetKey] = img
		}
	}, func() {
		g.charaImgMissing[sheetKey] = true
		prepareBase()
	})
}

// bossBattleBGM はボス戦(evTypeは"boss_N")の曲を返す(NewBattleSceneと同じ決め方)。
func bossBattleBGM(evType string) string {
	if evType == lastBossEventType {
		return bgmBattleLastBoss
	}
	return bgmBattleBoss
}

// needsCharaAsset はlookupCharaAssetのキーkeyの画像がまだ読み込まれておらず、
// 存在しないと分かってもいないかを返す。
func (g *Game) needsCharaAsset(key string) bool {
	if _, ok := g.CharaImgs[key]; ok {
		return false
	}
	return !g.charaImgMissing[key]
}
