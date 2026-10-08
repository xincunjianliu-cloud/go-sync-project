package main

import (
	"image"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func (s *FieldScene) Draw(screen *ebiten.Image) {
	if s == nil {
		return
	}

	if s.encounterEffectActive {
		s.drawEncounterEffect(screen)
		return
	}

	if s.isMsgActive && s.msgBackground != "" {
		if bg := s.game.GetBackgroundImage(s.msgBackground); bg != nil {
			screen.DrawImage(bg, nil)
		} else {
			screen.Fill(color.RGBA{0, 0, 0, 255})
		}
	} else {
		s.drawWorld(screen)
	}

	if s.isMsgActive {
		s.msg.DrawChara(screen, s.game)
	}

	if s.isMsgActive && s.msgIndex >= 0 && s.msgIndex < len(s.msgTexts) {
		s.msg.Draw(screen, s.msgTexts[s.msgIndex], s.game, 20)
	}

	if s.isMsgActive {
		drawMessageControlPanel(screen, s.game, s.autoMode, s.msgSkipHoldElapsed, endingSkipHoldSeconds)
	}

	s.drawMapNameBanner(screen)

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
		fillRect(screen, 0, 0,
			float64(gameWidth), float64(gameHeight),
			scaleAlpha(color.RGBA{0, 0, 0, 255}, s.cseFadeAlpha))
	}

	if s.isChoiceActive {
		camX, camY := s.cameraPosition()
		s.drawChoiceUI(screen, camX+s.screenShakeX, camY+s.screenShakeY)
	}

	if s.isItemGetActive {
		s.drawItemGetPopup(screen)
		if !s.itemGetAutoCloseOnly {
			drawBackButton(screen, s.game)
		}
	}

	if s.isLogActive {
		drawMessageLog(screen, s.game, s.msgLog, s.logScrollOffset, s.logCursorIndex)
		drawBackButton(screen, s.game)
	}

	showTouchPad := !s.isMsgActive && !s.isChoiceActive && !s.isCutscene && !s.isItemGetActive && !s.isLogActive && !s.wallFadeActive && !s.skillUpgradeTutorialActive
	if s.game.MobileMode {
		showActionButton := !s.isMsgActive && !s.isItemGetActive && !s.wallFadeActive && !s.skillUpgradeTutorialActive
		s.drawTouchControls(screen, showTouchPad, showActionButton)
	}
	if showTouchPad {
		drawHamburgerMenuButton(screen, s.game)
	}

	if s.skillUpgradeTutorialActive {
		s.drawSkillUpgradeTutorial(screen)
	}
}

func (s *FieldScene) drawWorld(screen *ebiten.Image) {
	screen.Fill(color.RGBA{30, 30, 30, 255})
	camX, camY := s.cameraPosition()
	camX += s.screenShakeX
	camY += s.screenShakeY

	// events/playerレイヤーの位置はTiled側のレイヤー順で決まる。マップ制作者が
	// レイヤーパネルで並べ替えるだけで、装飾タイルと設置物・プレイヤーの
	// 前後関係を変更できるようにするため、タイル描画ループの中で
	// これらのレイヤーに到達したタイミングで対応する内容を描画する。
	eventsLayerIdx, playerLayerIdx := -1, -1
	for i, layer := range s.tileMap.Layers {
		if layer.Type != "objectgroup" {
			continue
		}
		if eventsLayerIdx == -1 && isEventsLayer(layer) {
			eventsLayerIdx = i
		}
		if isPlayerLayer(layer) {
			playerLayerIdx = i
		}
	}

	eventsDrawn, playerDrawn := false, false
	for i, layer := range s.tileMap.Layers {
		if layer.Type == "tilelayer" && layer.drawn() {
			s.drawTileLayer(screen, layer, camX, camY)
		}
		if i == eventsLayerIdx {
			s.drawMapEvents(screen, camX, camY)
			eventsDrawn = true
		}
		if i == playerLayerIdx {
			s.drawEnemiesAndPlayer(screen, camX, camY)
			playerDrawn = true
		}
	}
	if !eventsDrawn {
		s.drawMapEvents(screen, camX, camY)
	}
	if !playerDrawn {
		s.drawEnemiesAndPlayer(screen, camX, camY)
	}

	s.drawDarkness(screen, camX, camY)
}

