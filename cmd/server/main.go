package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/lodthe/voice-chat/internal/server"
)

func main() {
	log.SetFlags(0)

	ctx, cancel := context.WithCancel(context.Background())
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	manager := server.NewManager(ctx)
	srv := server.NewServer(ctx, "0.0.0.0:8888", manager)

	go func() {
		err := srv.Serve()
		if err != nil {
			log.Fatalf("server failed: %v\n", err)
		}
	}()

	<-stop
	cancel()
}
