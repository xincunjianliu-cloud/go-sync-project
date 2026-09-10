package main

// menu_scene_skill.go: スキル選択・スキル強化・回復対象選択の更新処理

// reachableSkillLevel は指定キャラ・指定スキルでLvカーソルが到達できる上限を返す。
// 現在のレベルの次(curLv+1)までしか進められない（Lv2を強化しないとLv3を選べない、が絶対のルール）。
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
		m.menuState = menuStateMain
		return
	}
	if isMenuUpPressed() {
		m.skillCharIndex = (m.skillCharIndex - 1 + 4) % 4
		m.game.LastSkillCharIndex = m.skillCharIndex // ← 追加
	}
	if isMenuDownPressed() {
		m.skillCharIndex = (m.skillCharIndex + 1) % 4
		m.game.LastSkillCharIndex = m.skillCharIndex // ← 追加
	}
	if !isConfirmKeyPressed() {
		return
	}
	if m.game.PlayerHP[m.skillCharIndex] <= 0 {
		return
	}

	// ← 変更：記憶されているスキル行・Lvカーソルを復元（キャラが変わっていたら安全にクランプ）
	skills := m.game.CharacterSkills(m.skillCharIndex)
	m.skillSubIndex = m.game.LastSkillSubIndex
	if m.skillSubIndex < 0 || m.skillSubIndex >= len(skills) {
		m.skillSubIndex = 0
	}
	curLv := m.game.PlayerSkillLv[m.skillCharIndex][m.skillSubIndex]
	if curLv < 1 {
		curLv = 1
	}
	// ★修正：カーソル復元先はLv一覧の末尾までではなく、到達可能な上限(curLv+1)までに
	// クランプする。これを怠ると、以前に別スキル/別キャラでLv3まで見ていた時の
	// カーソル値が残ったまま復元され、Lv2を強化していないのにLv3を選べてしまっていた。
	m.skillLevelCursor = m.game.LastSkillLevelCursor
	reachable := m.reachableSkillLevel(m.skillCharIndex, m.skillSubIndex)
	if m.skillLevelCursor < 1 || m.skillLevelCursor > reachable {
		m.skillLevelCursor = curLv
	}

	m.skillLevelSelecting = false
	m.menuState = menuStateSkillSub
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

