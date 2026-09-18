package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func (s *BattleScene) currentItemDescription() string {
	items := s.battleUsableItems()
	if s.itemIndex < 0 || s.itemIndex >= len(items) {
		return ""
	}
	def, ok := GetItemDef(items[s.itemIndex].ItemID)
	if !ok {
		return ""
	}
	return def.Description
}

func (s *BattleScene) drawItemSubMenu(screen *ebiten.Image) {
	windowX, windowY, windowW, _ := s.battleSubPanelOrigin()

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(windowX, windowY)
	screen.DrawImage(s.game.SkillPanelImg, op)

	items := s.battleUsableItems()
	if len(items) == 0 {
		return
	}

	for i, slot := range items {
		def, ok := GetItemDef(slot.ItemID)
		if !ok {
			continue
		}
		labelCol := uiColorText
		selected := i == s.itemIndex
		if selected {
			labelCol = uiColorSelect
		}

		itemFace := s.game.FontFace(18)
		baseX := windowX + battleSubLabelOffsetX
		baseY := windowY + battleSubLabelOffsetY + float64(i)*battleSubRowHeight
		if selected {
			arrowOp := &text.DrawOptions{}
			arrowOp.GeoM.Translate(baseX, baseY)
			arrowOp.ColorScale.ScaleWithColor(labelCol)
			text.Draw(screen, "▶", itemFace, arrowOp)
		}
		op := &text.DrawOptions{}
		op.GeoM.Translate(baseX+text.Advance("▶ ", itemFace), baseY)
		op.ColorScale.ScaleWithColor(labelCol)
		text.Draw(screen, def.Name, itemFace, op)

		countOp := &text.DrawOptions{}
		countOp.GeoM.Translate(windowX+windowW-battleSubRightOffsetX, baseY)
		countOp.PrimaryAlign = text.AlignEnd
		countOp.ColorScale.ScaleWithColor(labelCol)
		text.Draw(screen, fmt.Sprintf("x%d", slot.Count), itemFace, countOp)
	}
}

func (s *BattleScene) drawItemTargetUI(screen *ebiten.Image) {
	isAll := s.itemTargetIndex == partySize

	def, ok := GetItemDef(s.pendingItemID)
	allowAll := ok && (def.Target == TargetAll || def.Target == TargetBoth)
	if allowAll {
		s.drawAllTargetRow(screen, isAll)
	}

	for i := 0; i < partySize; i++ {
		centerX := s.partyScreenX[i]
		centerY := s.partyScreenY[i]

		showArrow := isAll || s.itemTargetIndex == i
		if !showArrow {
			continue
		}

		arrowOp := &text.DrawOptions{}
		arrowOp.GeoM.Translate(centerX-16, centerY+20)
		arrowOp.ColorScale.ScaleWithColor(uiColorText)
		text.Draw(screen, "▶", s.game.FontFace(15), arrowOp)
	}
}

func (s *BattleScene) itemTargetDescriptionAndHint() (string, string) {
	def, ok := GetItemDef(s.pendingItemID)
	if !ok {
		return "", ""
	}
	return def.Description, ""
}
