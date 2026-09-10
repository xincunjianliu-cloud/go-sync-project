package main

// menu_scene_item.go: メニュー画面（フィールド）でのアイテム選択・使用対象選択の更新処理

// usableFieldItems はメニュー画面から使用できる（UsableInField）、
// かつ所持数1以上のアイテムスロット一覧を返す。
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

func (m *MenuScene) updateItemList() {
	if isEscapePressed() {
		m.menuState = menuStateMain
		return
	}

	items := m.usableFieldItems()
	if len(items) == 0 {
		return
	}
	if m.itemListIndex >= len(items) {
		m.itemListIndex = len(items) - 1
	}
	if m.itemListIndex < 0 {
		m.itemListIndex = 0
	}

	if isMenuDownPressed() {
		m.itemListIndex = (m.itemListIndex + 1) % len(items)
	}
	if isMenuUpPressed() {
		m.itemListIndex = (m.itemListIndex - 1 + len(items)) % len(items)
	}
	if !isConfirmKeyPressed() {
		return
	}

	m.pendingItemID = items[m.itemListIndex].ItemID
	m.itemTargetIndex = 0
	m.menuState = menuStateItemTarget
}

func (m *MenuScene) updateItemTarget() {
	if isEscapePressed() {
		m.pendingItemID = ""
		m.menuState = menuStateItemList
		return
	}

	def, ok := GetItemDef(m.pendingItemID)
	if !ok {
		m.pendingItemID = ""
		m.menuState = menuStateItemList
		return
	}
	allowAll := def.Target == TargetAll || def.Target == TargetBoth

	if isMenuUpPressed() {
		if m.itemTargetIndex < partySize {
			m.itemTargetIndex = (m.itemTargetIndex - 1 + partySize) % partySize
		}
	}
	if isMenuDownPressed() {
		if m.itemTargetIndex < partySize {
			m.itemTargetIndex = (m.itemTargetIndex + 1) % partySize
		}
	}
	if allowAll {
		if isMenuRightPressed() {
			m.itemTargetIndex = partySize
		}
		if isMenuLeftPressed() {
			if m.itemTargetIndex == partySize {
				m.itemTargetIndex = 0
			}
		}
	}

	if !isConfirmKeyPressed() {
		return
	}

	used := false
	if m.itemTargetIndex == partySize {
		for i := 0; i < partySize; i++ {
			if _, _, ok := applyItemEffect(m.game, def, i); ok {
				used = true
			}
		}
	} else {
		target := m.itemTargetIndex
		if !def.Revive && m.game.PlayerHP[target] <= 0 {
			return
		}
		if _, _, ok := applyItemEffect(m.game, def, target); ok {
			used = true
		}
	}

	if !used {
		return
	}

	m.game.ConsumeItem(m.pendingItemID, 1)

	// ← 変更：使用直後に自動で戻さず、対象選択画面のまま連続で使えるようにする。
	// そのアイテムを使い切った時だけ一覧画面へ戻す。
	if m.game.ItemCount(m.pendingItemID) <= 0 {
		m.pendingItemID = ""

		items := m.usableFieldItems()
		if m.itemListIndex >= len(items) {
			m.itemListIndex = len(items) - 1
		}
		if m.itemListIndex < 0 {
			m.itemListIndex = 0
		}
		m.menuState = menuStateItemList
	}
}
