package main

import "github.com/hajimehoshi/ebiten/v2"

func (m *MenuScene) hitTestItemListRows(rowCount int) (int, bool) {
	firstVisible, lastVisible := menuItemVisibleRange(rowCount, m.itemListIndex, m.itemListFirstVisible)
	rects := make([]tapRect, lastVisible-firstVisible)
	rowRight := itemCountX + 20
	if _, _, _, _, _, hasScrollBar := menuItemScrollBarGeometry(rowCount, firstVisible, lastVisible); hasScrollBar {
		barX := menuFrameRight - itemScrollBarRightGap - scrollBarTrackWidth
		rowRight = min(rowRight, barX-itemScrollBarTouchPadX)
	}
	for i := range rects {
		y := itemRowStartY + float64(i)*itemRowGapY
		rects[i] = tapRect{
			x: skillNameX - 4,
			y: y - itemRowGapY/2,
			w: rowRight - (skillNameX - 4),
			h: itemRowGapY,
		}
	}
	row, ok := hitTestTapRects(rects)
	if !ok {
		return 0, false
	}
	return firstVisible + row, true
}

func menuItemVisibleRowCount(itemCount int) int {
	if itemCount <= 0 {
		return 0
	}

	availableHeight := menuFrameBottom - itemRowStartY - itemRowGapY/2
	visibleRows := max(1, int(availableHeight/itemRowGapY)+1)
	return min(visibleRows, itemCount)
}

func menuItemVisibleRange(itemCount, selectedIndex, requestedFirst int) (first, last int) {
	visibleRows := menuItemVisibleRowCount(itemCount)
	if visibleRows <= 0 {
		return 0, 0
	}

	selectedIndex = max(0, min(selectedIndex, itemCount-1))
	first = max(0, min(requestedFirst, itemCount-visibleRows))
	if selectedIndex < first {
		first = selectedIndex
	} else if selectedIndex >= first+visibleRows {
		first = selectedIndex - visibleRows + 1
	}
	return first, first + visibleRows
}

func (m *MenuScene) syncItemListFirstVisible(itemCount int) {
	m.itemListFirstVisible, _ = menuItemVisibleRange(itemCount, m.itemListIndex, m.itemListFirstVisible)
}

func (m *MenuScene) updateItemListScrollBar(itemCount int) bool {
	firstVisible, lastVisible := menuItemVisibleRange(itemCount, m.itemListIndex, m.itemListFirstVisible)
	barX, trackY, trackHeight, _, thumbH, ok := menuItemScrollBarGeometry(itemCount, firstVisible, lastVisible)
	if !ok {
		m.itemScrollBarDrag.step(float64(gameWidth)+1, 0, 0, 0)
		return false
	}

	dragRectX := barX - itemScrollBarTouchPadX
	dragRectW := scrollBarCursorWidth + itemScrollBarTouchPadX*2
	delta := m.itemScrollBarDrag.step(dragRectX, trackY, dragRectW, trackHeight)
	moveRange := max(trackHeight-thumbH, 0)
	if delta != 0 && moveRange > 0 {
		visibleRows := lastVisible - firstVisible
		maxFirst := itemCount - visibleRows
		m.itemScrollBarDrag.stepScrollTop(
			-delta/moveRange*float64(maxFirst),
			1,
			&m.itemListFirstVisible,
			maxFirst,
			&m.itemScrollAccum,
		)
		m.itemListFirstVisible = max(0, min(m.itemListFirstVisible, maxFirst))
		last := m.itemListFirstVisible + visibleRows
		m.itemListIndex = max(m.itemListFirstVisible, min(m.itemListIndex, last-1))
	}
	return m.itemScrollBarDrag.active || m.itemScrollBarDrag.justTapped
}

func (m *MenuScene) scrollItemListByWheel(itemCount int, wheelY float64) {
	firstVisible, lastVisible := menuItemVisibleRange(itemCount, m.itemListIndex, m.itemListFirstVisible)
	visibleRows := lastVisible - firstVisible
	maxFirst := itemCount - visibleRows
	if maxFirst <= 0 || wheelY == 0 {
		return
	}

	stepScrollTopByWheel(
		wheelY,
		itemRowGapY,
		&m.itemListFirstVisible,
		maxFirst,
		&m.itemScrollAccum,
	)
	m.itemListFirstVisible = max(0, min(m.itemListFirstVisible, maxFirst))
	if m.itemListFirstVisible == 0 && m.itemScrollAccum > 0 ||
		m.itemListFirstVisible == maxFirst && m.itemScrollAccum < 0 {
		m.itemScrollAccum = 0
	}
	last := m.itemListFirstVisible + visibleRows
	m.itemListIndex = max(m.itemListFirstVisible, min(m.itemListIndex, last-1))
}

