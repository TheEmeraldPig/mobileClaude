// The Remote game server. Run it anywhere that can take WebSocket connections:
//
//	go run ./server                 (listens on :8080, or $PORT)
//
// Characters are saved in $REMOTE_DATA (default ./data). Set REMOTE_SECRET to a long
// random string that stays the same, so the backups the games keep stay valid after
// the server's disk is wiped (Render's free plan wipes it on every restart).
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TheEmeraldPig/mobileClaude/remote/srv"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	data := os.Getenv("REMOTE_DATA")
	if data == "" {
		data = "data"
	}
	s, err := srv.New(srv.Config{DataDir: data, Secret: srv.SecretFrom(data)})
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	done := make(chan struct{})
	go func() {
		s.Run(ctx) // saves everyone when ctx ends
		close(done)
	}()
	hs := &http.Server{Addr: ":" + port, Handler: s.Handler()}
	go func() {
		<-ctx.Done()
		sctx, c := context.WithTimeout(context.Background(), 5*time.Second)
		defer c()
		hs.Shutdown(sctx)
	}()
	log.Printf("Remote server on :%s", port)
	if err := hs.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
	<-done
}
