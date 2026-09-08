package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	fadeTimeNewGame  = 1.5
	fadeTimeContinue = 1.0
)

type TitleScene struct {
	game           *Game
	menuIndex      int
	hasSaveFile    bool
	confirmExit    bool // ← 追加：終了確認ダイアログを表示中か
	exitConfirmIdx int  // ← 追加：0=はい, 1=いいえ
}

func NewTitleScene(game *Game) *TitleScene {
	hasSave := false
	for i := 1; i <= maxSaveSlots; i++ {
		if _, err := os.Stat(saveFilePath(i)); err == nil {
			hasSave = true
			break
		}
	}

	initialIndex := 0
	if hasSave {
		initialIndex = 1
	}

	return &TitleScene{
		game:        game,
		menuIndex:   initialIndex,
		hasSaveFile: hasSave,
	}
}

func (s *TitleScene) Update(dt float64) Scene {
	if s == nil {
		return s
	}

	if s.confirmExit {
		if isMenuUpPressed() || isMenuDownPressed() {
			s.exitConfirmIdx = 1 - s.exitConfirmIdx
		}
		if isEscapePressed() {
			s.confirmExit = false
			return s
		}
		if isConfirmKeyPressed() {
			if s.exitConfirmIdx == 0 {
				os.Exit(0)
			}
			s.confirmExit = false
		}
		return s
	}

	if isMenuDownPressed() {
		s.menuIndex = (s.menuIndex + 1) % 3
	}
	if isMenuUpPressed() {
		s.menuIndex = (s.menuIndex - 1 + 3) % 3
	}

	if isConfirmKeyPressed() {
		if s.menuIndex == 0 {
			for i := 0; i < 4; i++ {
				s.game.PlayerHP[i] = s.game.PlayerMaxHP[i]
				s.game.PlayerMP[i] = s.game.PlayerMaxMP[i]
			}
			s.game.TotalPlayTime = 0
			field, err := NewRoomScene(s.game, "assets/maps/School_Map_1.tmj", 0, 0, "start_point", 0)
			if err != nil {
				return s
			}
			s.game.ChangeSceneWithFade(field, fadeTimeNewGame)
			return s

		} else if s.menuIndex == 1 && s.hasSaveFile {
			return NewLoadSlotScene(s.game, s)
		} else if s.menuIndex == 2 {
			s.confirmExit = true
			s.exitConfirmIdx = 1
		}
	}
	return s
}

func (s *TitleScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{10, 10, 30, 255})

	titleOp := &text.DrawOptions{}
	titleOp.GeoM.Translate(float64(gameWidth)/2, 160)
	titleOp.PrimaryAlign = text.AlignCenter
	titleOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, "七不思議討滅録", s.game.FontFace(15), titleOp)

	newGameText := "はじめから"
	if s.menuIndex == 0 {
		newGameText = "▶ はじめから"
	}
	opt1Op := &text.DrawOptions{}
	opt1Op.GeoM.Translate(float64(gameWidth)/2, 260)
	opt1Op.PrimaryAlign = text.AlignCenter
	if s.menuIndex == 0 {
		opt1Op.ColorScale.ScaleWithColor(uiColorSelect)
	} else {
		opt1Op.ColorScale.ScaleWithColor(uiColorText)
	}
	text.Draw(screen, newGameText, s.game.FontFace(15), opt1Op)

	loadGameText := "つづきから"
	opt2Op := &text.DrawOptions{}
	opt2Op.GeoM.Translate(float64(gameWidth)/2, 295)
	opt2Op.PrimaryAlign = text.AlignCenter
	if !s.hasSaveFile {
		opt2Op.ColorScale.ScaleWithColor(uiColorText)
	} else {
		if s.menuIndex == 1 {
			loadGameText = "▶ つづきから"
			opt2Op.ColorScale.ScaleWithColor(uiColorSelect)
		} else {
			opt2Op.ColorScale.ScaleWithColor(uiColorText)
		}
	}
	text.Draw(screen, loadGameText, s.game.FontFace(15), opt2Op)

	exitText := "ゲームを終了する"
	opt3Op := &text.DrawOptions{}
	opt3Op.GeoM.Translate(float64(gameWidth)/2, 330)
	opt3Op.PrimaryAlign = text.AlignCenter
	if s.menuIndex == 2 {
		exitText = "▶ ゲームを終了する"
		opt3Op.ColorScale.ScaleWithColor(uiColorSelect)
	} else {
		opt3Op.ColorScale.ScaleWithColor(uiColorText)
	}
	text.Draw(screen, exitText, s.game.FontFace(15), opt3Op)

	creditOp := &text.DrawOptions{}
	creditOp.GeoM.Translate(float64(gameWidth)/2, 350)
	creditOp.PrimaryAlign = text.AlignCenter
	creditOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, "(C) 2026 Project sitikai", s.game.FontFace(15), creditOp)

	if s.confirmExit {
		drawConfirmDialog(screen, s.game, "ゲームを終了しますか？", s.exitConfirmIdx, confirmImageOffsetX)
	}
}

