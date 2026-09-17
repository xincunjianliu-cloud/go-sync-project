package main

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	slotCardStartX = 300.0
	slotCardStartY = 55.0
	slotCardGapY   = 0.0

	slotNumOffsetX = 40.0
	slotNumYRatio  = 0.5
	slotNumSize    = 26.0

	slotThumbOffsetX = 334.0
	slotThumbYRatio  = 0.51
	slotThumbW       = 108.0
	slotThumbAspect  = 16.0 / 9.0
	slotThumbH       = slotThumbW / slotThumbAspect

	slotTextOffsetX  = 90.0
	slotTextStartY   = 25.0
	slotTextLineGap  = 16.0
	slotTextFontSize = 13.0

	slotsPerPageView = 4

	scrollBarX    = 800.0
	scrollBarTopY = 40.0
)

const (
	statusFontSize = 20.0

	statusFaceBoxX     = 270.0
	statusFaceBoxY     = 8.0
	statusFaceBoxW     = 195.0
	statusFaceBoxH     = 352.0
	statusFaceR        = 78.0
	statusFaceCYRat    = 0.31
	statusNameYRat     = 0.60
	statusNameFontSize = 18.0

	statusLeftLabelX = 220.0
	statusLeftValueX = 500.0
	statusLeftLineX  = 220.0
	statusLeftLineW  = 300.0
	statusLevelX     = statusLeftLabelX

	statusLvY  = 280.0
	statusExpY = 330.0
	statusHpY  = 390.0
	statusMpY  = 440.0

	statusRightLabelX = 570.0
	statusRightValueX = 900.0
	statusRightLineX  = 570.0
	statusRightLineW  = 350.0

	statusRightBoxY = 50.0
	statusRightRowH = 55.0

	statusPartyIconY         = 440.0
	statusPartyIconGap       = 90.0
	statusPartyIconStartX    = 610.0
	statusPartyIconFallbackR = 30.0

	statusLeftLineOffsetY  = statusFontSize*latinFontScale + 4.0
	statusRightLineOffsetY = statusFontSize*latinFontScale + 4.0
)

var statusLineColor = uiColorText

type statusStatRow struct {
	label string
	y     float64
}

var statusStatRows = []statusStatRow{
	{"物理攻撃力", statusRightBoxY + 0*statusRightRowH},
	{"魔法攻撃力", statusRightBoxY + 1*statusRightRowH},
	{"物理防御力", statusRightBoxY + 2*statusRightRowH},
	{"魔法防御力", statusRightBoxY + 3*statusRightRowH},
	{"すばやさ", statusRightBoxY + 4*statusRightRowH},
	{"運", statusRightBoxY + 5*statusRightRowH},
}

// drawStatusRow draws a label/value pair for the status screen. label may be
// Japanese ("物理攻撃力") or Latin ("Lv", "EXP", ...); value is always a
// number. Both are sized via DrawMixedText, which picks FontFace or
// LatinFontFace per string automatically, so callers only supply one
// nominal size instead of maintaining a matching Latin-scaled constant.
func drawStatusRow(screen *ebiten.Image, g *Game, label, value string, labelX, valueX, y, lineX, lineW, fontSize, lineOffsetY float64) {
	g.DrawMixedText(screen, label, fontSize, labelX, y, text.AlignStart, text.AlignStart, uiColorText)
	g.DrawMixedText(screen, value, fontSize, valueX, y, text.AlignEnd, text.AlignStart, uiColorText)

	ebitenutil.DrawRect(screen, lineX, y+lineOffsetY, lineW, 1, statusLineColor)
}

