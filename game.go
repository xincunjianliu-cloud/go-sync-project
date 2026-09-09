package main

import (
	"fmt"
	"image/color"
	"image/png"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	gameWidth   = 960
	gameHeight  = 540
	maxProgress = 100.0
	lineStartX  = 40.0
	lineWidth   = 400.0
)

const maxPlayerLevel = 50

// Scene はフィールド・バトル・メニューなど各画面が実装する共通インターフェース。
// Game は currentScene にこれを保持し、Update/Draw を委譲する。
type Scene interface {
	Update(dt float64) Scene
	Draw(screen *ebiten.Image)
}

// PlayerNames はパーティメンバーの表示名。
var PlayerNames = [partySize]string{
	"プレイヤー1",
	"プレイヤー2",
	"プレイヤー3",
	"プレイヤー4",
}

// BossNames はボスの表示名（1章1ボスで全4体）。
// 名前を変えたい時はここだけ書き換えればOK。
var BossNames = [4]string{
	"ボス1",
	"ボス2",
	"ボス3",
	"ボス4",
}

type Game struct {
	fontSource   *text.GoTextFaceSource
	currentScene Scene

	PlayerHP            [4]int
	PlayerMaxHP         [4]int
	PlayerMP            [4]int
	PlayerMaxMP         [4]int
	PlayerAtk           [4]int
	PlayerMagicAtk      [4]int
	PlayerDef           [4]int
	PlayerMagicDef      [4]int
	PlayerSpd           [4]int
	PlayerLuck          [4]int
	PlayerSP            [4]int
	PlayerSkillLv       [4][8]int // ← 修正：スキル数増加に対応するため4→8に拡張
	PlayerLv            [4]int
	PlayerEXP           [4]int
	PlayerNextEXP       [4]int
	PlayerAttackSprites [4]*ebiten.Image
	CommandIcons        [4]*ebiten.Image
	TimelineIcons       [4]*ebiten.Image
	SkillPanelImg       *ebiten.Image

	BossDefeatedFlags [4]bool
	EnemyImgs         map[string]*ebiten.Image
	Tilesets          map[string]*ebiten.Image
	TileImg           *ebiten.Image
	SpriteSheet       *ebiten.Image
	BossSpriteSheets  [4]*ebiten.Image
	BossImgs          [4]*ebiten.Image
	fadeAlpha         float64
	fadeMode          int
	fadeSpeed         float64
	pendingScene      Scene

	GoalImg  *ebiten.Image
	GaugeImg *ebiten.Image

	BattleBgImg *ebiten.Image
	BossBgImgs  [4]*ebiten.Image

	NameImg       *ebiten.Image // 通常時の名前プレート
	NameMyTurnImg *ebiten.Image // 自分のターン時の名前プレート

	TimelineBarImg     *ebiten.Image
	TimelineBarVertImg *ebiten.Image

	WindowImg *ebiten.Image
	CharaImgs map[string]*ebiten.Image

	LogEntryImg *ebiten.Image // 会話ログ1件分のテキストボックス背景

	// セーブ関連
	SaveConfirmBgImg     *ebiten.Image    // セーブ確認.png（はい/いいえ確認ダイアログの背景）
	SaveThumbFrameSelImg *ebiten.Image    // セーブスロット選択中.png（選択中の枠）
	SaveThumbFrameImg    *ebiten.Image    // セーブスロット.png（通常の枠）
	TotalPlayTime        float64          // 現在セッションの累積秒
	MenuEntryThumb       *ebiten.Image    // Mキーを押した瞬間のフィールド画面（96x54に縮小済み）
	SaveThumbs           [5]*ebiten.Image // ロード時に読み込んだサムネ

	MenuBgImg *ebiten.Image

	MenuSkillPanelImg *ebiten.Image

	PartyIconImgs [4]*ebiten.Image

	ExamineIconImg *ebiten.Image

	MessageSpeed int

	LastMenuIndex int // ← 追加：メニューの最後に選んだ項目を記憶

	LastSkillCharIndex   int
	LastSkillSubIndex    int
	LastSkillLevelCursor int
	LastStatusCharIndex  int

	// Game構造体に追加
	DisplayModeIndex int

	CurrentObjectiveID string

	MinimapPlayerIconImg    *ebiten.Image
	MinimapObjectiveIconImg *ebiten.Image

	Audio                  *AudioManager
	mouseLastX, mouseLastY int
	mouseIdleTime          float64
	cursorHidden           bool

	// ★追加：ダッシュのオン/オフ状態。マップ移動でFieldSceneが作り直されても
	// ダッシュ状態が消えないよう、シーンをまたいで生きるGame側に持たせる。
	IsDashing bool
}

