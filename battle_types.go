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

// lastBossEventType is the event type for the last boss (the 4th boss).
// Of the four bosses, only this one plays a dedicated last-boss BGM
// instead of the normal boss battle BGM.
const lastBossEventType = "boss_4"

const battleLogDuration = 0.85
const gameOverMessageDuration = 1.5

const skillActionLogDuration = 1.15

const enemyActionDuration = 1.0

// silentActionLogDuration is the delay before the next action starts after
// an action that shows no log (a normal attack). It's shorter than the
// delay for actions that do show a log.
const silentActionLogDuration = 0.35

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
	// nonLethal hits never take the enemy below 1 HP.
	nonLethal bool
}

type rewindSnapshot struct {
	valid           bool
	playerHP        [partySize]int
	playerBuffs     [partySize][]Buff
	playerDebuffs   [partySize][]Debuff
	partyATB        [partySize]float64
	coverCount      [partySize]int
	counterTimer    [partySize]float64
	counterBonus    [partySize]int
	guardHPPercent  [partySize]int
	guardCount      [partySize]int
	waitStance      [partySize]bool
	waitOrder       []int
	waitCancelOrder []int
	waitCancelHold  [partySize]float64
	deadWaitStuck   [partySize]bool
	playerPose      [partySize]int
	playerAnimTimer [partySize]float64
	playerFlash     [partySize]float64
	evadeOffsetX    [partySize]float64
	readySlideX     [partySize]float64
	returnDelay     [partySize]float64
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
	cmdSkill
	cmdWait
	cmdFlee
)

const (
	trackX         = 20.0
	trackY         = 53.0
	trackH         = 8.0
	iconSize       = 58.0 // must match timeline icon PNG width; atbIconSpacing uses it
	timelineStartX = trackX + 10.0

	logPanelY = 390.0
	logPanelH = 28.0
	cmdPanelX = 600.0
	cmdPanelY = 380.0
	cmdPanelW = 280.0
	cmdPanelH = 120.0
	hintY     = 524.0

	// partyNameBaseX/Y is the party member's name draw position (i=0). Every
	// other status-block element is positioned relative to this point via
	// partyNamePosition(i) in battle_draw_panels.go.
	partyNameBaseX = 25.0
	partyNameBaseY = 432.0

	statusBlockW   = 140.0
	statusBarH     = 5.5
	statusBarSlant = 8.0
	statusBlockGap = 50.0
	statusHPBarY   = 44.0
	statusMPBarY   = 71.0
	// statusValueGapAboveBar is the visible gap between the HP/MP value text
	// (including its outline) and the top edge of the bar.
	statusValueGapAboveBar = 1.0
	statusValueOffsetX     = 10.0

	// statIconGap/OffsetX/OffsetY position the per-stat up/down icons drawn
	// to the right of a party member's name (see drawPartyStatIcons in
	// battle_draw_hud.go); icons are drawn at their native PNG size
	// (assets/images/battle/stat_up.png and stat_down.png), never scaled.
	// statIconOffsetX is a single fixed offset used for every party member
	// (not measured from each member's actual name width); statIconGapX is
	// the separate, independently-tunable gap between consecutive icons.
	statIconGapX    = 0.0
	statIconOffsetX = 50.0
	statIconOffsetY = -5.0

	// statusValueFontSizeLarge/Small size the current/max numbers drawn by
	// drawStatusValue in both the battle HUD and the menu party list, so
	// they stay visually consistent between screens.
	statusValueFontSizeLarge = 20.0
	statusValueFontSizeSmall = 17.0

	// partyNameFontSize sizes the player name drawn on the name plate in
	// both the battle HUD and the menu party list.
	partyNameFontSize = 20.0

	descX        = 30.0
	descY        = 515.0
	descFontSize = 20.0
)

const (
	poseIdle   = 0
	poseReady  = 1
	poseDamage = 6
	poseDead   = 8
)

// spriteFrameW/H は味方1人が画面上で占める枠（足元の位置やタップ判定、
// ダメージ表示の基準）。スプライトシートのマス（battleSpriteCellW/H）を
// battleSpriteScale倍にして、この枠の下端・左右中央にそろえて描く。
const (
	spriteFrameW = 64
	spriteFrameH = 96

	battleSpriteScale = 2
)

