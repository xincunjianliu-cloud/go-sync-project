package main

func (s *BattleScene) battleUsableItems() []InventorySlot {
	return filterInventory(s.game.Inventory, func(def ItemDef) bool {
		if !def.UsableInBattle {
			return false
		}
		return !s.isBossBattle() || (!def.DoubleEXP && !def.GuaranteedEscape)
	})
}

func (s *BattleScene) hasAnyBattleUsableItem() bool {
	return len(s.battleUsableItems()) > 0
}

func (s *BattleScene) updateItemMenu() {
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
	firstVisible, lastVisible := s.battleItemVisibleRange(len(items))
	tappedIdx, tappedOk := s.hitTestBattleSubRowsFrom(firstVisible, lastVisible-firstVisible)
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
	if def.Target == TargetOther {
		s.useBattleItemWithoutTarget(def)
		s.pendingItemID = ""
		return
	}
	if def.Target == TargetAll {
		s.itemTargetIndex = partySize
	}
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
	allowSingle, allowAll := itemTargetModes(def)

	cycleLen := partySize
	if allowSingle && allowAll {
		cycleLen = partySize + 1
	} else if !allowSingle {
		cycleLen = 1
		s.itemTargetIndex = partySize
	}
	if allowSingle && isMenuUpPressed() {
		s.itemTargetIndex = (s.itemTargetIndex - 1 + cycleLen) % cycleLen
		s.game.Audio.PlaySEByKey("cursor")
	}
	if allowSingle && isMenuDownPressed() {
		s.itemTargetIndex = (s.itemTargetIndex + 1) % cycleLen
		s.game.Audio.PlaySEByKey("cursor")
	}

	tappedIdx, tappedOk := s.hitTestItemTargets(allowAll)
	if tappedOk && !allowSingle && tappedIdx != partySize {
		tappedOk = false
	}
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
	ctReductionTarget := -1
	if s.itemTargetIndex == partySize {
		for i := range partySize {
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
			if def.NextActionCTReduction > 0 {
				ctReductionTarget = target
			}
		}
	}

	if !used {
		s.game.Audio.PlaySEByKey("error")
		return
	}

	s.game.Audio.PlaySEByKey("decide")
	s.finishUsingBattleItem(def, ctReductionTarget)
	s.pendingItemID = ""
}

func (s *BattleScene) applyItemToTargetInBattle(def ItemDef, target int) bool {
	if target < 0 || target >= partySize {
		return false
	}
	if def.DoubleEXP {
		if s.isBossBattle() {
			return false
		}
		s.expBoostActive = true
		return true
	}
	if def.NextActionCTReduction > 0 {
		return s.game.PlayerHP[target] > 0
	}
	_, mpHealed, applied := applyItemEffect(s.game, def, target)
	if def.CureDebuff {
		s.PlayerDebuffs[target] = nil
	}
	if !applied {
		// In battle, an HP restorative can still be spent as an action even
		// when the target is already at full HP. Field use remains effect-only.
		if def.Revive || s.game.PlayerHP[target] <= 0 ||
			(def.HealHP <= 0 && def.HealHPPercent <= 0) {
			return false
		}
	}

	displayHPHeal := def.HealHP + s.game.PlayerMaxHP[target]*def.HealHPPercent/100
	if displayHPHeal > 0 {
		s.spawnDamagePop(DamagePop{
			Value:  displayHPHeal,
			X:      s.partyScreenX[target] + spriteFrameW/2,
			Y:      s.partyScreenY[target] - partyDamagePopOffsetY,
			Vy:     -45.0,
			Timer:  0.0,
			IsHeal: true,
		})
	}
	if mpHealed > 0 {
		s.spawnDamagePop(DamagePop{
			Value:  mpHealed,
			X:      s.partyScreenX[target] + spriteFrameW/2,
			Y:      s.partyScreenY[target] - partyDamagePopOffsetY - 15.0,
			Vy:     -45.0,
			Timer:  -0.1,
			IsHeal: true,
			IsMP:   true,
		})
	}
	return true
}

func (s *BattleScene) useBattleItemWithoutTarget(def ItemDef) {
	if def.GuaranteedEscape {
		if s.isBossBattle() {
			s.game.Audio.PlaySEByKey("error")
			s.battlePhase = phaseItemMenu
			return
		}
		s.game.ConsumeItem(def.ID, 1)
		s.fleeSucceeded = true
		s.battleLog = "逃げきれた！"
		s.battleLogTimer = 1.5
		s.battlePhase = phaseMessage
		return
	}
	s.finishUsingBattleItem(def, -1)
}

func (s *BattleScene) finishUsingBattleItem(def ItemDef, ctReductionTarget int) {
	s.game.ConsumeItem(def.ID, 1)
	s.addGaugePoint(def.GaugePoint)
	s.battleLog = def.Name
	s.battleLogTimer = battleLogDuration
	s.finishPlayerTurn(def.ReturnPosition)
	if ctReductionTarget >= 0 && ctReductionTarget < partySize {
		s.nextActionCTReductions[ctReductionTarget] += def.NextActionCTReduction
	}
}
