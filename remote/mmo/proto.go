package mmo

// Messages between the game and the server: JSON text over a WebSocket.
// Positions are world px on the current island (a mote's top-left corner).

// Protocol goes up when old games can no longer talk to the server
const Protocol = 1

// Client messages
type CMsg struct {
	T      string  `json:"t"`
	Ver    int     `json:"ver,omitempty"`
	Key    string  `json:"key,omitempty"`    // hello: the device's secret key (identifies the account)
	Name   string  `json:"name,omitempty"`   // hello, name
	Backup string  `json:"backup,omitempty"` // hello: the last signed copy of the character
	Epoch  int     `json:"e,omitempty"`      // in: which teleport this position follows
	X      float64 `json:"x,omitempty"`
	Y      float64 `json:"y,omitempty"`
	VX     float64 `json:"vx,omitempty"`
	VY     float64 `json:"vy,omitempty"`
	Face   int     `json:"f,omitempty"`
	Mode   int     `json:"m,omitempty"`  // petal orbit: normal, attack, defend
	Flags  int     `json:"fl,omitempty"` // FlagLift, FlagGround
	Isle   string  `json:"isle,omitempty"`
	Slot   int     `json:"slot,omitempty"`
	Slot2  int     `json:"slot2,omitempty"`
	Petal  PetalID `json:"p,omitempty"`
	Emote  int     `json:"emote,omitempty"`
}

// Server messages
type SMsg struct {
	T      string         `json:"t"`
	ID     int            `json:"id,omitempty"`
	Char   *Char          `json:"char,omitempty"`
	Away   *Away          `json:"away,omitempty"`
	Backup string         `json:"backup,omitempty"`
	Isle   string         `json:"isle,omitempty"`
	X      float64        `json:"x,omitempty"`
	Y      float64        `json:"y,omitempty"`
	Epoch  int            `json:"e,omitempty"`
	HP     float64        `json:"hp,omitempty"`
	MaxHP  float64        `json:"mhp,omitempty"`
	Time   float64        `json:"time,omitempty"` // server seconds (petals turn with it)
	P      []PSnap        `json:"p,omitempty"`
	M      []MSnap        `json:"m,omitempty"`
	Dmg    int            `json:"dmg,omitempty"`
	Dir    int            `json:"dir,omitempty"`
	Info   *PInfo         `json:"info,omitempty"`
	Emote  int            `json:"emote,omitempty"`
	Loot   *Loot          `json:"loot,omitempty"`
	Text   string         `json:"text,omitempty"`
	Online int            `json:"online,omitempty"`
	Counts map[string]int `json:"counts,omitempty"` // players on each island
}

const (
	FlagLift   = 1
	FlagGround = 2
	FlagDead   = 4
)

// Another player, in a snapshot
type PSnap struct {
	ID    int `json:"i"`
	X     int `json:"x"`
	Y     int `json:"y"`
	VX    int `json:"vx,omitempty"` // px per second
	VY    int `json:"vy,omitempty"`
	Face  int `json:"f,omitempty"`
	Flags int `json:"fl,omitempty"`
	HP    int `json:"hp"`
	MaxHP int `json:"mh"`
	Orbit int `json:"o"` // petal orbit radius, px
	Alive int `json:"a"` // bit i: petal i is out
}

// A creature, in a snapshot
type MSnap struct {
	ID    int `json:"i"`
	Kind  int `json:"k"` // index in Mobs
	Rar   int `json:"r,omitempty"`
	X     int `json:"x"`
	Y     int `json:"y"`
	VX    int `json:"vx,omitempty"`
	VY    int `json:"vy,omitempty"`
	Face  int `json:"f,omitempty"`
	HP    int `json:"h"` // health left, 0-1000
	Flags int `json:"fl,omitempty"`
}

const (
	MobHurt = 1 // hit just now
	MobBurn = 2
	MobSlow = 4
)

// What other players need to draw someone
type PInfo struct {
	ID    int       `json:"id"`
	Name  string    `json:"name"`
	Level int       `json:"lvl"`
	Load  []PetalID `json:"load"`
}

// Rewards from a kill
type Loot struct {
	Mob    string    `json:"mob"`
	Rar    int       `json:"r"`
	XP     int64     `json:"xp"`
	Coins  int64     `json:"coins"`
	Petals []PetalID `json:"petals,omitempty"`
	Levels int       `json:"levels,omitempty"`
	X      int       `json:"x"`
	Y      int       `json:"y"`
}

// Emotes shown over a mote
var Emotes = []string{"Hi!", "<3", "Haha", "GG", "Help!", "Follow me"}
