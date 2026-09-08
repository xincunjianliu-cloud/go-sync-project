package main

// battle_draw_hud.go: バトルHUD描画（タイムライン・ステータスバー・ゲージ・コマンド/スキルメニュー）
import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func (s *BattleScene) drawTimeline(screen *ebiten.Image) {
	drawX, drawY := s.timelineDrawOrigin()
	goalX := s.goalScreenX()
	centerY := trackCenterY()

	introX := 0.0
	if s.introActive && s.introPhase == 0 {
		introX = -s.introOffsetX
	}

	introVertY := 0.0
	if s.introActive && s.introPhase == 1 {
		introVertY = s.introVertOffsetY
	}

	if s.game.TimelineBarImg != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(drawX+introX+s.shakeX, drawY+s.shakeY)
		screen.DrawImage(s.game.TimelineBarImg, op)
	} else {
		ebitenutil.DrawRect(screen, trackX+introX, trackY, trackW, trackH, color.RGBA{35, 40, 55, 255})
	}

	if (!s.introActive || s.introPhase >= 1) && s.game.TimelineBarVertImg != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(goalX-timelineImgLayout.VertLineX+s.shakeX, introVertY+s.shakeY)
		screen.DrawImage(s.game.TimelineBarVertImg, op)
	}

	if (!s.introActive || s.introPhase >= 2) && s.game.GoalImg != nil {
		gw := float64(s.game.GoalImg.Bounds().Dx())
		gh := float64(s.game.GoalImg.Bounds().Dy())
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(goalX-gw/2+s.shakeX, centerY-gh/2+s.shakeY)
		screen.DrawImage(s.game.GoalImg, op)
	}

	if s.introActive && s.introPhase < 3 {
		return
	}

	order := make([]int, partySize+1)
	for i := range order {
		order[i] = i
	}
	for i := 0; i < len(order); i++ {
		for j := i + 1; j < len(order); j++ {
			if s.atbGauge[order[j]] < s.atbGauge[order[i]] {
				order[i], order[j] = order[j], order[i]
			}
		}
	}

	for _, actor := range order {
		if actor < partySize && s.game.PlayerHP[actor] <= 0 {
			continue
		}
		if actor == enemyID && s.enemyHP <= 0 {
			continue
		}
		isActive := s.waitingActor == actor && (s.battlePhase == phasePlayerMenu || s.battlePhase == phaseSkillMenu)
		// ★修正：位置の保持とサイズの保持を分ける
		// 位置保持：returnDelayTimer > 0（戻り始めるまで）
		isHoldingPosition := actor < partySize && s.returnDelayTimer[actor] > 0
		// サイズ保持：readySlideX < 0（スタート地点に到達したら通常サイズに戻す）
		isHoldingSize := actor < partySize && s.readySlideX[actor] < 0
		isMyselfWaiting := (actor < partySize && s.waitStance[actor]) || (isActive && s.commandIndex == 2)
		currentIconSize := float64(iconSize)
		if (isActive || isHoldingSize) && !isMyselfWaiting {
			currentIconSize = float64(iconSize) * 1.5
		}

		x := s.actorPosX(actor)
		ready := s.isActorReady(actor)
		if ready || ((isActive || isHoldingPosition) && !isMyselfWaiting) {
			x = goalX - (currentIconSize / 2)
		}

		y := centerY - currentIconSize/2

		if isMyselfWaiting {
			downCount := -1
			for orderIdx, actorIdx := range s.waitOrder {
				if actorIdx == actor {
					downCount = orderIdx
					break
				}
			}
			if downCount == -1 {
				downCount = len(s.waitOrder)
			}
			y += 75.0 + float64(downCount)*55.0
		}

		var iconImg *ebiten.Image
		if actor < partySize {
			iconImg = s.game.TimelineIcons[actor]
		} else {
			iconImg = s.game.GetEnemyImage(s.enemyName)
		}

		if iconImg != nil {
			iw := float64(iconImg.Bounds().Dx())
			ih := float64(iconImg.Bounds().Dy())

			imgScale := 1.0
			if (ready || isActive) && !isMyselfWaiting {
				imgScale = 1.7
			}

			scaledW := iw * imgScale
			scaledH := ih * imgScale

			offsetX := (currentIconSize - scaledW) / 2
			offsetY := (currentIconSize - scaledH) / 2

			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(imgScale, imgScale)
			op.GeoM.Translate(x+offsetX+s.shakeX, y+offsetY+s.shakeY)

			screen.DrawImage(iconImg, op)
		}
	}
}

