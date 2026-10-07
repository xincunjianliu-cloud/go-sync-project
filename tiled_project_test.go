//go:build !js

package main

import (
	"encoding/json"
	"io/fs"
	"os"
	"path"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const tiledProjectPath = "rpg.tiled-project"

// rpg.tiled-project が tiled_classes.go とゲームのデータ(アイテム・敵・曲・
// 会話・戦闘背景)に合っていること。合っていなければ作り直す:
//
//	PowerShell: $env:UPDATE_TILED_PROJECT=1; go test -run TestTiledProjectUpToDate .; Remove-Item Env:UPDATE_TILED_PROJECT
func TestTiledProjectUpToDate(t *testing.T) {
	ensureStoryDialoguesLoaded()
	want := tiledProjectJSON()
	wantBytes, err := json.MarshalIndent(want, "", "    ")
	if err != nil {
		t.Fatal(err)
	}
	if os.Getenv("UPDATE_TILED_PROJECT") != "" {
		if err := os.WriteFile(tiledProjectPath, append(wantBytes, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("%s を作り直しました", tiledProjectPath)
		return
	}
	data, err := os.ReadFile(tiledProjectPath)
	if err != nil {
		t.Fatalf("%s がありません: %v", tiledProjectPath, err)
	}
	// Tiledが保存し直すと並びや書式が変わるので、中身(JSONの値)で比べる。
	var got, wantAny map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("%s が読めません: %v", tiledProjectPath, err)
	}
	_ = json.Unmarshal(wantBytes, &wantAny)
	for _, key := range []string{"propertyTypes", "commands", "folders"} {
		if !reflect.DeepEqual(got[key], wantAny[key]) {
			t.Errorf("%s の %s が古いか、Tiledで書き換えられています。テストのコメントにあるコマンドで作り直してください", tiledProjectPath, key)
		}
	}
}

func listAssetDir(dir string) ([]string, error) {
	entries, err := fs.ReadDir(embeddedAssets, dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// classEnums はTiledで一覧から選べる値。ゲームのデータから作る。
func classEnums() []struct {
	Name   string
	Values []string
	Flags  bool
} {
	var items, enemies, bgms, stories, bgs []string
	for _, it := range ItemDatabase {
		items = append(items, it.Name)
	}
	for _, e := range EnemyDatabase {
		enemies = append(enemies, e.Name)
	}
	for _, b := range bgmLabels {
		bgms = append(bgms, b.Label)
	}
	for id := range storyDialogues {
		stories = append(stories, id)
	}
	sort.Strings(stories)
	if names, err := listBattleBgNames(); err == nil {
		bgs = names
	}
	bosses := []string{enumNone}
	for i := 1; i <= len(Game{}.BossDefeatedFlags); i++ {
		bosses = append(bosses, strconv.Itoa(i))
	}
	return []struct {
		Name   string
		Values []string
		Flags  bool
	}{
		{"アイテム", append([]string{enumUnset}, items...), false},
		{"敵", enemies, true},
		{"曲", bgms, false},
		{"戦闘背景", append([]string{enumDefault}, bgs...), false},
		{"会話", append([]string{enumNone}, stories...), false},
		{"ボス", bosses, false},
		{"向き", append([]string{enumKeep}, dirLabels...), false},
	}
}

// playCommand はTiledの「このマップで遊ぶ」コマンド(tools/playmap/play.bat)。
// extra は play.bat に渡すゲームの追加オプション。
func playCommand(name, shortcut, extra string) map[string]any {
	return map[string]any{
		"name":              name,
		"command":           "cmd",
		"arguments":         `/c tools\playmap\play.bat` + extra + ` %mapfile`,
		"workingDirectory":  "%projectpath",
		"shortcut":          shortcut,
		"showOutput":        true,
		"saveBeforeExecute": true,
		"enabled":           true,
	}
}

// tiledProjectJSON は rpg.tiled-project の中身(JSONにする前の形)。
func tiledProjectJSON() map[string]any {
	var types []any
	id := 1
	for _, e := range classEnums() {
		types = append(types, map[string]any{
			"id": id, "name": e.Name, "type": "enum",
			"storageType": "string", "values": e.Values, "valuesAsFlags": e.Flags,
		})
		id++
	}
	for _, c := range tiledClasses {
		members := []any{}
		for _, m := range c.Members {
			mm := map[string]any{"name": m.Name, "type": m.Type, "value": m.Default}
			if m.Enum != "" {
				mm["propertyType"] = m.Enum
			}
			members = append(members, mm)
		}
		types = append(types, map[string]any{
			"id": id, "name": c.Name, "type": "class", "useAs": []string{c.UseAs},
			"color": c.Color, "drawFill": true, "members": members,
		})
		id++
	}
	return map[string]any{
		"automappingRulesFile": "",
		"commands": []any{
			playCommand("このマップで遊ぶ(敵なし)", "F5", " -noencounter"),
			playCommand("このマップで遊ぶ(敵あり)", "Shift+F5", ""),
		},
		"extensionsPath": "extensions",
		"folders":        []string{"assets/maps", "assets/tilesets"},
		"properties":     []any{},
		"propertyTypes":  types,
	}
}

func listBattleBgNames() ([]string, error) {
	entries, err := listAssetDir("assets/images/battle/bg")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if strings.EqualFold(path.Ext(e), ".png") {
			names = append(names, strings.TrimSuffix(e, path.Ext(e)))
		}
	}
	sort.Strings(names)
	return names, nil
}
