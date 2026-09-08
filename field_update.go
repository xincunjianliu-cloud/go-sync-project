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

// openMessageLog は会話ログ画面を開く。開くたびに、スクロール位置・カーソルは
// どちらも「一番新しい（一番下の）ログ」を指すようにリセットする。
func (s *FieldScene) openMessageLog() {
	s.isLogActive = true
	s.logScrollOffset = 0
	s.logCursorIndex = len(s.msgLog) - 1
	if s.logCursorIndex < 0 {
		s.logCursorIndex = 0
	}
}

// moveLogCursor はログ画面のカーソルを delta 件分（-1で古い方、+1で新しい方）動かし、
// カーソルが表示中の4件からはみ出す場合はスクロール位置も追従させる。
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

	// 現在のスクロール位置での表示範囲 [startIndex, endIndex-1] を再現する。
	endIndex := n - int(s.logScrollOffset)
	startIndex := endIndex - logVisibleCount
	if startIndex < 0 {
		startIndex = 0
	}

	if s.logCursorIndex < startIndex {
		// カーソルが表示範囲より上（古い方）にはみ出した：スクロールして追従
		s.logScrollOffset = float64(n - s.logCursorIndex - logVisibleCount)
		if s.logScrollOffset < 0 {
			s.logScrollOffset = 0
		}
	} else if s.logCursorIndex > endIndex-1 {
		// カーソルが表示範囲より下（新しい方）にはみ出した：スクロールして追従
		s.logScrollOffset = float64(n - s.logCursorIndex - 1)
		if s.logScrollOffset < 0 {
			s.logScrollOffset = 0
		}
	}
}

