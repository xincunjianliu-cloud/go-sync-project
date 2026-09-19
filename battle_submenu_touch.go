package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	battleSubCmdCenterX = 860.0
	battleSubCmdCenterY = 440.0

	battleSubLabelOffsetX = 8.0
	battleSubLabelOffsetY = 22.0
	battleSubRowHeight    = 35.0
	battleSubRightOffsetX = 20.0

	battleSubLvBlockRightX = 66.0
	battleSubLvGap         = 2.0
	battleSubLvHitPadX     = 5.0

	skillLvLeftArrow  = "◀"
	skillLvRightArrow = "▶"
)

func skillLvText(lv int) string {
	return fmt.Sprintf("Lv%d", lv)
}

func (s *BattleScene) battleSubPanelOrigin() (windowX, windowY, windowW, windowH float64) {
	windowW = float64(s.game.SkillPanelImg.Bounds().Dx())
	windowH = float64(s.game.SkillPanelImg.Bounds().Dy())
	windowX = battleSubCmdCenterX - windowW/2
	windowY = battleSubCmdCenterY - windowH/2
	return
}

func (s *BattleScene) battleSubPanelRect() tapRect {
	x, y, w, h := s.battleSubPanelOrigin()
	return tapRect{x: x, y: y, w: w, h: h}
}

func (s *BattleScene) isTapOutsideBattleSubPanel() bool {
	pts := justPressedTouchPoints()
	if len(pts) == 0 {
		return false
	}
	r := s.battleSubPanelRect()
	for _, p := range pts {
		if !r.contains(p) {
			return true
		}
	}
	return false
}

func (s *BattleScene) battleSubRowRect(row int) tapRect {
	windowX, windowY, windowW, _ := s.battleSubPanelOrigin()
	y := windowY + battleSubLabelOffsetY + float64(row)*battleSubRowHeight
	return tapRect{
		x: windowX,
		y: y - battleSubRowHeight/2,
		w: windowW - battleSubRightOffsetX,
		h: battleSubRowHeight,
	}
}

func (s *BattleScene) hitTestBattleSubRows(rowCount int) (int, bool) {
	rects := make([]tapRect, rowCount)
	for i := 0; i < rowCount; i++ {
		rects[i] = s.battleSubRowRect(i)
	}
	return hitTestTapRects(rects)
}

func (s *BattleScene) skillLevelArrowRects(row, lv int, face *text.GoTextFace) (leftRect, rightRect tapRect, leftX, lvX, rightX, textY float64) {
	windowX, windowY, windowW, _ := s.battleSubPanelOrigin()
	textY = windowY + battleSubLabelOffsetY + float64(row)*battleSubRowHeight

	lvW := text.Advance(skillLvText(lv), face)
	leftArrowW := text.Advance(skillLvLeftArrow, face)
	rightArrowW := text.Advance(skillLvRightArrow, face)

	blockRightX := windowX + windowW - battleSubLvBlockRightX
	rightX = blockRightX - rightArrowW
	lvX = rightX - battleSubLvGap - lvW
	leftX = lvX - battleSubLvGap - leftArrowW

	rowTopY := textY - battleSubRowHeight/2
	leftRect = tapRect{x: leftX - battleSubLvHitPadX, y: rowTopY, w: leftArrowW + battleSubLvHitPadX*2, h: battleSubRowHeight}
	rightRect = tapRect{x: rightX - battleSubLvHitPadX, y: rowTopY, w: rightArrowW + battleSubLvHitPadX*2, h: battleSubRowHeight}
	return
}

func (s *BattleScene) handleSkillLevelArrowTaps(p int, skills []SkillDef) bool {
	pts := justPressedTouchPoints()
	if len(pts) == 0 {
		return false
	}
	face := s.game.FontFace(15)
	for i, sk := range skills {
		if !s.game.IsSkillUnlocked(p, i) {
			continue
		}
		curLv := s.game.PlayerSkillLv[p][i]
		if curLv < 1 {
			curLv = 1
		}
		if curLv > len(sk.Levels) {
			curLv = len(sk.Levels)
		}
		if curLv <= 1 {
			continue
		}
		lv := s.skillLevelCursors[p][i]
		if lv < 1 {
			lv = 1
		}
		if lv > curLv {
			lv = curLv
		}
		leftRect, rightRect, _, _, _, _ := s.skillLevelArrowRects(i, lv, face)
		for _, pt := range pts {
			if leftRect.contains(pt) {
				if lv > 1 {
					s.skillLevelCursors[p][i] = lv - 1
					s.lastSkillLevel[p][i] = lv - 1
					s.game.Audio.PlaySEByKey("cursor")
				}
				return true
			}
			if rightRect.contains(pt) {
				if lv < curLv {
					s.skillLevelCursors[p][i] = lv + 1
					s.lastSkillLevel[p][i] = lv + 1
					s.game.Audio.PlaySEByKey("cursor")
				}
				return true
			}
		}
	}
	return false
}