func (s *BattleScene) drawStatusBar(screen *ebiten.Image) {

	winY := statusBarY

	alpha := 1.0
	if s.introActive {
		if s.introPhase < 3 {
			alpha = 0.0
		} else {
			ratio := 1.0 - s.introCharOffsetX/introStartOffsetX
			const fadeStart = 0.7
			if ratio < fadeStart {
				alpha = 0.0
			} else {
				alpha = (ratio - fadeStart) / (1.0 - fadeStart)
				if alpha > 1.0 {
					alpha = 1.0
				}
			}
		}
	}
	for i := 0; i < partySize; i++ {
		xPos := statusStartX + float64(i)*(statusBlockW+statusBlockGap)

		sx := xPos + s.shakeX
		sy := winY + s.shakeY
		barX := sx - statusBarSlant

		isMyTurn := s.waitingActor == i &&
			(s.battlePhase == phasePlayerMenu || s.battlePhase == phaseSkillMenu ||
				s.battlePhase == phaseTargetSelect || s.battlePhase == phaseHealSelect)
		s.drawPartyName(screen, i, xPos, winY, alpha, isMyTurn)

		maxHP := s.game.PlayerMaxHP[i]
		hpRatio := 0.0
		if maxHP > 0 {
			hpRatio = float64(s.game.PlayerHP[i]) / float64(maxHP)
		}
		if hpRatio > 1.0 {
			hpRatio = 1.0
		}

		drawStatusValue(screen, sx, sy+statusHPTextY, s.game.PlayerHP[i], maxHP,
			s.game.FontFace(17.5), s.game.FontFace(14), alpha)

		drawSlantedStatusBar(screen, barX, sy+statusHPBarY, statusBlockW, statusBarH, statusBarSlant, hpRatio,
			scaleAlpha(color.RGBA{75, 171, 120, 255}, alpha),
			scaleAlpha(color.RGBA{20, 50, 30, 255}, alpha),
			scaleAlpha(color.RGBA{41, 94, 66, 255}, alpha))

		maxMP := s.game.PlayerMaxMP[i]
		mpRatio := 0.0
		if maxMP > 0 {
			mpRatio = float64(s.game.PlayerMP[i]) / float64(maxMP)
		}
		if mpRatio > 1.0 {
			mpRatio = 1.0
		}

		drawStatusValue(screen, sx, sy+statusMPTextY, s.game.PlayerMP[i], maxMP,
			s.game.FontFace(17.5), s.game.FontFace(14), alpha)

		drawSlantedStatusBar(screen, barX, sy+statusMPBarY, statusBlockW, statusBarH, statusBarSlant, mpRatio,
			scaleAlpha(color.RGBA{75, 105, 171, 255}, alpha),
			scaleAlpha(color.RGBA{20, 30, 55, 255}, alpha),
			scaleAlpha(color.RGBA{41, 58, 94, 255}, alpha))

		// ── デバフアイコン(簡易：文字表記)を名前欄の下に表示 ──
		if len(s.PlayerDebuffs[i]) > 0 {
			debuffOp := &text.DrawOptions{}
			debuffOp.GeoM.Translate(sx, sy-12)
			debuffOp.ColorScale.ScaleWithColor(color.RGBA{255, 120, 120, uint8(255 * alpha)})
			text.Draw(screen, fmt.Sprintf("弱体x%d", len(s.PlayerDebuffs[i])), s.game.FontFace(11), debuffOp)
		}
	}
}

