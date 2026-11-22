package main

import (
	"encoding/json"
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	creditLineGap               = 32.0
	creditSectionGap            = 24.0
	creditFontSize              = 20.0
	creditFallbackMusicDuration = 90.0
	endingLogoFadeInDuration    = 0.8
	endingLogoHoldDuration      = 3.0
	endingLogoFadeOutDuration   = 0.8
	endingLogoFadeOutStart      = endingLogoFadeInDuration + endingLogoHoldDuration
	endingLogoDuration          = endingLogoFadeOutStart + endingLogoFadeOutDuration
	endingBGMFinishBeforeLogo   = 0.5
)

type CreditPage struct {
	Lines []string `json:"lines"`
}

func loadCredits(path string) ([]CreditPage, error) {
	data, err := loadAssetBytes(path)
	if err != nil {
		return nil, err
	}
	var f struct {
		Pages []CreditPage `json:"pages"`
	}
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	return f.Pages, nil
}

const (
	endingPhaseCredits = iota
	endingPhaseLogo
	endingPhaseAskSave
	endingPhaseSaveSlot
	endingPhaseSlotConfirm
	endingPhaseSaveDone
)

type EndingScene struct {
	game  *Game
	field *FieldScene

	pages                 []CreditPage
	creditScrollDistance  float64
	creditMusicOffset     float64
	creditDuration        float64
	creditElapsed         float64
	creditFallbackElapsed float64
	creditPlaybackStarted bool
	creditPlaybackWarned  bool

	phase             int
	logoElapsed       float64
	logoMissingWarned bool
	endingBGMStopped  bool

	skipConfirm    bool
	skipConfirmIdx int

	askSaveIndex int

	slotIndex         int
	slotScrollTop     int
	slotData          [maxSaveSlots]*SaveData
	slotThumbs        [maxSaveSlots]*ebiten.Image
	slotDrag          dragScrollState
	slotScrollBarDrag dragScrollState
	slotDragAccum     float64

	confirmIndex  int
	pendingSlot   int
	saveResultMsg string
	saveDoneIndex int

	inputLockTicks int

	nextScene Scene
}

func NewEndingScene(game *Game, field *FieldScene) *EndingScene {
	pages, err := loadCredits("assets/credits.json")
	if err != nil || len(pages) == 0 {
		pages = []CreditPage{{Lines: []string{"THE END"}}}
	}
	return &EndingScene{
		game:                 game,
		field:                field,
		pages:                pages,
		creditScrollDistance: creditRollDistance(pages),
		creditDuration:       creditFallbackMusicDuration,
		phase:                endingPhaseCredits,
	}
}

// desiredBGM starts the ending track as the transition reveals the credits.
func (s *EndingScene) desiredBGM(transitionDuration float64) (string, float64, bool) {
	s.creditMusicOffset = transitionDuration
	return bgmEnding, 2.0, false
}

func creditRollDistance(pages []CreditPage) float64 {
	lineCount := 0
	sectionCount := 0
	for _, page := range pages {
		if len(page.Lines) == 0 {
			continue
		}
		lineCount += len(page.Lines)
		sectionCount++
	}
	sectionGaps := max(sectionCount-1, 0)
	contentHeight := float64(lineCount)*creditLineGap + float64(sectionGaps)*creditSectionGap
	return float64(gameHeight) + contentHeight
}

func creditScrollProgress(elapsed, duration float64) float64 {
	if duration <= 0 {
		return 1
	}
	progress := elapsed / duration
	return max(0, min(progress, 1))
}

func (s *EndingScene) Update(dt float64) Scene {
	if s == nil {
		return s
	}
	if s.nextScene != nil {
		return s.nextScene
	}

	if s.phase == endingPhaseCredits {
		if s.skipConfirm {
			s.updateSkipConfirm()
			return s
		}
		if isSkipKeyPressed() || isSkipHintJustPressed(s.game) {
			s.skipConfirm = true
			s.skipConfirmIdx = 0
			if s.game != nil {
				if s.game.Audio != nil {
					s.game.Audio.PauseBGM()
					s.game.Audio.PlaySEByKey("decide")
				}
			}
			return s
		}
	}

	switch s.phase {
	case endingPhaseCredits:
		s.updateCredits(dt)
	case endingPhaseLogo:
		s.logoElapsed += dt
		if !s.endingBGMStopped && s.logoElapsed >= endingLogoDuration-endingBGMFinishBeforeLogo {
			if s.game != nil && s.game.Audio != nil {
				s.game.Audio.StopBGM()
			}
			s.endingBGMStopped = true
		}
		if s.logoElapsed >= endingLogoDuration {
			s.enterAskSave()
		}
	case endingPhaseAskSave:
		s.updateAskSave()
	case endingPhaseSaveSlot:
		s.updateSaveSlot()
	case endingPhaseSlotConfirm:
		s.updateSlotConfirm()
	case endingPhaseSaveDone:
		s.updateSaveDone()
	}
	return s
}