const mouseIdleHideDelay = 2.0 // マウスカーソルを隠すまでの無操作時間（秒）

const (
	FadeNone = iota
	FadeOut
	FadeIn
)

func (g *Game) FontFace(size float64) *text.GoTextFace {
	return &text.GoTextFace{Source: g.fontSource, Size: size}
}

func NewGame(source *text.GoTextFaceSource) (*Game, error) {
	g := &Game{
		fontSource: source,
		Tilesets:   make(map[string]*ebiten.Image),
		CharaImgs:  make(map[string]*ebiten.Image),
	}

	g.PlayerHP[0], g.PlayerMaxHP[0] = 100, 100
	g.PlayerMP[0], g.PlayerMaxMP[0] = 15, 15
	g.PlayerAtk[0] = 15

	g.PlayerHP[1], g.PlayerMaxHP[1] = 70, 70
	g.PlayerMP[1], g.PlayerMaxMP[1] = 40, 40
	g.PlayerAtk[1] = 8

	g.PlayerHP[2], g.PlayerMaxHP[2] = 110, 110
	g.PlayerMP[2], g.PlayerMaxMP[2] = 20, 20
	g.PlayerAtk[2] = 12

	g.PlayerHP[3], g.PlayerMaxHP[3] = 90, 90
	g.PlayerMP[3], g.PlayerMaxMP[3] = 10, 10
	g.PlayerAtk[3] = 13

	// ★変更：すばやさ(PlayerSpd)は「タイムライン上でアイコンが進む速さ」そのものになったため、
	// キャラごとに個別の初期値を設定する（旧 playerSpeeds 配列の値を踏襲）。
	initialSpd := [4]int{24, 26, 22, 20}
	// ★変更：運(PlayerLuck)も会心率・回避率に使う実ステータスになったため、キャラごとに初期値を分ける。
	initialLuck := [4]int{5, 8, 4, 10}

	for i := 0; i < 4; i++ {
		g.PlayerMagicAtk[i] = 10 // 仮の初期値、後で個別調整
		g.PlayerDef[i] = 10
		g.PlayerMagicDef[i] = 10
		g.PlayerSpd[i] = initialSpd[i]
		g.PlayerLuck[i] = initialLuck[i]
		g.PlayerSP[i] = 0
	}

	for i := 0; i < 4; i++ {
		g.PlayerLv[i] = 1
		g.PlayerEXP[i] = 0
		// ★変更：固定値10ではなく、キャラごとに個別設定可能なEXPカーブ(stat_growth.go)から算出する。
		g.PlayerNextEXP[i] = PlayerExpCurve(i, 1)
	}

	// スキルレベルは1始まり。0のままだとLevels[lv-1]がLevels[-1]となりクラッシュするため必ず1で初期化する。
	for i := 0; i < 4; i++ {
		for j := 0; j < 8; j++ {
			g.PlayerSkillLv[i][j] = 1
		}
	}

	settings := LoadSettings()
	g.MessageSpeed = settings.MessageSpeed
	g.DisplayModeIndex = settings.DisplayModeIndex

	var err error

	roukaImg, _, err := ebitenutil.NewImageFromFile("assets/images/Tile_set_School_Set.png")
	if err != nil {
		return nil, fmt.Errorf("街のタイルセット画像の読み込みに失敗 %w", err)
	}
	g.Tilesets["rouka"] = roukaImg

	dungeonImg, _, err := ebitenutil.NewImageFromFile("assets/images/Tile_set_School_Set (12).png")
	if err != nil {
		return nil, fmt.Errorf("ダンジョンのタイルセット画像の読み込みに失敗 %w", err)
	}
	g.Tilesets["dungeon"] = dungeonImg

	defaultImg, _, err := ebitenutil.NewImageFromFile("assets/images/Tile_set_School_Set.png")
	if err != nil {
		return nil, fmt.Errorf("デフォルトタイルセット画像の読み込みに失敗 %w", err)
	}
	g.Tilesets["default"] = defaultImg

	g.TileImg = g.Tilesets["default"]

	if err := BuildObjectiveAndMapIndex(); err != nil {
		return nil, fmt.Errorf("目的地インデックスの構築に失敗 %w", err)
	}
	g.UpdateObjective()

	g.SpriteSheet, _, err = ebitenutil.NewImageFromFile("assets/images/player_walk.png")
	if err != nil {
		return nil, fmt.Errorf("プレイヤースプライトの読み込みに失敗 %w", err)
	}
	for i := 0; i < 4; i++ {
		path := fmt.Sprintf("assets/images/bossスプライト_%d.png", i+1)
		g.BossSpriteSheets[i], _, err = ebitenutil.NewImageFromFile(path)
		if err != nil {
			return nil, fmt.Errorf("ボス%dの歩行スプライト読み込みに失敗 %w", i+1, err)
		}
	}

	for i := 0; i < 4; i++ {
		path := fmt.Sprintf("assets/images/boss_%d.png", i+1)
		g.BossImgs[i], _, err = ebitenutil.NewImageFromFile(path)
		if err != nil {
			return nil, fmt.Errorf("ボス%dの画像読み込みに失敗 %w", i+1, err)
		}
	}

	g.EnemyImgs = make(map[string]*ebiten.Image)

	if g.EnemyImgs["フリーザ"], _, err = ebitenutil.NewImageFromFile("assets/images/スプライト-0001.png"); err != nil {
		return nil, fmt.Errorf("フリーザの画像読み込みに失敗 %w", err)
	}
	if g.EnemyImgs["セル"], _, err = ebitenutil.NewImageFromFile("assets/images/スプライト-0001.png"); err != nil {
		return nil, fmt.Errorf("セルの画像読み込みに失敗 %w", err)
	}
	if g.EnemyImgs["魔人ブウ"], _, err = ebitenutil.NewImageFromFile("assets/images/スプライト-0001.png"); err != nil {
		return nil, fmt.Errorf("魔人ブウの画像読み込みに失敗 %w", err)
	}

	for i := 0; i < 4; i++ {
		path := fmt.Sprintf("assets/images/player_attack_%d.png", i+1)
		g.PlayerAttackSprites[i], _, err = ebitenutil.NewImageFromFile(path)
		if err != nil {
			if g.SpriteSheet != nil {
				g.PlayerAttackSprites[i] = g.SpriteSheet
				fmt.Printf("警告: %s が見つからないため、既存のSpriteSheetで代用します\n", path)
			} else {
				return nil, fmt.Errorf("プレイヤー%dの攻撃・アクション画像ロード失敗: %w", i+1, err)
			}
		}
	}

	commandIconFiles := [4]string{
		"assets/images/アタック.png",
		"assets/images/スキル.png",
		"assets/images/待機.png",
		"assets/images/逃げる.png",
	}
	for i, path := range commandIconFiles {
		img, _, ferr := ebitenutil.NewImageFromFile(path)
		if ferr != nil {
			fmt.Printf("警告: %s の読み込みに失敗しました（アイコン非表示で続行）\n", path)
			g.CommandIcons[i] = nil
			continue
		}
		g.CommandIcons[i] = img
	}

	timelineIconFiles := [4]string{
		"assets/images/timeline_p1.png",
		"assets/images/timeline_p2.png",
		"assets/images/timeline_p3.png",
		"assets/images/timeline_p4.png",
	}
	for i, path := range timelineIconFiles {
		img, _, ferr := ebitenutil.NewImageFromFile(path)
		if ferr != nil {
			fmt.Printf("警告: %s の読み込みに失敗しました（デフォルト描画で代替）\n", path)
			g.TimelineIcons[i] = nil
			continue
		}
		g.TimelineIcons[i] = img
	}

	g.TimelineBarImg, _, err = ebitenutil.NewImageFromFile("assets/images/タイムライン横.png")
	if err != nil {
		fmt.Printf("警告: タイムラインバー画像の読み込みに失敗しました（矩形描画で代替します）\n")
		g.TimelineBarImg = nil
	}

	g.GaugeImg, _, err = ebitenutil.NewImageFromFile("assets/images/ゲージ.png")
	if err != nil {
		fmt.Printf("警告: ゲージ画像の読み込みに失敗しました\n")
		g.GaugeImg = nil
	}

	g.GoalImg, _, err = ebitenutil.NewImageFromFile("assets/images/ゴール.png")
	if err != nil {
		fmt.Printf("警告: ゴール画像の読み込みに失敗しました\n")
		g.GoalImg = nil
	}

	g.TimelineBarVertImg, _, err = ebitenutil.NewImageFromFile("assets/images/タイムラインバー縦.png")
	if err != nil {
		fmt.Printf("警告: タイムラインバー縦画像の読み込みに失敗しました\n")
		g.TimelineBarVertImg = nil
	}

	g.SkillPanelImg, _, err = ebitenutil.NewImageFromFile("assets/images/スキル拡張.png")
	if err != nil {
		fmt.Printf("警告: スキル拡張画像の読み込みに失敗しました\n")
		g.SkillPanelImg = nil
	}

	g.BattleBgImg, _, err = ebitenutil.NewImageFromFile("assets/images/battle_bg.png")
	if err != nil {
		fmt.Printf("警告: 通常戦闘の背景画像読み込みに失敗しました\n")
		g.BattleBgImg = nil
	}

	for i := 0; i < 4; i++ {
		path := fmt.Sprintf("assets/images/battle_bg_boss_%d.png", i+1)
		img, _, ferr := ebitenutil.NewImageFromFile(path)
		if ferr != nil {
			fmt.Printf("警告: %s の読み込みに失敗しました（ボス%dは通常背景で代替）\n", path, i+1)
			g.BossBgImgs[i] = nil
			continue
		}
		g.BossBgImgs[i] = img
	}

	g.WindowImg, _, _ = ebitenutil.NewImageFromFile("assets/images/window.png")

	g.NameImg, _, err = ebitenutil.NewImageFromFile("assets/images/name_normal.png")
	if err != nil {
		fmt.Printf("警告: 名前画像(通常)の読み込みに失敗しました（文字表示で代替します）\n")
		g.NameImg = nil
	}

	g.NameMyTurnImg, _, err = ebitenutil.NewImageFromFile("assets/images/name_myturn.png")
	if err != nil {
		fmt.Printf("警告: 名前画像(自分のターン)の読み込みに失敗しました（通常画像で代替します）\n")
		g.NameMyTurnImg = nil
	}

	g.LogEntryImg, _, err = ebitenutil.NewImageFromFile("assets/images/log_entry_box.png")
	if err != nil {
		fmt.Printf("警告: ログウィンドウ画像の読み込みに失敗しました（単色背景で代替します）\n")
		g.LogEntryImg = nil
	}

	g.ExamineIconImg, _, err = ebitenutil.NewImageFromFile("assets/images/調べる.png")
	if err != nil {
		fmt.Printf("警告: 調べるアイコン画像の読み込みに失敗しました\n")
		g.ExamineIconImg = nil
	}

	g.MinimapPlayerIconImg, _, err = ebitenutil.NewImageFromFile("assets/images/現在地.png")
	if err != nil {
		fmt.Printf("警告: 現在地アイコン画像の読み込みに失敗しました\n")
		g.MinimapPlayerIconImg = nil
	}

	g.MinimapObjectiveIconImg, _, err = ebitenutil.NewImageFromFile("assets/images/目的地.png")
	if err != nil {
		fmt.Printf("警告: 目的地アイコン画像の読み込みに失敗しました\n")
		g.MinimapObjectiveIconImg = nil
	}

	g.Audio = NewAudioManager()
	g.Audio.SetVolume(settings.BGMVolume) // ← 追加

	g.currentScene = NewTitleScene(g)

	for i := 0; i < 4; i++ {
		bossNum := i + 1
		path := fmt.Sprintf("assets/images/chara_boss%d.png", bossNum)
		img, _, ferr := ebitenutil.NewImageFromFile(path)
		if ferr != nil {
			fmt.Printf("警告: %s の読み込みに失敗（立ち絵非表示で続行）\n", path)
			continue
		}
		g.CharaImgs[BossNames[i]] = img
	}

	// ★追加：味方4人の立ち絵
	for i := 0; i < partySize; i++ {
		path := fmt.Sprintf("assets/images/chara_player_%d.png", i+1)
		img, _, ferr := ebitenutil.NewImageFromFile(path)
		if ferr != nil {
			fmt.Printf("警告: %s の読み込みに失敗（立ち絵非表示で続行）\n", path)
			continue
		}
		g.CharaImgs[PlayerNames[i]] = img
	}

	g.MenuBgImg, _, err = ebitenutil.NewImageFromFile("assets/images/メニュー画面.png")
	if err != nil {
		return nil, fmt.Errorf("メニュー背景画像の読み込みに失敗 %w", err)
	}

	g.MenuSkillPanelImg, _, err = ebitenutil.NewImageFromFile("assets/images/メニュー画面拡張.png")
	if err != nil {
		fmt.Printf("警告: メニュースキルパネル画像の読み込みに失敗しました\n")
		g.MenuSkillPanelImg = nil
	}

	partyIconFiles := [4]string{
		"assets/images/party_icon_1.png",
		"assets/images/party_icon_2.png",
		"assets/images/party_icon_3.png",
		"assets/images/party_icon_4.png",
	}
	for i, path := range partyIconFiles {
		img, _, ferr := ebitenutil.NewImageFromFile(path)
		if ferr != nil {
			fmt.Printf("警告: %s の読み込みに失敗しました（丸枠のみ表示）\n", path)
			g.PartyIconImgs[i] = nil
			continue
		}
		g.PartyIconImgs[i] = img
	}

	g.SaveConfirmBgImg, _, err = ebitenutil.NewImageFromFile("assets/images/セーブ確認.png")
	if err != nil {
		fmt.Printf("警告: セーブ確認画像の読み込みに失敗しました\n")
		g.SaveConfirmBgImg = nil
	}
	g.SaveThumbFrameSelImg, _, err = ebitenutil.NewImageFromFile("assets/images/セーブスロット選択中.png")
	if err != nil {
		fmt.Printf("警告: セーブ枠(選択中)画像の読み込みに失敗しました\n")
		g.SaveThumbFrameSelImg = nil
	}
	g.SaveThumbFrameImg, _, err = ebitenutil.NewImageFromFile("assets/images/セーブスロット.png")
	if err != nil {
		fmt.Printf("警告: セーブ枠画像の読み込みに失敗しました\n")
		g.SaveThumbFrameImg = nil
	}

	applyDisplayMode(g.DisplayModeIndex)

	if err := LoadDialogues("assets/dialogues"); err != nil {
		return nil, fmt.Errorf("セリフデータの読み込みに失敗 %w", err)
	}
	validateDialogueSlots()

	return g, nil
}

