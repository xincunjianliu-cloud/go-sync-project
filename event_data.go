package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type EventCommand struct {
	Speaker string
	Text    string
}

type BossDialogue struct {
	Commands     []EventCommand
	SpeakerSlots map[string]int // 0=左, 1=右
}

var bossBattleDialogues = map[int]BossDialogue{}
var bossClearDialogues = map[int]BossDialogue{}

// --- JSON読み込み用の中間構造体 ---

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

// speakerNameMap は JSON上の話者キー文字列 → ゲーム内の話者名定数 の対応表。
// 新しい話者を追加したらここに登録する。
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

// resolveSpeaker はJSON上の話者キーをゲーム内の話者名に変換する。
// "" と "SYSTEM_COMMAND" はそのまま通す（システムメッセージ用）。
func resolveSpeaker(key string) (string, error) {
	if key == "" || key == "SYSTEM_COMMAND" {
		return key, nil
	}
	if name, ok := speakerNameMap[key]; ok {
		return name, nil
	}
	return "", fmt.Errorf("未知の話者キー: %q", key)
}

// convertBossDialogue はJSON中間構造体をゲーム内の BossDialogue に変換する。
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

// LoadDialogues は assets/dialogues/ 以下の boss_N.json を全て読み込み、
// bossBattleDialogues / bossClearDialogues を構築する。
// ゲーム起動時に一度だけ呼ぶこと。
func LoadDialogues(dir string) error {
	for bossNum := 1; bossNum <= 4; bossNum++ {
		path := filepath.Join(dir, fmt.Sprintf("boss_%d.json", bossNum))

		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("警告: %s の読み込みに失敗しました（このボスのセリフは空になります）: %v\n", path, err)
			continue
		}

		var fileJSON bossDialogueFileJSON
		if err := json.Unmarshal(data, &fileJSON); err != nil {
			fmt.Printf("警告: %s のJSON解析に失敗しました（このボスのセリフは空になります）: %v\n", path, err)
			continue
		}

		bossBattleDialogues[bossNum] = filterEmptyCommands(convertBossDialogue(fileJSON.Battle, path))
		bossClearDialogues[bossNum] = filterEmptyCommands(convertBossDialogue(fileJSON.Clear, path))
	}
	return nil
}

// GetEventCommands は eventID (例: "event_boss_2", "event_boss_2_clear") を解析し、
// 対応する BossDialogue を返す。
// ※ event_story_ 系はここでは扱わない（field_scene.go 側で個別処理）。
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

// filterEmptyCommands は Text が空のコマンドを取り除く。
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

// validateDialogueSlots は、同じスロット(0 or 1)に
// 2人以上の話者が割り当てられていないかチェックする。
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