// ---------------------------------------------------------------------------
// SaveData
// ---------------------------------------------------------------------------

type SaveData struct {
	SlotID        int     `json:"slot_id"`
	LocationName  string  `json:"location_name"`
	CurrentMap    string  `json:"current_map"`
	PlayerX       float64 `json:"player_x"`
	PlayerY       float64 `json:"player_y"`
	PlayerDir     int     `json:"player_dir"`
	PlayerHP      [4]int  `json:"player_hp"`
	PlayerMaxHP   [4]int  `json:"player_max_hp"`
	PlayerMP      [4]int  `json:"player_mp"`
	PlayerMaxMP   [4]int  `json:"player_max_mp"`
	PlayerAtk     [4]int  `json:"player_atk"`
	PlayerLv      [4]int  `json:"player_lv"`
	PlayerEXP     [4]int  `json:"player_exp"`
	PlayerNextEXP [4]int  `json:"player_next_exp"`
	PlayTime      float64 `json:"play_time"`
	SavedAt       string  `json:"saved_at"`
}

func saveFilePath(slot int) string {
	return fmt.Sprintf("save_%d.json", slot)
}

func thumbFilePath(slot int) string {
	return fmt.Sprintf("save_thumb_%d.png", slot)
}

func LoadGame(slot int) (*SaveData, error) {
	file, err := os.ReadFile(saveFilePath(slot))
	if err != nil {
		return nil, err
	}
	var data SaveData
	if err := json.Unmarshal(file, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func LoadThumb(slot int) *ebiten.Image {
	img, _, err := ebitenutil.NewImageFromFile(thumbFilePath(slot))
	if err != nil {
		return nil
	}
	return img
}

// ---------------------------------------------------------------------------
// LoadSlotScene
// ---------------------------------------------------------------------------

type LoadSlotScene struct {
	game       *Game
	backScene  Scene
	slotIndex  int
	slotData   [maxSaveSlots]*SaveData
	slotThumbs [maxSaveSlots]*ebiten.Image
}

func NewLoadSlotScene(game *Game, backScene Scene) *LoadSlotScene {
	s := &LoadSlotScene{
		game:      game,
		backScene: backScene,
		slotIndex: 0,
	}
	for i := 0; i < maxSaveSlots; i++ {
		s.slotData[i], _ = LoadGame(i + 1)
		s.slotThumbs[i] = LoadThumb(i + 1)
	}
	for i := 0; i < maxSaveSlots; i++ {
		if s.slotData[i] != nil {
			s.slotIndex = i
			break
		}
	}
	return s
}

func (s *LoadSlotScene) Update(dt float64) Scene {
	if isEscapePressed() {
		return s.backScene
	}
	if isMenuUpPressed() {
		s.slotIndex = (s.slotIndex - 1 + maxSaveSlots) % maxSaveSlots
	}
	if isMenuDownPressed() {
		s.slotIndex = (s.slotIndex + 1) % maxSaveSlots
	}
	if isConfirmKeyPressed() {
		d := s.slotData[s.slotIndex]
		if d == nil {
			return s
		}
		s.game.TotalPlayTime = d.PlayTime
		s.game.PlayerHP = d.PlayerHP
		s.game.PlayerMaxHP = d.PlayerMaxHP
		s.game.PlayerMP = d.PlayerMP
		s.game.PlayerMaxMP = d.PlayerMaxMP
		s.game.PlayerAtk = d.PlayerAtk
		s.game.PlayerLv = d.PlayerLv
		s.game.PlayerEXP = d.PlayerEXP
		s.game.PlayerNextEXP = d.PlayerNextEXP

		field, err := NewRoomScene(s.game, d.CurrentMap, d.PlayerX, d.PlayerY, "", d.PlayerDir)
		if err != nil {
			return s
		}
		s.game.ChangeSceneWithFade(field, fadeTimeContinue)
		return s
	}
	return s
}

func (s *LoadSlotScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{10, 10, 30, 255})
	drawSlotList(screen, s.game, s.slotIndex, s.slotData, s.slotThumbs, false, slotsPerPageView/2, slotCardStartX-80, slotCardStartY)

	// メニュー画面と同じ位置に説明文を表示
	desc := menuCommandDescriptions["ロード"]
	if desc != "" {
		x := float64(gameWidth) - menuDescOffsetX
		y := float64(gameHeight) - menuDescOffsetY
		descOp := &text.DrawOptions{}
		descOp.GeoM.Translate(x, y)
		descOp.PrimaryAlign = text.AlignEnd
		descOp.ColorScale.ScaleWithColor(uiColorText)
		text.Draw(screen, desc, s.game.FontFace(14), descOp)
	}
}
