package main

// menu_draw_item.go: メニュー画面のアイテム一覧・使用対象選択の描画
// スキル拡張パネル(メニュー画面拡張.png)と同じ画像・レイアウトを流用する。

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	itemListRowGapY       = 40.0 // 行間（スキル一覧のskillSubRowGapYより狭くする）
	itemListCountFontSize = 28.0 // 所持数(x個)の文字サイズ（名前より大きく）
)

func (m *MenuScene) drawItemListMenu(screen *ebiten.Image) {
	img := m.game.MenuSkillPanelImg
	winX := skillSubPanelX
	winY := skillSubPanelY

	if img != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(winX, winY)
		screen.DrawImage(img, op)
	}

	items := m.usableFieldItems()
	if len(items) == 0 {
		return
	}

	for i, slot := range items {
		def, ok := GetItemDef(slot.ItemID)
		if !ok {
			continue
		}
		rowCenterY := winY + skillSubRowStartOffsetY + float64(i)*itemListRowGapY

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

		nameFace := m.game.FontFace(20)
		if selected {
			arrowOp := &text.DrawOptions{}
			arrowOp.GeoM.Translate(winX+skillSubNameOffsetX, rowCenterY)
			arrowOp.SecondaryAlign = text.AlignCenter
			arrowOp.ColorScale.ScaleWithColor(nameCol)
			text.Draw(screen, "▶", nameFace, arrowOp)
		}
		nameOp := &text.DrawOptions{}
		nameOp.GeoM.Translate(winX+skillSubNameOffsetX+text.Advance("▶ ", nameFace), rowCenterY)
		nameOp.SecondaryAlign = text.AlignCenter
		nameOp.ColorScale.ScaleWithColor(nameCol)
		text.Draw(screen, def.Name, nameFace, nameOp)

		countOp := &text.DrawOptions{}
		countOp.GeoM.Translate(winX+skillSubSPOffsetX, rowCenterY)
		countOp.SecondaryAlign = text.AlignCenter
		countOp.PrimaryAlign = text.AlignEnd
		countOp.ColorScale.ScaleWithColor(nameCol)
		text.Draw(screen, fmt.Sprintf("x%d", slot.Count), m.game.FontFace(itemListCountFontSize), countOp)
	}
}
