package main

// menu_scene_skill.go: スキル選択・スキル強化・回復対象選択の更新処理
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
	m.skillLevelCursor = m.game.LastSkillLevelCursor
	if m.skillLevelCursor < 1 || m.skillLevelCursor > len(skills[m.skillSubIndex].Levels) {
		m.skillLevelCursor = curLv
	}

	m.skillLevelSelecting = false
	m.upgradeResultMsg = ""
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
		if isEscapePressed() {
			m.skillLevelSelecting = false
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
		if !isConfirmKeyPressed() {
			return
		}

		skillIdx := m.skillSubIndex
		lv := m.skillLevelCursor

		if lv <= curLv {
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

		// ── 未解放レベル：強化確認へ ──
		m.upgradeConfirmIndex = 0
		m.upgradeResultMsg = ""
		m.menuState = menuStateSkillUpgrade
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
		m.upgradeResultMsg = ""

		// ← 追加：行とLvカーソルを記憶
		m.game.LastSkillSubIndex = m.skillSubIndex
		m.game.LastSkillLevelCursor = m.skillLevelCursor
	}

	if !isConfirmKeyPressed() {
		return
	}
	m.skillLevelSelecting = true
	m.game.LastSkillSubIndex = m.skillSubIndex       // ← 追加（念のため）
	m.game.LastSkillLevelCursor = m.skillLevelCursor // ← 追加（念のため）
}

// updateSkillUpgrade：SP消費してスキルレベルを上げる確認画面の入力処理。
func (m *MenuScene) updateSkillUpgrade() {
	if isEscapePressed() {
		m.menuState = menuStateSkillSub
		return
	}
	if isMenuUpPressed() || isMenuDownPressed() {
		m.upgradeConfirmIndex = 1 - m.upgradeConfirmIndex
	}
	if !isConfirmKeyPressed() {
		return
	}

	if m.upgradeConfirmIndex == 1 { // いいえ
		m.menuState = menuStateSkillSub
		return
	}

	charIdx := m.skillCharIndex
	skillIdx := m.skillSubIndex

	if !m.game.CanUpgradeSkill(charIdx, skillIdx) {
		m.upgradeResultMsg = "SPが足りません"
		return
	}
	ok := m.game.UpgradeSkill(charIdx, skillIdx)
	if ok {
		m.upgradeResultMsg = "スキルを強化しました！"
	} else {
		m.upgradeResultMsg = "強化できませんでした"
	}
	m.menuState = menuStateSkillSub
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

		m.pendingSkill = 0
		m.menuState = menuStateSkillSub
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