// damagePopHeadOffsetRatio positions damage/heal popups a fraction of the
// target's sprite height above its top edge, so the popup lands at roughly
// the same spot on the head regardless of how tall that particular sprite
// (party member or enemy) is.
const damagePopHeadOffsetRatio = 0.08

const partyDamagePopOffsetY = spriteFrameH * damagePopHeadOffsetRatio

// Damage popup lifetime in seconds: fully opaque until damagePopHoldTime,
// then fading out until damagePopLifetime.
const (
	damagePopHoldTime = 0.6
	damagePopLifetime = 1.1
)

const (
	poseWalk      = 11
	poseAttack    = 12
	poseReadyGlow = 13
	// poseSkill は攻撃スキルの専用アニメ（actionRowの行）を再生中。
	poseSkill = 14
	// poseHealCast は回復・補助スキルを使ったアニメ（actionRowの行）を再生中。
	poseHealCast = 18
	// poseRecv は回復・補助スキルを受けたアニメ（recvRowの行）を再生中。
	poseRecv = 19
)

// player_attack_N.png の共通アニメの行。
// この後ろに専用アニメのあるスキルの行が続き、どのスキルがどの行かは
// partySkillSpriteRow（スキル名で引く）にある。
const (
	spriteRowIdle = iota
	spriteRowDamage
	spriteRowWalk
	spriteRowCommand
	spriteRowAttack
	spriteRowBuffRecv
	spriteRowHealRecv
)

// battleSpriteFrameDur は待機・歩きなどくり返すアニメを1コマ表示する秒数。
// 15フレーム（60TPS）ごとに次のコマへ進む。
const battleSpriteFrameDur = 15.0 / 60.0

// battleActionFrameDur は通常攻撃・スキル・回復を受けるなど、1回だけ流す
// アニメを1コマ表示する秒数（3フレーム＝20コマ/秒）。アニメの長さは
// コマ数×この秒数になり、攻撃はアニメが終わった時にダメージが出る。
const battleActionFrameDur = 3.0 / 60.0

const (
	animNormal = iota
	animSkill
)

// glowLevelRange はスキル選択中のコマンド選択アニメで、スキルレベルごとに
// 使うコマの{開始, コマ数}。
var glowLevelRange = [4][2]int{
	{0, 1},
	{1, 1},
	{2, 1},
	{3, 6},
}

const (
	resSubStillWait    = 0
	resSubWinWait      = 1
	resSubResultFadeIn = 2
	resSubBarAnimate   = 3
	resSubDoneWait     = 4
)

const levelUpPauseDuration = 0.35

const (
	// waitIconFirstGap はゴール画像の下端と1人目の待機アイコンの
	// すき間、waitIconStackGap は待機アイコン同士のすき間（どちらもpx）。
	waitIconFirstGap      = 0.0
	waitIconStackGap      = 0.0
	introStartOffsetX     = 960.0
	introStartOffsetVertY = 540.0

	introHorizDecay = 0.0000001
	introVertDecay  = 0.0000001
	introCharDecay  = 0.0000001

	introGoalWait        = 0.15
	introATBWait         = 0.2
	introHorizToVertWait = 0.01
)

