package main

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// 戦闘背景の決め方:
//
//  1. ボス戦で、bossBattleBgにそのボスの専用背景がある → それを使う
//  2. 戦闘が始まったマップに"battlebg"プロパティがある → それを使う
//  3. どちらもない → battle_bg.png(BattleBgImg)を使う
//
// 背景は「景色の種類」ごとに1枚用意し、似た場所のマップでは同じ名前を
// 指定して使い回す(例: 学校の中のマップはどれも"school")。画像は
// assets/images/battle/bg/<名前>.png に置く。

// bossBattleBg は専用の戦闘背景を持つボス(ボス番号。boss_NのN)と、その
// 背景の名前。話の区切りになる大事なボスだけ書く。書いていないボスは、
// いるマップの背景を使う。
//
//	例: 4: "boss_final",  → assets/images/battle/bg/boss_final.png
var bossBattleBg = map[int]string{}

// mapBattleBgKey はマップ全体のカスタムプロパティ"battlebg"の値を返す。
// (Tiledのマッププロパティで、このマップで起きる戦闘の背景を名前で指定する。
// 例: "school")
func (m TiledMap) mapBattleBgKey() (string, bool) {
	for _, p := range m.Properties {
		if strings.EqualFold(p.Name, "battlebg") {
			if s, ok := p.Value.(string); ok && s != "" {
				return s, true
			}
		}
	}
	return "", false
}

func battleBgImagePath(key string) string {
	return fmt.Sprintf("assets/images/battle/bg/%s.png", key)
}

// bossBattleBgKey はボス(BossImgs等の添字idx)の専用背景の名前を返す。
func bossBattleBgKey(idx int) (string, bool) {
	key, ok := bossBattleBg[idx+1]
	return key, ok && key != ""
}

// battleBgKey はmapPathのマップで起きる、enemyTypeの戦闘の背景の名前を返す
// (無ければ""。battle_bg.pngを使う)。
func battleBgKey(mapPath, enemyType string) string {
	if idx, ok := bossIndexFromEnemyType(enemyType); ok {
		if key, ok := bossBattleBgKey(idx); ok {
			return key
		}
	}
	if tmap, ok := peekTiledMap(mapPath); ok {
		if key, ok := tmap.mapBattleBgKey(); ok {
			return key
		}
	}
	return ""
}

// mapBattleBgKeys はmapPathのマップにいる間に使う背景の名前(ザコ戦と、まだ
// 倒していないボスの専用背景)を返す。
func (g *Game) mapBattleBgKeys(tmap TiledMap) []string {
	var keys []string
	if key, ok := tmap.mapBattleBgKey(); ok {
		keys = append(keys, key)
	}
	for _, idx := range g.mapBossIndices(tmap) {
		if key, ok := bossBattleBgKey(idx); ok {
			keys = append(keys, key)
		}
	}
	return keys
}

// prepareBattleBgs はkeysの背景のうち、まだのものを優先度prioで裏で
// デコードし始め、全部終わって(成功・失敗どちらでも)いるかを返す。
// 失敗した(画像が無い)背景はbattle_bg.pngで代わりに描く。
func (g *Game) prepareBattleBgs(keys []string, prio int) bool {
	ready := true
	for _, key := range keys {
		path := battleBgImagePath(key)
		if g.battleBgImgs[key] != nil || g.asyncImageFailed[path] {
			continue
		}
		ready = false
		if g.asyncImagePending[path] {
			assetStore.request(path, prio)
			continue
		}
		g.decodeImageAsync(path, prio, func(img *ebiten.Image) {
			if g.battleBgImgs == nil {
				g.battleBgImgs = map[string]*ebiten.Image{}
			}
			if g.battleBgImgs[key] == nil {
				g.battleBgImgs[key] = img
			}
		}, func() { loadTrace("戦闘背景を読み込めませんでした: %s", path) })
	}
	return ready
}

// battleBgImage はこの戦闘で描く背景を返す。ふつうはロード地点やボス戦の
// 開始前に読み込み済み。万一まだなら裏で読み込みを始め、それまでは
// battle_bg.pngで描く。
func (s *BattleScene) battleBgImage() *ebiten.Image {
	if key := battleBgKey(s.originMap, s.enemyType); key != "" {
		if img := s.game.battleBgImgs[key]; img != nil {
			return img
		}
		s.game.prepareBattleBgs([]string{key}, prioUrgent)
	}
	return s.game.BattleBgImg
}
