package main

// battle_types.go: バトル関連の型定義・定数・BattleSceneの初期化

import (
	"image/color"
	"math/rand"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// ★変更：EnemyType / EnemyDatabase / BossDatabase は stats_config.go の
// EnemyStats / EnemyDatabase / BossDatabase に統合した（敵と味方のステータスを
// 1ファイルにまとめるため）。ここでは定義しない。

const (
	partySize = 4
	enemyID   = 4
	atbMax    = 100.0
)

const battleLogDuration = 0.7
const gameOverMessageDuration = 1.5

// 回避演出：スプライトを右にずらす量と、元の位置へ戻る速さ。
const (
	evadeDodgeShiftX      = 26.0
	evadeDodgeReturnSpeed = 140.0 // 1秒あたりに戻るpx数
)

const (
	phaseATB = iota
	phasePlayerMenu
	phaseSkillMenu
	phaseTargetSelect
	phaseHealSelect
	phaseItemMenu
	phaseItemTarget
	phaseMessage
	phaseBattleEnd
)

const (
	cmdNormalAttack = iota
	cmdWait
	cmdSkill
	cmdFlee
)

const (
	mpCostHeal    = 5
	mpCostHealAll = 3
	mpCostPower   = 8
)

const (
	trackX         = 20.0
	trackY         = 53.0
	trackH         = 8.0
	iconSize       = 40.0
	timelineStartX = trackX + 10.0

	logPanelY = 400.0
	logPanelH = 28.0
	cmdPanelX = 600.0
	cmdPanelY = 380.0
	cmdPanelW = 280.0
	cmdPanelH = 120.0
	hintY     = 524.0

	statusBarY     = 432.0
	statusBlockW   = 135.0
	statusBarH     = 5.5
	statusBarSlant = 8.0
	statusBlockGap = 30.0
	statusStartX   = 30.0
	statusHPTextY  = 22.0
	statusHPBarY   = 40.0
	statusMPTextY  = 52.0
	statusMPBarY   = 70.0

	descX        = 20.0
	descY        = 520.0
	descFontSize = 14.0
	descScale    = 1.0

	hintOffsetX  = 420.0
	hintFontSize = 14.0
	hintScale    = 1.0
)

const (
	poseIdle        = 0
	poseReady       = 1
	poseSwingUp     = 2
	poseSwingDown   = 3
	poseSwingReturn = 4
	poseCast        = 5
	poseDamage      = 6
	poseLowHP       = 7
	poseDead        = 8
	poseWin         = 9
	poseDefend      = 10
)

// ── 新スプライトシート対応（追加分） ──────────────────────────
const (
	spriteFrameW = 64
	spriteFrameH = 96
)

const (
	poseWalk           = 11 // 歩行・前進・スライドイン
	poseAttack         = 12 // 通常攻撃
	poseReadyGlow      = 13 // コマンド/スキル選択中の発光
	poseChargeApproach = 14 // 強撃：接近
	poseChargeAttack   = 15 // 強撃：攻撃〜エフェクト
	poseFireCast       = 16 // 炎魔法：発生
	poseFireLoop       = 17 // 炎魔法：持続ループ
	poseHealCast       = 18 // 回復（その場ループ）
)

var poseRow = map[int]int{
	poseIdle:           0,
	poseDamage:         1,
	poseWalk:           2,
	poseAttack:         3,
	poseReady:          4,
	poseReadyGlow:      4,
	poseChargeApproach: 5,
	poseChargeAttack:   6,
	poseFireCast:       7,
	poseFireLoop:       8,
	poseHealCast:       9,
	poseDefend:         0,
	poseLowHP:          0,
	poseDead:           0,
	poseWin:            0,
}

var poseFrameCount = map[int]int{
	poseIdle:           2,  // 0行目
	poseDamage:         2,  // 1行目
	poseWalk:           2,  // 2行目
	poseAttack:         7,  // 3行目
	poseReady:          1,  // 4行目・左端だけ
	poseReadyGlow:      10, // 4行目・全体
	poseChargeApproach: 4,  // 5行目
	poseChargeAttack:   11, // 6行目
	poseFireCast:       16, // 7行目
	poseFireLoop:       15, // 8行目
	poseHealCast:       14, // 9行目
}

var poseLoopFrameDur = map[int]float64{
	poseIdle:     0.25, // 2コマ
	poseWalk:     0.12, // 2コマ
	poseHealCast: 0.08, // 14コマ
	poseFireLoop: 0.05, // 15コマ
}

// 攻撃演出の種類
const (
	animNormal = iota
	animCharge
	animFireMagic
)

var glowLevelRange = [4][2]int{
	{0, 1}, // スキル以外：1コマ目
	{1, 1}, // Lv1：2コマ目
	{2, 1}, // Lv2：3コマ目
	{3, 7}, // Lv3：4〜10コマ目（7コマループ）
}

const (
	resSubStillWait    = 0
	resSubWinPose      = 1
	resSubResultFadeIn = 2
	resSubBarAnimate   = 3
	resSubDoneWait     = 4
)

// levelUpPauseDuration：EXPゲージが右端まで到達してから、
// 実際にレベルを上げて数字・ゲージをリセットするまでの静止時間(秒)。
const levelUpPauseDuration = 0.35

const (
	timelineIconOffsetY   = 46.0
	introStartOffsetX     = 960.0
	introStartOffsetVertY = 540.0

	introHorizDecay = 0.0000001
	introVertDecay  = 0.0000001
	introCharDecay  = 0.0000001

	introGoalWait        = 0.15
	introATBWait         = 0.2
	introHorizToVertWait = 0.01
)

const trackW = 920.0

const (
	gaugeMaxStage = 5
	gaugePointCap = 8
)

var gaugeStageThresholds = [gaugeMaxStage - 1]int{8, 8, 8, 8}
var gaugeStageAtkBonus = [gaugeMaxStage]int{0, 3, 5, 7, 20}

var gaugeStageColors = [gaugeMaxStage]color.RGBA{
	{90, 210, 120, 255}, // 1段階目：緑
	{235, 205, 60, 255}, // 2段階目：黄色
	{235, 140, 50, 255}, // 3段階目：オレンジ
	{225, 70, 70, 255},  // 4段階目：赤
	{175, 90, 225, 255}, // 5段階目：紫
}

const (
	gaugeTriX = 15.0
	gaugeTriY = 100.0
	gaugeTriW = 90.0
	gaugeTriH = 60.0
)

// ★変更：新しいゲージ画像(gage.png)を実測して判明した値。
// 上下の枠線・区切り線はすべて太さ2px。
// 1セグメント = 8ポイント分の斜め上昇(16px) + 区切り線(2px) = 18px間隔。
// 5セグメントぶんで90px、上端の枠線2pxを含めて画像全体92pxとぴったり合う。
const (
	gaugeImgBorder      = 2.0 // 画像の縁の太さ（塗りの内側インセットに使う）
	gaugeFillPerPoint   = 2.0 // 1ポイントあたりの塗りの高さ(px)
	gaugePointsPerStage = 8   // 1セグメント(段)あたりのポイント数
	gaugeDividerHeight  = 2.0 // 区切り線の太さ(px)。段をまたぐ際にこの分だけ余分に積む
)

const (
	resultPanelWidthRatio = 0.4

	resultTitleX        = 15.0
	resultTitleY        = 10.0
	resultTitleFontSize = 40.0

	resultExpLabelX   = 40.0
	resultExpValueX   = 250.0
	resultExpY        = 48.0
	resultExpFontSize = 20.0

	resultSpLabelX   = 40.0
	resultSpValueX   = 250.0
	resultSpY        = 68.0
	resultSpFontSize = 20.0

	resultDividerX = 10.0
	resultDividerY = 90.0
	resultDividerW = 350.0
	resultDividerH = 1.0

	resultItemsHeaderX        = 30.0
	resultItemsHeaderY        = 380.0
	resultItemsHeaderFontSize = 20.0

	resultItemsDividerX = 10.0
	resultItemsDividerY = 405.0
	resultItemsDividerW = 350.0
	resultItemsDividerH = 1.0

	resultItemsNameX    = 40.0
	resultItemsCountX   = 250.0
	resultItemsStartY   = 410.0
	resultItemsRowGap   = 20.0
	resultItemsFontSize = 15.0

	resultBarStartX = 45.0
	resultBarStartY = 130.0
	resultBarRowGap = 70.0
	resultBarWAbs   = 280.0
	resultBarH      = 10.0

	// ── パーティ各行の要素はすべて、その行のバー左上(barX, barY)からの
	// 相対オフセットで位置を決めている。符号の向きはX/Yとも共通：
	// プラスでバーより右・下、マイナスでバーより左・上。
	// 行全体の縦位置・横位置を調整したいときはこのブロックだけ見ればよい。
	resultNameOffsetX  = 0.0   // プレイヤー名
	resultLevelOffsetX = 250.0 // Lv表示
	resultNameOffsetY  = -30.0 // 名前・Lv共通の縦位置

	resultExpLabelOffsetX = 4.0 // "EXP"ラベル
	resultExpLabelOffsetY = -15.0

	// resultExpTextOffsetY：現在EXP／最大EXP数値の縦位置。他と同じ上端基準オフセットだが、
	// 下端揃え(SecondaryAlign=End)で描くため、実際の基準線はresultExpCurFontSize分だけ下にずれる
	// （battle_draw_panels.goのexpBaseY計算を参照）。
	resultExpTextOffsetY = -15.0

	resultLevelUpOffsetX = 4.0 // "LEVEL UP!"
	resultLevelUpOffsetY = 12.0

	resultNameFontSize     = 15.0
	resultExpCurFontSize   = 15.0
	resultExpMaxFontSize   = 13.0
	resultExpLabelFontSize = 15.0
	resultLevelUpFontSize  = 15.0

	resultHintX        = 24.0
	resultHintYFromBtm = 20.0
	resultHintFontSize = 15.0
)

var resultBarFillColor = color.RGBA{255, 200, 130, 255}
var resultBarBgColor = color.RGBA{30, 30, 40, 255}
var resultPanelBgColor = color.RGBA{0, 0, 0, 200}
var resultLevelUpColor = color.RGBA{255, 255, 100, 255}
var resultItemsDividerColor = color.RGBA{255, 255, 255, 255}

// ★変更：素早さは「PlayerSpd」ステータス（レベルアップで個別成長）に一本化したため、
// 固定配列だった playerSpeeds は廃止。初期値は game.go の initialSpd で設定している。

var actorColors = [partySize + 1]color.RGBA{
	{80, 220, 230, 255},
	{120, 200, 235, 255},
	{200, 150, 230, 255},
	{230, 130, 220, 255},
	{230, 90, 180, 255},
}

var actorLabels = [partySize + 1]string{"1", "2", "3", "4", "敵"}

var commandDescriptions = [4]string{
	"敵に物理ダメージを与える",
	"MPを消費して特殊な技を使う",
	"ATBを止めて味方との連携を狙う",
	"戦闘から離脱する",
}

var skillDescriptions = [3]string{
	"対象のHPを回復する",
	"通常より大きなダメージを与える",
	"ゲージレベル5で巻き戻し",
}

type BattleScene struct {
	game *Game

	preBattlePlayerHP [partySize]int
	preBattlePlayerMP [partySize]int

	enemyName          string
	enemyType          string
	enemyLv            int
	enemyHP            int
	enemyMaxHP         int
	enemyMP            int
	enemyMaxMP         int
	enemyPhysAtk       int
	enemyMagicAtk      int
	enemySpeed         float64
	enemyExp           int
	enemyDef           int
	enemyMagicDef      int
	enemyElement       Element
	enemyElementResist [elementalTypeCount]int

	// 回避時、スプライトを右にずらすための演出用オフセット
	evadeOffsetX [partySize]float64

	battlePhase  int
	isWon        bool
	commandIndex int
	skillIndex   int
	activePlayer int

	atbGauge        [partySize + 1]float64
	waitStance      [partySize]bool
	waitOrder       []int
	waitCancelOrder []int
	waitCancelHold  [partySize]float64
	deadWaitStuck   [partySize]bool
	waitingActor    int
	readyQueue      []int

	rewindActive             bool
	rewindTimer              float64
	rewindUsed               bool
	rewindExtraTurnAvailable [partySize]bool
	lastEnemyAttackTarget    int
	lastEnemyAttackPrevHP    int
	lastEnemyAttackDamage    int
	pendingDamage2           int
	pendingDamage2Scheduled  bool

	originMap       string
	originX         float64
	originY         float64
	originDir       int
	returnToText    bool
	postTextMessage string
	battleLog       string
	gameOverIdx     int

	fleeSucceeded bool
	enemyImage    *ebiten.Image

	playerPose           [partySize]int
	playerAnimTimer      [partySize]float64
	activeAttacker       int
	attackPhaseTimer     float64
	hitStopTimer         float64
	enemyActionWaitTimer float64

	damagePops []DamagePop

	flashAlpha float64

	resultSubPhase   int
	resultAnimTimer  float64
	drawPlayerLv     [partySize]int
	drawPlayerEXP    [partySize]int
	drawPlayerMaxEXP [partySize]int

	earnedGold  int
	earnedItems []EarnedItemEntry

	resultFadeAlpha float64

	clearDialogs   []EventCommand
	clearDialogIdx int

	isLevelUp [partySize]bool

	pendingDamage      int
	pendingDamageX     float64
	pendingDamageY     float64
	pendingDamageShake float64

	damageActive bool
	damageValue  int
	damageTimer  float64
	damageX      float64
	damageY      float64

	battleLogTimer  float64
	enemyDeathTimer float64
	enemyDeathPhase int
	enemyAlpha      float64
	deathParticles  []DeathParticle

	targetIndex  int
	pendingSkill int

	// ── アイテム使用（バトル中）関連 ──
	itemIndex       int    // アイテム一覧でのカーソル位置
	pendingItemID   string // 対象選択中に使用するアイテムID（""=未選択）
	itemTargetIndex int    // アイテムの対象選択カーソル（0〜3=個別、partySize=全体）
	enemyDrops      []ItemDrop

	skillLevelCursors [partySize][8]int
	healTargetIndex   int

	skillMenuOpenTimer float64

	readySlideX      [partySize]float64
	returnDelayTimer [partySize]float64
	enemyX           float64
	shakeX           float64
	shakeY           float64
	shakeTimer       float64
	shakeMaxDur      float64
	shakePower       float64
	shakeType        int

	partyScreenX [partySize]float64
	partyScreenY [partySize]float64

	introActive      bool
	introOffsetX     float64
	introProgress    float64
	introVertOffsetY float64
	introCharOffsetX float64
	introPhase       int
	introPhaseTimer  float64

	expStartEXP [partySize]int

	// levelUpPauseTimer は、EXPゲージが右端まで到達してから
	// レベルアップ処理（Lv加算・ゲージリセット）を行うまでの一時停止時間。
	// >0の間はゲージを満タンのまま止めておき、0になったら実際にレベルを上げる。
	levelUpPauseTimer [partySize]float64

	drawPlayerEXPF [partySize]float64

	gaugeStage          int
	gaugePoint          int
	gaugeColorAnimTimer float64 // ← 追加：MAX時のグラデーション用経過時間

	enemySP       int
	PlayerDebuffs [partySize][]Debuff
	EnemyDebuffs  []Debuff

	selectedSkillTarget SkillTarget

	lastCommandIndex [partySize]int
	lastSkillIndex   [partySize]int
	lastSkillLevel   [partySize][8]int

	// ── 新スプライトシート対応（追加分） ──
	attackAnimType       int
	chargeApproachOffset float64
	skillGlowLevel       int

	// ── 回復アニメーション管理 ──
	healingAnimTimer [partySize]float64
	healingCaster    int
}

type DamagePop struct {
	Value  int
	X      float64
	Y      float64
	Vy     float64
	Timer  float64
	IsHeal bool
}

type DeathParticle struct {
	X, Y   float64
	Vx, Vy float64
	Life   float64
	Size   float64
}

func NewBattleScene(game *Game, originMap string, originX, originY float64, originDir int, evType string, specificEnemyName string) *BattleScene {
	var chosen EnemyStats

	isBoss := strings.HasPrefix(evType, "boss_")

	if isBoss {
		if bossData, exists := BossDatabase[evType]; exists {
			chosen = bossData
		} else {
			chosen = EnemyStats{Name: "未知の強敵", Lv: 10, Exp: 100, HP: 200, MP: 30, PhysAtk: 18, MagicAtk: 14, PhysDef: 10, MagicDef: 10, Spd: 20, SP: 100}
		}
	} else if specificEnemyName != "" {
		found := false
		for _, enemy := range EnemyDatabase {
			if enemy.Name == specificEnemyName {
				chosen = enemy
				found = true
				break
			}
		}
		if !found {
			chosen = EnemyDatabase[rand.Intn(len(EnemyDatabase))]
		}
	} else {
		chosen = EnemyDatabase[rand.Intn(len(EnemyDatabase))]
	}

	s := &BattleScene{
		game: game,

		enemyName:          chosen.Name,
		enemyType:          evType,
		enemyLv:            chosen.Lv,
		enemyHP:            chosen.HP,
		enemyMaxHP:         chosen.HP,
		enemyMP:            chosen.MP,
		enemyMaxMP:         chosen.MP,
		enemyPhysAtk:       chosen.PhysAtk,
		enemyMagicAtk:      chosen.MagicAtk,
		enemySpeed:         float64(chosen.Spd),
		enemyExp:           chosen.Exp,
		enemySP:            chosen.SP,
		enemyDef:           chosen.PhysDef,
		enemyMagicDef:      chosen.MagicDef,
		enemyElement:       chosen.Element,
		enemyElementResist: chosen.ElementResist,
		enemyDrops:         chosen.Drops,
		battlePhase:        phaseATB,
		waitingActor:       -1,
		activeAttacker:     -1,
		originMap:          originMap,
		originX:            originX,
		originY:            originY,
		originDir:          originDir,
		returnToText:       isBoss,
		postTextMessage:    "",
		gameOverIdx:        0,

		skillMenuOpenTimer: 0.0,
		readySlideX:        [partySize]float64{},

		enemyX:           280.0,
		shakeX:           0.0,
		shakeY:           0.0,
		shakeTimer:       0.0,
		shakeMaxDur:      0.0,
		shakePower:       0.0,
		shakeType:        0,
		introActive:      true,
		introOffsetX:     introStartOffsetX,
		introVertOffsetY: introStartOffsetVertY,
		introCharOffsetX: introStartOffsetX,
		introPhase:       0,
		introPhaseTimer:  0.0,

		gaugeStage:    0,
		gaugePoint:    0,
		healingCaster: -1,
	}

	for i := 0; i < partySize; i++ {
		s.preBattlePlayerHP[i] = game.PlayerHP[i]
		s.preBattlePlayerMP[i] = game.PlayerMP[i]
	}

	for i := 0; i < partySize; i++ {
		s.atbGauge[i] = 40 + float64(rand.Intn(35))
	}
	s.atbGauge[enemyID] = float64(rand.Intn(20))

	s.enemyImage = game.GetEnemyImage(chosen.Name)

	if isBoss {
		game.Audio.PlayBGM(bgmBattleBoss)
	} else {
		game.Audio.PlayBGM(bgmBattleNormal)
	}

	return s
}

var gaugeCumThresholds = [gaugeMaxStage - 1]int{8, 16, 24, 32}

const gaugePoolMax = 32 + gaugePointCap
