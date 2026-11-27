//go:build !js

package main

import (
	"bytes"
	"fmt"
	"image/png"
	"slices"
	"testing"
)

// player_attack_N.png と、battle_sprite_layout.go の行ごとのコマ数・
// スキルの行の表が食い違っていないことを確認する。
func TestPartySpriteSheetsMatchLayout(t *testing.T) {
	skillLists := [partySize][]SkillDef{Player1Skills, Player2Skills, Player3Skills, Player4Skills}
	wantFrames := [partySize][]int{
		{2, 2, 2, 10, 14, 14, 7, 16, 17, 31, 16, 14, 26},
		{2, 2, 2, 10, 14, 14, 12, 24, 26, 12, 27, 27, 14},
		{2, 2, 2, 10, 14, 14, 10, 17, 16, 10, 10, 16, 16},
		{2, 2, 2, 10, 14, 14, 17, 32, 30, 31, 31, 31, 24},
	}
	for i := range partySize {
		path := fmt.Sprintf("assets/images/battle/player_attack_%d.png", i+1)
		raw, err := embeddedAssets.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		cfg, err := png.DecodeConfig(bytes.NewReader(raw))
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		rows := partySpriteRowFrames[i]
		if !slices.Equal(rows, wantFrames[i]) {
			t.Errorf("%s: 行ごとのコマ数%v、want %v", path, rows, wantFrames[i])
		}
		if len(rows) != spriteRowAttack+1+len(skillLists[i]) {
			t.Errorf("%s: 行数%dが共通アニメとスキル数%dに合いません", path, len(rows), len(skillLists[i]))
		}
		for r, n := range rows {
			if n <= 0 || (n-1)*battleSpriteCellW >= cfg.Width {
				t.Errorf("%s: 行%dのコマ数%dが幅%dに収まりません", path, r, n, cfg.Width)
			}
			if r*battleSpriteCellH >= cfg.Height {
				t.Errorf("%s: 行%dが高さ%dに収まりません", path, r, cfg.Height)
			}
		}
		if len(rows) <= spriteRowHealRecv {
			t.Errorf("%s: 共通アニメの行が足りません（%d行）", path, len(rows))
		}
		// アニメの表はスキル名で引くので、名前が技データと一致していること。
		names := map[string]bool{}
		for skillIdx, sk := range skillLists[i] {
			names[sk.Name] = true
			wantRow := spriteRowAttack + 1 + skillIdx
			if gotRow, ok := partySkillSpriteRow[i][sk.Name]; !ok {
				t.Errorf("player%d: スキル「%s」のアニメが battle_sprite_layout.go にありません", i+1, sk.Name)
			} else if gotRow != wantRow {
				t.Errorf("player%d: スキル「%s」の行%dは、習得順の行%dと異なります", i+1, sk.Name, gotRow, wantRow)
			}
		}
		for name, r := range partySkillSpriteRow[i] {
			if !names[name] {
				t.Errorf("player%d: battle_sprite_layout.go の「%s」はスキルにありません（名前の書き間違い？）", i+1, name)
			}
			if r < 0 || r >= len(rows) {
				t.Errorf("player%d: 「%s」の行%dがシートにありません", i+1, name, r)
				continue
			}
			if cut := partySkillHealSelfFrame[i][name]; cut < 0 || cut >= rows[r] {
				t.Errorf("player%d: 「%s」の区切り%dが行%dのコマ数を超えています", i+1, name, cut, r)
			}
		}
	}
}
