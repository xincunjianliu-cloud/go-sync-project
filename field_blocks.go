package main

import (
	"math"
	"math/rand"
	"strings"
)

type FieldBlock struct {
	ID            string
	X, Y          float64
	Width, Height float64
}

func blockKey(mapPath, id string) string {
	return mapPath + "|" + id
}

const blockNearMargin = chestTriggerMargin

func blockNear(b *FieldBlock, px, py float64) bool {
	return px >= b.X-blockNearMargin && px <= b.X+b.Width+blockNearMargin &&
		py >= b.Y-blockNearMargin && py <= b.Y+b.Height+blockNearMargin
}

func (s *FieldScene) findBlockByID(id string) *FieldBlock {
	for _, b := range s.blocks {
		if b.ID == id {
			return b
		}
	}
	return nil
}

func (s *FieldScene) beginPushingBlock(id string) {
	s.isPushingBlock = true
	s.pushingBlockID = id
	s.blockStepActive = false
	s.blockStepElapsed = 0
	s.nearExamineEvent = false
	s.nearBlockID = ""
}

func (s *FieldScene) endPushingBlock() {
	if s.blockStepActive {
		s.finishBlockStep()
	}
	s.isPushingBlock = false
	s.pushingBlockID = ""
	s.updateBlockDoors()
}

const blockStepDuration = 0.16

func blockStepVector(dir int) (dx, dy float64) {
	switch dir {
	case 0:
		return 0, 1
	case 1:
		return -1, 0
	case 2:
		return 1, 0
	case 3:
		return 0, -1
	}
	return 0, 0
}

func (s *FieldScene) updateBlockPush(dt float64, inputHeld bool) float64 {
	block := s.findBlockByID(s.pushingBlockID)
	if block == nil {
		s.endPushingBlock()
		return 0
	}

	if s.blockStepActive {
		prevPX, prevPY := s.px, s.py
		s.blockStepElapsed += dt
		t := s.blockStepElapsed / blockStepDuration
		if t >= 1 {
			s.finishBlockStep()
		} else {
			s.px = lerp(s.blockStepStartPX, s.blockStepTargetPX, t)
			s.py = lerp(s.blockStepStartPY, s.blockStepTargetPY, t)
			block.X = lerp(s.blockStepStartBX, s.blockStepTargetBX, t)
			block.Y = lerp(s.blockStepStartBY, s.blockStepTargetBY, t)
		}
		return math.Hypot(s.px-prevPX, s.py-prevPY)
	}

	if !inputHeld {
		return 0
	}

	stepX, stepY := blockStepVector(s.dir)
	if stepX == 0 && stepY == 0 {
		return 0
	}

	tileW := float64(s.tileMap.TileWidth)
	tileH := float64(s.tileMap.TileHeight)
	if tileW <= 0 {
		tileW = 32
	}
	if tileH <= 0 {
		tileH = 32
	}

	targetPX := s.px + stepX*tileW
	targetPY := s.py + stepY*tileH
	targetBX := block.X + stepX*tileW
	targetBY := block.Y + stepY*tileH

	pLeft, pTop, pRight, pBottom := s.collisionRectAt(targetPX, targetPY)
	blockBlocked := s.rectHitsObstacles(targetBX, targetBY, targetBX+block.Width, targetBY+block.Height, block.ID)
	playerBlocked := s.rectHitsObstacles(pLeft, pTop, pRight, pBottom, block.ID)
	if blockBlocked || playerBlocked {
		return 0
	}

	s.blockStepActive = true
	s.blockStepElapsed = 0
	s.blockStepStartPX, s.blockStepStartPY = s.px, s.py
	s.blockStepTargetPX, s.blockStepTargetPY = targetPX, targetPY
	s.blockStepStartBX, s.blockStepStartBY = block.X, block.Y
	s.blockStepTargetBX, s.blockStepTargetBY = targetBX, targetBY
	return 0
}

func (s *FieldScene) finishBlockStep() {
	block := s.findBlockByID(s.pushingBlockID)
	s.px, s.py = s.blockStepTargetPX, s.blockStepTargetPY
	s.blockStepActive = false
	if block == nil {
		return
	}
	block.X, block.Y = s.blockStepTargetBX, s.blockStepTargetBY
	if s.game.BlockPositions == nil {
		s.game.BlockPositions = make(map[string][2]float64)
	}
	s.game.BlockPositions[blockKey(s.currentMap, block.ID)] = [2]float64{block.X, block.Y}
}

