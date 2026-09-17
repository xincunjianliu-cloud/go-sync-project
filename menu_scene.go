package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	menuStateMain               = "main"
	menuStateSkillCharSel       = "skillCharSel"
	menuStateSkillSub           = "skillSub"
	menuStateHealTarget         = "healTarget"
	menuStateItemList           = "itemList"
	menuStateItemTarget         = "itemTarget"
	menuStateStatus             = "status"
	menuStateSaveSlot           = "saveSlot"
	menuStateLoadSlot           = "loadSlot"
	menuStateSaveConfirm        = "saveConfirm"
	menuStateLoadConfirm        = "loadConfirm"
	menuStateSaveDone           = "saveDone"
	menuStateOption             = "option"
	menuStateOptionResetConfirm = "optionResetConfirm"
	menuStateOptionResetDone    = "optionResetDone"

	maxSaveSlots = 20
	slotsPerPage = 4

	bgmVolumeStep = 0.01

	bgmVolumeRepeatDelayTicks    = 18
	bgmVolumeRepeatIntervalTicks = 2
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

const defaultBGMVolume = 0.
const defaultSEVolume = 0.
const defaultMasterVolume = 1.

const defaultMessageSpeed = 1
const defaultFullscreen = false
const defaultWindowWidth = gameWidth
const defaultWindowHeight = gameHeight
const defaultRememberCursor = true

type MenuScene struct {
	game              *Game
	backScene         Scene
	menuIndex         int
	commands          []string
	menuState         string
	skillCharIndex    int
	skillSubIndex     int
	healTargetIndex   int
	statusCharIndex   int
	statusReturnState string
	nextScene         Scene

	optionIndex int

	previewTicks int

	slotIndex         int
	slotScrollTop     int
	slotData          [maxSaveSlots]*SaveData
	slotThumbs        [maxSaveSlots]*ebiten.Image
	saveMode          bool
	saveResultMsg     string
	saveDoneIndex     int
	slotDrag          dragScrollState
	slotScrollBarDrag dragScrollState
	slotDragAccum     float64

	volumeDragActive     bool
	volumeDragRow        int
	volumeLeftHoldTicks  int
	volumeRightHoldTicks int

	confirmIndex           int
	pendingSlot            int
	showReturnTitleConfirm bool
	inputLockTicks         int

	upgradeProgress     float64
	upgradeHoldArmed    bool
	skillLevelCursor    int
	skillLevelSelecting bool
	pendingSkill        int
	pendingSkillLevel   int

	itemListIndex   int
	pendingItemID   string
	itemTargetIndex int

	notice      string
	noticeTicks int
}

func NewMenuScene(game *Game, backScene Scene) *MenuScene {
	return &MenuScene{
		game:      game,
		backScene: backScene,
		menuIndex: game.rememberedIndex(game.LastMenuIndex),
		commands:  []string{"アイテム", "スキル", "ステータス", "セーブ", "ロード", "オプション", "タイトルに戻る"},
		menuState: menuStateMain,
		slotIndex: game.rememberedIndex(game.LastSlotIndex),
	}
}

func (m *MenuScene) Update(dt float64) Scene {
	m.nextScene = nil
	m.previewTicks++
	m.tickNotice()

	if !m.isModalMenuState() && m.menuState != menuStateMain {
		if idx, ok := m.hitTestMainCommandList(); ok {
			m.game.Audio.PlaySEByKey("decide")
			m.enterCommand(idx)
			if m.nextScene != nil {
				return m.nextScene
			}
			return m
		}
	}

	if m.showReturnTitleConfirm {
		m.updateReturnTitleConfirm()
		if m.nextScene != nil {
			return m.nextScene
		}
		return m
	}

	switch m.menuState {
	case menuStateMain:
		m.updateMain()
	case menuStateSkillCharSel:
		m.updateSkillCharSel()
	case menuStateSkillSub:
		m.updateSkillSub()
	case menuStateHealTarget:
		m.updateHealTarget()
	case menuStateItemList:
		m.updateItemList()
	case menuStateItemTarget:
		m.updateItemTarget()
	case menuStateStatus:
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
	case menuStateOptionResetConfirm:
		m.updateOptionResetConfirm()
	case menuStateOptionResetDone:
		m.updateOptionResetDone()
	}

	if m.nextScene != nil {
		return m.nextScene
	}
	return m
}

