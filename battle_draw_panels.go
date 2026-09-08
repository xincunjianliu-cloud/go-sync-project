package main

// battle_draw_panels.go: 対象選択UI・リザルトパネル・各種描画ヘルパー
import (
	"fmt"
	"image/color"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func (s *BattleScene) drawTargetSelectUI(screen *ebiten.Image) {
	if s.enemyImage != nil {
		imgW := s.enemyImage.Bounds().Dx()
		imgH := s.enemyImage.Bounds().Dy()
		if imgW <= 32 && len(s.game.BossImgs) > 0 && s.game.BossImgs[0] != nil {
			imgW = s.game.BossImgs[0].Bounds().Dx()
			imgH = s.game.BossImgs[0].Bounds().Dy()
		}

		xPos := 280.0 - float64(imgW)/2
		yPos := 240.0 - float64(imgH)/2
		if yPos < 12 {
			yPos = 12
		}

		cursorX := xPos - 24.0
		cursorY := yPos + float64(imgH)/2 - 8.0

		op := &text.DrawOptions{}
		op.GeoM.Translate(cursorX+s.shakeX, cursorY+s.shakeY)
		op.ColorScale.ScaleWithColor(uiColorText)

		text.Draw(screen, "▶", s.game.FontFace(15), op)
	}
}

func (s *BattleScene) drawHealTargetUI(screen *ebiten.Image) {
	isAll := s.healTargetIndex == partySize

	for i := 0; i < partySize; i++ {
		centerX := s.partyScreenX[i]
		centerY := s.partyScreenY[i]

		showArrow := false
		if isAll {
			showArrow = s.game.PlayerHP[i] > 0
		} else {
			showArrow = s.healTargetIndex == i
		}

		if showArrow {
			arrowOp := &text.DrawOptions{}
			arrowOp.GeoM.Translate(centerX-16, centerY+20)
			arrowOp.ColorScale.ScaleWithColor(uiColorText)
			text.Draw(screen, "▶", s.game.FontFace(15), arrowOp)
		}
	}
}

// targetSelectDescriptionAndHint は対象選択画面(phaseTargetSelect)での
// 説明文とヒント文字列を返す。単体/全体を選べるスキルの場合のみ
// 「←単体 / 全体→」に相当するヒントを表示する（回復画面と同じ考え方）。
func (s *BattleScene) targetSelectDescriptionAndHint() (string, string) {
	p := s.waitingActor
	if p < 0 || p >= partySize || s.pendingSkill < 1 {
		return "", ""
	}
	skillIdx := s.pendingSkill - 1
	skills := s.game.CharacterSkills(p)
	if skillIdx < 0 || skillIdx >= len(skills) {
		return "", ""
	}
	lv := s.lastSkillLevel[p][skillIdx]
	if lv < 1 {
		lv = 1
	}
	if lv > len(skills[skillIdx].Levels) {
		lv = len(skills[skillIdx].Levels)
	}
	data := skills[skillIdx].Levels[lv-1]

	hint := ""
	if data.Target == TargetBoth {
		if s.selectedSkillTarget == TargetAll {
			hint = "←:単体に切替"
		} else {
			hint = "→:全体に切替"
		}
	}
	return data.Description, hint
}

func (s *BattleScene) drawResultPanel(screen *ebiten.Image) {

	panelAlpha := float32(1.0)
	if s.resultSubPhase == resSubResultFadeIn {
		panelAlpha = float32(s.resultAnimTimer / 0.5)
		if panelAlpha > 1.0 {
			panelAlpha = 1.0
		}
	}

	panelW := float64(gameWidth) * resultPanelWidthRatio
	panelH := float64(gameHeight)
	ebitenutil.DrawRect(screen, 0, 0, panelW, panelH,
		scaleAlpha(color.RGBA{0, 0, 0, 200}, float64(panelAlpha)))
	if s.resultFadeAlpha > 0 {
		alphaByte := uint8(255 * s.resultFadeAlpha)
		ebitenutil.DrawRect(screen, 0, 0, panelW, panelH, color.RGBA{0, 0, 0, alphaByte})
	}

	titleOp := &text.DrawOptions{}
	titleOp.GeoM.Translate(resultTitleX, resultTitleY)
	titleOp.ColorScale.ScaleWithColor(uiColorText)
	titleOp.ColorScale.ScaleAlpha(panelAlpha)
	text.Draw(screen, "Battle Results", s.game.FontFace(resultTitleFontSize), titleOp)

	// 獲得EXP（ラベル・数字別X）
	expLabelOp := &text.DrawOptions{}
	expLabelOp.GeoM.Translate(resultExpLabelX, resultExpY)
	expLabelOp.ColorScale.ScaleWithColor(uiColorText)
	expLabelOp.ColorScale.ScaleAlpha(panelAlpha)
	text.Draw(screen, "EXP", s.game.FontFace(resultExpFontSize), expLabelOp)

	expValueOp := &text.DrawOptions{}
	expValueOp.GeoM.Translate(resultExpValueX, resultExpY)
	expValueOp.ColorScale.ScaleWithColor(uiColorText)
	expValueOp.ColorScale.ScaleAlpha(panelAlpha)
	text.Draw(screen, fmt.Sprintf("%d", s.enemyExp), s.game.FontFace(resultExpFontSize), expValueOp)

	// 獲得SP（ラベル・数字別X）
	spLabelOp := &text.DrawOptions{}
	spLabelOp.GeoM.Translate(resultSpLabelX, resultSpY)
	spLabelOp.ColorScale.ScaleWithColor(uiColorText)
	spLabelOp.ColorScale.ScaleAlpha(panelAlpha)
	text.Draw(screen, "SP", s.game.FontFace(resultSpFontSize), spLabelOp)

	spValueOp := &text.DrawOptions{}
	spValueOp.GeoM.Translate(resultSpValueX, resultSpY)
	spValueOp.ColorScale.ScaleWithColor(uiColorText)
	spValueOp.ColorScale.ScaleAlpha(panelAlpha)
	text.Draw(screen, fmt.Sprintf("%d", s.enemySP), s.game.FontFace(resultSpFontSize), spValueOp)

	ebitenutil.DrawRect(screen, resultDividerX, resultDividerY, resultDividerW, resultDividerH, uiColorText)

	barW := resultBarWAbs

	for i := 0; i < partySize; i++ {
		nameColor := uiColorText
		if s.game.PlayerHP[i] <= 0 {
			nameColor = uiColorDead
		}

		barX := resultBarStartX
		barY := resultBarStartY + float64(i)*resultBarRowGap
		barH := resultBarH

		// プレイヤー名・レベル（別X）
		nameOp := &text.DrawOptions{}
		nameOp.GeoM.Translate(resultNameX, barY+resultNameOffsetY)
		nameOp.ColorScale.ScaleWithColor(nameColor)
		nameOp.ColorScale.ScaleAlpha(panelAlpha)
		text.Draw(screen, PlayerNames[i], s.game.FontFace(resultNameFontSize), nameOp)

		levelOp := &text.DrawOptions{}
		levelOp.GeoM.Translate(resultLevelX, barY+resultNameOffsetY)
		levelOp.ColorScale.ScaleWithColor(nameColor)
		levelOp.ColorScale.ScaleAlpha(panelAlpha)
		text.Draw(screen, fmt.Sprintf("Lv %d", s.game.PlayerLv[i]), s.game.FontFace(resultNameFontSize), levelOp)

		curExpStr := strconv.Itoa(s.drawPlayerEXP[i])
		maxExpStr := fmt.Sprintf("/%d", s.drawPlayerMaxEXP[i])
		expRightX := barX + barW
		expBaseY := barY + resultExpTextOffsetY

		maxOp := &text.DrawOptions{}
		maxOp.PrimaryAlign = text.AlignEnd
		maxOp.GeoM.Translate(expRightX, expBaseY+3)
		maxOp.ColorScale.ScaleWithColor(uiColorText)
		maxOp.ColorScale.ScaleAlpha(panelAlpha)
		text.Draw(screen, maxExpStr, s.game.FontFace(resultExpMaxFontSize), maxOp)

		maxExpW, _ := text.Measure(maxExpStr, s.game.FontFace(resultExpMaxFontSize), 0)

		curOp := &text.DrawOptions{}
		curOp.PrimaryAlign = text.AlignEnd
		curOp.GeoM.Translate(expRightX-maxExpW, expBaseY)
		curOp.ColorScale.ScaleWithColor(uiColorText)
		curOp.ColorScale.ScaleAlpha(panelAlpha)
		text.Draw(screen, curExpStr, s.game.FontFace(resultExpCurFontSize), curOp)

		expLabelOp2 := &text.DrawOptions{}
		expLabelOp2.GeoM.Translate(barX+resultExpLabelOffsetX, barY+resultExpLabelOffsetY)
		expLabelOp2.ColorScale.ScaleWithColor(uiColorText)
		expLabelOp2.ColorScale.ScaleAlpha(panelAlpha)
		text.Draw(screen, "EXP", s.game.FontFace(resultExpLabelFontSize), expLabelOp2)

		if s.resultSubPhase >= resSubBarAnimate {
			ebitenutil.DrawRect(screen, barX, barY, barW, barH, resultBarBgColor)
			if s.game.PlayerHP[i] > 0 && s.drawPlayerMaxEXP[i] > 0 {
				ratio := s.drawPlayerEXPF[i] / float64(s.drawPlayerMaxEXP[i])
				if ratio > 1.0 {
					ratio = 1.0
				}
				if ratio < 0.0 {
					ratio = 0.0
				}
				if int(barW*ratio) >= 1 {
					ebitenutil.DrawRect(screen, barX, barY, barW*ratio, barH, resultBarFillColor)
				}
			}
		}

		if s.isLevelUp[i] && s.game.PlayerHP[i] > 0 && s.drawPlayerLv[i] >= s.game.PlayerLv[i] {
			lvOp := &text.DrawOptions{}
			lvOp.GeoM.Translate(barX+resultLevelUpOffsetX, barY+resultLevelUpOffsetY)
			lvOp.ColorScale.ScaleWithColor(color.RGBA{255, 255, 100, 255})
			lvOp.ColorScale.ScaleAlpha(panelAlpha)
			text.Draw(screen, "LEVEL UP!", s.game.FontFace(resultLevelUpFontSize), lvOp)
		}
	}

	if s.resultSubPhase == resSubDoneWait {
		hintOp := &text.DrawOptions{}
		hintOp.GeoM.Translate(resultHintX, float64(gameHeight)-resultHintYFromBtm)
		hintOp.ColorScale.ScaleWithColor(uiColorText)
		text.Draw(screen, "Enter / Space / Z で進む", s.game.FontFace(resultHintFontSize), hintOp)
	}
}

func (s *BattleScene) drawControlHint(screen *ebiten.Image, hint string) {
	if hint == "" {
		return
	}
	hintOp := &text.DrawOptions{}
	hintOp.GeoM.Translate(float64(gameWidth)/2, hintY)
	hintOp.PrimaryAlign = text.AlignCenter
	hintOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, hint, s.game.FontFace(15), hintOp)
}

