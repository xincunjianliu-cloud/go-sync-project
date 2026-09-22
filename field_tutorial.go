package main

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// skillUpgradeTutorialPageIntro/SelectSkill/SpendSP/LevelUpEffects/
// KnockbackExample/AttackStatExample/LevelSwitch are the skill-upgrade
// tutorial's page indices, in the order they're shown. This tutorial folds
// together what used to be two separate one-time hints (the skill-upgrade
// unlock message and the battle skill submenu's "←→ switches level" hint),
// so its last page covers the level switch instead of a separate mechanism.
const (
	skillUpgradeTutorialPageIntro = iota
	skillUpgradeTutorialPageSelectSkill
	skillUpgradeTutorialPageSpendSP
	skillUpgradeTutorialPageLevelUpEffects
	skillUpgradeTutorialPageKnockbackExample
	skillUpgradeTutorialPageAttackStatExample
	skillUpgradeTutorialPageLevelSwitch
	skillUpgradeTutorialPageCount
)

func skillUpgradeTutorialBody(page int) string {
	switch page {
	case skillUpgradeTutorialPageIntro:
		return "スキル強化ができるようになりました。"
	case skillUpgradeTutorialPageSelectSkill:
		return "メニューの「スキル」を選択しよう。"
	case skillUpgradeTutorialPageSpendSP:
		return "スキルはバトルで入手した「SP」を使って\n" +
			"強化することができます。"
	case skillUpgradeTutorialPageLevelUpEffects:
		return "スキルレベルが上がると技の威力が上昇するほか、\n" +
			"新たな効果が付与されることもあります。\n" +
			"またそのスキルに関連したステータスが上昇します。"
	case skillUpgradeTutorialPageKnockbackExample:
		return "例えば「強撃」をレベル２に強化すると、\n" +
			"攻撃力アップに加えて敵をノックバックさせる\n" +
			"効果が付与されます。"
	case skillUpgradeTutorialPageAttackStatExample:
		return "また「強撃」のような攻撃系スキルを強化すると、\n" +
			"それに伴ってキャラクターの攻撃ステータスも\n" +
			"上昇します。"
	default:
		return "強化したスキルはレベル表示の横にある◀▶で\n" +
			"切り替えて使うことができます。"
	}
}

// skillUpgradeTutorialCharIdx/SkillIdx pick a single real character+skill
// (the hero's 強撃) as the tutorial's live example on every mockup page, so
// the name/levels/costs shown always come from this game's real data
// instead of invented placeholder text. 強撃 is always unlocked from level
// 1, so the tutorial always has something to show regardless of the
// player's actual progress.
const (
	skillUpgradeTutorialCharIdx  = 0
	skillUpgradeTutorialSkillIdx = 0
)

// skillUpgradeTutorialMockLevel is the skill level briefly forced onto the
// example above while its mockup pages render, so the "next level" gauge,
// cost and ◀▶ switch it's meant to illustrate are actually visible even
// though the player has (realistically) never upgraded anything yet at the
// point this tutorial fires.
const skillUpgradeTutorialMockLevel = 2

// skillUpgradeTutorialMenuCommands/Index mirror the real command list built
// in NewMenuScene (menu_scene.go) - kept in sync there, since this tutorial
// mocks up the pause menu's look without constructing (or navigating to) a
// real MenuScene.
var skillUpgradeTutorialMenuCommands = []string{"アイテム", "スキル", "ステータス", "セーブ", "ロード", "オプション", "タイトルに戻る"}

const skillUpgradeTutorialMenuCommandIndex = 1 // "スキル"

// withMockSkillLevel briefly forces the tutorial's example skill to
// skillUpgradeTutorialMockLevel, runs draw, and restores the real level
// immediately after - the same save/override/restore trick the gauge
// tutorial uses (see battleTutorialGaugePageRewindUnlock in
// battle_tutorial.go) to show UI state the player hasn't actually reached
// yet, without ever letting the override leak into real save data.
func (s *FieldScene) withMockSkillLevel(draw func()) {
	origLv := s.game.PlayerSkillLv[skillUpgradeTutorialCharIdx][skillUpgradeTutorialSkillIdx]
	s.game.PlayerSkillLv[skillUpgradeTutorialCharIdx][skillUpgradeTutorialSkillIdx] = skillUpgradeTutorialMockLevel
	draw()
	s.game.PlayerSkillLv[skillUpgradeTutorialCharIdx][skillUpgradeTutorialSkillIdx] = origLv
}

