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
	battleTutorialKindNone = iota
	battleTutorialKindBasics
	battleTutorialKindGauge
)

// nextBattleTutorialKind decides which one-time battle tutorial (if any)
// should play for the battle about to start. Only one plays per battle: the
// basics tutorial takes priority since the gauge tutorial's condition
// (boss 1 defeated) can only become true after the player has already
// fought at least once.
func nextBattleTutorialKind(game *Game) int {
	switch {
	case !game.SeenBattleTutorial:
		return battleTutorialKindBasics
	case game.BossDefeatedFlags[0] && !game.SeenGaugeTutorial:
		return battleTutorialKindGauge
	default:
		return battleTutorialKindNone
	}
}

// battleTutorialPageIntro/Status/MP/HP/Timeline/Commands/Conclusion are the
// basics tutorial's page indices, in the order they're shown.
const (
	battleTutorialPageIntro = iota
	battleTutorialPageStatus
	battleTutorialPageMP
	battleTutorialPageHP
	battleTutorialPageTimeline
	battleTutorialPageCommands
	battleTutorialPageConclusion
)

// battleTutorialGaugePageIntro/WhatIsIt/FillRule/AttackBoost/RewindUnlock/
// RewindEffect/RewindActionBoost/Conclusion are the gauge tutorial's page
// indices, in the order they're shown.
const (
	battleTutorialGaugePageIntro = iota
	battleTutorialGaugePageWhatIsIt
	battleTutorialGaugePageFillRule
	battleTutorialGaugePageAttackBoost
	battleTutorialGaugePageRewindUnlock
	battleTutorialGaugePageRewindEffect
	battleTutorialGaugePageRewindActionBoost
	battleTutorialGaugePageConclusion
)

func battleTutorialPageCountFor(kind int) int {
	switch kind {
	case battleTutorialKindBasics:
		return battleTutorialPageConclusion + 1
	case battleTutorialKindGauge:
		return battleTutorialGaugePageConclusion + 1
	default:
		return 0
	}
}

func battleTutorialHeadline(kind int) string {
	if kind == battleTutorialKindGauge {
		return "バトルの基本2"
	}
	return "バトルの基本"
}

// battleTutorialBody returns the page's dialogue text, drawn as a single
// text block below the title (or beside it on the intro page).
func battleTutorialBody(kind, page int) string {
	if kind == battleTutorialKindGauge {
		switch page {
		case battleTutorialGaugePageIntro:
			return "新しいバトル要素が解禁されました。"
		case battleTutorialGaugePageWhatIsIt:
			return "それは、このゲージです。"
		case battleTutorialGaugePageFillRule:
			return "行動するたびにこのゲージがたまり、\n" +
				"最大でレベル5まで上がります。"
		case battleTutorialGaugePageAttackBoost:
			return "ゲージのレベルが上がるごとに、\n" +
				"パーティ全体の攻撃力も上昇します。"
		case battleTutorialGaugePageRewindUnlock:
			return "ゲージを最大までためると、\n" +
				"時間を巻き戻せるようになります。"
		case battleTutorialGaugePageRewindEffect:
			return "巻き戻すとゲージは0になりますが、\n" +
				"敵の直前の攻撃をなかったことにできます。"
		case battleTutorialGaugePageRewindActionBoost:
			return "さらに、時間の歪みの影響で、\n" +
				"一定時間パーティの行動回数が2倍になります。"
		default:
			return "ここぞという場面で、\n" +
				"ぜひ使ってみてください。"
		}
	}

	switch page {
	case battleTutorialPageIntro:
		return "ここではバトルのルールを説明します。"
	case battleTutorialPageHP:
		return "操作キャラクター全員のHPが0になると\n" +
			"ゲームオーバーになります。\n" +
			"アイテムやスキルを使って戦いましょう。"
	case battleTutorialPageStatus:
		return "ここは操作キャラクターの状態を表示する場所です。\n" +
			"緑のゲージはHPで、0になるとそのキャラクターは\n" +
			"戦闘不能になります。"
	case battleTutorialPageMP:
		return "下の青いゲージはMPです。\n" +
			"スキルを使うときに消費します。"
	case battleTutorialPageCommands:
		return "操作キャラクターが行動できるようになったら、\n" +
			"ここでコマンドを選びます。\n\n" +
			"たたかう：ふつうの攻撃\n" +
			"スキル：MPを消費して特殊な行動\n" +
			"たいき：4人全員が待機すると大ダメージ\n" +
			"にげる：戦闘から逃げる\n" +
			"アイテム：回復アイテムなどを使う"
	case battleTutorialPageTimeline:
		return "ここはタイムラインです。\n" +
			"操作キャラクターのアイコンが一番右に\n" +
			"着くと、行動できるようになります。"
	default:
		return "バトルの基本は以上です。\n" +
			"あとは実際に戦って慣れていきましょう。がんばって！"
	}
}