func (m *MenuScene) drawStatusScreen(screen *ebiten.Image) {
	g := m.game
	i := m.statusCharIndex

	faceCX := statusFaceBoxX + statusFaceBoxW/2
	faceCY := statusFaceBoxY + statusFaceBoxH*statusFaceCYRat

	img := g.PartyIconImgs[i]
	iw := float64(img.Bounds().Dx())
	ih := float64(img.Bounds().Dy())
	const faceIconScale = 2.0
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(faceIconScale, faceIconScale)
	op.GeoM.Translate(faceCX-iw*faceIconScale/2, faceCY-ih*faceIconScale/2)
	screen.DrawImage(img, op)

	nameOp := &text.DrawOptions{}
	nameOp.GeoM.Translate(faceCX, statusFaceBoxY+statusFaceBoxH*statusNameYRat)
	nameOp.PrimaryAlign = text.AlignCenter
	nameOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, PlayerNames[i], g.FontFace(statusNameFontSize), nameOp)

	drawStatusRow(screen, g, "Lv", fmt.Sprintf("%d/%d", g.PlayerLv[i], maxPlayerLevel),
		statusLevelX, statusLeftValueX, statusLvY, statusLeftLineX, statusLeftLineW,
		statusFontSize, statusLeftLineOffsetY)

	drawStatusRow(screen, g, "EXP", fmt.Sprintf("%d/%d", g.PlayerEXP[i], g.PlayerNextEXP[i]),
		statusLeftLabelX, statusLeftValueX, statusExpY, statusLeftLineX, statusLeftLineW,
		statusFontSize, statusLeftLineOffsetY)

	drawStatusRow(screen, g, "HP", fmt.Sprintf("%d", g.PlayerMaxHP[i]),
		statusLeftLabelX, statusLeftValueX, statusHpY, statusLeftLineX, statusLeftLineW,
		statusFontSize, statusLeftLineOffsetY)

	drawStatusRow(screen, g, "MP", fmt.Sprintf("%d", g.PlayerMaxMP[i]),
		statusLeftLabelX, statusLeftValueX, statusMpY, statusLeftLineX, statusLeftLineW,
		statusFontSize, statusLeftLineOffsetY)

	statusValues := []int{
		g.PlayerAtk[i],
		g.PlayerMagicAtk[i],
		g.PlayerDef[i],
		g.PlayerMagicDef[i],
		g.PlayerSpd[i],
		g.PlayerLuck[i],
	}
	for idx, row := range statusStatRows {
		drawStatusRow(screen, g, row.label, fmt.Sprintf("%d", statusValues[idx]),
			statusRightLabelX, statusRightValueX, row.y, statusRightLineX, statusRightLineW,
			statusFontSize, statusRightLineOffsetY)
	}

	for idx := 0; idx < partySize; idx++ {
		cx := statusPartyIconStartX + float64(idx)*statusPartyIconGap
		cy := statusPartyIconY

		isSelected := idx == m.statusCharIndex

		img := g.PartyIconImgs[idx]
		iw := float64(img.Bounds().Dx())
		ih := float64(img.Bounds().Dy())
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(cx-iw/2, cy-ih/2)
		if !isSelected {
			op.ColorScale.Scale(0.45, 0.45, 0.45, 1.0)
		}
		screen.DrawImage(img, op)
	}

	leftArrowX, rightArrowX := statusPartyArrowX(g)

	leftOp := &text.DrawOptions{}
	leftOp.GeoM.Translate(leftArrowX, statusPartyIconY)
	leftOp.SecondaryAlign = text.AlignCenter
	leftOp.PrimaryAlign = text.AlignEnd
	leftOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, "◀", g.FontFace(18), leftOp)

	rightOp := &text.DrawOptions{}
	rightOp.GeoM.Translate(rightArrowX, statusPartyIconY)
	rightOp.SecondaryAlign = text.AlignCenter
	rightOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, "▶", g.FontFace(18), rightOp)
}

