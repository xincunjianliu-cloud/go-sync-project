package main

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func (s *FieldScene) Draw(screen *ebiten.Image) {
	if s == nil {
		return
	}
	screen.Fill(color.RGBA{30, 30, 30, 255})
	camX, camY := s.cameraPosition()

	// ★修正：タイルセット画像は s.mapTileImg（シーン固有）を使う。
	// s.game.TileImg はグローバル単一状態で、フェード遷移中に他シーンから
	// 上書きされると誤った画像で描画される事故が起きるため参照しない。
	// ★修正：レイヤーごとに再計算していた列数はマップ全体で不変なのでループ外へ。
	tilesetCols := 1
	if s.mapTileImg != nil && s.tileMap.TileWidth > 0 {
		tilesetCols = s.mapTileImg.Bounds().Dx() / s.tileMap.TileWidth
		if tilesetCols <= 0 {
			tilesetCols = 1
		}
	}

	for _, layer := range s.tileMap.Layers {
		if layer.Type != "tilelayer" {
			continue
		}

		for i, id := range layer.Data {
			if id == 0 {
				continue
			}
			tx := (i % s.tileMap.Width) * s.tileMap.TileWidth
			ty := (i / s.tileMap.Width) * s.tileMap.TileHeight
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(tx)+camX, float64(ty)+camY)
			tileID := id - 1

			sx := (tileID % tilesetCols) * s.tileMap.TileWidth
			sy := (tileID / tilesetCols) * s.tileMap.TileHeight
			rect := image.Rect(sx, sy, sx+s.tileMap.TileWidth, sy+s.tileMap.TileHeight)

			if s.mapTileImg != nil {
				screen.DrawImage(s.mapTileImg.SubImage(rect).(*ebiten.Image), op)
			}
		}
	}

	for _, e := range s.enemies {
		if idx, ok := bossIndexFromEnemyType(e.Type); ok {
			if s.game.BossSpriteSheets[idx] != nil {
				bossFrame := int((ebiten.Tick() / 20) % 2)
				bossW := 16
				bossH := 32

				bossSx := bossFrame * bossW
				bossSy := 0

				rect := image.Rect(bossSx, bossSy, bossSx+bossW, bossSy+bossH)

				opB := &ebiten.DrawImageOptions{}
				opB.GeoM.Translate(e.x+camX, (e.y-16)+camY)

				screen.DrawImage(s.game.BossSpriteSheets[idx].SubImage(rect).(*ebiten.Image), opB)
			} else {
				ebitenutil.DrawRect(screen, e.x+camX, e.y+camY, 16, 16, color.RGBA{150, 0, 255, 255})
			}
		}
	}

	state := MoveStateIdle
	if s.animCount > 0 {
		if s.game.IsDashing {
			state = MoveStateDash
		} else {
			state = MoveStateWalk
		}
	}
	frame := s.playerCfg.AnimFrame(s.dir, s.animCount)
	sx, sy, fw, fh := s.playerCfg.SourceRect(s.dir, state, frame)
	drawX, drawY := s.playerCfg.DrawTopLeftAt(s.px, s.py)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(s.playerCfg.Scale, s.playerCfg.Scale)
	op.GeoM.Translate(drawX+camX, drawY+camY)

	if s.game.SpriteSheet != nil {
		rect := image.Rect(sx, sy, sx+fw, sy+fh)
		screen.DrawImage(s.game.SpriteSheet.SubImage(rect).(*ebiten.Image), op)
	} else {
		ebitenutil.DrawRect(screen, s.px+camX, s.py+camY, 16, 16, color.RGBA{0, 255, 0, 255})
	}

	if s.isMsgActive {
		s.msg.DrawChara(screen, s.game.CharaImgs)
	}

	if s.isMsgActive && s.msgIndex >= 0 && s.msgIndex < len(s.msgTexts) {
		s.msg.Draw(screen, s.msgTexts[s.msgIndex], s.game.FontFace(15))
	}

	if s.isMsgActive {
		drawSkipHint(screen, s.game, s.msgSkipHoldElapsed, endingSkipHoldSeconds)
		drawMessageKeyGuide(screen, s.game)
	}

	if s.nearExamineEvent && !s.isMsgActive && !s.isChoiceActive && !s.isCutscene {
		s.drawPromptWithIcon(screen, s.game.ExamineIconImg, "▼ 調べる")
	}

	if s.nearDoorEvent && !s.isMsgActive && !s.isChoiceActive && !s.isCutscene {
		s.drawPromptWithIcon(screen, s.game.ExamineIconImg, "▼ 進む")
	}

	if s.cseFadeAlpha > 0 {
		ebitenutil.DrawRect(screen, 0, 0,
			float64(gameWidth), float64(gameHeight),
			scaleAlpha(color.RGBA{0, 0, 0, 255}, s.cseFadeAlpha))
	}

	if s.isChoiceActive {
		s.drawChoiceUI(screen, camX, camY)
	}

	if s.isLogActive {
		s.logScrollOffset = drawMessageLog(screen, s.game, s.msgLog, s.logScrollOffset, s.logCursorIndex)
	}
}

func (s *FieldScene) drawPromptWithIcon(screen *ebiten.Image, icon *ebiten.Image, label string) {
	face := s.game.FontFace(15)

	textW, textH := text.Measure(label, face, 0)

	centerX := float64(gameWidth) / 2
	baseY := float64(gameHeight) - 32

	textX := centerX - textW/2
	textY := baseY - textH/2

	if icon != nil {
		imgW := icon.Bounds().Dx()
		imgH := icon.Bounds().Dy()
		imgOp := &ebiten.DrawImageOptions{}
		imgOp.GeoM.Translate(
			centerX-float64(imgW)/2,
			baseY-float64(imgH)/2,
		)
		screen.DrawImage(icon, imgOp)
	}

	op := &text.DrawOptions{}
	op.GeoM.Translate(textX, textY)
	op.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, label, face, op)
}

// enemyDrawRect は現在の敵画像の表示位置とサイズを返す（ボス差し替え込み）
func (s *BattleScene) enemyDrawRect() (x, y, w, h float64) {
	if s.enemyImage == nil {
		return 280.0, 240.0, 0, 0
	}
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
	return xPos, yPos, float64(imgW), float64(imgH)
}

// enemyCenter は敵画像の中心座標を返す
func (s *BattleScene) enemyCenter() (float64, float64) {
	x, y, w, h := s.enemyDrawRect()
	return x + w/2, y + h/2
}

func (s *FieldScene) drawChoiceUI(screen *ebiten.Image, camX, camY float64) {
	const boxW, boxH = 160.0, 90.0

	bx := s.choiceAnchorX + camX - boxW/2
	by := s.choiceAnchorY + camY - boxH - 10

	if bx < 4 {
		bx = 4
	}
	if bx+boxW > gameWidth-4 {
		bx = gameWidth - 4 - boxW
	}
	if by < 4 {
		by = 4
	}

	ebitenutil.DrawRect(screen, bx, by, boxW, boxH, uiColorPanelBg)

	face := s.game.FontFace(14)

	qOp := &text.DrawOptions{}
	qOp.GeoM.Translate(bx+10, by+10)
	qOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, s.choiceQuestion, face, qOp)

	for i, opt := range s.choiceOptions {
		label := "　" + opt
		if i == s.choiceIndex {
			label = "▶" + opt
		}
		op := &text.DrawOptions{}
		op.GeoM.Translate(bx+10, by+40+float64(i)*24)
		op.ColorScale.ScaleWithColor(uiColorText)
		text.Draw(screen, label, face, op)
	}
}
