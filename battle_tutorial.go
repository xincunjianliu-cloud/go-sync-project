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

func battleTutorialPageCountFor(kind int) int {
	switch kind {
	case battleTutorialKindBasics:
		return 3
	case battleTutorialKindGauge:
		return 1
	default:
		return 0
	}
}

func battleTutorialHeadline(kind int) string {
	if kind == battleTutorialKindGauge {
		return "・バトルのきほん２"
	}
	return "・バトルのきほん"
}

type battleTutorialArrow struct {
	fromX, fromY float64
	toX, toY     float64
}

type battleTutorialCallout struct {
	labelX, labelY float64
	rightAlign     bool
	noArrow        bool
	text           string
	arrow          battleTutorialArrow
}

// battleTutorialCallouts returns the text + arrow annotations for one page
// of the given tutorial kind. Coordinates are derived from the same layout
// helpers the real HUD uses (partyNamePosition, commandIconPositions,
// goalScreenX/trackCenterY) so the arrows keep pointing at the right spot
// even if that layout changes later.
func (s *BattleScene) battleTutorialCallouts(kind, page int) []battleTutorialCallout {
	if kind == battleTutorialKindGauge {
		return []battleTutorialCallout{
			{
				labelX: 260, labelY: 110, noArrow: true,
				text: "新しいゲージが登場。\n\n" +
					"行動してゲージを溜めよう！\n" +
					"ゲージを溜めるごとに攻撃力アップする\n" +
					"さらにゲージを最大まで溜めると敵の行動を\n" +
					"無かったことにできる巻き戻しが使用可能！\n" +
					"さらに攻撃力upなど色んな恩恵があるよん",
			},
		}
	}

	switch page {
	case 0:
		sx, sy := partyNamePosition(0)
		hpY := sy + statusHPBarY + statusBarH/2
		mpY := sy + statusMPBarY + statusBarH/2
		barRight := sx + statusBlockW
		return []battleTutorialCallout{
			{
				labelX: barRight + 35, labelY: hpY - 8,
				text:  "ここのHPが0になったら負け",
				arrow: battleTutorialArrow{barRight + 30, hpY, barRight + 4, hpY},
			},
			{
				labelX: barRight + 35, labelY: mpY - 8,
				text:  "スキルに必要なMP",
				arrow: battleTutorialArrow{barRight + 30, mpY, barRight + 4, mpY},
			},
		}

	case 1:
		positions := commandIconPositions()
		attackX, attackY := positions[cmdNormalAttack][0], positions[cmdNormalAttack][1]
		skillX, skillY := positions[cmdSkill][0], positions[cmdSkill][1]
		waitX, waitY := positions[cmdWait][0], positions[cmdWait][1]
		fleeX, fleeY := positions[cmdFlee][0], positions[cmdFlee][1]

		// Label rows are spaced independently of the icons' own Y positions
		// (skill/wait share a Y) and kept within the 540px-tall screen; each
		// arrow still points at the icon's real coordinates.
		const labelX = 560.0
		const row1Y, row2Y, row3Y, row4Y = 330.0, 385.0, 440.0, 495.0
		return []battleTutorialCallout{
			{
				labelX: labelX, labelY: row1Y, rightAlign: true,
				text:  "ふつうのこうげき",
				arrow: battleTutorialArrow{labelX + 12, row1Y + 6, attackX - 28, attackY - 6},
			},
			{
				labelX: labelX, labelY: row2Y, rightAlign: true,
				text:  "ここで待機！全員揃って大ダメージ",
				arrow: battleTutorialArrow{labelX + 12, row2Y + 6, waitX - 28, waitY - 2},
			},
			{
				labelX: labelX, labelY: row3Y, rightAlign: true,
				text:  "スキルがつかえる",
				arrow: battleTutorialArrow{labelX + 12, row3Y + 6, skillX - 26, skillY + 8},
			},
			{
				labelX: labelX, labelY: row4Y, rightAlign: true,
				text:  "バトルから逃げれる",
				arrow: battleTutorialArrow{labelX + 12, row4Y + 6, fleeX - 8, fleeY + 22},
			},
		}

	default:
		goalX := s.goalScreenX()
		centerY := trackCenterY()
		const labelX = 90.0
		labelY := centerY + 90.0
		return []battleTutorialCallout{
			{
				labelX: labelX, labelY: labelY,
				text:  "ここのタイムラインで右に行くと行動開始",
				arrow: battleTutorialArrow{labelX + 230, labelY - 14, goalX - 30, centerY + 4},
			},
		}
	}
}

