package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
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
	Width      int          `json:"width"`
	Height     int          `json:"height"`
	TileWidth  int          `json:"tilewidth"`
	TileHeight int          `json:"tileheight"`
	Layers     []TiledLayer `json:"layers"`
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
	Polygon    []TiledPoint    `json:"polygon"` // ★追加：斜めの壁など多角形コリジョン用の頂点リスト（オブジェクトのx,yを原点とした相対座標）
	Properties []TiledProperty `json:"properties"`
}

// TiledPoint はTiledの多角形オブジェクトが持つ1頂点の相対座標。
type TiledPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type TiledProperty struct {
	Name  string      `json:"name"`
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

// ============================================================
// ★追加：Tiledオブジェクトのプロパティ読み取り共通ヘルパー
// これまで各所でバラバラに実装されていた
// 「プロパティ名で回して fmt.Sprintf("%v", ...) する」処理を統一する。
// プロパティ名はすべて小文字化して比較するので、
// Tiled側の大文字小文字の打ち間違いに強くなる。
// ============================================================

// objProps はオブジェクトの全プロパティを {小文字化した名前: 文字列値} の
// マップにして返す。
func objProps(obj TiledObject) map[string]string {
	m := make(map[string]string, len(obj.Properties))
	for _, p := range obj.Properties {
		m[strings.ToLower(p.Name)] = fmt.Sprintf("%v", p.Value)
	}
	return m
}

// objPropInt は指定プロパティ（数値）を int で返す。存在しない/数値でない場合は false。
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

// objPropBool は指定プロパティ（真偽値）を bool で返す。存在しない/真偽値でない場合は false。
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

// objContains は指定のワールド座標がオブジェクトの矩形内にあるかを返す。
func objContains(obj TiledObject, x, y float64) bool {
	return x >= obj.X && x <= obj.X+obj.Width && y >= obj.Y && y <= obj.Y+obj.Height
}

// objContainsMargin は objContains と同様だが、矩形をmarginぶん四方に広げた範囲で判定する。
// チェストのように本体に当たり判定があって乗れないオブジェクトを、
// 周囲から調べられるようにするために使う。
func objContainsMargin(obj TiledObject, x, y, margin float64) bool {
	return x >= obj.X-margin && x <= obj.X+obj.Width+margin &&
		y >= obj.Y-margin && y <= obj.Y+obj.Height+margin
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

// CollisionPolygon は斜めの壁など、矩形では表現できない形の当たり判定。
// Points はワールド座標に変換済みの頂点リスト（obj.X, obj.Y を加算済み）。
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
	msgTexts           []EventCommand
	msgIndex           int
	isMsgActive        bool
	msgSkipHoldElapsed float64
	autoMode           bool           // 🔥 追加：会話オート送りON/OFF
	autoWaitElapsed    float64        // 🔥 追加：オート時の待機経過秒数
	msgLog             []EventCommand // 🔥 追加：会話ログ（直近msgLogMaxEntries件）
	isLogActive        bool           // 🔥 追加：ログ画面を開いているか
	logScrollOffset    float64        // 🔥 追加：ログ画面のスクロール位置（下端=0、上に行くほど増える）
	logCursorIndex     int            // 🔥 追加：ログ画面で選択中のログのインデックス（msgLogの添字、0=一番古い）
	isMenuActive       bool
	menuIndex          int
	walkCooldown       float64
	safetyDistance     float64
	encounterWeight    float64
	isCutscene         bool
	cutsceneMessage    string
	cutsceneRoute      []MoveStep
	routeIndex         int
	currentStepDist    float64
	justDefeatedBoss   int
	msg                MessageSystem // 🔥 追加：テキスト描画システム

	cseFadeAlpha         float64
	cseFadeMode          int
	cseFadeSpeed         float64
	pendingCutsceneMsg   string
	pendingCutsceneRoute []MoveStep

	CseFadeOutDelay  float64 // 移動開始から何秒後にフェードアウト（デフォルト0.3）
	CseFadeInEarly   float64 // 移動終了の何秒前にフェードイン（デフォルト0.5）
	cseElapsed       float64
	cseTotalDuration float64
	CseDarkDuration  float64 // 暗い状態で止まる秒数
	cseDarkElapsed   float64 // 暗い状態の経過時間

	isChoiceActive  bool
	choiceQuestion  string
	choiceOptions   []string
	choiceIndex     int
	onChoiceConfirm func(selected int)
	choiceAnchorX   float64
	choiceAnchorY   float64

	// ★追加：アイテム入手時、画面中央に表示する専用ウィンドウ
	isItemGetActive bool
	itemGetName     string

	// ★追加：暗転フェード汎用コールバック
	onDarkCallback         func() // 暗転MAX到達時に1回だけ呼ばれる
	onFadeCompleteCallback func() // フェードイン完了時に1回だけ呼ばれる（未設定ならbeginMessage）
	nearExamineEvent       bool
	nearDoorEvent          bool   // ★追加：ドア（ワープ）に近づいているか
	pendingDoorMap         string // ★追加：確定キーで遷移する先のマップ
	pendingDoorPoint       string // ★追加：確定キーで遷移する先のスポーン名
	pendingDoorX           float64
	pendingDoorY           float64
	pendingDoorDir         int
	cseFadeInSpeed         float64

	objectiveDoorX   float64
	objectiveDoorY   float64
	hasObjectiveDoor bool
	// ★変更：isDashingはGame側(g.IsDashing)に移動したため削除。
	// マップ移動でFieldSceneが再生成されてもダッシュ状態を維持するため。
}

// NewRoomScene は指定マップでフィールドシーンを生成する。
// startDir: targetSpawnNameで方向が見つからなかった場合に使う初期向き
//
//	（ロード直後やバトルから戻るときなど、スポーンイベントを経由しない場合に使用）
func NewRoomScene(game *Game, mapPath string, startX, startY float64, targetSpawnName string, startDir int) (*FieldScene, error) {
	mapData, err := os.ReadFile(mapPath)
	if err != nil {
		return nil, err
	}
	var tmap TiledMap
	if err := json.Unmarshal(mapData, &tmap); err != nil {
		return nil, err
	}

	var roomEnemies []*EnemyField
	spawnX, spawnY := startX, startY
	spawnDir := startDir // デフォルト値。マップ側のspawnオブジェクトにdirプロパティがあれば上書きされる
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
					// ★追加：斜めの壁など、Tiledの多角形描画ツールで作ったオブジェクト。
					// polygonの座標はオブジェクト原点(obj.X, obj.Y)からの相対座標なので
					// ここでワールド座標に変換してから保持する。
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

	// ★追加：チェスト（event_chest_）は見た目どおり通行不可にしたいので、
	// オブジェクトの矩形（Width/Height）をそのまま壁判定にも使う。
	// collisionレイヤーに別途壁オブジェクトを置く必要はない。
	for _, layer := range tmap.Layers {
		if !strings.HasPrefix(layer.Name, "events") {
			continue
		}
		for _, obj := range layer.Objects {
			p := objProps(obj)
			if p["type"] != "event" || !strings.HasPrefix(p["text"], "event_chest_") {
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

	if targetSpawnName != "" && !spawnFound {
		fmt.Printf("警告: マップ%sにスポーン地点 %q が見つかりません。デフォルト座標にフォールバックします\n", mapPath, targetSpawnName)
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
	game.SpriteSheet = playerSheet // バトル等の互換用

	// ★修正：game.TileImg（グローバル単一状態）への書き込みを廃止。
	// 以前はここで game.TileImg = targetTileImg としていたため、
	// フェード遷移中に古いシーンが新しいマップのタイル画像で
	// 描画されてしまう可能性があった。描画は各シーンが持つ
	// scene.mapTileImg を参照するようにする（field_draw.go 参照）。

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
	}

	// 💡 修正点: ここにウィンドウの不透明度を設定するコードを追加します
	// 0.7の数値を 0.0(透明) 〜 1.0(不透明) の間で好みの透け具合に調整してください
	scene.msg.WindowAlpha = 0.7

	// window.png が Game にあれば MessageSystem に渡す
	if game.WindowImg != nil {
		scene.msg.WindowImg = game.WindowImg
	}

	game.Audio.PlayBGMFadeIn(bgmFieldSchool, 2.0)

	return scene, nil
}

var globalActiveFieldInstanceForSave *FieldScene

// locationNameFromMap はマップパスから表示用の場所名を返す
// 新しいマップを追加したらここに追記する
func locationNameFromMap(mapPath string) string {
	names := map[string]string{
		"assets/maps/School_Map_1.tmj": "理科室",
		"assets/maps/ダンジョンA.tmj":       "ダンジョンA",
		// 例: "assets/maps/School_Map_3.tmj": "屋上",
	}
	if name, ok := names[mapPath]; ok {
		return name
	}
	return mapPath
}

// isBossDefeated はboss_idの撃破済みフラグを返す（範囲外は未撃破扱い）
func isBossDefeated(g *Game, bossID int) bool {
	if bossID < 1 || bossID > 4 {
		return false
	}
	return g.BossDefeatedFlags[bossID-1]
}

// bossIndexFromEnemyType は "boss_1" のような文字列から0始まりのboss配列インデックスを返す。
// "boss_"で始まらない、または数値変換に失敗した場合は (-1, false) を返す。
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
		SlotID:            slot,
		LocationName:      locationNameFromMap(mapPath),
		CurrentMap:        mapPath,
		PlayerX:           x,
		PlayerY:           y,
		PlayerDir:         globalActiveFieldInstanceForSave.dir, // ← 追加：現在の向きを保存
		PlayerHP:          hp,
		PlayerMaxHP:       g.PlayerMaxHP,
		PlayerMP:          g.PlayerMP,
		PlayerMaxMP:       g.PlayerMaxMP,
		PlayerAtk:         g.PlayerAtk,
		PlayerMagicAtk:    g.PlayerMagicAtk,
		PlayerDef:         g.PlayerDef,
		PlayerMagicDef:    g.PlayerMagicDef,
		PlayerSpd:         g.PlayerSpd,
		PlayerLuck:        g.PlayerLuck,
		PlayerSP:          g.PlayerSP,
		PlayerSkillLv:     g.PlayerSkillLv,
		BossDefeatedFlags: g.BossDefeatedFlags,
		PlayerLv:          g.PlayerLv,
		PlayerEXP:         g.PlayerEXP,
		PlayerNextEXP:     g.PlayerNextEXP,
		PlayTime:          g.TotalPlayTime,
		SavedAt:           time.Now().Format("2006/01/02"),
		Inventory:         g.Inventory,
		OpenedChests:      g.OpenedChests,
	}

	file, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(saveFilePath(slot), file, 0666)
}

func (s *FieldScene) cameraPosition() (camX, camY float64) {
	anchorX, anchorY := s.playerCfg.CameraAnchorAt(s.px, s.py)

	camX = float64(gameWidth)/2 - anchorX
	camY = float64(gameHeight)/2 - anchorY

	// マップ全体のピクセルサイズ
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

// ============================================================
// 休憩イベント（旧 field_rest.go）
// ============================================================

const fadeTimeRest = 0.5

// startRestEvent は休憩の確認吹き出しを開く（msgシステム・BGMには一切触らない）
func (s *FieldScene) startRestEvent(obj TiledObject) {
	if s.isChoiceActive || s.cseFadeMode != 0 || s.isCutscene {
		return
	}
	s.choiceAnchorX = obj.X + obj.Width/2
	s.choiceAnchorY = obj.Y

	s.choiceQuestion = "休みますか？"
	s.choiceOptions = []string{"はい", "いいえ"}
	s.choiceIndex = 0
	s.isChoiceActive = true

	s.onChoiceConfirm = func(selected int) {
		if selected == 0 { // はい
			s.startRestFade()
		}
		// いいえ：何もせず通常操作に戻るだけ
	}
}

func (s *FieldScene) startRestFade() {
	if s.cseFadeMode != 0 { // 既にフェード処理中なら何もしない
		return
	}
	s.cseFadeAlpha = 0
	s.cseFadeMode = 2
	s.cseFadeSpeed = 1.0 / fadeTimeRest
	s.cseFadeInSpeed = 1.0 / fadeTimeRest
	s.cseDarkElapsed = 0
	s.onDarkCallback = s.healParty
	s.onFadeCompleteCallback = func() {}
}

func (s *FieldScene) healParty() {
	for i := 0; i < 4; i++ {
		s.game.PlayerHP[i] = s.game.PlayerMaxHP[i]
		s.game.PlayerMP[i] = s.game.PlayerMaxMP[i]
	}
}

// ============================================================
// チェスト（宝箱）イベント
// ============================================================

// chestKey は「マップパス＋座標」からチェスト1個分の一意なキーを作る。
// 開封済みかどうかをGame.OpenedChestsにこのキーで記録する。
func chestKey(mapPath string, obj TiledObject) string {
	return fmt.Sprintf("%s|%.1f_%.1f", mapPath, obj.X, obj.Y)
}

// openChest はチェスト（type=event, text="event_chest_<アイテムID>"）を
// 調べた時の処理。未開封ならアイテムを入手してフラグを立てる。
// 開封済みなら何もしない（調べても無反応）。
func (s *FieldScene) openChest(obj TiledObject, itemID string) {
	if s.game.OpenedChests == nil {
		s.game.OpenedChests = make(map[string]bool)
	}

	key := chestKey(s.currentMap, obj)
	if s.game.OpenedChests[key] {
		return
	}
	s.game.OpenedChests[key] = true
	// 決定押下と同フレームで「▼ 調べる」を消す（次のUpdateの再判定を待たない）
	s.nearExamineEvent = false

	def, ok := GetItemDef(itemID)
	if !ok {
		fmt.Printf("警告: チェストのアイテムID %q が見つかりません\n", itemID)
		s.msgTexts = []EventCommand{{Speaker: "", Text: "何も入っていなかった"}}
		s.msgIndex = 0
		s.beginMessage()
		return
	}
	s.game.AddItem(itemID, 1)
	s.openItemGetPopup(def.Name)
}

// openItemGetPopup はアイテム入手時に画面中央へ表示する専用ウィンドウを開く。
// 通常の会話メッセージ（下部ウィンドウ）とは別枠の表示で、
// 決定キーで閉じるまでプレイヤーの移動や他の操作を止める。
func (s *FieldScene) openItemGetPopup(itemName string) {
	s.itemGetName = itemName
	s.isItemGetActive = true
}

// updateObjectiveGuide は現在の目的地に応じて、
// 別マップにある場合の誘導先ドア座標をセットする。同マップならミニマップ側で処理するのでここでは何もしない。
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