// drawEncounterEffect は雑魚敵エンカウント発生時、startEncounterEffectで
// 捕えたフィールドの静止画をカメラが回転しながらズームインしていくように
// 描画する。プレイヤーは画面中央に固定されているため、画面中心を軸に
// 回転・拡大するだけで「カメラが回り込みながら迫る」演出になる。
func (s *FieldScene) drawEncounterEffect(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0, 0, 0, 255})
	if s.encounterSnapshot == nil {
		return
	}

	t := s.encounterEffectTimer / encounterEffectDuration
	if t > 1 {
		t = 1
	}
	eased := t * t // 徐々に加速しながら迫ってくる感じにするための ease-in

	scale := 1.0 + eased*(encounterEffectZoomEnd-1.0)
	rotation := eased * encounterEffectSpin

	w := s.encounterSnapshot.Bounds().Dx()
	h := s.encounterSnapshot.Bounds().Dy()

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(w)/2, -float64(h)/2)
	op.GeoM.Rotate(rotation)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(gameWidth)/2, float64(gameHeight)/2)
	op.Filter = ebiten.FilterLinear
	screen.DrawImage(s.encounterSnapshot, op)

	if t > 0.6 {
		darken := (t - 0.6) / 0.4
		fillRect(screen, 0, 0, float64(gameWidth), float64(gameHeight), color.NRGBA{0, 0, 0, uint8(220 * darken)})
	}
}

// buildMapTileImages はマップで使われているタイルを、GIDごとにタイルセット
// 画像から切り出しておく。描画のたびにSubImageを作らずに済む。
func buildMapTileImages(tmap TiledMap) ([]*ebiten.Image, error) {
	// 置かれているタイルと、そのアニメーションのコマを全部切り出す。
	needed := map[int]bool{}
	maxGID := 0
	for _, layer := range tmap.Layers {
		for _, id := range layer.Data {
			gid := id &^ gidFlagMask
			if gid == 0 || needed[gid] {
				continue
			}
			needed[gid] = true
			maxGID = max(maxGID, gid)
			for _, f := range tmap.tileAnims[gid] {
				needed[f.GID] = true
				maxGID = max(maxGID, f.GID)
			}
		}
	}
	tiles := make([]*ebiten.Image, maxGID+1)
	sheets := map[string]*ebiten.Image{}
	for gid := range needed {
		ts, ok := findTileset(tmap.tilesets, gid)
		if !ok || ts.imagePath == "" {
			// どのタイルセットにも属さない番号(テストで検出する)は描かない。
			continue
		}
		sheet, ok := sheets[ts.imagePath]
		if !ok {
			img, err := loadTilesetImage(ts.imagePath)
			if err != nil {
				return nil, err
			}
			sheet = img
			sheets[ts.imagePath] = img
		}
		x0, y0, x1, y1 := ts.tileRect(gid - ts.firstGID)
		tiles[gid] = sheet.SubImage(image.Rect(x0, y0, x1, y1)).(*ebiten.Image)
	}
	return tiles, nil
}

