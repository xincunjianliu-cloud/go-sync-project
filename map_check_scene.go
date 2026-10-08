package main

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// mapCheckScene は Tiled の F5(試し遊び)で、始める前にマップのまちがいを
// 一覧で見せる画面。決定キーかクリックで、そのまま遊び始める(作りかけでも
// 遊べるように止めはしない。Push すると GitHub のチェックで止まる)。
type mapCheckScene struct {
	game   *Game
	lines  []mapCheckLine
	scroll int
	start  func()
	done   bool
}

type mapCheckLine struct {
	text   string
	header bool
}

const (
	mapCheckFontSize = 16.0
	mapCheckLineH    = 22.0
	mapCheckMarginX  = 32.0
	mapCheckTop      = 84.0
	mapCheckBottom   = 56.0
)

// newMapCheckScene は、開いているマップ(openMap)のまちがいを先に、
// ほかのマップのまちがいを後に並べる。
func newMapCheckScene(g *Game, openMap string, issues []mapIssue, start func()) *mapCheckScene {
	s := &mapCheckScene{game: g, start: start}
	face := g.FontFace(mapCheckFontSize)
	maxW := float64(gameWidth) - mapCheckMarginX*2 - 16
	addGroup := func(title string, own bool) {
		first := true
		for _, is := range issues {
			if (is.Path == openMap) != own {
				continue
			}
			if first {
				s.lines = append(s.lines, mapCheckLine{text: title, header: true})
				first = false
			}
			msg := shortMapPaths(is.Msg)
			if !own {
				msg = shortMapPaths(is.Path) + ": " + msg
			}
			for i, l := range wrapByWidth(msg, face, maxW) {
				prefix := "・"
				if i > 0 {
					prefix = "　"
				}
				s.lines = append(s.lines, mapCheckLine{text: prefix + l})
			}
		}
	}
	addGroup("このマップ("+shortMapPaths(openMap)+")", true)
	addGroup("ほかのマップ・タイルセット", false)
	return s
}

// wrapByWidth は文を、表示の幅に収まるように折り返す。
func wrapByWidth(s string, face text.Face, maxW float64) []string {
	var out []string
	line := ""
	for _, r := range s {
		next := line + string(r)
		if line != "" && text.Advance(next, face) > maxW {
			out = append(out, line)
			next = string(r)
		}
		line = next
	}
	return append(out, line)
}

func (s *mapCheckScene) visibleLines() int {
	h := float64(gameHeight) - mapCheckTop - mapCheckBottom
	return int(h / mapCheckLineH)
}

func (s *mapCheckScene) Update(dt float64) Scene {
	if s.done {
		return s
	}
	maxScroll := max(len(s.lines)-s.visibleLines(), 0)
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyPageDown):
		s.scroll = min(s.scroll+3, maxScroll)
	case inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyPageUp):
		s.scroll = max(s.scroll-3, 0)
	}
	if _, wy := ebiten.Wheel(); wy != 0 {
		if wy < 0 {
			s.scroll = min(s.scroll+2, maxScroll)
		} else {
			s.scroll = max(s.scroll-2, 0)
		}
	}
	if isConfirmKeyPressed() || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		s.done = true
		s.start()
	}
	return s
}

func (s *mapCheckScene) Draw(screen *ebiten.Image) {
	fillRect(screen, 0, 0, float64(gameWidth), float64(gameHeight), color.NRGBA{24, 20, 28, 255})

	issueCount := 0
	for _, l := range s.lines {
		if !l.header && strings.HasPrefix(l.text, "・") {
			issueCount++
		}
	}
	title := &text.DrawOptions{}
	title.GeoM.Translate(mapCheckMarginX, 24)
	title.ColorScale.ScaleWithColor(color.NRGBA{255, 150, 150, 255})
	text.Draw(screen, fmt.Sprintf("マップのまちがい %d件", issueCount), s.game.FontFace(24), title)

	sub := &text.DrawOptions{}
	sub.GeoM.Translate(mapCheckMarginX, 56)
	sub.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, "このままでは GitHub のチェックで止まります。Tiled で直してから送ってください", s.game.FontFace(14), sub)

	face := s.game.FontFace(mapCheckFontSize)
	end := min(s.scroll+s.visibleLines(), len(s.lines))
	for i, l := range s.lines[s.scroll:end] {
		op := &text.DrawOptions{}
		x := mapCheckMarginX + 16
		clr := color.Color(uiColorText)
		if l.header {
			x = mapCheckMarginX
			clr = color.NRGBA{255, 220, 120, 255}
		}
		op.GeoM.Translate(x, mapCheckTop+float64(i)*mapCheckLineH)
		op.ColorScale.ScaleWithColor(clr)
		text.Draw(screen, l.text, face, op)
	}

	hint := "決定キー(Enter)・クリック: このまま遊ぶ"
	if len(s.lines) > s.visibleLines() {
		hint += "  ▲▼キー・ホイール: 続きを見る"
	}
	foot := &text.DrawOptions{}
	foot.GeoM.Translate(mapCheckMarginX, float64(gameHeight)-36)
	foot.ColorScale.ScaleWithColor(color.NRGBA{200, 200, 200, 255})
	text.Draw(screen, hint, s.game.FontFace(14), foot)
}

// shortMapPaths は画面に出すときに、マップのパスをファイル名だけにする。
func shortMapPaths(s string) string {
	return strings.ReplaceAll(s, "assets/maps/", "")
}