const (
	gaugeMaxStage = 5
	gaugePointCap = 8
)

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

	resultTitleX        = 16.0
	resultTitleY        = 6.0
	resultTitleFontSize = 40.0

	// EXP and SP share one row to leave room for the party rows.
	resultExpLabelX   = 32.0
	resultExpValueX   = 90.0
	resultExpY        = 54.0
	resultExpFontSize = 22.0

	resultSpLabelX   = 250.0
	resultSpValueX   = 296.0
	resultSpY        = 54.0
	resultSpFontSize = 22.0

	resultDividerX = 12.0
	// Both divider lines sit resultDividerGapY below the bottom of the text
	// line above them (its top Y plus its font size).
	resultDividerGapY = 2.0
	resultDividerY    = resultExpY + resultExpFontSize + resultDividerGapY
	resultDividerW    = 360.0
	resultDividerH    = 1.0

	resultItemsHeaderX = 24.0
	// The items section starts resultItemsSectionGapY below where a fifth
	// party row would, so the last row's LEVEL UP! line has room.
	resultItemsSectionGapY    = 8.0
	resultItemsHeaderY        = resultRowStartY + partySize*resultRowGap + resultItemsSectionGapY
	resultItemsHeaderFontSize = 22.0

	resultItemsDividerX = 12.0
	resultItemsDividerY = resultItemsHeaderY + resultItemsHeaderFontSize + resultDividerGapY
	resultItemsDividerW = 360.0
	resultItemsDividerH = 1.0

	// Earned items are listed in two columns (left to right, then down) so
	// several of them still fit above the hint line.
	resultItemsNameX     = 24.0
	resultItemsCountX    = 184.0
	resultItemsColumnGap = 176.0
	resultItemsColumns   = 2
	resultItemsStartY    = resultItemsDividerY + 6.0
	resultItemsRowGap    = 22.0
	resultItemsFontSize  = 16.0

	// Each party row, top to bottom: name / Lv, EXP label / numbers, the
	// EXP bar, then LEVEL UP! / スキル解放！. Offsets are from the row top.
	resultRowStartY = 86.0
	resultRowGap    = 78.0

	// The bar (and the name / Lv / EXP text aligned to it) is centered in
	// the panel.
	resultBarStartX  = (float64(gameWidth)*resultPanelWidthRatio - resultBarWAbs) / 2
	resultBarWAbs    = 320.0
	resultBarH       = 8.0
	resultBarOffsetY = 44.0

	resultExpLabelOffsetX = 4.0
	// resultExpTextGapY is the space between the EXP line's bottom and the bar.
	resultExpTextGapY = 2.0

	resultLevelUpOffsetX = 4.0
	resultLevelUpOffsetY = 55.0

	resultNameFontSize        = 22.0
	resultExpCurFontSize      = 18.0
	resultExpMaxFontSize      = 16.0
	resultExpLabelFontSize    = 18.0
	resultLevelUpFontSize     = 18.0
	resultSkillUnlockFontSize = 18.0

	resultHintX        = 24.0
	resultHintYFromBtm = 24.0
	resultHintFontSize = 18.0
)

var resultBarFillColor = color.RGBA{255, 200, 130, 255}
var resultBarBgColor = color.RGBA{30, 30, 40, 255}
var resultPanelBgColor = color.RGBA{0, 0, 0, 200}
var resultLevelUpColor = color.RGBA{255, 255, 100, 255}
var resultSkillUnlockColor = color.RGBA{150, 220, 255, 255}
var resultItemsDividerColor = color.RGBA{255, 255, 255, 255}

var commandDescriptions = [4]string{
	"敵に物理ダメージを与えます",
	"MPを消費して特殊な技を使います",
	"味方全員と連携して大ダメージを与えます",
	"戦闘から離脱します",
}