// drawSkillUpgradeTutorialMockup draws this tutorial's backdrop: a mockup of
// whichever real screen the current page is explaining, built either by
// reusing the actual MenuScene/BattleScene draw methods on a throwaway
// instance (so the look can never drift from the real UI) or, for the
// command list, drawing it directly with the real layout constants.
func (s *FieldScene) drawSkillUpgradeTutorialMockup(scene *ebiten.Image, page int) {
	// The level-switch page explains something that only ever appears
	// mid-battle, so it gets an actual battle screen as its backdrop rather
	// than the menu's - see drawBattleLevelSwitchMockup.
	if page == skillUpgradeTutorialPageLevelSwitch {
		s.drawBattleLevelSwitchMockup(scene)
		return
	}

	if bg := s.game.MenuBgImg; bg != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(
			float64(gameWidth)/float64(bg.Bounds().Dx()),
			float64(gameHeight)/float64(bg.Bounds().Dy()),
		)
		scene.DrawImage(bg, op)
	}
	drawMenuBgFrame(scene)

	switch page {
	case skillUpgradeTutorialPageIntro:
		// Nothing is selected yet on this page (menuIndex -1 matches no
		// row, so drawCommandList highlights none of them) - it's just
		// showing the menu that's about to be opened.
		mockMenu := &MenuScene{
			game:      s.game,
			commands:  skillUpgradeTutorialMenuCommands,
			menuIndex: -1,
		}
		mockMenu.drawCommandList(scene)

	case skillUpgradeTutorialPageSelectSkill:
		mockMenu := &MenuScene{
			game:      s.game,
			commands:  skillUpgradeTutorialMenuCommands,
			menuIndex: skillUpgradeTutorialMenuCommandIndex,
		}
		mockMenu.drawCommandList(scene)

	case skillUpgradeTutorialPageSpendSP, skillUpgradeTutorialPageLevelUpEffects, skillUpgradeTutorialPageKnockbackExample, skillUpgradeTutorialPageAttackStatExample:
		s.withMockSkillLevel(func() {
			mockMenu := &MenuScene{
				game:           s.game,
				skillCharIndex: skillUpgradeTutorialCharIdx,
				skillSubIndex:  skillUpgradeTutorialSkillIdx,
			}
			mockMenu.drawSkillSubMenu(scene)
		})
	}
}

// drawBattleLevelSwitchMockup renders the real battle background and party
// sprites (BattleScene.drawBackground/drawPartySprites on a throwaway
// instance) with the real battle skill submenu on top, so this page reads
// as "this is what you'll see in battle" instead of floating the same skill
// panel over the menu's own backdrop like the earlier pages.
func (s *FieldScene) drawBattleLevelSwitchMockup(scene *ebiten.Image) {
	mockBattle := &BattleScene{
		game:         s.game,
		waitingActor: skillUpgradeTutorialCharIdx,
		skillIndex:   skillUpgradeTutorialSkillIdx,
	}
	mockBattle.drawBackground(scene)
	mockBattle.drawPartySprites(scene)

	if s.game.SkillPanelImg == nil {
		return
	}
	s.withMockSkillLevel(func() {
		mockBattle.skillLevelCursors[skillUpgradeTutorialCharIdx][skillUpgradeTutorialSkillIdx] = skillUpgradeTutorialMockLevel
		mockBattle.drawSkillSubMenu(scene)
	})
}