// battleTutorialSpotlight is a soft, elliptical "hole" punched into the
// tutorial's dark overlay so the HUD element under discussion visibly lights
// up instead of just being pointed at.
type battleTutorialSpotlight struct {
	x, y             float64
	radiusX, radiusY float64
}

// battleTutorialSpotlights returns the HUD regions to light up for the given
// tutorial kind/page, using the same real layout helpers as the callouts.
func (s *BattleScene) battleTutorialSpotlights(kind, page int) []battleTutorialSpotlight {
	if kind == battleTutorialKindGauge {
		if s.game.GaugeImg == nil {
			return nil
		}
		w := float64(s.game.GaugeImg.Bounds().Dx())
		h := float64(s.game.GaugeImg.Bounds().Dy())
		return []battleTutorialSpotlight{
			{gaugeTriX + w/2, gaugeTriY + h/2, w/2 + 14, h/2 + 14},
		}
	}

	switch page {
	case 0:
		sx, sy := partyNamePosition(0)
		hpY := sy + statusHPBarY + statusBarH/2
		mpY := sy + statusMPBarY + statusBarH/2
		cx := sx + statusBlockW/2
		return []battleTutorialSpotlight{
			{cx, hpY, statusBlockW/2 + 20, 16},
			{cx, mpY, statusBlockW/2 + 20, 16},
		}

	case 1:
		positions := commandIconPositions()
		spots := make([]battleTutorialSpotlight, 0, len(positions))
		for _, pos := range positions {
			spots = append(spots, battleTutorialSpotlight{pos[0], pos[1], 34, 34})
		}
		return spots

	default:
		goalX := s.goalScreenX()
		centerY := trackCenterY()
		return []battleTutorialSpotlight{
			{goalX - 130, centerY, 230, 26},
		}
	}
}

// drawTutorialSpotlightOverlay dims the whole screen and then erases soft
// elliptical holes at each spotlight, reusing the same radial gradient mask
// (LightMaskImg) and erase-blend technique as the field scene's flashlight
// effect (see drawDarkness in field_draw.go): draw dark, then punch bright
// holes into it with BlendDestinationOut so the already-drawn HUD underneath
// shows back through.
func (s *BattleScene) drawTutorialSpotlightOverlay(screen *ebiten.Image, spots []battleTutorialSpotlight) {
	if s.tutorialOverlayImg == nil {
		s.tutorialOverlayImg = ebiten.NewImage(gameWidth, gameHeight)
	}
	overlay := s.tutorialOverlayImg
	overlay.Fill(color.RGBA{0, 0, 0, 180})

	mask := s.game.LightMaskImg
	if mask != nil {
		maskW := float64(mask.Bounds().Dx())
		maskH := float64(mask.Bounds().Dy())
		for _, sp := range spots {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale((sp.radiusX*2)/maskW, (sp.radiusY*2)/maskH)
			op.GeoM.Translate(sp.x-sp.radiusX, sp.y-sp.radiusY)
			op.Blend = ebiten.BlendDestinationOut
			overlay.DrawImage(mask, op)
		}
	}

	screen.DrawImage(overlay, &ebiten.DrawImageOptions{})
}

