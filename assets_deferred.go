package main

import (
	"fmt"
	"image"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// assetTier はタイトル画面の裏で読み込む画像の段階。どのファイルかではなく
// 「どの画面で使うか」で分ける。段階は小さい順に読み込み、各画面は自分が
// 必要な段階が終わった時点で先へ進める（例: 「はじめから」はフィールド段階
// だけ待てばよく、戦闘やボスの画像の読み込みは歩いている間に続く）。
// 新しい画像を追加するときは、使う画面に合わせて段階を選ぶこと。
type assetTier int

const (
	// assetTierField はフィールドを歩く・会話する・メニューを開くのに
	// 必要なもの（最初のマップのタイルセット、立ち絵、メニュー画面など）。
	assetTierField assetTier = iota
	// assetTierBattle は戦闘で使うもの（敵・ボス・戦闘UI）。
	assetTierBattle
	// assetTierRest は今すぐには使わないもの（他のマップのタイルセット）。
	// 間に合わなくても、そのマップに入るときに個別に読み込まれる。
	assetTierRest
	assetTierCount
)

func (t assetTier) String() string {
	switch t {
	case assetTierField:
		return "フィールド"
	case assetTierBattle:
		return "戦闘"
	case assetTierRest:
		return "その他"
	}
	return fmt.Sprintf("tier%d", int(t))
}

// deferredAssetAssign はパス1件と、その読み込み段階、デコード済み画像を
// Gameのどのフィールドに格納するかを表す。起動を速くするため、タイトル画面の
// 表示に不要な画像はすべてこのテーブル経由でバックグラウンド読み込みする。
type deferredAssetAssign struct {
	path   string
	tier   assetTier
	assign func(*Game, *ebiten.Image)
	// optional は読み込めなくてもゲーム全体の読み込みエラーにしないもの。
	optional bool
}

type heavyMsgKind int

const (
	heavyMsgImage heavyMsgKind = iota
	heavyMsgTotals
	heavyMsgTierDone
)

// decodedHeavyAsset はバックグラウンドgoroutineからUpdate()側へ結果を渡すための
// メッセージ。画像デコードまではgoroutine側で行い、ebiten.Imageの生成
// (GPUテクスチャ確保)は必ずUpdate()を呼ぶメインゴルーチン側で行う
// (Ebitengineのグラフィックス関連APIはメインゴルーチン以外からの呼び出しが
// 保証されていないため)。
type decodedHeavyAsset struct {
	kind   heavyMsgKind
	tier   assetTier
	totals [assetTierCount]int
	assign func(*Game, *ebiten.Image)
	img    image.Image
	err    error
	label  string
}

// loadHeavyAssetsAsync はタイトル画面表示後にバックグラウンドで実行され、
// フィールド・戦闘・メニューなど、タイトル画面自体には不要な画像とデータを
// 段階(assetTier)の順に読み込む。各段階が終わるたびにheavyMsgTierDoneを送り、
// 全部終わる(またはエラーで中断する)とg.heavyDecodedを閉じる。
func (g *Game) loadHeavyAssetsAsync() {
	defer close(g.heavyDecoded)

	// 画像のダウンロードは、マップやセリフデータの解析と並行して先に
	// 始めておく（段階ごとの優先度つきで、フィールド用が先に届く）。
	assignments := deferredAssetAssignments()
	for _, a := range assignments {
		assetStore.request(a.path, prioTierBase+int(a.tier))
	}

	if err := BuildObjectiveAndMapIndex(); err != nil {
		g.heavyDecoded <- decodedHeavyAsset{err: err, label: "目的地インデックスの構築"}
		return
	}
	g.UpdateObjective()

	if err := LoadDialogues("assets/dialogues"); err != nil {
		g.heavyDecoded <- decodedHeavyAsset{err: err, label: "セリフデータの読み込み"}
		return
	}

	tilesets := mapTilesetAssignments(assignments)
	for _, a := range tilesets {
		assetStore.request(a.path, prioTierBase+int(a.tier))
	}
	assetStore.request(fieldPlayerConfigPath, prioTierBase+int(assetTierField))

	all := append(assignments, tilesets...)
	var byTier [assetTierCount][]deferredAssetAssign
	var totals [assetTierCount]int
	for _, a := range all {
		byTier[a.tier] = append(byTier[a.tier], a)
		totals[a.tier]++
	}
	g.heavyDecoded <- decodedHeavyAsset{kind: heavyMsgTotals, totals: totals}

	for tier := range assetTierCount {
		for _, a := range byTier[tier] {
			img, err := decodeAssetImage(a.path)
			msg := decodedHeavyAsset{kind: heavyMsgImage, tier: tier, assign: a.assign, img: img, err: err, label: a.path}
			if err != nil && a.optional {
				// 各マップのタイルセットは、読み込めなくてもそのマップに入った
				// 時点でNewRoomSceneが改めて読み込み・エラー表示する。
				msg.err, msg.assign = nil, nil
			}
			g.heavyDecoded <- msg
			yieldToBrowser()
		}
		g.heavyDecoded <- decodedHeavyAsset{kind: heavyMsgTierDone, tier: tier}
	}
}

// mapTilesetAssignments はBuildObjectiveAndMapIndexが見つけた全マップの
// タイルセット画像を、tilesetImageCacheへ先に入れておくための
// 一覧を返す。これが無いと、初めて入るマップのタイルセット画像を
// ドア移動やロードの瞬間に同期デコードすることになり、画面が固まる。
// 最初のマップのものはフィールド段階、それ以外は後回しの段階にする。
// alreadyに同じパスがある画像は二重に読まない。
func mapTilesetAssignments(already []deferredAssetAssign) []deferredAssetAssign {
	seen := make(map[string]bool, len(already))
	for _, a := range already {
		seen[a.path] = true
	}
	var out []deferredAssetAssign
	// 複数のマップが同じ画像を使うので、最初のマップを先に見てフィールド段階に入れる。
	for _, mapPath := range append([]string{startMapPath}, allMapPaths...) {
		tmap, err := loadTiledMap(mapPath)
		if err != nil {
			continue
		}
		tier := assetTierRest
		if mapPath == startMapPath {
			tier = assetTierField
		}
		for _, p := range mapTilesetImagePaths(tmap) {
			if seen[p] {
				continue
			}
			seen[p] = true
			out = append(out, deferredAssetAssign{
				path:     p,
				tier:     tier,
				assign:   func(g *Game, img *ebiten.Image) { tilesetImageCache[p] = img },
				optional: true,
			})
		}
	}
	return out
}

// heavyPumpFrameBudget は1フレームのうちpumpHeavyAssetsがGPUテクスチャ
// 生成に使ってよい時間。溜まった画像を1フレームで全部処理すると、
// その1フレームが長引いてLoading表示やタイトル画面がカクつく。
const heavyPumpFrameBudget = 6 * time.Millisecond

// pumpHeavyAssets はUpdate()から毎フレーム呼ばれ、バックグラウンドで
// デコード済みの画像をebiten.Imageへ変換してGameへ反映する。
// チャンネルに溜まっている分をheavyPumpFrameBudgetの範囲で処理し、
// 送信側(loadHeavyAssetsAsync)がネットワーク待ちで詰まっている間は
// defaultに抜けてブロックしない。
func (g *Game) pumpHeavyAssets() {
	if g.heavyAssetsReady || g.heavyDecoded == nil {
		return
	}
	start := time.Now()
	for time.Since(start) < heavyPumpFrameBudget {
		select {
		case item, ok := <-g.heavyDecoded:
			if !ok {
				g.finishHeavyAssets()
				return
			}
			g.applyHeavyMsg(item)
		default:
			return
		}
	}
}

func (g *Game) applyHeavyMsg(item decodedHeavyAsset) {
	switch item.kind {
	case heavyMsgTotals:
		g.heavyTierTotal = item.totals
	case heavyMsgTierDone:
		g.heavyTierReady[item.tier] = true
		loadTrace("%s段階の読み込み完了 (%d枚, 累計取得%.1fMiB)",
			item.tier, g.heavyTierTotal[item.tier], mib(assetStore.totalFetchedBytes()))
		g.onTierReady(item.tier)
	default:
		g.heavyTierDone[item.tier]++
		if item.err != nil {
			if g.heavyAssetsErr == nil {
				g.heavyAssetsErr = fmt.Errorf("%sに失敗しました: %w", item.label, item.err)
			}
			return
		}
		if item.assign != nil {
			item.assign(g, newImageTraced(item.label, item.img))
		}
	}
}

// finishHeavyAssets は読み込み用goroutineが終わった(全段階完了、または
// エラーで中断した)ときに呼ばれる。中断した場合も待っている画面が
// 永久に止まらないよう、全段階を完了扱いにする（エラーはheavyAssetsErrで
// 呼び出し側が判断する）。
func (g *Game) finishHeavyAssets() {
	g.heavyAssetsReady = true
	for t := range assetTierCount {
		if !g.heavyTierReady[t] {
			g.heavyTierReady[t] = true
			g.onTierReady(t)
		}
	}
	if g.heavyAssetsErr != nil {
		loadTrace("読み込みエラー: %v", g.heavyAssetsErr)
	}
}

// onTierReady は段階の読み込みが終わったときに、その段階の後で使い始める
// 音をバックグラウンドで先に準備しておく。
func (g *Game) onTierReady(tier assetTier) {
	switch tier {
	case assetTierField:
		// 「はじめから」で最初に流れる曲。
		if tmap, err := loadTiledMap(startMapPath); err == nil {
			p, _ := mapBGMPath(tmap)
			g.Audio.Prewarm(prioSoon, p)
		}
	case assetTierBattle:
		// 最初の戦闘開始・勝利で待ちが出ないように。
		g.Audio.Prewarm(prioTierBase+int(assetTierRest), bgmBattleNormal, bgmVictoryIntro, bgmVictoryLoop)
	case assetTierRest:
		// まだ用意していないボスの画像も、ほかが全部終わった後に読んでおく。
		all := make([]int, len(g.BossImgs))
		for i := range all {
			all[i] = i
		}
		g.prepareBossImages(all, prioAudioPrefetch)
	}
}

// bossImageAsset はボス1体分の画像1枚と、その格納先。
type bossImageAsset struct {
	path string
	slot **ebiten.Image
}

// bossImageAssets はボス(BossImgs等の添字idx)の戦闘で使う画像の一覧を返す。
// ボスの画像は段階読み込みに入れず、そのボスがいるマップのロード地点や、
// ボス戦が始まる会話の開始時点で、ボスごとに読み込む。こうしておくと、
// ボスを増やしてもロード地点の待ちはそのマップにいるボスの分しか増えない。
func (g *Game) bossImageAssets(idx int) []bossImageAsset {
	if idx < 0 || idx >= len(g.BossImgs) {
		return nil
	}
	n := idx + 1
	return []bossImageAsset{
		{fmt.Sprintf("assets/images/battle/boss_%d.png", n), &g.BossImgs[idx]},
		{fmt.Sprintf("assets/images/battle/boss_%d_icon.png", n), &g.BossIconImgs[idx]},
		{fmt.Sprintf("assets/images/battle/boss_%d_icon_large.png", n), &g.BossIconLargeImgs[idx]},
	}
}

// prepareBossImages はbossesのボスの画像(専用の戦闘背景があればそれも)のうち、
// まだのものを優先度prioで裏でデコードし始め、全部終わって(成功・失敗
// どちらでも)いるかを返す。デコード中のものを、より高い優先度で頼み直すと
// 取得の順番が繰り上がる。
func (g *Game) prepareBossImages(bosses []int, prio int) bool {
	ready := true
	for _, idx := range bosses {
		if key, ok := bossBattleBgKey(idx); ok && !g.prepareBattleBgs([]string{key}, prio) {
			ready = false
		}
		for _, a := range g.bossImageAssets(idx) {
			if *a.slot != nil || g.asyncImageFailed[a.path] {
				continue
			}
			ready = false
			if g.asyncImagePending[a.path] {
				assetStore.request(a.path, prio)
				continue
			}
			path, slot := a.path, a.slot
			g.decodeImageAsync(path, prio, func(img *ebiten.Image) {
				if *slot == nil {
					*slot = img
				}
			}, func() { loadTrace("ボスの画像を読み込めませんでした: %s", path) })
		}
	}
	return ready
}

// prewarmSE は全効果音をバックグラウンドで先にデコードしておく。
// タイトル画面が出た直後に呼ぶ（カーソル移動や決定の音が最初から鳴るように）。
func (g *Game) prewarmSE() {
	paths := make([]string, 0, len(seByKey))
	for _, p := range seByKey {
		paths = append(paths, p)
	}
	g.Audio.Prewarm(prioSoon, paths...)
}

// deferredAssetAssignments はバックグラウンドで読み込む画像の一覧と、
// 読み込み段階、読み込み後にGameのどのフィールドへ入れるかを返す。
func deferredAssetAssignments() []deferredAssetAssign {
	var a []deferredAssetAssign
	add := func(tier assetTier, path string, assign func(*Game, *ebiten.Image)) {
		a = append(a, deferredAssetAssign{path: path, tier: tier, assign: assign})
	}

	// ---- フィールド段階: 歩く・会話・メニュー ----
	add(assetTierField, "assets/images/field/player_walk.png", func(g *Game, img *ebiten.Image) {
		g.SpriteSheet = img
		preloadedPlayerSprites["assets/images/field/player_walk.png"] = img
	})
	for i := range 4 {
		add(assetTierField, fmt.Sprintf("assets/images/field/bossスプライト_%d.png", i+1), func(g *Game, img *ebiten.Image) { g.BossSpriteSheets[i] = img })
	}
	add(assetTierField, "assets/images/field/map_name_banner.png", func(g *Game, img *ebiten.Image) { g.MapNameBannerImg = img })
	add(assetTierField, "assets/images/common/window.png", func(g *Game, img *ebiten.Image) { g.WindowImg = img })
	// メッセージ履歴とスキル強化チュートリアルは、戦闘フォルダの画像を
	// フィールドでも使う。
	add(assetTierField, "assets/images/battle/log_entry_box.png", func(g *Game, img *ebiten.Image) { g.LogEntryImg = img })
	add(assetTierField, "assets/images/battle/skill_panel.png", func(g *Game, img *ebiten.Image) { g.SkillPanelImg = img })
	add(assetTierField, "assets/images/field/調べる.png", func(g *Game, img *ebiten.Image) { g.ExamineIconImg = img })
	add(assetTierField, "assets/images/field/現在地.png", func(g *Game, img *ebiten.Image) { g.MinimapPlayerIconImg = img })
	add(assetTierField, "assets/images/field/目的地.png", func(g *Game, img *ebiten.Image) { g.MinimapObjectiveIconImg = img })
	add(assetTierField, "assets/images/field/chest.png", func(g *Game, img *ebiten.Image) { g.ChestImg = img })
	add(assetTierField, "assets/images/field/key_chest.png", func(g *Game, img *ebiten.Image) { g.KeyChestImg = img })
	add(assetTierField, "assets/images/field/locked_wall.png", func(g *Game, img *ebiten.Image) { g.LockedWallImg = img })
	add(assetTierField, "assets/images/field/lever_wall_open.png", func(g *Game, img *ebiten.Image) { g.LeverWallOpenImg = img })
	add(assetTierField, "assets/images/field/lever_wall_open_deco.png", func(g *Game, img *ebiten.Image) { g.LeverWallOpenDecoImg = img })
	add(assetTierField, "assets/images/field/lever.png", func(g *Game, img *ebiten.Image) { g.LeverImg = img })
	add(assetTierField, "assets/images/field/push_block.png", func(g *Game, img *ebiten.Image) { g.BlockImg = img })
	add(assetTierField, "assets/images/field/push_block_spot.png", func(g *Game, img *ebiten.Image) { g.BlockSpotImg = img })

	for i := range 4 {
		bossName := BossNames[i]
		add(assetTierField, fmt.Sprintf("assets/images/common/chara_boss%d.png", i+1), func(g *Game, img *ebiten.Image) { g.CharaImgs[bossName] = img })
	}
	for i := range partySize {
		playerName := PlayerNames[i]
		add(assetTierField, fmt.Sprintf("assets/images/common/chara_player_%d.png", i+1), func(g *Game, img *ebiten.Image) { g.CharaImgs[playerName] = img })
	}

	add(assetTierField, "assets/images/menu/メニュー画面.png", func(g *Game, img *ebiten.Image) { g.MenuBgImg = img })
	add(assetTierField, "assets/images/menu/メニュー画面拡張.png", func(g *Game, img *ebiten.Image) { g.MenuSkillPanelImg = img })
	for i := range 4 {
		add(assetTierField, fmt.Sprintf("assets/images/menu/party_icon_%d.png", i+1), func(g *Game, img *ebiten.Image) { g.PartyIconImgs[i] = img })
	}
	add(assetTierField, "assets/images/menu/セーブスロット選択中.png", func(g *Game, img *ebiten.Image) { g.SaveThumbFrameSelImg = img })
	add(assetTierField, "assets/images/menu/セーブスロット.png", func(g *Game, img *ebiten.Image) { g.SaveThumbFrameImg = img })

	// ---- 戦闘段階 ----

	// ボスの画像は段階に入れず、ボスごとに必要になった時点で読み込む
	// (bossImageAssets / prepareBossImages)。

	for _, enemy := range EnemyDatabase {
		name := enemy.Name
		add(assetTierBattle, fmt.Sprintf("assets/images/battle/enemy_%s.png", name), func(g *Game, img *ebiten.Image) { g.EnemyImgs[name] = img })
		add(assetTierBattle, fmt.Sprintf("assets/images/battle/enemy_%s_icon.png", name), func(g *Game, img *ebiten.Image) { g.EnemyIconImgs[name] = img })
		add(assetTierBattle, fmt.Sprintf("assets/images/battle/enemy_%s_icon_large.png", name), func(g *Game, img *ebiten.Image) { g.EnemyIconLargeImgs[name] = img })
	}

	for i := range 4 {
		add(assetTierBattle, fmt.Sprintf("assets/images/battle/player_attack_%d.png", i+1), func(g *Game, img *ebiten.Image) { g.PlayerAttackSprites[i] = img })
	}

	commandIconNames := [4]string{"attack", "skill", "wait", "flee"}
	for i, name := range commandIconNames {
		add(assetTierBattle, fmt.Sprintf("assets/images/battle/%s.png", name), func(g *Game, img *ebiten.Image) { g.CommandIcons[i] = img })
		add(assetTierBattle, fmt.Sprintf("assets/images/battle/%s_selected.png", name), func(g *Game, img *ebiten.Image) { g.CommandIconsSelected[i] = img })
	}

	for i := range 4 {
		add(assetTierBattle, fmt.Sprintf("assets/images/battle/timeline_p%d.png", i+1), func(g *Game, img *ebiten.Image) { g.TimelineIcons[i] = img })
		add(assetTierBattle, fmt.Sprintf("assets/images/battle/timeline_p%d_large.png", i+1), func(g *Game, img *ebiten.Image) { g.TimelineIconsLarge[i] = img })
	}

	add(assetTierBattle, "assets/images/battle/timeline_bar.png", func(g *Game, img *ebiten.Image) { g.TimelineBarImg = img })
	add(assetTierBattle, "assets/images/battle/gauge.png", func(g *Game, img *ebiten.Image) { g.GaugeImg = img })
	add(assetTierBattle, "assets/images/battle/goal.png", func(g *Game, img *ebiten.Image) { g.GoalImg = img })
	add(assetTierBattle, "assets/images/battle/item_button.png", func(g *Game, img *ebiten.Image) { g.ItemButtonImg = img })
	add(assetTierBattle, "assets/images/battle/timeline_bar_vertical.png", func(g *Game, img *ebiten.Image) { g.TimelineBarVertImg = img })
	add(assetTierBattle, "assets/images/battle/battle_bg.png", func(g *Game, img *ebiten.Image) { g.BattleBgImg = img })

	// statIconFileTags gives the assets/images/battle/stat_<tag>_{up,down}.png
	// filename fragment for each StatKind (StatAtk, StatMat, StatDef,
	// StatMdf, StatLuk in that order).
	statIconFileTags := [5]string{"atk", "mat", "def", "mdf", "luk"}
	for i, tag := range statIconFileTags {
		stat := i
		add(assetTierBattle, fmt.Sprintf("assets/images/battle/stat_%s_up.png", tag), func(g *Game, img *ebiten.Image) { g.StatIconUpImgs[stat] = img })
		add(assetTierBattle, fmt.Sprintf("assets/images/battle/stat_%s_down.png", tag), func(g *Game, img *ebiten.Image) { g.StatIconDownImgs[stat] = img })
	}

	add(assetTierBattle, "assets/images/battle/name_normal.png", func(g *Game, img *ebiten.Image) { g.NameImg = img })
	add(assetTierBattle, "assets/images/battle/name_myturn.png", func(g *Game, img *ebiten.Image) { g.NameMyTurnImg = img })
	add(assetTierBattle, "assets/images/battle/name_dead.png", func(g *Game, img *ebiten.Image) { g.NameDeadImg = img })

	return a
}
