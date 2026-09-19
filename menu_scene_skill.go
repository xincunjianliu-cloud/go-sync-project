package main

import "fmt"

func (m *MenuScene) hitTestSkillNameRows(rowCount int) (int, bool) {
	rects := make([]tapRect, rowCount)
	for i := 0; i < rowCount; i++ {
		y := skillRowStartY + float64(i)*skillRowGapY
		rects[i] = tapRect{
			x: skillNameX - 4,
			y: y - skillRowGapY/2,
			w: skillLevelStartX - skillNameX,
			h: skillRowGapY,
		}
	}
	return hitTestTapRects(rects)
}

func (m *MenuScene) skillLevelCellRect(rowIndex, lv int) tapRect {
	y := skillRowStartY + float64(rowIndex)*skillRowGapY
	x := skillLevelStartX + float64(lv-1)*skillLevelGapX
	return tapRect{x: x - skillLevelGapX/2, y: y - skillRowGapY/2, w: skillLevelGapX, h: skillRowGapY}
}

func (m *MenuScene) hitTestAnySkillLevelCell(skills []SkillDef) (int, int, bool) {
	var rects []tapRect
	var rows, lvs []int
	for row, sk := range skills {
		for lv := 1; lv <= len(sk.Levels); lv++ {
			rects = append(rects, m.skillLevelCellRect(row, lv))
			rows = append(rows, row)
			lvs = append(lvs, lv)
		}
	}
	idx, ok := hitTestTapRects(rects)
	if !ok {
		return 0, 0, false
	}
	return rows[idx], lvs[idx], true
}

func (m *MenuScene) isSkillLevelCellHeld(rowIndex, lv int) bool {
	r := m.skillLevelCellRect(rowIndex, lv)
	for _, p := range activeTouchPoints() {
		if r.contains(p) {
			return true
		}
	}
	return false
}

func (m *MenuScene) reachableSkillLevel(charIdx, skillIdx int) int {
	skills := m.game.CharacterSkills(charIdx)
	if skillIdx < 0 || skillIdx >= len(skills) {
		return 1
	}
	curLv := m.game.PlayerSkillLv[charIdx][skillIdx]
	if curLv < 1 {
		curLv = 1
	}
	maxLv := len(skills[skillIdx].Levels)
	reachable := curLv + 1
	if reachable > maxLv {
		reachable = maxLv
	}
	return reachable
}

func (m *MenuScene) updateSkillCharSel() {
	if isEscapePressed() {
		m.game.Audio.PlaySEByKey("cancel")
		m.menuState = menuStateMain
		return
	}
	if isMenuUpRepeat() {
		m.skillCharIndex = (m.skillCharIndex - 1 + partySize) % partySize
		m.game.LastSkillCharIndex = m.skillCharIndex
		m.game.Audio.PlaySEByKey("cursor")
	}
	if isMenuDownRepeat() {
		m.skillCharIndex = (m.skillCharIndex + 1) % partySize
		m.game.LastSkillCharIndex = m.skillCharIndex
		m.game.Audio.PlaySEByKey("cursor")
	}
	tapped := false
	if idx, ok := m.hitTestPartyRows(); ok {
		if idx != m.skillCharIndex {
			m.game.Audio.PlaySEByKey("cursor")
		}
		m.skillCharIndex = idx
		m.game.LastSkillCharIndex = m.skillCharIndex
		tapped = true
	}
	if !isConfirmKeyPressed() && !tapped {
		if unrelatedTapOutsideRects(menuMainContentRect()) {
			m.game.Audio.PlaySEByKey("cancel")
			m.menuState = menuStateMain
		}
		return
	}
	m.game.Audio.PlaySEByKey("decide")
	m.enterSkillCharacter(m.skillCharIndex)
}

