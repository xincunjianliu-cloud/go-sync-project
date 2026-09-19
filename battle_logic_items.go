package main

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
		s.game.Audio.PlaySEByKey("cursor")
	}
	if isMenuUpPressed() {
		s.itemIndex = (s.itemIndex - 1 + len(items)) % len(items)
		s.game.Audio.PlaySEByKey("cursor")
	}
	tappedIdx, tappedOk := s.hitTestBattleSubRows(len(items))
	tapped := tapSelectOrConfirm(tappedIdx, tappedOk, &s.itemIndex, s.game.Audio)
	if isEscapePressed() || (!tappedOk && s.isTapOutsideBattleSubPanel()) {
		s.game.Audio.PlaySEByKey("cancel")
		s.battlePhase = phasePlayerMenu
		return
	}
	if !isConfirmKeyPressed() && !tapped {
		return
	}

	def, ok := GetItemDef(items[s.itemIndex].ItemID)
	if !ok {
		return
	}
	s.game.Audio.PlaySEByKey("decide")
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

	cycleLen := partySize
	if allowAll {
		cycleLen = partySize + 1
	}
	if isMenuUpPressed() {
		s.itemTargetIndex = (s.itemTargetIndex - 1 + cycleLen) % cycleLen
		s.game.Audio.PlaySEByKey("cursor")
	}
	if isMenuDownPressed() {
		s.itemTargetIndex = (s.itemTargetIndex + 1) % cycleLen
		s.game.Audio.PlaySEByKey("cursor")
	}

	tappedIdx, tappedOk := s.hitTestItemTargets(allowAll)
	tapped := tapSelectOrConfirm(tappedIdx, tappedOk, &s.itemTargetIndex, s.game.Audio)

	hadTouch := len(justPressedTouchPoints()) > 0
	if isEscapePressed() || (hadTouch && !tappedOk) {
		s.game.Audio.PlaySEByKey("cancel")
		s.pendingItemID = ""
		s.battlePhase = phaseItemMenu
		return
	}
	if !isConfirmKeyPressed() && !tapped {
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
			s.game.Audio.PlaySEByKey("error")
			return
		}
		if s.applyItemToTargetInBattle(def, target) {
			used = true
		}
	}

	if !used {
		s.game.Audio.PlaySEByKey("error")
		return
	}

	s.game.Audio.PlaySEByKey("decide")
	s.game.ConsumeItem(s.pendingItemID, 1)
	s.battleLog = def.Name
	s.battleLogTimer = battleLogDuration
	s.pendingItemID = ""
	s.finishPlayerTurn(itemUseReturnPosition)
}

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
