package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
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

type Scene interface {
	Update(dt float64) Scene
	Draw(screen *ebiten.Image)
}

var PlayerNames = [partySize]string{
	"ブロリー",
	"パラガス",
	"ベジータ",
	"トランクスルー",
}

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
	PlayerSkillLv       [4][8]int
	PlayerLv            [4]int
	PlayerEXP           [4]int
	PlayerNextEXP       [4]int
	PlayerAttackSprites [4]*ebiten.Image
	CommandIcons        [4]*ebiten.Image
	TimelineIcons       [4]*ebiten.Image
	TimelineIconsLarge  [4]*ebiten.Image
	SkillPanelImg       *ebiten.Image

	BossDefeatedFlags  [4]bool
	EnemyImgs          map[string]*ebiten.Image
	EnemyIconImgs      map[string]*ebiten.Image
	EnemyIconLargeImgs map[string]*ebiten.Image
	Tilesets           map[string]*ebiten.Image
	TileImg            *ebiten.Image
	SpriteSheet        *ebiten.Image
	BossSpriteSheets   [4]*ebiten.Image
	BossImgs           [4]*ebiten.Image
	BossIconImgs       [4]*ebiten.Image
	BossIconLargeImgs  [4]*ebiten.Image
	fadeAlpha          float64
	fadeMode           int
	fadeSpeed          float64
	pendingScene       Scene

	GoalImg  *ebiten.Image
	GaugeImg *ebiten.Image

	BattleBgImg *ebiten.Image
	BossBgImgs  [4]*ebiten.Image

	NameImg       *ebiten.Image
	NameMyTurnImg *ebiten.Image

	TimelineBarImg     *ebiten.Image
	TimelineBarVertImg *ebiten.Image

	WindowImg *ebiten.Image
	CharaImgs map[string]*ebiten.Image

	LogEntryImg *ebiten.Image

	SaveThumbFrameSelImg *ebiten.Image
	SaveThumbFrameImg    *ebiten.Image
	TotalPlayTime        float64
	MenuEntryThumb       *ebiten.Image
	SaveThumbs           [5]*ebiten.Image

	MenuBgImg *ebiten.Image

	MenuSkillPanelImg *ebiten.Image

	PartyIconImgs [4]*ebiten.Image

	ExamineIconImg *ebiten.Image

	MessageSpeed int

	LastMenuIndex int

	LastSkillCharIndex   int
	LastSkillSubIndex    int
	LastSkillLevelCursor int
	LastStatusCharIndex  int
	LastSlotIndex        int

	Fullscreen   bool
	WindowWidth  int
	WindowHeight int

	RememberCursor bool

	lastWindowW             int
	lastWindowH             int
	windowResizeSettleTimer int

	CurrentObjectiveID string

	MinimapPlayerIconImg    *ebiten.Image
	MinimapObjectiveIconImg *ebiten.Image

	ChestImg      *ebiten.Image
	KeyChestImg   *ebiten.Image
	LockedWallImg *ebiten.Image
	LeverWallImg  *ebiten.Image
	LeverImg      *ebiten.Image

	BlockImg     *ebiten.Image
	BlockSpotImg *ebiten.Image
	BlockDoorImg *ebiten.Image

	LightMaskImg *ebiten.Image

	fieldImageCache map[string]*ebiten.Image

	Audio                  *AudioManager
	mouseLastX, mouseLastY int
	mouseIdleTime          float64
	cursorHidden           bool

	IsDashing bool

	MobileMode bool

	Inventory     []InventorySlot
	OpenedChests  map[string]bool
	UnlockedWalls map[string]bool
	Keys          map[string]int
	RaisedLevers  map[string]bool

	BlockPositions     map[string][2]float64
	UnlockedBlockDoors map[string]bool

	SeenAutoHealMapIntro map[string]bool
}

const mouseIdleHideDelay = 2.0

const (
	FadeNone = iota
	FadeOut
	FadeIn
)

func (g *Game) FontFace(size float64) *text.GoTextFace {
	return &text.GoTextFace{Source: g.fontSource, Size: size}
}

