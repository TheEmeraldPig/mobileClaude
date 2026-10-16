package game

import (
	"math"
	"math/rand"
)

// One run: the ship circles a planet; a tap flings it off along its path.
// Reaching the next planet's pull scores; flying off the screen or into the void ends the run.

type planet struct {
	x, y, r float64
	col     int
}

func (p planet) orbit() float64 { return p.r + 34 }

type rock struct {
	x, y0, r, phase, amp float64
}

func (k rock) y(t float64) float64 { return k.y0 + math.Sin(t*1.6+k.phase)*k.amp }

type star struct {
	x, y float64
	got  bool
}

type pt struct{ x, y float64 }

type run struct {
	rng     *rand.Rand
	planets []planet
	rocks   []rock
	stars   []star
	t       float64

	x, y, vx, vy float64
	on           int     // planet being circled, or -1 while flying
	from         int     // last planet circled
	ang, dir     float64 // angle and direction (+1/-1) around the planet
	rad          float64 // current distance from the planet, eases to its orbit
	flyT         float64
	trail        []pt

	speed   float64
	void    float64 // x of the void's edge
	camX    float64
	score   int
	starsN  int
	shields int
	perfect int // frames left to show "Perfect!"
	dead    bool
	revived bool
	adUsed  bool
	why     string

	magnet, gravity, slow float64
}

const runSpeed = 300.0

func newRun(seed int64, p *Profile, w float64) *run {
	r := &run{rng: rand.New(rand.NewSource(seed)), speed: runSpeed, dir: 1}
	r.magnet = 14 + 10*float64(p.Upg["magnet"])
	r.gravity = 12 + 7*float64(p.Upg["gravity"])
	r.slow = 1 - 0.1*float64(p.Upg["slow"])
	r.shields = p.Upg["shield"]
	r.planets = []planet{{w * 0.3, H / 2, 34, 0}}
	for len(r.planets) < 6 {
		r.addPlanet()
	}
	pl := r.planets[0]
	r.on, r.from, r.ang, r.rad = 0, 0, -math.Pi/2, pl.orbit()
	r.x, r.y = pl.x, pl.y-pl.orbit()
	r.void = -240
	return r
}

func (r *run) uniform(a, b float64) float64 { return a + r.rng.Float64()*(b-a) }

func (r *run) addPlanet() {
	n := len(r.planets)
	last := r.planets[n-1]
	gap := r.uniform(240, 320) + math.Min(float64(n)*3, 60)
	np := planet{
		x:   last.x + gap,
		y:   math.Max(120, math.Min(H-120, last.y+r.uniform(-170, 170))),
		r:   r.uniform(22, 40),
		col: r.rng.Intn(len(planetCols)),
	}
	r.planets = append(r.planets, np)
	// stars along the way
	dx, dy := np.x-last.x, np.y-last.y
	d := math.Hypot(dx, dy)
	for _, f := range []float64{0.35, 0.5, 0.65} {
		off := r.uniform(-30, 30)
		r.stars = append(r.stars, star{x: last.x + dx*f - dy/d*off, y: last.y + dy*f + dx/d*off})
	}
	// asteroids drifting between planets, more often further in
	if n >= 3 && r.rng.Float64() < math.Min(0.75, 0.3+float64(n)*0.02) {
		r.rocks = append(r.rocks, rock{
			x: last.x + dx*r.uniform(0.4, 0.6), y0: H / 2, r: r.uniform(12, 20),
			phase: r.uniform(0, 6.3), amp: r.uniform(60, 190),
		})
	}
}

// Launch flings the ship off along the direction it is moving.
func (r *run) Launch() {
	if r.on < 0 || r.dead {
		return
	}
	r.vx = -math.Sin(r.ang) * r.dir * r.speed
	r.vy = math.Cos(r.ang) * r.dir * r.speed
	r.from, r.on, r.flyT = r.on, -1, 0
}

