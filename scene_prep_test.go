package main

import (
	"testing"
	"time"
)

// マップへの切り替えは、行き先のタイルセットがまだデコードされていなければ
// 暗転したまま裏でデコードし、キャッシュに入ってから組み立てること。
func TestChangeSceneToMapWaitsForTileset(t *testing.T) {
	tmap, err := loadTiledMap(startMapPath)
	if err != nil {
		t.Fatal(err)
	}
	tilesetPath, ok := mapTilesetImagePath(tmap)
	if !ok {
		t.Fatal("start map has no tileset")
	}
	saved, hadSaved := tilesetImageCache[tilesetPath]
	delete(tilesetImageCache, tilesetPath)
	t.Cleanup(func() {
		if hadSaved {
			tilesetImageCache[tilesetPath] = saved
		}
	})

	field := &stubScene{"field"}
	g := newLoadingTestGame(&stubScene{"title"})
	g.heavyAssetsReady = true

	builtWithTileset := false
	built := 0
	g.ChangeSceneToMap(startMapPath, func() Scene {
		built++
		_, builtWithTileset = tilesetImageCache[tilesetPath]
		return field
	}, 0.1)
	if built != 0 {
		t.Fatal("built before the tileset was decoded")
	}

	deadline := time.Now().Add(5 * time.Second)
	for g.currentScene != field && time.Now().Before(deadline) {
		runFrames(g, 1)
		time.Sleep(2 * time.Millisecond)
	}
	if g.currentScene != field {
		t.Fatalf("scene never switched (fadeMode=%d, pending=%v)", g.fadeMode, g.asyncImagePending)
	}
	if built != 1 || !builtWithTileset {
		t.Fatalf("built=%d builtWithTileset=%v, want built once after the tileset was cached", built, builtWithTileset)
	}
}

// 暗転中の待ちが短いうちはLoadingを出さず、長引いたら出すこと。
func TestLoadingIndicatorIsDelayed(t *testing.T) {
	g := newLoadingTestGame(&stubScene{"field"})
	g.heavyTierReady[assetTierField] = true
	g.ChangeSceneWhenTierReady(assetTierBattle, func() Scene { return &stubScene{"battle"} }, 0.1)

	for g.fadeMode != FadeLoading {
		runFrames(g, 1)
	}
	runFrames(g, loadingIndicatorDelayTicks-2)
	if g.isLoading() {
		t.Fatal("Loading shown right after the screen went black")
	}
	runFrames(g, 3)
	if !g.isLoading() {
		t.Fatal("Loading not shown after a long wait")
	}
}
