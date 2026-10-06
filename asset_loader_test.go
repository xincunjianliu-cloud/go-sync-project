package main

import (
	"errors"
	"sync"
	"testing"
)

// blockingFetcher はテスト用の取得関数。gateを閉じるまで最初の取得を止め、
// 取得した順番を記録する。
type blockingFetcher struct {
	mu    sync.Mutex
	order []string
	gate  chan struct{}
	calls map[string]int
	fail  map[string]bool
}

func newBlockingFetcher() *blockingFetcher {
	return &blockingFetcher{gate: make(chan struct{}), calls: map[string]int{}, fail: map[string]bool{}}
}

func (f *blockingFetcher) fetch(path string) ([]byte, error) {
	<-f.gate
	f.mu.Lock()
	defer f.mu.Unlock()
	f.order = append(f.order, path)
	f.calls[path]++
	if f.fail[path] {
		return nil, errors.New("not found")
	}
	return []byte(path), nil
}

// 待ち行列は優先度順(同じ優先度は依頼順)に取得し、後から高い優先度で
// 頼み直したものは順番が繰り上がること。
func TestAssetLoaderFetchesByPriority(t *testing.T) {
	f := newBlockingFetcher()
	l := newAssetLoader(1, f.fetch)

	// 1本だけのワーカーが最初の1件で止まっている間に残りを積む。
	first := l.request("first", prioUrgent)
	l.request("rest", prioTierBase+int(assetTierRest))
	l.request("battle", prioTierBase+int(assetTierBattle))
	l.request("field-a", prioTierBase+int(assetTierField))
	l.request("field-b", prioTierBase+int(assetTierField))
	l.request("audio", prioAudioPrefetch)
	l.request("rest", prioSoon) // 繰り上げ
	close(f.gate)

	<-first.done
	l.wait([]string{"rest", "battle", "field-a", "field-b", "audio"})

	want := []string{"first", "rest", "field-a", "field-b", "battle", "audio"}
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.order) != len(want) {
		t.Fatalf("order = %v, want %v", f.order, want)
	}
	for i := range want {
		if f.order[i] != want[i] {
			t.Fatalf("order = %v, want %v", f.order, want)
		}
	}
}

// 同じパスは何度頼んでも1回しか取得せず、releaseした後に頼むと取得し直すこと。
func TestAssetLoaderDedupesAndReleases(t *testing.T) {
	f := newBlockingFetcher()
	close(f.gate)
	l := newAssetLoader(2, f.fetch)

	for i := 0; i < 3; i++ {
		data, err := l.get("a.png")
		if err != nil || string(data) != "a.png" {
			t.Fatalf("get = %q, %v", data, err)
		}
	}
	if !l.has("a.png") {
		t.Fatal("has(a.png) = false after get")
	}
	l.release("a.png")
	if l.has("a.png") {
		t.Fatal("has(a.png) = true after release")
	}
	if _, err := l.get("a.png"); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.calls["a.png"] != 2 {
		t.Fatalf("fetched a.png %d times, want 2 (once before and once after release)", f.calls["a.png"])
	}
}

