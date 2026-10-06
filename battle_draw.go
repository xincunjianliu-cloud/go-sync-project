package main

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func (s *BattleScene) Draw(screen *ebiten.Image) {
	if s.tutorialActive {
		s.drawBattleTutorial(screen)
		return
	}

	s.drawBackground(screen)
	s.drawEnemyHeader(screen)
	s.drawPartySprites(screen)

	if s.battlePhase == phaseBattleEnd && s.isWon {
		if s.resultSubPhase >= resSubWinPose {
		} else {
			s.drawUI(screen)
		}

		if s.resultSubPhase >= resSubResultFadeIn {
			s.drawResultPanel(screen)
		}

		if s.resultFadeAlpha > 0 {
			alphaByte := uint8(255 * s.resultFadeAlpha)
			fillRect(screen, 0, 0, float64(gameWidth), float64(gameHeight),
				color.RGBA{0, 0, 0, alphaByte})
		}
	} else {
		s.drawUI(screen)
	}
}

func (s *BattleScene) drawBackground(screen *ebiten.Image) {
	if bg := s.battleBgImage(); bg != nil {
		screen.DrawImage(bg, nil)
		return
	}
	screen.Fill(color.RGBA{20, 20, 25, 255})
}

func (s *BattleScene) drawEnemyHeader(screen *ebiten.Image) {
	for i := range s.enemies {
		e := &s.enemies[i]
		if e.DeathPhase == 3 || e.DeathPhase == 2 {
			continue
		}
		if e.Image == nil {
			continue
		}

		x, y, _, _ := s.enemyDrawRect(i)

		var g ebiten.GeoM
		g.Translate(x+s.shakeX, y+s.shakeY)
		if e.DeathPhase == 1 {
			op := &ebiten.DrawImageOptions{GeoM: g}
			op.ColorScale.ScaleAlpha(float32(e.Alpha))
			screen.DrawImage(e.Image, op)
			continue
		}

		intensity := 0.0
		if e.HitFlashTimer > 0 {
			r := e.HitFlashTimer / spriteFlashDuration
			intensity = r * r
		}
		drawWithHitFlash(screen, e.Image, g, intensity)
		if e.Scanned {
			s.drawEnemyScanInfo(screen, i)
		}
	}

	for _, p := range s.deathParticles {
		a := uint8(255 * p.Life)
		size := p.Size
		fillRect(screen, p.X-size/2+s.shakeX, p.Y-size/2+s.shakeY, size, size,
			color.RGBA{255, 255, 255, a})
	}
}

// partySpriteSrcRect computes the current sprite sheet and the sub-rect of
// the frame currently being displayed for party member i. Shared by drawing
// and by pixel-accurate touch hit-testing so both always agree on which
// frame is on screen.
func (s *BattleScene) partySpriteSrcRect(i int) (spriteSheet *ebiten.Image, srcRect image.Rectangle, ok bool) {
	spriteSheet = s.game.PlayerAttackSprites[i]
	if spriteSheet == nil {
		return nil, image.Rectangle{}, false
	}

	row, frame := s.partySpriteRowFrame(i)
	srcX := frame * battleSpriteCellW
	srcY := row * battleSpriteCellH
	srcRect = image.Rect(srcX, srcY, srcX+battleSpriteCellW, srcY+battleSpriteCellH)

	if !srcRect.In(spriteSheet.Bounds()) {
		srcRect = image.Rect(0, 0, battleSpriteCellW, battleSpriteCellH)
	}

	return spriteSheet, srcRect, true
}

