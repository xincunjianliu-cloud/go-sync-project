package main

import (
	"encoding/json"
	"fmt"
	"log"
	"path"
	"strconv"
	"strings"
)

type EventCommand struct {
	Speaker    string
	Text       string
	Expression int
}

type BossDialogue struct {
	Commands []EventCommand
	BGM      string
	// SpeakerSides is an optional "who stands on which side" hint
	// (0=left, 1=right). A speaker with no entry here is placed
	// automatically (see MessageSystem.UpdateCharaAnim).
	SpeakerSides map[string]int
	// Background is an optional image key (assets/images/backgrounds/<key>.png)
	// drawn full-screen in place of the map while this dialogue plays
	// (e.g. for an opening/visual-novel-style scene). Empty means "just
	// show the map as usual".
	Background string
}

var bossBattleDialogues = map[int]BossDialogue{}
var bossClearDialogues = map[int]BossDialogue{}

// storyDialogueEntry は1つの会話イベントの初回セリフと2回目以降セリフを
// 1つのJSONオブジェクトにまとめたもの(story.json参照)。Repeatは任意で、
// 省略された場合はFirstがそのまま繰り返される。
type storyDialogueEntry struct {
	First  BossDialogue
	Repeat BossDialogue
}

var storyDialogues = map[string]storyDialogueEntry{}

type dialogueCommandJSON struct {
	Speaker string `json:"speaker"`
	Text    string `json:"text"`
	// Expression はこのセリフで表示する表情差分のコマ番号(任意)。
	// 全キャラ共通で 0=通常 1=笑顔 2=怒り 3=驚き の固定割り当て。
	// その話者のchara_<キャラ>_sheet.pngのうち何番目のコマを使うかを指定する
	// (シートが用意されていないキャラの場合は無視され、通常の立ち絵を使う)。
	Expression int `json:"expression"`
}

type bossDialogueJSON struct {
	Commands []dialogueCommandJSON `json:"commands"`
	// BGM はこの会話中に流すBGMをbgmByKeyのキー名で指定する(任意)。
	// 省略時はそれまで流れていたBGMをそのまま継続する。
	BGM string `json:"bgm"`
	// SpeakerSlots はどの話者を左(0)/右(1)どちらの立ち絵枠に固定するかの
	// 任意指定。指定が無い話者は「直近喋っていない方の枠」に自動で入る。
	// 同じ側を2人以上に指定すると、その側の枠だけがその2人の間で
	// 入れ替わる(もう片方の枠には影響しない)。
	SpeakerSlots map[string]int `json:"speakerSlots"`
	// Background はこの会話中にマップの代わりに全画面表示する背景画像の
	// キー(assets/images/backgrounds/<キー>.png、任意)。省略時は
	// マップをそのまま表示する。
	Background string `json:"background"`
}

type bossDialogueFileJSON struct {
	Battle *bossDialogueJSON `json:"battle"`
	Clear  *bossDialogueJSON `json:"clear"`
}

var speakerNameMap = buildSpeakerNameMap()

func buildSpeakerNameMap() map[string]string {
	m := make(map[string]string, len(BossNames)+len(PlayerNames))
	for i, name := range BossNames {
		m[fmt.Sprintf("boss%d", i+1)] = name
	}
	for i, name := range PlayerNames {
		m[fmt.Sprintf("player%d", i+1)] = name
	}
	return m
}

// resolveSpeaker はJSON上の話者キーを表示名に変換する。"player1"/"boss1"の
// ようなキーは既知の名前に変換し、それ以外の文字列はNPCの表示名として
// そのまま使う(NPCを増やすたびにGoコード側の対応表を増やす必要がないため)。
func resolveSpeaker(key string) string {
	if key == "" || key == "SYSTEM_COMMAND" {
		return key
	}
	if name, ok := speakerNameMap[key]; ok {
		return name
	}
	return key
}

func convertBossDialogue(src *bossDialogueJSON) BossDialogue {
	if src == nil {
		return BossDialogue{}
	}

	commands := make([]EventCommand, 0, len(src.Commands))
	for _, c := range src.Commands {
		commands = append(commands, EventCommand{
			Speaker:    resolveSpeaker(c.Speaker),
			Text:       c.Text,
			Expression: c.Expression,
		})
	}

	var sides map[string]int
	if len(src.SpeakerSlots) > 0 {
		sides = make(map[string]int, len(src.SpeakerSlots))
		for key, side := range src.SpeakerSlots {
			sides[resolveSpeaker(key)] = side
		}
	}

	return BossDialogue{Commands: commands, BGM: src.BGM, SpeakerSides: sides, Background: src.Background}
}

