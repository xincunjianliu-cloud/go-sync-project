package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func (s *BattleScene) Draw(screen *ebiten.Image) {
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
			ebitenutil.DrawRect(screen, 0, 0, float64(gameWidth), float64(gameHeight),
				color.RGBA{0, 0, 0, alphaByte})
		}
	} else {
		s.drawUI(screen)
	}

	if s.tutorialActive {
		s.drawBattleTutorial(screen)
	}
}

func (s *BattleScene) drawBackground(screen *ebiten.Image) {
	if bg := s.battleBgImage(); bg != nil {
		screen.DrawImage(bg, nil)
		return
	}
	screen.Fill(color.RGBA{20, 20, 25, 255})
}

func (s *BattleScene) battleBgImage() *ebiten.Image {
	if strings.HasPrefix(s.enemyType, "boss_") {
		numStr := strings.TrimPrefix(s.enemyType, "boss_")
		if bossNum, err := strconv.Atoi(numStr); err == nil {
			idx := bossNum - 1
			if idx >= 0 && idx < len(s.game.BossBgImgs) && s.game.BossBgImgs[idx] != nil {
				return s.game.BossBgImgs[idx]
			}
		}
	}
	return s.game.BattleBgImg
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
	}

	for _, p := range s.deathParticles {
		a := uint8(255 * p.Life)
		size := p.Size
		ebitenutil.DrawRect(screen, p.X-size/2+s.shakeX, p.Y-size/2+s.shakeY, size, size,
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

	pose := s.playerPose[i]
	row, rowOk := poseRow[pose]
	if !rowOk {
		row = 0
		pose = poseIdle
	}

	var frame int

	switch pose {
	case poseAttack:
		frame = frameFromProgress(s.attackPhaseTimer/0.45, poseAttack)
	case poseChargeApproach:
		frame = frameFromProgress(s.attackPhaseTimer/0.20, poseChargeApproach)
	case poseChargeAttack:
		frame = frameFromProgress((s.attackPhaseTimer-0.20)/0.45, poseChargeAttack)
	case poseFireCast:
		frame = frameFromProgress(s.attackPhaseTimer/0.30, poseFireCast)
	case poseReady:
		frame = 0
	case poseReadyGlow:
		frame = glowFrameForLevel(s.skillGlowLevel, s.playerAnimTimer[i])
	case poseWalk:
		frame = spriteFrame(pose, s.playerAnimTimer[i])
	case poseHealCast:
		if i != s.healingCaster || s.healingAnimTimer[i] <= 0 {
			pose = poseIdle
			row = poseRow[poseIdle]
			frame = spriteFrame(poseIdle, s.playerAnimTimer[i])
		} else {
			dur := 1.5
			elapsed := dur - s.healingAnimTimer[i]
			frame = frameFromProgress(elapsed/dur, poseHealCast)
		}
	default:
		frame = spriteFrame(pose, s.playerAnimTimer[i])
	}

	srcX := frame * spriteFrameW
	srcY := row * spriteFrameH
	srcRect = image.Rect(srcX, srcY, srcX+spriteFrameW, srcY+spriteFrameH)

	if srcRect.Max.X > spriteSheet.Bounds().Dx() ||
		srcRect.Max.Y > spriteSheet.Bounds().Dy() {
		srcRect = image.Rect(0, 0, spriteFrameW, spriteFrameH)
	}

	return spriteSheet, srcRect, true
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

		if s.activeAttacker == i && (pose == poseChargeApproach || pose == poseChargeAttack) {
			centerX -= s.chargeApproachOffset
		}

		s.partyScreenX[i] = centerX
		s.partyScreenY[i] = centerY

		if pose == poseWin {
		}

		op.GeoM.Translate(centerX+s.shakeX, centerY+s.shakeY)

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
		if pop.Timer > 1.0 {
			fade := 1.0 - (pop.Timer-1.0)/0.6
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
		mainColor := color.RGBA{255, 255, 255, alpha}
		switch {
		case pop.IsMiss:
			msg = "MISS"
			mainColor = color.RGBA{170, 170, 170, alpha}
		case pop.IsHeal:
			mainColor = color.RGBA{140, 255, 160, alpha}
		case pop.IsCrit:
			numFontSize = 62.0
			mainColor = color.RGBA{255, 130, 40, alpha}
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
		s.drawCommandMenu(screen)
		s.drawBattleShortcutButtons(screen)
		s.drawBottomDescription(screen,
			commandDescriptions[s.commandIndex],
			"")
	case phaseSkillMenu:
		s.drawCommandMenu(screen)
		s.drawSkillSubMenu(screen)
		s.drawBottomDescription(screen, s.currentSkillDescription(), s.skillLevelHint())
	case phaseTargetSelect:
		s.drawTargetSelectUI(screen)
		targetDesc, targetHint := s.targetSelectDescriptionAndHint()
		s.drawBottomDescription(screen, targetDesc, targetHint)
	case phaseHealSelect:
		s.drawHealTargetUI(screen)
		healDesc := ""
		if skillIdx := s.pendingSkill - 1; skillIdx >= 0 {
			healDesc = s.game.CurrentSkillLevelData(s.waitingActor, skillIdx).Description
		}
		s.drawBottomDescription(screen, healDesc, "")
	case phaseItemMenu:
		s.drawCommandMenu(screen)
		s.drawItemSubMenu(screen)
		s.drawBottomDescription(screen, s.currentItemDescription(), "")
	case phaseItemTarget:
		s.drawItemTargetUI(screen)
		itemDesc, itemHint := s.itemTargetDescriptionAndHint()
		s.drawBottomDescription(screen, itemDesc, itemHint)
	}

	s.drawMyTurnOverlay(screen)
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