// partySpriteRowFrame は味方iの今のポーズで表示する、スプライトシートの行とコマ。
func (s *BattleScene) partySpriteRowFrame(i int) (row, frame int) {
	frames := func(row int) int {
		if row < 0 || row >= len(partySpriteRowFrames[i]) {
			return 1
		}
		return partySpriteRowFrames[i][row]
	}
	timer := s.playerAnimTimer[i]

	switch s.playerPose[i] {
	case poseDamage:
		return spriteRowDamage, loopFrame(frames(spriteRowDamage), timer)
	case poseWalk:
		return spriteRowWalk, loopFrame(frames(spriteRowWalk), timer)
	case poseReady:
		return spriteRowCommand, 0
	case poseReadyGlow:
		return spriteRowCommand, glowFrameForLevel(s.skillGlowLevel, timer)
	case poseAttack:
		return spriteRowAttack, actionFrame(frames(spriteRowAttack), s.attackPhaseTimer)
	case poseSkill:
		return s.actionRow[i], actionFrame(s.actionFrames[i], s.attackPhaseTimer)
	case poseHealCast:
		if i == s.healingCaster && s.healingAnimTimer[i] > 0 {
			n := s.actionFrames[i]
			return s.actionRow[i], actionFrame(n, actionAnimDuration(n)-s.healingAnimTimer[i])
		}
	case poseRecv:
		return s.recvRow[i], actionFrame(frames(s.recvRow[i]), timer)
	}
	return spriteRowIdle, loopFrame(frames(spriteRowIdle), timer)
}

// partySpriteOrigin は味方iのスプライトのマスを画面に描く左上の位置（揺れは含まない）。
// battleSpriteScale倍にしたマスの下端・左右中央を、画面上の枠
// （partyScreenX/Yから spriteFrameW×spriteFrameH）にそろえる。
func (s *BattleScene) partySpriteOrigin(i int) (x, y float64) {
	x = s.partyScreenX[i] + (spriteFrameW-battleSpriteCellW*battleSpriteScale)/2
	y = s.partyScreenY[i] + spriteFrameH - battleSpriteCellH*battleSpriteScale
	return x, y
}

func (s *BattleScene) drawPartySprites(screen *ebiten.Image) {
	for i := 0; i < partySize; i++ {
		spriteSheet, srcRect, ok := s.partySpriteSrcRect(i)
		if !ok {
			continue
		}
		pose := s.playerPose[i]

		frameImg := spriteSheet.SubImage(srcRect).(*ebiten.Image)

		op := &ebiten.DrawImageOptions{}
		baseX := 640.0
		baseY := 142.0
		centerX := baseX + float64(i)*32.0
		centerY := baseY + float64(i)*52.0

		centerX += s.readySlideX[i]
		centerX += s.introCharOffsetX
		centerX += s.evadeOffsetX[i]

		s.partyScreenX[i] = centerX
		s.partyScreenY[i] = centerY

		originX, originY := s.partySpriteOrigin(i)
		op.GeoM.Scale(battleSpriteScale, battleSpriteScale)
		op.GeoM.Translate(originX+s.shakeX, originY+s.shakeY)

		if pose == poseDead {
			pulse := float32(0.5 + 0.5*math.Sin(s.playerAnimTimer[i]*2.2))
			op.ColorScale.Scale(1.0, 1.0-pulse*0.85, 1.0-pulse*0.85, 1.0)
			screen.DrawImage(frameImg, op)
			continue
		}

		intensity := 0.0
		if s.playerFlashTimer[i] > 0 {
			r := s.playerFlashTimer[i] / spriteFlashDuration
			intensity = r * r
		}
		drawWithHitFlash(screen, frameImg, op.GeoM, intensity)
	}
}

