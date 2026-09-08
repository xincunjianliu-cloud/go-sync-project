package main

// menu_scene.go: メニューの状態定義・初期化・メイン/ステータス/セーブロードの更新処理

import (
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	menuStateMain                = "main"
	menuStateSkillCharSel        = "skillCharSel"
	menuStateSkillSub            = "skillSub"
	menuStateSkillUpgradeCharSel = "skillUpgradeCharSel" // ← 追加
	menuStateSkillUpgradeSub     = "skillUpgradeSub"     // ← 追加
	menuStateSkillUpgrade        = "skillUpgrade"
	menuStateHealTarget          = "healTarget"
	menuStateStatus              = "status"
	menuStateSaveSlot            = "saveSlot"
	menuStateLoadSlot            = "loadSlot"
	menuStateSaveConfirm         = "saveConfirm"
	menuStateLoadConfirm         = "loadConfirm"
	menuStateSaveDone            = "saveDone"
	menuStateOption              = "option"
	menuStateOptionAdjust        = "optionAdjust"
	menuStateMessageSpeedAdjust  = "messageSpeedAdjust"
	menuStateDisplayModeAdjust   = "displayModeAdjust"
	menuStateReturnTitleConfirm  = "returnTitleConfirm"

	maxSaveSlots = 20
	slotsPerPage = 4

	bgmVolumeStep = 0.05
)

type skillDef struct {
	name        string
	implemented bool
	mpCost      int
}

var menuSkillDefs = []skillDef{
	{name: "攻撃", implemented: false, mpCost: 0},
	{name: "回復", implemented: true, mpCost: mpCostHeal},
	{name: "もどる", implemented: true, mpCost: 0},
}

const (
	skillIdxAttack = 0
	skillIdxHeal   = 1
	skillIdxBack   = 2
)

// ↓ メニューのコードの上の方（定数を定義している場所）
const defaultBGMVolume = 0. // ★これ1つだけで管理します！

const defaultMessageSpeed = 1
const defaultDisplayModeIndex = 0

// ※ const resetBGMVolume = 0.4 は削除してOKです！

type MenuScene struct {
	game            *Game
	backScene       Scene
	menuIndex       int
	commands        []string
	menuState       string
	skillCharIndex  int
	skillSubIndex   int
	healTargetIndex int
	statusCharIndex int // ← 追加
	nextScene       Scene

	optionIndex int

	previewTicks int

	slotIndex     int
	slotData      [maxSaveSlots]*SaveData
	slotThumbs    [maxSaveSlots]*ebiten.Image
	saveMode      bool
	saveResultMsg string
	saveDoneIndex int

	confirmIndex int
	pendingSlot  int

	upgradeConfirmIndex int
	upgradeResultMsg    string
	skillLevelCursor    int // ← 追加：スキル行内での数字(Lv)カーソル
	skillLevelSelecting bool
	pendingSkill        int // ← 追加：メニューからの回復スキル使用時、対象スキルIndex+1（0=未使用）
	pendingSkillLevel   int // ← 追加：メニューからの回復スキル使用時、使用するLv
}

func NewMenuScene(game *Game, backScene Scene) *MenuScene {
	return &MenuScene{
		game:      game,
		backScene: backScene,
		menuIndex: game.LastMenuIndex,
		commands:  []string{"アイテム", "スキル", "ステータス", "セーブ", "ロード", "オプション", "タイトルに戻る"}, // ← 追加
		menuState: menuStateMain,
	}
}

