package main

// menu_scene_options.go: オプション画面（音量・メッセージ速度・表示モード）の更新処理と設定保存
func (m *MenuScene) updateOption() {
	if isEscapePressed() {
		m.menuState = menuStateMain
		return
	}
	if isMenuUpPressed() {
		m.optionIndex = (m.optionIndex - 1 + 4) % 4
	}
	if isMenuDownPressed() {
		m.optionIndex = (m.optionIndex + 1) % 4
	}
	if !isConfirmKeyPressed() {
		return
	}
	switch m.optionIndex {
	case 0:
		m.menuState = menuStateOptionAdjust // BGM
	case 1:
		m.menuState = menuStateDisplayModeAdjust // ← 【変更】画面サイズを上に
	case 2:
		m.menuState = menuStateMessageSpeedAdjust // ← 【変更】メッセージ速度を下に
	case 3:
		if m.game.Audio != nil {
			m.game.Audio.SetVolume(defaultBGMVolume)
		}
		m.game.MessageSpeed = defaultMessageSpeed
		m.game.DisplayModeIndex = defaultDisplayModeIndex
		applyDisplayMode(defaultDisplayModeIndex)
		m.persistSettings()
	}
}

func (m *MenuScene) updateOptionAdjust() {
	if isEscapePressed() || isConfirmKeyPressed() {
		m.menuState = menuStateOption
		return
	}
	if isMenuRightPressed() {
		m.changeBGMVolume(bgmVolumeStep)
		m.persistSettings() // ← 追加
	}
	if isMenuLeftPressed() {
		m.changeBGMVolume(-bgmVolumeStep)
		m.persistSettings() // ← 追加
	}
}

func (m *MenuScene) changeBGMVolume(delta float64) {
	if m.game.Audio == nil {
		return
	}
	v := m.game.Audio.volume + delta
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	m.game.Audio.SetVolume(v)
}

var menuCommandDescriptions = map[string]string{
	"アイテム":    "所持しているアイテムを使用・確認します",
	"スキル":     "覚えているスキルを確認できます",
	"ステータス":   "キャラクターの詳細なステータスを確認します", // ← 追加
	"セーブ":     "現在の状況をセーブします",
	"ロード":     "セーブデータをロードします",
	"オプション":   "ゲームを遊びやすいように設定できます",
	"タイトルに戻る": "タイトル画面に戻ります（未セーブの進行は失われます）",
}

var menuOptionDescriptions = map[int]string{
	0: "BGMの音量を調整します",
	1: "ウィンドウサイズ・フルスクリーンを切り替えます", // ← 【変更】番号を1に
	2: "メッセージの表示速度を変更します",        // ← 【変更】番号を2に
	3: "設定をすべて初期値に戻します",
}

var menuSkillDescriptions = map[int]string{
	skillIdxAttack: "攻撃スキルを発動します（現在は未実装です）",
	skillIdxHeal:   "対象のHPを回復します",
	skillIdxBack:   "キャラクター選択に戻ります",
}

const messageSpeedPreviewText = "メッセージはこの速度で表示されます"

func (m *MenuScene) previewRevealCount() int {
	runes := []rune(messageSpeedPreviewText)
	speedTicks := m.game.MessageSpeedTicks()
	if speedTicks <= 0 {
		speedTicks = 1
	}
	holdTicks := 40
	cycleLen := len(runes)*speedTicks + holdTicks
	if cycleLen <= 0 {
		return 0
	}
	pos := m.previewTicks % cycleLen
	count := pos / speedTicks
	if count > len(runes) {
		count = len(runes)
	}
	return count
}

func (m *MenuScene) updateMessageSpeedAdjust() {
	if isEscapePressed() || isConfirmKeyPressed() {
		m.menuState = menuStateOption
		return
	}
	if isMenuRightPressed() {
		if m.game.MessageSpeed < 2 {
			m.game.MessageSpeed++
			m.persistSettings() // ← 追加
		}
	}
	if isMenuLeftPressed() {
		if m.game.MessageSpeed > 0 {
			m.game.MessageSpeed--
			m.persistSettings() // ← 追加
		}
	}
}

// skillShortDescription はスキル行選択中（Lv未選択時）に表示するシンプルな説明を返す。
// 各スキルの詳細（Lv別）はここでは出さず、大枠の効果だけを一言で伝える。
var skillShortDescriptions = map[string]string{
	"回復": "対象を回復する",
	// 例：他のスキルもここに追記
	// "強撃":   "敵に大ダメージを与える",
	// "巻き戻し": "直前の行動を巻き戻す",
}

func skillShortDescription(skillName string) string {
	if d, ok := skillShortDescriptions[skillName]; ok {
		return d
	}
	return skillName // マップに無ければスキル名だけ表示
}

func (m *MenuScene) updateDisplayModeAdjust() {
	if isEscapePressed() || isConfirmKeyPressed() {
		m.menuState = menuStateOption
		return
	}
	if isMenuRightPressed() && m.game.DisplayModeIndex < len(displayModeLabels)-1 {
		m.game.DisplayModeIndex++
		applyDisplayMode(m.game.DisplayModeIndex)
		m.persistSettings()
	}
	if isMenuLeftPressed() && m.game.DisplayModeIndex > 0 {
		m.game.DisplayModeIndex--
		applyDisplayMode(m.game.DisplayModeIndex)
		m.persistSettings()
	}
}

func (m *MenuScene) persistSettings() {
	vol := defaultBGMVolume
	if m.game.Audio != nil {
		vol = m.game.Audio.volume
	}
	SaveSettings(GameSettings{
		BGMVolume:        vol,
		MessageSpeed:     m.game.MessageSpeed,
		DisplayModeIndex: m.game.DisplayModeIndex,
	})
}
