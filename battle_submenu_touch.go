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

	// battleSkillPanelScale enlarges the skill submenu in the skill-upgrade
	// tutorial's mockup only, so the level number and its ◀▶ switch arrows
	// read more clearly there - see drawSkillSubMenuEnlarged
	// (battle_draw_hud.go). Real battles draw the panel at normal size.
	battleSkillPanelScale = 1.4
)

func skillLvText(lv int) string {
	return fmt.Sprintf("Lv%d", lv)
}

// battleSkillPanelPivot is the point drawSkillSubMenuEnlarged scales the panel
// around: its unscaled right edge (windowX+windowW), which normally sits
// almost flush against the screen's right edge (gameWidth=960), paired with
// the panel's usual vertical center. Scaling around the panel's own center
// pushed that right edge - and the MP-cost text/border right against it -
// off-screen; anchoring here instead means the panel only grows left and
// up/down, into the open space around it, never past the screen edge.
func (s *BattleScene) battleSkillPanelPivot() (px, py float64) {
	windowX, _, windowW, _ := s.battleSubPanelOrigin()
	return windowX + windowW, battleSubCmdCenterY
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

func (s *BattleScene) skillLevelArrowRects(row, lv int, face text.Face) (leftRect, rightRect tapRect, leftX, lvX, rightX, textY float64) {
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
	// The "全体" boxes share a fixed size. The ally box sits at a fixed
	// position just under the party (allyAllTargetRowY is its top edge). The
	// enemy box keeps a fixed horizontal center, but its top edge sits
	// enemyAllTargetRowGapY below the lowest enemy of the battle's initial
	// formation (dead enemies included), so the gap looks the same for 2-4
	// enemies and the box never moves mid-battle.
	allTargetRowW            = 120.0
	allTargetRowH            = 24.0
	allyAllTargetRowY        = 402.0
	allyAllTargetRowCenterX  = 720.0
	enemyAllTargetRowCenterX = 240.0
	enemyAllTargetRowGapY    = 8.0

	// allTargetLabelFontSize sizes the "全体" label drawn inside the box by
	// drawAllTargetBox (shared by the ally and enemy rows).
	allTargetLabelFontSize = 18.0
)

// textInkCenterY returns the vertical center of str's actually drawn pixels,
// relative to the y passed to text.Draw with the default (top) alignment.
// text.AlignCenter centers the font's whole ascent+descent box instead, which
// leaves glyphs visibly off-center inside a tight box like the "全体" row.
func textInkCenterY(str string, face text.Face) float64 {
	top, bottom := 0.0, 0.0
	found := false
	for _, g := range text.AppendGlyphs(nil, str, face, nil) {
		if g.Image == nil {
			continue
		}
		gTop := g.Y
		gBottom := g.Y + float64(g.Image.Bounds().Dy())
		if !found || gTop < top {
			top = gTop
		}
		if !found || gBottom > bottom {
			bottom = gBottom
		}
		found = true
	}
	if !found {
		m := face.Metrics()
		return (m.HAscent + m.HDescent) / 2
	}
	return (top + bottom) / 2
}

func (s *BattleScene) allTargetRowRect() tapRect {
	return tapRect{
		x: allyAllTargetRowCenterX - allTargetRowW/2,
		y: allyAllTargetRowY,
		w: allTargetRowW,
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

	face := s.game.FontFace(allTargetLabelFontSize)
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
		arrowOp.GeoM.Translate(centerX-labelW/2-text.Advance("▶ ", face), centerY-textInkCenterY("▶", face))
		arrowOp.ColorScale.ScaleWithColor(col)
		text.Draw(screen, "▶", face, arrowOp)
	}

	op := &text.DrawOptions{}
	op.GeoM.Translate(centerX, centerY-textInkCenterY(label, face))
	op.PrimaryAlign = text.AlignCenter
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

// partyPortraitRowSpan scans frame-local row ly of party member i's current
// sprite frame and returns the leftmost/rightmost frame-local X holding a
// non-transparent pixel. ok is false if the row has no visible pixels.
func (s *BattleScene) partyPortraitRowSpan(i, ly int) (minX, maxX int, ok bool) {
	spriteSheet, srcRect, srcOk := s.partySpriteSrcRect(i)
	if !srcOk || ly < 0 || ly >= spriteFrameH {
		return 0, 0, false
	}
	minX, maxX = -1, -1
	for x := 0; x < spriteFrameW; x++ {
		_, _, _, a := spriteSheet.At(srcRect.Min.X+x, srcRect.Min.Y+ly).RGBA()
		if a > 0 {
			if minX == -1 {
				minX = x
			}
			maxX = x
		}
	}
	return minX, maxX, minX != -1
}

// partyPortraitPixelHit reports whether tap point p lands within party member
// i's visible silhouette on its currently displayed sprite frame. This is
// used (ahead of the padded bounding-box test) so that overlapping portraits
// in the diagonal party layout resolve to whichever character's visible art
// was actually tapped, not just whichever bounding box happens to be on top.
//
// Rather than requiring the exact tapped pixel to be opaque, it checks
// whether the tap falls between the leftmost and rightmost drawn pixels on
// that row. This keeps clicks working in small internal gaps in the art
// (e.g. the transparent space between a character's legs) while still
// excluding the empty margin outside the actual silhouette.
func (s *BattleScene) partyPortraitPixelHit(i int, p touchPoint) bool {
	lx := int(p.x - s.partyScreenX[i])
	ly := int(p.y - s.partyScreenY[i])
	if lx < 0 || ly < 0 || lx >= spriteFrameW || ly >= spriteFrameH {
		return false
	}
	minX, maxX, ok := s.partyPortraitRowSpan(i, ly)
	if !ok {
		return false
	}
	return lx >= minX && lx <= maxX
}

func (s *BattleScene) hitTestHealTargets() (int, bool) {
	return s.hitTestPartyTargets(true)
}

func (s *BattleScene) hitTestItemTargets(allowAll bool) (int, bool) {
	return s.hitTestPartyTargets(allowAll)
}

// hitTestPartyTargets resolves a tap/click against the party portraits. It
// first checks pixel-accurate hits front-to-back (so a rear character's
// visible feet win over a front character's mostly-transparent padding at
// that same point). Touch input then falls back to the padded bounding
// boxes for forgiving taps in the empty margin around a sprite; a mouse
// cursor is precise enough that clicks only register on the drawn art.
func (s *BattleScene) hitTestPartyTargets(allowAll bool) (int, bool) {
	order := partyPortraitHitOrder()
	touches, mouse := justPressedTouchAndMousePoints()

	for _, p := range touches {
		if i, ok := s.hitTestPartyTargetPoint(p, order, true, allowAll); ok {
			return i, true
		}
	}
	for _, p := range mouse {
		if i, ok := s.hitTestPartyTargetPoint(p, order, false, allowAll); ok {
			return i, true
		}
	}
	return -1, false
}

func (s *BattleScene) hitTestPartyTargetPoint(p touchPoint, order [partySize]int, allowPadded, allowAll bool) (int, bool) {
	for _, i := range order {
		if s.partyPortraitPixelHit(i, p) {
			return i, true
		}
	}
	if allowPadded {
		for _, i := range order {
			if s.partyPortraitRect(i).contains(p) {
				return i, true
			}
		}
	}
	if allowAll && s.allTargetRowRect().contains(p) {
		return partySize, true
	}
	return -1, false
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

func (s *BattleScene) enemyAllTargetRowRect() tapRect {
	// enemyDrawRect lays slots out by the total enemy count, so every slot
	// (alive or not) keeps its spot for the whole battle.
	bottom := 0.0
	for i := range s.enemies {
		_, y, _, h := s.enemyDrawRect(i)
		if y+h > bottom {
			bottom = y + h
		}
	}
	return tapRect{
		x: enemyAllTargetRowCenterX - allTargetRowW/2,
		y: bottom + enemyAllTargetRowGapY,
		w: allTargetRowW,
		h: allTargetRowH,
	}
}

func (s *BattleScene) drawEnemyAllTargetRow(screen *ebiten.Image, selected bool) {
	s.drawAllTargetBox(screen, s.enemyAllTargetRowRect(), selected)
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

// enemyPortraitRowSpan scans image-local row ly of the enemy in slot and
// returns the leftmost/rightmost image-local X holding a non-transparent
// pixel. ok is false if the row has no visible pixels.
func (s *BattleScene) enemyPortraitRowSpan(slot, ly int) (minX, maxX int, ok bool) {
	if slot < 0 || slot >= len(s.enemies) {
		return 0, 0, false
	}
	e := &s.enemies[slot]
	if e.Image == nil || e.HP <= 0 {
		return 0, 0, false
	}
	b := e.Image.Bounds()
	if ly < 0 || ly >= b.Dy() {
		return 0, 0, false
	}
	minX, maxX = -1, -1
	for x := 0; x < b.Dx(); x++ {
		_, _, _, a := e.Image.At(b.Min.X+x, b.Min.Y+ly).RGBA()
		if a > 0 {
			if minX == -1 {
				minX = x
			}
			maxX = x
		}
	}
	return minX, maxX, minX != -1
}

// enemyPortraitPixelHit mirrors partyPortraitPixelHit for enemies: it reports
// whether tap point p falls within the enemy's visible silhouette on that
// row (leftmost to rightmost drawn pixel), rather than requiring the exact
// tapped pixel to be opaque, so small internal gaps in the art don't create
// unclickable holes.
func (s *BattleScene) enemyPortraitPixelHit(slot int, p touchPoint) bool {
	if slot < 0 || slot >= len(s.enemies) {
		return false
	}
	e := &s.enemies[slot]
	if e.Image == nil || e.HP <= 0 {
		return false
	}
	x, y, w, h := s.enemyDrawRect(slot)
	lx := p.x - x
	ly := p.y - y
	if lx < 0 || ly < 0 || lx >= w || ly >= h {
		return false
	}
	minX, maxX, ok := s.enemyPortraitRowSpan(slot, int(ly))
	if !ok {
		return false
	}
	return int(lx) >= minX && int(lx) <= maxX
}

// hitTestEnemyTarget resolves a tap/click against the enemies, using the same
// two-tier approach as hitTestPartyTargets: pixel-accurate hits first, then
// (for touch only) a forgiving bounding-box fallback. Mouse clicks only
// register on the drawn art.
func (s *BattleScene) hitTestEnemyTarget() (int, bool) {
	alive := s.aliveEnemyIndices()
	forcedAll := s.currentTargetIsForcedAll()
	touches, mouse := justPressedTouchAndMousePoints()

	for _, p := range touches {
		if slot, ok := s.hitTestEnemyTargetPoint(p, alive, forcedAll, true); ok {
			return slot, true
		}
	}
	for _, p := range mouse {
		if slot, ok := s.hitTestEnemyTargetPoint(p, alive, forcedAll, false); ok {
			return slot, true
		}
	}
	return -1, false
}

func (s *BattleScene) hitTestEnemyTargetPoint(p touchPoint, alive []int, forcedAll, allowPadded bool) (int, bool) {
	slotFor := func(i int) int {
		if forcedAll {
			return maxEnemies
		}
		return i
	}

	for idx := len(alive) - 1; idx >= 0; idx-- {
		i := alive[idx]
		if s.enemyPortraitPixelHit(i, p) {
			return slotFor(i), true
		}
	}

	if allowPadded {
		for idx := len(alive) - 1; idx >= 0; idx-- {
			i := alive[idx]
			if r, ok := s.enemyTargetRect(i); ok && r.contains(p) {
				return slotFor(i), true
			}
		}
	}

	if s.currentTargetAllowsAll() && s.enemyAllTargetRowRect().contains(p) {
		return maxEnemies, true
	}

	return -1, false
}
