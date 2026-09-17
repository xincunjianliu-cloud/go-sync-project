package main

import (
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	logColumnWidth  = 640.0
	logBottomMargin = 12.0

	logEntryGap  = 35.0
	logTextLineH = 20.0

	logFallbackImageHeight = 110.0

	logImageStartX = 200.0
	logImageStartY = 30.0

	logNameOffsetX = 190.0
	logNameOffsetY = 4.0
	logTextOffsetX = 180.0
	logTextOffsetY = 30.0

	logVisibleCount  = 4
	logScrollStep    = 1.0
	logScrollBarGap  = 16.0
	logScrollBarMinH = 24.0

	logSelectPulseSpeed = 3.0
)

var logTextColor = color.RGBA{0, 0, 0, 255}

func lerpColor(a, b color.NRGBA, t float64) color.NRGBA {
	return color.NRGBA{
		R: uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		G: uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		B: uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
		A: 255,
	}
}

func selectedLogPulseColor(elapsed float64) color.NRGBA {
	lightRed := color.NRGBA{255, 140, 140, 255}
	darkRed := color.NRGBA{190, 80, 80, 255}
	t := (math.Sin(elapsed*logSelectPulseSpeed) + 1) / 2
	return lerpColor(lightRed, darkRed, t)
}

func wrapLines(s string) []string {
	return strings.Split(s, "\n")
}

type logEntryLayout struct {
	cmd   EventCommand
	lines []string
}

func drawMessageLog(screen *ebiten.Image, game *Game, log []EventCommand, scrollOffset float64, cursorIndex int) {
	ebitenutil.DrawRect(screen, 0, 0, float64(gameWidth), float64(gameHeight), color.NRGBA{0, 0, 0, 170})

	titleFace := game.LatinFontFace(30)
	titleOp := &text.DrawOptions{}
	titleOp.GeoM.Translate(logBottomMargin, logBottomMargin+game.latinBaselineAdjust(30, text.AlignStart))
	titleOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, "EVENT LOG", titleFace, titleOp)

	bounds := game.LogEntryImg.Bounds()
	imgH := float64(bounds.Dy())

	imgStartX := logImageStartX
	imgStartY := logImageStartY

	if len(log) == 0 {
		return
	}

	maxTextLinesF := (imgH - logTextOffsetY) / logTextLineH
	maxTextLines := int(maxTextLinesF)
	if maxTextLines < 1 {
		maxTextLines = 1
	}

	entries := make([]logEntryLayout, 0, len(log))
	for _, cmd := range log {
		lines := wrapLines(cmd.Text)
		if len(lines) > maxTextLines {
			lines = lines[:maxTextLines]
			last := lines[len(lines)-1]
			lines[len(lines)-1] = last + "…"
		}
		entries = append(entries, logEntryLayout{cmd: cmd, lines: lines})
	}

	maxScroll := float64(len(entries) - logVisibleCount)
	if maxScroll < 0 {
		maxScroll = 0
	}

	intScroll := math.Floor(scrollOffset)
	frac := scrollOffset - intScroll

	endIndex := len(entries) - int(intScroll)
	startIndex := endIndex - logVisibleCount

	rowPitch := imgH + logEntryGap

	imgW := float64(bounds.Dx())
	contentH := float64(logVisibleCount)*imgH + float64(logVisibleCount-1)*logEntryGap
	listRect := image.Rect(int(imgStartX), int(imgStartY), int(imgStartX+imgW), int(imgStartY+contentH))
	dst := screen.SubImage(listRect).(*ebiten.Image)

	for dispIdx := -1; dispIdx <= logVisibleCount; dispIdx++ {
		idx := startIndex + dispIdx
		if idx < 0 || idx >= len(entries) {
			continue
		}
		e := entries[idx]
		imgY := imgStartY + float64(dispIdx)*rowPitch + frac*rowPitch
		imgX := imgStartX
		isSelected := idx == cursorIndex

		imgOp := &ebiten.DrawImageOptions{}
		imgOp.GeoM.Translate(imgX, imgY)
		if isSelected {
			pulse := selectedLogPulseColor(game.TotalPlayTime)
			imgOp.ColorScale.Scale(
				float32(pulse.R)/255,
				float32(pulse.G)/255,
				float32(pulse.B)/255,
				1,
			)
		}
		dst.DrawImage(game.LogEntryImg, imgOp)

		speakerLabel := e.cmd.Speaker
		if speakerLabel == "" {
			speakerLabel = "・・・"
		}
		nameX := imgX + logNameOffsetX
		nameY := imgY + logNameOffsetY
		game.DrawMixedText(dst, speakerLabel, 13, nameX, nameY, text.AlignStart, text.AlignStart, logTextColor)

		textX := imgX + logTextOffsetX
		textY := imgY + logTextOffsetY
		for li, line := range e.lines {
			game.DrawMixedText(dst, line, 13, textX, textY+float64(li)*logTextLineH, text.AlignStart, text.AlignStart, logTextColor)
		}
	}

	if maxScroll > 0 {
		barX, trackY, trackHeight, thumbH, moveRange := logScrollBarGeometry(game, len(entries))
		ebitenutil.DrawRect(screen, barX, trackY, scrollBarTrackWidth, trackHeight, scrollBarTrackColor)

		ratio := 1 - scrollOffset/maxScroll
		if ratio < 0 {
			ratio = 0
		} else if ratio > 1 {
			ratio = 1
		}
		thumbY := trackY + ratio*moveRange

		ebitenutil.DrawRect(screen, barX, thumbY, scrollBarCursorWidth, thumbH, scrollBarCursorColor)
	}
}

