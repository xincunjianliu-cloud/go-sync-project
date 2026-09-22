package main

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const chestTriggerMargin = 32

// footstepIntervalTicks/dashFootstepIntervalTicks は歩行中に足音SEを鳴らす間隔(フレーム数)。
// ダッシュ中は歩幅が広く歩調が速いため、通常より短い間隔にしている。
const (
	footstepIntervalTicks     = 18
	dashFootstepIntervalTicks = 12
)

func (s *FieldScene) openMessageLog() {
	s.isLogActive = true
	s.logScrollOffset = 0
	s.logCursorIndex = len(s.msgLog) - 1
	if s.logCursorIndex < 0 {
		s.logCursorIndex = 0
	}
}

func (s *FieldScene) moveLogCursor(delta int) {
	n := len(s.msgLog)
	if n == 0 {
		return
	}

	s.logCursorIndex += delta
	if s.logCursorIndex < 0 {
		s.logCursorIndex = 0
	}
	if s.logCursorIndex > n-1 {
		s.logCursorIndex = n - 1
	}

	endIndex := n - int(s.logScrollOffset)
	startIndex := endIndex - logVisibleCount
	if startIndex < 0 {
		startIndex = 0
	}

	if s.logCursorIndex < startIndex {
		s.logScrollOffset = float64(n - s.logCursorIndex - logVisibleCount)
		if s.logScrollOffset < 0 {
			s.logScrollOffset = 0
		}
	} else if s.logCursorIndex > endIndex-1 {
		s.logScrollOffset = float64(n - s.logCursorIndex - 1)
		if s.logScrollOffset < 0 {
			s.logScrollOffset = 0
		}
	}
}

// applyFieldDebugCheats mirrors BattleScene.applyDebugCheats (Shift+key,
// gated by debugModeEnabled) for the one field-scene tutorial, so it can be
// checked without grinding 100 SP and winning a battle first.
func (s *FieldScene) applyFieldDebugCheats() {
	if !debugModeEnabled {
		return
	}
	if !ebiten.IsKeyPressed(ebiten.KeyShift) {
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyU) {
		s.skillUpgradeTutorialActive = true
	}
}