func (s *BattleScene) drawGaugeTriangle(screen *ebiten.Image) {
	// ★修正：shakeX/shakeYはSin波やランダム値による小数値なので、
	// そのまま使うと塗りつぶし側だけmath.Floorで整数に丸められ、
	// 枠画像側は小数位置のまま描画されるため、攻撃(揺れ)のたびに
	// 塗りと枠が斜め方向に最大1pxずれて「足りない」ように見えていた。
	// 揺れも含めた位置を先に整数ピクセルへ丸めることで、塗りと枠を
	// 常に同じピクセルグリッド上に揃える。
	x := math.Round(gaugeTriX + s.shakeX)
	y := math.Round(gaugeTriY + s.shakeY)
	pad := 1.0
	w := gaugeTriW
	h := gaugeTriH
	if s.game.GaugeImg != nil {
		w = float64(s.game.GaugeImg.Bounds().Dx())
		h = float64(s.game.GaugeImg.Bounds().Dy())
		// ★変更：新しいゲージ画像は縁の太さが2pxのドット絵なので、
		// 塗りつぶしをその内側にぴったり収めるためpadを画像の縁幅に合わせる。
		pad = gaugeImgBorder
	}
	ix := x + pad
	iy := y + pad
	iw := w - pad*2
	ih := h - pad*2

	if s.game.GaugeImg != nil {
		// ★変更：1ポイント=2px上昇だが、8ポイント(1段)ごとに区切り線2pxぶん
		// 追加でジャンプさせることで、区切り線の位置とちょうど噛み合わせる。
		// 例）8ポイント消化時点 = 16px(斜め上昇分) + 2px(区切り線をまたぐ分) = 18px
		completedStages := s.gaugePoint / gaugePointsPerStage
		remainder := s.gaugePoint % gaugePointsPerStage
		filledHeight := float64(completedStages)*(float64(gaugePointsPerStage)*gaugeFillPerPoint+gaugeDividerHeight) +
			float64(remainder)*gaugeFillPerPoint
		if filledHeight > ih {
			filledHeight = ih
		}
		if filledHeight < 0 {
			filledHeight = 0
		}

		if filledHeight > 0 {
			// ★修正：以前はiw*(1-(fillTopY-iy)/ih)のように浮動小数点の
			// 割り算・掛け算を経由していたため、本来ちょうど整数になる
			// はずの値が89.999999998のようにごくわずかにずれ、
			// math.Floorで切り捨てると本来より1px内側に描画されていた。
			// ここではfillTopY = iy+ih-filledHeight という関係から
			// ih-(fillTopY-iy) = filledHeight であることを使い、
			// 割り算を最後の1回だけにした整数演算に置き換えることで
			// 誤差なくぴったり境界を計算する。
			ixI := int(math.Round(ix))
			iyI := int(math.Round(iy))
			iwI := int(math.Round(iw))
			ihI := int(math.Round(ih))
			filledHeightI := int(math.Round(filledHeight))
			if filledHeightI > ihI {
				filledHeightI = ihI
			}

			fillTopYI := iyI + ihI - filledHeightI
			bottomYI := iyI + ihI
			// ★修正：整数演算にしても右端が1px足りなかった。原因は計算誤差
			// ではなく、vector.FillPathが指定した境界座標そのものは
			// 塗らない(半開区間で塗る)ため。境界ちょうどの列を確実に
			// 塗るには、右端の座標を+1して指定する必要がある。
			rightAtFillTopI := ixI + (iwI*filledHeightI)/ihI + 1

			fillTopY := float64(fillTopYI)
			bottomY := float64(bottomYI)
			rightAtFillTop := float64(rightAtFillTopI)

			var path vector.Path
			path.MoveTo(float32(ix), float32(bottomY))
			path.LineTo(float32(ix), float32(fillTopY))
			path.LineTo(float32(rightAtFillTop), float32(fillTopY))
			path.Close()

			// ★変更：ゲージ画像がドット絵(境界くっきり)なので、
			// アンチエイリアスをOFFにしてピクセルぴったりの境界にする。
			// ONのままだと境界が滲んで「1〜2pxずれて見える」原因になっていた。
			fillOpts := &vector.DrawPathOptions{AntiAlias: false}
			fillOpts.ColorScale.ScaleWithColor(color.RGBA{102, 204, 204, 255})
			vector.FillPath(screen, &path, nil, fillOpts)
		}

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(x, y)
		screen.DrawImage(s.game.GaugeImg, op)
		return
	}

	rows := gaugeMaxStage
	rowH := ih / float64(rows)

	filledColor := color.RGBA{255, 200, 60, 255}
	emptyColor := color.RGBA{45, 42, 38, 255}
	lineColor := color.RGBA{20, 20, 20, 255}

	var stageRatio float64
	if s.gaugeStage < gaugeMaxStage-1 {
		need := gaugeStageThresholds[s.gaugeStage]
		if need > 0 {
			stageRatio = float64(s.gaugePoint) / float64(need)
		}
	} else {
		stageRatio = float64(s.gaugePoint) / float64(gaugePointCap)
	}
	if stageRatio > 1.0 {
		stageRatio = 1.0
	}
	if stageRatio < 0.0 {
		stageRatio = 0.0
	}

	for row := 0; row < rows; row++ {
		topY := iy + ih - float64(row+1)*rowH
		botY := iy + ih - float64(row)*rowH

		leftX := ix
		rightAtTop := ix + iw*(1.0-(topY-iy)/ih)
		rightAtBot := ix + iw*(1.0-(botY-iy)/ih)

		isFilled := row < s.gaugeStage
		isCurrent := row == s.gaugeStage

		var path vector.Path
		path.MoveTo(float32(leftX), float32(botY))
		path.LineTo(float32(rightAtBot), float32(botY))
		path.LineTo(float32(rightAtTop), float32(topY))
		path.LineTo(float32(leftX), float32(topY))
		path.Close()

		c := emptyColor
		if isFilled {
			c = filledColor
		} else if isCurrent && stageRatio > 0 {
			c = color.RGBA{
				R: uint8(float64(emptyColor.R) + float64(filledColor.R-emptyColor.R)*stageRatio),
				G: uint8(float64(emptyColor.G) + float64(filledColor.G-emptyColor.G)*stageRatio),
				B: uint8(float64(emptyColor.B) + float64(filledColor.B-emptyColor.B)*stageRatio),
				A: 255,
			}
		}

		drawOpts := &vector.DrawPathOptions{AntiAlias: true}
		drawOpts.ColorScale.ScaleWithColor(c)
		vector.FillPath(screen, &path, nil, drawOpts)

		if row < rows {
			var linePath vector.Path
			linePath.MoveTo(float32(leftX), float32(topY))
			linePath.LineTo(float32(rightAtTop), float32(topY))
			strokeOpts := &vector.StrokeOptions{Width: 1}
			lineDrawOpts := &vector.DrawPathOptions{AntiAlias: true}
			lineDrawOpts.ColorScale.ScaleWithColor(lineColor)
			vector.StrokePath(screen, &linePath, strokeOpts, lineDrawOpts)
		}
	}

	var outline vector.Path
	outline.MoveTo(float32(x), float32(y+h))
	outline.LineTo(float32(x+w), float32(y+h))
	outline.LineTo(float32(x+w), float32(y))
	outline.Close()
	outlineStrokeOpts := &vector.StrokeOptions{Width: 1.5}
	outlineDrawOpts := &vector.DrawPathOptions{AntiAlias: true}
	outlineDrawOpts.ColorScale.ScaleWithColor(lineColor)
	vector.StrokePath(screen, &outline, outlineStrokeOpts, outlineDrawOpts)

	labelOp := &text.DrawOptions{}
	labelOp.GeoM.Translate(x+w+8, y+h-14)
	labelOp.ColorScale.ScaleWithColor(uiColorText)
	if s.gaugeStage >= gaugeMaxStage-1 {
		text.Draw(screen, fmt.Sprintf("MAX %d/%d", s.gaugePoint, gaugePointCap), s.game.FontFace(13), labelOp)
	} else {
		text.Draw(screen, fmt.Sprintf("Lv.%d", s.gaugeStage+1), s.game.FontFace(15), labelOp)
	}
}

