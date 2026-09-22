package main

import (
	"encoding/json"
	"fmt"
	"math"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	fadeTimeDoor      = 0.3
	fadeTimeBattleIn  = 0.4
	fadeTimeBossIn    = 1.2
	fadeTimeBattleOut = 0.4
	fadeTimeBossOut   = 1.5
)

// 雑魚敵エンカウント時、戦闘シーンへ切り替わる直前にフィールド画面を
// 静止画として捉え、カメラが回転しながら敵へ迫っていくような演出を行う。
const (
	encounterEffectDuration = 0.5
	encounterEffectZoomEnd  = 2.4
	encounterEffectSpin     = math.Pi / 5.0
)

type TiledMap struct {
	Width      int             `json:"width"`
	Height     int             `json:"height"`
	TileWidth  int             `json:"tilewidth"`
	TileHeight int             `json:"tileheight"`
	Layers     []TiledLayer    `json:"layers"`
	Properties []TiledProperty `json:"properties"`
	Tilesets   []TiledTileset  `json:"tilesets"`
}

// TiledTileset はTiledが.tmjに埋め込むタイルセット定義のうち、画像パスの
// 解決に使うフィールドだけを取り出したもの。複数タイルセットの合成には
// 対応しておらず、常に先頭の1件(Tilesets[0])だけを使用する。
type TiledTileset struct {
	Image string `json:"image"`
}

// mapBGMKey はマップ全体のカスタムプロパティ"bgm"の値を返す。
// (Tiledのマッププロパティで、このマップを歩いているときに流す曲を
// bgmByKeyのキー名で指定する。例: "field2")
func (m TiledMap) mapBGMKey() (string, bool) {
	for _, p := range m.Properties {
		if strings.EqualFold(p.Name, "bgm") {
			if s, ok := p.Value.(string); ok && s != "" {
				return s, true
			}
		}
	}
	return "", false
}

// mapDisplayName はマップ全体のカスタムプロパティ"displayname"の値を返す。
// (セーブスロットやミニマップに表示する地名。省略時はマップファイルの
// パスがそのまま表示名として使われる)
func (m TiledMap) mapDisplayName() (string, bool) {
	for _, p := range m.Properties {
		if strings.EqualFold(p.Name, "displayname") {
			if s, ok := p.Value.(string); ok && s != "" {
				return s, true
			}
		}
	}
	return "", false
}

// mapAutoHeal はマップ全体のカスタムプロパティ"autoheal"(bool)の値を返す。
// trueにすると、このマップに入った瞬間にパーティが全回復する
// (ダンジョンの入り口などに使う想定)。省略時はfalse。
func (m TiledMap) mapAutoHeal() bool {
	for _, p := range m.Properties {
		if strings.EqualFold(p.Name, "autoheal") {
			if b, ok := p.Value.(bool); ok {
				return b
			}
		}
	}
	return false
}

var tilesetImageCache = map[string]*ebiten.Image{}

// resolveTilesetImagePath はTiledが.tmjに書き出す、マップファイルからの
// 相対パス(例: "../images/field/Foo.png")を、assets/maps/を基準にした
// リポジトリ内の実パスに変換する。
func resolveTilesetImagePath(image string) string {
	image = strings.ReplaceAll(image, "\\", "/")
	return path.Clean(path.Join("assets/maps", image))
}

// mapTilesetImage は.tmj自身が指すタイルセット画像(Tilesets[0].Image)を
// 読み込む。新しいマップを追加したり、既存マップのタイルセット画像を
// 別のPNGに差し替えたりしても、Tiled側でその画像を指定するだけで
// 自動的に反映される(コード側に画像を個別登録する必要はない)。
// 同じ画像を複数マップが使い回す場合は2回目以降キャッシュから返す。
func mapTilesetImage(tmap TiledMap) (*ebiten.Image, error) {
	if len(tmap.Tilesets) == 0 || tmap.Tilesets[0].Image == "" {
		return nil, fmt.Errorf("マップにタイルセットが設定されていません")
	}
	imgPath := resolveTilesetImagePath(tmap.Tilesets[0].Image)

	if img, ok := tilesetImageCache[imgPath]; ok {
		return img, nil
	}

	img, err := loadAssetImage(imgPath)
	if err != nil {
		return nil, err
	}
	tilesetImageCache[imgPath] = img
	return img, nil
}