func (m *MenuScene) enterSkillCharacter(idx int) {
	m.skillCharIndex = idx
	m.game.LastSkillCharIndex = idx

	skills := m.game.CharacterSkills(m.skillCharIndex)
	m.skillSubIndex = m.game.rememberedIndex(m.game.LastSkillSubIndex)
	if m.skillSubIndex < 0 || m.skillSubIndex >= len(skills) {
		m.skillSubIndex = 0
	}
	curLv := m.game.PlayerSkillLv[m.skillCharIndex][m.skillSubIndex]
	if curLv < 1 {
		curLv = 1
	}
	m.skillLevelCursor = m.game.rememberedIndex(m.game.LastSkillLevelCursor)
	reachable := m.reachableSkillLevel(m.skillCharIndex, m.skillSubIndex)
	if m.skillLevelCursor < 1 || m.skillLevelCursor > reachable {
		m.skillLevelCursor = curLv
	}

	m.skillLevelSelecting = false
	m.upgradeProgress = 0
	m.upgradeHoldArmed = false
	m.clearNotice()
	m.menuState = menuStateSkillSub
}

func (m *MenuScene) trySkillLevelConfirm(skills []SkillDef, skillIdx, lv int) {
	data := skills[skillIdx].Levels[lv-1]
	caster := m.skillCharIndex
	switch {
	case !data.IsHeal:
		m.game.Audio.PlaySEByKey("error")
	case !m.game.IsSkillUnlocked(caster, skillIdx):
		m.game.Audio.PlaySEByKey("error")
	case m.game.PlayerHP[caster] <= 0:
		m.game.Audio.PlaySEByKey("error")
	case m.game.PlayerMP[caster] < data.MPCost:
		m.game.Audio.PlaySEByKey("error")
	default:
		m.game.Audio.PlaySEByKey("decide")
		m.pendingSkill = skillIdx + 1
		m.pendingSkillLevel = lv
		if m.healTargetIndex < 0 || m.healTargetIndex > partySize {
			m.healTargetIndex = 0
		}
		m.clearNotice()
		m.menuState = menuStateHealTarget
	}
}

func (m *MenuScene) upgradeBlocked(skillIdx int) bool {
	return !m.game.CanUpgradeSkill(m.skillCharIndex, skillIdx)
}

func (m *MenuScene) skillUsable(skillIdx int) bool {
	if skillIdx < 0 || skillIdx >= len(menuSkillDefs) {
		return false
	}
	def := menuSkillDefs[skillIdx]
	if !def.implemented {
		return false
	}
	if def.mpCost > 0 && m.game.PlayerMP[m.skillCharIndex] < def.mpCost {
		return false
	}
	return true
}

func (m *MenuScene) skillDisableReason(skillIdx int) string {
	if skillIdx < 0 || skillIdx >= len(menuSkillDefs) {
		return ""
	}
	def := menuSkillDefs[skillIdx]
	if !def.implemented {
		return ""
	}
	if def.mpCost > 0 && m.game.PlayerMP[m.skillCharIndex] < def.mpCost {
		return "MP不足"
	}
	return ""
}

func (m *MenuScene) nextEnabledSkillIndex(from, dir int) int {
	n := len(menuSkillDefs)
	idx := from
	for i := 0; i < n; i++ {
		idx = (idx + dir + n) % n
		if m.skillUsable(idx) {
			return idx
		}
	}
	return from
}

