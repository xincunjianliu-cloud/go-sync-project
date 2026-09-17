package main

import (
	"encoding/json"
	"fmt"
	"math"
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

type TiledMap struct {
	Width      int             `json:"width"`
	Height     int             `json:"height"`
	TileWidth  int             `json:"tilewidth"`
	TileHeight int             `json:"tileheight"`
	Layers     []TiledLayer    `json:"layers"`
	Properties []TiledProperty `json:"properties"`
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

type TiledLayer struct {
	Data    []int         `json:"data"`
	Name    string        `json:"name"`
	Type    string        `json:"type"`
	Objects []TiledObject `json:"objects"`
}

type TiledObject struct {
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

func resolveEventDialogue(text string) ([]EventCommand, map[string]int, string) {
	if id, ok := strings.CutPrefix(text, "event_story_"); ok {
		if d, found := GetStoryDialogue(id); found {
			return d.Commands, d.SpeakerSlots, d.BGM
		}
		return []EventCommand{{Speaker: "", Text: "……"}}, nil, ""
	}
	return dialoguePagesFromText(text), nil, ""
}

func (s *FieldScene) applyCutsceneMessage() {
	if s.cutsceneHasBossID {
		bd := GetEventCommands(s.cutsceneMessage, s.game)
		bossType := strings.TrimPrefix(s.cutsceneMessage, "event_")
		s.msgTexts = append(bd.Commands, EventCommand{
			Speaker: "SYSTEM_COMMAND",
			Text:    "START_BATTLE_" + bossType,
		})
		s.msg.SpeakerToSlot = bd.SpeakerSlots
		s.msgBGM = bd.BGM
		return
	}
	s.msgTexts, s.msg.SpeakerToSlot, s.msgBGM = resolveEventDialogue(s.cutsceneMessage)
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
	isCutscene         bool
	cutsceneMessage    string
	cutsceneHasBossID  bool
	cutsceneRoute      []MoveStep
	routeIndex         int
	currentStepDist    float64
	justDefeatedBoss   int
	msg                MessageSystem

	pendingAutoHealMessage bool

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
			isFixture := strings.HasPrefix(p["text"], "event_chest_") || p["text"] == "event_lever"
			if p["type"] != "event" || !isFixture {
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
			if p["type"] != "event" || p["text"] != "event_block" {
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

	var targetTileImg *ebiten.Image
	if strings.Contains(mapPath, "School") {
		targetTileImg = game.Tilesets["rouka"]
	} else if strings.Contains(mapPath, "dungeon") || strings.Contains(mapPath, "ダンジョン") {
		targetTileImg = game.Tilesets["dungeon"]
	} else {
		targetTileImg = game.Tilesets["default"]
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

	if autoHealMaps[mapPath] {
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

var autoHealMaps = map[string]bool{
	"assets/maps/ダンジョンA.tmj": true,
}

const autoHealIntroMessage = "ダンジョンに入ると自動的に回復します"
const autoHealIntroMessageDuration = 2.5

func locationNameFromMap(mapPath string) string {
	names := map[string]string{
		"assets/maps/School_Map_1.tmj": "理科室",
		"assets/maps/ダンジョンA.tmj":       "ダンジョンA",
	}
	if name, ok := names[mapPath]; ok {
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
		SlotID:               slot,
		LocationName:         locationNameFromMap(mapPath),
		CurrentMap:           mapPath,
		PlayerX:              x,
		PlayerY:              y,
		PlayerDir:            globalActiveFieldInstanceForSave.dir,
		PlayerHP:             hp,
		PlayerMaxHP:          g.PlayerMaxHP,
		PlayerMP:             g.PlayerMP,
		PlayerMaxMP:          g.PlayerMaxMP,
		PlayerAtk:            g.PlayerAtk,
		PlayerMagicAtk:       g.PlayerMagicAtk,
		PlayerDef:            g.PlayerDef,
		PlayerMagicDef:       g.PlayerMagicDef,
		PlayerSpd:            g.PlayerSpd,
		PlayerLuck:           g.PlayerLuck,
		PlayerSP:             g.PlayerSP,
		PlayerSkillLv:        g.PlayerSkillLv,
		BossDefeatedFlags:    g.BossDefeatedFlags,
		PlayerLv:             g.PlayerLv,
		PlayerEXP:            g.PlayerEXP,
		PlayerNextEXP:        g.PlayerNextEXP,
		PlayTime:             g.TotalPlayTime,
		SavedAt:              time.Now().Format("2006/01/02"),
		Inventory:            g.Inventory,
		OpenedChests:         g.OpenedChests,
		UnlockedWalls:        g.UnlockedWalls,
		Keys:                 g.Keys,
		RaisedLevers:         g.RaisedLevers,
		SeenAutoHealMapIntro: g.SeenAutoHealMapIntro,
		BlockPositions:       g.BlockPositions,
		UnlockedBlockDoors:   g.UnlockedBlockDoors,
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

func chestKey(mapPath string, obj TiledObject) string {
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
