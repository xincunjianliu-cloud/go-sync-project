package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	itemRowStartY     = 75.0
	itemRowGapY       = 55.0
	itemCountFontSize = 25.0
	itemCountX        = 915.0

	itemScrollBarRightGap = 12.0
	itemScrollBarMinH     = 24.0
	itemScrollBarTrackH   = 440.0
)

func (m *MenuScene) drawItemListMenu(screen *ebiten.Image) {
	items := m.menuItems()
	if len(items) == 0 {
		return
	}

	firstVisible, lastVisible := menuItemVisibleRange(len(items), m.itemListIndex, m.itemListFirstVisible)
	for i := firstVisible; i < lastVisible; i++ {
		slot := items[i]
		def, ok := GetItemDef(slot.ItemID)
		if !ok {
			continue
		}
		rowCenterY := itemRowStartY + float64(i-firstVisible)*itemRowGapY

		selected := false
		switch m.menuState {
		case menuStateItemList:
			selected = i == m.itemListIndex
		case menuStateItemTarget:
			selected = slot.ItemID == m.pendingItemID
		}

		nameCol := uiColorText
		if !def.UsableInField {
			nameCol = uiColorDisabled
		} else if selected {
			nameCol = uiColorSelect
		}

		nameFace := m.game.FontFace(skillNameFontSize)
		if selected {
			arrowOp := &text.DrawOptions{}
			arrowOp.GeoM.Translate(skillNameX, rowCenterY)
			arrowOp.SecondaryAlign = text.AlignCenter
			arrowOp.ColorScale.ScaleWithColor(nameCol)
			text.Draw(screen, "▶", nameFace, arrowOp)
		}
		nameOp := &text.DrawOptions{}
		nameOp.GeoM.Translate(skillNameX+text.Advance("▶ ", nameFace), rowCenterY)
		nameOp.SecondaryAlign = text.AlignCenter
		nameOp.ColorScale.ScaleWithColor(nameCol)
		text.Draw(screen, def.Name, nameFace, nameOp)

		countOp := &text.DrawOptions{}
		countOp.GeoM.Translate(itemCountX, rowCenterY)
		countOp.SecondaryAlign = text.AlignCenter
		countOp.PrimaryAlign = text.AlignEnd
		countOp.ColorScale.ScaleWithColor(nameCol)
		text.Draw(screen, fmt.Sprintf("x%d", slot.Count), m.game.FontFace(itemCountFontSize), countOp)
	}

	barX, trackY, trackHeight, thumbY, thumbH, ok := menuItemScrollBarGeometry(len(items), firstVisible, lastVisible)
	if ok {
		fillRect(screen, barX, trackY, scrollBarTrackWidth, trackHeight, scrollBarTrackColor)
		fillRect(screen, barX, thumbY, scrollBarCursorWidth, thumbH, scrollBarCursorColor)
	}
}