func menuItemScrollBarGeometry(itemCount, firstVisible, lastVisible int) (barX, trackY, trackHeight, thumbY, thumbH float64, ok bool) {
	visibleRows := lastVisible - firstVisible
	if itemCount <= visibleRows || visibleRows <= 0 {
		return 0, 0, 0, 0, 0, false
	}

	barX = menuFrameRight - itemScrollBarRightGap - scrollBarTrackWidth
	trackHeight = min(itemScrollBarTrackH, menuFrameBottom-menuFrameTop)
	trackY = menuFrameTop + (menuFrameBottom-menuFrameTop-trackHeight)/2
	thumbH = min(trackHeight, max(itemScrollBarMinH, trackHeight*float64(visibleRows)/float64(itemCount)))
	moveRange := max(trackHeight-thumbH, 0)
	maxStart := itemCount - visibleRows
	ratio := float64(firstVisible) / float64(maxStart)
	ratio = max(0, min(ratio, 1))
	thumbY = trackY + ratio*moveRange
	return barX, trackY, trackHeight, thumbY, thumbH, true
}

func (m *MenuScene) menuItems() []InventorySlot {
	return filterInventory(m.game.Inventory, func(def ItemDef) bool {
		return def.UsableInField || def.UsableInBattle
	})
}

func itemUsableInField(slot InventorySlot) bool {
	def, ok := GetItemDef(slot.ItemID)
	return ok && def.UsableInField
}

func nextItemListIndex(itemCount, from, dir int) (int, bool) {
	if itemCount <= 0 || dir == 0 {
		return 0, false
	}
	idx := (from + dir) % itemCount
	if idx < 0 {
		idx += itemCount
	}
	return idx, true
}

func (m *MenuScene) clampItemListIndex(itemCount int) {
	if itemCount <= 0 {
		m.itemListIndex = 0
		m.itemListFirstVisible = 0
		return
	}
	if m.itemListIndex >= itemCount {
		m.itemListIndex = itemCount - 1
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

	items := m.menuItems()
	if len(items) == 0 {
		if isConfirmKeyPressed() {
			m.game.Audio.PlaySEByKey("error")
		}
		return
	}
	m.clampItemListIndex(len(items))

	if isMenuDownRepeat() {
		if idx, ok := nextItemListIndex(len(items), m.itemListIndex, 1); ok && idx != m.itemListIndex {
			m.itemListIndex = idx
			m.game.Audio.PlaySEByKey("cursor")
		}
	}
	if isMenuUpRepeat() {
		if idx, ok := nextItemListIndex(len(items), m.itemListIndex, -1); ok && idx != m.itemListIndex {
			m.itemListIndex = idx
			m.game.Audio.PlaySEByKey("cursor")
		}
	}
	m.syncItemListFirstVisible(len(items))
	if m.updateItemListScrollBar(len(items)) {
		return
	}
	_, wheelY := ebiten.Wheel()
	m.scrollItemListByWheel(len(items), wheelY)

	tapped := false
	if idx, ok := m.hitTestItemListRows(len(items)); ok {
		if idx != m.itemListIndex {
			m.game.Audio.PlaySEByKey("cursor")
		}
		m.itemListIndex = idx
		m.syncItemListFirstVisible(len(items))
		tapped = true
	}
	if !isConfirmKeyPressed() && !tapped {
		return
	}

	if !itemUsableInField(items[m.itemListIndex]) {
		if tapped && !isConfirmKeyPressed() {
			return
		}
		m.game.Audio.PlaySEByKey("error")
		return
	}
	m.game.Audio.PlaySEByKey("decide")
	m.beginItemTarget(items[m.itemListIndex].ItemID)
}

func (m *MenuScene) beginItemTarget(itemID string) {
	m.pendingItemID = itemID
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

	items := m.menuItems()
	m.syncItemListFirstVisible(len(items))
	if m.updateItemListScrollBar(len(items)) {
		return
	}
	_, wheelY := ebiten.Wheel()
	m.scrollItemListByWheel(len(items), wheelY)
	if idx, ok := m.hitTestItemListRows(len(items)); ok {
		if items[idx].ItemID != m.pendingItemID {
			if !itemUsableInField(items[idx]) {
				return
			}
			m.game.Audio.PlaySEByKey("decide")
			m.itemListIndex = idx
			m.syncItemListFirstVisible(len(items))
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
		for i := range partySize {
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

	if m.game.ItemCount(m.pendingItemID) <= 0 {
		m.pendingItemID = ""
		m.clampItemListIndex(len(m.menuItems()))
		m.menuState = menuStateItemList
	}
}
