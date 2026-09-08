package main

// battle_draw.go: バトル画面の基本描画（背景・敵・パーティスプライト・UI振り分け）

import (
	"fmt"
	"image"
	"image/color"
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
	if s.enemyDeathPhase == 3 {
		return
	}
	if s.enemyDeathPhase == 2 {
		for _, p := range s.deathParticles {
			a := uint8(255 * p.Life)
			size := p.Size
			ebitenutil.DrawRect(screen, p.X-size/2+s.shakeX, p.Y-size/2+s.shakeY, size, size,
				color.RGBA{255, 255, 255, a})
		}
		return
	}
	if s.enemyImage == nil {
		return
	}

	imgW := s.enemyImage.Bounds().Dx()
	imgH := s.enemyImage.Bounds().Dy()
	if imgW <= 32 && len(s.game.BossImgs) > 0 && s.game.BossImgs[0] != nil {
		s.enemyImage = s.game.BossImgs[0]
		imgW = s.enemyImage.Bounds().Dx()
		imgH = s.enemyImage.Bounds().Dy()
	}

	xPos := 280.0 - float64(imgW)/2
	yPos := 240.0 - float64(imgH)/2
	if yPos < 12 {
		yPos = 12
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(xPos+s.shakeX, yPos+s.shakeY)
	if s.enemyDeathPhase == 1 {
		op.ColorScale.ScaleAlpha(float32(s.enemyAlpha))
	}
	screen.DrawImage(s.enemyImage, op)
}

// drawPartySprites：新レイアウト（96x144グリッド、poseAnimテーブル方式）で全キャラ共通描画
func (s *BattleScene) drawPartySprites(screen *ebiten.Image) {
	for i := 0; i < partySize; i++ {
		spriteSheet := s.game.PlayerAttackSprites[i]
		if spriteSheet == nil {
			continue
		}

		pose := s.playerPose[i]
		row, ok := poseRow[pose]
		if !ok {
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
			frame = 0 // 4行目・左端（スキル以外）
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
		srcRect := image.Rect(srcX, srcY, srcX+spriteFrameW, srcY+spriteFrameH)

		if srcRect.Max.X > spriteSheet.Bounds().Dx() ||
			srcRect.Max.Y > spriteSheet.Bounds().Dy() {
			srcRect = image.Rect(0, 0, spriteFrameW, spriteFrameH)
		}

		frameImg := spriteSheet.SubImage(srcRect).(*ebiten.Image)

		op := &ebiten.DrawImageOptions{}
		baseX := 640.0
		baseY := 170.0
		centerX := baseX + float64(i)*10.0
		centerY := baseY + float64(i)*50.0

		centerX += s.readySlideX[i]
		centerX += s.introCharOffsetX

		// 強撃だけ、選択時の前進位置からさらに接近する
		if s.activeAttacker == i && (pose == poseChargeApproach || pose == poseChargeAttack) {
			centerX -= s.chargeApproachOffset
		}

		s.partyScreenX[i] = centerX
		s.partyScreenY[i] = centerY

		if pose == poseWin {
			// 予備：勝利ポーズ演出用の微調整余地（現状フレームは通常と同じ）
		}

		op.GeoM.Translate(centerX+s.shakeX, centerY+s.shakeY)

		switch pose {
		case poseDead:
			op.ColorScale.Scale(0.5, 0.5, 0.5, 0.8)
		case poseLowHP:
			op.ColorScale.Scale(1.0, 0.75, 0.75, 1.0)
		case poseDamage:
			t := s.playerAnimTimer[i]
			flashStrength := float32(1.0 - t/0.4)
			if flashStrength < 0 {
				flashStrength = 0
			}
			op.ColorScale.Scale(
				1.0+flashStrength*0.8,
				1.0+flashStrength*0.8,
				1.0+flashStrength*0.8,
				1.0,
			)
		}

		screen.DrawImage(frameImg, op)
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
		msg := fmt.Sprintf("%d", pop.Value)

		for _, offset := range [][2]float64{{-1.2, 0}, {1.2, 0}, {0, -1.2}, {0, 1.2}} {
			shadowOp := &text.DrawOptions{}
			shadowOp.GeoM.Scale(2.2, 2.2)
			shadowOp.GeoM.Translate(pop.X+offset[0]+s.shakeX, pop.Y+offset[1]+s.shakeY)
			shadowOp.ColorScale.ScaleWithColor(color.RGBA{0, 0, 0, alpha})
			text.Draw(screen, msg, s.game.FontFace(16), shadowOp)
		}
		op := &text.DrawOptions{}
		op.GeoM.Scale(2.2, 2.2)
		op.GeoM.Translate(pop.X+s.shakeX, pop.Y+s.shakeY)
		if pop.IsHeal {
			op.ColorScale.ScaleWithColor(color.RGBA{140, 255, 160, alpha})
		} else {
			op.ColorScale.ScaleWithColor(color.RGBA{255, 220, 80, alpha})
		}
		text.Draw(screen, msg, s.game.FontFace(16), op)
	}

	switch s.battlePhase {
	case phasePlayerMenu:
		s.drawCommandMenu(screen)
		s.drawBottomDescription(screen,
			commandDescriptions[s.commandIndex],
			"")
	case phaseSkillMenu:
		s.drawCommandMenu(screen)
		s.drawSkillSubMenu(screen)
		s.drawBottomDescription(screen, s.currentSkillDescription(), "")
	case phaseTargetSelect:
		s.drawTargetSelectUI(screen)
		targetDesc, targetHint := s.targetSelectDescriptionAndHint()
		s.drawBottomDescription(screen, targetDesc, targetHint)
	case phaseHealSelect:
		s.drawHealTargetUI(screen)
		healHint := "→:全体回復に切替"
		if s.healTargetIndex == partySize {
			healHint = "←:個人選択に戻す"
		}
		healDesc := ""
		if skillIdx := s.pendingSkill - 1; skillIdx >= 0 {
			healDesc = s.game.CurrentSkillLevelData(s.waitingActor, skillIdx).Description
		}
		s.drawBottomDescription(screen, healDesc, healHint)
	}
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

func (s *BattleScene) currentSkillHint() string {
	p := s.waitingActor
	if p >= 0 && p < partySize {
		data := s.game.CurrentSkillLevelData(p, s.skillIndex)
		if data.Target == TargetBoth {
			return "←単体 / 全体→"
		}
	}
	return ""
}
