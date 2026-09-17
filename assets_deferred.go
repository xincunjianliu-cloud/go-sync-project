package main

import (
	"fmt"
	"image"

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
	validateDialogueSlots()

	assignments := deferredAssetAssignments()
	paths := make([]string, len(assignments))
	for i, a := range assignments {
		paths[i] = a.path
	}
	prefetchAssetBytes(paths)

	for _, a := range assignments {
		img, err := decodeAssetImage(a.path)
		g.heavyDecoded <- decodedHeavyAsset{assign: a.assign, img: img, err: err, label: a.path}
	}
}

// pumpHeavyAssets はUpdate()から毎フレーム呼ばれ、バックグラウンドで
// デコード済みの画像をebiten.Imageへ変換してGameへ反映する。
// チャンネルに溜まっている分を1フレームで一気に処理するが、送信側
// (loadHeavyAssetsAsync)がネットワーク待ちで詰まっている間はdefaultに
// 抜けてブロックしない。
func (g *Game) pumpHeavyAssets() {
	if g.heavyAssetsReady || g.heavyDecoded == nil {
		return
	}
	for {
		select {
		case item, ok := <-g.heavyDecoded:
			if !ok {
				g.heavyAssetsReady = true
				if g.heavyAssetsErr == nil {
					g.TileImg = g.Tilesets["default"]
				}
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

// deferredAssetAssignments はバックグラウンドで読み込む画像の一覧と、
// 読み込み後にGameのどのフィールドへ入れるかを返す。
func deferredAssetAssignments() []deferredAssetAssign {
	a := []deferredAssetAssign{
		{"assets/images/field/Tile_set_School_Set.png", func(g *Game, img *ebiten.Image) { g.Tilesets["rouka"] = img }},
		{"assets/images/field/Tile_set_School_Set (12).png", func(g *Game, img *ebiten.Image) { g.Tilesets["dungeon"] = img }},
		{"assets/images/field/Tile_set_School_Set.png", func(g *Game, img *ebiten.Image) { g.Tilesets["default"] = img }},
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
		"assets/images/battle/アタック.png",
		"assets/images/battle/スキル.png",
		"assets/images/battle/待機.png",
		"assets/images/battle/逃げる.png",
	}
	for i, path := range commandIconFiles {
		a = append(a, deferredAssetAssign{path, func(g *Game, img *ebiten.Image) { g.CommandIcons[i] = img }})
	}

	for i := 0; i < 4; i++ {
		a = append(a, deferredAssetAssign{fmt.Sprintf("assets/images/battle/timeline_p%d.png", i+1), func(g *Game, img *ebiten.Image) { g.TimelineIcons[i] = img }})
	}
	for i := 0; i < 4; i++ {
		a = append(a, deferredAssetAssign{fmt.Sprintf("assets/images/battle/timeline_p%d_large.png", i+1), func(g *Game, img *ebiten.Image) { g.TimelineIconsLarge[i] = img }})
	}

	a = append(a,
		deferredAssetAssign{"assets/images/battle/タイムライン横.png", func(g *Game, img *ebiten.Image) { g.TimelineBarImg = img }},
		deferredAssetAssign{"assets/images/battle/ゲージ.png", func(g *Game, img *ebiten.Image) { g.GaugeImg = img }},
		deferredAssetAssign{"assets/images/battle/ゴール.png", func(g *Game, img *ebiten.Image) { g.GoalImg = img }},
		deferredAssetAssign{"assets/images/battle/タイムラインバー縦.png", func(g *Game, img *ebiten.Image) { g.TimelineBarVertImg = img }},
		deferredAssetAssign{"assets/images/battle/スキル拡張.png", func(g *Game, img *ebiten.Image) { g.SkillPanelImg = img }},
		deferredAssetAssign{"assets/images/battle/battle_bg.png", func(g *Game, img *ebiten.Image) { g.BattleBgImg = img }},
	)

	for i := 0; i < 4; i++ {
		a = append(a, deferredAssetAssign{fmt.Sprintf("assets/images/battle/battle_bg_boss_%d.png", i+1), func(g *Game, img *ebiten.Image) { g.BossBgImgs[i] = img }})
	}

	a = append(a,
		deferredAssetAssign{"assets/images/common/window.png", func(g *Game, img *ebiten.Image) { g.WindowImg = img }},
		deferredAssetAssign{"assets/images/battle/name_normal.png", func(g *Game, img *ebiten.Image) { g.NameImg = img }},
		deferredAssetAssign{"assets/images/battle/name_myturn.png", func(g *Game, img *ebiten.Image) { g.NameMyTurnImg = img }},
		deferredAssetAssign{"assets/images/battle/log_entry_box.png", func(g *Game, img *ebiten.Image) { g.LogEntryImg = img }},
		deferredAssetAssign{"assets/images/field/調べる.png", func(g *Game, img *ebiten.Image) { g.ExamineIconImg = img }},
		deferredAssetAssign{"assets/images/field/現在地.png", func(g *Game, img *ebiten.Image) { g.MinimapPlayerIconImg = img }},
		deferredAssetAssign{"assets/images/field/目的地.png", func(g *Game, img *ebiten.Image) { g.MinimapObjectiveIconImg = img }},
		deferredAssetAssign{"assets/images/field/chest.png", func(g *Game, img *ebiten.Image) { g.ChestImg = img }},
		deferredAssetAssign{"assets/images/field/key_chest.png", func(g *Game, img *ebiten.Image) { g.KeyChestImg = img }},
		deferredAssetAssign{"assets/images/field/locked_wall.png", func(g *Game, img *ebiten.Image) { g.LockedWallImg = img }},
		deferredAssetAssign{"assets/images/field/lever_wall.png", func(g *Game, img *ebiten.Image) { g.LeverWallImg = img }},
		deferredAssetAssign{"assets/images/field/lever.png", func(g *Game, img *ebiten.Image) { g.LeverImg = img }},
	)

	for i := 0; i < 4; i++ {
		bossName := BossNames[i]
		a = append(a, deferredAssetAssign{fmt.Sprintf("assets/images/common/chara_boss%d.png", i+1), func(g *Game, img *ebiten.Image) { g.CharaImgs[bossName] = img }})
	}
	for i := 0; i < partySize; i++ {
		playerName := PlayerNames[i]
		a = append(a, deferredAssetAssign{fmt.Sprintf("assets/images/common/chara_player_%d.png", i+1), func(g *Game, img *ebiten.Image) { g.CharaImgs[playerName] = img }})
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