func (g *Game) LoadFieldImage(filename string) *ebiten.Image {
	if g.fieldImageCache == nil {
		g.fieldImageCache = make(map[string]*ebiten.Image)
	}
	if img, cached := g.fieldImageCache[filename]; cached {
		return img
	}

	img, err := loadAssetImage("assets/images/field/" + filename)
	if err != nil {
		fmt.Printf("警告: 画像 \"assets/images/field/%s\" の読み込みに失敗しました: %v\n", filename, err)
		g.fieldImageCache[filename] = nil
		return nil
	}
	g.fieldImageCache[filename] = img
	return img
}

func generateLightMaskImage(size int) *ebiten.Image {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	cx, cy := float64(size)/2, float64(size)/2
	maxR := float64(size) / 2

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) + 0.5 - cx
			dy := float64(y) + 0.5 - cy
			t := math.Sqrt(dx*dx+dy*dy) / maxR

			var a float64
			if t < 1 {
				a = 1 - t*t*(3-2*t)
			}
			img.Set(x, y, color.RGBA{255, 255, 255, uint8(a * 255)})
		}
	}
	return ebiten.NewImageFromImage(img)
}

const blockTileSize = 32

func generateBlockImage(size int) *ebiten.Image {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	fill := color.RGBA{150, 105, 60, 255}
	border := color.RGBA{90, 60, 30, 255}
	const borderW = 3

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if x < borderW || y < borderW || x >= size-borderW || y >= size-borderW {
				img.Set(x, y, border)
			} else {
				img.Set(x, y, fill)
			}
		}
	}

	drawDiagonalCross(img, size, border)

	return ebiten.NewImageFromImage(img)
}

func generateBlockSpotImage(size int) *ebiten.Image {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	markColor := color.RGBA{255, 225, 90, 220}
	const thickness = 3

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if x < thickness || y < thickness || x >= size-thickness || y >= size-thickness {
				img.Set(x, y, markColor)
			}
		}
	}

	return ebiten.NewImageFromImage(img)
}

func generateBlockDoorImage(size int) *ebiten.Image {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	fill := color.RGBA{110, 118, 138, 255}
	border := color.RGBA{55, 60, 74, 255}
	const borderW = 3

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if x < borderW || y < borderW || x >= size-borderW || y >= size-borderW {
				img.Set(x, y, border)
			} else {
				img.Set(x, y, fill)
			}
		}
	}

	return ebiten.NewImageFromImage(img)
}

func drawDiagonalCross(img *image.RGBA, size int, c color.RGBA) {
	const thickness = 2
	for x := 0; x < size; x++ {
		for _, y := range []int{x, size - 1 - x} {
			for dy := -thickness; dy <= thickness; dy++ {
				yy := y + dy
				if yy >= 0 && yy < size {
					img.Set(x, yy, c)
				}
			}
		}
	}
}

func (g *Game) rememberedIndex(v int) int {
	if !g.RememberCursor {
		return 0
	}
	return v
}

func (g *Game) ResetForNewGame() {
	for i := 0; i < partySize; i++ {
		st := PlayerStatsByLevel[0][i]
		g.PlayerHP[i], g.PlayerMaxHP[i] = st.HP, st.HP
		g.PlayerMP[i], g.PlayerMaxMP[i] = st.MP, st.MP
		g.PlayerAtk[i] = st.PhysAtk
		g.PlayerMagicAtk[i] = st.MagicAtk
		g.PlayerDef[i] = st.PhysDef
		g.PlayerMagicDef[i] = st.MagicDef
		g.PlayerSpd[i] = st.Spd
		g.PlayerLuck[i] = st.Luck
		g.PlayerSP[i] = 0
		g.PlayerLv[i] = 1
		g.PlayerEXP[i] = 0
		g.PlayerNextEXP[i] = PlayerExpToNextByLevel[0]
	}

	for i := 0; i < partySize; i++ {
		for j := 0; j < len(g.PlayerSkillLv[i]); j++ {
			g.PlayerSkillLv[i][j] = 1
		}
	}

	g.BossDefeatedFlags = [4]bool{}
	g.TotalPlayTime = 0
	g.Inventory = nil
	g.OpenedChests = make(map[string]bool)
	g.UnlockedWalls = make(map[string]bool)
	g.Keys = make(map[string]int)
	g.RaisedLevers = make(map[string]bool)
	g.SeenAutoHealMapIntro = make(map[string]bool)
	g.BlockPositions = make(map[string][2]float64)
	g.UnlockedBlockDoors = make(map[string]bool)

	g.LastMenuIndex = 0
	g.LastSkillCharIndex = 0
	g.LastSkillSubIndex = 0
	g.LastSkillLevelCursor = 0
	g.LastStatusCharIndex = 0
	g.LastSlotIndex = 0

	g.UpdateObjective()
}