const (
	itemButtonDescription = "回復アイテムなどを使います"
	rewindDescription     = "敵の直前の攻撃を無効化し、行動回数が一定時間2倍になります"
)

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
	// Scanned is set once みやぶる reveals this enemy's HP and weaknesses.
	Scanned bool

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

	// minActorSpd/maxActorSpd are the Spd range across every actor at
	// battle start, used to scale each actor's return-position bonus.
	minActorSpd, maxActorSpd float64

	atbGauge        [partySize + maxEnemies]float64
	waitStance      [partySize]bool
	waitOrder       []int
	waitCancelOrder []int
	waitCancelHold  [partySize]float64
	deadWaitStuck   [partySize]bool
	waitingActor    int

	rewindActive             bool
	rewindTimer              float64
	rewindUsed               bool
	rewindExtraTurnAvailable [partySize]bool
	rewindSnapshot           rewindSnapshot
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

	pendingEnemyHits          []pendingEnemyHit
	pendingEnemyHitTier       hitTier
	pendingEnemySkillName     string
	pendingEnemySkillEffects  []SkillEffect
	pendingEnemyIsAll         bool
	enemyActionReturnPosition float64

	damagePops []DamagePop

	resultSubPhase   int
	resultAnimTimer  float64
	drawPlayerLv     [partySize]int
	drawPlayerEXP    [partySize]int
	drawPlayerMaxEXP [partySize]int

	earnedItems []EarnedItemEntry

	resultFadeAlpha float64

	isLevelUp        [partySize]bool
	resultStartLevel [partySize]int

	pendingPlayerHits  []pendingPlayerHit
	pendingPlayerHits2 []pendingPlayerHit
	lastRollWasCrit    bool

	battleLogTimer float64
	deathParticles []DeathParticle

	targetIndex    int
	pendingSkill   int
	pendingSynergy bool

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

	// gaugeUnlocked is whether the gauge system (gauge, attack bonus,
	// rewind) is active this battle. It unlocks together with the gauge
	// tutorial, i.e. once boss 1 has been defeated.
	gaugeUnlocked       bool
	gaugeStage          int
	gaugePoint          int
	gaugeColorAnimTimer float64

	PlayerDebuffs [partySize][]Debuff
	PlayerBuffs   [partySize][]Buff

	// coverCount is how many more single-target enemy attacks on allies
	// this party member will take in their place (かばう).
	coverCount [partySize]int
	// counterTimer is the remaining counter-stance time (カウンター) and
	// counterBonus its extra damage percent.
	counterTimer [partySize]float64
	counterBonus [partySize]int
	// guardHPPercent/guardCount are 守りの祈り's threshold and remaining
	// charges on each party member.
	guardHPPercent [partySize]int
	guardCount     [partySize]int

	// forceCrit makes the next rollDamage a guaranteed critical hit.
	forceCrit bool
	// pendingDrain* describe the MP drain (ドレインアックス) of the player
	// attack whose hits are pending; pendingDrainDivisor 0 means none.
	pendingDrainActor   int
	pendingDrainDivisor int
	pendingDrainMax     int

	// debugStatIconTest tracks whether applyDebugCheats' Shift+B test
	// buffs/debuffs are currently applied, so pressing it again clears them.
	debugStatIconTest bool

	lastCommandIndex [partySize]int
	lastSkillIndex   [partySize]int
	lastSkillLevel   [partySize][8]int

	attackAnimType int
	skillGlowLevel int

	// actionRow/actionFrames はposeSkill・poseHealCastで再生する行とコマ数。
	actionRow    [partySize]int
	actionFrames [partySize]int
	// recvRow/recvTimer はposeRecvで再生する行と、終わるまでの残り秒数。
	recvRow   [partySize]int
	recvTimer [partySize]float64

	healingAnimTimer [partySize]float64
	healingCaster    int

	tutorialActive     bool
	tutorialKind       int
	tutorialPage       int
	tutorialOverlayImg *ebiten.Image
	tutorialSceneImg   *ebiten.Image

	// skillSubMenuImg is the scratch buffer drawSkillSubMenuEnlarged (used
	// only by the skill-upgrade tutorial mockup) renders into before
	// compositing it back onto screen scaled up by battleSkillPanelScale.
	skillSubMenuImg *ebiten.Image
}

type DamagePop struct {
	Value  int
	X      float64
	Y      float64
	Vy     float64
	Timer  float64
	IsHeal bool
	IsMP   bool // MP recovery; drawn blue, combined with IsHeal for motion
	IsCrit bool
	IsMiss bool
	// Label, when set, is drawn instead of the number (e.g. "無敵").
	Label string
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
		Skills:        enemySkillList(chosen.SkillIDs),
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
		gaugeUnlocked: gaugeSystemUnlocked(game),
		healingCaster: -1,

		tutorialKind: nextBattleTutorialKind(game),
		tutorialPage: 0,
	}
	s.tutorialActive = s.tutorialKind != battleTutorialKindNone
	switch s.tutorialKind {
	case battleTutorialKindBasics:
		game.SeenBattleTutorial = true
	case battleTutorialKindGauge:
		game.SeenGaugeTutorial = true
	}

	for i := range partySize {
		s.preBattlePlayerHP[i] = game.PlayerHP[i]
		s.preBattlePlayerMP[i] = game.PlayerMP[i]
	}

	s.initTimelinePositions()

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

// desiredBGM plays the battle BGM with a hard cut the instant the screen
// finishes fading to black. Cutting straight in without a fade-in better
// matches the tension of the encounter and the excitement of a boss fight.
func (s *BattleScene) desiredBGM(transitionDuration float64) (string, float64, bool) {
	return s.bgmPath, 0, true
}

var gaugeCumThresholds = [gaugeMaxStage - 1]int{8, 16, 24, 32}

const gaugePoolMax = 32 + gaugePointCap