const (
	allTargetRowGapY = 10.0
	allTargetRowH    = 30.0
	allTargetRowPadX = 12.0
)

func (s *BattleScene) allTargetRowRect() tapRect {
	left := s.partyScreenX[0]
	right := s.partyScreenX[0] + spriteFrameW
	bottom := s.partyScreenY[0] + spriteFrameH
	for i := 1; i < partySize; i++ {
		if s.partyScreenX[i] < left {
			left = s.partyScreenX[i]
		}
		if x := s.partyScreenX[i] + spriteFrameW; x > right {
			right = x
		}
		if y := s.partyScreenY[i] + spriteFrameH; y > bottom {
			bottom = y
		}
	}
	return tapRect{
		x: left - allTargetRowPadX,
		y: bottom + allTargetRowGapY,
		w: (right - left) + allTargetRowPadX*2,
		h: allTargetRowH,
	}
}

// drawAllTargetBox renders the "全体" selection box at rect r. Both the ally
// (heal) target UI and the enemy (attack) target UI call this same function
// so they always render identically.
func (s *BattleScene) drawAllTargetBox(screen *ebiten.Image, r tapRect, selected bool) {
	col := uiColorText
	boxFillCol := color.RGBA{45, 45, 55, 200}
	if selected {
		col = uiColorSelect
		boxFillCol = color.RGBA{41, 58, 94, 220}
	}

	ebitenutil.DrawRect(screen, r.x, r.y, r.w, r.h, boxFillCol)

	face := s.game.FontFace(15)
	label := "全体"
	centerX := r.x + r.w/2
	centerY := r.y + r.h/2

	if selected {
		const bw = 2.0
		ebitenutil.DrawRect(screen, r.x, r.y, r.w, bw, col)
		ebitenutil.DrawRect(screen, r.x, r.y+r.h-bw, r.w, bw, col)
		ebitenutil.DrawRect(screen, r.x, r.y, bw, r.h, col)
		ebitenutil.DrawRect(screen, r.x+r.w-bw, r.y, bw, r.h, col)

		labelW := text.Advance(label, face)
		arrowOp := &text.DrawOptions{}
		arrowOp.GeoM.Translate(centerX-labelW/2-text.Advance("▶ ", face), centerY)
		arrowOp.SecondaryAlign = text.AlignCenter
		arrowOp.ColorScale.ScaleWithColor(col)
		text.Draw(screen, "▶", face, arrowOp)
	}

	op := &text.DrawOptions{}
	op.GeoM.Translate(centerX, centerY)
	op.PrimaryAlign = text.AlignCenter
	op.SecondaryAlign = text.AlignCenter
	op.ColorScale.ScaleWithColor(col)
	text.Draw(screen, label, face, op)
}

func (s *BattleScene) drawAllTargetRow(screen *ebiten.Image, selected bool) {
	s.drawAllTargetBox(screen, s.allTargetRowRect(), selected)
}

func partyPortraitHitOrder() [partySize]int {
	var order [partySize]int
	for i := 0; i < partySize; i++ {
		order[i] = partySize - 1 - i
	}
	return order
}

func (s *BattleScene) hitTestHealTargets() (int, bool) {
	order := partyPortraitHitOrder()
	rects := make([]tapRect, 0, partySize+1)
	for _, i := range order {
		rects = append(rects, s.partyPortraitRect(i))
	}
	rects = append(rects, s.allTargetRowRect())
	idx, ok := hitTestTapRects(rects)
	if !ok {
		return -1, false
	}
	if idx == partySize {
		return partySize, true
	}
	return order[idx], true
}