func (s *EndingScene) updateCredits(dt float64) {
	playbackAvailable := false
	if s.game != nil && s.game.Audio != nil {
		if position, musicDuration, ok := s.game.Audio.BGMPlaybackInfo(bgmEnding); ok {
			s.creditDuration = musicDuration.Seconds() - s.creditMusicOffset
			s.creditElapsed = max(0, min(position.Seconds()-s.creditMusicOffset, s.creditDuration))
			s.creditPlaybackStarted = s.creditDuration > 0
			playbackAvailable = s.creditPlaybackStarted
		}
	}
	if !playbackAvailable && s.creditPlaybackStarted {
		s.creditElapsed = min(s.creditElapsed+dt, s.creditDuration)
	} else if !playbackAvailable {
		if !s.creditPlaybackWarned {
			loadTrace("エンディング曲の再生位置を取得できないため、スタッフロールを%.0f秒で再生", creditFallbackMusicDuration)
			s.creditPlaybackWarned = true
		}
		s.creditDuration = creditFallbackMusicDuration
		s.creditFallbackElapsed += dt
		s.creditElapsed = min(s.creditFallbackElapsed, s.creditDuration)
	}

	remaining := s.creditDuration - s.creditElapsed
	if remaining <= endingLogoDuration-endingBGMFinishBeforeLogo {
		s.enterEndingLogo()
	}
}

func (s *EndingScene) updateSkipConfirm() {
	if isMenuUpPressed() || isMenuDownPressed() {
		s.skipConfirmIdx = 1 - s.skipConfirmIdx
		s.game.Audio.PlaySEByKey("cursor")
	}
	if isEscapePressed() {
		s.skipConfirm = false
		s.resumeCredits()
		s.game.Audio.PlaySEByKey("cancel")
		return
	}
	tappedIdx, tappedOk := hitTestConfirmDialog(confirmImageOffsetX)
	tapped := tapSelectOrConfirm(tappedIdx, tappedOk, &s.skipConfirmIdx, s.game.Audio)
	if !tappedOk && unrelatedTapOutsideRects(confirmDialogRect()) {
		s.skipConfirm = false
		s.resumeCredits()
		s.game.Audio.PlaySEByKey("cancel")
		return
	}
	if !isConfirmKeyPressed() && !tapped {
		return
	}
	if s.skipConfirmIdx == 1 {
		s.skipConfirm = false
		s.resumeCredits()
		s.game.Audio.PlaySEByKey("cancel")
		return
	}
	s.game.Audio.PlaySEByKey("decide")
	s.skipConfirm = false
	s.game.Audio.StopBGM()
	s.endingBGMStopped = true
	s.enterEndingLogo()
}

func (s *EndingScene) resumeCredits() {
	if s.game != nil && s.game.Audio != nil {
		s.game.Audio.ResumeBGM()
	}
}

func (s *EndingScene) enterEndingLogo() {
	s.phase = endingPhaseLogo
	s.logoElapsed = 0
}

func (s *EndingScene) enterAskSave() {
	s.phase = endingPhaseAskSave
	s.askSaveIndex = 0
	lockDialogInput(&s.inputLockTicks)
}

func (s *EndingScene) updateAskSave() {
	if consumeDialogInputLock(&s.inputLockTicks) {
		return
	}
	if isMenuUpPressed() || isMenuDownPressed() {
		s.askSaveIndex = 1 - s.askSaveIndex
		s.game.Audio.PlaySEByKey("cursor")
	}
	tappedIdx, tappedOk := hitTestConfirmDialog(confirmImageOffsetX)
	tapped := tapSelectOrConfirm(tappedIdx, tappedOk, &s.askSaveIndex, s.game.Audio)
	if !isConfirmKeyPressed() && !tapped {
		return
	}
	if s.askSaveIndex == 1 {
		s.game.Audio.PlaySEByKey("cancel")
		s.nextScene = NewTitleScene(s.game)
		return
	}
	s.game.Audio.PlaySEByKey("decide")
	s.reloadSlotData()
	s.slotIndex = 0
	s.slotScrollTop = 0
	s.phase = endingPhaseSaveSlot
}