func (s *BattleScene) drawBottomDescription(screen *ebiten.Image, desc string, hint string) {
	descOp := &text.DrawOptions{}
	descOp.GeoM.Scale(descScale, descScale)
	descOp.GeoM.Translate(descX+s.shakeX, descY+s.shakeY)
	descOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, desc, s.game.FontFace(descFontSize), descOp)

	if hint == "" {
		return
	}

	hintOp := &text.DrawOptions{}
	hintOp.GeoM.Scale(hintScale, hintScale)
	hintOp.GeoM.Translate(descX+hintOffsetX+s.shakeX, descY+s.shakeY)
	hintOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, hint, s.game.FontFace(hintFontSize), hintOp)
}

func scaleAlpha(c color.RGBA, factor float64) color.RGBA {
	c.A = uint8(float64(c.A) * factor)
	return c
}

func drawStatusValue(screen *ebiten.Image, x, y float64, current, max int, faceLarge, faceSmall *text.GoTextFace, alpha float64) {
	curStr := strconv.Itoa(current)
	restStr := fmt.Sprintf(" / %d", max)

	for _, offset := range [][2]float64{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
		shadowOp := &text.DrawOptions{}
		shadowOp.GeoM.Translate(x+offset[0], y+offset[1])
		shadowOp.ColorScale.ScaleWithColor(color.RGBA{0, 0, 0, uint8(255 * alpha)}) // 影は専用色のため据え置き
		text.Draw(screen, curStr, faceLarge, shadowOp)

		restShadowOp := &text.DrawOptions{}
		restShadowOp.GeoM.Translate(x+text.Advance(curStr, faceLarge)+offset[0], y+2.5+offset[1])
		restShadowOp.ColorScale.ScaleWithColor(color.RGBA{0, 0, 0, uint8(255 * alpha)})
		text.Draw(screen, restStr, faceSmall, restShadowOp)
	}

	textColor := uiColorText
	textColor.A = uint8(255 * alpha)

	curOp := &text.DrawOptions{}
	curOp.GeoM.Translate(x, y)
	curOp.ColorScale.ScaleWithColor(textColor)
	text.Draw(screen, curStr, faceLarge, curOp)
	restX := x + text.Advance(curStr, faceLarge)
	restOp := &text.DrawOptions{}
	restOp.GeoM.Translate(restX, y+2.5)
	restOp.ColorScale.ScaleWithColor(textColor)
	text.Draw(screen, restStr, faceSmall, restOp)
}
func fillSlantedPath(screen *ebiten.Image, path *vector.Path, c color.RGBA) {
	drawOpts := &vector.DrawPathOptions{AntiAlias: false}
	drawOpts.ColorScale.ScaleWithColor(c)
	vector.FillPath(screen, path, nil, drawOpts)
}
func drawSlantedQuad(screen *ebiten.Image, x, y, w, h, slant float64, c color.RGBA) {
	if w <= 0 || h <= 0 {
		return
	}
	var path vector.Path
	path.MoveTo(float32(x+slant), float32(y))
	path.LineTo(float32(x+w), float32(y))
	path.LineTo(float32(x+w-slant), float32(y+h))
	path.LineTo(float32(x), float32(y+h))
	path.Close()
	fillSlantedPath(screen, &path, c)
}
func drawSlantedStatusBar(screen *ebiten.Image, x, y, w, h, slant, ratio float64, fill, empty, edge color.RGBA) {
	drawSlantedQuad(screen, x-1, y-1, w+2, h+2, slant, color.RGBA{0, 0, 0, empty.A})

	innerSpan := w - slant
	if innerSpan <= 0 {
		return
	}
	drawSlantedQuad(screen, x, y, w, h, slant, empty)
	if ratio > 0 {
		if ratio >= 1 {
			drawSlantedQuad(screen, x, y, w, h, slant, fill)
		} else {
			filledSpan := innerSpan * ratio
			var path vector.Path
			path.MoveTo(float32(x+slant), float32(y))
			path.LineTo(float32(x+slant+filledSpan), float32(y))
			path.LineTo(float32(x+filledSpan), float32(y+h))
			path.LineTo(float32(x), float32(y+h))
			path.Close()
			fillSlantedPath(screen, &path, fill)
		}
	}
}

