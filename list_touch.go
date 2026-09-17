package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type tapRect struct {
	x, y, w, h float64
}

func (r tapRect) contains(p touchPoint) bool {
	return p.inRect(r.x, r.y, r.w, r.h)
}

func hitTestTapRects(rects []tapRect) (int, bool) {
	for _, p := range justPressedTouchPoints() {
		for i, r := range rects {
			if r.contains(p) {
				return i, true
			}
		}
	}
	return -1, false
}

// tapSelectOrConfirm はタップ位置の項目を選択・確定するための二段階方式の
// 共通ロジック。1回目のタップは選択のみ(カーソルがジャンプする)、既に
// 選択済みの項目への2回目のタップで確定となる。選択がジャンプした瞬間にも
// キー操作のカーソル移動と同じ選択音を鳴らす。
func tapSelectOrConfirm(tappedIdx int, tappedOk bool, currentIdx *int, audio *AudioManager) bool {
	if !tappedOk {
		return false
	}
	alreadySelected := tappedIdx == *currentIdx
	*currentIdx = tappedIdx
	if !alreadySelected {
		audio.PlaySEByKey("cursor")
	}
	return alreadySelected
}

func tapArmSelectOrConfirm(tappedIdx int, tappedOk bool, currentIdx *int, armed *bool, audio *AudioManager) bool {
	if !tappedOk {
		return false
	}
	confirm := *armed && tappedIdx == *currentIdx
	if !confirm {
		audio.PlaySEByKey("cursor")
	}
	*currentIdx = tappedIdx
	*armed = true
	return confirm
}

func tapArmButtonConfirm(pressed bool, armed *bool, audio *AudioManager) bool {
	if !pressed {
		return false
	}
	confirm := *armed
	if !confirm {
		audio.PlaySEByKey("cursor")
	}
	*armed = true
	return confirm
}

type dragScrollState struct {
	active         bool
	lastX, lastY   float64
	startMoveTotal float64

	velocity float64

	justTapped bool
	tapX, tapY float64

	suppressUntilRelease bool
}

const (
	scrollFriction   = 0.90
	tapMoveThreshold = 12.0
)

func (d *dragScrollState) update(x, y, w, h float64) float64 {
	d.justTapped = false
	delta := 0.0
	found := false
	for _, p := range activeTouchPoints() {
		if p.inRect(x, y, w, h) {
			found = true
			if d.active {
				delta = p.y - d.lastY
				d.startMoveTotal += math.Abs(delta) + math.Abs(p.x-d.lastX)
			} else {
				d.startMoveTotal = 0
			}
			d.lastX, d.lastY = p.x, p.y
			break
		}
	}
	if d.suppressUntilRelease {
		d.active = found
		if !found {
			d.suppressUntilRelease = false
		}
		return 0
	}

	wasActive := d.active
	d.active = found
	if wasActive && !found && d.startMoveTotal < tapMoveThreshold {
		d.justTapped = true
		d.tapX, d.tapY = d.lastX, d.lastY
	}
	return delta
}

func (d *dragScrollState) step(x, y, w, h float64) float64 {
	delta := d.update(x, y, w, h)
	if d.active {
		d.velocity = delta
		return delta
	}
	if d.justTapped {
		d.velocity = 0
		return 0
	}
	if math.Abs(d.velocity) < 0.4 {
		d.velocity = 0
		return 0
	}
	d.velocity *= scrollFriction
	return d.velocity
}

const (
	overscrollMax        = 32.0
	overscrollResistance = 0.35
	overscrollSpringBack = 0.75

	wheelScrollPxPerNotch = 60.0
)

func (d *dragScrollState) stepScrollTop(moveAmt, cardH float64, scrollTop *int, maxScrollTop int, accum *float64) {
	if cardH <= 0 {
		return
	}
	if moveAmt != 0 {
		atTop := *scrollTop <= 0 && *accum >= 0
		atBottom := *scrollTop >= maxScrollTop && *accum <= 0
		if atTop || atBottom {
			*accum += moveAmt * overscrollResistance
			if *accum > overscrollMax {
				*accum = overscrollMax
			} else if *accum < -overscrollMax {
				*accum = -overscrollMax
			}
		} else {
			*accum += moveAmt
			for *accum <= -cardH {
				if *scrollTop >= maxScrollTop {
					break
				}
				*scrollTop++
				*accum += cardH
			}
			for *accum >= cardH {
				if *scrollTop <= 0 {
					break
				}
				*scrollTop--
				*accum -= cardH
			}
		}
	}

	if !d.active {
		if *scrollTop <= 0 && *accum > 0 {
			*accum *= overscrollSpringBack
			if *accum < 0.5 {
				*accum = 0
			}
		} else if *scrollTop >= maxScrollTop && *accum < 0 {
			*accum *= overscrollSpringBack
			if *accum > -0.5 {
				*accum = 0
			}
		}
	}
}

func (d *dragScrollState) stepContinuousScroll(moveAmt float64, pos *float64, minPos, maxPos float64) {
	if moveAmt != 0 {
		if (*pos <= minPos && moveAmt < 0) || (*pos >= maxPos && moveAmt > 0) {
			*pos += moveAmt * overscrollResistance
			if *pos < minPos-overscrollMax {
				*pos = minPos - overscrollMax
			} else if *pos > maxPos+overscrollMax {
				*pos = maxPos + overscrollMax
			}
		} else {
			*pos += moveAmt
			if *pos < minPos {
				*pos = minPos
			} else if *pos > maxPos {
				*pos = maxPos
			}
		}
	}

	if !d.active {
		if *pos < minPos {
			*pos += (minPos - *pos) * (1 - overscrollSpringBack)
			if minPos-*pos < 0.5 {
				*pos = minPos
			}
		} else if *pos > maxPos {
			*pos -= (*pos - maxPos) * (1 - overscrollSpringBack)
			if *pos-maxPos < 0.5 {
				*pos = maxPos
			}
		}
	}
}

func centeredTextRect(centerX, y float64, label string, face *text.GoTextFace, height float64) tapRect {
	w := text.Advance(label, face)
	return tapRect{x: centerX - w/2 - 10, y: y - 4, w: w + 20, h: height}
}