func (s *FieldScene) Update(dt float64) Scene {
	if s == nil {
		return s
	}

	s.applyFieldDebugCheats()

	if s.mapNameBannerActive {
		s.mapNameBannerElapsed += dt
		if s.mapNameBannerElapsed >= mapNameBannerShowDuration+mapNameBannerFadeDuration {
			s.mapNameBannerActive = false
		}
	}

	if s.encounterEffectActive {
		s.encounterEffectTimer += dt
		if s.encounterEffectTimer >= encounterEffectDuration && s.pendingBattleScene != nil {
			s.encounterEffectTimer = encounterEffectDuration
			battleScene := s.pendingBattleScene
			s.pendingBattleScene = nil
			s.game.ChangeSceneWithFade(battleScene, fadeTimeBattleIn)
		}
		return s
	}

	s.updateTouchStick()
	s.wallAnimTick++

	if s.isLogActive {
		if isMenuUpPressed() {
			s.moveLogCursor(-1)
			s.game.Audio.PlaySEByKey("cursor")
		}
		if isMenuDownPressed() {
			s.moveLogCursor(1)
			s.game.Audio.PlaySEByKey("cursor")
		}

		maxLogScroll := float64(len(s.msgLog) - logVisibleCount)
		if maxLogScroll < 0 {
			maxLogScroll = 0
		}

		barX, barY, barW, barH := logScrollBarRect(s.game, len(s.msgLog))
		if scrollBarMoveAmt := s.logScrollBarDrag.step(barX, barY, barW, barH); scrollBarMoveAmt != 0 {
			_, _, _, _, moveRange := logScrollBarGeometry(s.game, len(s.msgLog))
			if moveRange > 0 {
				s.logScrollBarDrag.stepContinuousScroll(-scrollBarMoveAmt/moveRange*maxLogScroll, &s.logScrollOffset, 0, maxLogScroll)
			}
		}
		scrollBarHeld := s.logScrollBarDrag.active

		rowPitch := float64(s.game.LogEntryImg.Bounds().Dy()) + logEntryGap

		if _, wheelY := ebiten.Wheel(); wheelY != 0 && !scrollBarHeld && rowPitch > 0 {
			s.logDrag.stepContinuousScroll(wheelY*wheelScrollPxPerNotch/rowPitch, &s.logScrollOffset, 0, maxLogScroll)
		}

		if !scrollBarHeld {
			barX, _, _, _, _ := logScrollBarGeometry(s.game, len(s.msgLog))
			moveAmt := s.logDrag.step(0, 0, barX-scrollBarTouchPad, float64(gameHeight))
			if moveAmt != 0 && rowPitch > 0 {
				s.logDrag.stepContinuousScroll(moveAmt/rowPitch, &s.logScrollOffset, 0, maxLogScroll)
			}
			if s.logDrag.justTapped {
				if idx, ok := hitTestLogEntries(s.game, len(s.msgLog), s.logScrollOffset, s.logDrag.tapX, s.logDrag.tapY); ok {
					if idx != s.logCursorIndex {
						s.game.Audio.PlaySEByKey("cursor")
					}
					s.logCursorIndex = idx
				}
			}
		}

		imgW := float64(s.game.LogEntryImg.Bounds().Dx())
		imgH := float64(s.game.LogEntryImg.Bounds().Dy())
		contentH := float64(logVisibleCount)*imgH + float64(logVisibleCount-1)*logEntryGap
		logRect := tapRect{x: logImageStartX, y: logImageStartY, w: imgW, h: contentH}
		barRect := tapRect{x: barX, y: barY, w: barW, h: barH}
		if unrelatedTapOutsideRects(logRect, barRect) {
			s.isLogActive = false
			s.game.Audio.PlaySEByKey("menu_toggle")
		}

		if isConfirmKeyPressed() || isEscapePressed() || isLogTogglePressed() {
			s.isLogActive = false
			s.game.Audio.PlaySEByKey("menu_toggle")
		}
		return s
	}

	currentSpeaker := ""
	currentExpression := 0
	if s.isMsgActive && s.msgIndex < len(s.msgTexts) {
		currentSpeaker = s.msgTexts[s.msgIndex].Speaker
		currentExpression = s.msgTexts[s.msgIndex].Expression
		s.msg.Speed = s.game.MessageSpeedTicks()
		s.msg.Tick()
		s.msg.UpdateCharaAnim(currentSpeaker, currentExpression)
	}
	if s.isChoiceActive {
		if isMenuUpPressed() || isMenuDownPressed() {
			s.choiceIndex = 1 - s.choiceIndex
			s.game.Audio.PlaySEByKey("cursor")
		}
		if isConfirmKeyPressed() {
			s.game.Audio.PlaySEByKey("decide")
			cb := s.onChoiceConfirm
			selected := s.choiceIndex
			s.isChoiceActive = false
			s.choiceOptions = nil
			s.onChoiceConfirm = nil
			if cb != nil {
				cb(selected)
			}
		}
		return s
	}

	if s.wallFadeActive {
		s.updateWallFade(dt)
		return s
	}

	if s.isItemGetActive {
		dismissed := false
		if s.itemGetAutoCloseOnly {
			s.itemGetAutoCloseTimer -= dt
			dismissed = s.itemGetAutoCloseTimer <= 0
		} else {
			dismissed = isConfirmKeyPressed() || isEscapePressed() || len(justPressedTouchPoints()) > 0
		}
		if dismissed {
			s.isItemGetActive = false
			s.itemGetName = ""
			s.itemGetSubLabel = ""
			s.itemGetPlainMessage = false
			s.itemGetAutoCloseOnly = false
			s.itemGetAutoCloseTimer = 0
		}
		return s
	}

	if s.skillUpgradeTutorialActive {
		if isConfirmKeyPressed() || len(justPressedTouchPoints()) > 0 {
			s.game.Audio.PlaySEByKey("decide")
			s.skillUpgradeTutorialPage++
			if s.skillUpgradeTutorialPage >= skillUpgradeTutorialPageCount {
				s.skillUpgradeTutorialActive = false
				s.skillUpgradeTutorialPage = 0
			}
		}
		return s
	}

	if s.justDefeatedBoss > 0 {
		s.applyDialogue(bossClearDialogues[s.justDefeatedBoss])
		s.msgIndex = 0
		s.beginMessage()
		s.justDefeatedBoss = 0
		return s
	}

	if s.pendingAutoHealMessage {
		s.pendingAutoHealMessage = false
		s.openCenterMessagePopupTimed(autoHealIntroMessage, autoHealIntroMessageDuration)
		return s
	}

	if s.isCutscene || s.cseFadeMode == 2 || s.cseFadeMode == 3 || s.cseFadeMode == 4 {
		if s.cseFadeMode == 2 {
			s.cseFadeAlpha += s.cseFadeSpeed * dt
			if s.cseFadeAlpha >= 1.0 {
				s.cseFadeAlpha = 1.0
				s.cseFadeMode = 3
				if s.onDarkCallback != nil {
					cb := s.onDarkCallback
					s.onDarkCallback = nil
					cb()
				}

			}
		}

		if s.cseFadeMode == 4 {
			s.cseFadeAlpha -= s.cseFadeSpeed * dt
			if s.cseFadeAlpha <= 0 {
				s.cseFadeAlpha = 0
				s.cseFadeMode = 0
				if !s.isCutscene {
					if s.onFadeCompleteCallback != nil {
						cb := s.onFadeCompleteCallback
						s.onFadeCompleteCallback = nil
						cb()
					} else {
						s.beginMessage()
					}
				}
			}
			if !s.isCutscene {
				return s
			}
		}

		if s.cseFadeMode == 3 {
			s.cseDarkElapsed += dt
			if s.cseDarkElapsed >= s.CseDarkDuration {
				s.cseDarkElapsed = 0
				s.cseFadeMode = 4
				if s.cseFadeInSpeed > 0 {
					s.cseFadeSpeed = s.cseFadeInSpeed
				} else {
					s.cseFadeSpeed = 1.0 / 0.4
				}
			}
		}

		if s.isCutscene {
			cutsceneSpeed := 100.0 * dt
			movedThisFrame := false
			s.cseElapsed += dt

			if s.routeIndex < len(s.cutsceneRoute) {
				step := s.cutsceneRoute[s.routeIndex]
				moveDist := cutsceneSpeed

				if s.currentStepDist+moveDist > step.Dist {
					moveDist = step.Dist - s.currentStepDist
				}

				s.dir = step.Dir
				switch step.Dir {
				case 0:
					s.py += moveDist
				case 1:
					s.px -= moveDist
				case 2:
					s.px += moveDist
				case 3:
					s.py -= moveDist
				}

				s.currentStepDist += moveDist
				movedThisFrame = true

				if s.currentStepDist >= step.Dist {
					s.routeIndex++
					s.currentStepDist = 0
				}
			}
			if movedThisFrame {
				s.animCount++
			} else {
				s.animCount = 0
			}

			if s.cseFadeMode == 1 && s.cseElapsed >= s.CseFadeOutDelay {
				s.cseFadeMode = 2
				s.cseFadeSpeed = 1.0 / 0.4
			}

			if s.routeIndex >= len(s.cutsceneRoute) {
				s.isCutscene = false
				s.animCount = 0

				s.px = math.Floor(s.px/16) * 16
				s.py = math.Floor(s.py/16) * 16
				if s.cutsceneMessage != "" {
					s.applyCutsceneMessage()
					s.msgIndex = 0
				}

				if s.cseFadeMode == 0 {
					s.beginMessage()
				} else if s.cseFadeMode != 4 {
					s.cseDarkElapsed = 0
				}
			}
		}
		return s
	}

	if s.isMsgActive {
		s.msg.Speed = s.game.MessageSpeedTicks()

		if isSkipKeyDown() || isSkipIconHeld(s.game) {
			s.msgSkipHoldElapsed += dt
			if s.msgSkipHoldElapsed >= endingSkipHoldSeconds {
				s.msgSkipHoldElapsed = 0
				s.game.Audio.PlaySEByKey("decide")
				return s.skipMessage()
			}
		} else {
			s.msgSkipHoldElapsed = 0
		}

		if isAutoTogglePressed() || isAutoIconJustPressed(s.game) {
			s.autoMode = !s.autoMode
			s.autoWaitElapsed = 0
			s.game.Audio.PlaySEByKey("menu_toggle")
		}

		if isLogTogglePressed() || isLogIconJustPressed(s.game) {
			s.game.Audio.PlaySEByKey("menu_toggle")
			s.openMessageLog()
			return s
		}

		finished := s.msg.IsFinished(s.msgTexts[s.msgIndex])

		if isMessageAdvancePressed() {
			s.game.Audio.PlaySEByKey("text_advance")
			if !finished {
				s.msg.SkipToEnd(s.msgTexts[s.msgIndex])
				return s
			}
			return s.advanceMessage()
		}

		if s.autoMode && finished {
			s.autoWaitElapsed += dt
			if s.autoWaitElapsed >= autoWaitDuration(s.msgTexts[s.msgIndex]) {
				s.autoWaitElapsed = 0
				return s.advanceMessage()
			}
		} else {
			s.autoWaitElapsed = 0
		}

		return s
	}

	playerFootX := s.px
	playerFootY := s.py

	if isConfirmKeyPressed() {
		if s.isPushingBlock {
			s.endPushingBlock()
			return s
		}

		if s.nearDoorEvent {
			if next := s.triggerDoorWarp(); next != nil {
				return next
			}
		}

		if s.nearBlockID != "" {
			s.beginPushingBlock(s.nearBlockID)
			return s
		}

		for _, layer := range s.tileMap.Layers {
			if !strings.HasPrefix(layer.Name, "events") {
				continue
			}
			for _, obj := range layer.Objects {
				p := objProps(obj)
				evType := p["type"]
				evText := p["text"]
				isKeyChest := isKeyChestObj(p)
				isItemChest := isItemChestObj(p)
				isChest := isKeyChest || isItemChest
				isLockedWall := isLockedWallObj(p)
				isLeverControlledWall := isLeverControlledWallObj(p)
				isLever := isLeverObj(p)

				if isLockedWall && s.wallIsOpen(obj) {
					continue
				}
				if isLeverControlledWall {
					continue
				}
				if isBlockObj(p) || isBlockSpotObj(p) || isBlockDoorObj(p) {
					continue
				}

				if isChest || isLockedWall || isLever {
					if !objContainsMargin(obj, playerFootX, playerFootY, chestTriggerMargin) {
						continue
					}
				} else if !objContains(obj, playerFootX, playerFootY) {
					continue
				}

				if evType == evTypeEvent && !strings.HasPrefix(evText, bossTextPrefix) {
					if isKeyChest {
						s.openKeyChest(obj, strings.TrimPrefix(evText, chestKeyTextPrefix))
						return s
					} else if isItemChest {
						s.openChest(obj, strings.TrimPrefix(evText, chestTextPrefix))
						return s
					} else if isLockedWall {
						s.examineWall(obj)
						return s
					} else if isLever {
						s.pullLever(obj)
						return s
					} else if evText != "" {
						key := chestKey(s.currentMap, obj)
						seen := s.game.SeenEvents[key]
						s.applyDialogue(resolveEventDialogue(evText, p["repeattext"], seen))
						if s.game.SeenEvents == nil {
							s.game.SeenEvents = make(map[string]bool)
						}
						s.game.SeenEvents[key] = true
					} else {
						s.applyDialogue(BossDialogue{Commands: []EventCommand{{Speaker: "", Text: "調べるとなにかあるかもしれない"}}})
					}

					s.msgIndex = 0
					s.beginMessage()
					return s
				}
			}
		}
	}

	if isDashTogglePressed() {
		s.game.IsDashing = !s.game.IsDashing
	}

	s.isDashingNow = s.game.IsDashing || s.touchStickDash()

	perFrame := s.playerCfg.MoveSpeed * dt
	if s.isDashingNow && !s.isPushingBlock {
		perFrame *= s.playerCfg.DashSpeedMultiplier
	}
	touchDx, touchDy := s.touchMoveDir()
	touchScale := s.touchStickSpeedScale()

	// Keyboard input always moves at full speed; the touch stick scales
	// smoothly from a light tap up to full speed as it's pushed further,
	// giving analog-feeling precision instead of an all-or-nothing snap.
	moveX, moveY := 0.0, 0.0
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyUp) {
		moveY = -perFrame
		s.dir = 3
	} else if touchDy < 0 {
		moveY = -perFrame * touchScale
		s.dir = 3
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyDown) {
		moveY = perFrame
		s.dir = 0
	} else if touchDy > 0 {
		moveY = perFrame * touchScale
		s.dir = 0
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyLeft) {
		moveX = -perFrame
		s.dir = 1
	} else if touchDx < 0 {
		moveX = -perFrame * touchScale
		s.dir = 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyRight) {
		moveX = perFrame
		s.dir = 2
	} else if touchDx > 0 {
		moveX = perFrame * touchScale
		s.dir = 2
	}

	prevPx, prevPy := s.px, s.py

	actualMovedDist := 0.0
	if s.isPushingBlock {
		actualMovedDist += s.updateBlockPush(dt, moveX != 0 || moveY != 0)
	} else if moveX != 0 && moveY != 0 {
		actualMovedDist += s.moveBoth(moveX, moveY)
	} else {
		actualMovedDist += s.moveAxis(moveX, true)
		actualMovedDist += s.moveAxis(moveY, false)
	}

	encounterMovedDist := math.Hypot(s.px-prevPx, s.py-prevPy)
	if !s.isPushingBlock && s.isWall(s.px, s.py) {
		s.resolveEmbeddedPosition()
	}

	if actualMovedDist > 0 {
		s.animCount++
		interval := footstepIntervalTicks
		seKey := "footstep"
		if s.isDashingNow {
			interval = dashFootstepIntervalTicks
			seKey = "dash_footstep"
		}
		if s.animCount%interval == 0 {
			s.game.Audio.PlaySEByKey(seKey)
		}
	} else {
		s.animCount = 0
	}
	moved := actualMovedDist > 0

	s.checkDoorProximity()
	s.updateObjectiveGuide()
	s.updateDarkness()
	if !s.isPushingBlock {
		s.updateBlockDoors()
	}
	s.updateScreenShake(dt)

	if inpututil.IsKeyJustPressed(ebiten.KeyM) || fieldTouchMenuPressed() {
		full := ebiten.NewImage(gameWidth, gameHeight)
		s.Draw(full)
		s.game.captureMenuEntryThumb(full)
		s.game.Audio.PlaySEByKey("menu_toggle")
		return NewMenuScene(s.game, s)
	}

	if isLogTogglePressed() {
		s.game.Audio.PlaySEByKey("menu_toggle")
		s.openMessageLog()
		return s
	}

	var targetEnemiesStr string
	targetMaxCount := 1
	s.nearExamineEvent = false
	s.nearBlockID = ""

	if !s.isPushingBlock {
		for _, b := range s.blocks {
			if s.blockIsLocked(b) {
				continue
			}
			if blockNear(b, playerFootX, playerFootY) {
				s.nearBlockID = b.ID
				s.nearExamineEvent = true
				break
			}
		}

		for _, layer := range s.tileMap.Layers {
			if !strings.HasPrefix(layer.Name, "events") {
				continue
			}
			for _, obj := range layer.Objects {
				p := objProps(obj)
				evType := p["type"]
				evText := p["text"]
				routeStr := p["route"]

				if isChestObj(p) {
					if !s.game.OpenedChests[chestKey(s.currentMap, obj)] &&
						objContainsMargin(obj, playerFootX, playerFootY, chestTriggerMargin) {
						s.nearExamineEvent = true
					}
				} else if isLockedWallObj(p) {
					if !s.wallIsOpen(obj) &&
						objContainsMargin(obj, playerFootX, playerFootY, chestTriggerMargin) {
						s.nearExamineEvent = true
					}
				} else if isLeverControlledWallObj(p) {
				} else if isLeverObj(p) {
					exhausted := p["oneway"] == "true" && s.game.RaisedLevers[p["id"]]
					if !exhausted && objContainsMargin(obj, playerFootX, playerFootY, chestTriggerMargin) {
						s.nearExamineEvent = true
					}
				} else if isBlockObj(p) || isBlockSpotObj(p) || isBlockDoorObj(p) {
				} else if evType == evTypeEvent && objContains(obj, playerFootX, playerFootY) {
					s.nearExamineEvent = true
				}

				if evType == evTypeTrigger {
					bossID, hasBossID := objPropInt(obj, "bossid")
					instant, _ := objPropBool(obj, "instant")

					if hasBossID && isBossDefeated(s.game, bossID) {
						continue
					}

					// ボスID無しのトリガーは1回きりの演出として扱う。踏むたびに
					// 同じ台詞が流れるのは不自然なので、2回目以降セリフに切り替える
					// のではなく、一度発火したら二度と発火しないようにする。
					var triggerKey string
					if !hasBossID {
						triggerKey = chestKey(s.currentMap, obj)
						if s.game.SeenEvents[triggerKey] {
							continue
						}
					}

					if segmentIntersectsRect(prevPx, prevPy, s.px, s.py, obj.X, obj.Y, obj.Width, obj.Height) {

						if hasBossID {
							s.pendingCutsceneMsg = fmt.Sprintf("%s%d", bossTextPrefix, bossID)
						} else {
							s.pendingCutsceneMsg = evText
							if s.game.SeenEvents == nil {
								s.game.SeenEvents = make(map[string]bool)
							}
							s.game.SeenEvents[triggerKey] = true
						}
						s.cutsceneHasBossID = hasBossID

						if instant || routeStr == "" || routeStr == "<nil>" {
							s.cutsceneMessage = s.pendingCutsceneMsg
							s.cseFadeAlpha = 0
							s.cseFadeMode = 2
							s.cseFadeSpeed = 1.0 / 0.4
							s.onDarkCallback = nil
							s.onFadeCompleteCallback = func() {
								s.applyCutsceneMessage()
								s.msgIndex = 0
								s.beginMessage()
							}
							return s
						}

						s.pendingCutsceneRoute = []MoveStep{}

						if routeStr != "" && routeStr != "<nil>" {
							steps := strings.Split(routeStr, ",")
							for _, step := range steps {
								step = strings.TrimSpace(step)
								var dir int
								var distStr string
								if strings.HasPrefix(step, "down") {
									dir = 0
									distStr = strings.TrimPrefix(step, "down")
								} else if strings.HasPrefix(step, "left") {
									dir = 1
									distStr = strings.TrimPrefix(step, "left")
								} else if strings.HasPrefix(step, "right") {
									dir = 2
									distStr = strings.TrimPrefix(step, "right")
								} else if strings.HasPrefix(step, "up") {
									dir = 3
									distStr = strings.TrimPrefix(step, "up")
								}
								dist, _ := strconv.ParseFloat(distStr, 64)
								s.pendingCutsceneRoute = append(s.pendingCutsceneRoute, MoveStep{Dir: dir, Dist: dist})
							}
						} else {
							s.pendingCutsceneRoute = append(s.pendingCutsceneRoute, MoveStep{Dir: s.dir, Dist: 0})
						}

						totalDist := 0.0
						for _, step := range s.pendingCutsceneRoute {
							totalDist += step.Dist
						}
						s.cseTotalDuration = totalDist / 100.0

						s.isCutscene = true
						s.cutsceneMessage = s.pendingCutsceneMsg
						s.cutsceneRoute = s.pendingCutsceneRoute
						s.routeIndex = 0
						s.currentStepDist = 0
						s.cseElapsed = 0
						s.cseFadeAlpha = 0
						s.cseFadeMode = 1
						return s
					}
				}

				if evType == evTypeEnemy && objContains(obj, playerFootX, playerFootY) {
					targetEnemiesStr = evText
					targetMaxCount = 1
					if n, ok := objPropInt(obj, "maxcount"); ok && n > 1 {
						targetMaxCount = n
						if targetMaxCount > maxEnemies {
							targetMaxCount = maxEnemies
						}
					}
				}
			}
		}
	}

	if moved && actualMovedDist > 0 {
		if s.safetyDistance > 0 {
			s.safetyDistance -= encounterMovedDist
		} else if targetEnemiesStr != "" {
			s.walkCooldown += encounterMovedDist
			if s.walkCooldown >= 32.0 {
				s.walkCooldown = 0
				s.encounterWeight += 0.01
				if rand.Float64()*100.0 < s.encounterWeight {
					s.encounterWeight = 0.0
					allowedEnemies := strings.Split(targetEnemiesStr, ",")
					count := 1 + rand.Intn(targetMaxCount)
					chosenEnemyNames := make([]string, count)
					for i := 0; i < count; i++ {
						chosenEnemyNames[i] = strings.TrimSpace(allowedEnemies[rand.Intn(len(allowedEnemies))])
					}
					s.game.Audio.PlaySEByKey("encounter")
					battleScene := NewBattleScene(s.game, s.currentMap, s.px, s.py, s.dir, "enemy", chosenEnemyNames)
					s.startEncounterEffect(battleScene)
					return s
				}
			}
		} else {
			s.walkCooldown = 0
			s.encounterWeight = 0.0
		}
	}
	return s
}

