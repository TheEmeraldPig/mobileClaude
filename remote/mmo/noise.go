package mmo

import "math"

// Seeded hashing and noise (the same functions as Mote's world.go, with the seed passed
// in instead of a global)

type gen struct{ seed uint32 }

func (g gen) hash32(a, b, c int) uint32 {
	h := g.seed ^ (uint32(int32(a)) * 0x27d4eb2d) ^ (uint32(int32(b)) * 0x165667b1) ^ (uint32(int32(c)) * 0x1b873593)
	h = (h ^ (h >> 15)) * 0x85ebca6b
	h = (h ^ (h >> 13)) * 0xc2b2ae35
	return h ^ (h >> 16)
}

func (g gen) hash01(a, b, c int) float64 { return float64(g.hash32(a, b, c)) / 4294967296 }

// Smooth 1D value noise in -1..1
func (g gen) noise1(x float64, scale float64, salt int) float64 {
	f := x / scale
	i := math.Floor(f)
	t := (1 - math.Cos((f-i)*math.Pi)) / 2
	ii := int(i)
	return (g.hash01(ii, salt, 7)*2-1)*(1-t) + (g.hash01(ii+1, salt, 7)*2-1)*t
}

// Smooth 2D value noise in 0..1
func (g gen) noise2(x, y, scale float64, salt int) float64 {
	fx, fy := x/scale, y/scale
	ix, iy := math.Floor(fx), math.Floor(fy)
	tx, ty := fx-ix, fy-iy
	tx, ty = tx*tx*(3-2*tx), ty*ty*(3-2*ty)
	i, j := int(ix), int(iy)
	a, b := g.hash01(i, j, salt), g.hash01(i+1, j, salt)
	c, d := g.hash01(i, j+1, salt), g.hash01(i+1, j+1, salt)
	return (a*(1-tx)+b*tx)*(1-ty) + (c*(1-tx)+d*tx)*ty
}

// A repeatable random sequence (mulberry32)
func (g gen) rng(k, salt int) func() float64 {
	a := g.hash32(k, salt, 1234567)
	return func() float64 {
		a += 0x6D2B79F5
		t := (a ^ (a >> 15)) * (1 | a)
		t = (t + (t^(t>>7))*(61|t)) ^ t
		return float64(t^(t>>14)) / 4294967296
	}
}
