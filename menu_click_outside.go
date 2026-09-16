package main

func unrelatedTapPressed(consumed bool) bool {
	return !consumed && len(justPressedTouchPoints()) > 0
}

func unrelatedTapOutsideRects(rects ...tapRect) bool {
	pts := justPressedTouchPoints()
	if len(pts) == 0 {
		return false
	}
	for _, p := range pts {
		inside := false
		for _, r := range rects {
			if r.contains(p) {
				inside = true
				break
			}
		}
		if !inside {
			return true
		}
	}
	return false
}

func menuMainContentRect() tapRect {
	return tapRect{x: menuFrameDividerX, y: menuFrameTop, w: menuFrameRight - menuFrameDividerX, h: menuFrameBottom - menuFrameTop}
}
