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
		m.game.Audio.PlaySEByKey("cancel")
		m.menuState = menuStateMain
		return
	}

	items := m.usableFieldItems()
	if len(items) == 0 {
		if isConfirmKeyPressed() {
			m.game.Audio.PlaySEByKey("error")
		}
		if unrelatedTapOutsideRects(menuMainContentRect()) {
			m.game.Audio.PlaySEByKey("cancel")
			m.menuState = menuStateMain
		}
		return
	}
	m.clampItemListIndex(len(items))

	if isMenuDownRepeat() {
		m.itemListIndex = (m.itemListIndex + 1) % len(items)
		m.game.Audio.PlaySEByKey("cursor")
	}
	if isMenuUpRepeat() {
		m.itemListIndex = (m.itemListIndex - 1 + len(items)) % len(items)
		m.game.Audio.PlaySEByKey("cursor")
	}
	tapped := false
	if idx, ok := m.hitTestItemListRows(len(items)); ok {
		if idx != m.itemListIndex {
			m.game.Audio.PlaySEByKey("cursor")
		}
		m.itemListIndex = idx
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
		m.game.Audio.PlaySEByKey("cancel")
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
			m.game.Audio.PlaySEByKey("decide")
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
			m.game.Audio.PlaySEByKey("cursor")
		}
		if isMenuDownRepeat() {
			m.itemTargetIndex = (m.itemTargetIndex + 1) % cycleLen
			m.game.Audio.PlaySEByKey("cursor")
		}
	}

	tappedIdx, tappedOk := m.hitTestPartyRowsWithAll(allowAll)
	hitPartyArea := tappedOk
	if tappedOk && tappedIdx < partySize && !allowSingle {
		tappedOk = false
	}
	tapped := tapSelectOrConfirm(tappedIdx, tappedOk, &m.itemTargetIndex, m.game.Audio)

	if !isConfirmKeyPressed() && !tapped {
		if !hitPartyArea && len(justPressedTouchPoints()) > 0 {
			m.game.Audio.PlaySEByKey("cancel")
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
			m.game.Audio.PlaySEByKey("error")
			return
		}
	} else {
		target := m.itemTargetIndex
		if _, _, ok := applyItemEffect(m.game, def, target); !ok {
			m.game.Audio.PlaySEByKey("error")
			return
		}
	}

	m.game.Audio.PlaySEByKey("decide")
	m.game.ConsumeItem(m.pendingItemID, 1)
	m.clearNotice()

	if m.game.ItemCount(m.pendingItemID) <= 0 {
		m.pendingItemID = ""
		m.clampItemListIndex(len(m.usableFieldItems()))
		m.menuState = menuStateItemList
	}
}
