package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	optionIdxBGM = iota
	optionIdxDisplayMode
	optionIdxMessageSpeed
	optionIdxCursorMemory
	optionIdxReset
	optionCount
)

func optionRowPositions() (barY, displayRowY, speedRowY, cursorRowY, resetY float64) {
	barY = volumePanelY
	dispHeaderY := barY + volumeBarH + 30
	lineY2 := dispHeaderY + 24
	displayRowY = lineY2 + 40
	speedRowY = displayRowY + 40
	descRowY := speedRowY + 30
	lineY3 := descRowY + 20
	cursorRowY = lineY3 + 30
	resetY = cursorRowY + volumeResetGapY
	return
}

const optionValueX = volumeGroupX + volumeLabelBarGap + 180

func (m *MenuScene) hitTestOptionList() (int, bool) {
	barY, displayRowY, speedRowY, cursorRowY, resetY := optionRowPositions()
	ys := []float64{barY + volumeBarH/2, displayRowY, speedRowY, cursorRowY, resetY + 10}
	rects := make([]tapRect, len(ys))
	for i, y := range ys {
		rects[i] = tapRect{x: volumeGroupX - 4, y: y - 16, w: volumeLabelBarGap + volumeBarW + 4, h: 32}
	}
	return hitTestTapRects(rects)
}

func optionPanelRect() tapRect {
	_, _, _, _, resetY := optionRowPositions()
	top := volumePanelY - 40
	return tapRect{
		x: volumeGroupX - 10,
		y: top,
		w: volumeLabelBarGap + volumeBarW + volumeBarPercentGap + 80,
		h: resetY + 40 - top,
	}
}

var uiMobileArrowsEnabled bool

func arrowHitRect(x, y float64) (rx, ry, rw, rh float64) {
	halfW, halfH := 14.0, 16.0
	if uiMobileArrowsEnabled {
		halfW, halfH = 22.0, 22.0
	}
	return x - halfW, y - halfH, halfW * 2, halfH * 2
}

func hitTestLeftRightArrow(x, y float64) bool {
	rx, ry, rw, rh := arrowHitRect(x, y)
	_, ok := hitTestTapRects([]tapRect{{x: rx, y: ry, w: rw, h: rh}})
	return ok
}

func (m *MenuScene) updateOption() {
	if isEscapePressed() {
		m.finishVolumeInput()
		m.menuState = menuStateMain
		return
	}

	if m.updateVolumeBarDrag() {
		m.optionIndex = optionIdxBGM
		return
	}

	if idx, dir, ok := m.hitTestOptionArrows(); ok {
		m.optionIndex = idx
		m.changeOptionValue(idx, dir)
		return
	}

	if isMenuUpRepeat() {
		m.optionIndex = (m.optionIndex - 1 + optionCount) % optionCount
	}
	if isMenuDownRepeat() {
		m.optionIndex = (m.optionIndex + 1) % optionCount
	}
	tappedIdx, tappedOk := m.hitTestOptionList()
	tapConfirm := tapSelectOrConfirm(tappedIdx, tappedOk, &m.optionIndex)

	if m.optionIndex == optionIdxBGM {
		m.updateVolumeKeys()
	} else {
		m.finishVolumeInput()
		if isMenuRightPressed() {
			m.changeOptionValue(m.optionIndex, +1)
		}
		if isMenuLeftPressed() {
			m.changeOptionValue(m.optionIndex, -1)
		}
	}

	if isConfirmKeyPressed() || tapConfirm {
		m.activateOption(m.optionIndex)
		return
	}

	if !tappedOk && unrelatedTapOutsideRects(optionPanelRect()) {
		m.finishVolumeInput()
		m.menuState = menuStateMain
	}
}