func (g *Game) Update() error {
	dt := 1.0 / 60.0

	g.updateMouseCursorVisibility(dt) // ★追加

	// プレイ時間の累積（フェード中も含めて常に加算）
	g.TotalPlayTime += dt
	g.Audio.Update(dt)

	if g.fadeMode == FadeOut {
		g.fadeAlpha += g.fadeSpeed * dt
		if g.fadeAlpha >= 1.0 {
			g.fadeAlpha = 1.0
			g.currentScene = g.pendingScene
			g.fadeMode = FadeIn
		}
		return nil
	} else if g.fadeMode == FadeIn {
		g.fadeAlpha -= g.fadeSpeed * dt
		if g.fadeAlpha <= 0 {
			g.fadeAlpha = 0
			g.fadeMode = FadeNone
		}
		return nil
	}

	if g.currentScene != nil {
		nextScene := g.currentScene.Update(dt)
		if nextScene != nil && nextScene != g.currentScene {
			g.pendingScene = nextScene
			g.fadeMode = FadeOut
			g.fadeAlpha = 0.0
			g.fadeSpeed = 1.0 / fadeTimeBattleOut
		}
	}
	return nil
}

func (g *Game) ChangeSceneWithFade(nextScene Scene, durationSeconds float64) {
	g.pendingScene = nextScene
	g.fadeMode = FadeOut
	g.fadeAlpha = 0.0
	g.fadeSpeed = 1.0 / durationSeconds
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.currentScene != nil {
		g.currentScene.Draw(screen)
	}

	if g.fadeAlpha > 0 {
		ebitenutil.DrawRect(screen, 0, 0, float64(gameWidth), float64(gameHeight), color.NRGBA{0, 0, 0, uint8(255 * g.fadeAlpha)})
	}
}
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return gameWidth, gameHeight
}