func (s *FieldScene) drawTileLayer(screen *ebiten.Image, layer TiledLayer, camX, camY float64) {
	op := &ebiten.DrawImageOptions{}
	if a := layer.opacity(); a < 1 {
		op.ColorScale.ScaleAlpha(a)
	}
	leverOnly := isLeverOpenLayer(layer)
	for i, id := range layer.Data {
		if id == 0 {
			continue
		}
		if leverOnly && !s.leverTileShown(i) {
			continue
		}
		gid := id &^ gidFlagMask
		if frames, ok := s.tileMap.tileAnims[gid]; ok {
			gid = animFrameGID(frames, int64(s.tileAnimMs))
		}
		if gid >= len(s.tileImgs) || s.tileImgs[gid] == nil {
			continue
		}
		tile := s.tileImgs[gid]
		tw, th := float64(tile.Bounds().Dx()), float64(tile.Bounds().Dy())
		tx := float64((i % s.tileMap.Width) * s.tileMap.TileWidth)
		ty := float64((i / s.tileMap.Width) * s.tileMap.TileHeight)

		op.GeoM.Reset()
		if id&(gidFlipH|gidFlipV|gidFlipD) != 0 {
			// Tiledと同じ順(対角線→左右→上下)で、タイルの中心を軸に反転する。
			op.GeoM.Translate(-tw/2, -th/2)
			if id&gidFlipD != 0 {
				var swap ebiten.GeoM
				swap.SetElement(0, 0, 0)
				swap.SetElement(0, 1, 1)
				swap.SetElement(1, 0, 1)
				swap.SetElement(1, 1, 0)
				op.GeoM.Concat(swap)
			}
			if id&gidFlipH != 0 {
				op.GeoM.Scale(-1, 1)
			}
			if id&gidFlipV != 0 {
				op.GeoM.Scale(1, -1)
			}
			op.GeoM.Translate(tw/2, th/2)
		}
		// マップのマスより大きいタイルは、Tiledと同じくマスの左下にそろえる。
		op.GeoM.Translate(tx+camX, ty+camY+float64(s.tileMap.TileHeight)-th)
		screen.DrawImage(tile, op)
	}
}

func (s *FieldScene) drawMapEvents(screen *ebiten.Image, camX, camY float64) {
	s.drawChests(screen, camX, camY)
	s.drawLockedWalls(screen, camX, camY)
	s.drawLeverWalls(screen, camX, camY)
	s.drawLevers(screen, camX, camY)
	s.drawBlockSpots(screen, camX, camY)
	s.drawBlocks(screen, camX, camY)
	s.drawBlockDoors(screen, camX, camY)
}

func (s *FieldScene) drawEnemiesAndPlayer(screen *ebiten.Image, camX, camY float64) {
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
}

