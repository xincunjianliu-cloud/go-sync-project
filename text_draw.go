package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type textRun struct {
	text  string
	latin bool
}

// isLatinRune reports whether r is in the half-width ASCII range that
// PixelMplus draws visually smaller than its full-width Japanese glyphs.
func isLatinRune(r rune) bool {
	return r < 0x80
}

// splitLatinRuns breaks s into maximal runs that are either entirely
// half-width (ASCII letters/digits/punctuation) or entirely full-width
// (Japanese text, and symbols like ▶◀ which are sized to match it).
func splitLatinRuns(s string) []textRun {
	runes := []rune(s)
	if len(runes) == 0 {
		return nil
	}
	var runs []textRun
	start := 0
	curLatin := isLatinRune(runes[0])
	for i := 1; i < len(runes); i++ {
		l := isLatinRune(runes[i])
		if l != curLatin {
			runs = append(runs, textRun{text: string(runes[start:i]), latin: curLatin})
			start = i
			curLatin = l
		}
	}
	runs = append(runs, textRun{text: string(runes[start:]), latin: curLatin})
	return runs
}

// secondaryAlignOffsetY returns the Y offset ebiten's text.Draw adds to the
// given origin for a face with metrics m under the given vertical alignment
// (mirrors the calculation in ebiten's text/v2 layout code). It's used to
// work out where a face's baseline actually lands, so that two faces with
// different metrics can be lined up on the same baseline instead of just
// the same nominal origin.
func secondaryAlignOffsetY(m text.Metrics, align text.Align) float64 {
	switch align {
	case text.AlignCenter:
		return (m.HAscent - m.HDescent) / 2
	case text.AlignEnd:
		return -m.HDescent
	default:
		return m.HAscent
	}
}

// latinBaselineAdjust returns the Y translation to add when drawing with
// LatinFontFace(size) so its baseline lines up with FontFace(size) text
// drawn at the same y and secondaryAlign. LatinFontFace asks for a larger
// nominal Size to compensate for PixelMplus drawing Latin glyphs visually
// smaller, but a larger Size also means a taller ascent/descent, which by
// itself would shift the Latin baseline away from Japanese text sharing the
// same y (e.g. drop it further below the top for AlignStart) instead of
// just making the glyphs bigger in place. This cancels that shift out.
func (g *Game) latinBaselineAdjust(size float64, align text.Align) float64 {
	ref := secondaryAlignOffsetY(g.FontFace(size).Metrics(), align)
	latin := secondaryAlignOffsetY(g.LatinFontFace(size).Metrics(), align)
	return ref - latin
}

// DrawMixedText draws str at the given nominal size, automatically switching
// to LatinFontFace for any ASCII run inside it, so a string that mixes
// Japanese and English/numbers (e.g. "MP:5" inside a Japanese sentence)
// reads at a consistent visual size instead of the ASCII part looking
// smaller, and sits on the same baseline instead of the enlarged glyphs
// dropping below the Japanese text. primaryAlign/secondaryAlign apply to
// the string as a whole, same as the fields on text.DrawOptions. Pure
// Japanese or pure ASCII strings still render at the position FontFace/
// LatinFontFace would put them relative to a same-y FontFace baseline.
func (g *Game) DrawMixedText(dst *ebiten.Image, str string, size, x, y float64, primaryAlign, secondaryAlign text.Align, col color.Color) {
	runs := splitLatinRuns(str)

	if len(runs) <= 1 {
		face := g.FontFace(size)
		drawY := y
		if len(runs) == 1 && runs[0].latin {
			face = g.LatinFontFace(size)
			drawY += g.latinBaselineAdjust(size, secondaryAlign)
		}
		op := &text.DrawOptions{}
		op.GeoM.Translate(x, drawY)
		op.PrimaryAlign = primaryAlign
		op.SecondaryAlign = secondaryAlign
		op.ColorScale.ScaleWithColor(col)
		text.Draw(dst, str, face, op)
		return
	}

	faces := make([]*text.GoTextFace, len(runs))
	widths := make([]float64, len(runs))
	total := 0.0
	for i, r := range runs {
		if r.latin {
			faces[i] = g.LatinFontFace(size)
		} else {
			faces[i] = g.FontFace(size)
		}
		widths[i] = text.Advance(r.text, faces[i])
		total += widths[i]
	}

	startX := x
	switch primaryAlign {
	case text.AlignCenter:
		startX -= total / 2
	case text.AlignEnd:
		startX -= total
	}

	latinAdjust := g.latinBaselineAdjust(size, secondaryAlign)
	cx := startX
	for i, r := range runs {
		drawY := y
		if r.latin {
			drawY += latinAdjust
		}
		op := &text.DrawOptions{}
		op.GeoM.Translate(cx, drawY)
		op.SecondaryAlign = secondaryAlign
		op.ColorScale.ScaleWithColor(col)
		text.Draw(dst, r.text, faces[i], op)
		cx += widths[i]
	}
}

// MeasureMixedText returns the size DrawMixedText would render str at,
// switching between FontFace and LatinFontFace per run the same way.
func (g *Game) MeasureMixedText(str string, size float64) (width, height float64) {
	runs := splitLatinRuns(str)
	if len(runs) == 0 {
		return 0, 0
	}
	for _, r := range runs {
		face := g.FontFace(size)
		if r.latin {
			face = g.LatinFontFace(size)
		}
		w, h := text.Measure(r.text, face, 0)
		width += w
		if h > height {
			height = h
		}
	}
	return width, height
}
