//go:build !js

package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

const startMapGoPath = "start_map_generated.go"

// マップ設定で「ゲームの最初のマップ」にしたマップが1つだけあり、
// start_map_generated.go の startMapPath と合っていること。合っていなければ
// go run ./tools/update(または UPDATE_TILED_PROJECT=1 でこのテスト)で作り直す。
func TestStartMapUpToDate(t *testing.T) {
	var starts []string
	for _, mapPath := range allMapFiles(t) {
		tmap, err := loadTiledMap(mapPath)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range tmap.Properties {
			if b, _ := propBool(p.Value); b && strings.EqualFold(p.Name, "startmap") {
				starts = append(starts, mapPath)
			}
		}
	}
	if len(starts) != 1 {
		t.Fatalf("マップ設定の「ゲームの最初のマップ」にチェックを入れたマップを1つだけにしてください(今: %d個 %v)", len(starts), starts)
	}
	start := starts[0]

	tmap, _ := loadTiledMap(start)
	hasStartPoint := false
	forEachEventObject(tmap, func(obj TiledObject, p map[string]string) {
		if obj.Name == "start_point" {
			hasStartPoint = true
		}
	})
	if !hasStartPoint {
		t.Errorf("%s: ゲームの最初のマップには、名前が start_point の着地点を置いてください(ニューゲームでそこから始まる)", start)
	}

	if os.Getenv("UPDATE_TILED_PROJECT") != "" {
		data, err := os.ReadFile(startMapGoPath)
		if err != nil {
			t.Fatal(err)
		}
		old := fmt.Sprintf("const startMapPath = %q", startMapPath)
		updated := strings.Replace(string(data), old, fmt.Sprintf("const startMapPath = %q", start), 1)
		if err := os.WriteFile(startMapGoPath, []byte(updated), 0o644); err != nil {
			t.Fatal(err)
		}
		if start != startMapPath {
			t.Logf("ゲームの最初のマップを %s にしました", start)
		}
		return
	}
	if start != startMapPath {
		t.Errorf("ゲームの最初のマップが %s に変わっています。go run ./tools/update を打ってください(今のゲームは %s から始まる)", start, startMapPath)
	}
}
