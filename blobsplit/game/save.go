package game

import (
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

// State is everything that is saved.
type State struct {
	Ver      int     `json:"ver"`
	Money    float64 `json:"money"`
	Gems     int     `json:"gems"` // DNA, spent in the market
	Points   int     `json:"points"`
	ValueLv  int     `json:"valueLv"`
	SpeedLv  int     `json:"speedLv"`
	CapLv    int     `json:"capLv"`
	Rebirths int     `json:"rebirths"`
	Auto     bool    `json:"auto"`
	AutoSell bool    `json:"autoSell"`
	Sound    bool    `json:"sound"`
	NoMusic  bool    `json:"noMusic"`
	BuyMode  int     `json:"buyMode"` // index into buyModes
	Found    []int   `json:"found"`   // times each mutation has been gained
	Board    []int   `json:"board"`   // mutation masks of the biotes on the board
	Earned   float64 `json:"earned"`

	Tokens   int             `json:"tokens"`   // from rebirths, for the token shop
	Lab      [][]int         `json:"lab"`      // DNA lab levels: [value, chance, spread][mutation]
	Perks    map[string]bool `json:"perks"`    // permanent token shop items
	LastSeen int64           `json:"lastSeen"` // unix time of the last save, for idle gains
	Rate     float64         `json:"rate"`     // recent income per second

	Tree  map[string]bool `json:"tree"`  // skill tree nodes unlocked
	Cards []string        `json:"cards"` // ability cards unlocked (all active)

	// the challenge: floors of five rooms
	Floor      int     `json:"floor"`
	Room       int     `json:"room"` // 0-4; 4 is the boss
	Boss       int     `json:"boss"`
	RoomEarned float64 `json:"roomEarned"`
	RoomBest   float64 `json:"roomBest"` // best single sale, or most mutated biotes sold
	RoomSells  int     `json:"roomSells"`
	RunFloors  int     `json:"runFloors"` // floors cleared since the last rebirth
	BestFloor  int     `json:"bestFloor"` // most floors cleared ever

	// day, night and events
	Clock     float64 `json:"clock"`
	Event     string  `json:"event"`
	EventLeft float64 `json:"eventLeft"`
	NextEvent float64 `json:"nextEvent"`

	// tutorial, visitors and the shop
	Tut        int                `json:"tut"`        // tutorial step; len(tutSteps) when done
	DrawerShut bool               `json:"drawerShut"` // the upgrades drawer is closed
	Buffs      map[string]float64 `json:"buffs"`      // timed buffs: seconds left
	NextVisit  float64            `json:"nextVisit"`  // seconds until the next visitor
	PtFrac     float64            `json:"ptFrac"`     // part skill point carried over (x1.5 boost)

	// from older versions
	OldBlobs []int          `json:"blobs,omitempty"`
	OldGemLv map[string]int `json:"gemLv,omitempty"`
}

const stateVer = 5

func newState() State {
	st := State{
		Ver: stateVer, Sound: true, Found: make([]int, nMut), Board: []int{0},
		Tree: map[string]bool{}, Floor: 1, Boss: rand.Intn(len(bosses)),
		Clock: dayLen * 0.05, NextEvent: 90, NextVisit: 60,
	}
	st.fill()
	return st
}

// fill makes sure the maps and tables exist.
func (st *State) fill() {
	if st.Perks == nil {
		st.Perks = map[string]bool{}
	}
	if st.Buffs == nil {
		st.Buffs = map[string]float64{}
	}
	for len(st.Lab) < nLab {
		st.Lab = append(st.Lab, nil)
	}
	for k := range st.Lab {
		for len(st.Lab[k]) < nMut {
			st.Lab[k] = append(st.Lab[k], 0)
		}
	}
}

func dataDir() string {
	if DataDir != "" {
		return DataDir
	}
	if d, err := os.UserConfigDir(); err == nil {
		return filepath.Join(d, "BlobSplit")
	}
	return "."
}

func savePath() string { return filepath.Join(dataDir(), "save.json") }

func loadState() State {
	st := newState()
	if os.Getenv("BLOB_FRESH") != "" {
		return st
	}
	b, err := readSave()
	if err != nil {
		return st
	}
	st.Ver = 0 // a save without a version is from Blob Split
	if json.Unmarshal(b, &st) != nil {
		return newState()
	}
	if st.Ver < 2 {
		// Blob Split saves: a board of single rarities becomes masks, and the gem
		// shop is replaced by the skill tree, so rebirths so far turn into points.
		st.Board = nil
		for _, r := range st.OldBlobs {
			if r > 0 && r < nMut {
				st.Board = append(st.Board, 1<<r)
			} else {
				st.Board = append(st.Board, 0)
			}
		}
		st.Points += 2 * st.Rebirths
		st.Auto, st.AutoSell = false, false
		st.Ver = 3
	}
	st.OldBlobs, st.OldGemLv = nil, nil
	if st.Tree == nil {
		st.Tree = map[string]bool{}
	}
	for len(st.Found) < nMut {
		st.Found = append(st.Found, 0)
	}
	if len(st.Board) == 0 {
		st.Board = []int{0}
	}
	if st.Ver < 3 {
		// rounds and the card market became floors: cards you had stay unlocked
		st.Floor, st.Room = 1, 0
		st.Ver = 3
	}
	if st.Floor < 1 {
		st.Floor = 1
	}
	if st.Ver < 4 {
		// the big card update: old cards become their new versions
		var keep []string
		for _, id := range st.Cards {
			if n, ok := oldCards[id]; ok {
				id = n
			}
			if cardByID(id) != nil {
				keep = append(keep, id)
			}
		}
		st.Cards = keep
		st.Ver = 4
	}
	if st.Ver < 5 {
		// the tutorial is for new players
		if st.Earned > 0 {
			st.Tut = len(tutSteps)
		}
		st.NextVisit = 60
		st.Ver = stateVer
	}
	st.fill()
	return st
}

func (g *Game) save() {
	if os.Getenv("BLOB_SHOT") != "" {
		return // screenshot runs don't touch the real save
	}
	g.st.LastSeen = time.Now().Unix()
	g.st.Board = g.st.Board[:0]
	for _, b := range g.blobs {
		g.st.Board = append(g.st.Board, b.m)
	}
	b, err := json.Marshal(g.st)
	if err != nil {
		return
	}
	writeSave(b)
}

var current *Game

// SaveNow is called by the Android app when it goes to the background.
func SaveNow() {
	if current != nil {
		current.mu.Lock()
		defer current.mu.Unlock()
		current.save()
	}
}
