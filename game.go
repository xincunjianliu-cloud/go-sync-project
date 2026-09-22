package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
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

	PlayerHP             [4]int
	PlayerMaxHP          [4]int
	PlayerMP             [4]int
	PlayerMaxMP          [4]int
	PlayerAtk            [4]int
	PlayerMagicAtk       [4]int
	PlayerDef            [4]int
	PlayerMagicDef       [4]int
	PlayerSpd            [4]int
	PlayerLuck           [4]int
	PlayerSP             [4]int
	PlayerSkillLv        [4][8]int
	PlayerLv             [4]int
	PlayerEXP            [4]int
	PlayerNextEXP        [4]int
	PlayerAttackSprites  [4]*ebiten.Image
	CommandIcons         [4]*ebiten.Image
	CommandIconsSelected [4]*ebiten.Image
	TimelineIcons        [4]*ebiten.Image
	TimelineIconsLarge   [4]*ebiten.Image
	SkillPanelImg        *ebiten.Image

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
	NameDeadImg   *ebiten.Image

	// StatIconUpImgs/StatIconDownImgs are the battle HUD's per-stat buff/debuff
	// icons, indexed by StatKind (StatAtk, StatMat, StatDef, StatMdf, StatLuk).
	StatIconUpImgs   [5]*ebiten.Image
	StatIconDownImgs [5]*ebiten.Image

	TimelineBarImg     *ebiten.Image
	TimelineBarVertImg *ebiten.Image

	WindowImg *ebiten.Image
	// CharaImgs は立ち絵画像のキャッシュ。基本形は表示名そのものをキーに
	// 持ち(プレイヤー・ボスは起動時に先読み済み)、表情差分は
	// "表示名\x00表情キー" の形でキーを作り、初めて必要になった時点で
	// GetCharaImage が遅延読み込みしてキャッシュする。
	CharaImgs map[string]*ebiten.Image
	// charaSlugs は表示名から立ち絵ファイル名の接頭辞への対応表
	// (例: ボス1の表示名 -> "boss1")。無い場合は表示名をそのまま
	// ファイル名として使う(NPCの立ち絵をassets/images/common/に
	// 置くだけで使えるようにするため)。
	charaSlugs map[string]string
	// charaImgMissing は存在しない立ち絵パスを記録し、毎フレーム同じ
	// 読み込み失敗を繰り返さないようにするためのキャッシュ。
	charaImgMissing map[string]bool

	// bgImgs はBossDialogue.Backgroundで指定された全画面背景画像の
	// キャッシュ(assets/images/backgrounds/<キー>.png)。GetBackgroundImage
	// が初めて必要になった時点で遅延読み込みする。
	bgImgs       map[string]*ebiten.Image
	bgImgMissing map[string]bool

	LogEntryImg *ebiten.Image

	SaveThumbFrameSelImg *ebiten.Image
	SaveThumbFrameImg    *ebiten.Image
	TotalPlayTime        float64
	MenuEntryThumb       *ebiten.Image
	SaveThumbs           [5]*ebiten.Image

	MenuBgImg *ebiten.Image

	TitleBgImg *ebiten.Image

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

	ChestImg             *ebiten.Image
	KeyChestImg          *ebiten.Image
	LockedWallImg        *ebiten.Image
	LeverWallOpenImg     *ebiten.Image
	LeverWallOpenDecoImg *ebiten.Image
	LeverImg             *ebiten.Image

	// LeverWallOpenImgs/LeverWallOpenDecoImgsは、壁オブジェクトの"img"
	// プロパティで場所ごとに絵を差し替えたい場合の追加分。キーはプロパティ値と
	// 同じ名前。該当キーが無ければLeverWallOpenImg/LeverWallOpenDecoImgに
	// フォールバックする(field_draw.goのdrawLeverWalls参照)。
	// 新しい見た目を足す手順:
	//   1. assets/images/field/lever_wall_open_<name>.png
	//      (見た目だけの壁ならlever_wall_open_deco_<name>.png)を追加
	//   2. assets_deferred.goのdeferredAssetAssignments()に1行登録
	//   3. Tiledの壁オブジェクトにimg=<name>プロパティを設定
	LeverWallOpenImgs     map[string]*ebiten.Image
	LeverWallOpenDecoImgs map[string]*ebiten.Image

	BlockImg     *ebiten.Image
	BlockSpotImg *ebiten.Image
	BlockDoorImg *ebiten.Image

	LightMaskImg *ebiten.Image

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

	SeenAutoHealMapIntro     map[string]bool
	SeenEvents               map[string]bool
	SeenBattleTutorial       bool
	SeenGaugeTutorial        bool
	SeenSkillUpgradeTutorial bool

	heavyDecoded     chan decodedHeavyAsset
	heavyAssetsReady bool
	heavyAssetsErr   error

	bootDecoded     chan bootResult
	bootReady       bool
	loadingAnimTime float64
}

