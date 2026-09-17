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
	fadeTimeNewGame  = 1.5
	fadeTimeContinue = 1.0
)

type TitleScene struct {
	game           *Game
	menuIndex      int
	hasSaveFile    bool
	confirmExit    bool
	exitConfirmIdx int
	inputLockTicks int
}

func NewTitleScene(game *Game) *TitleScene {
	hasSave := false
	for i := 1; i <= maxSaveSlots; i++ {
		if runtimeFileExists(saveFilePath(i)) {
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

// desiredBGM はゲームオーバーからタイトルに戻る場合など、画面フェードを
// 伴う遷移でタイトルBGMをフェードインさせる。起動直後の初回表示は
// game.go側で直接PlayBGMFadeInしているため、ここは通らない。
func (s *TitleScene) desiredBGM(transitionDuration float64) (string, float64, bool) {
	return bgmTitle, transitionDuration, false
}

func (s *TitleScene) Update(dt float64) Scene {
	if s == nil {
		return s
	}

	if s.confirmExit {
		if consumeDialogInputLock(&s.inputLockTicks) {
			return s
		}
		if isMenuUpPressed() || isMenuDownPressed() {
			s.exitConfirmIdx = 1 - s.exitConfirmIdx
			s.game.Audio.PlaySEByKey("cursor")
		}
		if isEscapePressed() {
			s.game.Audio.PlaySEByKey("cancel")
			s.confirmExit = false
			return s
		}
		tappedIdx, tappedOk := hitTestConfirmDialog(s.game, confirmImageOffsetX)
		confirmTapped := tapSelectOrConfirm(tappedIdx, tappedOk, &s.exitConfirmIdx, s.game.Audio)
		if isConfirmKeyPressed() || confirmTapped {
			if s.exitConfirmIdx == 0 {
				os.Exit(0)
			}
			s.game.Audio.PlaySEByKey("cancel")
			s.confirmExit = false
		}
		return s
	}

	if isMenuDownPressed() {
		s.menuIndex = (s.menuIndex + 1) % 3
		s.game.Audio.PlaySEByKey("cursor")
	}
	if isMenuUpPressed() {
		s.menuIndex = (s.menuIndex - 1 + 3) % 3
		s.game.Audio.PlaySEByKey("cursor")
	}

	tapped := false
	if idx, ok := s.hitTestMainMenu(); ok {
		s.menuIndex = idx
		tapped = true
	}

	if isConfirmKeyPressed() || tapped {
		if s.menuIndex == 0 {
			if !s.game.heavyAssetsReady || s.game.heavyAssetsErr != nil {
				s.game.Audio.PlaySEByKey("error")
				return s
			}
			s.game.Audio.PlaySEByKey("decide")
			s.game.ResetForNewGame()
			field, err := NewRoomScene(s.game, "assets/maps/School_Map_1.tmj", 0, 0, "start_point", 0)
			if err != nil {
				return s
			}
			s.game.ChangeSceneWithFade(field, fadeTimeNewGame)
			return s

		} else if s.menuIndex == 1 && s.hasSaveFile {
			if !s.game.heavyAssetsReady || s.game.heavyAssetsErr != nil {
				s.game.Audio.PlaySEByKey("error")
				return s
			}
			s.game.Audio.PlaySEByKey("decide")
			return NewLoadSlotScene(s.game, s)
		} else if s.menuIndex == 2 {
			s.game.Audio.PlaySEByKey("decide")
			s.confirmExit = true
			s.exitConfirmIdx = 1
			lockDialogInput(&s.inputLockTicks)
		}
	}
	return s
}

func (s *TitleScene) hitTestMainMenu() (int, bool) {
	face := s.game.FontFace(15)
	centerX := float64(gameWidth) / 2
	rects := []tapRect{
		centeredTextRect(centerX, 260, "はじめから", face, 30),
		centeredTextRect(centerX, 295, "つづきから", face, 30),
		centeredTextRect(centerX, 330, "ゲームを終了する", face, 30),
	}
	return hitTestTapRects(rects)
}

func (s *TitleScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{10, 10, 30, 255})

	titleOp := &text.DrawOptions{}
	titleOp.GeoM.Translate(float64(gameWidth)/2, 160)
	titleOp.PrimaryAlign = text.AlignCenter
	titleOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, "七不思議討滅録", s.game.FontFace(15), titleOp)

	titleMenuFace := s.game.FontFace(15)
	drawTitleMenuOption := func(label string, y float64, selected bool) {
		centerX := float64(gameWidth) / 2
		col := uiColorText
		if selected {
			col = uiColorSelect
			labelW := text.Advance(label, titleMenuFace)
			arrowOp := &text.DrawOptions{}
			arrowOp.GeoM.Translate(centerX-labelW/2-text.Advance("▶ ", titleMenuFace), y)
			arrowOp.ColorScale.ScaleWithColor(col)
			text.Draw(screen, "▶", titleMenuFace, arrowOp)
		}
		op := &text.DrawOptions{}
		op.GeoM.Translate(centerX, y)
		op.PrimaryAlign = text.AlignCenter
		op.ColorScale.ScaleWithColor(col)
		text.Draw(screen, label, titleMenuFace, op)
	}

	drawTitleMenuOption("はじめから", 260, s.menuIndex == 0)
	drawTitleMenuOption("つづきから", 295, s.hasSaveFile && s.menuIndex == 1)
	drawTitleMenuOption("ゲームを終了する", 330, s.menuIndex == 2)

	creditOp := &text.DrawOptions{}
	creditOp.GeoM.Translate(float64(gameWidth)/2, 350)
	creditOp.PrimaryAlign = text.AlignCenter
	creditOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, "(C) 2026 Project sitikai", s.game.LatinFontFace(15), creditOp)

	if s.game.heavyAssetsErr != nil {
		errOp := &text.DrawOptions{}
		errOp.GeoM.Translate(float64(gameWidth)/2, 390)
		errOp.PrimaryAlign = text.AlignCenter
		errOp.ColorScale.ScaleWithColor(color.RGBA{255, 120, 120, 255})
		text.Draw(screen, "データの読み込みに失敗しました", s.game.FontFace(12), errOp)
	} else if !s.game.heavyAssetsReady {
		hintOp := &text.DrawOptions{}
		hintOp.GeoM.Translate(float64(gameWidth)/2, 390)
		hintOp.PrimaryAlign = text.AlignCenter
		hintOp.ColorScale.ScaleWithColor(uiColorText)
		text.Draw(screen, "データを読み込み中...", s.game.FontFace(12), hintOp)
	}

	if s.confirmExit {
		drawConfirmDialog(screen, s.game, "ゲームを終了しますか？", s.exitConfirmIdx, confirmImageOffsetX)
	}
}

