package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

type stubScene struct{ name string }

func (s *stubScene) Update(dt float64) Scene   { return s }
func (s *stubScene) Draw(screen *ebiten.Image) {}
func runFrames(g *Game, n int) {
	for i := 0; i < n; i++ {
		g.Update()
	}
}
func newLoadingTestGame(current Scene) *Game {
	return &Game{currentScene: current, bootReady: true}
}

// 画像の読み込みが終わる前にシーン遷移を要求すると、暗転後もLoading表示の
// まま待ち、読み込み完了後に組み立ててフェードインすること。
func TestChangeSceneWhenReadyWaitsForHeavyAssets(t *testing.T) {
	title := &stubScene{"title"}
	field := &stubScene{"field"}
	g := newLoadingTestGame(title)

	built := 0
	g.ChangeSceneWhenReady(func() Scene { built++; return field }, 0.1)
	runFrames(g, 30)

	if g.fadeMode != FadeLoading {
		t.Fatalf("fadeMode = %d, want FadeLoading while heavy assets are not ready", g.fadeMode)
	}
	if built != 0 {
		t.Fatalf("build called %d times before heavy assets were ready", built)
	}
	if !g.isLoading() {
		t.Fatal("isLoading() = false during FadeLoading")
	}

	g.heavyAssetsReady = true
	runFrames(g, 1)
	if built != 1 || g.currentScene != field || g.fadeMode != FadeIn {
		t.Fatalf("after ready: built=%d scene=%v fadeMode=%d", built, g.currentScene, g.fadeMode)
	}
}

// buildが失敗(nil)したら元のシーンへ戻ること。
func TestChangeSceneWhenReadyFallsBackOnBuildFailure(t *testing.T) {
	title := &stubScene{"title"}
	g := newLoadingTestGame(title)
	g.ChangeSceneWhenReady(func() Scene { return nil }, 0.1)
	runFrames(g, 30)
	g.heavyAssetsReady = true
	runFrames(g, 1)
	if g.currentScene != title || g.fadeMode != FadeIn {
		t.Fatalf("scene=%v fadeMode=%d, want title + FadeIn", g.currentScene, g.fadeMode)
	}
}

// 読み込み済みなら通常のフェード遷移と同じく即座に組み立てること。
func TestChangeSceneWhenReadyBuildsImmediatelyWhenReady(t *testing.T) {
	field := &stubScene{"field"}
	g := newLoadingTestGame(&stubScene{"title"})
	g.heavyAssetsReady = true
	g.ChangeSceneWhenReady(func() Scene { return field }, 0.1)
	if g.pendingScene != field || g.fadeMode != FadeOut {
		t.Fatalf("pendingScene=%v fadeMode=%d", g.pendingScene, g.fadeMode)
	}
	runFrames(g, 10)
	if g.currentScene != field {
		t.Fatalf("currentScene=%v, want field", g.currentScene)
	}
}

// BGMの再生開始はデコード完了まで保留され、その間IsLoadingがtrueになること。
func TestAudioStartWaitsForDecode(t *testing.T) {
	a := &AudioManager{
		pcmCache:   map[string][]byte{},
		pcmLoading: map[string]bool{"x.mp3": true}, // デコード中扱い
		pcmFailed:  map[string]bool{},
	}
	started := false
	a.startWhenDecoded(func() { started = true }, "x.mp3")
	if started || !a.IsLoading() {
		t.Fatalf("started=%v IsLoading=%v, want pending", started, a.IsLoading())
	}

	a.pcmMu.Lock()
	delete(a.pcmLoading, "x.mp3")
	a.pcmCache["x.mp3"] = []byte{1, 2, 3, 4}
	a.pcmMu.Unlock()
	a.tryPendingStart()
	if !started || a.IsLoading() {
		t.Fatalf("started=%v IsLoading=%v after decode finished", started, a.IsLoading())
	}
}
