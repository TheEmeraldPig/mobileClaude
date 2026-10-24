// Mote relay: lets players join each other's worlds with a short code, from anywhere.
//
// A host opens /host and gets a 5-letter code. Friends open /join?code=XXXXX. The relay
// only passes messages along; the host's game runs the world. Up to 3 guests per host.
// The protocol is in mote/relaycore.
//
// Run it anywhere that serves HTTP; PORT sets the port (default 8080). See README.md.
// (Optional: the game also hosts on the local network without any server.)
package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/TheEmeraldPig/mobileClaude/mote/relaycore"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	m := relaycore.Mux()
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Mote relay ok, " + strconv.Itoa(relaycore.Rooms()) + " worlds open\n"))
	})
	log.Printf("Mote relay listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, m))
}
