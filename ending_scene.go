package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"

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
	data, err := os.ReadFile(path)
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
	endingPhaseAskSave     // 「保存しますか？」はい/いいえ
	endingPhaseSaveSlot    // スロット一覧から選択
	endingPhaseSlotConfirm // 上書き確認
	endingPhaseSaveDone    // 保存完了メッセージ
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

	slotIndex  int
	slotData   [maxSaveSlots]*SaveData
	slotThumbs [maxSaveSlots]*ebiten.Image

	confirmIndex  int
	pendingSlot   int
	saveResultMsg string
	saveDoneIndex int

	nextScene Scene
}

func NewEndingScene(game *Game, field *FieldScene) *EndingScene {
	pages, err := loadCredits("assets/credits.json")
	if err != nil || len(pages) == 0 {
		fmt.Printf("警告: クレジットデータの読み込みに失敗しました（デフォルト表示に切替）: %v\n", err)
		pages = []CreditPage{{Lines: []string{"THE END"}}}
	}
	return &EndingScene{game: game, field: field, pages: pages, phase: endingPhaseFadeIn}
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
}

func (s *EndingScene) updateAskSave() {
	if isMenuUpPressed() || isMenuDownPressed() {
		s.askSaveIndex = 1 - s.askSaveIndex
	}
	if !isConfirmKeyPressed() {
		return
	}
	if s.askSaveIndex == 1 { // いいえ → タイトルへ
		s.nextScene = NewTitleScene(s.game)
		return
	}
	// はい → スロット選択へ
	s.reloadSlotData()
	s.slotIndex = 0
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
	}
	if isMenuDownPressed() {
		s.slotIndex = (s.slotIndex + 1) % maxSaveSlots
	}
	if isEscapePressed() {
		// スロット選択をキャンセル → 保存するか聞く画面に戻る
		s.phase = endingPhaseAskSave
		s.askSaveIndex = 0
		return
	}
	if !isConfirmKeyPressed() {
		return
	}
	s.pendingSlot = s.slotIndex + 1
	s.confirmIndex = 0
	s.phase = endingPhaseSlotConfirm
}

func (s *EndingScene) updateSlotConfirm() {
	if isMenuUpPressed() || isMenuDownPressed() {
		s.confirmIndex = 1 - s.confirmIndex
	}
	if isEscapePressed() {
		s.phase = endingPhaseSaveSlot
		return
	}
	if !isConfirmKeyPressed() {
		return
	}
	if s.confirmIndex == 1 { // いいえ → スロット一覧に戻る
		s.phase = endingPhaseSaveSlot
		return
	}

	slot := s.pendingSlot
	if s.field == nil {
		s.saveResultMsg = "セーブに失敗しました（保存元のデータが見つかりません）"
	} else {
		// ★修正：SaveGameはglobalActiveFieldInstanceForSave経由で
		// プレイヤー情報を取得するが、この代入が抜けていたため
		// メニューから一度もセーブしていない状態でクリアデータを
		// 保存しようとすると必ず「game instance not found」で
		// 失敗していた。menu_scene.goと同様にここでも設定する。
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
}

func (s *EndingScene) updateSaveDone() {
	if !isConfirmKeyPressed() {
		return
	}
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
		drawSlotList(screen, s.game, s.slotIndex, s.slotData, s.slotThumbs, true, slotsPerPageView/2, slotCardStartX-80, slotCardStartY)
		return
	case endingPhaseSlotConfirm:
		drawSlotList(screen, s.game, s.slotIndex, s.slotData, s.slotThumbs, true, slotsPerPageView/2, slotCardStartX-80, slotCardStartY)
		drawConfirmDialog(screen, s.game, "セーブしますか？", s.confirmIndex, confirmImageOffsetX)
		return
	case endingPhaseSaveDone:
		drawSlotList(screen, s.game, s.slotIndex, s.slotData, s.slotThumbs, true, slotsPerPageView/2, slotCardStartX-80, slotCardStartY)
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
	face := s.game.FontFace(creditFontSize)
	totalH := float64(len(page.Lines)) * creditLineGap
	startY := float64(gameHeight)/2 - totalH/2

	for i, line := range page.Lines {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(gameWidth)/2, startY+float64(i)*creditLineGap)
		op.PrimaryAlign = text.AlignCenter
		op.ColorScale.Scale(1, 1, 1, float32(s.alpha))
		text.Draw(screen, line, face, op)
	}
}
