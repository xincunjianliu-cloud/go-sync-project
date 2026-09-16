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

func convertBossDialogue(src *bossDialogueJSON, fileName string) BossDialogue {
	if src == nil {
		return BossDialogue{}
	}

	commands := make([]EventCommand, 0, len(src.Commands))
	for i, c := range src.Commands {
		speaker, err := resolveSpeaker(c.Speaker)
		if err != nil {
			fmt.Printf("警告: %s の %d番目のコマンドで話者解決に失敗: %v（このコマンドはスキップ）\n", fileName, i, err)
			continue
		}
		commands = append(commands, EventCommand{Speaker: speaker, Text: c.Text})
	}

	slots := make(map[string]int, len(src.SpeakerSlots))
	for key, slot := range src.SpeakerSlots {
		speaker, err := resolveSpeaker(key)
		if err != nil {
			fmt.Printf("警告: %s の speakerSlots で話者解決に失敗: %v（このエントリはスキップ）\n", fileName, err)
			continue
		}
		slots[speaker] = slot
	}

	return BossDialogue{Commands: commands, SpeakerSlots: slots}
}

func LoadDialogues(dir string) error {
	for bossNum := 1; bossNum <= 4; bossNum++ {
		filePath := path.Join(dir, fmt.Sprintf("boss_%d.json", bossNum))

		data, err := loadAssetBytes(filePath)
		if err != nil {
			fmt.Printf("警告: %s の読み込みに失敗しました（このボスのセリフは空になります）: %v\n", filePath, err)
			continue
		}

		var fileJSON bossDialogueFileJSON
		if err := json.Unmarshal(data, &fileJSON); err != nil {
			fmt.Printf("警告: %s のJSON解析に失敗しました（このボスのセリフは空になります）: %v\n", filePath, err)
			continue
		}

		bossBattleDialogues[bossNum] = filterEmptyCommands(convertBossDialogue(fileJSON.Battle, filePath))
		bossClearDialogues[bossNum] = filterEmptyCommands(convertBossDialogue(fileJSON.Clear, filePath))
	}

	if err := loadStoryDialogues(dir); err != nil {
		fmt.Printf("警告: %v\n", err)
	}

	return nil
}

func loadStoryDialogues(dir string) error {
	filePath := path.Join(dir, "story.json")

	data, err := loadAssetBytes(filePath)
	if err != nil {
		return nil
	}

	var fileJSON map[string]bossDialogueJSON
	if err := json.Unmarshal(data, &fileJSON); err != nil {
		return fmt.Errorf("%s のJSON解析に失敗しました: %w", filePath, err)
	}

	for id, entry := range fileJSON {
		entry := entry
		storyDialogues[id] = filterEmptyCommands(convertBossDialogue(&entry, fmt.Sprintf("%s[%s]", filePath, id)))
	}
	return nil
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

func validateDialogueSlots() {
	check := func(label string, dialogues map[int]BossDialogue) {
		for bossNum, d := range dialogues {
			seen := map[int]string{}
			for speaker, slot := range d.SpeakerSlots {
				if other, ok := seen[slot]; ok {
					fmt.Printf("警告: %s ボス%d でスロット%dに複数の話者が割り当てられています（%s, %s）\n",
						label, bossNum, slot, other, speaker)
					continue
				}
				seen[slot] = speaker
			}
		}
	}
	check("bossBattleDialogues", bossBattleDialogues)
	check("bossClearDialogues", bossClearDialogues)
}