func logScrollBarGeometry(game *Game, entryCount int) (barX, trackY, trackHeight, thumbH, moveRange float64) {
	imgH := float64(game.LogEntryImg.Bounds().Dy())
	colX := (float64(gameWidth) - logColumnWidth) / 2
	barX = colX + logColumnWidth + logScrollBarGap
	trackHeight = float64(logVisibleCount)*imgH + float64(logVisibleCount-1)*logEntryGap
	trackY = (float64(gameHeight) - trackHeight) / 2
	if entryCount <= 0 {
		entryCount = 1
	}
	thumbH = trackHeight * (float64(logVisibleCount) / float64(entryCount))
	if thumbH < logScrollBarMinH {
		thumbH = logScrollBarMinH
	}
	moveRange = trackHeight - thumbH
	if moveRange < 0 {
		moveRange = 0
	}
	return
}

func logScrollBarRect(game *Game, entryCount int) (x, y, w, h float64) {
	barX, trackY, trackHeight, _, _ := logScrollBarGeometry(game, entryCount)
	x = barX - scrollBarTouchPad
	y = trackY
	w = scrollBarCursorWidth + scrollBarTouchPad*2
	h = trackHeight
	return
}

func hitTestLogEntries(game *Game, entryCount int, scrollOffset, tapX, tapY float64) (int, bool) {
	if entryCount == 0 {
		return -1, false
	}
	bounds := game.LogEntryImg.Bounds()
	imgW := float64(bounds.Dx())
	imgH := float64(bounds.Dy())
	rowPitch := imgH + logEntryGap

	intScroll := math.Floor(scrollOffset)
	frac := scrollOffset - intScroll
	endIndex := entryCount - int(intScroll)
	startIndex := endIndex - logVisibleCount

	p := touchPoint{tapX, tapY}
	for dispIdx := 0; dispIdx < logVisibleCount; dispIdx++ {
		idx := startIndex + dispIdx
		if idx < 0 || idx >= entryCount {
			continue
		}
		imgY := logImageStartY + float64(dispIdx)*rowPitch + frac*rowPitch
		r := tapRect{x: logImageStartX, y: imgY, w: imgW, h: imgH}
		if r.contains(p) {
			return idx, true
		}
	}
	return -1, false
}
