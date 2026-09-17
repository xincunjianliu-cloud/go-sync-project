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

var storyDialogues = map[string]BossDialogue{}

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

func loadStoryDialogues(dir string) {
	filePath := path.Join(dir, "story.json")

	data, err := loadAssetBytes(filePath)
	if err != nil {
		return
	}

	var fileJSON map[string]bossDialogueJSON
	if err := json.Unmarshal(data, &fileJSON); err != nil {
		return
	}

	for id, entry := range fileJSON {
		entry := entry
		storyDialogues[id] = filterEmptyCommands(convertBossDialogue(&entry))
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

func GetStoryDialogue(id string) (BossDialogue, bool) {
	d, ok := storyDialogues[id]
	return d, ok
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
