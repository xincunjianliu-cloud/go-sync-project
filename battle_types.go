package main

// battle_types.go: バトル関連の型定義・定数・BattleSceneの初期化

import (
	"image/color"
	"math/rand"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

type EnemyType struct {
	Name     string
	MaxHP    int
	DmgMin   int
	DmgRange int
	Speed    float64
	Exp      int
	SP       int
	Def      int
	MagicDef int
}

var EnemyDatabase = []EnemyType{
	{Name: "フリーザ", MaxHP: 160, DmgMin: 12, DmgRange: 6, Speed: 22, Exp: 45, SP: 2, Def: 8, MagicDef: 8},
	{Name: "セル", MaxHP: 200, DmgMin: 15, DmgRange: 5, Speed: 26, Exp: 60, SP: 3, Def: 10, MagicDef: 9},
	{Name: "魔人ブウ", MaxHP: 350, DmgMin: 10, DmgRange: 8, Speed: 16, Exp: 90, SP: 4, Def: 12, MagicDef: 14},
}

var BossDatabase = map[string]EnemyType{
	"boss_1": {Name: BossNames[0], MaxHP: 200, DmgMin: 12, DmgRange: 6, Speed: 18, Exp: 80, SP: 300, Def: 12, MagicDef: 10},
	"boss_2": {Name: BossNames[1], MaxHP: 250, DmgMin: 14, DmgRange: 6, Speed: 20, Exp: 100, SP: 400, Def: 14, MagicDef: 12},
	"boss_3": {Name: BossNames[2], MaxHP: 300, DmgMin: 16, DmgRange: 7, Speed: 22, Exp: 130, SP: 500, Def: 16, MagicDef: 14},
	"boss_4": {Name: BossNames[3], MaxHP: 600, DmgMin: 26, DmgRange: 10, Speed: 30, Exp: 500, SP: 1919, Def: 26, MagicDef: 24},
}

const (
	partySize = 4
	enemyID   = 4
	atbMax    = 100.0
)

const battleLogDuration = 0.7

const (
	phaseATB = iota
	phasePlayerMenu
	phaseSkillMenu
	phaseTargetSelect
	phaseHealSelect
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
	trackX   = 20.0
	trackY   = 53.0
	trackH   = 8.0
	iconSize = 40.0

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
var gaugeStageAtkBonus = [gaugeMaxStage]int{0, 3, 5, 7, 10}

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

	resultTitleX        = 24.0
	resultTitleY        = 20.0
	resultTitleFontSize = 40.0

	resultExpLabelX   = 40.0
	resultExpValueX   = 250.0
	resultExpY        = 75.0
	resultExpFontSize = 20.0

	resultSpLabelX   = 40.0
	resultSpValueX   = 250.0
	resultSpY        = 100.0
	resultSpFontSize = 20.0

	resultDividerX = 20.0
	resultDividerY = 125.0
	resultDividerW = 350.0
	resultDividerH = 1.0

	resultBarStartX = 45.0
	resultBarStartY = 180.0
	resultBarRowGap = 95.0
	resultBarWAbs   = 280.0
	resultBarH      = 10.0

	resultNameX        = 49.0
	resultLevelX       = 290.0
	resultNameOffsetY  = -35.0
	resultNameFontSize = 15.0

	resultExpCurFontSize = 18.0
	resultExpMaxFontSize = 13.0
	resultExpTextOffsetY = -20.0

	resultExpLabelOffsetX  = 4.0
	resultExpLabelOffsetY  = -15.0
	resultExpLabelFontSize = 16.0

	resultLevelUpOffsetX  = 4.0
	resultLevelUpOffsetY  = 12.0
	resultLevelUpFontSize = 15.0

	resultHintX        = 24.0
	resultHintYFromBtm = 20.0
	resultHintFontSize = 15.0
)

var resultBarFillColor = color.RGBA{255, 200, 130, 255}
var resultBarBgColor = color.RGBA{30, 30, 40, 255}

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

	enemyName     string
	enemyType     string
	enemyHP       int
	enemyMaxHP    int
	enemyDmgMin   int
	enemyDmgRange int
	enemySpeed    float64
	enemyExp      int
	enemyDef      int
	enemyMagicDef int

	battlePhase  int
	isWon        bool
	commandIndex int
	skillIndex   int
	activePlayer int

	atbGauge     [partySize + 1]float64
	waitStance   [partySize]bool
	waitOrder    []int
	waitingActor int
	readyQueue   []int

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

	enemyImage *ebiten.Image

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
	earnedItems []string

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

	drawPlayerEXPF [partySize]float64

	gaugeStage int
	gaugePoint int

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
	var chosen EnemyType

	isBoss := strings.HasPrefix(evType, "boss_")

	if isBoss {
		if bossData, exists := BossDatabase[evType]; exists {
			chosen = bossData
		} else {
			chosen = EnemyType{Name: "未知の強敵", MaxHP: 200, DmgMin: 10, DmgRange: 5, Speed: 20, Exp: 100, SP: 100, Def: 10, MagicDef: 10}
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

		enemyName:       chosen.Name,
		enemyType:       evType,
		enemyHP:         chosen.MaxHP,
		enemyMaxHP:      chosen.MaxHP,
		enemyDmgMin:     chosen.DmgMin,
		enemyDmgRange:   chosen.DmgRange,
		enemySpeed:      chosen.Speed,
		enemyExp:        chosen.Exp,
		enemySP:         chosen.SP,
		enemyDef:        chosen.Def,
		enemyMagicDef:   chosen.MagicDef,
		battlePhase:     phaseATB,
		waitingActor:    -1,
		activeAttacker:  -1,
		originMap:       originMap,
		originX:         originX,
		originY:         originY,
		originDir:       originDir,
		returnToText:    isBoss,
		postTextMessage: "",
		gameOverIdx:     0,

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