// battleTutorialBox is the rectangular HUD region a tutorial page
// highlights, in the same coordinate space as the real HUD layout helpers
// (partyNamePosition, commandIconPositions, goalScreenX/trackCenterY). The
// tutorial's dark overlay is punched through here so the element underneath
// shows at full brightness, framed with a white border.
type battleTutorialBox struct {
	x, y, w, h float64
}

// gaugeHighlightBox returns the highlight box around the gauge UI element,
// independent of tutorial page. It's used both to draw the box itself (only
// on the pages battleTutorialBox calls for) and to anchor the gauge
// tutorial's body text underneath the gauge on every page, including ones
// with no visible box.
func (s *BattleScene) gaugeHighlightBox() *battleTutorialBox {
	img := s.game.GaugeImg
	if img == nil {
		return nil
	}
	w := float64(img.Bounds().Dx())
	h := float64(img.Bounds().Dy())
	const pad = 14.0
	return &battleTutorialBox{gaugeTriX - pad, gaugeTriY - pad, w + pad*2, h + pad*2}
}

// battleTutorialBox returns the highlight box for the given tutorial
// kind/page, or nil if that page has nothing to highlight.
func (s *BattleScene) battleTutorialBox(kind, page int) *battleTutorialBox {
	if kind == battleTutorialKindGauge {
		if page == battleTutorialGaugePageIntro {
			return nil
		}
		return s.gaugeHighlightBox()
	}

	switch page {
	case battleTutorialPageHP, battleTutorialPageStatus, battleTutorialPageMP:
		sx, sy := partyNamePosition(0)
		const padTop, padSide, padBottom = 14.0, 18.0, 12.0
		top := sy - padTop
		bottom := sy + statusMPBarY + statusBarH + padBottom
		return &battleTutorialBox{sx - padSide, top, statusBlockW + padSide*2, bottom - top}

	case battleTutorialPageCommands:
		positions := commandIconPositions()
		minX, maxX := positions[0][0], positions[0][0]
		minY, maxY := positions[0][1], positions[0][1]
		for _, p := range positions[1:] {
			minX = math.Min(minX, p[0])
			maxX = math.Max(maxX, p[0])
			minY = math.Min(minY, p[1])
			maxY = math.Max(maxY, p[1])
		}
		const pad = 40.0
		return &battleTutorialBox{minX - pad, minY - pad, (maxX - minX) + pad*2, (maxY - minY) + pad*2}

	case battleTutorialPageTimeline:
		centerY := trackCenterY()
		const edgeMargin = 6.0
		const vertPad = 56.0
		top := centerY - vertPad
		if top < edgeMargin {
			top = edgeMargin
		}
		left := edgeMargin
		right := gameWidth - edgeMargin
		return &battleTutorialBox{left, top, right - left, centerY + vertPad - top}

	default:
		return nil
	}
}

