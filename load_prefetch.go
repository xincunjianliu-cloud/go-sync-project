package main

import (
	"fmt"
	"strings"
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
			if fmt.Sprintf("boss_%d", bossID) == lastBossEventType {
				paths = append(paths, bgmBattleLastBoss)
			} else {
				paths = append(paths, bgmBattleBoss)
			}
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
// 背景画像と、会話中に切り替えるBGMを先に取得しておく。立ち絵は会話中に
// 初めて表示する瞬間に読み込まれる(GetCharaImage)ので、先に届いていれば
// その瞬間の待ちがなくなる。
func (g *Game) prefetchDialogueAssets(bd BossDialogue) {
	seen := map[string]bool{}
	var paths []string
	var bgm []string
	for _, cmd := range bd.Commands {
		if cmd.Speaker == systemSpeaker {
			if strings.HasPrefix(cmd.Text, cmdPlayBGMPrefix) {
				if p, ok := resolveBGMKey(strings.TrimPrefix(cmd.Text, cmdPlayBGMPrefix)); ok {
					bgm = append(bgm, p)
				}
			}
			continue
		}
		if cmd.Speaker == "" || seen[cmd.Speaker] {
			continue
		}
		seen[cmd.Speaker] = true
		sheetPath, basePath := g.charaAssetPaths(cmd.Speaker)
		if g.needsCharaAsset(cmd.Speaker + "#sheet") {
			paths = append(paths, sheetPath)
		}
		if g.needsCharaAsset(cmd.Speaker) {
			paths = append(paths, basePath)
		}
	}
	if bd.Background != "" && g.bgImgs[bd.Background] == nil && !g.bgImgMissing[bd.Background] {
		paths = append(paths, backgroundImagePath(bd.Background))
	}
	requestAssets(paths, prioSoon)
	if g.Audio != nil {
		g.Audio.PrefetchBGM(bgm...)
	}
}

// needsCharaAsset はlookupCharaAssetのキーkeyの画像がまだ読み込まれておらず、
// 存在しないと分かってもいないかを返す。
func (g *Game) needsCharaAsset(key string) bool {
	if _, ok := g.CharaImgs[key]; ok {
		return false
	}
	return !g.charaImgMissing[key]
}
