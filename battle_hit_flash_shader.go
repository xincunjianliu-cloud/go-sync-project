package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

var hitFlashShaderSrc = []byte(`
package main

var Intensity float

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	c := imageSrc0UnsafeAt(srcPos)
	white := vec4(c.a, c.a, c.a, c.a)
	return mix(c, white, Intensity) * color
}
`)

var hitFlashShader *ebiten.Shader

func init() {
	s, err := ebiten.NewShader(hitFlashShaderSrc)
	if err != nil {
		panic(err)
	}
	hitFlashShader = s
}

func drawWithHitFlash(dst *ebiten.Image, src *ebiten.Image, g ebiten.GeoM, intensity float64) {
	if intensity <= 0 || hitFlashShader == nil {
		dst.DrawImage(src, &ebiten.DrawImageOptions{GeoM: g})
		return
	}
	if intensity > 1 {
		intensity = 1
	}

	op := &ebiten.DrawRectShaderOptions{}
	op.GeoM = g
	op.Images[0] = src
	op.Uniforms = map[string]interface{}{"Intensity": float32(intensity)}

	w := src.Bounds().Dx()
	h := src.Bounds().Dy()
	dst.DrawRectShader(w, h, hitFlashShader, op)
}