// startEncounterEffect は現在のフィールド画面を静止画として捉え、
// カメラが回転しながらズームインしていく演出を開始する。演出が終わると
// Update側でbattleSceneへフェード切り替えする。
func (s *FieldScene) startEncounterEffect(battleScene *BattleScene) {
	snapshot := ebiten.NewImage(gameWidth, gameHeight)
	s.drawWorld(snapshot)

	s.encounterSnapshot = snapshot
	s.encounterEffectActive = true
	s.encounterEffectTimer = 0
	s.pendingBattleScene = battleScene
}

func rectsOverlap(ax1, ay1, ax2, ay2, bx1, by1, bx2, by2 float64) bool {
	return ax1 < bx2 && ax2 > bx1 && ay1 < by2 && ay2 > by1
}

func segmentIntersectsRect(x1, y1, x2, y2, rx, ry, rw, rh float64) bool {
	dx := x2 - x1
	dy := y2 - y1
	p := [4]float64{-dx, dx, -dy, dy}
	q := [4]float64{x1 - rx, rx + rw - x1, y1 - ry, ry + rh - y1}
	tMin, tMax := 0.0, 1.0
	for i := 0; i < 4; i++ {
		if p[i] == 0 {
			if q[i] < 0 {
				return false
			}
			continue
		}
		t := q[i] / p[i]
		if p[i] < 0 {
			if t > tMax {
				return false
			}
			if t > tMin {
				tMin = t
			}
		} else {
			if t < tMin {
				return false
			}
			if t < tMax {
				tMax = t
			}
		}
	}
	return true
}