func (s *FieldScene) Update(dt float64) Scene {
	if s == nil {
		return s
	}

	// -------------------------------------------------------------------------
	// 会話ログ画面を開いている間は、閉じる操作以外を受け付けない
	// -------------------------------------------------------------------------
	if s.isLogActive {
		if isMenuUpPressed() {
			s.moveLogCursor(-1)
		}
		if isMenuDownPressed() {
			s.moveLogCursor(1)
		}
		if isConfirmKeyPressed() || isEscapePressed() || isLogTogglePressed() {
			s.isLogActive = false
		}
		return s
	}

	currentSpeaker := ""
	if s.isMsgActive && s.msgIndex < len(s.msgTexts) {
		currentSpeaker = s.msgTexts[s.msgIndex].Speaker
		s.msg.Speed = s.game.MessageSpeedTicks()
		s.msg.Tick()
		s.msg.UpdateCharaAnim(currentSpeaker, s.game.CharaImgs)
	}
	// ★追加：選択肢（はい/いいえ）操作中の処理
	if s.isChoiceActive {
		if isMenuUpPressed() || isMenuDownPressed() {
			s.choiceIndex = 1 - s.choiceIndex
		}
		if isConfirmKeyPressed() {
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

	// -------------------------------------------------------------------------
	// ボスを倒した直後のクリアダイアログ割り込み処理
	// -------------------------------------------------------------------------
	if s.justDefeatedBoss > 0 {
		cd := bossClearDialogues[s.justDefeatedBoss]
		s.msgTexts = cd.Commands
		s.msg.SpeakerToSlot = cd.SpeakerSlots
		s.msgIndex = 0
		s.beginMessage() // ← この中でReset()される
		s.justDefeatedBoss = 0
		return s
	}

	// -------------------------------------------------------------------------
	// カットシーン（自動移動イベントなど）実行中の更新処理
	// -------------------------------------------------------------------------
	if s.isCutscene || s.cseFadeMode == 2 || s.cseFadeMode == 3 || s.cseFadeMode == 4 {
		// フェードアウト中
		if s.cseFadeMode == 2 {
			s.cseFadeAlpha += s.cseFadeSpeed * dt
			if s.cseFadeAlpha >= 1.0 {
				s.cseFadeAlpha = 1.0
				s.cseFadeMode = 3 // フェードイン待ちへ
				if s.onDarkCallback != nil {
					cb := s.onDarkCallback
					s.onDarkCallback = nil
					cb()
				}

			}
		}

		// フェードイン中
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
					s.cseFadeSpeed = 1.0 / 0.4 // 従来のトリガー移動用デフォルト
				}
			}
		}

		// 移動処理
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

			// 歩き始めてN秒後にフェードアウト開始
			if s.cseFadeMode == 1 && s.cseElapsed >= s.CseFadeOutDelay {
				s.cseFadeMode = 2
				s.cseFadeSpeed = 1.0 / 0.4
			}

			// 移動完了
			if s.routeIndex >= len(s.cutsceneRoute) {
				s.isCutscene = false
				s.animCount = 0

				s.px = math.Floor(s.px/16) * 16
				s.py = math.Floor(s.py/16) * 16
				if s.cutsceneMessage != "" {
					bd := GetEventCommands(s.cutsceneMessage, s.game)
					s.msgTexts = bd.Commands
					bossType := strings.TrimPrefix(s.cutsceneMessage, "event_")
					s.msgTexts = append(s.msgTexts, EventCommand{
						Speaker: "SYSTEM_COMMAND",
						Text:    "START_BATTLE_" + bossType,
					})
					s.msg.SpeakerToSlot = bd.SpeakerSlots
					s.msgIndex = 0
				}

				// 移動完了したら暗転待機へ（まだ暗転中でなければ強制暗転）
				if s.cseFadeMode == 0 {
					s.beginMessage()
				} else if s.cseFadeMode != 4 {
					// 暗転待機をリセットして待機継続
					s.cseDarkElapsed = 0
				}
			}
		}
		return s
	}

	// -------------------------------------------------------------------------
	// メッセージ（会話）ウィンドウ表示中の更新処理
	// -------------------------------------------------------------------------
	if s.isMsgActive {
		s.msg.Speed = s.game.MessageSpeedTicks()

		if isSkipKeyDown() {
			s.msgSkipHoldElapsed += dt
			if s.msgSkipHoldElapsed >= endingSkipHoldSeconds {
				s.msgSkipHoldElapsed = 0
				return s.skipMessage()
			}
		} else {
			s.msgSkipHoldElapsed = 0
		}

		// オートON/OFFの切り替え
		if isAutoTogglePressed() {
			s.autoMode = !s.autoMode
			s.autoWaitElapsed = 0
		}

		// 会話ログ画面を開く
		if isLogTogglePressed() {
			s.openMessageLog()
			return s
		}

		finished := s.msg.IsFinished(s.msgTexts[s.msgIndex])

		if isConfirmKeyPressed() {
			// テキストがまだ表示中なら全文スキップ
			if !finished {
				s.msg.SkipToEnd(s.msgTexts[s.msgIndex])
				return s
			}
			// 表示完了済みなら次のページへ
			return s.advanceMessage()
		}

		// オート送り：表示完了後、文字数に応じた待機時間が経過したら自動で次へ
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
		// ★追加：ドア（ワープ）が優先。近くにいれば遷移して終了。
		if s.nearDoorEvent {
			if next := s.triggerDoorWarp(); next != nil {
				return next
			}
		}

		// ★変更：向いている方向へのオフセットを廃止。
		// 壁に貼るのではなく床に置いたイベントに「乗ったらEnterで反応」する形にする。
		for _, layer := range s.tileMap.Layers {
			if !strings.HasPrefix(layer.Name, "events") {
				continue
			}
			for _, obj := range layer.Objects {
				if !objContains(obj, playerFootX, playerFootY) {
					continue
				}
				p := objProps(obj)
				evType := p["type"]
				evText := p["text"]

				if evType == "event" && !strings.HasPrefix(evText, "event_boss_") {
					if evText == "event_rest" {
						s.startRestEvent(obj)
						return s
					} else if evText != "" {
						if strings.HasPrefix(evText, "event_story_") {
							d := GetEventCommands(evText, s.game)
							s.msgTexts = d.Commands
							s.msg.SpeakerToSlot = d.SpeakerSlots
						} else {
							pages := strings.Split(evText, "|")
							s.msgTexts = []EventCommand{}
							for _, pageText := range pages {
								s.msgTexts = append(s.msgTexts, EventCommand{
									Speaker: "",
									Text:    pageText,
								})
							}
						}
					} else {
						s.msgTexts = []EventCommand{{Speaker: "", Text: "調べるとなにかあるかもしれない"}}
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

	perFrame := s.playerCfg.MoveSpeed * dt
	if s.game.IsDashing {
		perFrame *= s.playerCfg.DashSpeedMultiplier
	}
	moveX, moveY := 0.0, 0.0
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyUp) {
		moveY = -perFrame
		s.dir = 3
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyDown) {
		moveY = perFrame
		s.dir = 0
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyLeft) {
		moveX = -perFrame
		s.dir = 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyRight) {
		moveX = perFrame
		s.dir = 2
	}

	prevPx, prevPy := s.px, s.py

	actualMovedDist := 0.0
	if moveX != 0 && moveY != 0 {
		actualMovedDist += s.moveBoth(moveX, moveY)
	} else {
		actualMovedDist += s.moveAxis(moveX, true)
		actualMovedDist += s.moveAxis(moveY, false)
	}

	// エンカウント判定用の実移動距離（斜め移動時の二重加算を防ぐため、直線距離で算出）
	encounterMovedDist := math.Hypot(s.px-prevPx, s.py-prevPy)
	if s.isWall(s.px, s.py) {
		s.resolveEmbeddedPosition()
	}

	if actualMovedDist > 0 {
		s.animCount++
	} else {
		s.animCount = 0
	}
	moved := actualMovedDist > 0

	s.checkDoorProximity()
	s.updateObjectiveGuide()

	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		full := ebiten.NewImage(gameWidth, gameHeight)
		s.Draw(full)
		s.game.captureMenuEntryThumb(full)
		return NewMenuScene(s.game, s)
	}

	if isLogTogglePressed() {
		s.openMessageLog()
		return s
	}

	var targetEnemiesStr string
	s.nearExamineEvent = false

	for _, layer := range s.tileMap.Layers {
		if !strings.HasPrefix(layer.Name, "events") {
			continue
		}
		for _, obj := range layer.Objects {
			p := objProps(obj)
			evType := p["type"]
			evText := p["text"]
			routeStr := p["route"]

			if evType == "event" && objContains(obj, playerFootX, playerFootY) {
				s.nearExamineEvent = true
			}

			if evType == "trigger" {
				bossID, hasBossID := objPropInt(obj, "bossid")
				instant, _ := objPropBool(obj, "instant")

				if hasBossID && isBossDefeated(s.game, bossID) {
					continue
				}

				if objContains(obj, playerFootX, playerFootY) {

					if hasBossID {
						s.pendingCutsceneMsg = fmt.Sprintf("event_boss_%d", bossID)
					} else {
						s.pendingCutsceneMsg = evText
					}

					if instant || routeStr == "" || routeStr == "<nil>" {
						s.cutsceneMessage = s.pendingCutsceneMsg
						s.cseFadeAlpha = 0
						s.cseFadeMode = 2
						s.cseFadeSpeed = 1.0 / 0.4
						s.onDarkCallback = nil
						s.onFadeCompleteCallback = func() {
							bd := GetEventCommands(s.cutsceneMessage, s.game)
							s.msgTexts = bd.Commands
							bossType := strings.TrimPrefix(s.cutsceneMessage, "event_")
							s.msgTexts = append(s.msgTexts, EventCommand{
								Speaker: "SYSTEM_COMMAND",
								Text:    "START_BATTLE_" + bossType,
							})
							s.msg.SpeakerToSlot = bd.SpeakerSlots
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
					s.cseFadeMode = 1 // 移動開始、フェードアウト待ち
					return s
				}
			}

			if evType == "enemy" && objContains(obj, playerFootX, playerFootY) {
				targetEnemiesStr = evText
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
					chosenEnemyName := strings.TrimSpace(allowedEnemies[rand.Intn(len(allowedEnemies))])
					// ← s.dir を渡す
					battleScene := NewBattleScene(s.game, s.currentMap, s.px, s.py, s.dir, "enemy", chosenEnemyName)
					s.game.ChangeSceneWithFade(battleScene, fadeTimeBattleIn)
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

func rectsOverlap(ax1, ay1, ax2, ay2, bx1, by1, bx2, by2 float64) bool {
	return ax1 < bx2 && ax2 > bx1 && ay1 < by2 && ay2 > by1
}

func (s *FieldScene) collisionRectAt(x, y float64) (left, top, right, bottom float64) {
	return s.playerCfg.CollisionRectAt(x, y)
}

func (s *FieldScene) isWall(x, y float64) bool {
	if s.tileMap.TileWidth == 0 || s.tileMap.TileHeight == 0 {
		return true
	}

	left, top, right, bottom := s.collisionRectAt(x, y)
	// --- タイルレイヤー(kabe)によるチェック ---
	checkPoints := [][2]float64{{left, top}, {right, top}, {left, bottom}, {right, bottom}}
	for _, pt := range checkPoints {
		tileX := int(pt[0] / float64(s.tileMap.TileWidth))
		tileY := int(pt[1] / float64(s.tileMap.TileHeight))
		if tileX < 0 || tileY < 0 || tileX >= s.tileMap.Width || tileY >= s.tileMap.Height {
			return true
		}
		index := tileY*s.tileMap.Width + tileX
		for _, layer := range s.tileMap.Layers {
			if layer.Name == "kabe" && index >= 0 && index < len(layer.Data) && layer.Data[index] != 0 {
				return true
			}
		}
	}

	// --- Tiledで置いたオブジェクト(collisionレイヤー)によるチェック ---
	for _, r := range s.collisions {
		if rectsOverlap(left, top, right, bottom, r.X, r.Y, r.X+r.Width, r.Y+r.Height) {
			return true
		}
	}

	// ★追加：斜めの壁など多角形オブジェクトによるチェック。
	// 判定用の矩形(left,top,right,bottom)の4隅のいずれかが多角形の内側にあれば壁とみなす。
	for _, poly := range s.collisionPolygons {
		for _, pt := range checkPoints {
			if pointInPolygon(pt[0], pt[1], poly.Points) {
				return true
			}
		}
	}

	return false
}

// pointInPolygon は点(x,y)が多角形poly内にあるかをレイキャスト法で判定する。
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

// checkDoor は足元がドア範囲内かを判定し、pendingDoor情報をセットするだけにする。
// 実際の遷移は Update 内で決定キーが押されたときに行う。
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

			if targetMap != "" {
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
}

// triggerDoorWarp は決定キーが押されたときに実際にマップ遷移を行う
func (s *FieldScene) triggerDoorWarp() Scene {
	if !s.nearDoorEvent {
		return nil
	}
	nextRoom, err := NewRoomScene(s.game, s.pendingDoorMap, s.pendingDoorX, s.pendingDoorY, s.pendingDoorPoint, s.pendingDoorDir)
	if err != nil {
		return nil
	}
	s.game.ChangeSceneWithFade(nextRoom, fadeTimeDoor)
	return s
}

// 会話を開始する：メッセージウィンドウを開き、会話BGMに切り替える
func (s *FieldScene) beginMessage() {
	s.msg.Reset() // ← 追加：新しい会話を開始するたびに立ち絵状態をリセット
	s.isMsgActive = true
	s.msg.Start()
	s.game.Audio.PlayBGM(bgmMessage)
}

// 会話を終了する：メッセージウィンドウを閉じ、フィールドBGMに戻す
func (s *FieldScene) endMessage() {
	s.isMsgActive = false
	s.msgTexts = nil
	s.msgIndex = 0
	s.autoMode = false        // 🔥 追加：会話ごとにオートはリセットする
	s.autoWaitElapsed = 0
	s.nearExamineEvent = false // ★追加：会話終了時にプロンプト状態をリセット
	s.game.Audio.PlayBGM(bgmFieldSchool)
}

// --- オート送り・ログ関連 ---

const (
	autoBaseSeconds    = 0.4  // オート時、表示完了後の最低待機秒数
	autoPerCharSeconds = 0.06 // オート時、1文字あたりの追加待機秒数
	msgLogMaxEntries   = 30   // 会話ログの最大保持件数
)

// autoWaitDuration はオート送りの待機時間を文字数に比例して算出する。
func autoWaitDuration(cmd EventCommand) float64 {
	return autoBaseSeconds + float64(len([]rune(cmd.Text)))*autoPerCharSeconds
}

// appendMessageLog は表示し終えたページをログに追加する。
// SYSTEM_COMMAND（戦闘開始・エンディング開始などの内部制御コマンド）は記録しない。
func (s *FieldScene) appendMessageLog(cmd EventCommand) {
	if cmd.Speaker == "SYSTEM_COMMAND" || cmd.Text == "" {
		return
	}
	s.msgLog = append(s.msgLog, cmd)
	if len(s.msgLog) > msgLogMaxEntries {
		s.msgLog = s.msgLog[len(s.msgLog)-msgLogMaxEntries:]
	}
}

// advanceMessage は現在のページの表示が完了した状態で「次へ進む」処理をまとめたもの。
// 決定キーによる手動送りと、オート送りの両方から呼ばれる。
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
			s.autoMode = false // 🔥 追加：戦闘に入るのでオートは解除

			// ← s.dir を渡す
			battleScene := NewBattleScene(s.game, s.currentMap, s.px, s.py, s.dir, bossType, "")
			s.game.ChangeSceneWithFade(battleScene, fadeTimeBossIn)
			return s
		}
		if cmd.Speaker == "SYSTEM_COMMAND" && cmd.Text == "START_ENDING" {
			s.isMsgActive = false
			s.msgTexts = nil
			s.msgIndex = 0
			s.autoMode = false // 🔥 追加：エンディングに入るのでオートは解除

			endingScene := NewEndingScene(s.game, s)
			s.game.ChangeSceneWithFade(endingScene, fadeTimeBossOut)
			return s
		}
		s.msg.Start() // 次のページのタイプライターをリセット
	}

	if s.msgIndex >= len(s.msgTexts) {
		if s.onChoiceConfirm != nil {
			s.autoMode = false // 🔥 追加：選択肢に入るのでオートは解除（誤送り防止）
			s.isChoiceActive = true
			s.choiceIndex = 0
		} else {
			s.endMessage()
		}
	}
	return s
}

// skipMessage は会話を長押しスキップしたときに呼ばれる。
// 会話の途中にSYSTEM_COMMAND（戦闘開始やエンディング開始）が含まれていれば
// それを即座に実行し、無ければ会話をそのまま終了する。
func (s *FieldScene) skipMessage() Scene {
	s.autoMode = false // 🔥 追加：長押しスキップした場合もオートは解除
	for _, cmd := range s.msgTexts {
		if cmd.Speaker != "SYSTEM_COMMAND" {
			continue
		}
		if strings.HasPrefix(cmd.Text, "START_BATTLE_") {
			bossType := strings.TrimPrefix(cmd.Text, "START_BATTLE_")
			s.isMsgActive = false
			s.msgTexts = nil
			s.msgIndex = 0
			battleScene := NewBattleScene(s.game, s.currentMap, s.px, s.py, s.dir, bossType, "")
			s.game.ChangeSceneWithFade(battleScene, fadeTimeBossIn)
			return s
		}
		if cmd.Text == "START_ENDING" {
			s.isMsgActive = false
			s.msgTexts = nil
			s.msgIndex = 0
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
