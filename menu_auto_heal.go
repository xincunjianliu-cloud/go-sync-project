package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// 全快ボタン。メニューを開いた最初の画面で、パーティ一覧の下（サブ画面で
// 「全体」行が出るのと同じ場所・同じ見た目の箱）に表示し、押すとメニューで
// 使える回復スキルを自動で使って、全員のHPが満タンになるかMPが尽きるまで回復する。
const (
	// autoHealKeyGapX は箱の右に出すキー表示(H)との間隔。
	autoHealKeyGapX = 8.0
	// autoHealMaxCasts は無限ループ防止の上限（実際はMPが先に尽きる）。
	autoHealMaxCasts = 1000
)

// autoHealVisible はボタンを出すかどうか。メニューを開いた最初の画面
// （コマンド選択中）だけで表示する。
func (m *MenuScene) autoHealVisible() bool {
	return m.menuState == menuStateMain && !m.isModalMenuState()
}

func (m *MenuScene) isAutoHealPressed() bool {
	if !m.autoHealVisible() {
		return false
	}
	return inpututil.IsKeyJustPressed(ebiten.KeyH) || tapInsideRect(m.allTargetRowRect())
}

func (m *MenuScene) drawAutoHealButton(screen *ebiten.Image, statusX float64) {
	if !m.autoHealVisible() {
		return
	}
	game := m.game
	col := uiColorText
	if _, ok := game.bestAutoHealCast(); !ok {
		col = uiColorDisabled
	}

	box, y := m.allTargetRowBox(statusX)
	fillRect(screen, box.x, box.y, box.w, box.h, color.RGBA{45, 45, 55, 200})

	op := &text.DrawOptions{}
	op.GeoM.Translate(box.x+box.w/2, y)
	op.PrimaryAlign = text.AlignCenter
	op.SecondaryAlign = text.AlignCenter
	op.ColorScale.ScaleWithColor(col)
	text.Draw(screen, "全快", game.FontFace(partyNameFontSize), op)

	if !game.MobileMode {
		keyOp := &text.DrawOptions{}
		keyOp.GeoM.Translate(box.x+box.w+autoHealKeyGapX, y)
		keyOp.SecondaryAlign = text.AlignCenter
		keyOp.ColorScale.ScaleWithColor(col)
		text.Draw(screen, "H", game.FontFace(ctrlLabelFontSize), keyOp)
	}
}

// menuHealAmount はメニューで回復スキルを使ったときの回復量。
func (g *Game) menuHealAmount(caster, power int) int {
	amount := max(int(float64(g.PlayerMagicAtk[caster])*float64(power)/100.0*10), 1)
	return amount
}

type autoHealCast struct {
	caster int
	cost   int
	amount int
	target int // all のときは無視
	all    bool
	gain   int // 実際に回復するHPの合計（最大HPを超える分は含めない）
}

// bestAutoHealCast は今使える回復スキルの中から、消費MPあたりの実回復量が
// 最も多いもの（同率なら回復量が多いもの）を選ぶ。単体回復はHPの減りが
// 一番大きい味方に使う。戦闘不能の味方は回復しない。
func (g *Game) bestAutoHealCast() (autoHealCast, bool) {
	missing := func(i int) int {
		if g.PlayerHP[i] <= 0 {
			return 0
		}
		return max(g.PlayerMaxHP[i]-g.PlayerHP[i], 0)
	}
	neediest := 0
	for i := 1; i < partySize; i++ {
		if missing(i) > missing(neediest) {
			neediest = i
		}
	}
	if missing(neediest) == 0 {
		return autoHealCast{}, false
	}

	var best autoHealCast
	found := false
	consider := func(c autoHealCast) {
		if c.gain <= 0 {
			return
		}
		// gain/cost の比較を掛け算で行う（cost=0 でも割り算しない）。
		if !found || c.gain*best.cost > best.gain*c.cost ||
			(c.gain*best.cost == best.gain*c.cost && c.gain > best.gain) {
			best = c
			found = true
		}
	}

	for caster := range partySize {
		if g.PlayerHP[caster] <= 0 {
			continue
		}
		for skillIdx, sk := range g.CharacterSkills(caster) {
			if !g.IsSkillUnlocked(caster, skillIdx) {
				continue
			}
			curLv := max(g.PlayerSkillLv[caster][skillIdx], 1)
			for lv := 1; lv <= curLv && lv <= len(sk.Levels); lv++ {
				data := sk.Levels[lv-1]
				if !data.IsHeal || data.MPCost > g.PlayerMP[caster] {
					continue
				}
				if (data.Target == TargetSingle || data.Target == TargetBoth) && data.PowerSingle > 0 {
					amount := g.menuHealAmount(caster, data.PowerSingle)
					consider(autoHealCast{caster: caster, cost: data.MPCost, amount: amount,
						target: neediest, gain: min(amount, missing(neediest))})
				}
				if (data.Target == TargetAll || data.Target == TargetBoth) && data.PowerAll > 0 {
					amount := g.menuHealAmount(caster, data.PowerAll)
					gain := 0
					for i := range partySize {
						gain += min(amount, missing(i))
					}
					consider(autoHealCast{caster: caster, cost: data.MPCost, amount: amount,
						all: true, gain: gain})
				}
			}
		}
	}
	return best, found
}

func (g *Game) healPartyMember(i, amount int) {
	if g.PlayerHP[i] <= 0 {
		return
	}
	g.PlayerHP[i] = min(g.PlayerHP[i]+amount, g.PlayerMaxHP[i])
}

// AutoHealParty は回復スキルを繰り返し使って味方のHPを回復し、
// 1回でも回復したら true を返す。
func (g *Game) AutoHealParty() bool {
	healed := false
	for range autoHealMaxCasts {
		c, ok := g.bestAutoHealCast()
		if !ok {
			break
		}
		g.PlayerMP[c.caster] -= c.cost
		if c.all {
			for i := range partySize {
				g.healPartyMember(i, c.amount)
			}
		} else {
			g.healPartyMember(c.target, c.amount)
		}
		healed = true
	}
	return healed
}

func (m *MenuScene) tryAutoHeal() {
	if m.game.AutoHealParty() {
		m.game.Audio.PlaySEByKey("heal")
	} else {
		m.game.Audio.PlaySEByKey("error")
	}
}
