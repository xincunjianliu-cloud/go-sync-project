package main

import "slices"

type confirmResult int

const (
	confirmPending confirmResult = iota
	confirmYes
	confirmNo
)

func confirmDialogRect() tapRect {
	return tapRect{
		x: float64(gameWidth)/2 - confirmPanelW/2 + confirmImageOffsetX,
		y: float64(gameHeight)/2 - confirmPanelH/2 + confirmImageOffsetY,
		w: confirmPanelW,
		h: confirmPanelH,
	}
}

func (m *MenuScene) pollConfirmDialog() confirmResult {
	if consumeDialogInputLock(&m.inputLockTicks) {
		return confirmPending
	}
	if isEscapePressed() {
		m.game.Audio.PlaySEByKey("cancel")
		return confirmNo
	}
	if isMenuUpPressed() || isMenuDownPressed() {
		m.confirmIndex = 1 - m.confirmIndex
		m.game.Audio.PlaySEByKey("cursor")
	}
	if idx, ok := hitTestConfirmDialog(confirmImageOffsetX); ok {
		m.confirmIndex = idx
		if idx == 0 {
			m.game.Audio.PlaySEByKey("decide")
			return confirmYes
		}
		m.game.Audio.PlaySEByKey("cancel")
		return confirmNo
	}
	if pts := justPressedTouchPoints(); len(pts) > 0 {
		if !confirmDialogRect().contains(pts[0]) {
			m.game.Audio.PlaySEByKey("cancel")
			return confirmNo
		}
		return confirmPending
	}
	if isConfirmKeyPressed() {
		if m.confirmIndex == 0 {
			m.game.Audio.PlaySEByKey("decide")
			return confirmYes
		}
		m.game.Audio.PlaySEByKey("cancel")
		return confirmNo
	}
	return confirmPending
}

func (m *MenuScene) pollMessageDialog() bool {
	if consumeDialogInputLock(&m.inputLockTicks) {
		return false
	}
	pressed := isEscapePressed() || isConfirmKeyPressed() || len(justPressedTouchPoints()) > 0
	if pressed {
		m.game.Audio.PlaySEByKey("decide")
	}
	return pressed
}

func (m *MenuScene) isModalMenuState() bool {
	if m.showReturnTitleConfirm {
		return true
	}
	switch m.menuState {
	case menuStateSaveConfirm, menuStateLoadConfirm, menuStateSaveDone,
		menuStateOptionResetConfirm, menuStateOptionResetDone:
		return true
	}
	return false
}

func tapInsideRect(r tapRect) bool {
	return slices.ContainsFunc(justPressedTouchPoints(), r.contains)
}
