package game

import (
	"image"
	"math"
)

// IconImage draws the app icon (a blob on a violet pixel tile) at size x size.
func IconImage(size int) *image.RGBA {
	const n = 24
	art := image.NewRGBA(image.Rect(0, 0, n, n))
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			c := mix(colL, colS, float64(x+y)/(2*n))
			edge := x == 0 || y == 0 || x == n-1 || y == n-1
			corner := (x < 2 || x > n-3) && (y < 2 || y > n-3)
			switch {
			case corner:
				continue
			case edge:
				c = colO
			case x == 1 || y == 1:
				c = colL
			case x == n-2 || y == n-2:
				c = colS
			}
			art.SetRGBA(x, y, c)
		}
	}
	paste(art, blobSprite(0, false), 3, 4)
	return scaleUp(art, size)
}

// FeatureImage draws the 1024x500 Google Play feature graphic: dusk sky, moon, blobs.
func FeatureImage() *image.RGBA {
	const w, h = 256, 125
	art := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		t := float64(y) / h
		c := mix(rgb(0x1b1036), rgb(0xe0703a), t)
		for x := 0; x < w; x++ {
			cc := c
			if math.Hypot(float64(x)-60, float64(y)-45) < 30 {
				cc = rgb(0xffcf6a)
			}
			if t > 0.8+0.05*math.Sin(float64(x)/20) {
				cc = rgb(0x2a1238)
			}
			art.SetRGBA(x, y, cc)
		}
	}
	for i, r := range []int{0, 1, 0, 2, 0, 4, 0, 3, 0, 6, 0} {
		paste(art, blobSprite(r, false), 16+i*21, 56+int(6*math.Sin(float64(i))))
	}
	img := image.NewRGBA(image.Rect(0, 0, 1024, 500))
	for y := 0; y < 500; y++ {
		for x := 0; x < 1024; x++ {
			img.Set(x, y, art.At(x/4, y/4))
		}
	}
	return img
}

func paste(dst, src *image.RGBA, ox, oy int) {
	b := src.Bounds()
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			if c := src.RGBAAt(x, y); c.A > 0 {
				dst.SetRGBA(ox+x, oy+y, c)
			}
		}
	}
}

func scaleUp(src *image.RGBA, size int) *image.RGBA {
	n := src.Bounds().Dx()
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.SetRGBA(x, y, src.RGBAAt(x*n/size, y*n/size))
		}
	}
	return img
}
