//go:build !js

package main

import (
	"io/fs"
	"testing"
)

// マップの書き間違いのチェック。中身は map_check.go にあり、Tiled の F5
// (試し遊び)でも同じチェックが動く。ここでは go test から呼ぶ。

func allMapFiles(t *testing.T) []string {
	t.Helper()
	paths, err := fs.Glob(embeddedAssets, "assets/maps/*.tmj")
	if err != nil {
		t.Fatal(err)
	}
	return paths
}

func runMapCheck(t *testing.T, fn func(*mapChecker)) {
	t.Helper()
	c := &mapChecker{}
	fn(c)
	for _, issue := range c.issues {
		t.Error(issue.String())
	}
}

func TestMapSettings(t *testing.T)     { runMapCheck(t, checkMapSettings) }
func TestMapObjects(t *testing.T)      { runMapCheck(t, checkMapObjects) }
func TestMapTilesets(t *testing.T)     { runMapCheck(t, checkMapTilesets) }
func TestTilesetFiles(t *testing.T)    { runMapCheck(t, checkTilesetFiles) }
func TestMapWarps(t *testing.T)        { runMapCheck(t, checkMapWarps) }
func TestMapLeverObjects(t *testing.T) { runMapCheck(t, checkMapLeverObjects) }

// 通れないレイヤー(四角・タイル)が、通れるレバー壁(机の橋など)をふさいでいないこと。
func TestCollisionDoesNotBlockLeverPaths(t *testing.T) {
	runMapCheck(t, checkCollisionDoesNotBlockLeverPaths)
}

// レバーの壁が開いたときの見た目が「レバーで出るレイヤー」に描いてあること。
func TestLeverWallsHaveOpenTiles(t *testing.T) { runMapCheck(t, checkLeverWallsHaveOpenTiles) }

// 「歩く道順」の2通りの書き方(マス数・ピクセル)が同じ動きになること。
func TestParseRoute(t *testing.T) {
	ja, err := parseRoute("上4、右２マス, 下0.5")
	if err != "" {
		t.Fatal(err)
	}
	en, err := parseRoute("up128,right64,down16")
	if err != "" {
		t.Fatal(err)
	}
	want := []MoveStep{{Dir: 3, Dist: 128}, {Dir: 2, Dist: 64}, {Dir: 0, Dist: 16}}
	for _, got := range [][]MoveStep{ja, en} {
		if len(got) != len(want) {
			t.Fatalf("道順 = %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("道順[%d] = %v, want %v", i, got[i], want[i])
			}
		}
	}
	if steps, err := parseRoute(""); err != "" || steps != nil {
		t.Error("空の道順を間違いと判定しました")
	}
	for _, bad := range []string{"upp120", "up", "上", "前4"} {
		if _, err := parseRoute(bad); err == "" {
			t.Errorf("間違った道順 %q を見逃しました", bad)
		}
	}
}

func ensureStoryDialoguesLoaded() {
	if len(storyDialogues) == 0 {
		loadStoryDialogues("assets/dialogues")
	}
}