func (m *MenuScene) updateSkillSub() {
	if idx, ok := m.hitTestPartyRows(); ok {
		if idx != m.skillCharIndex {
			m.game.Audio.PlaySEByKey("decide")
			m.enterSkillCharacter(idx)
		}
		return
	}
	areaTapped := tapInsideRect(menuMainContentRect())

	skills := m.game.CharacterSkills(m.skillCharIndex)
	n := len(skills)
	if n == 0 {
		return
	}

	curLv := m.game.PlayerSkillLv[m.skillCharIndex][m.skillSubIndex]
	if curLv < 1 {
		curLv = 1
	}
	maxLv := len(skills[m.skillSubIndex].Levels)
	reachable := curLv + 1
	if reachable > maxLv {
		reachable = maxLv
	}

	if m.skillLevelSelecting {
		tappedLvConfirm := false
		levelAreaTapped := false
		if rowT, lvT, ok := m.hitTestAnySkillLevelCell(skills); ok {
			levelAreaTapped = true
			rowReachable := m.reachableSkillLevel(m.skillCharIndex, rowT)
			if lvT > rowReachable {
				lvT = rowReachable
			}
			if rowT != m.skillSubIndex {
				m.skillSubIndex = rowT
				m.skillLevelCursor = lvT
				m.upgradeProgress = 0
				m.upgradeHoldArmed = false
				m.game.LastSkillSubIndex = m.skillSubIndex
				m.game.LastSkillLevelCursor = m.skillLevelCursor
				m.game.Audio.PlaySEByKey("cursor")
				return
			}
			if lvT != m.skillLevelCursor {
				m.upgradeProgress = 0
				m.upgradeHoldArmed = false
			}
			tappedLvConfirm = tapSelectOrConfirm(lvT, true, &m.skillLevelCursor, m.game.Audio)
			m.game.LastSkillLevelCursor = m.skillLevelCursor
		} else if rowT, ok := m.hitTestSkillNameRows(n); ok {
			levelAreaTapped = true
			if rowT != m.skillSubIndex {
				m.skillSubIndex = rowT
				newCurLv := m.game.PlayerSkillLv[m.skillCharIndex][rowT]
				if newCurLv < 1 {
					newCurLv = 1
				}
				m.skillLevelCursor = newCurLv
				m.upgradeProgress = 0
				m.upgradeHoldArmed = false
				m.game.LastSkillSubIndex = m.skillSubIndex
				m.game.LastSkillLevelCursor = m.skillLevelCursor
				m.game.Audio.PlaySEByKey("cursor")
				return
			}
		}
		heldTouch := m.isSkillLevelCellHeld(m.skillSubIndex, m.skillLevelCursor)

		if !isConfirmKeyDown() && !heldTouch {
			m.upgradeHoldArmed = true
			m.upgradeProgress = 0
		}
		if isEscapePressed() {
			m.game.Audio.PlaySEByKey("cancel")
			m.skillLevelSelecting = false
			m.upgradeHoldArmed = false
			return
		}
		if unrelatedTapPressed(levelAreaTapped || areaTapped) {
			m.game.Audio.PlaySEByKey("cancel")
			m.skillLevelSelecting = false
			m.upgradeHoldArmed = false
			return
		}
		if up, down := isMenuUpRepeat(), isMenuDownRepeat(); up || down {
			if up {
				m.skillSubIndex = (m.skillSubIndex - 1 + n) % n
			} else {
				m.skillSubIndex = (m.skillSubIndex + 1) % n
			}
			newCurLv := m.game.PlayerSkillLv[m.skillCharIndex][m.skillSubIndex]
			if newCurLv < 1 {
				newCurLv = 1
			}
			m.skillLevelCursor = newCurLv
			m.upgradeProgress = 0
			m.upgradeHoldArmed = false
			m.game.LastSkillSubIndex = m.skillSubIndex
			m.game.LastSkillLevelCursor = m.skillLevelCursor
			m.game.Audio.PlaySEByKey("cursor")
			return
		}
		if isMenuRightPressed() {
			if m.skillLevelCursor < reachable {
				m.skillLevelCursor++
				m.game.LastSkillLevelCursor = m.skillLevelCursor
				m.game.Audio.PlaySEByKey("cursor")
			}
		}
		if isMenuLeftPressed() {
			if m.skillLevelCursor > 1 {
				m.skillLevelCursor--
				m.game.LastSkillLevelCursor = m.skillLevelCursor
				m.game.Audio.PlaySEByKey("cursor")
			}
		}
		if dg, ok := pressedDigitKey(); ok {
			if dg > reachable {
				dg = reachable
			}
			if dg != m.skillLevelCursor {
				m.skillLevelCursor = dg
				m.game.LastSkillLevelCursor = m.skillLevelCursor
				m.game.Audio.PlaySEByKey("cursor")
			}
		}
		skillIdx := m.skillSubIndex
		lv := m.skillLevelCursor

		if lv <= curLv {
			if !isConfirmKeyPressed() && !tappedLvConfirm {
				return
			}
			m.trySkillLevelConfirm(skills, skillIdx, lv)
			return
		}

		if m.upgradeBlocked(m.skillSubIndex) {
			m.upgradeProgress = 0
			if isConfirmKeyPressed() || tappedLvConfirm {
				m.game.Audio.PlaySEByKey("error")
			}
			return
		}
		if !m.upgradeHoldArmed || (!isConfirmKeyDown() && !heldTouch) {
			m.upgradeProgress = 0
			return
		}
		m.upgradeProgress += 1.0 / (1.5 * 60.0)
		if m.upgradeProgress < 1 {
			return
		}

		m.upgradeProgress = 0
		if m.game.UpgradeSkill(m.skillCharIndex, m.skillSubIndex) {
			m.skillLevelCursor = m.game.PlayerSkillLv[m.skillCharIndex][m.skillSubIndex]
			m.game.LastSkillLevelCursor = m.skillLevelCursor
			m.showNotice(fmt.Sprintf("%sをLv%dに強化しました", skills[m.skillSubIndex].Name, m.skillLevelCursor))
			m.game.Audio.PlaySEByKey("decide")
			m.upgradeHoldArmed = false
		}
		return
	}

	if isEscapePressed() {
		m.game.Audio.PlaySEByKey("cancel")
		m.menuState = menuStateSkillCharSel
		return
	}

	if rowT, lvT, ok := m.hitTestAnySkillLevelCell(skills); ok {
		rowReachable := m.reachableSkillLevel(m.skillCharIndex, rowT)
		if lvT > rowReachable {
			lvT = rowReachable
		}
		m.skillSubIndex = rowT
		m.skillLevelCursor = lvT
		m.skillLevelSelecting = true
		m.upgradeProgress = 0
		m.upgradeHoldArmed = false
		m.game.LastSkillSubIndex = m.skillSubIndex
		m.game.LastSkillLevelCursor = m.skillLevelCursor
		m.game.Audio.PlaySEByKey("decide")
		return
	}

	prevIndex := m.skillSubIndex
	if isMenuDownRepeat() {
		m.skillSubIndex = (m.skillSubIndex + 1) % n
	}
	if isMenuUpRepeat() {
		m.skillSubIndex = (m.skillSubIndex - 1 + n) % n
	}
	tappedIdx, tappedOk := m.hitTestSkillNameRows(n)
	tapConfirmed := tapSelectOrConfirm(tappedIdx, tappedOk, &m.skillSubIndex, m.game.Audio)
	if m.skillSubIndex != prevIndex {
		newCurLv := m.game.PlayerSkillLv[m.skillCharIndex][m.skillSubIndex]
		if newCurLv < 1 {
			newCurLv = 1
		}
		m.skillLevelCursor = newCurLv

		m.game.LastSkillSubIndex = m.skillSubIndex
		m.game.LastSkillLevelCursor = m.skillLevelCursor
		m.game.Audio.PlaySEByKey("cursor")
	}

	if isMenuRightPressed() || isConfirmKeyPressed() || tapConfirmed {
		m.skillLevelSelecting = true
		m.upgradeProgress = 0
		m.upgradeHoldArmed = false
		if reachable := m.reachableSkillLevel(m.skillCharIndex, m.skillSubIndex); m.skillLevelCursor > reachable {
			m.skillLevelCursor = reachable
		}
		m.game.LastSkillSubIndex = m.skillSubIndex
		m.game.LastSkillLevelCursor = m.skillLevelCursor
		m.game.Audio.PlaySEByKey("decide")
		return
	}

	if unrelatedTapPressed(tappedOk || areaTapped) {
		m.game.Audio.PlaySEByKey("cancel")
		m.menuState = menuStateSkillCharSel
	}
}

