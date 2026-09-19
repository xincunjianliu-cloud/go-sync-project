package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	optionIdxMaster = iota
	optionIdxBGM
	optionIdxSE
	optionIdxDisplayMode
	optionIdxMessageSpeed
	optionIdxCursorMemory
	optionIdxReset
	optionCount
)

func isVolumeOptionRow(idx int) bool {
	return idx == optionIdxBGM || idx == optionIdxSE || idx == optionIdxMaster
}

func optionRowPositions() (masterBarY, bgmBarY, seBarY, displayRowY, speedRowY, descRowY, sysHeaderY, cursorRowY, resetY float64) {
	masterBarY = volumePanelY
	bgmBarY = masterBarY + volumeRowGapY
	seBarY = bgmBarY + volumeRowGapY
	dispHeaderY := seBarY + volumeBarH + 25
	lineY2 := dispHeaderY + 24
	displayRowY = lineY2 + 33
	speedRowY = displayRowY + 37
	descRowY = speedRowY + 27
	sysHeaderY = descRowY + 37
	lineY4 := sysHeaderY + 24
	cursorRowY = lineY4 + 33
	resetY = cursorRowY + volumeResetGapY
	return
}

// optionBoxCenterX はメニュー枠の縦区切り線から右端までの、右側ボックスの
// 水平中央。オプション画面の各項目はここを基準に中央揃えする。
const optionBoxCenterX = (menuFrameDividerX + menuFrameRight) / 2

// controlRowLabelValueOffset は画面モード/メッセージ速度/カーソル記憶の各行で
// ラベル起点(optionCtrlLabelX)から値の中心(optionValueX)までの水平距離。
const controlRowLabelValueOffset float64 = volumeLabelBarGap + 180

// optionArrowGap は画面モード/メッセージ速度/カーソル記憶3行共通の、
// 値中心(optionValueX)から◀▶矢印までの距離。3行とも同じ値にすることで
// 矢印が縦一列に揃う。3つの値のうち最も幅が広い「フルスクリーン」
// （約126px）が矢印と重ならない最小限の余白を基準にしている。
const (
	optionArrowGap        float64 = 82
	optionDisplayArrowGap float64 = optionArrowGap
	optionSpeedArrowGap   float64 = optionArrowGap
	optionCursorArrowGap  float64 = optionArrowGap
)

// controlRowHalfWidth はラベル起点から右矢印(3行のうち最大到達幅)までを
// 含めた行全体の半幅。これでラベル起点をボックス中央に対して逆算する。
const controlRowHalfWidth float64 = (controlRowLabelValueOffset + optionDisplayArrowGap) / 2

const optionCtrlLabelX = optionBoxCenterX - controlRowHalfWidth
const optionValueX = optionCtrlLabelX + controlRowLabelValueOffset

func (m *MenuScene) hitTestOptionList() (int, bool) {
	masterBarY, bgmBarY, seBarY, displayRowY, speedRowY, _, _, cursorRowY, resetY := optionRowPositions()
	volumeRects := []tapRect{
		{x: volumeGroupX - 4, y: masterBarY + volumeBarH/2 - 16, w: volumeLabelBarGap + volumeBarW + 4, h: 32},
		{x: volumeGroupX - 4, y: bgmBarY + volumeBarH/2 - 16, w: volumeLabelBarGap + volumeBarW + 4, h: 32},
		{x: volumeGroupX - 4, y: seBarY + volumeBarH/2 - 16, w: volumeLabelBarGap + volumeBarW + 4, h: 32},
	}
	ctrlYs := []float64{displayRowY, speedRowY, cursorRowY}
	ctrlRects := make([]tapRect, len(ctrlYs))
	for i, y := range ctrlYs {
		ctrlRects[i] = tapRect{x: optionCtrlLabelX - 4, y: y - 16, w: controlRowHalfWidth*2 + 4, h: 32}
	}
	resetRect := tapRect{x: volumeGroupX - 4, y: resetY + 10 - 16, w: volumeLabelBarGap + volumeBarW + 4, h: 32}

	rects := append(append(volumeRects, ctrlRects...), resetRect)
	return hitTestTapRects(rects)
}