type TiledLayer struct {
	Data    []int         `json:"data"`
	Name    string        `json:"name"`
	Type    string        `json:"type"`
	Objects []TiledObject `json:"objects"`
}

type TiledObject struct {
	ID         int             `json:"id"`
	X          float64         `json:"x"`
	Y          float64         `json:"y"`
	Width      float64         `json:"width"`
	Height     float64         `json:"height"`
	Name       string          `json:"name"`
	Polygon    []TiledPoint    `json:"polygon"`
	Properties []TiledProperty `json:"properties"`
}

type TiledPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type TiledProperty struct {
	Name  string      `json:"name"`
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

func objProps(obj TiledObject) map[string]string {
	m := make(map[string]string, len(obj.Properties))
	for _, p := range obj.Properties {
		m[strings.ToLower(p.Name)] = fmt.Sprintf("%v", p.Value)
	}
	return m
}

func objPropInt(obj TiledObject, name string) (int, bool) {
	name = strings.ToLower(name)
	for _, p := range obj.Properties {
		if strings.ToLower(p.Name) == name {
			if v, ok := p.Value.(float64); ok {
				return int(v), true
			}
		}
	}
	return 0, false
}

func objPropBool(obj TiledObject, name string) (bool, bool) {
	name = strings.ToLower(name)
	for _, p := range obj.Properties {
		if strings.ToLower(p.Name) == name {
			if v, ok := p.Value.(bool); ok {
				return v, true
			}
		}
	}
	return false, false
}

func objContains(obj TiledObject, x, y float64) bool {
	return x >= obj.X && x <= obj.X+obj.Width && y >= obj.Y && y <= obj.Y+obj.Height
}

func objContainsMargin(obj TiledObject, x, y, margin float64) bool {
	return x >= obj.X-margin && x <= obj.X+obj.Width+margin &&
		y >= obj.Y-margin && y <= obj.Y+obj.Height+margin
}

func dialoguePagesFromText(text string) []EventCommand {
	pages := strings.Split(text, "|")
	cmds := make([]EventCommand, 0, len(pages))
	for _, pageText := range pages {
		cmds = append(cmds, EventCommand{Speaker: "", Text: pageText})
	}
	return cmds
}

// resolveEventDialogue は会話イベントの表示内容を解決する。seenがtrue
// (=このオブジェクトと過去に一度会話済み)の場合、まず個別に用意された
// 2回目以降用のセリフ(event_story_系なら"<id>_repeat"、直書きテキスト系なら
// repeatText プロパティ)を探し、無ければ通常時のセリフにフォールバックする。
func resolveEventDialogue(text string, repeatText string, seen bool) BossDialogue {
	if id, ok := strings.CutPrefix(text, storyTextPrefix); ok {
		if d, found := GetStoryDialogue(id, seen); found {
			return d
		}
		return BossDialogue{Commands: []EventCommand{{Speaker: "", Text: "……"}}}
	}
	if seen && repeatText != "" {
		return BossDialogue{Commands: dialoguePagesFromText(repeatText)}
	}
	return BossDialogue{Commands: dialoguePagesFromText(text)}
}

func (s *FieldScene) applyDialogue(bd BossDialogue) {
	s.msgTexts = bd.Commands
	s.msgBGM = bd.BGM
	s.msg.SpeakerSides = bd.SpeakerSides
	s.msgBackground = bd.Background
}

func (s *FieldScene) applyCutsceneMessage() {
	if s.cutsceneHasBossID {
		bd := GetEventCommands(s.cutsceneMessage, s.game)
		bossType := strings.TrimPrefix(s.cutsceneMessage, "event_")
		bd.Commands = append(bd.Commands, EventCommand{
			Speaker: "SYSTEM_COMMAND",
			Text:    "START_BATTLE_" + bossType,
		})
		s.applyDialogue(bd)
		return
	}
	// トリガー型の演出は一度発火したら二度と発火しない(呼び出し元でSeenEvents
	// により再発火自体をブロックしている)ので、2回目以降セリフの分岐は不要。
	s.applyDialogue(resolveEventDialogue(s.cutsceneMessage, "", false))
}

type EnemyField struct {
	x, y    float64
	Boss    bool
	Type    string
	Message string
}

type MoveStep struct {
	Dir  int
	Dist float64
}

type CollisionRect struct {
	X, Y, Width, Height float64
}

type CollisionPolygon struct {
	Points []TiledPoint
}

type FieldScene struct {
	game               *Game
	tileMap            TiledMap
	mapTileImg         *ebiten.Image
	collisions         []CollisionRect
	collisionPolygons  []CollisionPolygon
	playerCfg          FieldPlayerConfig
	px, py             float64
	dir                int
	animCount          int
	enemies            []*EnemyField
	currentMap         string
	mapBGM             string
	msgBGM             string
	msgBackground      string
	msgTexts           []EventCommand
	msgIndex           int
	isMsgActive        bool
	msgSkipHoldElapsed float64
	autoMode           bool
	autoWaitElapsed    float64
	msgLog             []EventCommand
	isLogActive        bool
	logScrollOffset    float64
	logCursorIndex     int
	logDrag            dragScrollState
	logScrollBarDrag   dragScrollState
	isMenuActive       bool
	menuIndex          int
	walkCooldown       float64
	safetyDistance     float64
	encounterWeight    float64

	encounterEffectActive bool
	encounterEffectTimer  float64
	encounterSnapshot     *ebiten.Image
	pendingBattleScene    *BattleScene
	isCutscene            bool
	cutsceneMessage       string
	cutsceneHasBossID     bool
	cutsceneRoute         []MoveStep
	routeIndex            int
	currentStepDist       float64
	justDefeatedBoss      int
	msg                   MessageSystem

	pendingAutoHealMessage       bool
	skillUpgradeTutorialActive   bool
	skillUpgradeTutorialPage     int
	skillUpgradeTutorialSceneImg *ebiten.Image
	skillUpgradeTutorialOverlay  *ebiten.Image

	touchStickActive   bool
	touchStickDX       float64
	touchStickDY       float64
	touchStickOriginX  float64
	touchStickOriginY  float64
	touchStickTouchID  ebiten.TouchID
	touchStickUseMouse bool
	isDashingNow       bool

	cseFadeAlpha         float64
	cseFadeMode          int
	cseFadeSpeed         float64
	pendingCutsceneMsg   string
	pendingCutsceneRoute []MoveStep

	CseFadeOutDelay  float64
	CseFadeInEarly   float64
	cseElapsed       float64
	cseTotalDuration float64
	CseDarkDuration  float64
	cseDarkElapsed   float64

	isChoiceActive  bool
	choiceQuestion  string
	choiceOptions   []string
	choiceIndex     int
	onChoiceConfirm func(selected int)
	choiceAnchorX   float64
	choiceAnchorY   float64

	isItemGetActive       bool
	itemGetName           string
	itemGetSubLabel       string
	itemGetPlainMessage   bool
	itemGetAutoCloseOnly  bool
	itemGetAutoCloseTimer float64

	wallFadeActive bool
	wallFadeKey    string
	wallFadeAlpha  float64
	wallAnimTick   int

	isDarknessActive bool
	darknessRadius   float64
	darknessOverlay  *ebiten.Image

	onDarkCallback         func()
	onFadeCompleteCallback func()
	nearExamineEvent       bool
	nearDoorEvent          bool
	pendingDoorMap         string
	pendingDoorPoint       string
	pendingDoorX           float64
	pendingDoorY           float64
	pendingDoorDir         int
	cseFadeInSpeed         float64

	objectiveDoorX   float64
	objectiveDoorY   float64
	hasObjectiveDoor bool

	blocks         []*FieldBlock
	isPushingBlock bool
	pushingBlockID string
	nearBlockID    string

	blockStepActive   bool
	blockStepElapsed  float64
	blockStepStartPX  float64
	blockStepStartPY  float64
	blockStepTargetPX float64
	blockStepTargetPY float64
	blockStepStartBX  float64
	blockStepStartBY  float64
	blockStepTargetBX float64
	blockStepTargetBY float64

	screenShakeX      float64
	screenShakeY      float64
	screenShakeTimer  float64
	screenShakeMaxDur float64
	screenShakePower  float64

	mapNameBannerActive  bool
	mapNameBannerText    string
	mapNameBannerElapsed float64
}

func NewRoomScene(game *Game, mapPath string, startX, startY float64, targetSpawnName string, startDir int) (*FieldScene, error) {
	mapData, err := loadAssetBytes(mapPath)
	if err != nil {
		return nil, err
	}
	var tmap TiledMap
	if err := json.Unmarshal(mapData, &tmap); err != nil {
		return nil, err
	}

	var roomEnemies []*EnemyField
	spawnX, spawnY := startX, startY
	spawnDir := startDir
	spawnFound := false

	for _, layer := range tmap.Layers {
		if strings.HasPrefix(layer.Name, "events") {
			for _, obj := range layer.Objects {

				if targetSpawnName != "" && obj.Name == targetSpawnName {
					spawnX = obj.X
					spawnY = obj.Y
					spawnFound = true
					if v, ok := objPropInt(obj, "dir"); ok {
						spawnDir = v
					}
					continue
				}

				p := objProps(obj)
				evType := p["type"]
				evText := p["text"]
				bossID, hasBossID := objPropInt(obj, "bossid")

				if evType == "boss" && hasBossID {
					if isBossDefeated(game, bossID) {
						continue
					}
					roomEnemies = append(roomEnemies, &EnemyField{
						x:       obj.X,
						y:       obj.Y,
						Boss:    true,
						Type:    fmt.Sprintf("boss_%d", bossID),
						Message: evText,
					})
				}
			}
		}
	}

	var collisionRects []CollisionRect
	var collisionPolygons []CollisionPolygon
	for _, layer := range tmap.Layers {
		if layer.Name == "collision" && layer.Type == "objectgroup" {
			for _, obj := range layer.Objects {
				if len(obj.Polygon) > 0 {
					pts := make([]TiledPoint, len(obj.Polygon))
					for i, p := range obj.Polygon {
						pts[i] = TiledPoint{X: obj.X + p.X, Y: obj.Y + p.Y}
					}
					collisionPolygons = append(collisionPolygons, CollisionPolygon{Points: pts})
					continue
				}
				collisionRects = append(collisionRects, CollisionRect{
					X:      obj.X,
					Y:      obj.Y,
					Width:  obj.Width,
					Height: obj.Height,
				})
			}
		}
	}

	for _, layer := range tmap.Layers {
		if !strings.HasPrefix(layer.Name, "events") {
			continue
		}
		for _, obj := range layer.Objects {
			p := objProps(obj)
			if !isChestObj(p) && !isLeverObj(p) {
				continue
			}
			collisionRects = append(collisionRects, CollisionRect{
				X:      obj.X,
				Y:      obj.Y,
				Width:  obj.Width,
				Height: obj.Height,
			})
		}
	}

	var blocks []*FieldBlock
	for _, layer := range tmap.Layers {
		if !strings.HasPrefix(layer.Name, "events") {
			continue
		}
		for _, obj := range layer.Objects {
			p := objProps(obj)
			if !isBlockObj(p) {
				continue
			}
			id := p["id"]
			if id == "" {
				continue
			}
			bx, by := obj.X, obj.Y
			if pos, ok := game.BlockPositions[blockKey(mapPath, id)]; ok {
				bx, by = pos[0], pos[1]
			}
			blocks = append(blocks, &FieldBlock{ID: id, X: bx, Y: by, Width: obj.Width, Height: obj.Height})
		}
	}

	if targetSpawnName != "" && !spawnFound {
		spawnX = 190
		spawnY = 320
	}

	targetTileImg := game.Tilesets["default"]
	if len(tmap.Tilesets) > 0 && tmap.Tilesets[0].Image != "" {
		img, err := mapTilesetImage(tmap)
		if err != nil {
			return nil, fmt.Errorf("タイルセット画像の読み込みに失敗しました(%s): %w", mapPath, err)
		}
		targetTileImg = img
	}

	playerCfg, playerSheet, err := LoadFieldPlayerConfig("assets/field_player.json")
	if err != nil {
		return nil, err
	}
	game.SpriteSheet = playerSheet

	scene := &FieldScene{
		game:              game,
		tileMap:           tmap,
		mapTileImg:        targetTileImg,
		playerCfg:         playerCfg,
		px:                spawnX,
		py:                spawnY,
		dir:               spawnDir,
		enemies:           roomEnemies,
		currentMap:        mapPath,
		isMenuActive:      false,
		menuIndex:         0,
		walkCooldown:      0,
		safetyDistance:    64.0,
		encounterWeight:   0.0,
		justDefeatedBoss:  0,
		CseFadeOutDelay:   0.3,
		CseFadeInEarly:    0.5,
		CseDarkDuration:   1.0,
		collisions:        collisionRects,
		collisionPolygons: collisionPolygons,
		touchStickTouchID: touchStickNoTouch,
		touchStickOriginX: touchPadCenterX,
		touchStickOriginY: touchPadCenterY,
		blocks:            blocks,
	}

	scene.msg.WindowAlpha = 0.7

	scene.msg.WindowImg = game.WindowImg

	mapBGM := bgmField1
	if key, ok := tmap.mapBGMKey(); ok {
		if path, found := resolveBGMKey(key); found {
			mapBGM = path
		}
	}
	scene.mapBGM = mapBGM

	if name, ok := tmap.mapDisplayName(); ok {
		scene.mapNameBannerActive = true
		scene.mapNameBannerText = name
	}

	if tmap.mapAutoHeal() {
		scene.healParty()
		if game.SeenAutoHealMapIntro == nil {
			game.SeenAutoHealMapIntro = make(map[string]bool)
		}
		if !game.SeenAutoHealMapIntro[mapPath] {
			game.SeenAutoHealMapIntro[mapPath] = true
			scene.pendingAutoHealMessage = true
		}
	}

	return scene, nil
}

// desiredBGM はシーン切り替えの画面フェードと同じ長さでマップBGMを
// フェードインさせる。ドア移動のような短い暗転(0.3秒)なら曲もさっと
// 切り替わり、タイトルからのニューゲーム(1.5秒)ならゆっくり立ち上がる。
func (s *FieldScene) desiredBGM(transitionDuration float64) (string, float64, bool) {
	return s.mapBGM, transitionDuration, false
}

var globalActiveFieldInstanceForSave *FieldScene

const autoHealIntroMessage = "ダンジョンに入ると自動的に回復します"
const autoHealIntroMessageDuration = 2.5

// マップ入場時に左上へ表示する地名バナーの表示時間(フェード開始前)と、
// そこからフェードアウトし切るまでの時間。
const (
	mapNameBannerShowDuration = 2.2
	mapNameBannerFadeDuration = 0.8
)

// locationNameFromMap はセーブデータに記録する地名を返す。マップ全体の
// カスタムプロパティ"displayname"があればそれを使い、無ければマップの
// パスをそのまま表示名として使う。
func locationNameFromMap(mapPath string) string {
	data, err := loadAssetBytesCached(mapPath)
	if err != nil {
		return mapPath
	}
	var tmap TiledMap
	if err := json.Unmarshal(data, &tmap); err != nil {
		return mapPath
	}
	if name, ok := tmap.mapDisplayName(); ok {
		return name
	}
	return mapPath
}

func isBossDefeated(g *Game, bossID int) bool {
	if bossID < 1 || bossID > 4 {
		return false
	}
	return g.BossDefeatedFlags[bossID-1]
}

func bossIndexFromEnemyType(enemyType string) (int, bool) {
	if !strings.HasPrefix(enemyType, "boss_") {
		return -1, false
	}
	numStr := strings.TrimPrefix(enemyType, "boss_")
	bossNum, err := strconv.Atoi(numStr)
	if err != nil {
		return -1, false
	}
	idx := bossNum - 1
	if idx < 0 || idx >= 4 {
		return -1, false
	}
	return idx, true
}

func SaveGame(slot int, mapPath string, x, y float64, hp [4]int) error {
	if globalActiveFieldInstanceForSave == nil || globalActiveFieldInstanceForSave.game == nil {
		return fmt.Errorf("game instance not found")
	}
	g := globalActiveFieldInstanceForSave.game

	data := SaveData{
		SlotID:                   slot,
		LocationName:             locationNameFromMap(mapPath),
		CurrentMap:               mapPath,
		PlayerX:                  x,
		PlayerY:                  y,
		PlayerDir:                globalActiveFieldInstanceForSave.dir,
		PlayerHP:                 hp,
		PlayerMaxHP:              g.PlayerMaxHP,
		PlayerMP:                 g.PlayerMP,
		PlayerMaxMP:              g.PlayerMaxMP,
		PlayerAtk:                g.PlayerAtk,
		PlayerMagicAtk:           g.PlayerMagicAtk,
		PlayerDef:                g.PlayerDef,
		PlayerMagicDef:           g.PlayerMagicDef,
		PlayerSpd:                g.PlayerSpd,
		PlayerLuck:               g.PlayerLuck,
		PlayerSP:                 g.PlayerSP,
		PlayerSkillLv:            g.PlayerSkillLv,
		BossDefeatedFlags:        g.BossDefeatedFlags,
		PlayerLv:                 g.PlayerLv,
		PlayerEXP:                g.PlayerEXP,
		PlayerNextEXP:            g.PlayerNextEXP,
		PlayTime:                 g.TotalPlayTime,
		SavedAt:                  time.Now().Format("2006/01/02"),
		Inventory:                g.Inventory,
		OpenedChests:             g.OpenedChests,
		UnlockedWalls:            g.UnlockedWalls,
		Keys:                     g.Keys,
		RaisedLevers:             g.RaisedLevers,
		SeenAutoHealMapIntro:     g.SeenAutoHealMapIntro,
		SeenEvents:               g.SeenEvents,
		SeenBattleTutorial:       g.SeenBattleTutorial,
		SeenGaugeTutorial:        g.SeenGaugeTutorial,
		SeenSkillUpgradeTutorial: g.SeenSkillUpgradeTutorial,
		BlockPositions:           g.BlockPositions,
		UnlockedBlockDoors:       g.UnlockedBlockDoors,
	}

	file, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return writeRuntimeFile(saveFilePath(slot), file)
}

func (s *FieldScene) cameraPosition() (camX, camY float64) {
	anchorX, anchorY := s.playerCfg.CameraAnchorAt(s.px, s.py)

	camX = float64(gameWidth)/2 - anchorX
	camY = float64(gameHeight)/2 - anchorY

	mapPxW := float64(s.tileMap.Width * s.tileMap.TileWidth)
	mapPxH := float64(s.tileMap.Height * s.tileMap.TileHeight)

	if mapPxW > float64(gameWidth) {
		minCamX := float64(gameWidth) - mapPxW
		maxCamX := 0.0
		if camX > maxCamX {
			camX = maxCamX
		}
		if camX < minCamX {
			camX = minCamX
		}
	} else {
		camX = (float64(gameWidth) - mapPxW) / 2
	}

	if mapPxH > float64(gameHeight) {
		minCamY := float64(gameHeight) - mapPxH
		maxCamY := 0.0
		if camY > maxCamY {
			camY = maxCamY
		}
		if camY < minCamY {
			camY = minCamY
		}
	} else {
		camY = (float64(gameHeight) - mapPxH) / 2
	}

	camX = math.Floor(camX + 0.5)
	camY = math.Floor(camY + 0.5)

	return camX, camY
}

func (s *FieldScene) healParty() {
	for i := 0; i < 4; i++ {
		s.game.PlayerHP[i] = s.game.PlayerMaxHP[i]
		s.game.PlayerMP[i] = s.game.PlayerMaxMP[i]
	}
}

// chestKey はマップ内のオブジェクト1つを一意に識別するキーを返す。
// TiledのIDはレベル調整でオブジェクトを動かしても変わらない安定した
// 識別子なので、あればそれを使う。ID未設定(0)のときだけ従来通り座標に
// フォールバックする(コード内で手動組み立てしたTiledObject向けの保険で、
// Tiledで実際に配置したオブジェクトは常にID>0を持つ)。
func chestKey(mapPath string, obj TiledObject) string {
	if obj.ID != 0 {
		return fmt.Sprintf("%s|id%d", mapPath, obj.ID)
	}
	return fmt.Sprintf("%s|%.1f_%.1f", mapPath, obj.X, obj.Y)
}

func (s *FieldScene) openChest(obj TiledObject, itemID string) {
	if s.game.OpenedChests == nil {
		s.game.OpenedChests = make(map[string]bool)
	}

	key := chestKey(s.currentMap, obj)
	if s.game.OpenedChests[key] {
		return
	}
	s.game.OpenedChests[key] = true
	s.nearExamineEvent = false
	s.game.Audio.PlaySEByKey("treasure_open")

	def, ok := GetItemDef(itemID)
	if !ok {
		s.msgTexts = []EventCommand{{Speaker: "", Text: "何も入っていなかった"}}
		s.msgBGM = ""
		s.msgIndex = 0
		s.beginMessage()
		return
	}
	s.game.AddItem(itemID, 1)
	s.openItemGetPopup(def.Name, "")
}

func (s *FieldScene) openItemGetPopup(itemName string, subLabel string) {
	s.itemGetName = itemName
	s.itemGetSubLabel = subLabel
	s.itemGetPlainMessage = false
	s.isItemGetActive = true
	s.game.Audio.PlaySEByKey("item_get")
}

func (s *FieldScene) openCenterMessagePopup(message string) {
	s.itemGetName = message
	s.itemGetSubLabel = ""
	s.itemGetPlainMessage = true
	s.itemGetAutoCloseOnly = false
	s.itemGetAutoCloseTimer = 0
	s.isItemGetActive = true
}

// openCenterMessagePopupTimed is like openCenterMessagePopup but ignores all
// player input (decide key, escape, tap) and instead closes itself once
// duration seconds have passed, so the player can't accidentally blow past
// the message out of habit (e.g. mashing the action key while walking).
func (s *FieldScene) openCenterMessagePopupTimed(message string, duration float64) {
	s.openCenterMessagePopup(message)
	s.itemGetAutoCloseOnly = true
	s.itemGetAutoCloseTimer = duration
}

func (s *FieldScene) openKeyChest(obj TiledObject, keyName string) {
	if s.game.OpenedChests == nil {
		s.game.OpenedChests = make(map[string]bool)
	}

	key := chestKey(s.currentMap, obj)
	if s.game.OpenedChests[key] {
		return
	}
	s.game.OpenedChests[key] = true
	s.nearExamineEvent = false
	s.game.Audio.PlaySEByKey("treasure_open")

	if keyName == "" {
		s.msgTexts = []EventCommand{{Speaker: "", Text: "何も入っていなかった"}}
		s.msgBGM = ""
		s.msgIndex = 0
		s.beginMessage()
		return
	}

	if s.game.Keys == nil {
		s.game.Keys = make(map[string]int)
	}
	s.game.Keys[keyName]++

	subLabel := ""
	if names, ok := loadWallKeyGroups()[keyName]; ok {
		subLabel = fmt.Sprintf("残り%d個", remainingKeysInGroup(names, s.game.Keys))
	}
	s.openItemGetPopup(keyName, subLabel)
}

const wallFadeDuration = 0.5

func (s *FieldScene) examineWall(obj TiledObject) {
	if s.game.UnlockedWalls == nil {
		s.game.UnlockedWalls = make(map[string]bool)
	}

	key := chestKey(s.currentMap, obj)
	if s.game.UnlockedWalls[key] {
		return
	}

	names := splitKeyNames(objProps(obj)["keys"])
	missing := remainingKeysInGroup(names, s.game.Keys)

	s.nearExamineEvent = false

	if missing > 0 {
		s.game.Audio.PlaySEByKey("error")
		s.openCenterMessagePopup("ここから先に進むには鍵が必要なようだ")
		return
	}

	s.wallFadeActive = true
	s.wallFadeKey = key
	s.wallFadeAlpha = 1.0
	s.game.Audio.PlaySEByKey("sliding_door")
}

func (s *FieldScene) updateWallFade(dt float64) {
	s.wallFadeAlpha -= dt / wallFadeDuration
	if s.wallFadeAlpha <= 0 {
		s.wallFadeAlpha = 0
		s.wallFadeActive = false
		s.game.UnlockedWalls[s.wallFadeKey] = true
		s.openCenterMessagePopup("閉ざされた壁が解放された！")
	}
}

func (s *FieldScene) wallIsOpen(obj TiledObject) bool {
	if leverID := objProps(obj)["lever"]; leverID != "" {
		return s.game.RaisedLevers[leverID]
	}
	return s.game.UnlockedWalls[chestKey(s.currentMap, obj)]
}

func (s *FieldScene) pullLever(obj TiledObject) {
	if s.game.RaisedLevers == nil {
		s.game.RaisedLevers = make(map[string]bool)
	}

	id := objProps(obj)["id"]

	s.nearExamineEvent = false
	if objProps(obj)["oneway"] == "true" && s.game.RaisedLevers[id] {
		s.game.Audio.PlaySEByKey("error")
		return
	}
	s.game.RaisedLevers[id] = !s.game.RaisedLevers[id]
	s.game.Audio.PlaySEByKey("lever")
}

const defaultDarknessRadius = 110.0

func (s *FieldScene) updateDarkness() {
	s.isDarknessActive = false

	for _, layer := range s.tileMap.Layers {
		if !strings.HasPrefix(layer.Name, "events") {
			continue
		}
		for _, obj := range layer.Objects {
			p := objProps(obj)
			if p["type"] != "darkness" {
				continue
			}
			if leverID := p["lever"]; leverID != "" && s.game.RaisedLevers[leverID] {
				continue
			}
			if !objContains(obj, s.px, s.py) {
				continue
			}

			s.isDarknessActive = true
			s.darknessRadius = defaultDarknessRadius
			return
		}
	}
}

func (s *FieldScene) updateObjectiveGuide() {
	s.hasObjectiveDoor = false

	loc, ok := s.game.CurrentObjectiveLocation()
	if !ok {
		return
	}
	if loc.MapPath == s.currentMap {
		return
	}

	door, ok := FindNextDoorTowards(s.currentMap, loc.MapPath)
	if !ok {
		return
	}
	s.objectiveDoorX = door.DoorX
	s.objectiveDoorY = door.DoorY
	s.hasObjectiveDoor = true
}
