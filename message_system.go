package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// =========================================================
// MessageSystem
// フィールドシーン用のテキスト描画システム。
// タイプライター効果・▼点滅・window.png対応。
// キャラ立ち絵のスライドイン/アウト＋明暗フェード対応。
//
// 使い方:
//   1. FieldScene に MessageSystem フィールドを追加する
//      msg MessageSystem
//sss
//      g.CharaImgs = map[string]*ebiten.Image{ "パラガス": img, ... }
//
//   3. msg.SpeakerToSlot に話者名→スロット番号を登録する
//      スロット 0 = 左側、スロット 1 = 右側
//      s.msg.SpeakerToSlot = map[string]int{ "パラガス": 0, "ブロリー": 1 }
//
//   4. メッセージ開始時に Start() を呼ぶ
//      s.msg.Start()

//
//   5. Update() 内で Tick() と UpdateCharaAnim() を呼ぶ（毎フレーム）
//      s.msg.Tick()
//      s.msg.UpdateCharaAnim(currentSpeaker, charaImgs)
//
//   6. Draw() 内で DrawChara() → Draw() の順に呼ぶ
//      s.msg.DrawChara(screen, charaImgs)
//      s.msg.Draw(screen, cmd, fontFace)
// =========================================================

const (
	msgWinX             = 0
	msgWinY             = 450
	msgWinWidth         = 960
	msgWinHeight        = 90
	msgTextSpeedDefault = 5
	msgTextStartX       = msgWinX + 20

	// ── キャラ立ち絵：左スロット(0) ──
	charaInsideX  = -50.0  // 画面内で静止する位置（左端基準）
	charaOutsideX = -300.0 // 画面外（登場前・退場後）の位置

	// ── キャラ立ち絵：右スロット(1) ──
	// 左と同じ考え方で、右端からの距離として管理する
	charaRightInsideX  = 50.0  // 画面内で静止する時の、右端からの距離
	charaRightOutsideX = 300.0 // 画面外（登場前・退場後）の、右端からの距離

	charaSlideSpeed = 0.12
	charaFadeSpeed  = 0.08
	charaDarkTone   = 0.45

	// ── テキスト位置：解像度に対する比率 ──
	msgNameXRatio = 160.0 / float64(gameWidth)
	msgNameYRatio = 410.0 / float64(gameHeight)

	msgTextStartXRatio = 180.0 / float64(gameWidth)
	msgTextStartYRatio = 450.0 / float64(gameHeight) // 話者名の有無に関わらず常にこの位置

	msgArrowXRatio = 720.0 / float64(gameWidth)
	msgArrowYRatio = 510.0 / float64(gameHeight)

	msgLineSpacingExtra = 8.0 // 行間の追加余白
	msgArrowFontSize    = 10.0
)

// charaState はキャラ1体分のアニメーション状態を持つ
// charaState はキャラ1体分のアニメーション状態を持つ
type charaState struct {
	x       float64 // 現在の位置（左スロット=左基準のX、右スロット=右端からの距離）
	targetX float64 // 目標位置
	light   float32 // 現在の輝度 (0.0〜1.0)
	spawned bool    // 一度でも登場したか
}

type MessageSystem struct {
	ticks    int
	msgStart int

	Speed int // 文字送り速度（tick数）。0以下ならデフォルトを使う ← 追加

	WindowImg   *ebiten.Image
	WindowAlpha float32

	// 話者名 → スロット番号 (0=左, 1=右) のマッピング
	// 外から設定する: s.msg.SpeakerToSlot = map[string]int{"パラガス": 0, "ブロリー": 1}
	SpeakerToSlot map[string]int

	// スロット 0（左）・スロット 1（右）のアニメーション状態
	slots [2]charaState
}

// Start はメッセージ表示を開始する。新しいページに切り替わるたびに呼ぶこと。

func (m *MessageSystem) speed() int {
	if m.Speed <= 0 {
		return msgTextSpeedDefault
	}
	return m.Speed
}
func (m *MessageSystem) Start() {
	m.msgStart = m.ticks

}

// Tick は毎フレーム Update() 内で呼ぶ。
func (m *MessageSystem) Tick() {
	m.ticks++
}

// IsFinished は現在のメッセージの文字がすべて表示されたか返す。
func (m *MessageSystem) IsFinished(cmd EventCommand) bool {
	runes := []rune(cmd.Text)
	count := (m.ticks - m.msgStart) / m.speed()
	return count >= len(runes)
}