func (s *BattleScene) drawBattleTutorial(screen *ebiten.Image) {
	s.drawTutorialSpotlightOverlay(screen, s.battleTutorialSpotlights(s.tutorialKind, s.tutorialPage))

	if s.tutorialKind == battleTutorialKindBasics && s.tutorialPage == 1 {
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
			screen.DrawImage(icon, op)
		}
	}

	titleFace := s.game.FontFace(24)
	titleOp := &text.DrawOptions{}
	titleOp.GeoM.Translate(40, 40)
	titleOp.ColorScale.ScaleWithColor(color.White)
	text.Draw(screen, battleTutorialHeadline(s.tutorialKind), titleFace, titleOp)

	labelFace := s.game.FontFace(18)
	lineSpacing := labelFace.Metrics().HAscent + labelFace.Metrics().HDescent + 4
	arrowColor := color.RGBA{235, 235, 235, 255}

	for _, c := range s.battleTutorialCallouts(s.tutorialKind, s.tutorialPage) {
		if !c.noArrow {
			drawBattleTutorialArrow(screen, c.arrow, arrowColor)
		}

		lines := strings.Split(c.text, "\n")
		for i, line := range lines {
			op := &text.DrawOptions{}
			if c.rightAlign {
				op.PrimaryAlign = text.AlignEnd
			}
			op.GeoM.Translate(c.labelX, c.labelY+float64(i)*lineSpacing)
			op.ColorScale.ScaleWithColor(color.White)
			text.Draw(screen, line, labelFace, op)
		}
	}

	if math.Mod(s.gaugeColorAnimTimer, 1.0) < 0.6 {
		hint := "▼ 決定で次へ"
		if s.tutorialPage >= battleTutorialPageCountFor(s.tutorialKind)-1 {
			hint = "▼ 決定ではじめる"
		}
		hintFace := s.game.FontFace(16)
		hintOp := &text.DrawOptions{}
		hintOp.PrimaryAlign = text.AlignEnd
		hintOp.GeoM.Translate(float64(gameWidth)-24, float64(gameHeight)-32)
		hintOp.ColorScale.ScaleWithColor(uiColorText)
		text.Draw(screen, hint, hintFace, hintOp)
	}
}

// partyHasLeveledSkill reports whether any party member currently has a
// skill raised past level 1, i.e. whether the skill submenu's level cursor
// (←/→) actually has more than one level to switch between right now.
func partyHasLeveledSkill(game *Game) bool {
	for i := 0; i < partySize; i++ {
		skills := game.CharacterSkills(i)
		for j, sk := range skills {
			if len(sk.Levels) > 1 && j < len(game.PlayerSkillLv[i]) && game.PlayerSkillLv[i][j] >= 2 {
				return true
			}
		}
	}
	return false
}

// skillLevelHint returns the one-time "←→ switches skill level" hint text
// for the skill submenu's bottom description bar. It only fires for the
// first battle in which a leveled-up skill is actually available (tracked by
// skillLevelHintActive, frozen at battle start), and only while the
// currently selected skill has more than one level to switch between.
func (s *BattleScene) skillLevelHint() string {
	if !s.skillLevelHintActive {
		return ""
	}
	p := s.waitingActor
	if p < 0 || p >= partySize {
		return ""
	}
	skills := s.game.CharacterSkills(p)
	if s.skillIndex < 0 || s.skillIndex >= len(skills) {
		return ""
	}
	if len(skills[s.skillIndex].Levels) < 2 || s.game.PlayerSkillLv[p][s.skillIndex] < 2 {
		return ""
	}
	return "←→でスキルのレベルを切り替え（高いほど効果アップ）"
}

func drawBattleTutorialArrow(screen *ebiten.Image, a battleTutorialArrow, col color.Color) {
	vector.StrokeLine(screen, float32(a.fromX), float32(a.fromY), float32(a.toX), float32(a.toY), 2, col, true)

	angle := math.Atan2(a.toY-a.fromY, a.toX-a.fromX)
	const headLen = 9.0
	const headSpread = 0.5
	p1x := a.toX - headLen*math.Cos(angle-headSpread)
	p1y := a.toY - headLen*math.Sin(angle-headSpread)
	p2x := a.toX - headLen*math.Cos(angle+headSpread)
	p2y := a.toY - headLen*math.Sin(angle+headSpread)
	fillTriPath(screen, col, [][2]float64{{a.toX, a.toY}, {p1x, p1y}, {p2x, p2y}})
}
