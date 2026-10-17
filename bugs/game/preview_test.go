package game

import (
	"image"
	"image/draw"
	"image/png"
	"os"
	"testing"
)

// BUGS_PREVIEW=dir paints every bug and slug into a contact sheet.
func TestPreview(t *testing.T) {
	dir := os.Getenv("BUGS_PREVIEW")
	if dir == "" {
		t.Skip()
	}
	k := 2.0
	var tiles []*image.RGBA
	for id := range bugs {
		c := newCanvas(-90, -140, 100, 10, k)
		paintBugWings(c, id, false)
		paintBugBody(c, id)
		paintFace(c, bugs[id], 0)
		paintBugWings(c, id, true)
		tiles = append(tiles, c.rgba())
	}
	for i := range slugKinds {
		c := newCanvas(-90, -140, 100, 10, k)
		paintSlug(c, slugKinds[i], false)
		paintSlugEyes(c, slugKinds[i], false, 0)
		tiles = append(tiles, c.rgba())
	}
	c := newCanvas(-90, -140, 100, 10, k)
	paintSlug(c, bossKinds[0], true)
	paintSlugEyes(c, bossKinds[0], true, 0)
	tiles = append(tiles, c.rgba())
	tw, th := tiles[0].Bounds().Dx(), tiles[0].Bounds().Dy()
	sheet := image.NewRGBA(image.Rect(0, 0, tw*4, th*((len(tiles)+3)/4)))
	draw.Draw(sheet, sheet.Bounds(), &image.Uniform{rgb(0x9a, 0xd0, 0xf0)}, image.Point{}, draw.Src)
	for i, tl := range tiles {
		draw.Draw(sheet, tl.Bounds().Add(image.Pt(i%4*tw, i/4*th)), tl, image.Point{}, draw.Over)
	}
	f, _ := os.Create(dir + "/sheet.png")
	png.Encode(f, sheet)
	f.Close()
}

// BUGS_ICON=file paints the app icon (Rory on a sunny disc), 192 px.
func TestIcon(t *testing.T) {
	out := os.Getenv("BUGS_ICON")
	if out == "" {
		t.Skip()
	}
	k := 192.0 / 200
	c := newCanvas(-100, -100, 100, 100, k)
	c.fill(rrect(-100, -100, 200, 200, 44), rgb(0x2a, 0x12, 0x45))
	c.grad(rrect(-94, -94, 188, 188, 40), rgb(0x6c, 0xd0, 0xff), rgb(0x8c, 0xe0, 0x5a), -94, 94)
	c.glowIn(circ(40, -50, 20), rgb(0xff, 0xf4, 0xc0), 60, 0.5)
	c.vol(hillSh(-94, 94, 20, 94, func(x float64) float64 { return 46 + x*x*0.002 }), mat{base: rgb(0x6c, 0xc8, 0x40), flat: 0.4, depth: 20})
	inner := newCanvas(-90, -140, 100, 10, k*1.3)
	paintBugBody(inner, 0)
	paintFace(inner, bugs[0], 0)
	img := c.rgba()
	src := inner.rgba()
	off := image.Pt(-23, -4) // puts the bug's middle near the icon's centre
	draw.Draw(img, src.Bounds().Add(off), src, image.Point{}, draw.Over)
	f, _ := os.Create(out)
	png.Encode(f, img)
	f.Close()
}