// skillUpgradeTutorialBox returns the highlight box for the given page, or
// nil on pages with nothing to highlight (the intro). Coordinates reuse the
// exact same layout constants as the real screens being mocked up
// (menu_draw.go / battle_submenu_touch.go) so the box always lands on the
// element it names regardless of later layout tweaks there.
func (s *FieldScene) skillUpgradeTutorialBox(page int) *battleTutorialBox {
	switch page {
	case skillUpgradeTutorialPageSelectSkill:
		cmdFace := s.game.FontFace(cmdListFontSize)
		label := "▶ " + skillUpgradeTutorialMenuCommands[skillUpgradeTutorialMenuCommandIndex]
		rowW := text.Advance(label, cmdFace)
		rowH := cmdFace.Metrics().HAscent + cmdFace.Metrics().HDescent
		baseX := cmdListStartX
		baseY := cmdListStartY + float64(skillUpgradeTutorialMenuCommandIndex)*cmdListRowGapY
		const padX, padTop, padBottom = 14.0, 8.0, 10.0
		return &battleTutorialBox{baseX - padX, baseY - padTop, rowW + padX*2, rowH + padTop + padBottom}

	case skillUpgradeTutorialPageSpendSP:
		spFace := s.game.FontFace(skillSPHeaderFontSize)
		spText := fmt.Sprintf("所持SP: %d", s.game.PlayerSP[skillUpgradeTutorialCharIdx])
		spW := text.Advance(spText, spFace)
		spH := spFace.Metrics().HAscent + spFace.Metrics().HDescent
		const padX, padY = 14.0, 10.0
		return &battleTutorialBox{skillSPHeaderX - spW - padX, skillSPHeaderY - padY, spW + padX*2, spH + padY*2}

	case skillUpgradeTutorialPageKnockbackExample, skillUpgradeTutorialPageAttackStatExample:
		maxLv := len(s.game.CharacterSkills(skillUpgradeTutorialCharIdx)[skillUpgradeTutorialSkillIdx].Levels)
		firstX := skillLevelStartX
		lastX := skillLevelStartX + float64(maxLv-1)*skillLevelGapX
		lvFace := s.game.FontFace(skillLevelFontSize)
		lvH := lvFace.Metrics().HAscent + lvFace.Metrics().HDescent
		centerY := skillRowStartY + skillLevelOffsetY
		const padX, padTop, bottomReach = 40.0, 16.0, 40.0
		top := centerY - lvH/2 - padTop
		bottom := centerY + bottomReach
		left := firstX - skillLevelGapX/2 - padX
		right := lastX + skillLevelGapX/2 + padX
		return &battleTutorialBox{left, top, right - left, bottom - top}

	case skillUpgradeTutorialPageLevelSwitch:
		if s.game.SkillPanelImg == nil {
			return nil
		}
		windowW := float64(s.game.SkillPanelImg.Bounds().Dx())
		windowX := battleSubCmdCenterX - windowW/2
		windowY := battleSubCmdCenterY - float64(s.game.SkillPanelImg.Bounds().Dy())/2
		lvFace := s.game.FontFace(18)
		lvW := text.Advance(skillLvText(skillUpgradeTutorialMockLevel), lvFace)
		leftArrowW := text.Advance(skillLvLeftArrow, lvFace)
		rightArrowW := text.Advance(skillLvRightArrow, lvFace)
		blockRightX := windowX + windowW - battleSubLvBlockRightX
		rightX := blockRightX - rightArrowW
		lvX := rightX - battleSubLvGap - lvW
		leftX := lvX - battleSubLvGap - leftArrowW
		textY := windowY + battleSubLabelOffsetY
		rowH := lvFace.Metrics().HAscent + lvFace.Metrics().HDescent
		const padX, padY = 14.0, 10.0
		return &battleTutorialBox{leftX - padX, textY - padY, (blockRightX - leftX) + padX*2, rowH + padY*2}

	default:
		return nil
	}
}

