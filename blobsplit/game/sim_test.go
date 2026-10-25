package game

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"testing"
)

// TestSim plays the game with a simple bot, for balancing: it splits as soon as it
// can, sells when the board is full, buys the cheapest upgrade it can afford and
// rebirths once its floor climbing has stalled for 5 minutes. Run with BLOB_SIM=minutes (it is skipped otherwise):
//
//	BLOB_SIM=120 xvfb-run -a go test -run TestSim -v ./game
func TestSim(t *testing.T) {
	mins, err := strconv.ParseFloat(os.Getenv("BLOB_SIM"), 64)
	if err != nil {
		t.Skip("set BLOB_SIM=minutes to run the balance simulation")
	}
	rand.Seed(1)
	g := testGame(t)
	g.st.NextEvent = 1e9 // no events: they make runs noisy
	lastFloor, lastReb := 0, 0
	lastGain, runFloors := 0.0, 0
	for tick := 0; tick < int(mins*60*60); tick++ {
		g.t += dt
		g.st.Clock += dt
		g.m = g.computeMods()
		g.cool += dt
		g.trySplit(false)
		if len(g.blobs) >= g.capacity() {
			g.sell()
		}
		if tick%60 == 0 {
			for {
				best, bc := -1, 0.0
				for i := 0; i < nUps; i++ {
					if *g.upLv(i) < upMax(i) {
						if c := upCost(i, *g.upLv(i)); c <= g.st.Money && (best < 0 || c < bc) {
							best, bc = i, c
						}
					}
				}
				if best < 0 {
					break
				}
				g.buyCard(best, true)
			}
			// rebirth only once climbing has stalled for 5 minutes
			if g.st.Money >= rebirthReq(g.st.Rebirths) && g.t-lastGain > 300 {
				g.rebirth(false)
				g.panel = ""
			}
		}
		if g.st.RunFloors != runFloors {
			runFloors, lastGain = g.st.RunFloors, g.t
		}
		at := fmt.Sprintf("%5.1f min", g.t/60)
		if g.st.BestFloor > lastFloor {
			lastFloor = g.st.BestFloor
			t.Logf("%s  best floor %d (floor %d, cards %d)", at, lastFloor, g.st.Floor, len(g.st.Cards))
		}
		if g.st.Rebirths > lastReb {
			lastReb = g.st.Rebirths
			t.Logf("%s  REBIRTH %d  (points %d, DNA %d, tokens %d)", at, lastReb, g.st.Points, g.st.Gems, g.st.Tokens)
		}
	}
	t.Logf("now on: %+v (sells used %d)", g.roomReq(), g.st.RoomSells)
	t.Logf("end: money %s, value lv %d, floor %d room %d, best %d, rebirths %d", fmtNum(g.st.Money), g.st.ValueLv, g.st.Floor, g.st.Room+1, g.st.BestFloor, g.st.Rebirths)
}