func (r *run) hit(why string) {
	if r.dead {
		return
	}
	r.dead, r.why = true, why
}

func (r *run) step(dt, w float64) {
	if r.dead {
		return
	}
	r.t += dt
	if r.on >= 0 {
		pl := r.planets[r.on]
		r.rad += (pl.orbit() - r.rad) * math.Min(1, dt*6)
		r.ang += r.dir * r.speed / r.rad * dt
		r.x, r.y = pl.x+math.Cos(r.ang)*r.rad, pl.y+math.Sin(r.ang)*r.rad
	} else {
		r.flyT += dt
		r.x += r.vx * dt
		r.y += r.vy * dt
		for i := r.from + 1; i < len(r.planets) && r.planets[i].x < r.x+400; i++ {
			pl := r.planets[i]
			rx, ry := r.x-pl.x, r.y-pl.y
			d := math.Hypot(rx, ry)
			if d < pl.r+4 {
				r.hit("You crashed into a planet")
				return
			}
			if d <= pl.orbit()+r.gravity {
				r.on, r.rad, r.ang = i, d, math.Atan2(ry, rx)
				r.dir = 1
				if rx*r.vy-ry*r.vx < 0 {
					r.dir = -1
				}
				gained := i - r.from
				r.from = i
				if math.Abs(d-pl.orbit()) < 10 {
					gained++
					r.perfect = 50
				}
				r.score += gained
				r.speed = runSpeed + math.Min(float64(r.score)*3, 160)
				for len(r.planets) < i+6 {
					r.addPlanet()
				}
				break
			}
		}
		if r.on < 0 && (r.y < -40 || r.y > H+40 || r.flyT > 2.6) {
			r.hit("You drifted off into space")
			return
		}
	}
	// stars and asteroids
	for i := range r.stars {
		s := &r.stars[i]
		if !s.got && math.Hypot(s.x-r.x, s.y-r.y) < r.magnet {
			s.got = true
			r.starsN++
		}
	}
	for i := range r.rocks {
		k := &r.rocks[i]
		if k.r > 0 && math.Hypot(k.x-r.x, k.y(r.t)-r.y) < k.r+7 {
			if r.shields > 0 {
				r.shields--
				k.r = 0 // smashed
			} else {
				r.hit("An asteroid hit you")
				return
			}
		}
	}
	// the void creeps in from the left, faster the further you get
	vs := (45 + math.Min(float64(r.score)*4, 150)) * r.slow
	r.void = math.Max(r.void+vs*dt, r.camX-120)
	if r.x < r.void+6 {
		r.hit("The void caught you")
		return
	}
	// camera follows the planet being circled, or the ship in flight
	tx := r.x - w*0.35
	if r.on >= 0 {
		tx = r.planets[r.on].x - w*0.3
	}
	if tx > r.camX {
		r.camX += (tx - r.camX) * math.Min(1, dt*3)
	}
	r.trail = append(r.trail, pt{r.x, r.y})
	if len(r.trail) > 22 {
		r.trail = r.trail[1:]
	}
	if r.perfect > 0 {
		r.perfect--
	}
	// forget what is far behind
	for len(r.stars) > 0 && r.stars[0].x < r.camX-300 {
		r.stars = r.stars[1:]
	}
	for len(r.rocks) > 0 && r.rocks[0].x < r.camX-300 {
		r.rocks = r.rocks[1:]
	}
}

// Revive puts the ship back around the last planet and pushes the void back.
func (r *run) Revive() {
	pl := r.planets[r.from]
	r.on, r.rad, r.ang, r.dir = r.from, pl.orbit(), -math.Pi/2, 1
	r.x, r.y = pl.x, pl.y-pl.orbit()
	r.void = math.Min(r.void, pl.x-500)
	r.dead, r.revived, r.trail = false, true, nil
	for i := range r.rocks {
		if math.Abs(r.rocks[i].x-pl.x) < 200 {
			r.rocks[i].r = 0
		}
	}
}
