//go:build !js

package main

import (
	"bytes"
	"fmt"
	"image/png"
	"testing"
)

// player_attack_N.png と、battle_sprite_layout.go の行ごとのコマ数・
// スキルの行の表が食い違っていないこと（シートだけ差し替えて表を作り直し
// 忘れると、存在しないマスを切り出してしまう）。
func TestPartySpriteSheetsMatchLayout(t *testing.T) {
	skillLists := [partySize][]SkillDef{Player1Skills, Player2Skills, Player3Skills, Player4Skills}
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
		if cfg.Height != len(rows)*battleSpriteCellH {
			t.Errorf("%s: 高さ%dが行数%d×%dと合いません", path, cfg.Height, len(rows), battleSpriteCellH)
		}
		for r, n := range rows {
			if n <= 0 || n*battleSpriteCellW > cfg.Width {
				t.Errorf("%s: 行%dのコマ数%dが幅%dに収まりません", path, r, n, cfg.Width)
			}
		}
		if len(rows) <= spriteRowHealRecv {
			t.Errorf("%s: 共通アニメの行が足りません（%d行）", path, len(rows))
		}
		// アニメの表はスキル名で引くので、名前が技データと一致していること。
		names := map[string]bool{}
		for _, sk := range skillLists[i] {
			names[sk.Name] = true
			if _, ok := partySkillSpriteRow[i][sk.Name]; !ok {
				t.Errorf("player%d: スキル「%s」のアニメが battle_sprite_layout.go にありません", i+1, sk.Name)
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
