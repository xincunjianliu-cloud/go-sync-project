package main

// battle_logic_items.go: バトル中のアイテム選択・使用対象選択の更新処理
// スキル選択(phaseSkillMenu/phaseHealSelect)と全く同じ操作感になるようにしてある。

// battleUsableItems はバトル中に使用できる（UsableInBattle）、
// かつ所持数1以上のアイテムスロット一覧を返す。
func (s *BattleScene) battleUsableItems() []InventorySlot {
	var list []InventorySlot
	for _, slot := range s.game.Inventory {
		if slot.Count <= 0 {
			continue
		}
		def, ok := GetItemDef(slot.ItemID)
		if !ok || !def.UsableInBattle {
			continue
		}
		list = append(list, slot)
	}
	return list
}

func (s *BattleScene) hasAnyBattleUsableItem() bool {
	return len(s.battleUsableItems()) > 0
}

func (s *BattleScene) updateItemMenu(dt float64) {
	p := s.waitingActor
	if p < 0 || p >= partySize {
		s.battlePhase = phasePlayerMenu
		return
	}

	items := s.battleUsableItems()
	if len(items) == 0 {
		s.battlePhase = phasePlayerMenu
		return
	}
	if s.itemIndex >= len(items) {
		s.itemIndex = len(items) - 1
	}
	if s.itemIndex < 0 {
		s.itemIndex = 0
	}

	if isMenuDownPressed() {
		s.itemIndex = (s.itemIndex + 1) % len(items)
	}
	if isMenuUpPressed() {
		s.itemIndex = (s.itemIndex - 1 + len(items)) % len(items)
	}
	if isEscapePressed() {
		s.battlePhase = phasePlayerMenu
		return
	}
	if !isConfirmKeyPressed() {
		return
	}

	def, ok := GetItemDef(items[s.itemIndex].ItemID)
	if !ok {
		return
	}
	s.pendingItemID = def.ID
	s.itemTargetIndex = 0
	s.battlePhase = phaseItemTarget
}

func (s *BattleScene) updateItemTargetSelect() {
	p := s.waitingActor
	if p < 0 || p >= partySize {
		s.battlePhase = phaseItemMenu
		return
	}

	def, ok := GetItemDef(s.pendingItemID)
	if !ok {
		s.pendingItemID = ""
		s.battlePhase = phaseItemMenu
		return
	}
	allowAll := def.Target == TargetAll || def.Target == TargetBoth

	if isMenuUpPressed() {
		if s.itemTargetIndex < partySize {
			s.itemTargetIndex = (s.itemTargetIndex - 1 + partySize) % partySize
		}
	}
	if isMenuDownPressed() {
		if s.itemTargetIndex < partySize {
			s.itemTargetIndex = (s.itemTargetIndex + 1) % partySize
		}
	}
	if allowAll {
		if isMenuRightPressed() {
			s.itemTargetIndex = partySize
		}
		if isMenuLeftPressed() {
			if s.itemTargetIndex == partySize {
				s.itemTargetIndex = 0
			}
		}
	}
	if isEscapePressed() {
		s.pendingItemID = ""
		s.battlePhase = phaseItemMenu
		return
	}
	if !isConfirmKeyPressed() {
		return
	}

	used := false
	if s.itemTargetIndex == partySize {
		for i := 0; i < partySize; i++ {
			if s.applyItemToTargetInBattle(def, i) {
				used = true
			}
		}
	} else {
		target := s.itemTargetIndex
		if !def.Revive && s.game.PlayerHP[target] <= 0 {
			return
		}
		if s.applyItemToTargetInBattle(def, target) {
			used = true
		}
	}

	if !used {
		return
	}

	s.game.ConsumeItem(s.pendingItemID, 1)
	s.battleLog = def.Name
	s.battleLogTimer = battleLogDuration
	s.pendingItemID = ""
	s.finishPlayerTurn(true)
}

// applyItemToTargetInBattle はアイテム効果を対象1人に適用し、
// 回復量をダメージポップ（回復表示）として出す。
func (s *BattleScene) applyItemToTargetInBattle(def ItemDef, target int) bool {
	if target < 0 || target >= partySize {
		return false
	}
	hpHealed, mpHealed, applied := applyItemEffect(s.game, def, target)
	if def.CureDebuff {
		s.PlayerDebuffs[target] = nil
	}
	if !applied {
		return false
	}

	if hpHealed > 0 {
		s.damagePops = append(s.damagePops, DamagePop{
			Value:  hpHealed,
			X:      s.partyScreenX[target],
			Y:      s.partyScreenY[target] - 30.0,
			Vy:     -80.0,
			Timer:  0.0,
			IsHeal: true,
		})
	}
	if mpHealed > 0 {
		s.damagePops = append(s.damagePops, DamagePop{
			Value:  mpHealed,
			X:      s.partyScreenX[target] - 10.0,
			Y:      s.partyScreenY[target] - 45.0,
			Vy:     -80.0,
			Timer:  -0.1,
			IsHeal: true,
		})
	}
	return true
}