func (s *EndingScene) reloadSlotData() {
	for i := range maxSaveSlots {
		s.slotData[i], _ = LoadGame(i + 1)
		s.game.loadThumb(i+1, func(img *ebiten.Image) { s.slotThumbs[i] = img })
	}
}

func (s *EndingScene) updateSaveSlot() {
	if isMenuUpPressed() {
		s.slotIndex = (s.slotIndex - 1 + maxSaveSlots) % maxSaveSlots
		s.slotScrollTop = clampSlotScrollTop(s.slotScrollTop, s.slotIndex)
		s.game.Audio.PlaySEByKey("cursor")
	}
	if isMenuDownPressed() {
		s.slotIndex = (s.slotIndex + 1) % maxSaveSlots
		s.slotScrollTop = clampSlotScrollTop(s.slotScrollTop, s.slotIndex)
		s.game.Audio.PlaySEByKey("cursor")
	}
	if isEscapePressed() {
		s.game.Audio.PlaySEByKey("cancel")
		s.phase = endingPhaseAskSave
		s.askSaveIndex = 0
		return
	}

	cardH := float64(s.game.SaveThumbFrameImg.Bounds().Dy())

	barX, barY, barW, barH := slotScrollBarRect(slotCardStartX-80, slotCardStartY)
	if scrollBarMoveAmt := s.slotScrollBarDrag.step(barX, barY, barW, barH); scrollBarMoveAmt != 0 {
		if moveRange := slotScrollBarMoveRange(); moveRange > 0 {
			s.slotScrollBarDrag.stepScrollTop(-scrollBarMoveAmt/moveRange*float64(maxSlotScrollTop)*cardH, cardH, &s.slotScrollTop, maxSlotScrollTop, &s.slotDragAccum)
		}
	}
	scrollBarHeld := s.slotScrollBarDrag.active

	if _, wheelY := ebiten.Wheel(); wheelY != 0 && !scrollBarHeld {
		s.slotDrag.stepScrollTop(wheelY*wheelScrollPxPerNotch, cardH, &s.slotScrollTop, maxSlotScrollTop, &s.slotDragAccum)
	}

	dragArea := cardH * slotsPerPageView
	dragRight := slotScrollBarX(slotCardStartX-80) - scrollBarTouchPad
	moveAmt := s.slotDrag.step(0, slotCardStartY, dragRight, dragArea)

	tapped := false
	if s.slotDrag.justTapped {
		if idx, ok := hitTestSlotList(s.game, s.slotDrag.tapX, s.slotDrag.tapY, s.slotScrollTop, slotCardStartX-80, slotCardStartY); ok {
			s.slotIndex = idx
			tapped = true
		}
	}

	if !scrollBarHeld {
		s.slotDrag.stepScrollTop(moveAmt, cardH, &s.slotScrollTop, maxSlotScrollTop, &s.slotDragAccum)
	}

	if !isConfirmKeyPressed() && !tapped {
		return
	}
	s.game.Audio.PlaySEByKey("decide")
	s.pendingSlot = s.slotIndex + 1
	s.confirmIndex = 0
	s.phase = endingPhaseSlotConfirm
	lockDialogInput(&s.inputLockTicks)
}

func (s *EndingScene) updateSlotConfirm() {
	if consumeDialogInputLock(&s.inputLockTicks) {
		return
	}
	if isMenuUpPressed() || isMenuDownPressed() {
		s.confirmIndex = 1 - s.confirmIndex
		s.game.Audio.PlaySEByKey("cursor")
	}
	if isEscapePressed() {
		s.game.Audio.PlaySEByKey("cancel")
		s.phase = endingPhaseSaveSlot
		return
	}
	tappedIdx, tappedOk := hitTestConfirmDialog(confirmImageOffsetX)
	tapped := tapSelectOrConfirm(tappedIdx, tappedOk, &s.confirmIndex, s.game.Audio)
	if !isConfirmKeyPressed() && !tapped {
		return
	}
	if s.confirmIndex == 1 {
		s.game.Audio.PlaySEByKey("cancel")
		s.phase = endingPhaseSaveSlot
		return
	}
	s.game.Audio.PlaySEByKey("decide")

	slot := s.pendingSlot
	if s.field == nil {
		s.saveResultMsg = "セーブに失敗しました（保存元のデータが見つかりません）"
	} else {
		globalActiveFieldInstanceForSave = s.field
		err := SaveGame(slot, s.field.currentMap, s.field.px, s.field.py, s.game.PlayerHP)
		s.game.saveThumbToFile(slot)
		if err != nil {
			s.saveResultMsg = fmt.Sprintf("セーブに失敗しました（%v）", err)
		} else {
			s.saveResultMsg = "セーブしました"
		}
	}
	s.saveDoneIndex = 0
	s.phase = endingPhaseSaveDone
	lockDialogInput(&s.inputLockTicks)
}

