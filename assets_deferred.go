package main

import (
	"fmt"
	"image"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// deferredAssetAssign はパス1件と、デコード済み画像をGameのどのフィールドに
// 格納するかを表す。起動を速くするため、タイトル画面の表示に不要な画像は
// すべてこのテーブル経由でバックグラウンド読み込みする。
type deferredAssetAssign struct {
	path   string
	assign func(*Game, *ebiten.Image)
}

// decodedHeavyAsset はバックグラウンドgoroutineからUpdate()側へ結果を渡すための
// メッセージ。画像デコードまではgoroutine側で行い、ebiten.Imageの生成
// (GPUテクスチャ確保)は必ずUpdate()を呼ぶメインゴルーチン側で行う
// (Ebitengineのグラフィックス関連APIはメインゴルーチン以外からの呼び出しが
// 保証されていないため)。
type decodedHeavyAsset struct {
	assign func(*Game, *ebiten.Image)
	img    image.Image
	err    error
	label  string
}

// loadHeavyAssetsAsync はタイトル画面表示後にバックグラウンドで実行され、
// 戦闘・ボス・敵・メニューなど、タイトル画面自体には不要な画像とデータを
// まとめて読み込む。完了(またはエラー)するとg.heavyDecodedを閉じる。
func (g *Game) loadHeavyAssetsAsync() {
	defer close(g.heavyDecoded)

	if err := BuildObjectiveAndMapIndex(); err != nil {
		g.heavyDecoded <- decodedHeavyAsset{err: err, label: "目的地インデックスの構築"}
		return
	}
	g.UpdateObjective()

	if err := LoadDialogues("assets/dialogues"); err != nil {
		g.heavyDecoded <- decodedHeavyAsset{err: err, label: "セリフデータの読み込み"}
		return
	}

	assignments := deferredAssetAssignments()
	tilesets := mapTilesetAssignments(assignments)
	paths := make([]string, 0, len(assignments)+len(tilesets)+1)
	for _, a := range assignments {
		paths = append(paths, a.path)
	}
	for _, a := range tilesets {
		paths = append(paths, a.path)
	}
	paths = append(paths, fieldPlayerConfigPath)
	prefetchAssetBytes(paths)

	for _, a := range assignments {
		img, err := decodeAssetImage(a.path)
		g.heavyDecoded <- decodedHeavyAsset{assign: a.assign, img: img, err: err, label: a.path}
		yieldToBrowser()
	}

	// 各マップのタイルセットは、読み込めなくてもそのマップに入った時点で
	// NewRoomSceneが改めて読み込み・エラー表示するので、ここでの失敗は
	// ゲーム全体の読み込みエラー扱いにはしない。
	for _, a := range tilesets {
		img, err := decodeAssetImage(a.path)
		if err == nil {
			g.heavyDecoded <- decodedHeavyAsset{assign: a.assign, img: img, label: a.path}
		}
		yieldToBrowser()
	}
}

// mapTilesetAssignments はBuildObjectiveAndMapIndexが見つけた全マップの
// タイルセット画像を、mapTilesetImageのキャッシュへ先に入れておくための
// 一覧を返す。これが無いと、初めて入るマップのタイルセット画像を
// ドア移動やロードの瞬間に同期デコードすることになり、画面が固まる。
// alreadyに同じパスがある画像(デフォルトのタイルセット)は二重に読まない。
func mapTilesetAssignments(already []deferredAssetAssign) []deferredAssetAssign {
	seen := make(map[string]bool, len(already))
	for _, a := range already {
		seen[a.path] = true
	}
	var out []deferredAssetAssign
	for _, mapPath := range allMapPaths {
		tmap, err := loadTiledMap(mapPath)
		if err != nil {
			continue
		}
		p, ok := mapTilesetImagePath(tmap)
		if !ok || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, deferredAssetAssign{p, func(g *Game, img *ebiten.Image) { tilesetImageCache[p] = img }})
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
				g.heavyAssetsReady = true
				if g.heavyAssetsErr == nil {
					g.TileImg = g.Tilesets["default"]
				}
				g.prewarmAudio()
				return
			}
			if item.err != nil {
				if g.heavyAssetsErr == nil {
					g.heavyAssetsErr = fmt.Errorf("%sに失敗しました: %w", item.label, item.err)
				}
				continue
			}
			if item.assign != nil {
				item.assign(g, ebiten.NewImageFromImage(item.img))
			}
		default:
			return
		}
	}
}

