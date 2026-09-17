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

	heavyDecoded     chan decodedHeavyAsset
	heavyAssetsReady bool
	heavyAssetsErr   error
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

// latinFontScale compensates for PixelMplus drawing Latin letters/digits
// visually smaller than Japanese glyphs at the same Size, so a bigger value
// here makes English/numbers render bigger relative to Japanese text.
const latinFontScale = 1.2

// LatinFontFace is for text.Draw calls whose whole string is Latin
// letters/digits (no Japanese), so it reads at the same visual size as
// Japanese text drawn with FontFace(size).
func (g *Game) LatinFontFace(size float64) *text.GoTextFace {
	return &text.GoTextFace{Source: g.fontSource, Size: size * latinFontScale}
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

	g.EnemyImgs = make(map[string]*ebiten.Image)
	g.EnemyIconImgs = make(map[string]*ebiten.Image)
	g.EnemyIconLargeImgs = make(map[string]*ebiten.Image)

	g.LightMaskImg = generateLightMaskImage(256)

	g.BlockImg = generateBlockImage(blockTileSize)
	g.BlockSpotImg = generateBlockSpotImage(blockTileSize)
	g.BlockDoorImg = generateBlockDoorImage(blockTileSize)

	g.Audio = NewAudioManager()
	g.Audio.SetVolume(settings.BGMVolume)
	g.Audio.SetSEVolume(settings.SEVolume)
	g.Audio.SetMasterVolume(settings.MasterVolume)

	g.currentScene = NewTitleScene(g)
	g.Audio.PlayBGMFadeIn(bgmTitle, 2.0)

	applyDisplayMode(g.Fullscreen, g.WindowWidth, g.WindowHeight)
	if !g.Fullscreen {
		g.WindowWidth, g.WindowHeight = ebiten.WindowSize()
	}
	g.lastWindowW, g.lastWindowH = g.WindowWidth, g.WindowHeight

	// ボス・敵・メニューなどタイトル画面自体には不要な画像とデータは、
	// タイトル画面を即座に表示できるようバックグラウンドで読み込む。
	// 完了まではTitleScene側でheavyAssetsReadyを見て先の画面に進ませない。
	g.heavyDecoded = make(chan decodedHeavyAsset, 32)
	go g.loadHeavyAssetsAsync()

	return g, nil
}

func (g *Game) Update() error {
	dt := 1.0 / 60.0

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