// drawSkillUpgradeTutorial renders the skill-upgrade tutorial the same way
// the battle scene's tutorial does (see drawBattleTutorial in
// battle_tutorial.go): a mocked-up real screen dimmed everywhere except a
// highlight box, composited as a smaller inset with room for the title, a
// white border framing the box, and body text positioned beside/above/below
// it so the two never overlap.
func (s *FieldScene) drawSkillUpgradeTutorial(screen *ebiten.Image) {
	if s.skillUpgradeTutorialSceneImg == nil {
		s.skillUpgradeTutorialSceneImg = ebiten.NewImage(gameWidth, gameHeight)
	}
	scene := s.skillUpgradeTutorialSceneImg
	scene.Clear()
	s.drawSkillUpgradeTutorialMockup(scene, s.skillUpgradeTutorialPage)

	box := s.skillUpgradeTutorialBox(s.skillUpgradeTutorialPage)

	if s.skillUpgradeTutorialOverlay == nil {
		s.skillUpgradeTutorialOverlay = ebiten.NewImage(gameWidth, gameHeight)
	}
	dimImageExceptBox(scene, s.skillUpgradeTutorialOverlay, s.game.MenuBgImg, box)

	// The inset scene below only covers a shrunken, centered portion of the
	// screen (see tutorialInsetScale), leaving a margin around it - fill
	// that margin with the same menu backdrop first so it doesn't show the
	// raw field map peeking out from behind, matching drawBattleTutorial.
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

	var boxScreenX, boxScreenY, boxScreenW, boxScreenH float64
	if box != nil {
		boxScreenX, boxScreenY, boxScreenW, boxScreenH = box.toScreen()
		vector.StrokeRect(screen, float32(boxScreenX), float32(boxScreenY), float32(boxScreenW), float32(boxScreenH), tutorialBoxBorderWidth, color.White, true)
	}

	const (
		titleX = 24.0
		titleY = 24.0
	)
	titleStr := "スキル強化"
	titleFace := s.game.FontFace(44)
	titleOp := &text.DrawOptions{}
	titleOp.GeoM.Translate(titleX, titleY)
	titleOp.ColorScale.ScaleWithColor(color.White)
	text.Draw(screen, titleStr, titleFace, titleOp)

	bodyFace := s.game.FontFace(24)
	lineSpacing := bodyFace.Metrics().HAscent + bodyFace.Metrics().HDescent + 6
	lines := strings.Split(skillUpgradeTutorialBody(s.skillUpgradeTutorialPage), "\n")

	// A page with no highlight box (currently just the intro) has its text
	// sit at the plain default spot below the title, unified with every
	// other box-less page instead of sitting beside the title on its own.
	// Legibility against whatever's on screen there comes from
	// dimImageExceptBox always dimming the scene, box or no box, rather
	// than from this text's position.
	bodyX := 40.0
	const bodyDefaultY = 100.0
	bodyY := bodyDefaultY
	if box != nil {
		boxScreenBottom := boxScreenY + boxScreenH
		boxScreenRight := boxScreenX + boxScreenW
		textHeight := float64(len(lines)) * lineSpacing
		const gap = 30.0

		if boxCenterY := boxScreenY + boxScreenH/2; boxCenterY < float64(gameHeight)/2 {
			if boxScreenY < bodyDefaultY+textHeight && boxScreenBottom > bodyDefaultY {
				bodyY = boxScreenBottom + gap
			}
		} else {
			bodyY = boxScreenY - textHeight - gap
			if bodyY < bodyDefaultY {
				bodyY = bodyDefaultY
			}
		}

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

	for i, line := range lines {
		op := &text.DrawOptions{}
		op.GeoM.Translate(bodyX, bodyY+float64(i)*lineSpacing)
		op.ColorScale.ScaleWithColor(color.White)
		text.Draw(screen, line, bodyFace, op)
	}

	if (s.wallAnimTick/30)%2 == 0 {
		hint := "▼ 決定で次へ"
		if s.skillUpgradeTutorialPage >= skillUpgradeTutorialPageCount-1 {
			hint = "▼ 決定で閉じる"
		}
		hintFace := s.game.FontFace(18)
		hintOp := &text.DrawOptions{}
		hintOp.PrimaryAlign = text.AlignEnd
		hintOp.GeoM.Translate(float64(gameWidth)-24, float64(gameHeight)-32)
		hintOp.ColorScale.ScaleWithColor(uiColorText)
		text.Draw(screen, hint, hintFace, hintOp)
	}
}
