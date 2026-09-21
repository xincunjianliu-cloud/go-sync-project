package main

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const skillUpgradeTutorialBody = "今まで集めてきたSPを消費して\n" +
	"スキルの強化が可能になった。\n" +
	"スキルを強化するとスキル威力や効果量が上昇するよ"

func (s *FieldScene) drawSkillUpgradeTutorial(screen *ebiten.Image) {
	vector.DrawFilledRect(screen, 0, 0, float32(gameWidth), float32(gameHeight), color.RGBA{0, 0, 0, 90}, true)

	titleFace := s.game.FontFace(24)
	titleOp := &text.DrawOptions{}
	titleOp.GeoM.Translate(40, 40)
	titleOp.ColorScale.ScaleWithColor(color.White)
	text.Draw(screen, "・スキルの強化", titleFace, titleOp)

	bodyFace := s.game.FontFace(18)
	lineSpacing := bodyFace.Metrics().HAscent + bodyFace.Metrics().HDescent + 4
	for i, line := range strings.Split(skillUpgradeTutorialBody, "\n") {
		op := &text.DrawOptions{}
		op.GeoM.Translate(40, 100+float64(i)*lineSpacing)
		op.ColorScale.ScaleWithColor(color.White)
		text.Draw(screen, line, bodyFace, op)
	}

	if (s.wallAnimTick/30)%2 == 0 {
		hintFace := s.game.FontFace(16)
		hintOp := &text.DrawOptions{}
		hintOp.PrimaryAlign = text.AlignEnd
		hintOp.GeoM.Translate(float64(gameWidth)-24, float64(gameHeight)-32)
		hintOp.ColorScale.ScaleWithColor(uiColorText)
		text.Draw(screen, "▼ 決定で閉じる", hintFace, hintOp)
	}
}