func (m *MenuScene) updateHealTarget() {
	if isEscapePressed() {
		m.game.Audio.PlaySEByKey("cancel")
		m.pendingSkill = 0
		m.menuState = menuStateSkillSub
		return
	}
	const cycleLen = partySize + 1
	if isMenuUpPressed() {
		m.healTargetIndex = (m.healTargetIndex - 1 + cycleLen) % cycleLen
		m.game.Audio.PlaySEByKey("cursor")
	}
	if isMenuDownPressed() {
		m.healTargetIndex = (m.healTargetIndex + 1) % cycleLen
		m.game.Audio.PlaySEByKey("cursor")
	}
	tappedIdx, tappedOk := m.hitTestPartyRowsWithAll(true)
	tapped := tapSelectOrConfirm(tappedIdx, tappedOk, &m.healTargetIndex, m.game.Audio)
	if !isConfirmKeyPressed() && !tapped {
		if unrelatedTapOutsideRects(menuMainContentRect()) {
			m.game.Audio.PlaySEByKey("cancel")
			m.pendingSkill = 0
			m.menuState = menuStateSkillSub
		}
		return
	}
	caster := m.skillCharIndex

	if m.pendingSkill > 0 {
		skillIdx := m.pendingSkill - 1
		lv := m.pendingSkillLevel
		if lv < 1 {
			lv = 1
		}
		skills := m.game.CharacterSkills(caster)
		data := skills[skillIdx].Levels[lv-1]
		cost := data.MPCost

		if m.game.PlayerHP[caster] <= 0 || m.game.PlayerMP[caster] < cost {
			m.game.Audio.PlaySEByKey("error")
			m.pendingSkill = 0
			m.menuState = menuStateSkillSub
			return
		}

		if m.healTargetIndex != partySize && m.game.PlayerHP[m.healTargetIndex] <= 0 {
			m.game.Audio.PlaySEByKey("error")
			return
		}

		m.game.Audio.PlaySEByKey("heal")

		if m.healTargetIndex == partySize {
			m.game.PlayerMP[caster] -= cost
			healAmount := int(float64(m.game.PlayerMagicAtk[caster]) * float64(data.PowerAll) / 100.0 * 10)
			if healAmount < 1 {
				healAmount = 1
			}
			for i := 0; i < partySize; i++ {
				if m.game.PlayerHP[i] <= 0 {
					continue
				}
				m.game.PlayerHP[i] += healAmount
				if m.game.PlayerHP[i] > m.game.PlayerMaxHP[i] {
					m.game.PlayerHP[i] = m.game.PlayerMaxHP[i]
				}
			}
		} else {
			target := m.healTargetIndex
			m.game.PlayerMP[caster] -= cost
			healAmount := int(float64(m.game.PlayerMagicAtk[caster]) * float64(data.PowerSingle) / 100.0 * 10)
			if healAmount < 1 {
				healAmount = 1
			}
			m.game.PlayerHP[target] += healAmount
			if m.game.PlayerHP[target] > m.game.PlayerMaxHP[target] {
				m.game.PlayerHP[target] = m.game.PlayerMaxHP[target]
			}
		}

		if m.game.PlayerMP[caster] < cost {
			m.pendingSkill = 0
			m.menuState = menuStateSkillSub
		}
		return
	}

	if m.game.PlayerHP[caster] <= 0 || m.game.PlayerMP[caster] < mpCostHeal {
		m.game.Audio.PlaySEByKey("error")
		m.menuState = menuStateSkillSub
		return
	}

	if m.healTargetIndex == 4 {
		if m.game.PlayerMP[caster] < mpCostHealAll {
			m.game.Audio.PlaySEByKey("error")
			m.menuState = menuStateSkillSub
			return
		}
		m.game.Audio.PlaySEByKey("heal")
		m.game.PlayerMP[caster] -= mpCostHealAll
		for i := 0; i < 4; i++ {
			if m.game.PlayerHP[i] > 0 {
				heal := m.game.PlayerMaxHP[i] / 4
				m.game.PlayerHP[i] += heal
				if m.game.PlayerHP[i] > m.game.PlayerMaxHP[i] {
					m.game.PlayerHP[i] = m.game.PlayerMaxHP[i]
				}
			}
		}
	} else {
		target := m.healTargetIndex
		if m.game.PlayerHP[target] <= 0 {
			m.game.Audio.PlaySEByKey("error")
			return
		}
		m.game.Audio.PlaySEByKey("heal")
		m.game.PlayerMP[caster] -= mpCostHeal
		heal := m.game.PlayerMaxHP[target] / 2
		m.game.PlayerHP[target] += heal
		if m.game.PlayerHP[target] > m.game.PlayerMaxHP[target] {
			m.game.PlayerHP[target] = m.game.PlayerMaxHP[target]
		}
	}

	if m.game.PlayerMP[caster] < mpCostHeal {
		m.menuState = menuStateSkillSub
	}
}