func (s *BattleScene) drawCommandMenu(screen *ebiten.Image) {
	centerX := 860.0
	centerY := 440.0
	spacing := 48.0

	positions := [4][2]float64{
		{centerX, centerY - spacing},
		{centerX - spacing, centerY},
		{centerX + spacing, centerY},
		{centerX, centerY + spacing},
	}

	for i, pos := range positions {
		icon := s.game.CommandIcons[i]
		if icon == nil {
			continue
		}

		iw := icon.Bounds().Dx()
		ih := icon.Bounds().Dy()

		op := &ebiten.DrawImageOptions{}

		scale := 1.0
		if s.commandIndex == i {
			scale = 1.3
		}
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(pos[0]-float64(iw)*scale/2, pos[1]-float64(ih)*scale/2)

		if s.commandIndex != i {
			op.ColorScale.Scale(0.6, 0.6, 0.6, 1.0)
		}

		screen.DrawImage(icon, op)
	}
}

// drawSkillSubMenu：主人公はHeroSkills(4項目)、他キャラは旧仕様(3項目)の説明・MP消費を表示
// ★変更：コマンドボタンの上に即時表示（アニメーションなし）。行間・横位置は調整用変数で管理。
func (s *BattleScene) drawSkillSubMenu(screen *ebiten.Image) {
	// コマンドメニューの中心（drawCommandMenuのcenterX, centerYと合わせる）
	cmdCenterX := 860.0
	cmdCenterY := 440.0

	windowW := 140.0
	windowH := 72.0
	if s.game.SkillPanelImg != nil {
		windowW = float64(s.game.SkillPanelImg.Bounds().Dx())
		windowH = float64(s.game.SkillPanelImg.Bounds().Dy())
	}

	// ★調整用：ウィンドウ自体の位置（コマンドボタン中心に被せる）
	windowX := cmdCenterX - windowW/2
	windowY := cmdCenterY - windowH/2

	if s.game.SkillPanelImg != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(windowX, windowY)
		screen.DrawImage(s.game.SkillPanelImg, op)
	} else {
		ebitenutil.DrawRect(screen, windowX, windowY, windowW, windowH, color.RGBA{10, 10, 20, 240})
	}

	// ★調整用：スキル項目の描画パラメータ
	const (
		labelOffsetX = 15.0 // ウィンドウ左端からラベルまでの距離
		labelOffsetY = 20.0 // ウィンドウ上端から1行目までの距離
		rowHeight    = 35.0 // 行間（大きくすると行が開く）
		mpOffsetX    = 20.0 // ウィンドウ右端からMP表示までの距離
	)

	p := s.waitingActor
	skills := s.game.CharacterSkills(p)

	var labels []string
	var costTexts []string
	var disabledList []bool

	for i, sk := range skills {
		curLv := s.game.PlayerSkillLv[p][i]
		if curLv < 1 {
			curLv = 1
		}
		if curLv > len(sk.Levels) {
			curLv = len(sk.Levels)
		}

		// 表示Lv：全行ともカーソル位置(skillLevelCursors)をそのまま表示する
		lv := s.skillLevelCursors[p][i]
		if lv < 1 {
			lv = 1
		}
		if lv > curLv {
			lv = curLv
		}

		data := sk.Levels[lv-1]
		labels = append(labels, fmt.Sprintf("%d:%s Lv%d", i+1, sk.Name, lv))
		costTexts = append(costTexts, fmt.Sprintf("MP%d", s.effectiveMPCost(data.MPCost)))
		disabledList = append(disabledList, s.game.PlayerMP[p] < s.effectiveMPCost(data.MPCost))
	}

	for i, label := range labels {
		prefix := "  "
		labelCol := uiColorText
		mpCol := uiColorText
		insufficient := disabledList[i]

		if insufficient {
			mpCol = uiColorDanger // MPの数値だけ、不足していれば赤
		}
		if i == s.skillIndex {
			prefix = "▶"
			labelCol = uiColorSelect // 選択中は常にこの色（MP不足でも変えない）
		}

		op := &text.DrawOptions{}
		op.GeoM.Translate(windowX+labelOffsetX, windowY+labelOffsetY+float64(i)*rowHeight)
		op.ColorScale.ScaleWithColor(labelCol)
		text.Draw(screen, prefix+label, s.game.FontFace(15), op)

		mpOp := &text.DrawOptions{}
		mpOp.GeoM.Translate(windowX+windowW-mpOffsetX, windowY+labelOffsetY+float64(i)*rowHeight)
		mpOp.PrimaryAlign = text.AlignEnd
		mpOp.ColorScale.ScaleWithColor(mpCol)
		text.Draw(screen, costTexts[i], s.game.FontFace(15), mpOp)
	}
}