// assetPreloadPaths はNewGame内で読み込む画像アセットのパス一覧を返す。
// prefetchAssetBytesにまとめて渡し、Web版のfetchを並列化するために使う。
// 実際のロード処理・エラーメッセージはNewGame本体側の記述が正なので、
// ここでの列挙が万一漏れていても起動は壊れず、その分だけ先読みされずに
// 通常どおり逐次フェッチされるだけになる。
func assetPreloadPaths() []string {
	paths := []string{
		"assets/images/field/Tile_set_School_Set.png",
		"assets/images/field/Tile_set_School_Set (12).png",
		"assets/images/field/player_walk.png",
		"assets/images/battle/タイムライン横.png",
		"assets/images/battle/ゲージ.png",
		"assets/images/battle/ゴール.png",
		"assets/images/battle/タイムラインバー縦.png",
		"assets/images/battle/スキル拡張.png",
		"assets/images/battle/battle_bg.png",
		"assets/images/common/window.png",
		"assets/images/battle/name_normal.png",
		"assets/images/battle/name_myturn.png",
		"assets/images/battle/log_entry_box.png",
		"assets/images/field/調べる.png",
		"assets/images/field/現在地.png",
		"assets/images/field/目的地.png",
		"assets/images/field/chest.png",
		"assets/images/field/key_chest.png",
		"assets/images/field/locked_wall.png",
		"assets/images/field/lever_wall.png",
		"assets/images/field/lever.png",
		"assets/images/menu/メニュー画面.png",
		"assets/images/menu/メニュー画面拡張.png",
		"assets/images/menu/セーブスロット選択中.png",
		"assets/images/menu/セーブスロット.png",
		"assets/images/battle/アタック.png",
		"assets/images/battle/スキル.png",
		"assets/images/battle/待機.png",
		"assets/images/battle/逃げる.png",
	}

	for i := 0; i < 4; i++ {
		n := i + 1
		paths = append(paths,
			fmt.Sprintf("assets/images/field/bossスプライト_%d.png", n),
			fmt.Sprintf("assets/images/battle/boss_%d.png", n),
			fmt.Sprintf("assets/images/battle/boss_%d_icon.png", n),
			fmt.Sprintf("assets/images/battle/boss_%d_icon_large.png", n),
			fmt.Sprintf("assets/images/battle/player_attack_%d.png", n),
			fmt.Sprintf("assets/images/battle/timeline_p%d.png", n),
			fmt.Sprintf("assets/images/battle/timeline_p%d_large.png", n),
			fmt.Sprintf("assets/images/battle/battle_bg_boss_%d.png", n),
			fmt.Sprintf("assets/images/common/chara_boss%d.png", n),
			fmt.Sprintf("assets/images/menu/party_icon_%d.png", n),
		)
	}

	for _, enemy := range EnemyDatabase {
		paths = append(paths,
			fmt.Sprintf("assets/images/battle/enemy_%s.png", enemy.Name),
			fmt.Sprintf("assets/images/battle/enemy_%s_icon.png", enemy.Name),
			fmt.Sprintf("assets/images/battle/enemy_%s_icon_large.png", enemy.Name),
		)
	}

	for i := 0; i < partySize; i++ {
		paths = append(paths, fmt.Sprintf("assets/images/common/chara_player_%d.png", i+1))
	}

	return paths
}

