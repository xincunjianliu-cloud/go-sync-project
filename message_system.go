package main

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	msgWinX             = 0
	msgWinY             = 450
	msgWinWidth         = 960
	msgWinHeight        = 90
	msgTextSpeedDefault = 5
	msgTextStartX       = msgWinX + 20

	charaInsideX  = -50.0
	charaOutsideX = -300.0

	charaRightInsideX  = 50.0
	charaRightOutsideX = 300.0

	charaSlideSpeed = 0.12
	charaFadeSpeed  = 0.08
	charaDarkTone   = 0.45

	msgNameXRatio = 160.0 / float64(gameWidth)
	msgNameYRatio = 410.0 / float64(gameHeight)

	msgTextStartXRatio = 180.0 / float64(gameWidth)
	msgTextStartYRatio = 450.0 / float64(gameHeight)

	msgArrowXRatio = 720.0 / float64(gameWidth)
	msgArrowYRatio = 510.0 / float64(gameHeight)

	msgLineSpacingExtra = 8.0
	msgArrowFontSize    = 10.0
)

type charaState struct {
	x       float64
	targetX float64
	light   float32
	spawned bool
}

type MessageSystem struct {
	ticks    int
	msgStart int

	Speed int

	WindowImg   *ebiten.Image
	WindowAlpha float32

	SpeakerToSlot map[string]int

	slots [2]charaState
}

func (m *MessageSystem) speed() int {
	if m.Speed <= 0 {
		return msgTextSpeedDefault
	}
	return m.Speed
}
func (m *MessageSystem) Start() {
	m.msgStart = m.ticks

}

func (m *MessageSystem) Tick() {
	m.ticks++
}

func (m *MessageSystem) IsFinished(cmd EventCommand) bool {
	runes := []rune(cmd.Text)
	count := (m.ticks - m.msgStart) / m.speed()
	return count >= len(runes)
}

func (m *MessageSystem) SkipToEnd(cmd EventCommand) {
	runes := []rune(cmd.Text)
	m.msgStart = m.ticks - len(runes)*m.speed()
}

func (m *MessageSystem) UpdateCharaAnim(currentSpeaker string, charaImgs map[string]*ebiten.Image) {
	if m.SpeakerToSlot == nil {
		return
	}

	activeSlot := -1
	if currentSpeaker != "" && currentSpeaker != "SYSTEM_COMMAND" {
		if slot, ok := m.SpeakerToSlot[currentSpeaker]; ok {
			activeSlot = slot
			if !m.slots[slot].spawned {
				if slot == 0 {
					m.slots[slot].x = charaOutsideX
					m.slots[slot].targetX = charaInsideX
				} else {
					m.slots[slot].x = charaRightOutsideX
					m.slots[slot].targetX = charaRightInsideX
				}
				m.slots[slot].spawned = true
			}
		}
	}

	for i := range m.slots {
		if !m.slots[i].spawned {
			continue
		}

		m.slots[i].x += (m.slots[i].targetX - m.slots[i].x) * charaSlideSpeed

		var target float32 = charaDarkTone
		if i == activeSlot || activeSlot == -1 {
			target = 1.0
		}
		if m.slots[i].light < target {
			m.slots[i].light += charaFadeSpeed
			if m.slots[i].light > target {
				m.slots[i].light = target
			}
		} else if m.slots[i].light > target {
			m.slots[i].light -= charaFadeSpeed
			if m.slots[i].light < target {
				m.slots[i].light = target
			}
		}
	}
}

func (m *MessageSystem) DrawChara(screen *ebiten.Image, charaImgs map[string]*ebiten.Image) {
	if m.SpeakerToSlot == nil {
		return
	}

	for speaker, slot := range m.SpeakerToSlot {
		s := &m.slots[slot]
		if !s.spawned {
			continue
		}
		img := charaImgs[speaker]

		charaY := 0.0
		imgW := float64(img.Bounds().Dx())

		var drawX float64
		if slot == 0 {
			drawX = s.x
		} else {
			drawX = float64(gameWidth) - imgW + s.x
		}

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(drawX, charaY)
		op.ColorScale.Scale(s.light, s.light, s.light, 1.0)
		op.Filter = ebiten.FilterNearest
		screen.DrawImage(img, op)
	}
}

func (m *MessageSystem) Draw(screen *ebiten.Image, cmd EventCommand, fontFace *text.GoTextFace) {
	winOp := &ebiten.DrawImageOptions{}
	winOp.GeoM.Translate(0, 0)
	winOp.Filter = ebiten.FilterNearest
	screen.DrawImage(m.WindowImg, winOp)

	if cmd.Speaker != "" && cmd.Speaker != "SYSTEM_COMMAND" {
		nameOp := &text.DrawOptions{}
		nameX := float64(gameWidth) * msgNameXRatio
		nameY := float64(gameHeight) * msgNameYRatio
		nameOp.GeoM.Translate(nameX, nameY)
		nameOp.ColorScale.ScaleWithColor(uiColorText)
		text.Draw(screen, ""+cmd.Speaker+"", fontFace, nameOp)
	}

	runes := []rune(cmd.Text)
	count := (m.ticks - m.msgStart) / m.speed()
	if count > len(runes) {
		count = len(runes)
	}
	visibleText := string(runes[:count])

	lines := strings.Split(visibleText, "\n")

	textOp := &text.DrawOptions{}
	textOp.LineSpacing = fontFace.Metrics().HAscent + fontFace.Metrics().HDescent + msgLineSpacingExtra
	textOp.ColorScale.ScaleWithColor(uiColorText)

	startY := float64(gameHeight) * msgTextStartYRatio
	startX := float64(gameWidth) * msgTextStartXRatio

	for i, line := range lines {
		if i >= 3 {
			break
		}
		textOp.GeoM.Reset()
		textOp.GeoM.Translate(startX, startY+float64(i)*textOp.LineSpacing)
		text.Draw(screen, line, fontFace, textOp)
	}

	if count >= len(runes) && len(runes) > 0 {
		if (m.ticks/30)%2 == 0 {
			arrowFace := &text.GoTextFace{Source: fontFace.Source, Size: msgArrowFontSize}
			arrowOp := &text.DrawOptions{}
			arrowX := float64(gameWidth) * msgArrowXRatio
			arrowY := float64(gameHeight) * msgArrowYRatio
			arrowOp.GeoM.Translate(arrowX, arrowY)
			arrowOp.ColorScale.ScaleWithColor(uiColorText)
			text.Draw(screen, "▼", arrowFace, arrowOp)
		}
	}
}

func (m *MessageSystem) Reset() {
	m.slots[0] = charaState{}
	m.slots[1] = charaState{}
}