type SaveData struct {
	SlotID            int       `json:"slot_id"`
	LocationName      string    `json:"location_name"`
	CurrentMap        string    `json:"current_map"`
	PlayerX           float64   `json:"player_x"`
	PlayerY           float64   `json:"player_y"`
	PlayerDir         int       `json:"player_dir"`
	PlayerHP          [4]int    `json:"player_hp"`
	PlayerMaxHP       [4]int    `json:"player_max_hp"`
	PlayerMP          [4]int    `json:"player_mp"`
	PlayerMaxMP       [4]int    `json:"player_max_mp"`
	PlayerAtk         [4]int    `json:"player_atk"`
	PlayerMagicAtk    [4]int    `json:"player_magic_atk"`
	PlayerDef         [4]int    `json:"player_def"`
	PlayerMagicDef    [4]int    `json:"player_magic_def"`
	PlayerSpd         [4]int    `json:"player_spd"`
	PlayerLuck        [4]int    `json:"player_luck"`
	PlayerSP          [4]int    `json:"player_sp"`
	PlayerSkillLv     [4][8]int `json:"player_skill_lv"`
	PlayerLv          [4]int    `json:"player_lv"`
	PlayerEXP         [4]int    `json:"player_exp"`
	PlayerNextEXP     [4]int    `json:"player_next_exp"`
	BossDefeatedFlags [4]bool   `json:"boss_defeated_flags"`
	PlayTime          float64   `json:"play_time"`
	SavedAt           string    `json:"saved_at"`

	Inventory            []InventorySlot       `json:"inventory"`
	OpenedChests         map[string]bool       `json:"opened_chests"`
	UnlockedWalls        map[string]bool       `json:"unlocked_walls"`
	Keys                 map[string]int        `json:"keys"`
	RaisedLevers         map[string]bool       `json:"raised_levers"`
	SeenAutoHealMapIntro map[string]bool       `json:"seen_auto_heal_map_intro"`
	BlockPositions       map[string][2]float64 `json:"block_positions"`
	UnlockedBlockDoors   map[string]bool       `json:"unlocked_block_doors"`
}

func saveFilePath(slot int) string {
	return fmt.Sprintf("save_%d.json", slot)
}

func thumbFilePath(slot int) string {
	return fmt.Sprintf("save_thumb_%d.png", slot)
}