func (s *FieldScene) collisionRectAt(x, y float64) (left, top, right, bottom float64) {
	return s.playerCfg.CollisionRectForDirAt(s.dir, x, y)
}

func (s *FieldScene) isWall(x, y float64) bool {
	if s.tileMap.TileWidth == 0 || s.tileMap.TileHeight == 0 {
		return true
	}
	left, top, right, bottom := s.collisionRectAt(x, y)
	return s.rectHitsObstacles(left, top, right, bottom, "")
}

func (s *FieldScene) rectHitsObstacles(left, top, right, bottom float64, excludeBlockID string) bool {
	checkPoints := [][2]float64{{left, top}, {right, top}, {left, bottom}, {right, bottom}}
	for _, pt := range checkPoints {
		tileX := int(pt[0] / float64(s.tileMap.TileWidth))
		tileY := int(pt[1] / float64(s.tileMap.TileHeight))
		if tileX < 0 || tileY < 0 || tileX >= s.tileMap.Width || tileY >= s.tileMap.Height {
			return true
		}
		index := tileY*s.tileMap.Width + tileX
		for _, layer := range s.tileMap.Layers {
			if layer.Name == wallTileLayerName && index >= 0 && index < len(layer.Data) && layer.Data[index] != 0 {
				return true
			}
		}
	}

	for _, r := range s.collisions {
		if rectsOverlap(left, top, right, bottom, r.X, r.Y, r.X+r.Width, r.Y+r.Height) {
			return true
		}
	}

	for _, layer := range s.tileMap.Layers {
		if !strings.HasPrefix(layer.Name, "events") {
			continue
		}
		for _, obj := range layer.Objects {
			p := objProps(obj)
			switch {
			case isWallObj(p):
				if s.wallIsOpen(obj) && !isLeverWallVisualOnly(p) {
					continue
				}
			case isBlockDoorObj(p):
				if s.blockDoorIsOpen(obj) {
					continue
				}
			default:
				continue
			}
			if rectsOverlap(left, top, right, bottom, obj.X, obj.Y, obj.X+obj.Width, obj.Y+obj.Height) {
				return true
			}
		}
	}

	for _, poly := range s.collisionPolygons {
		for _, pt := range checkPoints {
			if pointInPolygon(pt[0], pt[1], poly.Points) {
				return true
			}
		}
	}

	for _, b := range s.blocks {
		if b.ID == excludeBlockID {
			continue
		}
		if rectsOverlap(left, top, right, bottom, b.X, b.Y, b.X+b.Width, b.Y+b.Height) {
			return true
		}
	}

	return false
}