func statusPartyArrowX(g *Game) (leftX, rightX float64) {
	const arrowGap = 5.0

	firstHalfW := float64(g.PartyIconImgs[0].Bounds().Dx()) / 2
	leftX = statusPartyIconStartX - firstHalfW - arrowGap

	lastIdx := partySize - 1
	lastCX := statusPartyIconStartX + float64(lastIdx)*statusPartyIconGap
	lastHalfW := float64(g.PartyIconImgs[lastIdx].Bounds().Dx()) / 2
	rightX = lastCX + lastHalfW + arrowGap
	return
}

func (m *MenuScene) lookupCharaImage(i int) *ebiten.Image {
	return nil
}

const maxSlotScrollTop = maxSaveSlots - slotsPerPageView

func clampSlotScrollTop(scrollTop, selectedIndex int) int {
	if selectedIndex < scrollTop {
		scrollTop = selectedIndex
	} else if selectedIndex > scrollTop+slotsPerPageView-1 {
		scrollTop = selectedIndex - slotsPerPageView + 1
	}
	if scrollTop < 0 {
		scrollTop = 0
	} else if scrollTop > maxSlotScrollTop {
		scrollTop = maxSlotScrollTop
	}
	return scrollTop
}

func drawSlotList(screen *ebiten.Image, g *Game, selectedIndex int, slotData [maxSaveSlots]*SaveData, slotThumbs [maxSaveSlots]*ebiten.Image, saveMode bool, scrollTop int, cardStartX, cardStartY, dragOffsetY float64) {

	cardH := float64(g.SaveThumbFrameImg.Bounds().Dy())
	cardW := float64(g.SaveThumbFrameImg.Bounds().Dx())

	listRect := image.Rect(int(cardStartX), int(cardStartY), int(cardStartX+cardW), int(cardStartY+cardH*slotsPerPageView))
	dst := screen.SubImage(listRect).(*ebiten.Image)

	for dispIdx := -1; dispIdx <= slotsPerPageView; dispIdx++ {
		i := scrollTop + dispIdx
		if i < 0 || i >= maxSaveSlots {
			continue
		}

		isSelected := i == selectedIndex

		cardX := cardStartX
		cardY := cardStartY + dragOffsetY + float64(dispIdx)*(cardH+slotCardGapY)

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(cardX, cardY)
		dst.DrawImage(g.SaveThumbFrameImg, op)
		if isSelected {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(cardX, cardY)
			dst.DrawImage(g.SaveThumbFrameSelImg, op)
		}

		numOp := &text.DrawOptions{}
		numOp.GeoM.Translate(cardX+slotNumOffsetX, cardY+cardH*slotNumYRatio)
		numOp.SecondaryAlign = text.AlignCenter
		numOp.ColorScale.ScaleWithColor(uiColorText)
		text.Draw(dst, fmt.Sprintf("%02d", i+1), g.LatinFontFace(slotNumSize), numOp)

		thumbX := cardX + slotThumbOffsetX
		thumbY := cardY + cardH*slotThumbYRatio - slotThumbH/2
		if slotThumbs[i] != nil {
			drawImageFitAspect(dst, slotThumbs[i], thumbX, thumbY, slotThumbW, slotThumbH)
		}

		d := slotData[i]
		tx := cardX + slotTextOffsetX
		ty := cardY + slotTextStartY

		if d != nil {
			locOp := &text.DrawOptions{}
			locOp.GeoM.Translate(tx, ty)
			locOp.ColorScale.ScaleWithColor(uiColorText)
			text.Draw(dst, d.LocationName, g.FontFace(slotTextFontSize), locOp)

			lvOp := &text.DrawOptions{}
			lvOp.GeoM.Translate(tx, ty+slotTextLineGap)
			lvOp.ColorScale.ScaleWithColor(uiColorText)
			text.Draw(dst, fmt.Sprintf("Lv %d", d.PlayerLv[0]), g.LatinFontFace(slotTextFontSize), lvOp)

			timeOp := &text.DrawOptions{}
			timeOp.GeoM.Translate(tx, ty+slotTextLineGap*2)
			timeOp.ColorScale.ScaleWithColor(uiColorText)
			text.Draw(dst, FormatPlayTime(d.PlayTime), g.LatinFontFace(slotTextFontSize), timeOp)

			dateOp := &text.DrawOptions{}
			dateOp.GeoM.Translate(tx, ty+slotTextLineGap*3)
			dateOp.ColorScale.ScaleWithColor(uiColorText)
			text.Draw(dst, d.SavedAt, g.LatinFontFace(slotTextFontSize), dateOp)
		}
	}

	barX := scrollBarX + (cardStartX - slotCardStartX)
	barTop := scrollBarTopY + (cardStartY - slotCardStartY)
	drawSaveScrollBar(screen, g, float64(scrollTop)-dragOffsetY/cardH, barX, barTop)
}

