package main

import (
	"image/color"
	"math/rand"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	partySize  = 4
	maxEnemies = 4
	atbMax     = 100.0
)

// lastBossEventType はラスボス(4体目のボス)のイベント種別。ボス4体のうち
// この1体だけ通常のボス戦BGMではなくラスボス専用BGMを流す。
const lastBossEventType = "boss_4"

const battleLogDuration = 0.7
const gameOverMessageDuration = 1.5

const skillActionLogDuration = 1.0

const enemyActionDuration = 1.0

const enemyWindupDuration = 0.4

const (
	evadeDodgeShiftX      = 26.0
	evadeDodgeReturnSpeed = 140.0
)

type hitTier int

const (
	hitTierNone hitTier = iota
	hitTierWeak
	hitTierStrong
	hitTierSynergy
)

const (
	hitStopWeak   = 0.08
	hitStopStrong = 0.15
)

const hitFlashDuration = 0.25

const spriteFlashDuration = 0.12

type pendingEnemyHit struct {
	target int
	dmg    int
	evaded bool
}

type pendingPlayerHit struct {
	slot int
	dmg  int
	crit bool
}

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
	statusBlockGap = 42.0
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

const (
	spriteFrameW = 64
	spriteFrameH = 96
)

const (
	poseWalk           = 11
	poseAttack         = 12
	poseReadyGlow      = 13
	poseChargeApproach = 14
	poseChargeAttack   = 15
	poseFireCast       = 16
	poseFireLoop       = 17
	poseHealCast       = 18
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
	poseIdle:           2,
	poseDamage:         2,
	poseWalk:           2,
	poseAttack:         7,
	poseReady:          1,
	poseReadyGlow:      10,
	poseChargeApproach: 4,
	poseChargeAttack:   11,
	poseFireCast:       16,
	poseFireLoop:       15,
	poseHealCast:       14,
}

var poseLoopFrameDur = map[int]float64{
	poseIdle:     0.25,
	poseWalk:     0.12,
	poseHealCast: 0.08,
	poseFireLoop: 0.05,
}

const (
	animNormal = iota
	animCharge
	animFireMagic
)

var glowLevelRange = [4][2]int{
	{0, 1},
	{1, 1},
	{2, 1},
	{3, 7},
}

const (
	resSubStillWait    = 0
	resSubWinPose      = 1
	resSubResultFadeIn = 2
	resSubBarAnimate   = 3
	resSubDoneWait     = 4
)

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
	{90, 210, 120, 255},
	{235, 205, 60, 255},
	{235, 140, 50, 255},
	{225, 70, 70, 255},
	{175, 90, 225, 255},
}

const (
	gaugeTriX = 15.0
	gaugeTriY = 100.0
	gaugeTriW = 90.0
	gaugeTriH = 60.0
)