func pointInPolygon(x, y float64, poly []TiledPoint) bool {
	inside := false
	n := len(poly)
	if n < 3 {
		return false
	}
	for i, j := 0, n-1; i < n; j, i = i, i+1 {
		xi, yi := poly[i].X, poly[i].Y
		xj, yj := poly[j].X, poly[j].Y
		if ((yi > y) != (yj > y)) &&
			(x < (xj-xi)*(y-yi)/(yj-yi)+xi) {
			inside = !inside
		}
	}
	return inside
}

func (s *FieldScene) resolveEmbeddedPosition() {
	if s == nil {
		return
	}
	if !s.isWall(s.px, s.py) {
		return
	}

	bestX, bestY := s.px, s.py
	bestDist := math.Inf(1)
	for step := 1; step <= 32; step++ {
		for _, axis := range []struct {
			dx float64
			dy float64
		}{
			{dx: -1, dy: 0},
			{dx: 1, dy: 0},
			{dx: 0, dy: -1},
			{dx: 0, dy: 1},
			{dx: -1, dy: -1},
			{dx: 1, dy: -1},
			{dx: -1, dy: 1},
			{dx: 1, dy: 1},
		} {
			nx := s.px + axis.dx*float64(step)
			ny := s.py + axis.dy*float64(step)
			if !s.isWall(nx, ny) {
				dist := math.Hypot(nx-s.px, ny-s.py)
				if dist < bestDist {
					bestX, bestY = nx, ny
					bestDist = dist
				}
			}
		}
	}
	if math.IsInf(bestDist, 1) {
		return
	}
	s.px, s.py = bestX, bestY
}