func (s *BattleScene) hitTestItemTargets(allowAll bool) (int, bool) {
	order := partyPortraitHitOrder()
	rects := make([]tapRect, 0, partySize+1)
	for _, i := range order {
		rects = append(rects, s.partyPortraitRect(i))
	}
	if allowAll {
		rects = append(rects, s.allTargetRowRect())
	}
	idx, ok := hitTestTapRects(rects)
	if !ok {
		return -1, false
	}
	if idx == partySize {
		return partySize, true
	}
	return order[idx], true
}

func (s *BattleScene) partyPortraitRect(i int) tapRect {
	return tapRect{
		x: s.partyScreenX[i] - 8,
		y: s.partyScreenY[i] - 8,
		w: spriteFrameW + 16,
		h: spriteFrameH + 24,
	}
}

func (s *BattleScene) enemyTargetRect(slot int) (tapRect, bool) {
	if slot < 0 || slot >= len(s.enemies) {
		return tapRect{}, false
	}
	e := &s.enemies[slot]
	if e.Image == nil || e.HP <= 0 {
		return tapRect{}, false
	}
	x, y, w, h := s.enemyDrawRect(slot)
	return tapRect{x: x, y: y, w: w, h: h}, true
}

const (
	enemyAllTargetRowGapY = 10.0
	enemyAllTargetRowH    = 30.0
	enemyAllTargetRowPadX = 12.0
)

func (s *BattleScene) enemyAllTargetRowRect() tapRect {
	alive := s.aliveEnemyIndices()
	if len(alive) == 0 {
		return tapRect{}
	}
	left, top, w0, h0 := s.enemyDrawRect(alive[0])
	right := left + w0
	bottom := top + h0
	for _, i := range alive[1:] {
		x, y, w, h := s.enemyDrawRect(i)
		if x < left {
			left = x
		}
		if x+w > right {
			right = x + w
		}
		if y+h > bottom {
			bottom = y + h
		}
	}
	return tapRect{
		x: left - enemyAllTargetRowPadX,
		y: bottom + enemyAllTargetRowGapY,
		w: (right - left) + enemyAllTargetRowPadX*2,
		h: enemyAllTargetRowH,
	}
}

func (s *BattleScene) drawEnemyAllTargetRow(screen *ebiten.Image, selected bool) {
	r := s.enemyAllTargetRowRect()
	if r.w <= 0 {
		return
	}
	s.drawAllTargetBox(screen, r, selected)
}

func (s *BattleScene) currentSkillTargetType() SkillTarget {
	p := s.waitingActor
	if p < 0 || p >= partySize || s.pendingSkill < 1 {
		return TargetSingle
	}
	skillIdx := s.pendingSkill - 1
	skills := s.game.CharacterSkills(p)
	if skillIdx < 0 || skillIdx >= len(skills) {
		return TargetSingle
	}
	return s.game.CurrentSkillLevelData(p, skillIdx).Target
}

func (s *BattleScene) currentTargetAllowsAll() bool {
	t := s.currentSkillTargetType()
	if t != TargetAll && t != TargetBoth {
		return false
	}
	return len(s.aliveEnemyIndices()) > 1
}

func (s *BattleScene) currentTargetIsForcedAll() bool {
	return s.currentSkillTargetType() == TargetAll
}

func (s *BattleScene) currentAttackIsAllTarget() bool {
	switch s.currentSkillTargetType() {
	case TargetAll:
		return true
	case TargetBoth:
		return s.targetIndex == maxEnemies
	default:
		return false
	}
}

func (s *BattleScene) hitTestEnemyTarget() (int, bool) {
	alive := s.aliveEnemyIndices()
	forcedAll := s.currentTargetIsForcedAll()
	rects := make([]tapRect, 0, len(alive)+1)
	slots := make([]int, 0, len(alive)+1)
	for idx := len(alive) - 1; idx >= 0; idx-- {
		i := alive[idx]
		r, ok := s.enemyTargetRect(i)
		if !ok {
			continue
		}
		rects = append(rects, r)
		if forcedAll {
			slots = append(slots, maxEnemies)
		} else {
			slots = append(slots, i)
		}
	}
	if s.currentTargetAllowsAll() {
		rects = append(rects, s.enemyAllTargetRowRect())
		slots = append(slots, maxEnemies)
	}
	idx, ok := hitTestTapRects(rects)
	if !ok {
		return -1, false
	}
	return slots[idx], true
}
