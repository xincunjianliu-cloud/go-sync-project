package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	itemRowGapY       = 40.0
	itemCountFontSize = 28.0
	itemCountX        = 922.0
)

func (m *MenuScene) drawItemListMenu(screen *ebiten.Image) {
	items := m.usableFieldItems()
	if len(items) == 0 {
		return
	}

	for i, slot := range items {
		def, ok := GetItemDef(slot.ItemID)
		if !ok {
			continue
		}
		rowCenterY := skillRowStartY + float64(i)*itemRowGapY

		selected := false
		if m.menuState == menuStateItemList {
			selected = i == m.itemListIndex
		} else if m.menuState == menuStateItemTarget {
			selected = slot.ItemID == m.pendingItemID
		}

		nameCol := uiColorText
		if selected {
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
		text.Draw(screen, fmt.Sprintf("x%d", slot.Count), m.game.LatinFontFace(itemCountFontSize), countOp)
	}
}