func (s *FieldScene) moveAxis(delta float64, isX bool) float64 {
	if delta == 0 {
		return 0
	}
	step := 1.0
	if delta < 0 {
		step = -1.0
	}
	remaining := math.Abs(delta)
	moved := 0.0
	for remaining > 0 {
		amt := math.Min(remaining, 1.0)
		nx, ny := s.px, s.py
		if isX {
			nx += step * amt
		} else {
			ny += step * amt
		}
		if !s.isWall(nx, ny) {
			if isX {
				s.px += step * amt
			} else {
				s.py += step * amt
			}
			moved += amt
		} else {
			break
		}
		remaining -= amt
	}
	return moved
}

func (s *FieldScene) moveBoth(dx, dy float64) float64 {
	if dx == 0 && dy == 0 {
		return 0
	}
	steps := int(math.Ceil(math.Max(math.Abs(dx), math.Abs(dy))))
	if steps <= 0 {
		steps = 1
	}
	stepX := dx / float64(steps)
	stepY := dy / float64(steps)
	moved := 0.0
	for i := 0; i < steps; i++ {
		nx := s.px + stepX
		ny := s.py + stepY
		if !s.isWall(nx, ny) {
			s.px, s.py = nx, ny
			moved += math.Abs(stepX) + math.Abs(stepY)
		} else {
			if !s.isWall(s.px+stepX, s.py) {
				s.px += stepX
				moved += math.Abs(stepX)
			}
			if !s.isWall(s.px, s.py+stepY) {
				s.py += stepY
				moved += math.Abs(stepY)
			}
		}
	}
	return moved
}