func (m *MenuScene) hitTestOptionArrows() (int, int, bool) {
	_, displayRowY, speedRowY, cursorRowY, _ := optionRowPositions()
	type arrow struct {
		idx, dir int
		x, y     float64
		visible  bool
	}
	arrows := []arrow{
		{optionIdxDisplayMode, -1, optionValueX - 70, displayRowY, true},
		{optionIdxDisplayMode, +1, optionValueX + 70, displayRowY, true},
		{optionIdxMessageSpeed, -1, optionValueX - 60, speedRowY, m.game.MessageSpeed > 0},
		{optionIdxMessageSpeed, +1, optionValueX + 60, speedRowY, m.game.MessageSpeed < 2},
		{optionIdxCursorMemory, -1, optionValueX - 70, cursorRowY, true},
		{optionIdxCursorMemory, +1, optionValueX + 70, cursorRowY, true},
	}
	for _, a := range arrows {
		if a.visible && hitTestLeftRightArrow(a.x, a.y) {
			return a.idx, a.dir, true
		}
	}
	return 0, 0, false
}

func (m *MenuScene) changeOptionValue(idx, dir int) {
	switch idx {
	case optionIdxDisplayMode:
		m.toggleDisplayMode()
	case optionIdxMessageSpeed:
		next := m.game.MessageSpeed + dir
		if next < 0 || next > 2 {
			return
		}
		m.game.MessageSpeed = next
		m.previewTicks = 0
		m.persistSettings()
	case optionIdxCursorMemory:
		m.game.RememberCursor = !m.game.RememberCursor
		m.persistSettings()
	}
}

func (m *MenuScene) activateOption(idx int) {
	switch idx {
	case optionIdxBGM:
	case optionIdxDisplayMode:
		m.toggleDisplayMode()
	case optionIdxMessageSpeed:
		m.game.MessageSpeed = (m.game.MessageSpeed + 1) % 3
		m.previewTicks = 0
		m.persistSettings()
	case optionIdxCursorMemory:
		m.game.RememberCursor = !m.game.RememberCursor
		m.persistSettings()
	case optionIdxReset:
		m.finishVolumeInput()
		m.confirmIndex = 1
		m.menuState = menuStateOptionResetConfirm
		lockDialogInput(&m.inputLockTicks)
	}
}

func (m *MenuScene) toggleDisplayMode() {
	m.game.Fullscreen = !m.game.Fullscreen
	w, h := m.game.WindowWidth, m.game.WindowHeight
	if w <= 0 || h <= 0 {
		w, h = defaultWindowWidth, defaultWindowHeight
	}
	applyDisplayMode(m.game.Fullscreen, w, h)
	if !m.game.Fullscreen {
		m.game.WindowWidth, m.game.WindowHeight = ebiten.WindowSize()
		m.game.lastWindowW, m.game.lastWindowH = m.game.WindowWidth, m.game.WindowHeight
	}
	m.persistSettings()
}

func (m *MenuScene) performOptionReset() {
	if m.game.Audio != nil {
		m.game.Audio.SetVolume(defaultBGMVolume)
	}
	m.game.MessageSpeed = defaultMessageSpeed
	m.game.Fullscreen = defaultFullscreen
	m.game.WindowWidth = defaultWindowWidth
	m.game.WindowHeight = defaultWindowHeight
	m.game.RememberCursor = defaultRememberCursor
	applyDisplayMode(defaultFullscreen, defaultWindowWidth, defaultWindowHeight)
	m.game.lastWindowW, m.game.lastWindowH = defaultWindowWidth, defaultWindowHeight
	m.persistSettings()
}

func (m *MenuScene) updateOptionResetConfirm() {
	switch m.pollConfirmDialog() {
	case confirmPending:
		return
	case confirmNo:
		m.menuState = menuStateOption
		return
	}
	m.performOptionReset()
	m.menuState = menuStateOptionResetDone
	lockDialogInput(&m.inputLockTicks)
}

func (m *MenuScene) updateOptionResetDone() {
	if m.pollMessageDialog() {
		m.menuState = menuStateOption
	}
}