// prewarmAudio は画像の読み込みが一通り終わった後、次に鳴りそうな曲と
// 全効果音をバックグラウンドで先にデコードしておく。これで最初の戦闘開始・
// 勝利・ニューゲーム時に、曲のデコード待ちが発生しにくくなる。
// (BGMは1曲あたり数十MBのPCMになるので、全曲ではなく使用頻度の高いものに絞る)
func (g *Game) prewarmAudio() {
	paths := []string{bgmBattleNormal, bgmVictoryIntro, bgmVictoryLoop, bgmField1}
	if tmap, err := loadTiledMap(startMapPath); err == nil {
		if key, ok := tmap.mapBGMKey(); ok {
			if p, found := resolveBGMKey(key); found {
				paths = append(paths, p)
			}
		}
	}
	for _, p := range seByKey {
		paths = append(paths, p)
	}
	g.Audio.Prewarm(paths...)
}

// deferredAssetAssignments はバックグラウンドで読み込む画像の一覧と、
// 読み込み後にGameのどのフィールドへ入れるかを返す。
func deferredAssetAssignments() []deferredAssetAssign {
	// 各マップ自身のタイルセット画像は.tmjの"tilesets"欄からmapTilesetImageが
	// 都度読み込むため、ここでの事前登録は不要。"default"は、万一マップに
	// タイルセットが設定されていない場合のフォールバック用に残しておく。
	a := []deferredAssetAssign{
		{"assets/images/field/Tile_set_School_Set.png", func(g *Game, img *ebiten.Image) {
			g.Tilesets["default"] = img
			tilesetImageCache["assets/images/field/Tile_set_School_Set.png"] = img
		}},
		{"assets/images/field/player_walk.png", func(g *Game, img *ebiten.Image) { g.SpriteSheet = img }},
	}

	for i := 0; i < 4; i++ {
		a = append(a,
			deferredAssetAssign{fmt.Sprintf("assets/images/field/bossスプライト_%d.png", i+1), func(g *Game, img *ebiten.Image) { g.BossSpriteSheets[i] = img }},
			deferredAssetAssign{fmt.Sprintf("assets/images/battle/boss_%d.png", i+1), func(g *Game, img *ebiten.Image) { g.BossImgs[i] = img }},
			deferredAssetAssign{fmt.Sprintf("assets/images/battle/boss_%d_icon.png", i+1), func(g *Game, img *ebiten.Image) { g.BossIconImgs[i] = img }},
			deferredAssetAssign{fmt.Sprintf("assets/images/battle/boss_%d_icon_large.png", i+1), func(g *Game, img *ebiten.Image) { g.BossIconLargeImgs[i] = img }},
		)
	}

	for _, enemy := range EnemyDatabase {
		name := enemy.Name
		a = append(a,
			deferredAssetAssign{fmt.Sprintf("assets/images/battle/enemy_%s.png", name), func(g *Game, img *ebiten.Image) { g.EnemyImgs[name] = img }},
			deferredAssetAssign{fmt.Sprintf("assets/images/battle/enemy_%s_icon.png", name), func(g *Game, img *ebiten.Image) { g.EnemyIconImgs[name] = img }},
			deferredAssetAssign{fmt.Sprintf("assets/images/battle/enemy_%s_icon_large.png", name), func(g *Game, img *ebiten.Image) { g.EnemyIconLargeImgs[name] = img }},
		)
	}

	for i := 0; i < 4; i++ {
		a = append(a, deferredAssetAssign{fmt.Sprintf("assets/images/battle/player_attack_%d.png", i+1), func(g *Game, img *ebiten.Image) { g.PlayerAttackSprites[i] = img }})
	}

	commandIconFiles := [4]string{
		"assets/images/battle/attack.png",
		"assets/images/battle/skill.png",
		"assets/images/battle/wait.png",
		"assets/images/battle/flee.png",
	}
	for i, path := range commandIconFiles {
		a = append(a, deferredAssetAssign{path, func(g *Game, img *ebiten.Image) { g.CommandIcons[i] = img }})
	}

	commandIconSelectedFiles := [4]string{
		"assets/images/battle/attack_selected.png",
		"assets/images/battle/skill_selected.png",
		"assets/images/battle/wait_selected.png",
		"assets/images/battle/flee_selected.png",
	}
	for i, path := range commandIconSelectedFiles {
		a = append(a, deferredAssetAssign{path, func(g *Game, img *ebiten.Image) { g.CommandIconsSelected[i] = img }})
	}

	for i := 0; i < 4; i++ {
		a = append(a, deferredAssetAssign{fmt.Sprintf("assets/images/battle/timeline_p%d.png", i+1), func(g *Game, img *ebiten.Image) { g.TimelineIcons[i] = img }})
	}
	for i := 0; i < 4; i++ {
		a = append(a, deferredAssetAssign{fmt.Sprintf("assets/images/battle/timeline_p%d_large.png", i+1), func(g *Game, img *ebiten.Image) { g.TimelineIconsLarge[i] = img }})
	}

	a = append(a,
		deferredAssetAssign{"assets/images/battle/timeline_bar.png", func(g *Game, img *ebiten.Image) { g.TimelineBarImg = img }},
		deferredAssetAssign{"assets/images/battle/gauge.png", func(g *Game, img *ebiten.Image) { g.GaugeImg = img }},
		deferredAssetAssign{"assets/images/battle/goal.png", func(g *Game, img *ebiten.Image) { g.GoalImg = img }},
		deferredAssetAssign{"assets/images/battle/timeline_bar_vertical.png", func(g *Game, img *ebiten.Image) { g.TimelineBarVertImg = img }},
		deferredAssetAssign{"assets/images/battle/skill_panel.png", func(g *Game, img *ebiten.Image) { g.SkillPanelImg = img }},
		deferredAssetAssign{"assets/images/battle/battle_bg.png", func(g *Game, img *ebiten.Image) { g.BattleBgImg = img }},
		deferredAssetAssign{"assets/images/field/map_name_banner.png", func(g *Game, img *ebiten.Image) { g.MapNameBannerImg = img }},
	)

	for i := 0; i < 4; i++ {
		a = append(a, deferredAssetAssign{fmt.Sprintf("assets/images/battle/battle_bg_boss_%d.png", i+1), func(g *Game, img *ebiten.Image) { g.BossBgImgs[i] = img }})
	}

	// statIconFileTags gives the assets/images/battle/stat_<tag>_{up,down}.png
	// filename fragment for each StatKind (StatAtk, StatMat, StatDef,
	// StatMdf, StatLuk in that order).
	statIconFileTags := [5]string{"atk", "mat", "def", "mdf", "luk"}
	for i, tag := range statIconFileTags {
		stat := i
		a = append(a,
			deferredAssetAssign{fmt.Sprintf("assets/images/battle/stat_%s_up.png", tag), func(g *Game, img *ebiten.Image) { g.StatIconUpImgs[stat] = img }},
			deferredAssetAssign{fmt.Sprintf("assets/images/battle/stat_%s_down.png", tag), func(g *Game, img *ebiten.Image) { g.StatIconDownImgs[stat] = img }},
		)
	}

	a = append(a,
		deferredAssetAssign{"assets/images/common/window.png", func(g *Game, img *ebiten.Image) { g.WindowImg = img }},
		deferredAssetAssign{"assets/images/battle/name_normal.png", func(g *Game, img *ebiten.Image) { g.NameImg = img }},
		deferredAssetAssign{"assets/images/battle/name_myturn.png", func(g *Game, img *ebiten.Image) { g.NameMyTurnImg = img }},
		deferredAssetAssign{"assets/images/battle/name_dead.png", func(g *Game, img *ebiten.Image) { g.NameDeadImg = img }},
		deferredAssetAssign{"assets/images/battle/log_entry_box.png", func(g *Game, img *ebiten.Image) { g.LogEntryImg = img }},
		deferredAssetAssign{"assets/images/field/調べる.png", func(g *Game, img *ebiten.Image) { g.ExamineIconImg = img }},
		deferredAssetAssign{"assets/images/field/現在地.png", func(g *Game, img *ebiten.Image) { g.MinimapPlayerIconImg = img }},
		deferredAssetAssign{"assets/images/field/目的地.png", func(g *Game, img *ebiten.Image) { g.MinimapObjectiveIconImg = img }},
		deferredAssetAssign{"assets/images/field/chest.png", func(g *Game, img *ebiten.Image) { g.ChestImg = img }},
		deferredAssetAssign{"assets/images/field/key_chest.png", func(g *Game, img *ebiten.Image) { g.KeyChestImg = img }},
		deferredAssetAssign{"assets/images/field/locked_wall.png", func(g *Game, img *ebiten.Image) { g.LockedWallImg = img }},
		deferredAssetAssign{"assets/images/field/lever_wall_open.png", func(g *Game, img *ebiten.Image) { g.LeverWallOpenImg = img }},
		deferredAssetAssign{"assets/images/field/lever_wall_open_deco.png", func(g *Game, img *ebiten.Image) { g.LeverWallOpenDecoImg = img }},
		deferredAssetAssign{"assets/images/field/lever.png", func(g *Game, img *ebiten.Image) { g.LeverImg = img }},
		deferredAssetAssign{"assets/images/field/push_block.png", func(g *Game, img *ebiten.Image) { g.BlockImg = img }},
		deferredAssetAssign{"assets/images/field/push_block_spot.png", func(g *Game, img *ebiten.Image) { g.BlockSpotImg = img }},
	)

	for i := 0; i < 4; i++ {
		bossName := BossNames[i]
		slug := fmt.Sprintf("boss%d", i+1)
		a = append(a, deferredAssetAssign{fmt.Sprintf("assets/images/common/chara_%s.png", slug), func(g *Game, img *ebiten.Image) { g.CharaImgs[bossName] = img }})
	}
	for i := 0; i < partySize; i++ {
		playerName := PlayerNames[i]
		slug := fmt.Sprintf("player_%d", i+1)
		a = append(a, deferredAssetAssign{fmt.Sprintf("assets/images/common/chara_%s.png", slug), func(g *Game, img *ebiten.Image) { g.CharaImgs[playerName] = img }})
	}

	a = append(a,
		deferredAssetAssign{"assets/images/menu/メニュー画面.png", func(g *Game, img *ebiten.Image) { g.MenuBgImg = img }},
		deferredAssetAssign{"assets/images/menu/メニュー画面拡張.png", func(g *Game, img *ebiten.Image) { g.MenuSkillPanelImg = img }},
	)

	partyIconFiles := [4]string{
		"assets/images/menu/party_icon_1.png",
		"assets/images/menu/party_icon_2.png",
		"assets/images/menu/party_icon_3.png",
		"assets/images/menu/party_icon_4.png",
	}
	for i, path := range partyIconFiles {
		a = append(a, deferredAssetAssign{path, func(g *Game, img *ebiten.Image) { g.PartyIconImgs[i] = img }})
	}

	a = append(a,
		deferredAssetAssign{"assets/images/menu/セーブスロット選択中.png", func(g *Game, img *ebiten.Image) { g.SaveThumbFrameSelImg = img }},
		deferredAssetAssign{"assets/images/menu/セーブスロット.png", func(g *Game, img *ebiten.Image) { g.SaveThumbFrameImg = img }},
	)

	return a
}