func (s *FieldScene) checkDoorProximity() {
	px, py := s.px, s.py
	s.nearDoorEvent = false

	for _, layer := range s.tileMap.Layers {
		if !strings.HasPrefix(layer.Name, "events") {
			continue
		}
		for _, obj := range layer.Objects {
			if !objContains(obj, px, py) {
				continue
			}
			p := objProps(obj)
			targetMap := p["targetmap"]
			targetPoint := p["targetpoint"]

			if targetMap == "" {
				continue
			}

			if bossID, ok := objPropInt(obj, "requireboss"); ok && bossID > 0 && !isBossDefeated(s.game, bossID) {
				continue
			}

			s.nearDoorEvent = true
			s.pendingDoorMap = targetMap
			s.pendingDoorPoint = targetPoint
			s.pendingDoorX = s.px
			s.pendingDoorY = s.py
			s.pendingDoorDir = s.dir
			return
		}
	}
}

func (s *FieldScene) triggerDoorWarp() Scene {
	if !s.nearDoorEvent {
		return nil
	}
	nextRoom, err := NewRoomScene(s.game, s.pendingDoorMap, s.pendingDoorX, s.pendingDoorY, s.pendingDoorPoint, s.pendingDoorDir)
	if err != nil {
		return nil
	}
	s.game.Audio.PlaySEByKey("door")
	s.game.ChangeSceneWithFade(nextRoom, fadeTimeDoor)
	return s
}

