package main

// battle_draw_items.go: バトル中のアイテム選択・使用対象選択の描画
// スキルサブメニュー(drawSkillSubMenu)と同じ画像・レイアウトを流用する。

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
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
	cmdCenterX := 860.0
	cmdCenterY := 440.0

	windowW := 140.0
	windowH := 72.0
	if s.game.SkillPanelImg != nil {
		windowW = float64(s.game.SkillPanelImg.Bounds().Dx())
		windowH = float64(s.game.SkillPanelImg.Bounds().Dy())
	}

	windowX := cmdCenterX - windowW/2
	windowY := cmdCenterY - windowH/2

	if s.game.SkillPanelImg != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(windowX, windowY)
		screen.DrawImage(s.game.SkillPanelImg, op)
	} else {
		ebitenutil.DrawRect(screen, windowX, windowY, windowW, windowH, color.RGBA{10, 10, 20, 240})
	}

	const (
		labelOffsetX = 15.0
		labelOffsetY = 20.0
		rowHeight    = 35.0
		countOffsetX = 20.0
	)

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

		itemFace := s.game.FontFace(15)
		baseX := windowX + labelOffsetX
		baseY := windowY + labelOffsetY + float64(i)*rowHeight
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
		countOp.GeoM.Translate(windowX+windowW-countOffsetX, windowY+labelOffsetY+float64(i)*rowHeight)
		countOp.PrimaryAlign = text.AlignEnd
		countOp.ColorScale.ScaleWithColor(labelCol)
		text.Draw(screen, fmt.Sprintf("x%d", slot.Count), s.game.FontFace(15), countOp)
	}
}

func (s *BattleScene) drawItemTargetUI(screen *ebiten.Image) {
	isAll := s.itemTargetIndex == partySize

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
	hint := ""
	if def.Target == TargetAll || def.Target == TargetBoth {
		if s.itemTargetIndex == partySize {
			hint = "←:個人選択に戻す"
		} else {
			hint = "→:全体に切替"
		}
	}
	return def.Description, hint
}