func (s *FieldScene) drawChests(screen *ebiten.Image, camX, camY float64) {
	const chestFrameW, chestFrameH = 40, 32

	for _, layer := range s.tileMap.Layers {
		if !isEventsLayer(layer) {
			continue
		}
		for _, obj := range layer.Objects {
			p := objProps(obj)
			if !isChestObj(p) {
				continue
			}

			frame := 0
			if s.game.OpenedChests[chestKey(s.currentMap, obj)] {
				frame = 1
			}

			chestImg := s.game.ChestImg
			if isKeyChestObj(p) && s.game.KeyChestImg != nil {
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

// lockedWallFrameCount/lockedWallTicksPerFrame は施錠された壁(レバー式を除く)の
// アニメーション設定。常に揺らめいて見えるよう、開錠前は無条件で再生し続ける。
const (
	lockedWallFrameCount    = 4
	lockedWallTicksPerFrame = 8
)

func (s *FieldScene) drawLockedWalls(screen *ebiten.Image, camX, camY float64) {
	for _, layer := range s.tileMap.Layers {
		if !isEventsLayer(layer) {
			continue
		}
		for _, obj := range layer.Objects {
			p := objProps(obj)
			if !isLockedWallObj(p) {
				continue
			}
			if s.wallIsOpen(obj) {
				continue
			}

			img := s.game.LockedWallImg
			if img == nil {
				continue
			}

			frameW := float64(img.Bounds().Dx()) / lockedWallFrameCount
			frameH := float64(img.Bounds().Dy())
			frame := (s.wallAnimTick / lockedWallTicksPerFrame) % lockedWallFrameCount
			if frameW <= 0 || frameH <= 0 {
				continue
			}
			sx := frame * int(frameW)
			srcRect := image.Rect(sx, 0, sx+int(frameW), int(frameH))

			op := &ebiten.DrawImageOptions{}
			if obj.Width > 0 && obj.Height > 0 {
				op.GeoM.Scale(obj.Width/frameW, obj.Height/frameH)
			}
			op.GeoM.Translate(obj.X+camX, obj.Y+camY)

			if s.wallFadeActive && chestKey(s.currentMap, obj) == s.wallFadeKey {
				op.ColorScale.ScaleAlpha(float32(s.wallFadeAlpha))
			}

			screen.DrawImage(img.SubImage(srcRect).(*ebiten.Image), op)
		}
	}
}

const leverWallTileSize = 32

// drawLeverWalls はレバー連動の壁を描画する。閉状態では何も描かず、マップの
// 壁タイルそのものの見た目に任せる。開いた時だけ、通行可能かどうかに応じた
// 32×32の絵を、引き伸ばさずオブジェクトの範囲いっぱいに敷き詰めて表示する。
func (s *FieldScene) drawLeverWalls(screen *ebiten.Image, camX, camY float64) {
	for _, layer := range s.tileMap.Layers {
		if !isEventsLayer(layer) {
			continue
		}
		for _, obj := range layer.Objects {
			p := objProps(obj)
			if !isLeverControlledWallObj(p) {
				continue
			}
			if !s.wallIsOpen(obj) {
				continue
			}
			// leveropenレイヤーのタイルを置いた壁は、そのタイルが見た目になる。
			if s.tileMap.leverWallsWithTiles[obj.ID] {
				continue
			}

			img := s.game.LeverWallOpenImg
			imgs := s.game.LeverWallOpenImgs
			if isLeverWallVisualOnly(p) {
				img = s.game.LeverWallOpenDecoImg
				imgs = s.game.LeverWallOpenDecoImgs
			}
			if key := leverWallImageKey(p); key != "" && imgs[key] != nil {
				img = imgs[key]
			}
			if img == nil {
				continue
			}

			srcRect := image.Rect(0, 0, leverWallTileSize, leverWallTileSize)
			tile := img.SubImage(srcRect).(*ebiten.Image)

			cols := int(obj.Width) / leverWallTileSize
			rows := int(obj.Height) / leverWallTileSize
			if cols < 1 {
				cols = 1
			}
			if rows < 1 {
				rows = 1
			}

			for row := 0; row < rows; row++ {
				for col := 0; col < cols; col++ {
					op := &ebiten.DrawImageOptions{}
					op.GeoM.Translate(obj.X+float64(col*leverWallTileSize)+camX, obj.Y+float64(row*leverWallTileSize)+camY)
					screen.DrawImage(tile, op)
				}
			}
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
		if !isEventsLayer(layer) {
			continue
		}
		for _, obj := range layer.Objects {
			p := objProps(obj)
			if !isLeverObj(p) {
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
	imgW := float64(img.Bounds().Dx())
	imgH := float64(img.Bounds().Dy())

	for _, layer := range s.tileMap.Layers {
		if !isEventsLayer(layer) {
			continue
		}
		for _, obj := range layer.Objects {
			p := objProps(obj)
			if !isBlockSpotObj(p) {
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
	imgW := float64(img.Bounds().Dx())
	imgH := float64(img.Bounds().Dy())

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
		if !isEventsLayer(layer) {
			continue
		}
		for _, obj := range layer.Objects {
			p := objProps(obj)
			if !isBlockDoorObj(p) {
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

// mapNameBannerAlpha はマップ名バナーの不透明度を返す。表示開始からしばらくは
// 不透明のまま静止させ、その後mapNameBannerFadeDuration秒かけて0まで
// フェードアウトさせる。
func (s *FieldScene) mapNameBannerAlpha() float64 {
	if !s.mapNameBannerActive {
		return 0
	}
	if s.mapNameBannerElapsed < mapNameBannerShowDuration {
		return 1
	}
	fadeT := (s.mapNameBannerElapsed - mapNameBannerShowDuration) / mapNameBannerFadeDuration
	if fadeT >= 1 {
		return 0
	}
	return 1 - fadeT
}

const (
	mapNameBannerMarginY  = 16.0
	mapNameBannerPadX     = 24.0
	mapNameBannerFontSize = 32.0
)

// drawMapNameBanner はバナー画像を反転せずそのまま、左端を画面左端に
// そろえて描画する。濃い端が画面左端に接し、そこから右へ伸びて丸く
// フェードしていく見た目になる。地名は左寄せで重ねる。
func (s *FieldScene) drawMapNameBanner(screen *ebiten.Image) {
	alpha := s.mapNameBannerAlpha()
	banner := s.game.MapNameBannerImg
	if alpha <= 0 || banner == nil {
		return
	}

	boxY := mapNameBannerMarginY

	imgOp := &ebiten.DrawImageOptions{}
	imgOp.GeoM.Translate(0, boxY)
	imgOp.ColorScale.ScaleAlpha(float32(alpha))
	screen.DrawImage(banner, imgOp)

	bannerH := float64(banner.Bounds().Dy())
	face := s.game.FontFace(mapNameBannerFontSize)
	_, textH := text.Measure(s.mapNameBannerText, face, 0)

	op := &text.DrawOptions{}
	op.GeoM.Translate(mapNameBannerPadX, boxY+(bannerH-textH)/2)
	op.ColorScale.ScaleWithColor(uiColorText)
	op.ColorScale.ScaleAlpha(float32(alpha))
	text.Draw(screen, s.mapNameBannerText, face, op)
}

func (s *FieldScene) drawPromptWithIcon(screen *ebiten.Image, icon *ebiten.Image, label string) {
	face := s.game.FontFace(20)

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
	enemyBaseX = 200.0
	partyBaseY = 142.0

	partyLikeSpacingY = 52.0

	enemyZigzagSpacingX = 30.0

	enemyCenterRefX = enemyBaseX
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

	var groundY float64
	if n == 1 && strings.HasPrefix(s.enemyType, "boss_") {
		// ボス単体の時は縦位置を一番下の味方の足元に合わせる。
		groundY = partyBaseY + float64(partySize-1)*partyLikeSpacingY + spriteFrameH
	} else {
		// Vertical position still follows the party's diagonal spacing reference.
		offsetY := float64(slot) - float64(n-1)/2.0
		groundY = enemyCenterRefY + offsetY*partyLikeSpacingY
	}
	y = groundY - h

	// Horizontal position alternates left/right to form a zigzag row.
	xOffset := enemyZigzagSpacingX
	if slot%2 == 1 {
		xOffset = -xOffset
	}
	x = enemyCenterRefX + xOffset

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

	label := s.itemGetName
	if !s.itemGetPlainMessage {
		label += "を手に入れた！"
	}
	textW, textH := text.Measure(label, s.game.FontFace(20), 0)

	var subW, subH float64
	if s.itemGetSubLabel != "" {
		subW, subH = text.Measure(s.itemGetSubLabel, s.game.FontFace(15), 0)
	}

	fillRect(screen, bx, by, boxW, boxH, uiColorText)
	fillRect(screen, bx+borderWidth, by+borderWidth, boxW-borderWidth*2, boxH-borderWidth*2, uiColorPanelBg)

	blockH := textH
	if s.itemGetSubLabel != "" {
		blockH += subH + 6
	}
	topY := by + boxH/2 - blockH/2

	labelOp := &text.DrawOptions{}
	labelOp.GeoM.Translate(bx+boxW/2-textW/2, topY)
	labelOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, label, s.game.FontFace(20), labelOp)

	if s.itemGetSubLabel != "" {
		subOp := &text.DrawOptions{}
		subOp.GeoM.Translate(bx+boxW/2-subW/2, topY+textH+6)
		subOp.ColorScale.ScaleWithColor(uiColorText)
		text.Draw(screen, s.itemGetSubLabel, s.game.FontFace(15), subOp)
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

	fillRect(screen, bx, by, boxW, boxH, uiColorPanelBg)

	face := s.game.FontFace(14)

	questionOp := &text.DrawOptions{}
	questionOp.GeoM.Translate(bx+10, by+10)
	questionOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, s.choiceQuestion, face, questionOp)

	choiceArrowGap := text.Advance("▶", face)
	for i, opt := range s.choiceOptions {
		baseX, baseY := bx+10, by+40+float64(i)*24
		if i == s.choiceIndex {
			arrowOp := &text.DrawOptions{}
			arrowOp.GeoM.Translate(baseX, baseY)
			arrowOp.ColorScale.ScaleWithColor(uiColorText)
			text.Draw(screen, "▶", face, arrowOp)
		}
		optOp := &text.DrawOptions{}
		optOp.GeoM.Translate(baseX+choiceArrowGap, baseY)
		optOp.ColorScale.ScaleWithColor(uiColorText)
		text.Draw(screen, opt, face, optOp)
	}
}
