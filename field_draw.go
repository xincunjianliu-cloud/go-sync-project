package main

import (
	"image"
	"image/color"
	"strings"

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
	camX += s.screenShakeX
	camY += s.screenShakeY

	tilesetCols := 1
	if s.tileMap.TileWidth > 0 {
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

			screen.DrawImage(s.mapTileImg.SubImage(rect).(*ebiten.Image), op)
		}
	}

	s.drawChests(screen, camX, camY)
	s.drawLockedWalls(screen, camX, camY)
	s.drawLevers(screen, camX, camY)
	s.drawBlockSpots(screen, camX, camY)
	s.drawBlocks(screen, camX, camY)
	s.drawBlockDoors(screen, camX, camY)

	for _, e := range s.enemies {
		if idx, ok := bossIndexFromEnemyType(e.Type); ok {
			bossFrame := int((ebiten.Tick() / 20) % 2)
			bossW := 64
			bossH := 96

			bossSx := bossFrame * bossW
			bossSy := 0

			rect := image.Rect(bossSx, bossSy, bossSx+bossW, bossSy+bossH)

			opB := &ebiten.DrawImageOptions{}
			opB.GeoM.Translate(e.x+camX, (e.y-16)+camY)

			screen.DrawImage(s.game.BossSpriteSheets[idx].SubImage(rect).(*ebiten.Image), opB)
		}
	}

	state := MoveStateIdle
	if s.animCount > 0 {
		if s.isDashingNow {
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

	rect := image.Rect(sx, sy, sx+fw, sy+fh)
	screen.DrawImage(s.game.SpriteSheet.SubImage(rect).(*ebiten.Image), op)

	s.drawDarkness(screen, camX, camY)

	if s.isMsgActive {
		s.msg.DrawChara(screen, s.game.CharaImgs)
	}

	if s.isMsgActive && s.msgIndex >= 0 && s.msgIndex < len(s.msgTexts) {
		s.msg.Draw(screen, s.msgTexts[s.msgIndex], s.game.FontFace(15))
	}

	if s.isMsgActive {
		drawMessageControlPanel(screen, s.game, s.autoMode, s.msgSkipHoldElapsed, endingSkipHoldSeconds)
	}

	if s.nearExamineEvent && !s.isMsgActive && !s.isChoiceActive && !s.isCutscene && !s.isItemGetActive {
		s.drawPromptWithIcon(screen, s.game.ExamineIconImg, "▼ 調べる")
	}

	if s.nearDoorEvent && !s.isMsgActive && !s.isChoiceActive && !s.isCutscene && !s.isItemGetActive {
		s.drawPromptWithIcon(screen, s.game.ExamineIconImg, "▼ 進む")
	}

	if s.isPushingBlock && !s.isMsgActive && !s.isChoiceActive && !s.isCutscene && !s.isItemGetActive {
		s.drawPromptWithIcon(screen, s.game.ExamineIconImg, "▼ はなす")
	}

	if s.cseFadeAlpha > 0 {
		ebitenutil.DrawRect(screen, 0, 0,
			float64(gameWidth), float64(gameHeight),
			scaleAlpha(color.RGBA{0, 0, 0, 255}, s.cseFadeAlpha))
	}

	if s.isChoiceActive {
		s.drawChoiceUI(screen, camX, camY)
	}

	if s.isItemGetActive {
		s.drawItemGetPopup(screen)
		drawBackButton(screen, s.game)
	}

	if s.isLogActive {
		drawMessageLog(screen, s.game, s.msgLog, s.logScrollOffset, s.logCursorIndex)
		drawBackButton(screen, s.game)
	}

	showTouchPad := !s.isMsgActive && !s.isChoiceActive && !s.isCutscene && !s.isItemGetActive && !s.isLogActive && !s.wallFadeActive
	if s.game.MobileMode {
		showActionButton := !s.isMsgActive && !s.isItemGetActive && !s.wallFadeActive
		s.drawTouchControls(screen, showTouchPad, showActionButton)
	}
	if showTouchPad {
		drawHamburgerMenuButton(screen, s.game)
	}
}

func (s *FieldScene) drawChests(screen *ebiten.Image, camX, camY float64) {
	const chestFrameW, chestFrameH = 40, 32

	for _, layer := range s.tileMap.Layers {
		if !strings.HasPrefix(layer.Name, "events") {
			continue
		}
		for _, obj := range layer.Objects {
			p := objProps(obj)
			if p["type"] != "event" {
				continue
			}
			if !strings.HasPrefix(p["text"], "event_chest_") {
				continue
			}

			frame := 0
			if s.game.OpenedChests[chestKey(s.currentMap, obj)] {
				frame = 1
			}

			chestImg := s.game.ChestImg
			if strings.HasPrefix(p["text"], "event_chest_key_") && s.game.KeyChestImg != nil {
				chestImg = s.game.KeyChestImg
			}

			sx := frame * chestFrameW
			rect := image.Rect(sx, 0, sx+chestFrameW, chestFrameH)

			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(obj.X+camX, obj.Y+camY)
			screen.DrawImage(chestImg.SubImage(rect).(*ebiten.Image), op)
		}
	}
}

func (s *FieldScene) drawLockedWalls(screen *ebiten.Image, camX, camY float64) {
	for _, layer := range s.tileMap.Layers {
		if !strings.HasPrefix(layer.Name, "events") {
			continue
		}
		for _, obj := range layer.Objects {
			p := objProps(obj)
			if p["type"] != "event" || p["text"] != "event_wall" {
				continue
			}

			isLeverWall := p["lever"] != ""
			open := s.wallIsOpen(obj)

			if !isLeverWall && open {
				continue
			}

			img := s.game.LockedWallImg
			if isLeverWall {
				img = s.game.LeverWallImg
			}
			if filename := p["image"]; filename != "" {
				if custom := s.game.LoadFieldImage(filename); custom != nil {
					img = custom
				}
			}
			if img == nil {
				continue
			}

			var srcRect image.Rectangle
			var frameW, frameH float64
			if isLeverWall {
				fw := img.Bounds().Dx() / 2
				fh := img.Bounds().Dy()
				if fw <= 0 || fh <= 0 {
					continue
				}
				frame := 0
				if open {
					frame = 1
				}
				sx := frame * fw
				srcRect = image.Rect(sx, 0, sx+fw, fh)
				frameW, frameH = float64(fw), float64(fh)
			} else {
				srcRect = img.Bounds()
				frameW = float64(img.Bounds().Dx())
				frameH = float64(img.Bounds().Dy())
			}
			if frameW <= 0 || frameH <= 0 {
				continue
			}

			op := &ebiten.DrawImageOptions{}
			if obj.Width > 0 && obj.Height > 0 {
				op.GeoM.Scale(obj.Width/frameW, obj.Height/frameH)
			}
			op.GeoM.Translate(obj.X+camX, obj.Y+camY)

			if s.wallFadeActive && !isLeverWall && chestKey(s.currentMap, obj) == s.wallFadeKey {
				op.ColorScale.ScaleAlpha(float32(s.wallFadeAlpha))
			}

			screen.DrawImage(img.SubImage(srcRect).(*ebiten.Image), op)
		}
	}
}

const (
	leverFrameW = 12
	leverFrameH = 20
)

func (s *FieldScene) drawLevers(screen *ebiten.Image, camX, camY float64) {
	img := s.game.LeverImg
	if img == nil {
		return
	}
	const frameW, frameH = leverFrameW, leverFrameH

	for _, layer := range s.tileMap.Layers {
		if !strings.HasPrefix(layer.Name, "events") {
			continue
		}
		for _, obj := range layer.Objects {
			p := objProps(obj)
			if p["type"] != "event" || p["text"] != "event_lever" {
				continue
			}

			frame := 0
			if s.game.RaisedLevers[p["id"]] {
				frame = 1
			}

			sx := frame * frameW
			rect := image.Rect(sx, 0, sx+frameW, frameH)

			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(obj.X+camX, obj.Y+camY)
			screen.DrawImage(img.SubImage(rect).(*ebiten.Image), op)
		}
	}
}

func (s *FieldScene) drawBlockSpots(screen *ebiten.Image, camX, camY float64) {
	img := s.game.BlockSpotImg
	if img == nil {
		return
	}
	imgW := float64(img.Bounds().Dx())
	imgH := float64(img.Bounds().Dy())
	if imgW <= 0 || imgH <= 0 {
		return
	}

	for _, layer := range s.tileMap.Layers {
		if !strings.HasPrefix(layer.Name, "events") {
			continue
		}
		for _, obj := range layer.Objects {
			p := objProps(obj)
			if p["type"] != "event" || p["text"] != "event_blockspot" {
				continue
			}

			op := &ebiten.DrawImageOptions{}
			if obj.Width > 0 && obj.Height > 0 {
				op.GeoM.Scale(obj.Width/imgW, obj.Height/imgH)
			}
			op.GeoM.Translate(obj.X+camX, obj.Y+camY)
			screen.DrawImage(img, op)
		}
	}
}

func (s *FieldScene) drawBlocks(screen *ebiten.Image, camX, camY float64) {
	img := s.game.BlockImg
	if img == nil {
		return
	}
	imgW := float64(img.Bounds().Dx())
	imgH := float64(img.Bounds().Dy())
	if imgW <= 0 || imgH <= 0 {
		return
	}

	for _, b := range s.blocks {
		op := &ebiten.DrawImageOptions{}
		if b.Width > 0 && b.Height > 0 {
			op.GeoM.Scale(b.Width/imgW, b.Height/imgH)
		}
		op.GeoM.Translate(b.X+camX, b.Y+camY)
		screen.DrawImage(img, op)
	}
}

func (s *FieldScene) drawBlockDoors(screen *ebiten.Image, camX, camY float64) {
	img := s.game.BlockDoorImg
	if img == nil {
		return
	}
	imgW := float64(img.Bounds().Dx())
	imgH := float64(img.Bounds().Dy())
	if imgW <= 0 || imgH <= 0 {
		return
	}

	for _, layer := range s.tileMap.Layers {
		if !strings.HasPrefix(layer.Name, "events") {
			continue
		}
		for _, obj := range layer.Objects {
			p := objProps(obj)
			if p["type"] != "event" || p["text"] != "event_blockdoor" {
				continue
			}
			if s.blockDoorIsOpen(obj) {
				continue
			}

			op := &ebiten.DrawImageOptions{}
			if obj.Width > 0 && obj.Height > 0 {
				op.GeoM.Scale(obj.Width/imgW, obj.Height/imgH)
			}
			op.GeoM.Translate(obj.X+camX, obj.Y+camY)
			screen.DrawImage(img, op)
		}
	}
}

func (s *FieldScene) drawDarkness(screen *ebiten.Image, camX, camY float64) {
	if !s.isDarknessActive {
		return
	}
	mask := s.game.LightMaskImg
	if mask == nil {
		return
	}

	if s.darknessOverlay == nil {
		s.darknessOverlay = ebiten.NewImage(gameWidth, gameHeight)
	}
	overlay := s.darknessOverlay
	overlay.Fill(color.RGBA{0, 0, 0, 255})

	drawX, drawY := s.playerCfg.DrawTopLeftAt(s.px, s.py)
	spriteW := float64(s.playerCfg.FrameWidth) * s.playerCfg.Scale
	spriteH := float64(s.playerCfg.FrameHeight) * s.playerCfg.Scale
	centerX := drawX + spriteW/2 + camX
	centerY := drawY + spriteH/2 + camY

	maskW := float64(mask.Bounds().Dx())
	scale := (s.darknessRadius * 2) / maskW

	holeOp := &ebiten.DrawImageOptions{}
	holeOp.GeoM.Scale(scale, scale)
	holeOp.GeoM.Translate(centerX-s.darknessRadius, centerY-s.darknessRadius)
	holeOp.Blend = ebiten.BlendDestinationOut
	overlay.DrawImage(mask, holeOp)

	screen.DrawImage(overlay, &ebiten.DrawImageOptions{})
}

func (s *FieldScene) drawPromptWithIcon(screen *ebiten.Image, icon *ebiten.Image, label string) {
	face := s.game.FontFace(15)

	textW, textH := text.Measure(label, face, 0)

	centerX := float64(gameWidth) / 2
	baseY := float64(gameHeight) - 32

	textX := centerX - textW/2
	textY := baseY - textH/2

	imgW := icon.Bounds().Dx()
	imgH := icon.Bounds().Dy()
	imgOp := &ebiten.DrawImageOptions{}
	imgOp.GeoM.Translate(
		centerX-float64(imgW)/2,
		baseY-float64(imgH)/2,
	)
	screen.DrawImage(icon, imgOp)

	op := &text.DrawOptions{}
	op.GeoM.Translate(textX, textY)
	op.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, label, face, op)
}

