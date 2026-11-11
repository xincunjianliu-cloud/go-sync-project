package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	itemRowStartY     = 42.0
	itemRowGapY       = 37.5
	itemCountFontSize = 25.0
	itemCountX        = 922.0
)

func (m *MenuScene) drawItemListMenu(screen *ebiten.Image) {
	items := m.menuItems()
	if len(items) == 0 {
		return
	}

	for i, slot := range items {
		def, ok := GetItemDef(slot.ItemID)
		if !ok {
			continue
		}
		rowCenterY := itemRowStartY + float64(i)*itemRowGapY

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
}
