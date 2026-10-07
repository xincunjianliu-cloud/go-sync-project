package main

import "strings"

// マップの"events"レイヤーに置かれたオブジェクトのtype/textから種類を
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

	// wallTileLayerName/floorTileLayerNameはタイルレイヤー名の規約。
	// 当たり判定(field_update.go)とミニマップの色分け(menu_draw_misc.go)の
	// 両方がこれを見るので、名前を変えるときはここだけ直せばよい。
	wallTileLayerName  = "kabe"
	floorTileLayerName = "yuka"
)

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

// leverWallImageKey は壁オブジェクトの"img"プロパティを返す。空文字なら
// 場所ごとの絵の指定なしとして扱い、デフォルトの
// LeverWallOpenImg/LeverWallOpenDecoImgを使う(game.goのコメント参照)。
func leverWallImageKey(p map[string]string) string {
	return p["img"]
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