func NewGame(source *text.GoTextFaceSource) (*Game, error) {
	g := &Game{
		fontSource:           source,
		Tilesets:             make(map[string]*ebiten.Image),
		CharaImgs:            make(map[string]*ebiten.Image),
		OpenedChests:         make(map[string]bool),
		UnlockedWalls:        make(map[string]bool),
		Keys:                 make(map[string]int),
		RaisedLevers:         make(map[string]bool),
		SeenAutoHealMapIntro: make(map[string]bool),
		BlockPositions:       make(map[string][2]float64),
		UnlockedBlockDoors:   make(map[string]bool),
	}

	g.ResetForNewGame()
	g.MobileMode = detectMobileMode()

	settings := LoadSettings()
	g.MessageSpeed = settings.MessageSpeed
	g.Fullscreen = settings.Fullscreen
	g.WindowWidth = settings.WindowWidth
	g.WindowHeight = settings.WindowHeight
	g.RememberCursor = settings.RememberCursor

	var err error

	prefetchAssetBytes(assetPreloadPaths())

	roukaImg, err := loadAssetImage("assets/images/field/Tile_set_School_Set.png")
	if err != nil {
		return nil, fmt.Errorf("街のタイルセット画像の読み込みに失敗 %w", err)
	}
	g.Tilesets["rouka"] = roukaImg

	dungeonImg, err := loadAssetImage("assets/images/field/Tile_set_School_Set (12).png")
	if err != nil {
		return nil, fmt.Errorf("ダンジョンのタイルセット画像の読み込みに失敗 %w", err)
	}
	g.Tilesets["dungeon"] = dungeonImg

	defaultImg, err := loadAssetImage("assets/images/field/Tile_set_School_Set.png")
	if err != nil {
		return nil, fmt.Errorf("デフォルトタイルセット画像の読み込みに失敗 %w", err)
	}
	g.Tilesets["default"] = defaultImg

	g.TileImg = g.Tilesets["default"]

	if err := BuildObjectiveAndMapIndex(); err != nil {
		return nil, fmt.Errorf("目的地インデックスの構築に失敗 %w", err)
	}
	g.UpdateObjective()

	g.SpriteSheet, err = loadAssetImage("assets/images/field/player_walk.png")
	if err != nil {
		return nil, fmt.Errorf("プレイヤースプライトの読み込みに失敗 %w", err)
	}
	for i := 0; i < 4; i++ {
		path := fmt.Sprintf("assets/images/field/bossスプライト_%d.png", i+1)
		g.BossSpriteSheets[i], err = loadAssetImage(path)
		if err != nil {
			return nil, fmt.Errorf("ボス%dの歩行スプライト読み込みに失敗 %w", i+1, err)
		}
	}

	for i := 0; i < 4; i++ {
		path := fmt.Sprintf("assets/images/battle/boss_%d.png", i+1)
		g.BossImgs[i], err = loadAssetImage(path)
		if err != nil {
			return nil, fmt.Errorf("ボス%dの画像読み込みに失敗 %w", i+1, err)
		}
	}

	for i := 0; i < 4; i++ {
		path := fmt.Sprintf("assets/images/battle/boss_%d_icon.png", i+1)
		img, ferr := loadAssetImage(path)
		if ferr != nil {
			return nil, fmt.Errorf("ボス%dのアイコン画像読み込みに失敗 %w", i+1, ferr)
		}
		g.BossIconImgs[i] = img
	}

	for i := 0; i < 4; i++ {
		path := fmt.Sprintf("assets/images/battle/boss_%d_icon_large.png", i+1)
		img, ferr := loadAssetImage(path)
		if ferr != nil {
			return nil, fmt.Errorf("ボス%dの拡大アイコン画像読み込みに失敗 %w", i+1, ferr)
		}
		g.BossIconLargeImgs[i] = img
	}

	g.EnemyImgs = make(map[string]*ebiten.Image)
	g.EnemyIconImgs = make(map[string]*ebiten.Image)
	g.EnemyIconLargeImgs = make(map[string]*ebiten.Image)

	for _, enemy := range EnemyDatabase {
		battlePath := fmt.Sprintf("assets/images/battle/enemy_%s.png", enemy.Name)
		img, ferr := loadAssetImage(battlePath)
		if ferr != nil {
			return nil, fmt.Errorf("%sの画像読み込みに失敗 %w", enemy.Name, ferr)
		}
		g.EnemyImgs[enemy.Name] = img

		iconPath := fmt.Sprintf("assets/images/battle/enemy_%s_icon.png", enemy.Name)
		iconImg, ferr := loadAssetImage(iconPath)
		if ferr != nil {
			return nil, fmt.Errorf("%sのアイコン画像読み込みに失敗 %w", enemy.Name, ferr)
		}
		g.EnemyIconImgs[enemy.Name] = iconImg

		iconLargePath := fmt.Sprintf("assets/images/battle/enemy_%s_icon_large.png", enemy.Name)
		iconLargeImg, ferr := loadAssetImage(iconLargePath)
		if ferr != nil {
			return nil, fmt.Errorf("%sの拡大アイコン画像読み込みに失敗 %w", enemy.Name, ferr)
		}
		g.EnemyIconLargeImgs[enemy.Name] = iconLargeImg
	}

	for i := 0; i < 4; i++ {
		path := fmt.Sprintf("assets/images/battle/player_attack_%d.png", i+1)
		g.PlayerAttackSprites[i], err = loadAssetImage(path)
		if err != nil {
			return nil, fmt.Errorf("プレイヤー%dの攻撃・アクション画像ロード失敗: %w", i+1, err)
		}
	}

	commandIconFiles := [4]string{
		"assets/images/battle/アタック.png",
		"assets/images/battle/スキル.png",
		"assets/images/battle/待機.png",
		"assets/images/battle/逃げる.png",
	}
	for i, path := range commandIconFiles {
		img, ferr := loadAssetImage(path)
		if ferr != nil {
			return nil, fmt.Errorf("コマンドアイコン画像読み込みに失敗 %w", ferr)
		}
		g.CommandIcons[i] = img
	}

	timelineIconFiles := [4]string{
		"assets/images/battle/timeline_p1.png",
		"assets/images/battle/timeline_p2.png",
		"assets/images/battle/timeline_p3.png",
		"assets/images/battle/timeline_p4.png",
	}
	for i, path := range timelineIconFiles {
		img, ferr := loadAssetImage(path)
		if ferr != nil {
			return nil, fmt.Errorf("タイムラインアイコン画像読み込みに失敗 %w", ferr)
		}
		g.TimelineIcons[i] = img
	}

	timelineIconLargeFiles := [4]string{
		"assets/images/battle/timeline_p1_large.png",
		"assets/images/battle/timeline_p2_large.png",
		"assets/images/battle/timeline_p3_large.png",
		"assets/images/battle/timeline_p4_large.png",
	}
	for i, path := range timelineIconLargeFiles {
		img, ferr := loadAssetImage(path)
		if ferr != nil {
			return nil, fmt.Errorf("タイムライン拡大アイコン画像読み込みに失敗 %w", ferr)
		}
		g.TimelineIconsLarge[i] = img
	}

	g.TimelineBarImg, err = loadAssetImage("assets/images/battle/タイムライン横.png")
	if err != nil {
		return nil, fmt.Errorf("タイムラインバー画像の読み込みに失敗 %w", err)
	}

	g.GaugeImg, err = loadAssetImage("assets/images/battle/ゲージ.png")
	if err != nil {
		return nil, fmt.Errorf("ゲージ画像の読み込みに失敗 %w", err)
	}

	g.GoalImg, err = loadAssetImage("assets/images/battle/ゴール.png")
	if err != nil {
		return nil, fmt.Errorf("ゴール画像の読み込みに失敗 %w", err)
	}

	g.TimelineBarVertImg, err = loadAssetImage("assets/images/battle/タイムラインバー縦.png")
	if err != nil {
		return nil, fmt.Errorf("タイムラインバー縦画像の読み込みに失敗 %w", err)
	}

	g.SkillPanelImg, err = loadAssetImage("assets/images/battle/スキル拡張.png")
	if err != nil {
		return nil, fmt.Errorf("スキル拡張画像の読み込みに失敗 %w", err)
	}

	g.BattleBgImg, err = loadAssetImage("assets/images/battle/battle_bg.png")
	if err != nil {
		return nil, fmt.Errorf("通常戦闘の背景画像読み込みに失敗 %w", err)
	}

	for i := 0; i < 4; i++ {
		path := fmt.Sprintf("assets/images/battle/battle_bg_boss_%d.png", i+1)
		img, ferr := loadAssetImage(path)
		if ferr != nil {
			return nil, fmt.Errorf("ボス%dの戦闘背景画像読み込みに失敗 %w", i+1, ferr)
		}
		g.BossBgImgs[i] = img
	}

	g.WindowImg, err = loadAssetImage("assets/images/common/window.png")
	if err != nil {
		return nil, fmt.Errorf("ウィンドウ画像の読み込みに失敗 %w", err)
	}

	g.NameImg, err = loadAssetImage("assets/images/battle/name_normal.png")
	if err != nil {
		return nil, fmt.Errorf("名前画像(通常)の読み込みに失敗 %w", err)
	}

	g.NameMyTurnImg, err = loadAssetImage("assets/images/battle/name_myturn.png")
	if err != nil {
		return nil, fmt.Errorf("名前画像(自分のターン)の読み込みに失敗 %w", err)
	}

	g.LogEntryImg, err = loadAssetImage("assets/images/battle/log_entry_box.png")
	if err != nil {
		return nil, fmt.Errorf("ログウィンドウ画像の読み込みに失敗 %w", err)
	}

	g.ExamineIconImg, err = loadAssetImage("assets/images/field/調べる.png")
	if err != nil {
		return nil, fmt.Errorf("調べるアイコン画像の読み込みに失敗 %w", err)
	}

	g.MinimapPlayerIconImg, err = loadAssetImage("assets/images/field/現在地.png")
	if err != nil {
		return nil, fmt.Errorf("現在地アイコン画像の読み込みに失敗 %w", err)
	}

	g.MinimapObjectiveIconImg, err = loadAssetImage("assets/images/field/目的地.png")
	if err != nil {
		return nil, fmt.Errorf("目的地アイコン画像の読み込みに失敗 %w", err)
	}

	g.ChestImg, err = loadAssetImage("assets/images/field/chest.png")
	if err != nil {
		return nil, fmt.Errorf("チェスト画像の読み込みに失敗 %w", err)
	}

	g.KeyChestImg, err = loadAssetImage("assets/images/field/key_chest.png")
	if err != nil {
		return nil, fmt.Errorf("鍵チェスト画像の読み込みに失敗 %w", err)
	}

	g.LockedWallImg, err = loadAssetImage("assets/images/field/locked_wall.png")
	if err != nil {
		return nil, fmt.Errorf("封印された壁画像の読み込みに失敗 %w", err)
	}

	g.LeverWallImg, err = loadAssetImage("assets/images/field/lever_wall.png")
	if err != nil {
		return nil, fmt.Errorf("レバーで開く壁画像の読み込みに失敗 %w", err)
	}

	g.LeverImg, err = loadAssetImage("assets/images/field/lever.png")
	if err != nil {
		return nil, fmt.Errorf("レバー画像の読み込みに失敗 %w", err)
	}

	g.LightMaskImg = generateLightMaskImage(256)

	g.BlockImg = generateBlockImage(blockTileSize)
	g.BlockSpotImg = generateBlockSpotImage(blockTileSize)
	g.BlockDoorImg = generateBlockDoorImage(blockTileSize)

	g.Audio = NewAudioManager()
	g.Audio.SetVolume(settings.BGMVolume)

	g.currentScene = NewTitleScene(g)

	for i := 0; i < 4; i++ {
		bossNum := i + 1
		path := fmt.Sprintf("assets/images/common/chara_boss%d.png", bossNum)
		img, ferr := loadAssetImage(path)
		if ferr != nil {
			return nil, fmt.Errorf("ボス%dの立ち絵読み込みに失敗 %w", bossNum, ferr)
		}
		g.CharaImgs[BossNames[i]] = img
	}

	for i := 0; i < partySize; i++ {
		path := fmt.Sprintf("assets/images/common/chara_player_%d.png", i+1)
		img, ferr := loadAssetImage(path)
		if ferr != nil {
			return nil, fmt.Errorf("プレイヤー%dの立ち絵読み込みに失敗 %w", i+1, ferr)
		}
		g.CharaImgs[PlayerNames[i]] = img
	}

	g.MenuBgImg, err = loadAssetImage("assets/images/menu/メニュー画面.png")
	if err != nil {
		return nil, fmt.Errorf("メニュー背景画像の読み込みに失敗 %w", err)
	}

	g.MenuSkillPanelImg, err = loadAssetImage("assets/images/menu/メニュー画面拡張.png")
	if err != nil {
		return nil, fmt.Errorf("メニュースキルパネル画像の読み込みに失敗 %w", err)
	}

	partyIconFiles := [4]string{
		"assets/images/menu/party_icon_1.png",
		"assets/images/menu/party_icon_2.png",
		"assets/images/menu/party_icon_3.png",
		"assets/images/menu/party_icon_4.png",
	}
	for i, path := range partyIconFiles {
		img, ferr := loadAssetImage(path)
		if ferr != nil {
			return nil, fmt.Errorf("パーティアイコン画像の読み込みに失敗 %w", ferr)
		}
		g.PartyIconImgs[i] = img
	}

	g.SaveThumbFrameSelImg, err = loadAssetImage("assets/images/menu/セーブスロット選択中.png")
	if err != nil {
		return nil, fmt.Errorf("セーブ枠(選択中)画像の読み込みに失敗 %w", err)
	}
	g.SaveThumbFrameImg, err = loadAssetImage("assets/images/menu/セーブスロット.png")
	if err != nil {
		return nil, fmt.Errorf("セーブ枠画像の読み込みに失敗 %w", err)
	}

	applyDisplayMode(g.Fullscreen, g.WindowWidth, g.WindowHeight)
	if !g.Fullscreen {
		g.WindowWidth, g.WindowHeight = ebiten.WindowSize()
	}
	g.lastWindowW, g.lastWindowH = g.WindowWidth, g.WindowHeight

	if err := LoadDialogues("assets/dialogues"); err != nil {
		return nil, fmt.Errorf("セリフデータの読み込みに失敗 %w", err)
	}
	validateDialogueSlots()

	return g, nil
}