func (g *Game) GetEnemyImage(name string) *ebiten.Image {
	if img, exists := g.EnemyImgs[name]; exists {
		return img
	}
	if len(g.BossImgs) > 0 {
		return g.BossImgs[0]
	}
	return nil
}

func (g *Game) captureMenuEntryThumb(full *ebiten.Image) {
	thumb := ebiten.NewImage(196, 110)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(196.0/float64(gameWidth), 110.0/float64(gameHeight))
	thumb.DrawImage(full, op)
	g.MenuEntryThumb = thumb
}

// saveThumbToFile は保持しているサムネイルを指定スロットのPNGとして書き出す。
func (g *Game) saveThumbToFile(slot int) {
	if g.MenuEntryThumb == nil {
		return
	}
	path := fmt.Sprintf("save_thumb_%d.png", slot)
	f, err := os.Create(path)
	if err != nil {
		fmt.Printf("警告: サムネイルの保存に失敗しました: %v\n", err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, g.MenuEntryThumb); err != nil {
		fmt.Printf("警告: サムネイルのエンコードに失敗しました: %v\n", err)
	}
}

// FormatPlayTime は秒数を "HHH:MM" 形式（最大999:59）に変換する
func FormatPlayTime(seconds float64) string {
	m := int(seconds) / 60
	h := m / 60
	if h > 999 {
		h = 999
		m = 999*60 + 59
	}
	return fmt.Sprintf("%03d:%02d", h, m%60)
}

// MessageSpeedTicks は現在のメッセージ速度設定に応じた、
// 1文字表示するのに必要なtick数を返す（値が小さいほど速い）
func (g *Game) MessageSpeedTicks() int {
	switch g.MessageSpeed {
	case 0:
		return 9 // 遅い
	case 2:
		return 2 // 速い
	default:
		return 5 // 普通
	}
}

func (g *Game) updateMouseCursorVisibility(dt float64) {
	x, y := ebiten.CursorPosition()
	if x != g.mouseLastX || y != g.mouseLastY {
		g.mouseLastX, g.mouseLastY = x, y
		g.mouseIdleTime = 0
		if g.cursorHidden {
			ebiten.SetCursorMode(ebiten.CursorModeVisible)
			g.cursorHidden = false
		}
		return
	}

	g.mouseIdleTime += dt
	if !g.cursorHidden && g.mouseIdleTime >= mouseIdleHideDelay {
		ebiten.SetCursorMode(ebiten.CursorModeHidden)
		g.cursorHidden = true
	}
}