func (m *MenuScene) Update(dt float64) Scene {
	m.nextScene = nil
	m.previewTicks++

	switch m.menuState {
	case menuStateMain:
		m.updateMain()
	case menuStateSkillCharSel:
		m.updateSkillCharSel()
	case menuStateSkillSub:
		m.updateSkillSub()
	case menuStateSkillUpgrade:
		m.updateSkillUpgrade()
	case menuStateHealTarget:
		m.updateHealTarget()
	case menuStateStatus: // ← 追加
		m.updateStatus()
	case menuStateSaveSlot, menuStateLoadSlot:
		m.updateSlot()
	case menuStateSaveConfirm:
		m.updateSaveConfirm()
	case menuStateLoadConfirm:
		m.updateLoadConfirm()
	case menuStateSaveDone:
		m.updateSaveDone()
	case menuStateOption:
		m.updateOption()
	case menuStateReturnTitleConfirm: // ← 追加
		m.updateReturnTitleConfirm()
	case menuStateOptionAdjust:
		m.updateOptionAdjust()
	case menuStateMessageSpeedAdjust:
		m.updateMessageSpeedAdjust()
	case menuStateDisplayModeAdjust: // ← 追加
		m.updateDisplayModeAdjust() // ← 追加
	}

	if m.nextScene != nil {
		return m.nextScene
	}
	return m
}

func (m *MenuScene) updateMain() {
	if isEscapePressed() {
		m.nextScene = m.backScene
		return
	}
	if isMenuDownPressed() {
		m.menuIndex = (m.menuIndex + 1) % len(m.commands)
		m.game.LastMenuIndex = m.menuIndex // ← 追加：移動しただけで記憶
	}
	if isMenuUpPressed() {
		m.menuIndex = (m.menuIndex - 1 + len(m.commands)) % len(m.commands)
		m.game.LastMenuIndex = m.menuIndex // ← 追加：移動しただけで記憶
	}
	if !isConfirmKeyPressed() {
		return
	}
	if !isConfirmKeyPressed() {
		return
	}
	m.game.LastMenuIndex = m.menuIndex // ← この行は残しておいてOK（重複だが害はない、削除しても良い）
	switch m.menuIndex {
	case 0: // アイテム
	case 1: // スキル
		m.skillCharIndex = m.game.LastSkillCharIndex
		m.menuState = menuStateSkillCharSel
	case 2: // ステータス
		m.statusCharIndex = m.game.LastStatusCharIndex
		m.menuState = menuStateStatus
	case 3: // セーブ
		m.enterSlotScreen(true)
	case 4: // ロード
		m.enterSlotScreen(false)
	case 5: // オプション
		m.menuState = menuStateOption
	case 6: // タイトルに戻る ← 追加
		m.confirmIndex = 1 // デフォルト「いいえ」にしておくと事故防止になる
		m.menuState = menuStateReturnTitleConfirm

	}
}

func (m *MenuScene) updateStatus() {
	if isEscapePressed() {
		m.menuState = menuStateMain
		return
	}
	if isMenuRightPressed() {
		m.statusCharIndex = (m.statusCharIndex + 1) % partySize
		m.game.LastStatusCharIndex = m.statusCharIndex // ← 追加
	}
	if isMenuLeftPressed() {
		m.statusCharIndex = (m.statusCharIndex - 1 + partySize) % partySize
		m.game.LastStatusCharIndex = m.statusCharIndex // ← 追加
	}
}

func (m *MenuScene) enterSlotScreen(save bool) {
	m.saveMode = save
	m.slotIndex = 0
	for i := 0; i < maxSaveSlots; i++ {
		m.slotData[i], _ = LoadGame(i + 1)
		m.slotThumbs[i] = LoadThumb(i + 1)
	}
	if save {
		m.menuState = menuStateSaveSlot
	} else {
		m.menuState = menuStateLoadSlot
	}
}

// reloadSlotData はスロット一覧だけを再読込する（カーソル位置は変更しない）
func (m *MenuScene) reloadSlotData() {
	for i := 0; i < maxSaveSlots; i++ {
		m.slotData[i], _ = LoadGame(i + 1)
		m.slotThumbs[i] = LoadThumb(i + 1)
	}
}