func (s *EndingScene) updateSaveDone() {
	if consumeDialogInputLock(&s.inputLockTicks) {
		return
	}
	tapped := len(justPressedTouchPoints()) > 0
	if !isConfirmKeyPressed() && !tapped {
		return
	}
	s.game.Audio.PlaySEByKey("decide")
	s.nextScene = NewTitleScene(s.game)
}

func (s *EndingScene) Draw(screen *ebiten.Image) {
	if s == nil {
		return
	}
	screen.Fill(color.RGBA{0, 0, 0, 255})

	switch s.phase {
	case endingPhaseAskSave:
		drawConfirmDialog(screen, s.game, "クリアデータを保存しますか？", s.askSaveIndex, confirmImageOffsetX)
		return
	case endingPhaseLogo:
		s.drawEndingLogo(screen)
		return
	case endingPhaseSaveSlot:
		drawSlotList(screen, s.game, s.slotIndex, s.slotData, s.slotThumbs, s.slotScrollTop, slotCardStartX-80, slotCardStartY, s.slotDragAccum)
		return
	case endingPhaseSlotConfirm:
		drawSlotList(screen, s.game, s.slotIndex, s.slotData, s.slotThumbs, s.slotScrollTop, slotCardStartX-80, slotCardStartY, s.slotDragAccum)
		drawConfirmDialog(screen, s.game, "セーブしますか？", s.confirmIndex, confirmImageOffsetX)
		return
	case endingPhaseSaveDone:
		drawSlotList(screen, s.game, s.slotIndex, s.slotData, s.slotThumbs, s.slotScrollTop, slotCardStartX-80, slotCardStartY, s.slotDragAccum)
		drawConfirmDialog(screen, s.game, s.saveResultMsg, 0, confirmImageOffsetX, false)
		return
	}

	if s.phase == endingPhaseCredits {
		drawSkipHint(screen, s.game)
		progress := creditScrollProgress(s.creditElapsed, s.creditDuration)
		scrollOffset := s.creditScrollDistance * progress
		alpha := 1.0
		col := color.RGBA{255, 255, 255, uint8(alpha * 255)}
		lineY := float64(gameHeight) - scrollOffset
		for _, page := range s.pages {
			for _, line := range page.Lines {
				op := &text.DrawOptions{}
				op.GeoM.Translate(float64(gameWidth)/2, lineY)
				op.PrimaryAlign = text.AlignCenter
				op.ColorScale.ScaleWithColor(col)
				text.Draw(screen, line, s.game.FontFace(creditFontSize), op)
				lineY += creditLineGap
			}
			if len(page.Lines) > 0 {
				lineY += creditSectionGap
			}
		}
	}
	if s.skipConfirm {
		drawConfirmDialog(screen, s.game,
			"エンディングをスキップしますか？",
			s.skipConfirmIdx, confirmImageOffsetX)
	}
}

func (s *EndingScene) drawEndingLogo(screen *ebiten.Image) {
	if s.game == nil || s.game.TitleBgImg == nil {
		if !s.logoMissingWarned {
			loadTrace("エンディングロゴを表示できません: assets/images/title/title_bg.png が未読み込みです")
			s.logoMissingWarned = true
		}
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.ColorScale.ScaleAlpha(float32(endingLogoAlpha(s.logoElapsed)))
	screen.DrawImage(s.game.TitleBgImg, op)
}

func endingLogoAlpha(elapsed float64) float64 {
	if elapsed < endingLogoFadeInDuration {
		return max(0, elapsed/endingLogoFadeInDuration)
	}
	if elapsed > endingLogoFadeOutStart {
		return max(0, (endingLogoDuration-elapsed)/endingLogoFadeOutDuration)
	}
	return 1
}