// 失敗した取得は1回だけエラーを返し、次に頼まれたら取得し直すこと。
func TestAssetLoaderRetriesAfterError(t *testing.T) {
	f := newBlockingFetcher()
	close(f.gate)
	f.fail["missing.png"] = true
	l := newAssetLoader(1, f.fetch)

	if _, err := l.get("missing.png"); err == nil {
		t.Fatal("get(missing.png) error = nil")
	}
	if _, err := l.get("missing.png"); err == nil {
		t.Fatal("second get(missing.png) error = nil")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.calls["missing.png"] != 2 {
		t.Fatalf("fetched missing.png %d times, want 2", f.calls["missing.png"])
	}
}

func newCapTestAudio() *AudioManager {
	return &AudioManager{
		pcmCache:   map[string][]byte{},
		pcmLoading: map[string]bool{},
		pcmFailed:  map[string]bool{},
	}
}

// BGMのPCMは合計容量が上限を超えたら古い曲から外し、直近の曲と再生待ちの曲は
// 残すこと。
func TestBGMCacheEvictsByBytes(t *testing.T) {
	a := newCapTestAudio()
	const track = maxCachedBGMBytes / 3
	paths := []string{"assets/bgm/a.mp3", "assets/bgm/b.mp3", "assets/bgm/c.mp3", "assets/bgm/d.mp3"}
	for _, p := range paths {
		a.pcmCache[p] = make([]byte, track+1)
	}
	a.pendingPaths = []string{"assets/bgm/a.mp3"} // 再生待ち: 外してはいけない

	for _, p := range paths {
		a.touchBGM(p)
	}

	if _, ok := a.pcmCache["assets/bgm/a.mp3"]; !ok {
		t.Error("pending track a was evicted")
	}
	if _, ok := a.pcmCache["assets/bgm/b.mp3"]; ok {
		t.Error("oldest non-pending track b was kept")
	}
	for _, p := range paths[2:] {
		if _, ok := a.pcmCache[p]; !ok {
			t.Errorf("recent track %s was evicted", p)
		}
	}
}

// 戦闘段階が必要な遷移は、フィールド段階だけ終わった状態では暗転したまま
// 待ち、戦闘段階が終わったら組み立てること。
func TestChangeSceneWhenTierReadyWaitsForBattleTier(t *testing.T) {
	field := &stubScene{"field"}
	battle := &stubScene{"battle"}
	g := newLoadingTestGame(field)
	g.heavyTierReady[assetTierField] = true

	built := 0
	g.ChangeSceneWhenTierReady(assetTierBattle, func() Scene { built++; return battle }, 0.1)
	runFrames(g, 30)
	if built != 0 || g.fadeMode != FadeLoading {
		t.Fatalf("built=%d fadeMode=%d, want waiting in FadeLoading", built, g.fadeMode)
	}

	g.heavyTierReady[assetTierBattle] = true
	runFrames(g, 1)
	if built != 1 || g.currentScene != battle {
		t.Fatalf("after battle tier: built=%d scene=%v", built, g.currentScene)
	}
}

// 「はじめから」(フィールド段階)は戦闘段階の読み込みを待たないこと。
func TestChangeSceneWhenReadyNeedsOnlyFieldTier(t *testing.T) {
	field := &stubScene{"field"}
	g := newLoadingTestGame(&stubScene{"title"})
	g.heavyTierReady[assetTierField] = true
	g.ChangeSceneWhenReady(func() Scene { return field }, 0.1)
	runFrames(g, 10)
	if g.currentScene != field {
		t.Fatalf("currentScene=%v, want field without waiting for the battle tier", g.currentScene)
	}
}

// 進捗は待っている段階までの枚数の割合になること。
func TestLoadProgressCountsTiersUpToTarget(t *testing.T) {
	g := newLoadingTestGame(&stubScene{"title"})
	g.heavyTierTotal = [assetTierCount]int{10, 30, 5}
	g.heavyTierDone = [assetTierCount]int{5, 0, 0}

	p, ok := g.loadProgress()
	if !ok || p != 0.5 {
		t.Fatalf("title progress = %v, %v; want 0.5 of the field tier", p, ok)
	}

	g.heavyTierReady[assetTierField] = true
	g.heavyTierDone[assetTierField] = 10
	g.heavyTierDone[assetTierBattle] = 10
	if _, ok := g.loadProgress(); ok {
		t.Fatal("progress shown while nothing is waiting")
	}

	g.ChangeSceneWhenTierReady(assetTierBattle, func() Scene { return &stubScene{"battle"} }, 0.1)
	runFrames(g, 30)
	p, ok = g.loadProgress()
	if !ok || p != 0.5 {
		t.Fatalf("battle wait progress = %v, %v; want (10+10)/(10+30)=0.5", p, ok)
	}
}

// ロード地点で固定した曲は、容量を超えても古い順に外さず、固定していない曲を
// 先に外すこと。
func TestBGMCacheKeepsPinnedTracks(t *testing.T) {
	a := newCapTestAudio()
	const track = maxCachedBGMBytes / 3
	paths := []string{"assets/bgm/a.mp3", "assets/bgm/b.mp3", "assets/bgm/c.mp3", "assets/bgm/d.mp3"}
	for _, p := range paths {
		a.pcmCache[p] = make([]byte, track+1)
	}
	a.PinBGM([]string{"assets/bgm/a.mp3"})

	for _, p := range paths {
		a.touchBGM(p)
	}

	if _, ok := a.pcmCache["assets/bgm/a.mp3"]; !ok {
		t.Error("pinned (oldest) track a was evicted")
	}
	if _, ok := a.pcmCache["assets/bgm/b.mp3"]; ok {
		t.Error("unpinned track b was kept while over the cap")
	}
}