func optionPanelRect() tapRect {
	_, _, _, _, _, _, _, _, resetY := optionRowPositions()
	top := volumePanelY - 40
	return tapRect{
		x: volumeGroupX - 10,
		y: top,
		w: volumeContentW + 10,
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
		m.game.Audio.PlaySEByKey("cancel")
		m.finishVolumeInput()
		m.menuState = menuStateMain
		return
	}

	if m.updateVolumeBarDrag() {
		m.optionIndex = m.volumeDragRow
		return
	}

	if idx, dir, ok := m.hitTestOptionArrows(); ok {
		m.optionIndex = idx
		m.changeOptionValue(idx, dir)
		m.game.Audio.PlaySEByKey("cursor")
		return
	}

	if isMenuUpRepeat() {
		m.optionIndex = (m.optionIndex - 1 + optionCount) % optionCount
		m.game.Audio.PlaySEByKey("cursor")
	}
	if isMenuDownRepeat() {
		m.optionIndex = (m.optionIndex + 1) % optionCount
		m.game.Audio.PlaySEByKey("cursor")
	}
	tappedIdx, tappedOk := m.hitTestOptionList()
	tapConfirm := tapSelectOrConfirm(tappedIdx, tappedOk, &m.optionIndex, m.game.Audio)

	if isVolumeOptionRow(m.optionIndex) {
		m.updateVolumeKeys()
	} else {
		m.finishVolumeInput()
		if isMenuRightPressed() {
			if m.changeOptionValue(m.optionIndex, +1) {
				m.game.Audio.PlaySEByKey("cursor")
			} else {
				m.game.Audio.PlaySEByKey("error")
			}
		}
		if isMenuLeftPressed() {
			if m.changeOptionValue(m.optionIndex, -1) {
				m.game.Audio.PlaySEByKey("cursor")
			} else {
				m.game.Audio.PlaySEByKey("error")
			}
		}
	}

	if isConfirmKeyPressed() || tapConfirm {
		// 値の変更は矢印(キー/タップ)とボリュームのドラッグのみで行う。
		// 確定キーは「リセット」以外のどの行でも値を変えない。
		if m.activateOption(m.optionIndex) {
			m.game.Audio.PlaySEByKey("decide")
		}
		return
	}

	if !tappedOk && unrelatedTapOutsideRects(optionPanelRect()) {
		m.game.Audio.PlaySEByKey("cancel")
		m.finishVolumeInput()
		m.menuState = menuStateMain
	}
}

func (m *MenuScene) hitTestOptionArrows() (int, int, bool) {
	_, _, _, displayRowY, speedRowY, _, _, cursorRowY, _ := optionRowPositions()
	type arrow struct {
		idx, dir int
		x, y     float64
		visible  bool
	}
	arrows := []arrow{
		{optionIdxDisplayMode, -1, optionValueX - optionDisplayArrowGap, displayRowY, true},
		{optionIdxDisplayMode, +1, optionValueX + optionDisplayArrowGap, displayRowY, true},
		{optionIdxMessageSpeed, -1, optionValueX - optionSpeedArrowGap, speedRowY, m.game.MessageSpeed > 0},
		{optionIdxMessageSpeed, +1, optionValueX + optionSpeedArrowGap, speedRowY, m.game.MessageSpeed < 2},
		{optionIdxCursorMemory, -1, optionValueX - optionCursorArrowGap, cursorRowY, true},
		{optionIdxCursorMemory, +1, optionValueX + optionCursorArrowGap, cursorRowY, true},
	}
	for _, a := range arrows {
		if a.visible && hitTestLeftRightArrow(a.x, a.y) {
			return a.idx, a.dir, true
		}
	}
	return 0, 0, false
}

// changeOptionValue は画面サイズ/メッセージ速度/カーソル記憶の値をdir方向に
// 1段階変える。値が実際に変化した場合はtrueを返す。メッセージ速度は上限・
// 下限で止まり(ループしない)、これは確定キーでの変更(activateOption)とも
// 挙動を揃えるためのもの。
func (m *MenuScene) changeOptionValue(idx, dir int) bool {
	switch idx {
	case optionIdxDisplayMode:
		m.toggleDisplayMode()
		return true
	case optionIdxMessageSpeed:
		next := m.game.MessageSpeed + dir
		if next < 0 || next > 2 {
			return false
		}
		m.game.MessageSpeed = next
		m.previewTicks = 0
		m.persistSettings()
		return true
	case optionIdxCursorMemory:
		m.game.RememberCursor = !m.game.RememberCursor
		m.persistSettings()
		return true
	}
	return false
}

