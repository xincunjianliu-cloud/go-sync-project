package main

import "strings"

// マップのしかけレイヤーに置かれたオブジェクトのtype/textから種類を
// 判定する処理をここに集約する。同じ文字列比較を複数ファイルにバラバラに
// 書くと、命名規則を変えたときに直し漏れが起きやすいため、
// 「このオブジェクトは宝箱か/壁か/レバーか」の判定はすべてここを経由する。

const (
	evTypeEvent    = "event"
	evTypeTrigger  = "trigger"
	evTypeBoss     = "boss"
	evTypeEnemy    = "enemy"
	evTypeDarkness = "darkness"

	evTextWall      = "event_wall"
	evTextLever     = "event_lever"
	evTextBlock     = "event_block"
	evTextBlockSpot = "event_blockspot"
	evTextBlockDoor = "event_blockdoor"

	chestTextPrefix    = "event_chest_"
	chestKeyTextPrefix = "event_chest_key_"
	bossTextPrefix     = "event_boss_"
	storyTextPrefix    = "event_story_"
)

// isBlockingLayer はTiledでクラス「通れないレイヤー」にしたレイヤーか
// (読み込み時に名札 blocking=true に読み替わる)。タイルレイヤーなら描いた
// タイルが、オブジェクトレイヤーなら置いた四角・多角形が通れない。
// 当たり判定(field_update.go / field_scene.go)とミニマップの色分け
// (menu_draw_misc.go)がこれを見る。
func isBlockingLayer(layer TiledLayer) bool { return layerFlag(layer, "blocking") }

// isEventsLayer はクラス「しかけレイヤー」にしたオブジェクトレイヤーか
// (名札 eventlayer=true)。宝箱・ドアなどのしかけはここに置いたものだけが動く。
func isEventsLayer(layer TiledLayer) bool { return layerFlag(layer, "eventlayer") }

// isPlayerLayer はクラス「主人公の高さ」にしたレイヤーか(名札 playerlayer=true)。
// 主人公はこのレイヤーの位置(重なり順)で描かれる。
func isPlayerLayer(layer TiledLayer) bool { return layerFlag(layer, "playerlayer") }

func layerFlag(layer TiledLayer, key string) bool {
	for _, p := range layer.Properties {
		if b, _ := propBool(p.Value); b && strings.EqualFold(p.Name, key) {
			return true
		}
	}
	return false
}

// 会話データ中の演出命令。Speakerがsystemスピーカーの行は台詞ではなく
// Textを命令として解釈する(tools/dialoguegenが生成する文字列と対応)。
const (
	systemSpeaker        = "SYSTEM_COMMAND"
	cmdStopBGM           = "STOP_BGM"
	cmdPlayBGMPrefix     = "PLAY_BGM_"
	cmdStartBattlePrefix = "START_BATTLE_"
	cmdStartEnding       = "START_ENDING"
)

func isKeyChestObj(p map[string]string) bool {
	return p["type"] == evTypeEvent && strings.HasPrefix(p["text"], chestKeyTextPrefix)
}

func isItemChestObj(p map[string]string) bool {
	return p["type"] == evTypeEvent && strings.HasPrefix(p["text"], chestTextPrefix) && !isKeyChestObj(p)
}

func isChestObj(p map[string]string) bool {
	return isKeyChestObj(p) || isItemChestObj(p)
}

// isWallObj は壁オブジェクトか。type=event, text=event_wall のほか、
// type/textを書かずにleverだけを付けたオブジェクトもレバー壁として扱う。
func isWallObj(p map[string]string) bool {
	if p["type"] == "" && p["text"] == "" && p["lever"] != "" {
		return true
	}
	return p["type"] == evTypeEvent && p["text"] == evTextWall
}

// isLeverControlledWallObj はレバーで開閉するタイプの壁(leverプロパティ有り)。
func isLeverControlledWallObj(p map[string]string) bool {
	return isWallObj(p) && p["lever"] != ""
}

// isLockedWallObj は鍵で開けるタイプの壁(leverプロパティ無し)。
func isLockedWallObj(p map[string]string) bool {
	return isWallObj(p) && p["lever"] == ""
}

// isLeverWallVisualOnly はレバーを上げても見た目が変わるだけで、
// 通行可否には影響しない壁(passable="false"指定)。
func isLeverWallVisualOnly(p map[string]string) bool {
	return p["passable"] == "false"
}

func isLeverObj(p map[string]string) bool {
	return p["type"] == evTypeEvent && p["text"] == evTextLever
}

func isBlockObj(p map[string]string) bool {
	return p["type"] == evTypeEvent && p["text"] == evTextBlock
}

func isBlockSpotObj(p map[string]string) bool {
	return p["type"] == evTypeEvent && p["text"] == evTextBlockSpot
}

func isBlockDoorObj(p map[string]string) bool {
	return p["type"] == evTypeEvent && p["text"] == evTextBlockDoor
}