// SkipToEnd は文字送りをスキップして全文を即時表示する。
func (m *MessageSystem) SkipToEnd(cmd EventCommand) {
	runes := []rune(cmd.Text)
	m.msgStart = m.ticks - len(runes)*m.speed()
}

// UpdateCharaAnim はキャラのスライド・明暗を毎フレーム更新する。
// currentSpeaker: 現在の話者名。charaImgs: game.CharaImgs を渡す。
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

		// スライド更新（左右とも同じ式）
		m.slots[i].x += (m.slots[i].targetX - m.slots[i].x) * charaSlideSpeed

		// 明暗フェード更新
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

// DrawChara はウィンドウより前、プレイヤーより後に呼ぶ（ウィンドウの手前に立つ）。
// charaImgs: game.CharaImgs を渡す。
func (m *MessageSystem) DrawChara(screen *ebiten.Image, charaImgs map[string]*ebiten.Image) {
	if m.SpeakerToSlot == nil {
		return
	}

	for speaker, slot := range m.SpeakerToSlot {
		s := &m.slots[slot]
		if !s.spawned {
			continue
		}
		img, ok := charaImgs[speaker]
		if !ok || img == nil {
			continue
		}

		charaY := 0.0
		imgW := float64(img.Bounds().Dx())

		var drawX float64
		if slot == 0 {
			drawX = s.x
		} else {
			// 右端基準：x=0の時、画像の右端が画面の右端にぴったり合う
			drawX = float64(gameWidth) - imgW + s.x
		}

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(drawX, charaY)
		op.ColorScale.Scale(s.light, s.light, s.light, 1.0)
		op.Filter = ebiten.FilterNearest
		screen.DrawImage(img, op)
	}
}

// Draw はメッセージウィンドウとテキストを描画する。
// isMsgActive が true のときだけ呼ぶこと。
func (m *MessageSystem) Draw(screen *ebiten.Image, cmd EventCommand, fontFace *text.GoTextFace) {
	// --- 1. ウィンドウ背景 ---
	alpha := m.WindowAlpha
	if alpha == 0 {
		alpha = 1.0
	}

	if m.WindowImg != nil {
		winOp := &ebiten.DrawImageOptions{}
		winOp.GeoM.Translate(0, 0)
		winOp.ColorScale.Scale(1.0, 1.0, 1.0, alpha)
		winOp.Filter = ebiten.FilterNearest
		screen.DrawImage(m.WindowImg, winOp)
	} else {
		alphaByte := uint8(255 * alpha)
		ebitenutil.DrawRect(screen, msgWinX, msgWinY, msgWinWidth, msgWinHeight, color.RGBA{0, 0, 0, alphaByte})
	}

	// --- 2. 話者名 ---
	if cmd.Speaker != "" && cmd.Speaker != "SYSTEM_COMMAND" {
		nameOp := &text.DrawOptions{}
		nameX := float64(gameWidth) * msgNameXRatio
		nameY := float64(gameHeight) * msgNameYRatio
		nameOp.GeoM.Translate(nameX, nameY)
		nameOp.ColorScale.ScaleWithColor(uiColorText)
		text.Draw(screen, ""+cmd.Speaker+"", fontFace, nameOp)
	}

	// --- 3. タイプライター効果 ---
	runes := []rune(cmd.Text)
	count := (m.ticks - m.msgStart) / m.speed()
	if count > len(runes) {
		count = len(runes)
	}
	visibleText := string(runes[:count])

	// マージンなし：ウィンドウ幅そのまま折り返し判定に使う
	textMaxWidth := float64(msgWinWidth)
	var lines []string
	var currentLine string
	for _, r := range visibleText {
		if r == '\n' {
			lines = append(lines, currentLine)
			currentLine = ""
			continue
		}
		testLine := currentLine + string(r)
		if text.Advance(testLine, fontFace) > textMaxWidth {
			lines = append(lines, currentLine)
			currentLine = string(r)
		} else {
			currentLine = testLine
		}
	}
	if currentLine != "" || len(lines) == 0 {
		lines = append(lines, currentLine)
	}

	textOp := &text.DrawOptions{}
	textOp.LineSpacing = fontFace.Metrics().HAscent + fontFace.Metrics().HDescent + msgLineSpacingExtra
	textOp.ColorScale.ScaleWithColor(uiColorText)

	// 話者名の有無に関わらず常に同じY座標
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

	// --- 4. ▼ 点滅 ---
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

// message_system.go に追加
func (m *MessageSystem) Reset() {
	m.slots[0] = charaState{}
	m.slots[1] = charaState{}
}