const (
	gaugeImgBorder      = 2.0
	gaugeFillPerPoint   = 2.0
	gaugePointsPerStage = 8
	gaugeDividerHeight  = 2.0
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

	resultNameOffsetX  = 0.0
	resultLevelOffsetX = 250.0
	resultNameOffsetY  = -30.0

	resultExpLabelOffsetX = 4.0
	resultExpLabelOffsetY = -15.0

	resultExpTextOffsetY = -15.0

	resultLevelUpOffsetX = 4.0
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

type EnemyUnit struct {
	Name              string
	Lv                int
	HP, MaxHP         int
	MP, MaxMP         int
	PhysAtk, MagicAtk int
	Def, MagicDef     int
	Speed             float64
	Exp, SP           int
	Element           Element
	ElementResist     [elementalTypeCount]int
	Skills            []EnemySkill
	Drops             []ItemDrop
	Debuffs           []Debuff

	Image *ebiten.Image

	Alpha         float64
	DeathTimer    float64
	DeathPhase    int
	HitFlashTimer float64
}

type BattleScene struct {
	game *Game

	preBattlePlayerHP [partySize]int
	preBattlePlayerMP [partySize]int

	enemyType       string
	enemyNames      []string
	enemies         []EnemyUnit
	actingEnemySlot int
	bgmPath         string

	evadeOffsetX [partySize]float64

	battlePhase       int
	isWon             bool
	commandIndex      int
	commandTapArmed   bool
	rewindButtonArmed bool
	itemButtonArmed   bool
	skillIndex        int
	activePlayer      int

	atbGauge        [partySize + maxEnemies]float64
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

	playerPose           [partySize]int
	playerAnimTimer      [partySize]float64
	activeAttacker       int
	attackPhaseTimer     float64
	hitStopTimer         float64
	enemyActionWaitTimer float64
	enemyIsActing        bool
	enemyWindupTimer     float64

	enemyHitStopTimer float64
	playerFlashTimer  [partySize]float64

	pendingEnemyHits         []pendingEnemyHit
	pendingEnemyHitTier      hitTier
	pendingEnemySkillName    string
	pendingEnemySkillEffects []SkillEffect
	pendingEnemyIsAll        bool

	damagePops []DamagePop

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

	pendingPlayerHits  []pendingPlayerHit
	pendingPlayerHits2 []pendingPlayerHit
	lastRollWasCrit    bool

	damageActive bool
	damageValue  int
	damageTimer  float64
	damageX      float64
	damageY      float64

	battleLogTimer float64
	deathParticles []DeathParticle

	targetIndex  int
	pendingSkill int

	itemIndex       int
	pendingItemID   string
	itemTargetIndex int

	skillLevelCursors [partySize][8]int
	healTargetIndex   int

	skillMenuOpenTimer float64

	readySlideX      [partySize]float64
	returnDelayTimer [partySize]float64

	tlFrozenX     [partySize]float64
	tlFrozenY     [partySize]float64
	tlFrozenSize  [partySize]float64
	tlFrozenLarge [partySize]bool

	shakeX      float64
	shakeY      float64
	shakeTimer  float64
	shakeMaxDur float64
	shakePower  float64
	shakeType   int

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

	levelUpPauseTimer [partySize]float64

	drawPlayerEXPF [partySize]float64

	gaugeStage          int
	gaugePoint          int
	gaugeColorAnimTimer float64

	PlayerDebuffs [partySize][]Debuff
	PlayerBuffs   [partySize][]Buff

	selectedSkillTarget SkillTarget

	lastCommandIndex [partySize]int
	lastSkillIndex   [partySize]int
	lastSkillLevel   [partySize][8]int

	attackAnimType       int
	chargeApproachOffset float64
	skillGlowLevel       int

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
	IsCrit bool
	IsMiss bool
}

type DeathParticle struct {
	X, Y   float64
	Vx, Vy float64
	Life   float64
	Size   float64
}

func resolveEnemyStats(name string) EnemyStats {
	for _, enemy := range EnemyDatabase {
		if enemy.Name == name {
			return enemy
		}
	}
	return EnemyDatabase[rand.Intn(len(EnemyDatabase))]
}

func newEnemyUnit(game *Game, evType string, chosen EnemyStats) EnemyUnit {
	return EnemyUnit{
		Name:          chosen.Name,
		Lv:            chosen.Lv,
		HP:            chosen.HP,
		MaxHP:         chosen.HP,
		MP:            chosen.MP,
		MaxMP:         chosen.MP,
		PhysAtk:       chosen.PhysAtk,
		MagicAtk:      chosen.MagicAtk,
		Def:           chosen.PhysDef,
		MagicDef:      chosen.MagicDef,
		Speed:         float64(chosen.Spd),
		Exp:           chosen.Exp,
		SP:            chosen.SP,
		Element:       chosen.Element,
		ElementResist: chosen.ElementResist,
		Skills:        chosen.Skills,
		Drops:         chosen.Drops,
		Image:         game.GetEnemyBattleImage(evType, chosen.Name),
		Alpha:         1.0,
	}
}

func NewBattleScene(game *Game, originMap string, originX, originY float64, originDir int, evType string, enemyNames []string) *BattleScene {
	isBoss := strings.HasPrefix(evType, "boss_")

	var enemies []EnemyUnit
	if isBoss {
		var chosen EnemyStats
		if bossData, exists := BossDatabase[evType]; exists {
			chosen = bossData
		} else {
			chosen = EnemyStats{Name: "未知の強敵", Lv: 10, Exp: 100, HP: 200, MP: 30, PhysAtk: 18, MagicAtk: 14, PhysDef: 10, MagicDef: 10, Spd: 20, SP: 100}
		}
		enemies = []EnemyUnit{newEnemyUnit(game, evType, chosen)}
	} else {
		if len(enemyNames) == 0 {
			enemyNames = []string{EnemyDatabase[rand.Intn(len(EnemyDatabase))].Name}
		}
		if len(enemyNames) > maxEnemies {
			enemyNames = enemyNames[:maxEnemies]
		}
		enemies = make([]EnemyUnit, len(enemyNames))
		for i, name := range enemyNames {
			enemies[i] = newEnemyUnit(game, evType, resolveEnemyStats(name))
		}
	}

	s := &BattleScene{
		game: game,

		enemyType:       evType,
		enemyNames:      enemyNames,
		enemies:         enemies,
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
	for i := range s.enemies {
		s.atbGauge[s.enemyActorIndex(i)] = float64(rand.Intn(20))
	}

	if evType == lastBossEventType {
		s.bgmPath = bgmBattleLastBoss
	} else if isBoss {
		s.bgmPath = bgmBattleBoss
	} else {
		s.bgmPath = bgmBattleNormal
	}

	game.Audio.PlaySEByKey("battle_start")

	return s
}

// desiredBGM は戦闘BGMを暗転しきった瞬間にハードカットで鳴らす。フェード
// インさせず一気に切り替わる方が、遭遇の緊張感やボス戦の盛り上がりに合う。
func (s *BattleScene) desiredBGM(transitionDuration float64) (string, float64, bool) {
	return s.bgmPath, 0, true
}

var gaugeCumThresholds = [gaugeMaxStage - 1]int{8, 16, 24, 32}

const gaugePoolMax = 32 + gaugePointCap