func hitTestSlotList(g *Game, tapX, tapY float64, scrollTop int, cardStartX, cardStartY float64) (int, bool) {
	cardW := float64(g.SaveThumbFrameImg.Bounds().Dx())
	cardH := float64(g.SaveThumbFrameImg.Bounds().Dy())

	var rects []tapRect
	var absIdx []int
	for dispIdx := 0; dispIdx < slotsPerPageView; dispIdx++ {
		i := scrollTop + dispIdx
		if i >= maxSaveSlots {
			break
		}
		cardY := cardStartY + float64(dispIdx)*(cardH+slotCardGapY)
		rects = append(rects, tapRect{x: cardStartX, y: cardY, w: cardW, h: cardH})
		absIdx = append(absIdx, i)
	}
	p := touchPoint{tapX, tapY}
	for i, r := range rects {
		if r.contains(p) {
			return absIdx[i], true
		}
	}
	return -1, false
}

func drawConfirmDialog(screen *ebiten.Image, game *Game, message string, selectedIndex int, horizontalOffset float64, showChoices ...bool) {
	displayChoices := true
	if len(showChoices) > 0 {
		displayChoices = showChoices[0]
	}

	winW := confirmPanelW
	winH := confirmPanelH

	winX := float64(gameWidth)/2 - winW/2 + horizontalOffset
	winY := float64(gameHeight)/2 - winH/2 + confirmImageOffsetY

	ebitenutil.DrawRect(screen, winX, winY, winW, winH, uiColorText)
	ebitenutil.DrawRect(screen, winX+confirmPanelBorderWidth, winY+confirmPanelBorderWidth, winW-confirmPanelBorderWidth*2, winH-confirmPanelBorderWidth*2, uiColorConfirmBg)

	face := game.FontFace(confirmFontSize)
	lineOp := &text.DrawOptions{}
	lineOp.PrimaryAlign = text.AlignCenter
	lineOp.LineSpacing = face.Metrics().HAscent + face.Metrics().HDescent + 4
	lineOp.ColorScale.ScaleWithColor(uiColorText)
	if displayChoices {
		lineOp.GeoM.Translate(winX+winW/2+confirmTextOffsetX, winY+confirmTextOffsetY)
	} else {
		lineOp.GeoM.Translate(winX+winW/2+confirmTextOffsetX, winY+winH/2)
		lineOp.SecondaryAlign = text.AlignCenter
	}
	text.Draw(screen, message, face, lineOp)

	if displayChoices {
		choices := []string{"はい", "いいえ"}
		choiceFace := game.FontFace(confirmFontSize)
		for j, choice := range choices {
			centerX := winX + winW/2 + confirmChoiceOffsetX
			centerY := winY + confirmChoiceStartY + float64(j)*confirmChoiceGap
			col := uiColorText
			if j == selectedIndex {
				col = uiColorSelect
				labelW := text.Advance(choice, choiceFace)
				arrowOp := &text.DrawOptions{}
				arrowOp.GeoM.Translate(centerX-labelW/2-text.Advance("▶ ", choiceFace), centerY)
				arrowOp.ColorScale.ScaleWithColor(col)
				text.Draw(screen, "▶", choiceFace, arrowOp)
			}
			choiceOp := &text.DrawOptions{}
			choiceOp.GeoM.Translate(centerX, centerY)
			choiceOp.PrimaryAlign = text.AlignCenter
			choiceOp.ColorScale.ScaleWithColor(col)
			text.Draw(screen, choice, choiceFace, choiceOp)
		}
	}
}

