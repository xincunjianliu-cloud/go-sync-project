package main

import (
	"encoding/json"
	"fmt"
	"path"
	"strconv"
	"strings"
)

type EventCommand struct {
	Speaker string
	Text    string
}

type BossDialogue struct {
	Commands     []EventCommand
	SpeakerSlots map[string]int
	BGM          string
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
}

type bossDialogueJSON struct {
	Commands     []dialogueCommandJSON `json:"commands"`
	SpeakerSlots map[string]int        `json:"speakerSlots"`
	// BGM はこの会話中に流すBGMをbgmByKeyのキー名で指定する(任意)。
	// 省略時はそれまで流れていたBGMをそのまま継続する。
	BGM string `json:"bgm"`
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

func resolveSpeaker(key string) (string, error) {
	if key == "" || key == "SYSTEM_COMMAND" {
		return key, nil
	}
	if name, ok := speakerNameMap[key]; ok {
		return name, nil
	}
	return "", fmt.Errorf("未知の話者キー: %q", key)
}

func convertBossDialogue(src *bossDialogueJSON) BossDialogue {
	if src == nil {
		return BossDialogue{}
	}

	commands := make([]EventCommand, 0, len(src.Commands))
	for _, c := range src.Commands {
		speaker, err := resolveSpeaker(c.Speaker)
		if err != nil {
			continue
		}
		commands = append(commands, EventCommand{Speaker: speaker, Text: c.Text})
	}

	slots := make(map[string]int, len(src.SpeakerSlots))
	for key, slot := range src.SpeakerSlots {
		speaker, err := resolveSpeaker(key)
		if err != nil {
			continue
		}
		slots[speaker] = slot
	}

	return BossDialogue{Commands: commands, SpeakerSlots: slots, BGM: src.BGM}
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

func loadStoryDialogues(dir string) {
	filePath := path.Join(dir, "story.json")

	data, err := loadAssetBytes(filePath)
	if err != nil {
		return
	}

	var fileJSON map[string]storyDialogueFileJSON
	if err := json.Unmarshal(data, &fileJSON); err != nil {
		return
	}

	for id, entry := range fileJSON {
		storyDialogues[id] = storyDialogueEntry{
			First:  filterEmptyCommands(convertBossDialogue(entry.First)),
			Repeat: filterEmptyCommands(convertBossDialogue(entry.Repeat)),
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
