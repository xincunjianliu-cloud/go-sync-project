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

// ロード地点以外の切り替えでは、待ちが長引いてもLoadingを出さないこと。
func TestNoLoadingOutsideLoadPoints(t *testing.T) {
	g := newLoadingTestGame(&stubScene{"field"})
	g.heavyTierReady[assetTierField] = true
	g.ChangeSceneWhenTierReady(assetTierBattle, func() Scene { return &stubScene{"battle"} }, 0.1)

	for g.fadeMode != FadeLoading {
		runFrames(g, 1)
	}
	runFrames(g, 120)
	if g.fadeMode != FadeLoading {
		t.Fatal("switched before the battle tier was ready")
	}
	if g.isLoading() {
		t.Fatal("Loading shown outside a load point")
	}
}

// ロード地点では、準備が済んでいても暗転したらすぐLoadingを出し、
// loadPointMinTicksのあいだは出し続けてから画面を明けること。
func TestLoadPointAlwaysShowsLoading(t *testing.T) {
	field := &stubScene{"field"}
	g := newLoadingTestGame(&stubScene{"title"})
	g.heavyAssetsReady = true
	g.ChangeScenePrepared(scenePrep{tier: assetTierBattle, loadPoint: true}, func() Scene { return field }, 0.1)

	for g.fadeMode != FadeLoading {
		runFrames(g, 1)
	}
	if !g.isLoading() {
		t.Fatal("Loading not shown right after the screen went black at a load point")
	}
	runFrames(g, loadPointMinTicks-2)
	if g.currentScene == field || !g.isLoading() {
		t.Fatal("load point ended before the minimum display time")
	}
	runFrames(g, 3)
	if g.currentScene != field {
		t.Fatal("load point never switched to the next scene")
	}
	if g.isLoading() {
		t.Fatal("Loading still shown after the load point finished")
	}
}

// ボス戦への切り替えは、そのボスの画像がまだなら暗転したまま裏で読み込み、
// そろってから組み立てること(ボスの画像は段階読み込みに入っていない)。
func TestBossBattleWaitsForBossImages(t *testing.T) {
	battle := &stubScene{"battle"}
	g := newLoadingTestGame(&stubScene{"field"})
	g.heavyAssetsReady = true

	builtWithImages := false
	g.ChangeSceneToBossBattle("boss_1", func() Scene {
		builtWithImages = g.BossImgs[0] != nil && g.BossIconImgs[0] != nil &&
			g.BossIconLargeImgs[0] != nil && g.BossBgImgs[0] != nil
		return battle
	}, 0.1)

	deadline := time.Now().Add(5 * time.Second)
	for g.currentScene != battle && time.Now().Before(deadline) {
		runFrames(g, 1)
		time.Sleep(2 * time.Millisecond)
	}
	if g.currentScene != battle {
		t.Fatalf("scene never switched (pending=%v)", g.asyncImagePending)
	}
	if !builtWithImages {
		t.Fatal("boss battle was built before its images were loaded")
	}
	if g.BossImgs[1] != nil {
		t.Error("another boss's images were loaded for boss_1's battle")
	}
}
