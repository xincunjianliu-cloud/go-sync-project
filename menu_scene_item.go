package main

func (m *MenuScene) hitTestItemListRows(rowCount int) (int, bool) {
	rects := make([]tapRect, rowCount)
	for i := 0; i < rowCount; i++ {
		y := skillRowStartY + float64(i)*itemRowGapY
		rects[i] = tapRect{
			x: skillNameX - 4,
			y: y - itemRowGapY/2,
			w: itemCountX - skillNameX + 20,
			h: itemRowGapY,
		}
	}
	return hitTestTapRects(rects)
}

func (m *MenuScene) usableFieldItems() []InventorySlot {
	var list []InventorySlot
	for _, slot := range m.game.Inventory {
		if slot.Count <= 0 {
			continue
		}
		def, ok := GetItemDef(slot.ItemID)
		if !ok || !def.UsableInField {
			continue
		}
		list = append(list, slot)
	}
	return list
}

func (m *MenuScene) clampItemListIndex(n int) {
	if m.itemListIndex >= n {
		m.itemListIndex = n - 1
	}
	if m.itemListIndex < 0 {
		m.itemListIndex = 0
	}
}

func (m *MenuScene) updateItemList() {
	if isEscapePressed() {
		m.menuState = menuStateMain
		return
	}

	items := m.usableFieldItems()
	if len(items) == 0 {
		if isConfirmKeyPressed() {
			m.showNotice("メニューから使えるアイテムを持っていません")
		}
		if unrelatedTapOutsideRects(menuMainContentRect()) {
			m.menuState = menuStateMain
		}
		return
	}
	m.clampItemListIndex(len(items))

	if isMenuDownRepeat() {
		m.itemListIndex = (m.itemListIndex + 1) % len(items)
	}
	if isMenuUpRepeat() {
		m.itemListIndex = (m.itemListIndex - 1 + len(items)) % len(items)
	}
	tapped := false
	if idx, ok := m.hitTestItemListRows(len(items)); ok {
		m.itemListIndex = idx
		tapped = true
	}
	if !isConfirmKeyPressed() && !tapped {
		if unrelatedTapOutsideRects(menuMainContentRect()) {
			m.menuState = menuStateMain
		}
		return
	}

	m.beginItemTarget(items[m.itemListIndex].ItemID)
}

func (m *MenuScene) beginItemTarget(itemID string) {
	m.pendingItemID = itemID
	m.clearNotice()
	if def, ok := GetItemDef(itemID); ok {
		allowSingle, allowAll := itemTargetModes(def)
		switch {
		case !allowSingle:
			m.itemTargetIndex = partySize
		case !allowAll && m.itemTargetIndex == partySize:
			m.itemTargetIndex = 0
		}
	}
	m.menuState = menuStateItemTarget
}

func itemTargetModes(def ItemDef) (allowSingle, allowAll bool) {
	return def.Target != TargetAll, def.Target == TargetAll || def.Target == TargetBoth
}

func (m *MenuScene) updateItemTarget() {
	if isEscapePressed() {
		m.pendingItemID = ""
		m.menuState = menuStateItemList
		return
	}

	def, ok := GetItemDef(m.pendingItemID)
	if !ok || m.game.ItemCount(m.pendingItemID) <= 0 {
		m.pendingItemID = ""
		m.menuState = menuStateItemList
		return
	}
	allowSingle, allowAll := itemTargetModes(def)
	def.CureDebuff = false

	items := m.usableFieldItems()
	if idx, ok := m.hitTestItemListRows(len(items)); ok {
		if items[idx].ItemID != m.pendingItemID {
			m.itemListIndex = idx
			m.beginItemTarget(items[idx].ItemID)
		}
		return
	}

	if allowSingle {
		cycleLen := partySize
		if allowAll {
			cycleLen = partySize + 1
		}
		if isMenuUpRepeat() {
			m.itemTargetIndex = (m.itemTargetIndex - 1 + cycleLen) % cycleLen
		}
		if isMenuDownRepeat() {
			m.itemTargetIndex = (m.itemTargetIndex + 1) % cycleLen
		}
	}

	tappedIdx, tappedOk := m.hitTestPartyRowsWithAll(allowAll)
	hitPartyArea := tappedOk
	if tappedOk && tappedIdx < partySize && !allowSingle {
		tappedOk = false
	}
	tapped := tapSelectOrConfirm(tappedIdx, tappedOk, &m.itemTargetIndex)

	if !isConfirmKeyPressed() && !tapped {
		if !hitPartyArea && len(justPressedTouchPoints()) > 0 {
			m.pendingItemID = ""
			m.menuState = menuStateItemList
		}
		return
	}

	used := false
	if m.itemTargetIndex == partySize {
		for i := 0; i < partySize; i++ {
			if _, _, ok := applyItemEffect(m.game, def, i); ok {
				used = true
			}
		}
		if !used {
			m.showNotice("使っても効果がありません")
			return
		}
	} else {
		target := m.itemTargetIndex
		if _, _, ok := applyItemEffect(m.game, def, target); !ok {
			m.showNotice(itemNoEffectReason(m.game, def, target))
			return
		}
	}

	m.game.ConsumeItem(m.pendingItemID, 1)
	m.clearNotice()

	if m.game.ItemCount(m.pendingItemID) <= 0 {
		m.pendingItemID = ""
		m.clampItemListIndex(len(m.usableFieldItems()))
		m.menuState = menuStateItemList
	}
}

func itemNoEffectReason(g *Game, def ItemDef, target int) string {
	name := PlayerNames[target]
	dead := g.PlayerHP[target] <= 0
	switch {
	case dead && !def.Revive:
		return name + "は戦闘不能のため効果がありません"
	case !dead && def.Revive:
		return name + "は戦闘不能ではありません"
	}
	return name + "に使っても効果がありません"
}