func (m *MenuScene) hitTestMainCommandList() (int, bool) {
	rects := make([]tapRect, len(m.commands))
	for i := range m.commands {
		y := cmdListStartY + float64(i)*cmdListRowGapY
		rects[i] = tapRect{x: cmdListStartX - 4, y: y - 4, w: menuFrameDividerX - cmdListStartX, h: cmdListRowGapY}
	}
	return hitTestTapRects(rects)
}

const (
	partyRowHitX = menuStatusOffsetX - 70
	partyRowHitW = 290
)

func (m *MenuScene) hitTestPartyRows() (int, bool) {
	rects := make([]tapRect, partySize)
	for i := 0; i < partySize; i++ {
		itemY := menuStatusStartY + float64(i)*menuStatusSpacingY
		rects[i] = tapRect{x: partyRowHitX, y: itemY - 20, w: partyRowHitW, h: 100}
	}
	return hitTestTapRects(rects)
}

func (m *MenuScene) allTargetRowRect() tapRect {
	y := menuStatusStartY + partySize*menuStatusSpacingY - drawAllTargetRowYOffset
	return tapRect{x: partyRowHitX, y: y - 20, w: partyRowHitW, h: 60}
}

func (m *MenuScene) hitTestPartyRowsWithAll(allowAll bool) (int, bool) {
	rects := make([]tapRect, 0, partySize+1)
	for i := 0; i < partySize; i++ {
		itemY := menuStatusStartY + float64(i)*menuStatusSpacingY
		rects = append(rects, tapRect{x: partyRowHitX, y: itemY - 20, w: partyRowHitW, h: 100})
	}
	if allowAll {
		rects = append(rects, m.allTargetRowRect())
	}
	return hitTestTapRects(rects)
}

func (m *MenuScene) updateMain() {
	if isEscapePressed() || isMenuCloseKeyPressed() {
		m.game.Audio.PlaySEByKey("menu_toggle")
		m.nextScene = m.backScene
		return
	}
	if isMenuDownRepeat() {
		m.menuIndex = (m.menuIndex + 1) % len(m.commands)
		m.game.LastMenuIndex = m.menuIndex
		m.game.Audio.PlaySEByKey("cursor")
	}
	if isMenuUpRepeat() {
		m.menuIndex = (m.menuIndex - 1 + len(m.commands)) % len(m.commands)
		m.game.LastMenuIndex = m.menuIndex
		m.game.Audio.PlaySEByKey("cursor")
	}
	tapped := false
	if idx, ok := m.hitTestMainCommandList(); ok {
		m.menuIndex = idx
		m.game.LastMenuIndex = m.menuIndex
		tapped = true
	}
	if !isConfirmKeyPressed() && !tapped {
		if unrelatedTapOutsideRects(menuMainContentRect()) {
			m.game.Audio.PlaySEByKey("menu_toggle")
			m.nextScene = m.backScene
		}
		return
	}
	m.game.Audio.PlaySEByKey("decide")
	m.enterCommand(m.menuIndex)
}

func (m *MenuScene) openStatusFor(idx int, back string) {
	m.statusCharIndex = idx
	m.game.LastStatusCharIndex = idx
	m.statusReturnState = back
	m.menuState = menuStateStatus
}

func (m *MenuScene) statusBackState() string {
	if m.statusReturnState == "" {
		return menuStateMain
	}
	return m.statusReturnState
}

