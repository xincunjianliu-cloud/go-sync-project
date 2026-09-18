package main

import (
	"encoding/json"
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	creditFadeInTime      = 1.0
	creditFirstFadeInTime = 3.0
	creditHoldTime        = 2.5
	creditFadeOutTime     = 1.0
	creditLineGap         = 32.0
	creditFontSize        = 20.0

	endingSkipHoldSeconds = 1.0
)

type creditPageJSON struct {
	Lines []string `json:"lines"`
}
type creditsFileJSON struct {
	Pages []creditPageJSON `json:"pages"`
}
type CreditPage struct {
	Lines []string
}

func loadCredits(path string) ([]CreditPage, error) {
	data, err := loadAssetBytes(path)
	if err != nil {
		return nil, err
	}
	var f creditsFileJSON
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	pages := make([]CreditPage, 0, len(f.Pages))
	for _, p := range f.Pages {
		pages = append(pages, CreditPage{Lines: p.Lines})
	}
	return pages, nil
}

const (
	endingPhaseFadeIn = iota
	endingPhaseHold
	endingPhaseFadeOut
	endingPhaseAskSave
	endingPhaseSaveSlot
	endingPhaseSlotConfirm
	endingPhaseSaveDone
)

type EndingScene struct {
	game  *Game
	field *FieldScene

	pages     []CreditPage
	pageIndex int

	phase int
	alpha float64

	holdElapsed     float64
	skipHoldElapsed float64

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
	return &EndingScene{game: game, field: field, pages: pages, phase: endingPhaseFadeIn}
}

// desiredBGM はボス撃破後の暗転(fadeTimeBossOut)が明けた瞬間に、余韻を
// 持たせて2秒かけてエンディング曲をフェードインさせる。暗転の長さに
// 関わらずこの秒数は固定でよい(締めの演出なので画面遷移と揃える必要はない)。
func (s *EndingScene) desiredBGM(transitionDuration float64) (string, float64, bool) {
	return bgmEnding, 2.0, false
}

func (s *EndingScene) Update(dt float64) Scene {
	if s == nil {
		return s
	}
	if s.nextScene != nil {
		return s.nextScene
	}

	if s.phase == endingPhaseFadeIn || s.phase == endingPhaseHold || s.phase == endingPhaseFadeOut {
		if isSkipKeyDown() {
			s.skipHoldElapsed += dt
			if s.skipHoldElapsed >= endingSkipHoldSeconds {
				s.game.Audio.PlaySEByKey("decide")
				s.pageIndex = len(s.pages)
				s.enterAskSave()
				return s
			}
		} else {
			s.skipHoldElapsed = 0
		}
	}

	switch s.phase {
	case endingPhaseFadeIn:
		fadeInTime := creditFadeInTime
		if s.pageIndex == 0 {
			fadeInTime = creditFirstFadeInTime
		}
		s.alpha += dt / fadeInTime
		if s.alpha >= 1.0 {
			s.alpha = 1.0
			s.phase = endingPhaseHold
			s.holdElapsed = 0
		}
	case endingPhaseHold:
		s.holdElapsed += dt
		if s.holdElapsed >= creditHoldTime {
			s.phase = endingPhaseFadeOut
		}
	case endingPhaseFadeOut:
		s.alpha -= dt / creditFadeOutTime
		if s.alpha <= 0 {
			s.alpha = 0
			s.pageIndex++
			if s.pageIndex >= len(s.pages) {
				s.enterAskSave()
			} else {
				s.phase = endingPhaseFadeIn
			}
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
	tappedIdx, tappedOk := hitTestConfirmDialog(s.game, confirmImageOffsetX)
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
	for i := 0; i < maxSaveSlots; i++ {
		s.slotData[i], _ = LoadGame(i + 1)
		s.slotThumbs[i] = LoadThumb(i + 1)
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
	tappedIdx, tappedOk := hitTestConfirmDialog(s.game, confirmImageOffsetX)
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
	case endingPhaseSaveSlot:
		drawSlotList(screen, s.game, s.slotIndex, s.slotData, s.slotThumbs, true, s.slotScrollTop, slotCardStartX-80, slotCardStartY, s.slotDragAccum)
		return
	case endingPhaseSlotConfirm:
		drawSlotList(screen, s.game, s.slotIndex, s.slotData, s.slotThumbs, true, s.slotScrollTop, slotCardStartX-80, slotCardStartY, s.slotDragAccum)
		drawConfirmDialog(screen, s.game, "セーブしますか？", s.confirmIndex, confirmImageOffsetX)
		return
	case endingPhaseSaveDone:
		drawSlotList(screen, s.game, s.slotIndex, s.slotData, s.slotThumbs, true, s.slotScrollTop, slotCardStartX-80, slotCardStartY, s.slotDragAccum)
		drawConfirmDialog(screen, s.game, s.saveResultMsg, 0, confirmImageOffsetX, false)
		return
	}

	if s.phase == endingPhaseFadeIn || s.phase == endingPhaseHold || s.phase == endingPhaseFadeOut {
		drawSkipHint(screen, s.game, s.skipHoldElapsed, endingSkipHoldSeconds)
	}

	if s.pageIndex >= len(s.pages) {
		return
	}
	page := s.pages[s.pageIndex]
	totalH := float64(len(page.Lines)) * creditLineGap
	startY := float64(gameHeight)/2 - totalH/2
	col := color.RGBA{255, 255, 255, uint8(s.alpha * 255)}

	for i, line := range page.Lines {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(gameWidth)/2, startY+float64(i)*creditLineGap)
		op.PrimaryAlign = text.AlignCenter
		op.ColorScale.ScaleWithColor(col)
		text.Draw(screen, line, s.game.FontFace(creditFontSize), op)
	}
}