func (m *MenuScene) updateVolumeBarDrag() bool {
	barY, _, _, _, _ := optionRowPositions()
	barX := volumeGroupX + volumeLabelBarGap

	var pt touchPoint
	found := false
	if m.volumeDragActive {
		if pts := activeTouchPoints(); len(pts) > 0 {
			pt, found = pts[0], true
		}
	} else {
		for _, p := range justPressedTouchPoints() {
			if p.inRect(barX, barY-8, volumeBarW, volumeBarH+16) {
				pt, found = p, true
				break
			}
		}
	}

	if !found {
		if m.volumeDragActive {
			m.volumeDragActive = false
			m.persistSettings()
		}
		return false
	}

	m.volumeDragActive = true
	ratio := (pt.x - barX) / volumeBarW
	if ratio < 0 {
		ratio = 0
	} else if ratio > 1 {
		ratio = 1
	}
	if m.game.Audio != nil {
		m.game.Audio.SetVolume(ratio)
	}
	return true
}

func (m *MenuScene) updateVolumeKeys() {
	rightHeld := ebiten.IsKeyPressed(ebiten.KeyRight) || ebiten.IsKeyPressed(ebiten.KeyD)
	leftHeld := ebiten.IsKeyPressed(ebiten.KeyLeft) || ebiten.IsKeyPressed(ebiten.KeyA)
	wasHeld := m.volumeRightHoldTicks > 0 || m.volumeLeftHoldTicks > 0

	if rightHeld {
		m.volumeRightHoldTicks++
	} else {
		m.volumeRightHoldTicks = 0
	}
	if leftHeld {
		m.volumeLeftHoldTicks++
	} else {
		m.volumeLeftHoldTicks = 0
	}

	if repeatFires(m.volumeRightHoldTicks, bgmVolumeRepeatDelayTicks, bgmVolumeRepeatIntervalTicks) {
		m.changeBGMVolume(bgmVolumeStep)
	}
	if repeatFires(m.volumeLeftHoldTicks, bgmVolumeRepeatDelayTicks, bgmVolumeRepeatIntervalTicks) {
		m.changeBGMVolume(-bgmVolumeStep)
	}
	if wasHeld && !rightHeld && !leftHeld {
		m.persistSettings()
	}
}

func (m *MenuScene) finishVolumeInput() {
	if m.volumeDragActive || m.volumeRightHoldTicks > 0 || m.volumeLeftHoldTicks > 0 {
		m.persistSettings()
	}
	m.volumeDragActive = false
	m.volumeRightHoldTicks = 0
	m.volumeLeftHoldTicks = 0
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
	"スキル":     "スキルの確認・強化、回復スキルの使用ができます",
	"ステータス":   "キャラクターの詳細なステータスを確認します",
	"セーブ":     "現在の状況をセーブします",
	"ロード":     "セーブデータをロードします",
	"オプション":   "ゲームを遊びやすいように設定できます",
	"タイトルに戻る": "タイトル画面に戻ります（未セーブの進行は失われます）",
}

var menuOptionDescriptions = map[int]string{
	optionIdxBGM:          "BGMの音量を調整します（←→ / バーをタップ・ドラッグ）",
	optionIdxDisplayMode:  "フルスクリーン/ウィンドウを切り替えます（ウィンドウは端をドラッグしてサイズ変更できます）",
	optionIdxMessageSpeed: "メッセージの表示速度を変更します（←→）",
	optionIdxCursorMemory: "ONにすると、次にこのメニューを開いた時も前回選んでいた項目にカーソルが合った状態にします",
	optionIdxReset:        "設定をすべて初期値に戻します",
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

var skillShortDescriptions = map[string]string{
	"回復": "対象を回復する",
}

func skillShortDescription(skillName string) string {
	if d, ok := skillShortDescriptions[skillName]; ok {
		return d
	}
	return skillName
}

func (m *MenuScene) persistSettings() {
	m.game.SaveGameSettings()
}