func (m *MenuScene) enterCommand(idx int) {
	m.menuIndex = idx
	m.game.LastMenuIndex = idx
	m.clearNotice()
	switch idx {
	case 0:
		m.pendingItemID = ""
		m.menuState = menuStateItemList
	case 1:
		m.skillCharIndex = m.game.rememberedIndex(m.game.LastSkillCharIndex)
		m.menuState = menuStateSkillCharSel
	case 2:
		m.openStatusFor(m.game.rememberedIndex(m.game.LastStatusCharIndex), menuStateMain)
	case 3:
		m.enterSlotScreen(true)
	case 4:
		m.enterSlotScreen(false)
	case 5:
		m.menuState = menuStateOption
	case 6:
		m.confirmIndex = 1
		m.showReturnTitleConfirm = true
		lockDialogInput(&m.inputLockTicks)
		m.resetSlotDrag()
	}
}

func (m *MenuScene) hitTestStatusPartyIcons() (int, bool) {
	rects := make([]tapRect, partySize)
	for i := 0; i < partySize; i++ {
		cx := statusPartyIconStartX + float64(i)*statusPartyIconGap
		rects[i] = tapRect{x: cx - statusPartyIconGap/2, y: statusPartyIconY - 45, w: statusPartyIconGap, h: 90}
	}
	return hitTestTapRects(rects)
}

func (m *MenuScene) updateStatus() {
	if isEscapePressed() {
		m.game.Audio.PlaySEByKey("cancel")
		m.menuState = m.statusBackState()
		return
	}
	if isMenuRightPressed() {
		m.statusCharIndex = (m.statusCharIndex + 1) % partySize
		m.game.LastStatusCharIndex = m.statusCharIndex
		m.game.Audio.PlaySEByKey("cursor")
	}
	if isMenuLeftPressed() {
		m.statusCharIndex = (m.statusCharIndex - 1 + partySize) % partySize
		m.game.LastStatusCharIndex = m.statusCharIndex
		m.game.Audio.PlaySEByKey("cursor")
	}
	if idx, ok := m.hitTestStatusPartyIcons(); ok {
		if idx != m.statusCharIndex {
			m.game.Audio.PlaySEByKey("cursor")
		}
		m.statusCharIndex = idx
		m.game.LastStatusCharIndex = m.statusCharIndex
	}

	leftArrowX, rightArrowX := statusPartyArrowX(m.game)
	if hitTestLeftRightArrow(leftArrowX, statusPartyIconY) {
		m.statusCharIndex = (m.statusCharIndex - 1 + partySize) % partySize
		m.game.LastStatusCharIndex = m.statusCharIndex
		m.game.Audio.PlaySEByKey("cursor")
	}
	if hitTestLeftRightArrow(rightArrowX, statusPartyIconY) {
		m.statusCharIndex = (m.statusCharIndex + 1) % partySize
		m.game.LastStatusCharIndex = m.statusCharIndex
		m.game.Audio.PlaySEByKey("cursor")
	}
	if unrelatedTapOutsideRects(menuMainContentRect()) {
		m.game.Audio.PlaySEByKey("cancel")
		m.menuState = m.statusBackState()
	}
}

func (m *MenuScene) enterSlotScreen(save bool) {
	m.saveMode = save
	if m.slotIndex < 0 || m.slotIndex >= maxSaveSlots {
		m.slotIndex = 0
	}
	m.slotScrollTop = clampSlotScrollTop(0, m.slotIndex)
	m.slotDragAccum = 0
	for i := 0; i < maxSaveSlots; i++ {
		m.slotData[i], _ = LoadGame(i + 1)
		m.slotThumbs[i] = LoadThumb(i + 1)
	}
	m.resetSlotDrag()
	if save {
		m.menuState = menuStateSaveSlot
	} else {
		m.menuState = menuStateLoadSlot
	}
}

func (m *MenuScene) resetSlotDrag() {
	m.slotDrag = dragScrollState{suppressUntilRelease: true}
}

