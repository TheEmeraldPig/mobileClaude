package game

import (
	"image"
	"image/draw"
	"image/png"
	"os"
	"testing"
)

// KITCHEN_PREVIEW=dir paints every chef, monster, boss and food into a contact sheet.
func TestPreview(t *testing.T) {
	dir := os.Getenv("KITCHEN_PREVIEW")
	if dir == "" {
		t.Skip()
	}
	var tiles []*image.RGBA
	for id := range chefs {
		c := newCanvas(-80, -130, 80, 60, 1.5)
		paintChef(c, id)
		paintChefFace(c, id, 0)
		tiles = append(tiles, c.rgba())
	}
	for id := range monsters {
		c := newCanvas(-140, -260, 140, 12, 0.86)
		paintMonster(c, id, false)
		paintMonFace(c, monsters[id], false, 0)
		tiles = append(tiles, c.rgba())
	}
	for id := range bosses {
		c := newCanvas(-140, -260, 140, 12, 0.86)
		paintMonster(c, id, true)
		paintMonFace(c, bosses[id], true, 0)
		tiles = append(tiles, c.rgba())
	}
	food := newCanvas(0, 0, 240, 190, 1)
	for i, k := range []int{0, 1, 2, 3, 4, 5, 6, 7, 100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, fiKnife, fiLadle, fiBox, fiGoldBox, fiHeart, fiPlate, fiFlame} {
		f := newCanvas(-23, -23, 23, 23, 1)
		paintFood(f, k, 1)
		_ = i
		fi := f.rgba()
		x, y := i%5*48, i/5*31
		sub := image.NewRGBA(image.Rect(0, 0, food.w, food.h))
		_ = sub
		for py := 0; py < fi.Bounds().Dy(); py++ {
			for px := 0; px < fi.Bounds().Dx(); px++ {
				if x+px < food.w && y+py < food.h {
					o := ((y+py)*food.w + x + px) * 4
					a := float32(fi.Pix[(py*fi.Stride+px*4)+3]) / 255
					for ch := 0; ch < 4; ch++ {
						food.pix[o+ch] = float32(fi.Pix[py*fi.Stride+px*4+ch])/255 + food.pix[o+ch]*(1-a)
					}
				}
			}
		}
	}
	tiles = append(tiles, food.rgba())
	tw, th := 240, 240
	sheet := image.NewRGBA(image.Rect(0, 0, tw*6, th*((len(tiles)+5)/6)))
	draw.Draw(sheet, sheet.Bounds(), &image.Uniform{rgb(0xa8, 0xd8, 0xf0)}, image.Point{}, draw.Src)
	for i, tl := range tiles {
		draw.Draw(sheet, tl.Bounds().Add(image.Pt(i%6*tw, i/6*th)), tl, image.Point{}, draw.Over)
	}
	f, _ := os.Create(dir + "/sheet.png")
	png.Encode(f, sheet)
	f.Close()
}

// KITCHEN_ICON=file paints the app icon (Pip in front of a pizza), 192 px.
func TestIcon(t *testing.T) {
	out := os.Getenv("KITCHEN_ICON")
	if out == "" {
		t.Skip()
	}
	k := 192.0 / 200
	c := newCanvas(-100, -100, 100, 100, k)
	c.fill(rrect(-100, -100, 200, 200, 44), rgb(0x1a, 0x0e, 0x20))
	c.grad(rrect(-94, -94, 188, 188, 40), rgb(0xff, 0xb0, 0x40), rgb(0xff, 0x6a, 0x3a), -94, 94)
	c.glowIn(circ(0, -10, 40), rgb(0xff, 0xf0, 0xb0), 70, 0.5)
	paintFood(c.sub(-58, -28, 1), fiDish+dPizza, 1.9)
	img := c.rgba()
	b := newCanvas(-80, -130, 80, 60, k*1.05)
	paintChef(b, 0)
	paintChefFace(b, 0, 0)
	src := b.rgba()
	draw.Draw(img, src.Bounds().Add(image.Pt(18, 6)), src, image.Point{}, draw.Over)
	f, _ := os.Create(out)
	png.Encode(f, img)
	f.Close()
}