func (m *MenuScene) updateSlot() {
	if isEscapePressed() {
		m.menuState = menuStateMain
		return
	}
	if isMenuUpPressed() {
		m.slotIndex = (m.slotIndex - 1 + maxSaveSlots) % maxSaveSlots
	}
	if isMenuDownPressed() {
		m.slotIndex = (m.slotIndex + 1) % maxSaveSlots
	}
	if !isConfirmKeyPressed() {
		return
	}

	slot := m.slotIndex + 1

	if m.saveMode {
		m.pendingSlot = slot
		m.confirmIndex = 0
		m.menuState = menuStateSaveConfirm
	} else {
		d := m.slotData[m.slotIndex]
		if d == nil {
			return
		}
		m.pendingSlot = slot
		m.confirmIndex = 0
		m.menuState = menuStateLoadConfirm
	}
}

func (m *MenuScene) updateSaveConfirm() {
	if isMenuUpPressed() || isMenuDownPressed() {
		m.confirmIndex = 1 - m.confirmIndex
	}
	if isEscapePressed() {
		m.menuState = menuStateSaveSlot
		return
	}
	if !isConfirmKeyPressed() {
		return
	}

	if m.confirmIndex == 1 {
		m.menuState = menuStateSaveSlot
		return
	}

	slot := m.pendingSlot
	if field, ok := m.backScene.(*FieldScene); ok {
		globalActiveFieldInstanceForSave = field
		err := SaveGame(slot, field.currentMap, field.px, field.py, m.game.PlayerHP)
		m.game.saveThumbToFile(slot)
		if err != nil {
			m.saveResultMsg = "セーブに失敗しました"
		} else {
			m.saveResultMsg = "セーブしました"
		}
	}
	m.menuState = menuStateSaveDone
}

func (m *MenuScene) updateReturnTitleConfirm() {
	if isMenuUpPressed() || isMenuDownPressed() {
		m.confirmIndex = 1 - m.confirmIndex
	}
	if isEscapePressed() {
		m.menuState = menuStateMain
		return
	}
	if !isConfirmKeyPressed() {
		return
	}
	if m.confirmIndex == 1 { // いいえ
		m.menuState = menuStateMain
		return
	}
	// はい：タイトルへ
	m.nextScene = NewTitleScene(m.game)
}

func (m *MenuScene) updateLoadConfirm() {
	if isMenuUpPressed() || isMenuDownPressed() {
		m.confirmIndex = 1 - m.confirmIndex
	}
	if isEscapePressed() {
		m.menuState = menuStateLoadSlot
		return
	}
	if !isConfirmKeyPressed() {
		return
	}

	if m.confirmIndex == 1 {
		m.menuState = menuStateLoadSlot
		return
	}

	d := m.slotData[m.pendingSlot-1]
	if d == nil {
		m.menuState = menuStateLoadSlot
		return
	}
	m.game.TotalPlayTime = d.PlayTime
	m.game.PlayerHP = d.PlayerHP
	m.game.PlayerMaxHP = d.PlayerMaxHP
	m.game.PlayerMP = d.PlayerMP
	m.game.PlayerMaxMP = d.PlayerMaxMP
	m.game.PlayerAtk = d.PlayerAtk
	m.game.PlayerLv = d.PlayerLv
	m.game.PlayerEXP = d.PlayerEXP
	m.game.PlayerNextEXP = d.PlayerNextEXP

	field, err := NewRoomScene(m.game, d.CurrentMap, d.PlayerX, d.PlayerY, "", d.PlayerDir)
	if err != nil {
		m.menuState = menuStateLoadSlot
		return
	}
	m.game.ChangeSceneWithFade(field, fadeTimeContinue)
}

func (m *MenuScene) updateSaveDone() {
	if isEscapePressed() {
		m.reloadSlotData()
		m.menuState = menuStateSaveSlot
		return
	}
	if !isConfirmKeyPressed() {
		return
	}
	// セーブ画面に戻る（カーソルは保存したスロットのまま、一覧だけ最新化）
	m.reloadSlotData()
	m.slotIndex = m.pendingSlot - 1
	m.menuState = menuStateSaveSlot
}