func (m *MenuScene) reloadSlotData() {
	for i := 0; i < maxSaveSlots; i++ {
		m.slotData[i], _ = LoadGame(i + 1)
		m.slotThumbs[i] = LoadThumb(i + 1)
	}
}

func (m *MenuScene) updateSlot() {
	if isEscapePressed() {
		m.game.Audio.PlaySEByKey("cancel")
		m.menuState = menuStateMain
		return
	}
	if isMenuUpRepeat() {
		m.slotIndex = (m.slotIndex - 1 + maxSaveSlots) % maxSaveSlots
		m.slotScrollTop = clampSlotScrollTop(m.slotScrollTop, m.slotIndex)
		m.slotDragAccum = 0
		m.game.Audio.PlaySEByKey("cursor")
	}
	if isMenuDownRepeat() {
		m.slotIndex = (m.slotIndex + 1) % maxSaveSlots
		m.slotScrollTop = clampSlotScrollTop(m.slotScrollTop, m.slotIndex)
		m.slotDragAccum = 0
		m.game.Audio.PlaySEByKey("cursor")
	}

	cardH := float64(m.game.SaveThumbFrameImg.Bounds().Dy())

	barX, barY, barW, barH := slotScrollBarRect(slotCardStartX, slotCardStartY)
	if scrollBarMoveAmt := m.slotScrollBarDrag.step(barX, barY, barW, barH); scrollBarMoveAmt != 0 {
		if moveRange := slotScrollBarMoveRange(); moveRange > 0 {
			m.slotScrollBarDrag.stepScrollTop(-scrollBarMoveAmt/moveRange*float64(maxSlotScrollTop)*cardH, cardH, &m.slotScrollTop, maxSlotScrollTop, &m.slotDragAccum)
		}
	}
	scrollBarHeld := m.slotScrollBarDrag.active

	if _, wheelY := ebiten.Wheel(); wheelY != 0 && !scrollBarHeld {
		m.slotDrag.stepScrollTop(wheelY*wheelScrollPxPerNotch, cardH, &m.slotScrollTop, maxSlotScrollTop, &m.slotDragAccum)
	}

	dragArea := cardH * slotsPerPageView
	dragRight := slotScrollBarX(slotCardStartX) - scrollBarTouchPad
	moveAmt := m.slotDrag.step(menuFrameDividerX, slotCardStartY, dragRight-menuFrameDividerX, dragArea)

	tapped := false
	if m.slotDrag.justTapped {
		if idx, ok := hitTestSlotList(m.game, m.slotDrag.tapX, m.slotDrag.tapY, m.slotScrollTop, slotCardStartX, slotCardStartY); ok {
			m.slotIndex = idx
			tapped = true
		}
	}

	if !scrollBarHeld {
		m.slotDrag.stepScrollTop(moveAmt, cardH, &m.slotScrollTop, maxSlotScrollTop, &m.slotDragAccum)
	}

	if !isConfirmKeyPressed() && !tapped {
		cardW := float64(m.game.SaveThumbFrameImg.Bounds().Dx())
		cardRect := tapRect{x: slotCardStartX, y: slotCardStartY, w: cardW, h: dragArea}
		barRect := tapRect{x: barX, y: barY, w: barW, h: barH}
		if unrelatedTapOutsideRects(cardRect, barRect) {
			m.game.Audio.PlaySEByKey("cancel")
			m.menuState = menuStateMain
		}
		return
	}

	slot := m.slotIndex + 1
	m.game.LastSlotIndex = m.slotIndex

	if m.saveMode {
		if _, ok := m.backScene.(*FieldScene); !ok {
			m.game.Audio.PlaySEByKey("error")
			m.showNotice("ここではセーブできません")
			return
		}
		m.game.Audio.PlaySEByKey("decide")
		m.pendingSlot = slot
		m.confirmIndex = 0
		m.menuState = menuStateSaveConfirm
		lockDialogInput(&m.inputLockTicks)
	} else {
		d := m.slotData[m.slotIndex]
		if d == nil {
			m.game.Audio.PlaySEByKey("error")
			m.showNotice("このスロットにはセーブデータがありません")
			return
		}
		m.game.Audio.PlaySEByKey("decide")
		m.pendingSlot = slot
		m.confirmIndex = 0
		m.menuState = menuStateLoadConfirm
		lockDialogInput(&m.inputLockTicks)
	}
}

