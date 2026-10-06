package main

import (
	"math"
	"math/rand"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func isLowHP(hp, maxHP int) bool {
	if maxHP <= 0 {
		return false
	}
	return float64(hp)/float64(maxHP) <= 0.25
}

// skillSpriteAnim は味方pのskillIdx番目のスキルで再生するスプライトシートの行と
// コマ数、回復の「自分が受ける」部分が始まるコマ（なければ0）を返す。
// 表はスキル名で引く。表にないスキルは通常攻撃のアニメにする。
func (s *BattleScene) skillSpriteAnim(p, skillIdx int) (row, frames, healSelfFrom int) {
	row = spriteRowAttack
	skills := s.game.CharacterSkills(p)
	if skillIdx >= 0 && skillIdx < len(skills) {
		name := skills[skillIdx].Name
		if r, ok := partySkillSpriteRow[p][name]; ok {
			row = r
		}
		healSelfFrom = partySkillHealSelfFrame[p][name]
	}
	return row, partySpriteRowFrames[p][row], healSelfFrom
}

// startCast は回復・補助スキルを使ったアニメ（スプライトシートのrow行の
// 先頭framesコマ）をactorに再生させる。
func (s *BattleScene) startCast(actor, row, frames int) {
	if actor < 0 || actor >= partySize {
		return
	}
	s.healingCaster = actor
	s.playerPose[actor] = poseHealCast
	s.playerAnimTimer[actor] = 0
	s.actionRow[actor] = row
	s.actionFrames[actor] = frames
	s.healingAnimTimer[actor] = actionAnimDuration(frames)
}

// startRecv は回復・補助スキルを受けたアニメ（スプライトシートのrow行）を
// targetに再生させる。
func (s *BattleScene) startRecv(target, row int) {
	if target < 0 || target >= partySize || s.game.PlayerHP[target] <= 0 {
		return
	}
	s.recvRow[target] = row
	s.recvTimer[target] = actionAnimDuration(partySpriteRowFrames[target][row])
	s.playerPose[target] = poseRecv
	s.playerAnimTimer[target] = 0
}

func (s *BattleScene) endCast(actor int) {
	if actor < 0 || actor >= partySize {
		return
	}
	if s.healingCaster == actor {
		s.healingCaster = -1
	}
	s.playerPose[actor] = poseIdle
	s.playerAnimTimer[actor] = 0
	s.healingAnimTimer[actor] = 0
}

func (s *BattleScene) tickATB(dt float64) {
	for i := 0; i < partySize; i++ {
		if s.waitStance[i] || s.waitCancelHold[i] > 0 || s.atbGauge[i] >= atbMax || s.game.PlayerHP[i] <= 0 {
			continue
		}
		speed := atbBaseSpeed
		if s.rewindActive {
			speed *= 1.5
		}
		s.atbGauge[i] += speed * dt
		if s.atbGauge[i] > atbMax {
			s.atbGauge[i] = atbMax
		}
	}
	for i := range s.enemies {
		if s.enemies[i].HP <= 0 {
			continue
		}
		actor := s.enemyActorIndex(i)
		if s.atbGauge[actor] >= atbMax {
			continue
		}
		s.atbGauge[actor] += atbBaseSpeed * dt
		if s.atbGauge[actor] > atbMax {
			s.atbGauge[actor] = atbMax
		}
	}
}

func (s *BattleScene) updateRewind(dt float64) {
	if !s.rewindActive {
		return
	}
	s.rewindTimer -= dt
	if s.rewindTimer <= 0 {
		s.rewindTimer = 0
		s.rewindActive = false
		for i := 0; i < partySize; i++ {
			s.rewindExtraTurnAvailable[i] = false
		}
		s.battleLog = "巻き戻し効果が切れた"
		s.battleLogTimer = battleLogDuration
	}
}

func (s *BattleScene) fleeSuccessRate() int {
	if strings.HasPrefix(s.enemyType, "boss_") {
		return 0
	}
	alive := s.aliveEnemyIndices()
	if len(alive) == 0 {
		return 100
	}
	totalLv := 0
	for _, i := range alive {
		totalLv += s.enemies[i].Lv
	}
	avgLv := float64(totalLv) / float64(len(alive))
	rate := 70 + int((float64(s.game.PlayerLv[0])-avgLv)*10)
	if rate < 10 {
		return 10
	}
	if rate > 100 {
		return 100
	}
	return rate
}

const atbBaseSpeed = 10.0

const (
	// Return-to-timeline position (0-100) after each non-skill action.
	// Each of these is individually configurable; all default to 5 for now.
	normalAttackReturnPosition      = 5.0
	enemyNormalAttackReturnPosition = 5.0
	itemUseReturnPosition           = 5.0
	rewindReturnPosition            = 5.0
	synergyReturnPosition           = 5.0
	waitCancelReturnPosition        = 5.0
	fleeFailReturnPosition          = 5.0
)

// normalAttackGaugePoint is how many synergy gauge points a normal attack
// adds. Skill uses instead add their own SkillLevelData.GaugePoint.
const normalAttackGaugePoint = 1

// spdReturnBonusMax is the extra return position (0-100 units) the fastest
// actor in a battle gets on top of every return position; the slowest gets
// 0 and everyone else is interpolated linearly by Spd. Since every gauge
// fills at the same atbBaseSpeed, this is what makes Spd matter each turn.
const spdReturnBonusMax = 10.0

// spdStartBonusMax is the same idea for the battle-start position: the
// fastest actor starts this far along, the slowest at 0. It is much wider
// than spdReturnBonusMax so Spd gaps stay visible at the start instead of
// being swallowed by the atbIconSpacing push-apart (~7 units per icon).
const spdStartBonusMax = 40.0

// atbIconSpacing is the minimum gap, in the same 0-100 units as atbGauge,
// between two icons on the timeline so they don't overlap.
func atbIconSpacing() float64 {
	trackLen := goalAnchorLayout.X - timelineStartX
	if trackLen <= 0 {
		return 0
	}
	return iconSize / trackLen * atbMax
}

func clampAtbPosition(pos float64) float64 {
	if pos < 0 {
		return 0
	}
	if pos > atbMax {
		return atbMax
	}
	return pos
}

// spdBonus maps spd linearly onto 0..bonusMax across [minSpd, maxSpd].
func spdBonus(spd, minSpd, maxSpd, bonusMax float64) float64 {
	if maxSpd <= minSpd {
		return 0
	}
	r := (spd - minSpd) / (maxSpd - minSpd)
	return bonusMax * min(max(r, 0), 1)
}

// timelineOccupant is another actor's icon already sitting on the track.
type timelineOccupant struct {
	pos, spd float64
}

// resolveTimelinePos returns where an actor with the given spd should land
// when placed at pos, so its icon doesn't overlap any occupant. If pos is
// taken, the newcomer slots in front (right) of the occupant when it is
// strictly faster and behind (left) otherwise, continuing in that direction
// past any further occupants until it finds a free spot. If that runs off
// the track it tries the opposite direction, and failing both keeps pos.
// Landing at atbMax is never resolved: the goal is a queue, not the track.
func resolveTimelinePos(pos, spd float64, others []timelineOccupant, spacing float64) float64 {
	pos = clampAtbPosition(pos)
	if pos >= atbMax || spacing <= 0 {
		return pos
	}
	const eps = 1e-6
	blocker := func(p float64) (timelineOccupant, bool) {
		for _, o := range others {
			if math.Abs(o.pos-p) < spacing-eps {
				return o, true
			}
		}
		return timelineOccupant{}, false
	}
	first, hit := blocker(pos)
	if !hit {
		return pos
	}
	dir := -1.0
	if spd > first.spd {
		dir = 1.0
	}
	for _, d := range []float64{dir, -dir} {
		p, o := pos, first
		for range len(others) + 1 {
			p = o.pos + d*spacing
			if p < 0 || p >= atbMax {
				break
			}
			var blocked bool
			if o, blocked = blocker(p); !blocked {
				return p
			}
		}
	}
	return pos
}

// actorOnTrack reports whether actor's icon is currently drawn on the
// timeline track (as opposed to dead, at the goal, or in the wait rows).
func (s *BattleScene) actorOnTrack(actor int) bool {
	if actor < partySize {
		if s.game.PlayerHP[actor] <= 0 || s.deadWaitStuck[actor] ||
			s.waitStance[actor] || s.waitCancelHold[actor] > 0 {
			return false
		}
	} else if s.enemies[enemySlotFromActor(actor)].HP <= 0 {
		return false
	}
	return s.atbGauge[actor] < atbMax
}

// placeOnTimeline sets actor's gauge to pos, nudged by resolveTimelinePos
// so it doesn't land on top of another icon already on the track.
func (s *BattleScene) placeOnTimeline(actor int, pos float64) {
	var others []timelineOccupant
	for a := 0; a < partySize+len(s.enemies); a++ {
		if a != actor && s.actorOnTrack(a) {
			others = append(others, timelineOccupant{s.atbGauge[a], s.actorSpd(a)})
		}
	}
	s.atbGauge[actor] = resolveTimelinePos(pos, s.actorSpd(actor), others, atbIconSpacing())
}

// actorReturnBonus is actor's Spd-based bonus added to return positions.
func (s *BattleScene) actorReturnBonus(actor int) float64 {
	return spdBonus(s.actorSpd(actor), s.minActorSpd, s.maxActorSpd, spdReturnBonusMax)
}

func (s *BattleScene) resetPlayerGaugeTo(actor int, pos float64) {
	if actor < 0 || actor >= partySize {
		return
	}
	s.placeOnTimeline(actor, pos+s.actorReturnBonus(actor))
}

func (s *BattleScene) resetEnemyGaugeTo(slot int, pos float64) {
	if slot < 0 || slot >= len(s.enemies) {
		return
	}
	actor := s.enemyActorIndex(slot)
	s.placeOnTimeline(actor, pos+s.actorReturnBonus(actor))
}

// initTimelinePositions sets every actor's starting gauge to their
// spdStartBonusMax-scaled Spd bonus, placing slowest first so faster actors
// slot in front on ties.
func (s *BattleScene) initTimelinePositions() {
	n := partySize + len(s.enemies)
	s.minActorSpd, s.maxActorSpd = math.Inf(1), math.Inf(-1)
	order := make([]int, n)
	for a := range order {
		order[a] = a
		spd := s.actorSpd(a)
		s.minActorSpd = min(s.minActorSpd, spd)
		s.maxActorSpd = max(s.maxActorSpd, spd)
		// Park everyone at the goal so unplaced actors don't count as
		// occupants while the others are being placed.
		s.atbGauge[a] = atbMax
	}
	sort.SliceStable(order, func(i, j int) bool {
		return s.actorSpd(order[i]) < s.actorSpd(order[j])
	})
	for _, a := range order {
		s.placeOnTimeline(a, spdBonus(s.actorSpd(a), s.minActorSpd, s.maxActorSpd, spdStartBonusMax))
	}
}

// pendingActionReturnPosition resolves the return position for whatever
// player action is currently recorded in s.pendingSkill.
func (s *BattleScene) pendingActionReturnPosition() float64 {
	p := s.waitingActor
	if p < 0 || p >= partySize || s.pendingSkill < 1 {
		return normalAttackReturnPosition
	}
	skillIdx := s.pendingSkill - 1
	skills := s.game.CharacterSkills(p)
	if skillIdx < 0 || skillIdx >= len(skills) {
		return normalAttackReturnPosition
	}
	levels := skills[skillIdx].Levels
	lv := s.lastSkillLevel[p][skillIdx]
	if lv < 1 || lv > len(levels) {
		lv = 1
	}
	return levels[lv-1].ReturnPosition
}

func (s *BattleScene) actorPosX(actor int) float64 {
	start := timelineStartX
	end := s.goalScreenX()
	ratio := s.atbGauge[actor] / atbMax
	if ratio > 1 {
		ratio = 1
	}
	return start + ratio*(end-start) - iconSize/2
}

func (s *BattleScene) isActorReady(actor int) bool {
	if actor >= 0 && actor < partySize && s.waitCancelHold[actor] > 0 {
		return false
	}
	return s.atbGauge[actor] >= atbMax
}

func (s *BattleScene) tryStartNextActor() {
	if s.battleLogTimer > 0 {
		return
	}
	best := -1
	for i := 0; i < partySize; i++ {
		if !s.isActorReady(i) || s.waitStance[i] {
			continue
		}
		if best < 0 || s.actsBefore(i, best) {
			best = i
		}
	}
	if best >= 0 {
		s.openPlayerMenu(best)
		return
	}
	bestEnemy := -1
	for i := range s.enemies {
		if s.enemies[i].HP <= 0 {
			continue
		}
		actor := s.enemyActorIndex(i)
		if !s.isActorReady(actor) {
			continue
		}
		if bestEnemy < 0 || s.actsBefore(actor, s.enemyActorIndex(bestEnemy)) {
			bestEnemy = i
		}
	}
	if bestEnemy >= 0 {
		s.actingEnemySlot = bestEnemy
		s.waitingActor = s.enemyActorIndex(bestEnemy)
		s.enemyIsActing = true
		s.enemyWindupTimer = enemyWindupDuration
	}
}

// actorSpd returns the Spd stat of a party member or enemy actor index.
func (s *BattleScene) actorSpd(actor int) float64 {
	if actor < partySize {
		return float64(s.game.PlayerSpd[actor])
	}
	return s.enemies[enemySlotFromActor(actor)].Speed
}

// actsBefore reports whether actor a should take its turn before b: the
// higher gauge wins, and on a tie (e.g. both capped at atbMax) the higher
// Spd goes first. Remaining ties keep the existing index order.
func (s *BattleScene) actsBefore(a, b int) bool {
	if s.atbGauge[a] != s.atbGauge[b] {
		return s.atbGauge[a] > s.atbGauge[b]
	}
	return s.actorSpd(a) > s.actorSpd(b)
}

func (s *BattleScene) openPlayerMenu(actor int) {
	s.waitingActor = actor
	s.activePlayer = actor + 1
	s.commandIndex = s.lastCommandIndex[actor]
	s.pendingSynergy = false
	s.commandTapArmed = false
	s.rewindButtonArmed = false
	s.itemButtonArmed = false
	s.battlePhase = phasePlayerMenu
	s.playerPose[actor] = poseReady
	s.playerAnimTimer[actor] = 0
	s.readySlideX[actor] = 0.0
}

func hitTestCommandMenu(game *Game) (int, bool) {
	positions := commandIconPositions()
	rects := make([]tapRect, 4)
	for i, pos := range positions {
		icon := game.CommandIcons[i]
		iw := float64(icon.Bounds().Dx())
		ih := float64(icon.Bounds().Dy())
		rects[i] = tapRect{x: pos[0] - iw/2, y: pos[1] - ih/2, w: iw, h: ih}
	}
	return hitTestTapRects(rects)
}

func (s *BattleScene) updatePlayerMenu() Scene {
	if isMenuUpPressed() {
		s.commandIndex = cmdNormalAttack
		s.game.Audio.PlaySEByKey("cursor")
	}
	if isMenuLeftPressed() {
		s.commandIndex = cmdSkill
		s.game.Audio.PlaySEByKey("cursor")
	}
	if isMenuRightPressed() {
		s.commandIndex = cmdWait
		s.game.Audio.PlaySEByKey("cursor")
	}
	if isMenuDownPressed() {
		s.commandIndex = cmdFlee
		s.game.Audio.PlaySEByKey("cursor")
	}
	if isMenuUpPressed() || isMenuLeftPressed() || isMenuRightPressed() || isMenuDownPressed() {
		s.rewindButtonArmed = false
		s.itemButtonArmed = false
	}
	tappedIdx, tappedOk := hitTestCommandMenu(s.game)
	rewindTapped := isRewindButtonJustPressed(s.game)
	// The item button's widened touch area overlaps the skill/flee tap rects;
	// a tap inside a command icon goes to that icon only.
	itemTapped := !tappedOk && isItemButtonJustPressed()

	if tappedOk {
		s.rewindButtonArmed = false
		s.itemButtonArmed = false
	}
	if rewindTapped {
		s.commandTapArmed = false
		s.itemButtonArmed = false
	}
	if itemTapped {
		s.commandTapArmed = false
		s.rewindButtonArmed = false
	}

	tapped := tapArmSelectOrConfirm(tappedIdx, tappedOk, &s.commandIndex, &s.commandTapArmed, s.game.Audio)
	rewindConfirm := tapArmButtonConfirm(rewindTapped, &s.rewindButtonArmed, s.game.Audio)
	itemConfirm := tapArmButtonConfirm(itemTapped, &s.itemButtonArmed, s.game.Audio)

	if inpututil.IsKeyJustPressed(ebiten.KeyF) || rewindConfirm {
		p := s.waitingActor
		if p >= 0 && p < partySize && s.canUseRewind(p) {
			s.game.Audio.PlaySEByKey("decide")
			s.executeRewind(p)
			return nil
		}
		s.game.Audio.PlaySEByKey("error")
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyI) || itemConfirm {
		p := s.waitingActor
		if p >= 0 && p < partySize && s.hasAnyBattleUsableItem() {
			s.game.Audio.PlaySEByKey("decide")
			s.itemIndex = 0
			s.battlePhase = phaseItemMenu
			return nil
		}
		s.game.Audio.PlaySEByKey("error")
	}

	confirm := isConfirmKeyPressed() || tapped

	if !confirm {
		return nil
	}

	p := s.waitingActor
	if p < 0 || p >= partySize {
		s.battlePhase = phaseATB
		s.waitingActor = -1
		return nil
	}

	if s.commandIndex == cmdWait {
		if !s.hasFullPartyForSynergy() {
			s.game.Audio.PlaySEByKey("error")
			return nil
		}
	}

	s.game.Audio.PlaySEByKey("decide")
	s.lastCommandIndex[p] = s.commandIndex

	switch s.commandIndex {
	case cmdNormalAttack:
		s.pendingSkill = 0
		s.pendingSynergy = false
		s.targetIndex = s.firstAliveEnemySlot()
		s.battlePhase = phaseTargetSelect
		return nil
	case cmdSkill:
		skills := s.game.CharacterSkills(p)
		s.skillIndex = s.lastSkillIndex[p]
		if s.skillIndex < 0 || s.skillIndex >= len(skills) || !s.game.IsSkillUnlocked(p, s.skillIndex) {
			s.skillIndex = s.firstUnlockedSkillIndex(p, skills)
		}
		for i := range s.skillLevelCursors[p] {
			if s.skillLevelCursors[p][i] < 1 {
				lv := s.lastSkillLevel[p][i]
				if lv < 1 {
					lv = 1
				}
				s.skillLevelCursors[p][i] = lv
			}
		}
		s.skillMenuOpenTimer = 0.1
		s.battlePhase = phaseSkillMenu
		return nil
	case cmdWait:
		// The member completing the 4-person wait picks the synergy target.
		if s.othersAllInWaitStance(p) {
			s.pendingSkill = 0
			s.pendingSynergy = true
			s.targetIndex = s.firstAliveEnemySlot()
			s.battlePhase = phaseTargetSelect
			return nil
		}
		s.enterWaitStance(p)
		s.waitingActor = -1
		s.battlePhase = phaseATB
		s.tryStartNextActor()
		return nil
	case cmdFlee:
		if rand.Intn(100) >= s.fleeSuccessRate() {
			s.resetPlayerGaugeTo(p, fleeFailReturnPosition)
			s.waitingActor = -1
			s.battlePhase = phaseATB
			s.battleLog = "逃げられなかった"
			s.battleLogTimer = 1.5
			return nil
		}
		s.fleeSucceeded = true
		s.battleLog = "逃げきれた！"
		s.battleLogTimer = 1.5
		s.battlePhase = phaseMessage
		return nil
	}
	return nil
}

func (s *BattleScene) updateSkillMenu(dt float64) {
	if s.skillMenuOpenTimer > 0 {
		s.skillMenuOpenTimer -= 1.0 / 60.0
		return
	}
	p := s.waitingActor
	if p < 0 || p >= partySize {
		return
	}
	skills := s.game.CharacterSkills(p)
	menuLen := len(skills)

	if isMenuDownPressed() {
		s.skillIndex = s.nextUnlockedSkillIndex(p, s.skillIndex, 1, menuLen)
		s.lastSkillIndex[p] = s.skillIndex
		s.game.Audio.PlaySEByKey("cursor")
	}
	if isMenuUpPressed() {
		s.skillIndex = s.nextUnlockedSkillIndex(p, s.skillIndex, -1, menuLen)
		s.lastSkillIndex[p] = s.skillIndex
		s.game.Audio.PlaySEByKey("cursor")
	}

	arrowTapped := s.handleSkillLevelArrowTaps(p, skills)
	tappedIdx, tappedOk := -1, false
	if !arrowTapped {
		// 一覧は覚えているスキルだけを詰めて並べているので、行からスキルに直す。
		if row, ok := s.hitTestBattleSubRows(menuLen); ok {
			tappedIdx, tappedOk = s.game.SkillAtDisplayRow(p, row)
		}
	}
	tapped := tapSelectOrConfirm(tappedIdx, tappedOk, &s.skillIndex, s.game.Audio)
	if tappedOk {
		s.lastSkillIndex[p] = s.skillIndex
	}

	curLv := s.game.PlayerSkillLv[p][s.skillIndex]
	if curLv < 1 {
		curLv = 1
	}
	if curLv > len(skills[s.skillIndex].Levels) {
		curLv = len(skills[s.skillIndex].Levels)
	}

	if s.skillLevelCursors[p][s.skillIndex] < 1 {
		s.skillLevelCursors[p][s.skillIndex] = 1
	}
	if s.skillLevelCursors[p][s.skillIndex] > curLv {
		s.skillLevelCursors[p][s.skillIndex] = curLv
	}

	if isMenuRightPressed() {
		if s.skillLevelCursors[p][s.skillIndex] < curLv {
			s.skillLevelCursors[p][s.skillIndex]++
			s.game.Audio.PlaySEByKey("cursor")
		}
	}
	if isMenuLeftPressed() {
		if s.skillLevelCursors[p][s.skillIndex] > 1 {
			s.skillLevelCursors[p][s.skillIndex]--
			s.game.Audio.PlaySEByKey("cursor")
		}
	}
	if isEscapePressed() || (!arrowTapped && !tappedOk && s.isTapOutsideBattleSubPanel()) {
		s.game.Audio.PlaySEByKey("cancel")
		s.battlePhase = phasePlayerMenu
		return
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		if s.canUseRewind(p) {
			s.game.Audio.PlaySEByKey("decide")
			s.executeRewind(p)
			return
		}
	}

	confirm := isConfirmKeyPressed() || tapped
	if !confirm {
		return
	}

	lv := s.skillLevelCursors[p][s.skillIndex]
	if lv < 1 {
		lv = 1
	}
	data := skills[s.skillIndex].Levels[lv-1]
	if !s.game.IsSkillUnlocked(p, s.skillIndex) {
		s.game.Audio.PlaySEByKey("error")
		return
	}
	if s.game.PlayerMP[p] < s.effectiveMPCost(data.MPCost) {
		s.game.Audio.PlaySEByKey("error")
		return
	}
	if data.targetsAlly() && data.Target != TargetSelf && len(s.allyAllTargets(data)) == 0 {
		// e.g. レイズ with nobody down.
		s.game.Audio.PlaySEByKey("error")
		return
	}
	s.game.Audio.PlaySEByKey("decide")
	s.pendingSkill = s.skillIndex + 1

	if s.skillIndex < len(s.lastSkillLevel[p]) {
		s.lastSkillLevel[p][s.skillIndex] = lv
	}

	if data.Target == TargetSelf {
		s.executeSelfSkill()
		return
	}
	if data.targetsAlly() {
		s.healTargetIndex = s.firstAllyTarget(data)
		s.battlePhase = phaseHealSelect
		return
	}

	if data.Target == TargetAll {
		s.targetIndex = maxEnemies
	} else {
		s.targetIndex = s.firstAliveEnemySlot()
	}
	s.battlePhase = phaseTargetSelect
}