// beginMessage は会話を開始する。会話データにbgmキーが指定されている場合のみ
// BGMを切り替える。指定がなければ、歩行中に流れていたBGMをそのまま継続する
// (会話のたびに毎回曲が途切れるのを避けるため)。
func (s *FieldScene) beginMessage() {
	if len(s.msgTexts) == 0 {
		// 中身の無い会話(全行が空テキストでfilterEmptyCommandsに落とされた
		// プレースホルダー等)。メッセージ欄を開いてもmsgTexts[0]に何も無く
		// 進行できなくなるので、そもそも開かない。
		return
	}
	s.msg.Reset()
	s.isMsgActive = true
	s.msg.Start()
	if s.msgBGM != "" {
		if path, ok := resolveBGMKey(s.msgBGM); ok {
			s.game.Audio.FadeOutThenPlay(path, msgBGMFadeOut, msgBGMFadeIn, false)
		}
	}
}

func (s *FieldScene) endMessage() {
	s.isMsgActive = false
	s.msgTexts = nil
	s.msgIndex = 0
	s.msgBGM = ""
	s.msgBackground = ""
	s.autoMode = false
	s.autoWaitElapsed = 0
	s.nearExamineEvent = false
	s.game.Audio.FadeOutThenPlay(s.mapBGM, msgBGMFadeOut, msgBGMFadeIn, false)
}

// 会話専用BGM(msgBGM)は画面の暗転を伴わないため、音だけで短くクロスフェード
// させて曲の切り替わりを自然に見せる。
const (
	msgBGMFadeOut = 0.4
	msgBGMFadeIn  = 0.6
)

const (
	autoBaseSeconds    = 0.4
	autoPerCharSeconds = 0.06
	msgLogMaxEntries   = 30
)

func autoWaitDuration(cmd EventCommand) float64 {
	return autoBaseSeconds + float64(len([]rune(cmd.Text)))*autoPerCharSeconds
}

func (s *FieldScene) appendMessageLog(cmd EventCommand) {
	if cmd.Speaker == "SYSTEM_COMMAND" || cmd.Text == "" {
		return
	}
	s.msgLog = append(s.msgLog, cmd)
	if len(s.msgLog) > msgLogMaxEntries {
		s.msgLog = s.msgLog[len(s.msgLog)-msgLogMaxEntries:]
	}
}

func (s *FieldScene) advanceMessage() Scene {
	s.appendMessageLog(s.msgTexts[s.msgIndex])

	s.msgIndex++
	if s.msgIndex < len(s.msgTexts) {
		cmd := s.msgTexts[s.msgIndex]
		if cmd.Speaker == "SYSTEM_COMMAND" && strings.HasPrefix(cmd.Text, "START_BATTLE_") {
			bossType := strings.TrimPrefix(cmd.Text, "START_BATTLE_")

			s.isMsgActive = false
			s.msgTexts = nil
			s.msgIndex = 0
			s.autoMode = false

			battleScene := NewBattleScene(s.game, s.currentMap, s.px, s.py, s.dir, bossType, nil)
			s.game.ChangeSceneWithFade(battleScene, fadeTimeBossIn)
			return s
		}
		if cmd.Speaker == "SYSTEM_COMMAND" && cmd.Text == "START_ENDING" {
			s.isMsgActive = false
			s.msgTexts = nil
			s.msgIndex = 0
			s.autoMode = false
			thumb := ebiten.NewImage(gameWidth, gameHeight)
			s.Draw(thumb)
			s.game.captureMenuEntryThumb(thumb)

			endingScene := NewEndingScene(s.game, s)
			s.game.ChangeSceneWithFade(endingScene, fadeTimeBossOut)
			return s
		}
		s.msg.Start()
	}

	if s.msgIndex >= len(s.msgTexts) {
		if s.onChoiceConfirm != nil {
			s.autoMode = false
			s.isChoiceActive = true
			s.choiceIndex = 0
		} else {
			s.endMessage()
		}
	}
	return s
}

func (s *FieldScene) skipMessage() Scene {
	s.autoMode = false
	for _, cmd := range s.msgTexts {
		if cmd.Speaker != "SYSTEM_COMMAND" {
			continue
		}
		if strings.HasPrefix(cmd.Text, "START_BATTLE_") {
			bossType := strings.TrimPrefix(cmd.Text, "START_BATTLE_")
			s.isMsgActive = false
			s.msgTexts = nil
			s.msgIndex = 0
			battleScene := NewBattleScene(s.game, s.currentMap, s.px, s.py, s.dir, bossType, nil)
			s.game.ChangeSceneWithFade(battleScene, fadeTimeBossIn)
			return s
		}
		if cmd.Text == "START_ENDING" {
			s.isMsgActive = false
			s.msgTexts = nil
			s.msgIndex = 0
			thumb := ebiten.NewImage(gameWidth, gameHeight)
			s.Draw(thumb)
			s.game.captureMenuEntryThumb(thumb)
			endingScene := NewEndingScene(s.game, s)
			s.game.ChangeSceneWithFade(endingScene, fadeTimeBossOut)
			return s
		}
	}
	if s.onChoiceConfirm != nil {
		s.isMsgActive = false
		s.isChoiceActive = true
		s.choiceIndex = 0
	} else {
		s.endMessage()
	}
	return s
}