func hitTestConfirmDialog(game *Game, horizontalOffset float64) (int, bool) {
	winW := confirmPanelW
	winH := confirmPanelH
	winX := float64(gameWidth)/2 - winW/2 + horizontalOffset
	winY := float64(gameHeight)/2 - winH/2 + confirmImageOffsetY

	choices := []string{"はい", "いいえ"}
	rects := make([]tapRect, len(choices))
	const choiceHitW = 240.0
	for j := range choices {
		centerX := winX + winW/2 + confirmChoiceOffsetX
		centerY := winY + confirmChoiceStartY + float64(j)*confirmChoiceGap
		rects[j] = tapRect{x: centerX - choiceHitW/2, y: centerY - 6, w: choiceHitW, h: confirmChoiceGap}
	}
	return hitTestTapRects(rects)
}

const (
	scrollBarTrackWidth  = 3.0
	scrollBarTrackHeight = 456.0
	scrollBarCursorWidth = 3.0
	scrollBarCursorH     = 129.0
)

var (
	scrollBarTrackColor  = color.NRGBA{255, 255, 255, 120}
	scrollBarCursorColor = color.NRGBA{255, 0, 12, 235}
)

const scrollBarTouchPad = 16.0

func slotScrollBarX(cardStartX float64) float64 {
	return scrollBarX + (cardStartX - slotCardStartX)
}

func slotScrollBarTop(cardStartY float64) float64 {
	return scrollBarTopY + (cardStartY - slotCardStartY)
}

func slotScrollBarRect(cardStartX, cardStartY float64) (x, y, w, h float64) {
	x = slotScrollBarX(cardStartX) - scrollBarTouchPad
	y = slotScrollBarTop(cardStartY)
	w = scrollBarCursorWidth + scrollBarTouchPad*2
	h = scrollBarTrackHeight
	return
}

func slotScrollBarMoveRange() float64 {
	moveRange := scrollBarTrackHeight - scrollBarCursorH
	if moveRange < 0 {
		return 0
	}
	return moveRange
}

func drawSaveScrollBar(screen *ebiten.Image, g *Game, scrollTop float64, scrollBarX, barTop float64) {
	ebitenutil.DrawRect(screen, scrollBarX, barTop, scrollBarTrackWidth, scrollBarTrackHeight, scrollBarTrackColor)

	moveRange := scrollBarTrackHeight - scrollBarCursorH
	if moveRange < 0 {
		moveRange = 0
	}

	ratio := 0.0
	if maxSlotScrollTop > 0 {
		ratio = scrollTop / float64(maxSlotScrollTop)
		if ratio < 0 {
			ratio = 0
		} else if ratio > 1 {
			ratio = 1
		}
	}
	cursorY := barTop + ratio*moveRange

	ebitenutil.DrawRect(screen, scrollBarX, cursorY, scrollBarCursorWidth, scrollBarCursorH, scrollBarCursorColor)
}

func drawImageFitAspect(screen *ebiten.Image, img *ebiten.Image, x, y, w, h float64) {
	if img == nil {
		return
	}
	iw := float64(img.Bounds().Dx())
	ih := float64(img.Bounds().Dy())
	if iw == 0 || ih == 0 {
		return
	}
	scale := w / iw
	if sh := h / ih; sh < scale {
		scale = sh
	}
	drawW := iw * scale
	drawH := ih * scale
	offsetX := x + (w-drawW)/2
	offsetY := y + (h-drawH)/2

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(offsetX, offsetY)
	screen.DrawImage(img, op)
}