// bootResult はloadBootAssetsAsyncが読み込む、起動直後に最低限必要な
// アセット(フォント・タイトル背景・タイトルBGM)の結果をUpdate()側へ渡す。
type bootResult struct {
	fontSource *text.GoTextFaceSource
	titleBg    image.Image
}

const mouseIdleHideDelay = 2.0

const (
	FadeNone = iota
	FadeOut
	FadeIn
)

func (g *Game) FontFace(size float64) text.Face {
	if f, ok := multiFaceCache[size]; ok {
		return f
	}
	base := &text.GoTextFace{Source: g.fontSource, Size: size}
	mf, err := text.NewMultiFace(base, triangleFallbackFace(size))
	if err != nil {
		panic(err)
	}
	multiFaceCache[size] = mf
	return mf
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
	g.SeenEvents = make(map[string]bool)
	g.SeenBattleTutorial = false
	g.SeenGaugeTutorial = false
	g.SeenSkillUpgradeTutorial = false
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

func NewGame() *Game {
	g := &Game{
		Tilesets:             make(map[string]*ebiten.Image),
		CharaImgs:            make(map[string]*ebiten.Image),
		charaSlugs:           buildCharaSlugs(),
		charaImgMissing:      make(map[string]bool),
		bgImgs:               make(map[string]*ebiten.Image),
		bgImgMissing:         make(map[string]bool),
		OpenedChests:         make(map[string]bool),
		UnlockedWalls:        make(map[string]bool),
		Keys:                 make(map[string]int),
		RaisedLevers:         make(map[string]bool),
		SeenAutoHealMapIntro: make(map[string]bool),
		SeenEvents:           make(map[string]bool),
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

	g.EnemyImgs = make(map[string]*ebiten.Image)
	g.EnemyIconImgs = make(map[string]*ebiten.Image)
	g.EnemyIconLargeImgs = make(map[string]*ebiten.Image)
	g.LeverWallOpenImgs = make(map[string]*ebiten.Image)
	g.LeverWallOpenDecoImgs = make(map[string]*ebiten.Image)

	g.LightMaskImg = generateLightMaskImage(256)

	g.BlockImg = generateBlockImage(blockTileSize)
	g.BlockSpotImg = generateBlockSpotImage(blockTileSize)
	g.BlockDoorImg = generateBlockDoorImage(blockTileSize)

	g.Audio = NewAudioManager()
	g.Audio.SetVolume(settings.BGMVolume)
	g.Audio.SetSEVolume(settings.SEVolume)
	g.Audio.SetMasterVolume(settings.MasterVolume)

	applyDisplayMode(g.Fullscreen, g.WindowWidth, g.WindowHeight)
	if !g.Fullscreen {
		g.WindowWidth, g.WindowHeight = ebiten.WindowSize()
	}
	g.lastWindowW, g.lastWindowH = g.WindowWidth, g.WindowHeight

	// フォント・タイトル背景・タイトルBGMは起動直後の表示に最低限必要な
	// アセットだが、Web版はすべてネットワーク取得になる。順番に待つと
	// フェッチ回数分の往復時間が積み重なって初回表示が遅くなるため、
	// 3つとも並列にバックグラウンドで取得する。完了までのUpdate/Drawは
	// pumpBootAssets/drawLoadingIndicatorがローディング表示だけを進める。
	g.bootDecoded = make(chan bootResult, 1)
	go g.loadBootAssetsAsync()

	return g
}

// loadBootAssetsAsync は起動直後に必要なフォント・タイトル背景・タイトルBGMを
// 並列に先読みし、フォントソースの生成(CPUのみで完結しGPUテクスチャを
// 作らないためメインゴルーチン以外でも安全)まで済ませてbootDecodedへ送る。
// タイトルBGMはここではまだ再生しない(AudioManager.PlayBGMFadeInの再生開始は
// pumpBootAssets側、メインゴルーチンで行う)が、バイト列だけprefetchAssetBytes
// 経由でキャッシュしておくことで、再生時にネットワーク待ちが発生しなくなる。
func (g *Game) loadBootAssetsAsync() {
	defer close(g.bootDecoded)

	prefetchAssetBytes([]string{
		"assets/fonts/k8x12.ttf",
		"assets/images/title/title_bg.png",
		bgmTitle,
	})

	fontData, err := loadAssetBytesCached("assets/fonts/k8x12.ttf")
	if err != nil {
		log.Fatal("フォントファイルの読み込みに失敗しました: ", err)
	}
	source, err := text.NewGoTextFaceSource(bytes.NewReader(fontData))
	if err != nil {
		log.Fatal("フォントソースの生成に失敗しました: ", err)
	}

	titleBg, _ := decodeAssetImage("assets/images/title/title_bg.png")

	g.bootDecoded <- bootResult{fontSource: source, titleBg: titleBg}
}

// pumpBootAssets はUpdate()から毎フレーム呼ばれ、起動直後のフォント・
// タイトル背景の読み込み完了を待つ。完了したらタイトル画面を組み立てて
// BGMを再生し、続けて戦闘・メニュー用画像のバックグラウンド読み込みを
// 開始する。
func (g *Game) pumpBootAssets() {
	if g.bootReady || g.bootDecoded == nil {
		return
	}
	select {
	case item, ok := <-g.bootDecoded:
		if !ok {
			return
		}
		g.fontSource = item.fontSource
		if item.titleBg != nil {
			g.TitleBgImg = ebiten.NewImageFromImage(item.titleBg)
		}

		g.currentScene = NewTitleScene(g)
		g.Audio.PlayBGMFadeIn(bgmTitle, 2.0)

		// ボス・敵・メニューなどタイトル画面自体には不要な画像とデータは、
		// タイトル画面を即座に表示できるようバックグラウンドで読み込む。
		// 完了まではTitleScene側でheavyAssetsReadyを見て先の画面に進ませない。
		g.heavyDecoded = make(chan decodedHeavyAsset, 32)
		go g.loadHeavyAssetsAsync()

		g.bootReady = true
	default:
	}
}

func (g *Game) Update() error {
	dt := 1.0 / 60.0

	g.loadingAnimTime += dt
	g.pumpBootAssets()
	if !g.bootReady {
		return nil
	}

	g.pumpHeavyAssets()

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
			g.ChangeSceneWithFade(nextScene, fadeTimeBattleOut)
		}
	}
	return nil
}