func (s *BattleScene) drawUI(screen *ebiten.Image) {
	s.drawTimeline(screen)
	s.drawStatusBar(screen)
	s.drawGaugeTriangle(screen)

	if s.introActive {
		return
	}

	if s.battleLog != "" {
		s.drawLogWindowBackground(screen)
		s.drawBattleMessage(screen)
	}

	s.drawDirectMessages(screen)

	for _, pop := range s.damagePops {
		if pop.Timer < 0 {
			continue
		}
		alpha := uint8(255)
		if pop.Timer > damagePopHoldTime {
			fade := 1.0 - (pop.Timer-damagePopHoldTime)/(damagePopLifetime-damagePopHoldTime)
			if fade < 0 {
				fade = 0
			}
			alpha = uint8(255 * fade)
		}

		// Hit numbers punch upward once and settle back down (a single
		// hump, no repeated bouncing); heals keep the classic float-up-
		// and-fade motion (handled in Update).
		drawY := pop.Y
		if !pop.IsHeal {
			const wobbleDecay = 9.0
			wobbleAmp := math.Abs(pop.Vy) * 0.16
			t := wobbleDecay * pop.Timer
			drawY -= wobbleAmp * t * math.Exp(1.0-t)
		}

		msg := fmt.Sprintf("%d", pop.Value)
		numFontSize := 52.0
		base := damagePopColor(pop)
		mainColor := color.RGBA{base.R, base.G, base.B, alpha}
		if pop.Label != "" {
			msg = pop.Label
		} else if pop.IsMiss {
			msg = "MISS"
		} else if pop.IsCrit && !pop.IsHeal {
			numFontSize = 62.0
		}
		numFace := s.game.FontFace(numFontSize)

		for _, offset := range [][2]float64{{-2.0, 0}, {2.0, 0}, {0, -2.0}, {0, 2.0}} {
			shadowOp := &text.DrawOptions{}
			shadowOp.PrimaryAlign = text.AlignCenter
			shadowOp.SecondaryAlign = text.AlignCenter
			shadowOp.GeoM.Translate(pop.X+offset[0]+s.shakeX, drawY+offset[1]+s.shakeY)
			shadowOp.ColorScale.ScaleWithColor(color.RGBA{0, 0, 0, alpha})
			text.Draw(screen, msg, numFace, shadowOp)
		}
		op := &text.DrawOptions{}
		op.PrimaryAlign = text.AlignCenter
		op.SecondaryAlign = text.AlignCenter
		op.GeoM.Translate(pop.X+s.shakeX, drawY+s.shakeY)
		op.ColorScale.ScaleWithColor(mainColor)
		text.Draw(screen, msg, numFace, op)

		if pop.IsCrit {
			critFace := s.game.FontFace(24.0)
			const critLabelGapY = 48.0
			labelOp := &text.DrawOptions{}
			labelOp.PrimaryAlign = text.AlignCenter
			labelOp.SecondaryAlign = text.AlignCenter
			labelOp.GeoM.Translate(pop.X+1.0+s.shakeX, drawY-critLabelGapY+s.shakeY)
			labelOp.ColorScale.ScaleWithColor(color.RGBA{0, 0, 0, alpha})
			text.Draw(screen, "CRITICAL", critFace, labelOp)

			labelOp2 := &text.DrawOptions{}
			labelOp2.PrimaryAlign = text.AlignCenter
			labelOp2.SecondaryAlign = text.AlignCenter
			labelOp2.GeoM.Translate(pop.X+s.shakeX, drawY-critLabelGapY-1.0+s.shakeY)
			labelOp2.ColorScale.ScaleWithColor(color.RGBA{255, 130, 40, alpha})
			text.Draw(screen, "CRITICAL", critFace, labelOp2)
		}
	}

	switch s.battlePhase {
	case phasePlayerMenu:
		s.drawCommandMenu(screen, s.commandIndex)
		s.drawBattleShortcutButtons(screen)
		s.drawBottomDescription(screen, s.playerMenuDescription())
	case phaseSkillMenu:
		s.drawCommandMenu(screen, s.commandIndex)
		s.drawSkillSubMenu(screen)
		s.drawBottomDescription(screen, s.currentSkillDescription())
	case phaseTargetSelect:
		s.drawTargetSelectUI(screen)
		s.drawBottomDescription(screen, s.targetSelectDescription())
	case phaseHealSelect:
		s.drawHealTargetUI(screen)
		healDesc := ""
		if skillIdx := s.pendingSkill - 1; skillIdx >= 0 {
			healDesc = s.game.CurrentSkillLevelData(s.waitingActor, skillIdx).Description
		}
		s.drawBottomDescription(screen, healDesc)
	case phaseItemMenu:
		s.drawCommandMenu(screen, -1)
		ix, iy := itemButtonCenter()
		drawItemIcon(screen, s.game.ItemButtonImg, ix, iy, true)
		s.drawItemSubMenu(screen)
		s.drawBottomDescription(screen, s.currentItemDescription())
	case phaseItemTarget:
		s.drawItemTargetUI(screen)
		s.drawBottomDescription(screen, s.itemTargetDescription())
	}

	s.drawMyTurnOverlay(screen)
}