func LoadDialogues(dir string) error {
	for bossNum := 1; bossNum <= 4; bossNum++ {
		filePath := path.Join(dir, fmt.Sprintf("boss_%d.json", bossNum))

		data, err := loadAssetBytes(filePath)
		if err != nil {
			continue
		}

		var fileJSON bossDialogueFileJSON
		if err := json.Unmarshal(data, &fileJSON); err != nil {
			continue
		}

		bossBattleDialogues[bossNum] = filterEmptyCommands(convertBossDialogue(fileJSON.Battle))
		bossClearDialogues[bossNum] = filterEmptyCommands(convertBossDialogue(fileJSON.Clear))
	}

	loadStoryDialogues(dir)

	return nil
}

type storyDialogueFileJSON struct {
	First  *bossDialogueJSON `json:"first"`
	Repeat *bossDialogueJSON `json:"repeat"`
}

// loadStoryDialogues はassets/dialogues/story/以下の会話ファイルを読み込む。
// 1ファイルに全イベントをまとめず複数ファイルに分けているのは、
// どこか1箇所のJSON構文ミスがそのファイルだけに影響を留め、他の会話まで
// 巻き添えで読み込み不能にしないため。どのファイルを読むかは
// story/_index.json(ファイル名の配列)で管理する。web版はディレクトリの
// 一覧取得ができない(静的ファイルとしてfetchするだけ)ため、ネイティブ版と
// 挙動を揃えるためにあえてこの一覧ファイルを使っている。
func loadStoryDialogues(dir string) {
	storyDir := path.Join(dir, "story")
	indexPath := path.Join(storyDir, "_index.json")

	indexData, err := loadAssetBytes(indexPath)
	if err != nil {
		log.Printf("会話ファイル一覧の読み込み失敗 %s: %v", indexPath, err)
		return
	}

	var files []string
	if err := json.Unmarshal(indexData, &files); err != nil {
		log.Printf("会話ファイル一覧の構文エラー %s: %v", indexPath, err)
		return
	}

	for _, file := range files {
		filePath := path.Join(storyDir, file)

		data, err := loadAssetBytes(filePath)
		if err != nil {
			log.Printf("会話ファイルの読み込み失敗 %s: %v", filePath, err)
			continue
		}

		var fileJSON map[string]storyDialogueFileJSON
		if err := json.Unmarshal(data, &fileJSON); err != nil {
			log.Printf("会話ファイルの構文エラー %s: %v", filePath, err)
			continue
		}

		for id, entry := range fileJSON {
			if _, dup := storyDialogues[id]; dup {
				log.Printf("会話idが重複しています: %q (%s)", id, filePath)
			}
			storyDialogues[id] = storyDialogueEntry{
				First:  filterEmptyCommands(convertBossDialogue(entry.First)),
				Repeat: filterEmptyCommands(convertBossDialogue(entry.Repeat)),
			}
		}
	}
}

func GetEventCommands(eventID string, game *Game) BossDialogue {
	const prefix = "event_boss_"
	if !strings.HasPrefix(eventID, prefix) {
		return BossDialogue{}
	}
	rest := eventID[len(prefix):]

	isClear := false
	if strings.HasSuffix(rest, "_clear") {
		isClear = true
		rest = strings.TrimSuffix(rest, "_clear")
	}

	bossNum, err := strconv.Atoi(rest)
	if err != nil || bossNum < 1 || bossNum > 4 {
		return BossDialogue{}
	}

	if isClear {
		return bossClearDialogues[bossNum]
	}

	if game.BossDefeatedFlags[bossNum-1] {
		return BossDialogue{}
	}
	return bossBattleDialogues[bossNum]
}

// GetStoryDialogue はstory.jsonのidに対応する会話を返す。repeatがtrueで
// かつそのidに"repeat"セリフが個別に用意されていればそちらを、
// 無ければ"first"(初回セリフ)をそのまま返す。
func GetStoryDialogue(id string, repeat bool) (BossDialogue, bool) {
	entry, ok := storyDialogues[id]
	if !ok {
		return BossDialogue{}, false
	}
	if repeat && len(entry.Repeat.Commands) > 0 {
		return entry.Repeat, true
	}
	return entry.First, true
}

func filterEmptyCommands(d BossDialogue) BossDialogue {
	filtered := make([]EventCommand, 0, len(d.Commands))
	for _, cmd := range d.Commands {
		if cmd.Text == "" {
			continue
		}
		filtered = append(filtered, cmd)
	}
	d.Commands = filtered
	return d
}