func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

func (s *FieldScene) findBlockSpot(spotID string) (TiledObject, bool) {
	for _, layer := range s.tileMap.Layers {
		if !strings.HasPrefix(layer.Name, "events") {
			continue
		}
		for _, obj := range layer.Objects {
			p := objProps(obj)
			if isBlockSpotObj(p) && p["id"] == spotID {
				return obj, true
			}
		}
	}
	return TiledObject{}, false
}

const blockSpotTolerance = 4.0

func blockOnSpot(b *FieldBlock, spot TiledObject) bool {
	return math.Abs(b.X-spot.X) <= blockSpotTolerance && math.Abs(b.Y-spot.Y) <= blockSpotTolerance
}

func (s *FieldScene) blockSpotFilled(spotID string) bool {
	spot, ok := s.findBlockSpot(spotID)
	if !ok {
		return false
	}
	for _, b := range s.blocks {
		if blockOnSpot(b, spot) {
			return true
		}
	}
	return false
}

func (s *FieldScene) blockDoorIsOpen(obj TiledObject) bool {
	return s.game.UnlockedBlockDoors[chestKey(s.currentMap, obj)]
}

func (s *FieldScene) spotBelongsToOpenDoor(spotID string) bool {
	for _, layer := range s.tileMap.Layers {
		if !strings.HasPrefix(layer.Name, "events") {
			continue
		}
		for _, obj := range layer.Objects {
			p := objProps(obj)
			if !isBlockDoorObj(p) || !s.blockDoorIsOpen(obj) {
				continue
			}
			for _, id := range splitKeyNames(p["spots"]) {
				if id == spotID {
					return true
				}
			}
		}
	}
	return false
}

// blockIsLocked reports whether a block has already unlocked its door by
// sitting on its spot, and should no longer be pushable.
func (s *FieldScene) blockIsLocked(b *FieldBlock) bool {
	for _, layer := range s.tileMap.Layers {
		if !strings.HasPrefix(layer.Name, "events") {
			continue
		}
		for _, obj := range layer.Objects {
			p := objProps(obj)
			if !isBlockSpotObj(p) {
				continue
			}
			if !blockOnSpot(b, obj) {
				continue
			}
			if spotID := p["id"]; spotID != "" && s.spotBelongsToOpenDoor(spotID) {
				return true
			}
		}
	}
	return false
}

func (s *FieldScene) updateBlockDoors() {
	for _, layer := range s.tileMap.Layers {
		if !strings.HasPrefix(layer.Name, "events") {
			continue
		}
		for _, obj := range layer.Objects {
			p := objProps(obj)
			if !isBlockDoorObj(p) {
				continue
			}
			if s.blockDoorIsOpen(obj) {
				continue
			}

			spotIDs := splitKeyNames(p["spots"])
			if len(spotIDs) == 0 {
				continue
			}

			allFilled := true
			for _, spotID := range spotIDs {
				if !s.blockSpotFilled(spotID) {
					allFilled = false
					break
				}
			}
			if !allFilled {
				continue
			}

			if s.game.UnlockedBlockDoors == nil {
				s.game.UnlockedBlockDoors = make(map[string]bool)
			}
			s.game.UnlockedBlockDoors[chestKey(s.currentMap, obj)] = true
			s.triggerDoorOpenShake()
		}
	}
}

const (
	doorOpenShakeDuration = 0.4
	doorOpenShakePower    = 6.0
)

func (s *FieldScene) triggerDoorOpenShake() {
	s.screenShakeTimer = doorOpenShakeDuration
	s.screenShakeMaxDur = doorOpenShakeDuration
	s.screenShakePower = doorOpenShakePower
}

func (s *FieldScene) updateScreenShake(dt float64) {
	if s.screenShakeTimer <= 0 {
		return
	}
	s.screenShakeTimer -= dt
	if s.screenShakeTimer <= 0 {
		s.screenShakeTimer = 0
		s.screenShakeX = 0
		s.screenShakeY = 0
		return
	}
	progress := s.screenShakeTimer / s.screenShakeMaxDur
	bellCurve := math.Sin(progress * math.Pi)
	power := s.screenShakePower * bellCurve
	s.screenShakeX = (rand.Float64()*2.0 - 1.0) * power
	s.screenShakeY = (rand.Float64()*2.0 - 1.0) * power
}