// activateOption は確定キー(Enter/Z等)でoptionIndexの行を確定した時の処理。
// 値の変更は矢印(キー/タップ)とボリュームのドラッグのみで行うため、確定キーで
// 実際に何かが起こるのは「リセット」だけ。戻り値はSE切り替え(decide/無音)の
// 判定に使われる。
func (m *MenuScene) activateOption(idx int) bool {
	if idx != optionIdxReset {
		return false
	}
	m.finishVolumeInput()
	m.confirmIndex = 1
	m.menuState = menuStateOptionResetConfirm
	lockDialogInput(&m.inputLockTicks)
	return true
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
		m.game.Audio.SetSEVolume(defaultSEVolume)
		m.game.Audio.SetMasterVolume(defaultMasterVolume)
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

// volumeRowSetter はidx(optionIdxBGM/SE/Master)に対応する音量セッターを返す。
func (m *MenuScene) volumeRowSetter(idx int) func(float64) {
	if m.game.Audio == nil {
		return nil
	}
	switch idx {
	case optionIdxBGM:
		return m.game.Audio.SetVolume
	case optionIdxSE:
		return m.game.Audio.SetSEVolume
	case optionIdxMaster:
		return m.game.Audio.SetMasterVolume
	}
	return nil
}

// volumeRowValue はidx(optionIdxBGM/SE/Master)に対応する現在の音量を返す。
func (m *MenuScene) volumeRowValue(idx int) float64 {
	if m.game.Audio == nil {
		return 0
	}
	switch idx {
	case optionIdxBGM:
		return m.game.Audio.volume
	case optionIdxSE:
		return m.game.Audio.seVolume
	case optionIdxMaster:
		return m.game.Audio.masterVolume
	}
	return 0
}

func (m *MenuScene) updateVolumeBarDrag() bool {
	masterBarY, bgmBarY, seBarY, _, _, _, _, _, _ := optionRowPositions()
	barX := volumeGroupX + volumeLabelBarGap
	rows := []struct {
		idx int
		y   float64
	}{
		{optionIdxMaster, masterBarY},
		{optionIdxBGM, bgmBarY},
		{optionIdxSE, seBarY},
	}

	var pt touchPoint
	found := false
	if m.volumeDragActive {
		if pts := activeTouchPoints(); len(pts) > 0 {
			pt, found = pts[0], true
		}
	} else {
		const hitHalfH = 16.0
		hitX := barX - volumeKnobRSel
		hitW := volumeBarW + volumeKnobRSel*2
		for _, p := range justPressedTouchPoints() {
			for _, row := range rows {
				midY := row.y + volumeBarH/2
				if p.inRect(hitX, midY-hitHalfH, hitW, hitHalfH*2) {
					pt, found = p, true
					m.volumeDragRow = row.idx
					break
				}
			}
			if found {
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
	if set := m.volumeRowSetter(m.volumeDragRow); set != nil {
		set(ratio)
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
		m.changeRowVolume(m.optionIndex, bgmVolumeStep)
	}
	if repeatFires(m.volumeLeftHoldTicks, bgmVolumeRepeatDelayTicks, bgmVolumeRepeatIntervalTicks) {
		m.changeRowVolume(m.optionIndex, -bgmVolumeStep)
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

func (m *MenuScene) changeRowVolume(idx int, delta float64) {
	set := m.volumeRowSetter(idx)
	if set == nil {
		return
	}
	set(clampVolume(m.volumeRowValue(idx) + delta))
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
	optionIdxSE:           "効果音の音量を調整します（←→ / バーをタップ・ドラッグ）",
	optionIdxMaster:       "ゲーム全体の音量を調整します（←→ / バーをタップ・ドラッグ）",
	optionIdxDisplayMode:  "フルスクリーン/ウィンドウを切り替えます（ウィンドウは端をドラッグしてサイズ変更できます）",
	optionIdxMessageSpeed: "メッセージの表示速度を変更します（←→）",
	optionIdxCursorMemory: "ONにすると、次にこのメニューを開いた時も前回選んでいた項目にカーソルが合った状態にします",
	optionIdxReset:        "設定をすべて初期値に戻します",
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