// dimImageExceptBox darkens dst (which already holds a fully-rendered scene)
// using the menu background image as a tint source, so it reads as UI chrome
// rather than a flat black scrim, then punches a hard-edged rectangular hole
// at box so whatever's underneath there shows through at full brightness.
// overlay is a caller-owned scratch image (reused across frames to avoid
// reallocating), and dst/overlay must both be gameWidth x gameHeight. The
// white border framing the hole is drawn separately, directly on screen
// (see drawBattleTutorial/drawSkillUpgradeTutorial), so its thickness stays
// a fixed, crisp pixel width instead of being scaled down along with the
// rest of dst by the inset transform.
//
// A page with nothing to highlight (box == nil, e.g. each tutorial's intro
// message) still dims the whole scene uniformly, just without punching any
// hole through it, so the white body text reads clearly against it instead
// of whatever full-brightness content happens to sit underneath. Shared by
// the battle tutorial and the field scene's skill-upgrade tutorial so both
// dim their scene identically.
func dimImageExceptBox(dst *ebiten.Image, overlay *ebiten.Image, menuBg *ebiten.Image, box *battleTutorialBox) {
	overlay.Clear()

	if menuBg != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(
			float64(gameWidth)/float64(menuBg.Bounds().Dx()),
			float64(gameHeight)/float64(menuBg.Bounds().Dy()),
		)
		// MenuBgImg has no alpha channel of its own, so without scaling it
		// down here it would paint over the scene opaquely instead of
		// tinting it - the dimmed area needs to stay a bit visible.
		op.ColorScale.ScaleAlpha(0.72)
		overlay.DrawImage(menuBg, op)
	} else {
		overlay.Fill(color.RGBA{0, 0, 0, 180})
	}

	if box != nil {
		var path vector.Path
		path.MoveTo(float32(box.x), float32(box.y))
		path.LineTo(float32(box.x+box.w), float32(box.y))
		path.LineTo(float32(box.x+box.w), float32(box.y+box.h))
		path.LineTo(float32(box.x), float32(box.y+box.h))
		path.Close()
		vector.FillPath(overlay, &path, &vector.FillOptions{}, &vector.DrawPathOptions{
			Blend: ebiten.BlendDestinationOut,
		})
	}

	dst.DrawImage(overlay, &ebiten.DrawImageOptions{})
}

func (s *BattleScene) dimSceneExceptBox(dst *ebiten.Image, box *battleTutorialBox) {
	if s.tutorialOverlayImg == nil {
		s.tutorialOverlayImg = ebiten.NewImage(gameWidth, gameHeight)
	}
	dimImageExceptBox(dst, s.tutorialOverlayImg, s.game.MenuBgImg, box)
}

// tutorialInsetScale/X/Y place the dimmed battle frame as a smaller inset
// centered on the screen (extra room above for the title) instead of filling
// it edge to edge, so the menu background frames it on every side like the
// reference mockups.
const (
	tutorialInsetScale = 0.78
	tutorialInsetY     = 100.0

	// tutorialBoxBorderWidth is the highlight box's outline thickness, drawn
	// in screen space (see drawBattleTutorial) so every page's box gets the
	// exact same crisp width regardless of the box's own size or position.
	tutorialBoxBorderWidth = 2.0
)

func tutorialInsetX() float64 {
	return (gameWidth - gameWidth*tutorialInsetScale) / 2
}

// toScreen converts box from the unscaled scene's coordinate space into
// screen space, using the same inset transform the scene itself is drawn
// with, so UI drawn directly on screen (the border, the body text) can be
// placed relative to it.
func (box *battleTutorialBox) toScreen() (x, y, w, h float64) {
	return box.x*tutorialInsetScale + tutorialInsetX(),
		box.y*tutorialInsetScale + tutorialInsetY,
		box.w * tutorialInsetScale,
		box.h * tutorialInsetScale
}