// bgmScene は遷移先のシーンが希望するBGMを表す。ChangeSceneWithFade はこれを
// 実装しているシーンに対してのみBGM切り替えを行う。切り替え自体は画面が
// 暗転しきった瞬間に発生するよう、フェードアウトを画面フェードと同じ
// durationSeconds で開始する(Game.Update内で両者は毎フレーム同じdtで
// 進むため、暗転完了と曲の無音化がほぼ同時に起きる)。
type bgmScene interface {
	desiredBGM(transitionDuration float64) (path string, fadeInDuration float64, hardCut bool)
}

func (g *Game) ChangeSceneWithFade(nextScene Scene, durationSeconds float64) {
	g.pendingScene = nextScene
	g.fadeMode = FadeOut
	g.fadeAlpha = 0.0
	g.fadeSpeed = 1.0 / durationSeconds
	if bs, ok := nextScene.(bgmScene); ok {
		path, fadeIn, hardCut := bs.desiredBGM(durationSeconds)
		g.Audio.FadeOutThenPlay(path, durationSeconds, fadeIn, hardCut)
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	if !g.bootReady {
		screen.Fill(color.RGBA{10, 10, 30, 255})
		drawLoadingIndicator(screen, g.loadingAnimTime)
		return
	}

	if g.currentScene != nil {
		g.currentScene.Draw(screen)
	}

	if g.fadeAlpha > 0 {
		ebitenutil.DrawRect(screen, 0, 0, float64(gameWidth), float64(gameHeight), color.NRGBA{0, 0, 0, uint8(255 * g.fadeAlpha)})
	}

	if !g.heavyAssetsReady {
		drawLoadingIndicator(screen, g.loadingAnimTime)
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
		return
	}
	writeRuntimeFile(path, buf.Bytes())
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

// buildCharaSlugs はボス・プレイヤーの表示名から立ち絵ファイル名の
// 接頭辞への対応表を作る(例: ボス1の表示名 -> "boss1")。NPCなど
// この表に無い名前はGetCharaImageが表示名そのものをファイル名として使う。
func buildCharaSlugs() map[string]string {
	m := make(map[string]string, len(BossNames)+partySize)
	for i, name := range BossNames {
		m[name] = fmt.Sprintf("boss%d", i+1)
	}
	for i := 0; i < partySize; i++ {
		m[PlayerNames[i]] = fmt.Sprintf("player_%d", i+1)
	}
	return m
}

// charaExprSheetFrames はchara_<キャラ>_sheet.pngが横一列に並べて持つ
// 表情差分のコマ数。全キャラ共通で 0=通常 1=笑顔 2=怒り 3=驚き の固定割り当て。
const charaExprSheetFrames = 4

// GetCharaImage は話者の表示名と表情番号(0=通常)から立ち絵画像を返す。
// そのキャラのchara_<キャラ>_sheet.pngが用意されていれば表情差分シートと
// みなして横charaExprSheetFrames等分し、expression番目のコマを切り出す。
// シートが無ければ従来どおり単一画像のchara_<キャラ>.pngを使い、
// (まだ表情差分を作っていないキャラの場合)expressionは無視される。
// どちらの読み込み結果もキャッシュするので、シーン中は毎フレーム
// ディスク(埋め込みアセット)を読みにいかない。
func (g *Game) GetCharaImage(speaker string, expression int) *ebiten.Image {
	if speaker == "" {
		return nil
	}
	slug, ok := g.charaSlugs[speaker]
	if !ok {
		slug = speaker
	}

	sheetPath := fmt.Sprintf("assets/images/common/chara_%s_sheet.png", slug)
	if sheet := g.lookupCharaAsset(speaker+"#sheet", sheetPath); sheet != nil {
		idx := expression
		if idx < 0 || idx >= charaExprSheetFrames {
			idx = 0
		}
		frameW := sheet.Bounds().Dx() / charaExprSheetFrames
		rect := image.Rect(idx*frameW, 0, (idx+1)*frameW, sheet.Bounds().Dy())
		return sheet.SubImage(rect).(*ebiten.Image)
	}

	basePath := fmt.Sprintf("assets/images/common/chara_%s.png", slug)
	return g.lookupCharaAsset(speaker, basePath)
}

func (g *Game) lookupCharaAsset(key, path string) *ebiten.Image {
	if img, ok := g.CharaImgs[key]; ok {
		return img
	}
	if g.charaImgMissing[key] {
		return nil
	}

	img, err := loadAssetImage(path)
	if err != nil {
		g.charaImgMissing[key] = true
		return nil
	}
	g.CharaImgs[key] = img
	return img
}

// GetBackgroundImage はBossDialogue.Backgroundで指定されたキーから
// assets/images/backgrounds/<キー>.pngを遅延読み込みして返す。無ければ
// nilを返す(呼び出し側はマップ描画やベタ塗りにフォールバックする)。
func (g *Game) GetBackgroundImage(key string) *ebiten.Image {
	if key == "" {
		return nil
	}
	if img, ok := g.bgImgs[key]; ok {
		return img
	}
	if g.bgImgMissing[key] {
		return nil
	}

	path := fmt.Sprintf("assets/images/backgrounds/%s.png", key)
	img, err := loadAssetImage(path)
	if err != nil {
		g.bgImgMissing[key] = true
		return nil
	}
	g.bgImgs[key] = img
	return img
}