func (g *Game) Update() error {
	dt := 1.0 / 60.0

	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		g.MobileMode = !g.MobileMode
	}
	_, inFieldScene := g.currentScene.(*FieldScene)
	fieldMobileControlsEnabled = g.MobileMode && inFieldScene
	uiMobileArrowsEnabled = g.MobileMode

	g.updateMouseCursorVisibility(dt)
	g.updateWindowSizeTracking()

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

func (g *Game) GetEnemyBattleImage(evType, name string) *ebiten.Image {
	if idx, ok := bossIndexFromEnemyType(evType); ok {
		return g.BossImgs[idx]
	}
	return g.EnemyImgs[name]
}

func (g *Game) GetEnemyTimelineIcon(evType, name string) *ebiten.Image {
	if idx, ok := bossIndexFromEnemyType(evType); ok {
		return g.BossIconImgs[idx]
	}
	return g.EnemyIconImgs[name]
}

func (g *Game) GetEnemyTimelineIconLarge(evType, name string) *ebiten.Image {
	if idx, ok := bossIndexFromEnemyType(evType); ok {
		return g.BossIconLargeImgs[idx]
	}
	return g.EnemyIconLargeImgs[name]
}

func (g *Game) captureMenuEntryThumb(full *ebiten.Image) {
	thumb := ebiten.NewImage(196, 110)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(196.0/float64(gameWidth), 110.0/float64(gameHeight))
	thumb.DrawImage(full, op)
	g.MenuEntryThumb = thumb
}

func (g *Game) saveThumbToFile(slot int) {
	if g.MenuEntryThumb == nil {
		return
	}
	path := fmt.Sprintf("save_thumb_%d.png", slot)
	var buf bytes.Buffer
	if err := png.Encode(&buf, g.MenuEntryThumb); err != nil {
		fmt.Printf("警告: サムネイルのエンコードに失敗しました: %v\n", err)
		return
	}
	if err := writeRuntimeFile(path, buf.Bytes()); err != nil {
		fmt.Printf("警告: サムネイルの保存に失敗しました: %v\n", err)
	}
}

func FormatPlayTime(seconds float64) string {
	m := int(seconds) / 60
	h := m / 60
	if h > 999 {
		h = 999
		m = 999*60 + 59
	}
	return fmt.Sprintf("%03d:%02d", h, m%60)
}

func (g *Game) MessageSpeedTicks() int {
	switch g.MessageSpeed {
	case 0:
		return 9
	case 2:
		return 2
	default:
		return 5
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