func (m *MenuScene) saveConfirmMessage() string {
	if i := m.pendingSlot - 1; i >= 0 && i < maxSaveSlots && m.slotData[i] != nil {
		return "上書きセーブしますか？"
	}
	return "セーブしますか？"
}

func (m *MenuScene) updateSaveConfirm() {
	switch m.pollConfirmDialog() {
	case confirmPending:
		return
	case confirmNo:
		m.resetSlotDrag()
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
	lockDialogInput(&m.inputLockTicks)
}

func (m *MenuScene) updateReturnTitleConfirm() {
	switch m.pollConfirmDialog() {
	case confirmPending:
		return
	case confirmNo:
		m.showReturnTitleConfirm = false
		return
	}
	m.nextScene = NewTitleScene(m.game)
}

func (m *MenuScene) updateLoadConfirm() {
	switch m.pollConfirmDialog() {
	case confirmPending:
		return
	case confirmNo:
		m.resetSlotDrag()
		m.menuState = menuStateLoadSlot
		return
	}

	d := m.slotData[m.pendingSlot-1]
	if d == nil {
		m.resetSlotDrag()
		m.menuState = menuStateLoadSlot
		m.showNotice("このスロットにはセーブデータがありません")
		return
	}
	m.game.TotalPlayTime = d.PlayTime
	m.game.PlayerHP = d.PlayerHP
	m.game.PlayerMaxHP = d.PlayerMaxHP
	m.game.PlayerMP = d.PlayerMP
	m.game.PlayerMaxMP = d.PlayerMaxMP
	m.game.PlayerAtk = d.PlayerAtk
	m.game.PlayerMagicAtk = d.PlayerMagicAtk
	m.game.PlayerDef = d.PlayerDef
	m.game.PlayerMagicDef = d.PlayerMagicDef
	m.game.PlayerSpd = d.PlayerSpd
	m.game.PlayerLuck = d.PlayerLuck
	m.game.PlayerSP = d.PlayerSP
	m.game.PlayerSkillLv = d.PlayerSkillLv
	m.game.BossDefeatedFlags = d.BossDefeatedFlags
	m.game.PlayerLv = d.PlayerLv
	m.game.PlayerEXP = d.PlayerEXP
	m.game.PlayerNextEXP = d.PlayerNextEXP
	m.game.Inventory = d.Inventory
	m.game.OpenedChests = d.OpenedChests
	m.game.UnlockedWalls = d.UnlockedWalls
	m.game.Keys = d.Keys
	m.game.RaisedLevers = d.RaisedLevers
	m.game.SeenAutoHealMapIntro = d.SeenAutoHealMapIntro
	m.game.BlockPositions = d.BlockPositions
	m.game.UnlockedBlockDoors = d.UnlockedBlockDoors

	field, err := NewRoomScene(m.game, d.CurrentMap, d.PlayerX, d.PlayerY, "", d.PlayerDir)
	if err != nil {
		m.resetSlotDrag()
		m.menuState = menuStateLoadSlot
		m.showNotice("ロードに失敗しました")
		return
	}
	m.game.ChangeSceneWithFade(field, fadeTimeContinue)
}

func (m *MenuScene) updateSaveDone() {
	if !m.pollMessageDialog() {
		return
	}
	m.reloadSlotData()
	m.slotIndex = m.pendingSlot - 1
	m.resetSlotDrag()
	m.menuState = menuStateSaveSlot
}