func (s *BattleScene) drawBattleTutorial(screen *ebiten.Image) {
	if s.tutorialSceneImg == nil {
		s.tutorialSceneImg = ebiten.NewImage(gameWidth, gameHeight)
	}
	scene := s.tutorialSceneImg
	scene.Clear()
	s.drawBackground(scene)
	s.drawEnemyHeader(scene)
	s.drawPartySprites(scene)

	if s.tutorialKind == battleTutorialKindGauge && s.tutorialPage == battleTutorialGaugePageRewindUnlock {
		// This page introduces the rewind ability, which only unlocks once
		// the gauge is full, so the gauge is drawn maxed out here (including
		// its max-stage rainbow color) rather than at its real, empty,
		// pre-battle value.
		origPoint, origStage := s.gaugePoint, s.gaugeStage
		s.gaugePoint = gaugePoolMax
		s.gaugeStage = gaugeMaxStage - 1
		s.drawUI(scene)
		s.gaugePoint, s.gaugeStage = origPoint, origStage
	} else {
		s.drawUI(scene)
	}

	box := s.battleTutorialBox(s.tutorialKind, s.tutorialPage)

	if s.tutorialKind == battleTutorialKindBasics && s.tutorialPage == battleTutorialPageCommands {
		positions := commandIconPositions()
		for i, pos := range positions {
			icon := s.game.CommandIcons[i]
			if icon == nil {
				continue
			}
			iw := icon.Bounds().Dx()
			ih := icon.Bounds().Dy()
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(pos[0]-float64(iw)/2, pos[1]-float64(ih)/2)
			scene.DrawImage(icon, op)
		}
		ix, iy := itemButtonCenter()
		drawItemIcon(scene, ix, iy, battleIconR, false)
	}

	s.dimSceneExceptBox(scene, box)

	if bg := s.game.MenuBgImg; bg != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(
			float64(gameWidth)/float64(bg.Bounds().Dx()),
			float64(gameHeight)/float64(bg.Bounds().Dy()),
		)
		screen.DrawImage(bg, op)
	} else {
		screen.Fill(color.RGBA{10, 10, 20, 255})
	}

	insetOp := &ebiten.DrawImageOptions{}
	insetOp.GeoM.Scale(tutorialInsetScale, tutorialInsetScale)
	insetOp.GeoM.Translate(tutorialInsetX(), tutorialInsetY)
	screen.DrawImage(scene, insetOp)

	// The highlight box's border is drawn here, directly in screen space,
	// rather than pre-scale on the scene, so it comes out the same crisp
	// tutorialBoxBorderWidth on every page regardless of the box's own size.
	var boxScreenX, boxScreenY, boxScreenW, boxScreenH float64
	if box != nil {
		boxScreenX, boxScreenY, boxScreenW, boxScreenH = box.toScreen()
		vector.StrokeRect(screen, float32(boxScreenX), float32(boxScreenY), float32(boxScreenW), float32(boxScreenH), tutorialBoxBorderWidth, color.White, true)
	}

	const (
		tutorialTitleX = 24.0
		tutorialTitleY = 24.0
	)
	titleStr := battleTutorialHeadline(s.tutorialKind)
	titleFace := s.game.FontFace(44)
	titleOp := &text.DrawOptions{}
	titleOp.GeoM.Translate(tutorialTitleX, tutorialTitleY)
	titleOp.ColorScale.ScaleWithColor(color.White)
	text.Draw(screen, titleStr, titleFace, titleOp)

	bodyFace := s.game.FontFace(24)
	lineSpacing := bodyFace.Metrics().HAscent + bodyFace.Metrics().HDescent + 6
	lines := strings.Split(battleTutorialBody(s.tutorialKind, s.tutorialPage), "\n")

	// A page with no highlight box (each tutorial's intro/conclusion-style
	// messages) has its text sit at the plain default spot below the title,
	// unified across every such page instead of some sitting beside the
	// title and others below it. Legibility against whatever's on screen
	// there comes from dimImageExceptBox always dimming the scene, box or
	// no box, rather than from this text's position.
	isCommandsPage := s.tutorialKind == battleTutorialKindBasics && s.tutorialPage == battleTutorialPageCommands
	bodyX := 40.0
	const bodyDefaultY = 100.0
	bodyY := bodyDefaultY
	if box != nil && s.tutorialKind == battleTutorialKindGauge {
		// The rest of the gauge tutorial always explains the same on-screen
		// element, so its text stays anchored below the gauge on every page
		// instead of jumping around with the box placement rules below.
		if gaugeBox := s.gaugeHighlightBox(); gaugeBox != nil {
			_, gy, _, gh := gaugeBox.toScreen()
			const gap = 30.0
			bodyY = gy + gh + gap
		}
	} else if box != nil {
		boxScreenBottom := boxScreenY + boxScreenH
		boxScreenRight := boxScreenX + boxScreenW
		textHeight := float64(len(lines)) * lineSpacing
		const gap = 30.0

		if boxCenterY := boxScreenY + boxScreenH/2; boxCenterY < float64(gameHeight)/2 {
			// A box in the upper half (timeline, gauge) can reach into the
			// text's default spot under the title, so push the text below
			// it instead of letting them overlap.
			if boxScreenY < bodyDefaultY+textHeight && boxScreenBottom > bodyDefaultY {
				bodyY = boxScreenBottom + gap
			}
		} else {
			// A box in the lower half (status block, commands) would
			// otherwise leave the text stranded up at the default position,
			// far from the thing it's describing, so pull it down to sit
			// just above the box instead.
			bodyY = boxScreenY - textHeight - gap
			if bodyY < bodyDefaultY {
				bodyY = bodyDefaultY
			}
		}

		// A box sitting clearly in the right portion of the screen (like the
		// command icons) reads better with the text pulled over next to it
		// instead of staying pinned to the far-left margin; a box that spans
		// most of the width (like the timeline's) isn't "on a side" so it's
		// left alone. The block is anchored by its widest line's left edge
		// (rather than right-aligning every line individually) so that,
		// combined with the label/colon alignment below, the "：" column
		// stays put regardless of how long each description happens to be.
		boxCenterX := (boxScreenX + boxScreenRight) / 2
		if boxCenterX > float64(gameWidth)*0.6 {
			maxLineWidth := 0.0
			for _, line := range lines {
				if adv := text.Advance(line, bodyFace); adv > maxLineWidth {
					maxLineWidth = adv
				}
			}
			bodyX = boxScreenRight - maxLineWidth
		}
	}

	// The command list's four "ラベル：説明" lines get their labels measured
	// so every line's "：" lands in the same column, instead of drifting
	// with each label's own character count.
	labelDescGap := 0.0
	if isCommandsPage {
		maxLabelAdvance := 0.0
		for _, line := range lines {
			if idx := strings.Index(line, "："); idx >= 0 {
				if adv := text.Advance(line[:idx], bodyFace); adv > maxLabelAdvance {
					maxLabelAdvance = adv
				}
			}
		}
		labelDescGap = maxLabelAdvance
	}

	for i, line := range lines {
		y := bodyY + float64(i)*lineSpacing
		if isCommandsPage {
			if idx := strings.Index(line, "："); idx >= 0 {
				labelOp := &text.DrawOptions{}
				labelOp.GeoM.Translate(bodyX, y)
				labelOp.ColorScale.ScaleWithColor(color.White)
				text.Draw(screen, line[:idx], bodyFace, labelOp)

				descOp := &text.DrawOptions{}
				descOp.GeoM.Translate(bodyX+labelDescGap, y)
				descOp.ColorScale.ScaleWithColor(color.White)
				text.Draw(screen, line[idx:], bodyFace, descOp)
				continue
			}
		}
		op := &text.DrawOptions{}
		op.GeoM.Translate(bodyX, y)
		op.ColorScale.ScaleWithColor(color.White)
		text.Draw(screen, line, bodyFace, op)
	}

	if math.Mod(s.gaugeColorAnimTimer, 1.0) < 0.6 {
		hint := "▼ 決定で次へ"
		if s.tutorialPage >= battleTutorialPageCountFor(s.tutorialKind)-1 {
			hint = "▼ 決定ではじめる"
		}
		hintFace := s.game.FontFace(18)
		hintOp := &text.DrawOptions{}
		hintOp.PrimaryAlign = text.AlignEnd
		hintOp.GeoM.Translate(float64(gameWidth)-24, float64(gameHeight)-32)
		hintOp.ColorScale.ScaleWithColor(uiColorText)
		text.Draw(screen, hint, hintFace, hintOp)
	}
}