const (
	enemyBaseX = 240.0
	partyBaseY = 142.0

	partyLikeSpacingX = 20.0
	partyLikeSpacingY = 52.0

	enemyCenterRefX = enemyBaseX + float64(maxEnemies-1)/2.0*partyLikeSpacingX
	enemyCenterRefY = partyBaseY + float64(maxEnemies-1)/2.0*partyLikeSpacingY + spriteFrameH
)

func (s *BattleScene) enemyDrawRect(slot int) (x, y, w, h float64) {
	n := len(s.enemies)
	if slot < 0 || slot >= n || s.enemies[slot].Image == nil {
		return enemyCenterRefX, enemyCenterRefY, 0, 0
	}
	img := s.enemies[slot].Image
	w = float64(img.Bounds().Dx())
	h = float64(img.Bounds().Dy())

	offset := float64(slot) - float64(n-1)/2.0
	x = enemyCenterRefX + offset*partyLikeSpacingX
	groundY := enemyCenterRefY + offset*partyLikeSpacingY
	y = groundY - h
	return x, y, w, h
}

func (s *BattleScene) enemyCenter(slot int) (float64, float64) {
	x, y, w, h := s.enemyDrawRect(slot)
	return x + w/2, y + h/2
}

func (s *FieldScene) drawItemGetPopup(screen *ebiten.Image) {
	const (
		boxW, boxH  = 420.0, 130.0
		borderWidth = 3.0
	)

	bx := float64(gameWidth)/2 - boxW/2
	by := float64(gameHeight)/2 - boxH/2

	nameFace := s.game.FontFace(16)
	label := s.itemGetName
	if !s.itemGetPlainMessage {
		label += "を手に入れた！"
	}
	textW, textH := text.Measure(label, nameFace, 0)

	subFace := s.game.FontFace(13)
	var subW, subH float64
	if s.itemGetSubLabel != "" {
		subW, subH = text.Measure(s.itemGetSubLabel, subFace, 0)
	}

	ebitenutil.DrawRect(screen, bx, by, boxW, boxH, uiColorText)
	ebitenutil.DrawRect(screen, bx+borderWidth, by+borderWidth, boxW-borderWidth*2, boxH-borderWidth*2, uiColorPanelBg)

	blockH := textH
	if s.itemGetSubLabel != "" {
		blockH += subH + 6
	}
	topY := by + boxH/2 - blockH/2

	nameOp := &text.DrawOptions{}
	nameOp.GeoM.Translate(bx+boxW/2-textW/2, topY)
	nameOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, label, nameFace, nameOp)

	if s.itemGetSubLabel != "" {
		subOp := &text.DrawOptions{}
		subOp.GeoM.Translate(bx+boxW/2-subW/2, topY+textH+6)
		subOp.ColorScale.ScaleWithColor(uiColorText)
		text.Draw(screen, s.itemGetSubLabel, subFace, subOp)
	}
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

	choiceArrowGap := text.Advance("▶", face)
	for i, opt := range s.choiceOptions {
		baseX, baseY := bx+10, by+40+float64(i)*24
		if i == s.choiceIndex {
			arrowOp := &text.DrawOptions{}
			arrowOp.GeoM.Translate(baseX, baseY)
			arrowOp.ColorScale.ScaleWithColor(uiColorText)
			text.Draw(screen, "▶", face, arrowOp)
		}
		op := &text.DrawOptions{}
		op.GeoM.Translate(baseX+choiceArrowGap, baseY)
		op.ColorScale.ScaleWithColor(uiColorText)
		text.Draw(screen, opt, face, op)
	}
}