const namePlateOffsetX = -10.0 // ← 調整用：左にずらす量（マイナスで左へ）

func (s *BattleScene) drawPartyName(screen *ebiten.Image, i int, xPos, winY, alpha float64, isMyTurn bool) {
	nameImg := s.game.NameImg
	if isMyTurn && s.game.NameMyTurnImg != nil {
		nameImg = s.game.NameMyTurnImg
	}

	sx := xPos + namePlateOffsetX + s.shakeX
	sy := winY + s.shakeY

	nameCol := uiColorText
	if s.game.PlayerHP[i] <= 0 {
		nameCol = uiColorDead
	}

	if nameImg == nil {
		nameOp := &text.DrawOptions{}
		nameOp.GeoM.Translate(sx, sy+4)
		nameOp.ColorScale.ScaleWithColor(nameCol)
		nameOp.ColorScale.ScaleAlpha(float32(alpha))
		text.Draw(screen, PlayerNames[i], s.game.FontFace(15), nameOp)
		return
	}

	// ★変更：statusBlockW に合わせた拡縮をやめ、原寸のまま表示する
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(sx, sy)
	op.ColorScale.ScaleAlpha(float32(alpha))
	if s.game.PlayerHP[i] <= 0 {
		op.ColorScale.Scale(0.5, 0.5, 0.5, 1.0)
	}
	screen.DrawImage(nameImg, op)

	// プレート画像の上に名前テキストを重ねて表示
	nameTextOp := &text.DrawOptions{}
	nameTextOp.GeoM.Translate(sx+6, sy+4) // ← 位置微調整はここ
	nameTextOp.ColorScale.ScaleWithColor(nameCol)
	nameTextOp.ColorScale.ScaleAlpha(float32(alpha))
	text.Draw(screen, PlayerNames[i], s.game.FontFace(13), nameTextOp)
}