// playerMenuDescription returns the bottom description for the command menu.
// A shortcut button (item / rewind) that has been tapped once (armed) shows
// its own description instead of the highlighted command's, since the first
// tap only selects it and the second tap confirms.
func (s *BattleScene) playerMenuDescription() string {
	switch {
	case s.itemButtonArmed:
		return itemButtonDescription
	case s.rewindButtonArmed:
		return rewindDescription
	}
	return commandDescriptions[s.commandIndex]
}

func (s *BattleScene) currentSkillDescription() string {
	p := s.waitingActor
	if p >= 0 && p < partySize {
		return s.skillLevelDataAtCursor(p, s.skillIndex).Description
	}
	return ""
}

func (s *BattleScene) skillLevelDataAtCursor(p, skillIdx int) SkillLevelData {
	skills := s.game.CharacterSkills(p)
	if skillIdx < 0 || skillIdx >= len(skills) {
		return SkillLevelData{}
	}
	levels := skills[skillIdx].Levels
	if len(levels) == 0 {
		return SkillLevelData{}
	}
	lv := s.skillLevelCursors[p][skillIdx]
	if lv < 1 {
		lv = 1
	}
	if lv > len(levels) {
		lv = len(levels)
	}
	return levels[lv-1]
}

// Damage popup colors, all in one place.
var (
	popColorDamage = color.RGBA{255, 255, 255, 255} // damage dealt to enemies
	popColorCrit   = color.RGBA{255, 130, 40, 255}  // critical hit
	popColorHeal   = color.RGBA{140, 255, 160, 255} // HP recovery
	popColorMPHeal = color.RGBA{120, 190, 255, 255} // MP recovery
	popColorMiss   = color.RGBA{170, 170, 170, 255} // miss
)

// damagePopColor picks the popup color for pop (alpha is applied by the caller).
func damagePopColor(pop DamagePop) color.RGBA {
	switch {
	case pop.IsMiss:
		return popColorMiss
	case pop.IsMP:
		return popColorMPHeal
	case pop.IsHeal:
		return popColorHeal
	case pop.IsCrit:
		return popColorCrit
	}
	return popColorDamage
}

var elementNames = [elementalTypeCount]string{"炎", "雷", "氷", "風"}

// drawEnemyScanInfo shows the HP and elemental weaknesses (negative
// resistances) みやぶる revealed, centered above the enemy in slot.
func (s *BattleScene) drawEnemyScanInfo(screen *ebiten.Image, slot int) {
	e := &s.enemies[slot]
	x, y, w, _ := s.enemyDrawRect(slot)
	weak := ""
	for i, r := range e.ElementResist {
		if r < 0 {
			weak += elementNames[i]
		}
	}
	if weak == "" {
		weak = "なし"
	}
	msg := fmt.Sprintf("HP %d/%d\n弱点:%s", e.HP, e.MaxHP, weak)
	face := s.game.FontFace(15)
	for _, off := range [][2]float64{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
		op := &text.DrawOptions{}
		op.PrimaryAlign = text.AlignCenter
		op.SecondaryAlign = text.AlignEnd
		op.LineSpacing = 17
		op.GeoM.Translate(x+w/2+off[0]+s.shakeX, y-4+off[1]+s.shakeY)
		op.ColorScale.ScaleWithColor(color.Black)
		text.Draw(screen, msg, face, op)
	}
	op := &text.DrawOptions{}
	op.PrimaryAlign = text.AlignCenter
	op.SecondaryAlign = text.AlignEnd
	op.LineSpacing = 17
	op.GeoM.Translate(x+w/2+s.shakeX, y-4+s.shakeY)
	op.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, msg, face, op)
}
