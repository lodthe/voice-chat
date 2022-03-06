package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/lodthe/voice-chat/internal/server"
)

func main() {
	log.SetFlags(0)

	var address string
	flag.StringVar(&address, "address", "0.0.0.0:9000", "address for listening")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	manager := server.NewManager(ctx)
	srv := server.NewServer(ctx, address, manager)

	go func() {
		err := srv.Serve()
		if err != nil {
			log.Fatalf("server failed: %v\n", err)
		}
	}()

	<-stop
	cancel()
}