func LoadGame(slot int) (*SaveData, error) {
	file, err := readRuntimeFile(saveFilePath(slot))
	if err != nil {
		return nil, err
	}
	var data SaveData
	if err := json.Unmarshal(file, &data); err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(file, &raw); err != nil {
		return nil, err
	}
	if _, ok := raw["player_magic_atk"]; !ok {
		for i := 0; i < partySize; i++ {
			level := data.PlayerLv[i]
			if level < 1 || level > len(PlayerStatsByLevel) {
				level = 1
			}
			st := PlayerStatsByLevel[level-1][i]
			data.PlayerMagicAtk[i] = st.MagicAtk
			data.PlayerDef[i] = st.PhysDef
			data.PlayerMagicDef[i] = st.MagicDef
			data.PlayerSpd[i] = st.Spd
			data.PlayerLuck[i] = st.Luck
		}
	}
	if _, ok := raw["player_skill_lv"]; !ok {
		for i := 0; i < partySize; i++ {
			for j := 0; j < len(data.PlayerSkillLv[i]); j++ {
				data.PlayerSkillLv[i][j] = 1
			}
		}
	}
	return &data, nil
}

func LoadThumb(slot int) *ebiten.Image {
	img, err := loadRuntimeImage(thumbFilePath(slot))
	if err != nil {
		return nil
	}
	return img
}

type LoadSlotScene struct {
	game              *Game
	backScene         Scene
	slotIndex         int
	slotScrollTop     int
	slotData          [maxSaveSlots]*SaveData
	slotThumbs        [maxSaveSlots]*ebiten.Image
	slotDrag          dragScrollState
	slotScrollBarDrag dragScrollState
	slotDragAccum     float64
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
	s.slotScrollTop = clampSlotScrollTop(0, s.slotIndex)
	return s
}

func (s *LoadSlotScene) Update(dt float64) Scene {
	if isEscapePressed() {
		s.game.Audio.PlaySEByKey("cancel")
		return s.backScene
	}
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

	if isConfirmKeyPressed() || tapped {
		d := s.slotData[s.slotIndex]
		if d == nil {
			s.game.Audio.PlaySEByKey("error")
			return s
		}
		s.game.Audio.PlaySEByKey("decide")
		s.game.TotalPlayTime = d.PlayTime
		s.game.PlayerHP = d.PlayerHP
		s.game.PlayerMaxHP = d.PlayerMaxHP
		s.game.PlayerMP = d.PlayerMP
		s.game.PlayerMaxMP = d.PlayerMaxMP
		s.game.PlayerAtk = d.PlayerAtk
		s.game.PlayerMagicAtk = d.PlayerMagicAtk
		s.game.PlayerDef = d.PlayerDef
		s.game.PlayerMagicDef = d.PlayerMagicDef
		s.game.PlayerSpd = d.PlayerSpd
		s.game.PlayerLuck = d.PlayerLuck
		s.game.PlayerSP = d.PlayerSP
		s.game.PlayerSkillLv = d.PlayerSkillLv
		s.game.BossDefeatedFlags = d.BossDefeatedFlags
		s.game.PlayerLv = d.PlayerLv
		s.game.PlayerEXP = d.PlayerEXP
		s.game.PlayerNextEXP = d.PlayerNextEXP
		s.game.Inventory = d.Inventory
		s.game.OpenedChests = d.OpenedChests
		s.game.UnlockedWalls = d.UnlockedWalls
		s.game.Keys = d.Keys
		s.game.RaisedLevers = d.RaisedLevers
		s.game.SeenAutoHealMapIntro = d.SeenAutoHealMapIntro
		s.game.BlockPositions = d.BlockPositions
		s.game.UnlockedBlockDoors = d.UnlockedBlockDoors

		field, err := NewRoomScene(s.game, d.CurrentMap, d.PlayerX, d.PlayerY, "", d.PlayerDir)
		if err != nil {
			return s
		}
		s.game.ChangeSceneWithFade(field, fadeTimeContinue)
		return s
	}

	cardW := float64(s.game.SaveThumbFrameImg.Bounds().Dx())
	cardRect := tapRect{x: slotCardStartX - 80, y: slotCardStartY, w: cardW, h: dragArea}
	barRect := tapRect{x: barX, y: barY, w: barW, h: barH}
	if unrelatedTapOutsideRects(cardRect, barRect) {
		s.game.Audio.PlaySEByKey("cancel")
		return s.backScene
	}
	return s
}

func (s *LoadSlotScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{10, 10, 30, 255})
	drawSlotList(screen, s.game, s.slotIndex, s.slotData, s.slotThumbs, false, s.slotScrollTop, slotCardStartX-80, slotCardStartY, s.slotDragAccum)
	drawBackButton(screen, s.game)

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
