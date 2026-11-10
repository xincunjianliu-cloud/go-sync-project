package main

import (
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	mapBubbleMaxTextWidth = 360.0
	mapBubbleWidth        = 404.0
	mapBubbleHeight       = 200.0
	mapBubblePaddingX     = 22.0
	mapBubblePaddingY     = 12.0
	mapBubbleGap          = 10.0
	mapBubbleTailWidth    = 36.0
	mapBubbleTailLength   = 48.0
	mapBubbleCornerRadius = 18.0
)

func (s *FieldScene) drawMapSpeechBubble(screen *ebiten.Image, cmd EventCommand, fontSize float64) {
	face := s.game.FontFace(fontSize)
	lines := wrapMapBubbleText(cmd.Text, face, mapBubbleMaxTextWidth)
	lineHeight := face.Metrics().HAscent + face.Metrics().HDescent + 7
	for len(lines) > 0 && float64(len(lines))*lineHeight > mapBubbleHeight-mapBubblePaddingY*2 && fontSize > 1 {
		fontSize--
		face = s.game.FontFace(fontSize)
		lines = wrapMapBubbleText(cmd.Text, face, mapBubbleMaxTextWidth)
		lineHeight = face.Metrics().HAscent + face.Metrics().HDescent + 7
	}

	bubbleW, bubbleH := mapBubbleWidth, mapBubbleHeight

	camX, camY := s.cameraPosition()
	camX += s.screenShakeX
	camY += s.screenShakeY
	playerX, playerY := s.playerCfg.DrawTopLeftAt(s.px, s.py)
	playerW := float64(s.playerCfg.FrameWidth) * s.playerCfg.Scale
	playerH := float64(s.playerCfg.FrameHeight) * s.playerCfg.Scale
	charRight := playerX + camX + playerW
	charTop := playerY + camY
	charCenterX := playerX + camX + playerW/2
	charCenterY := charTop + playerH*0.38

	bubbleOnRight := charRight+mapBubbleGap+bubbleW <= float64(gameWidth)-12
	bubbleX := charRight + mapBubbleGap
	if !bubbleOnRight {
		bubbleX = playerX + camX - bubbleW - mapBubbleGap
	}
	bubbleX = max(8, min(bubbleX, float64(gameWidth)-bubbleW-8))
	bubbleY := charCenterY - bubbleH - mapBubbleGap*2
	tailAtTop := false
	if bubbleY < 4 {
		bubbleY = charTop + playerH + mapBubbleGap
		tailAtTop = true
	}
	bubbleY = max(4, min(bubbleY, float64(gameHeight)-bubbleH-4))

	tailBaseX := max(bubbleX+mapBubbleCornerRadius+mapBubbleTailWidth/2,
		min(charCenterX, bubbleX+bubbleW-mapBubbleCornerRadius-mapBubbleTailWidth/2))
	tailTipY := charTop + playerH*0.5
	bubble := mapSpeechBubblePath(bubbleX, bubbleY, bubbleW, bubbleH, tailBaseX, charCenterX, tailTipY, tailAtTop)
	fillMapBubblePath(screen, &bubble, color.White)
	drawOpts := &vector.DrawPathOptions{AntiAlias: true}
	drawOpts.ColorScale.ScaleWithColor(color.Black)
	vector.StrokePath(screen, &bubble, &vector.StrokeOptions{Width: 3, LineJoin: vector.LineJoinRound}, drawOpts)

	runes := []rune(cmd.Text)
	count := min((s.msg.ticks-s.msg.msgStart)/s.msg.speed(), len(runes))
	visibleLines := wrapMapBubbleText(string(runes[:count]), face, mapBubbleWidth-mapBubblePaddingX*2)
	textStartY := bubbleY + mapBubblePaddingY + face.Metrics().HAscent
	for i, line := range visibleLines {
		op := &text.DrawOptions{}
		op.GeoM.Translate(bubbleX+mapBubblePaddingX, textStartY+float64(i)*lineHeight)
		op.ColorScale.ScaleWithColor(color.RGBA{35, 35, 35, 255})
		text.Draw(screen, line, face, op)
	}
}

func mapSpeechBubblePath(x, y, w, h, tailBaseX, tipX, tipY float64, tailAtTop bool) vector.Path {
	radius := min(mapBubbleCornerRadius, min(w/4, h/4))
	tailHalfWidth := mapBubbleTailWidth / 2
	tailBaseX = max(x+radius+tailHalfWidth, min(tailBaseX, x+w-radius-tailHalfWidth))
	tailRightX := tailBaseX + tailHalfWidth
	tailLeftX := tailBaseX - tailHalfWidth
	baseY := y + h
	if tailAtTop {
		baseY = y
	}
	dx, dy := tipX-tailBaseX, tipY-baseY
	tailLength := math.Hypot(dx, dy)
	if tailLength > mapBubbleTailLength {
		scale := mapBubbleTailLength / tailLength
		tipX = tailBaseX + dx*scale
		tipY = baseY + dy*scale
	}

	var path vector.Path
	path.MoveTo(float32(x+radius), float32(y))
	if tailAtTop {
		path.LineTo(float32(tailLeftX), float32(y))
		path.LineTo(float32(tipX), float32(tipY))
		path.LineTo(float32(tailRightX), float32(y))
	}
	path.LineTo(float32(x+w-radius), float32(y))
	appendBubbleArc(&path, x+w-radius, y+radius, radius, -math.Pi/2, 0)
	path.LineTo(float32(x+w), float32(y+h-radius))
	appendBubbleArc(&path, x+w-radius, y+h-radius, radius, 0, math.Pi/2)
	if !tailAtTop {
		path.LineTo(float32(tailRightX), float32(baseY))
		path.LineTo(float32(tipX), float32(tipY))
		path.LineTo(float32(tailLeftX), float32(baseY))
	}
	path.LineTo(float32(x+radius), float32(y+h))
	appendBubbleArc(&path, x+radius, y+h-radius, radius, math.Pi/2, math.Pi)
	path.LineTo(float32(x), float32(y+radius))
	appendBubbleArc(&path, x+radius, y+radius, radius, math.Pi, math.Pi*1.5)
	path.Close()
	return path
}

func appendBubbleArc(path *vector.Path, cx, cy, radius, start, end float64) {
	const segments = 8
	for i := 1; i <= segments; i++ {
		angle := start + (end-start)*float64(i)/segments
		path.LineTo(float32(cx+radius*math.Cos(angle)), float32(cy+radius*math.Sin(angle)))
	}
}

func fillMapBubblePath(screen *ebiten.Image, path *vector.Path, fill color.Color) {
	var scale ebiten.ColorScale
	scale.ScaleWithColor(fill)
	vector.FillPath(screen, path, &vector.FillOptions{}, &vector.DrawPathOptions{AntiAlias: true, ColorScale: scale})
}

func wrapMapBubbleText(value string, face text.Face, maxWidth float64) []string {
	var lines []string
	for _, paragraph := range strings.Split(value, "\n") {
		lines = append(lines, wrapByWidth(paragraph, face, maxWidth)...)
	}
	return lines
}