func (s *BattleScene) drawLogWindowBackground(screen *ebiten.Image) {
	winX := 0.0
	winW := float64(gameWidth)
	winY := logPanelY
	winH := 28.0
	winImg := ebiten.NewImage(int(winW), int(winH))
	winImg.Fill(color.RGBA{0, 0, 128, 200})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(winX, winY)
	screen.DrawImage(winImg, op)
	borderColor := color.RGBA{200, 200, 255, 255}
	ebitenutil.DrawRect(screen, winX, winY, winW, 1, borderColor)
	ebitenutil.DrawRect(screen, winX, winY+winH-1, winW, 1, borderColor)
}

func (s *BattleScene) drawBattleMessage(screen *ebiten.Image) {
	if s.battleLog == "" {
		return
	}
	op := &text.DrawOptions{}
	op.PrimaryAlign = text.AlignCenter
	op.GeoM.Translate(float64(gameWidth)/2, logPanelY+6)
	op.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, s.battleLog, s.game.FontFace(15), op)
}

func (s *BattleScene) drawDirectMessages(screen *ebiten.Image) {
	if s.battlePhase == phaseBattleEnd && !s.isWon {
		winX, winY, winW, winH := 380.0, 235.0, 200.0, 70.0
		ebitenutil.DrawRect(screen, winX, winY, winW, winH, color.RGBA{40, 10, 10, 220})
		ebitenutil.DrawRect(screen, winX, winY, winW, 1, uiColorDanger)
		ebitenutil.DrawRect(screen, winX, winY+winH, winW, 1, uiColorDanger)
		ebitenutil.DrawRect(screen, winX, winY, 1, winH, uiColorDanger)
		ebitenutil.DrawRect(screen, winX+winW, winY, 1, winH, uiColorDanger)

		retryText := "  Retry Game"
		if s.gameOverIdx == 0 {
			retryText = "> Retry Game"
		}
		op1 := &text.DrawOptions{}
		op1.GeoM.Translate(winX+24, winY+18)
		op1.ColorScale.ScaleWithColor(uiColorText)
		text.Draw(screen, retryText, s.game.FontFace(15), op1)

		titleText := "  Title Screen"
		if s.gameOverIdx == 1 {
			titleText = "> Title Screen"
		}
		op2 := &text.DrawOptions{}
		op2.GeoM.Translate(winX+24, winY+42)
		if s.gameOverIdx == 1 {
			op2.ColorScale.ScaleWithColor(uiColorDanger)
		} else {
			op2.ColorScale.ScaleWithColor(uiColorText)
		}
		text.Draw(screen, titleText, s.game.FontFace(15), op2)
	}
}