// updateSkillSub：主人公(index0)選択時はHeroSkillsの4項目、他キャラは従来通り。
// 「E」キーでスキル強化画面へ遷移(主人公のみ)。
func (m *MenuScene) updateSkillSub() {
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

	// ── Lv選択モード：左右でLvカーソル移動、決定で実行、ESCで行選択に戻る ──
	if m.skillLevelSelecting {
		if !isConfirmKeyDown() {
			m.upgradeHoldArmed = true
			m.upgradeProgress = 0
		}
		if isEscapePressed() {
			m.skillLevelSelecting = false
			m.upgradeHoldArmed = false
			return
		}
		if isMenuRightPressed() {
			if m.skillLevelCursor < reachable {
				m.skillLevelCursor++
				m.game.LastSkillLevelCursor = m.skillLevelCursor // ← 追加
			}
		}
		if isMenuLeftPressed() {
			if m.skillLevelCursor > 1 {
				m.skillLevelCursor--
				m.game.LastSkillLevelCursor = m.skillLevelCursor // ← 追加
			}
		}
		skillIdx := m.skillSubIndex
		lv := m.skillLevelCursor

		if lv <= curLv {
			if !isConfirmKeyPressed() {
				return
			}
			// ── 解放済みレベル：メニューから使用（回復系のみ） ──
			data := skills[skillIdx].Levels[lv-1]
			if data.IsHeal {
				m.pendingSkill = skillIdx + 1
				m.pendingSkillLevel = lv
				m.healTargetIndex = 0
				m.menuState = menuStateHealTarget
			}
			return
		}

		// ── 未解放レベル：決定キー長押しで強化 ──
		if !m.game.CanUpgradeSkill(m.skillCharIndex, m.skillSubIndex) {
			m.upgradeProgress = 0
			return
		}
		if !m.upgradeHoldArmed || !isConfirmKeyDown() {
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
		}
		return
	}

	// ── 行選択モード：上下でスキル行移動、決定でLv選択モードへ ──
	if isEscapePressed() {
		m.menuState = menuStateSkillCharSel
		return
	}

	prevIndex := m.skillSubIndex
	if isMenuDownPressed() {
		m.skillSubIndex = (m.skillSubIndex + 1) % n
	}
	if isMenuUpPressed() {
		m.skillSubIndex = (m.skillSubIndex - 1 + n) % n
	}
	if m.skillSubIndex != prevIndex {
		// 行を切り替えたら現在Lvにカーソルを合わせる
		newCurLv := m.game.PlayerSkillLv[m.skillCharIndex][m.skillSubIndex]
		if newCurLv < 1 {
			newCurLv = 1
		}
		m.skillLevelCursor = newCurLv

		// ← 追加：行とLvカーソルを記憶
		m.game.LastSkillSubIndex = m.skillSubIndex
		m.game.LastSkillLevelCursor = m.skillLevelCursor
	}

	if !isConfirmKeyPressed() {
		return
	}
	m.skillLevelSelecting = true
	m.upgradeProgress = 0
	m.upgradeHoldArmed = false
	// ★修正：Lv選択モードに入る直前にも到達可能上限でクランプし、
	// 古いカーソル値が残っていてもLv2未強化のままLv3を選べないようにする。
	if reachable := m.reachableSkillLevel(m.skillCharIndex, m.skillSubIndex); m.skillLevelCursor > reachable {
		m.skillLevelCursor = reachable
	}
	m.game.LastSkillSubIndex = m.skillSubIndex       // ← 追加（念のため）
	m.game.LastSkillLevelCursor = m.skillLevelCursor // ← 追加（念のため）
}

func (m *MenuScene) updateHealTarget() {
	if isEscapePressed() {
		m.pendingSkill = 0
		m.menuState = menuStateSkillSub
		return
	}
	if isMenuUpPressed() {
		if m.healTargetIndex < 4 {
			m.healTargetIndex = (m.healTargetIndex - 1 + 4) % 4
		}
	}
	if isMenuDownPressed() {
		if m.healTargetIndex < 4 {
			m.healTargetIndex = (m.healTargetIndex + 1) % 4
		}
	}
	if isMenuRightPressed() {
		m.healTargetIndex = 4
	}
	if isMenuLeftPressed() {
		if m.healTargetIndex == 4 {
			m.healTargetIndex = 0
		}
	}
	if !isConfirmKeyPressed() {
		return
	}

	caster := m.skillCharIndex

	// ── 新スキル体系（主人公のHeroSkills）をメニューから使用する場合 ──
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
			m.pendingSkill = 0
			m.menuState = menuStateSkillSub
			return
		}

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
			if m.game.PlayerHP[target] <= 0 {
				return
			}
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

		// ← 変更：使用直後に自動で戻さず、対象選択画面のまま連続で使えるようにする。
		// MPが足りなくなった時だけスキル一覧画面へ戻す。
		if m.game.PlayerMP[caster] < cost {
			m.pendingSkill = 0
			m.menuState = menuStateSkillSub
		}
		return
	}

	// ── 既存の旧仕様（固定値回復、主人公以外）はそのまま ──
	if m.game.PlayerHP[caster] <= 0 || m.game.PlayerMP[caster] < mpCostHeal {
		m.menuState = menuStateSkillSub
		return
	}

	if m.healTargetIndex == 4 {
		if m.game.PlayerMP[caster] < mpCostHealAll {
			m.menuState = menuStateSkillSub
			return
		}
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
			return
		}
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
