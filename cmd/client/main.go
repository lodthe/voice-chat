package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os/signal"
	"syscall"

	"github.com/lodthe/voice-chat/internal/client"
	"github.com/lodthe/voice-chat/pkg/vprotocol"
	"github.com/pkg/errors"
)

func main() {
	log.SetFlags(0)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	messageInput := make(chan *vprotocol.Message, 64)
	messageOutput := make(chan *vprotocol.Message, 64)
	audioInput := make(chan []byte, 64)
	audioOutput := make(chan []byte, 64)
	userInput := make(chan string, 8)
	userOutput := make(chan string, 8)

	handler := client.NewHandler(ctx, messageInput, messageOutput, userInput, userOutput, audioInput, audioOutput)
	messenger := client.NewMessenger(ctx, userInput, userOutput)

	audio, err := client.NewAudio(ctx, client.DefaultAudioConfig, audioInput, audioOutput)
	if err != nil {
		log.Fatalf("failed to initialize audio: %v\n", err)
	}

	var connection *client.Connection
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		fmt.Printf("Server address (ip:port): ")

		var address string
		_, _ = fmt.Scanf("%s", &address)

		connection, err = client.NewConnection(ctx, address, messageInput, messageOutput)
		if err == nil {
			break
		}

		fmt.Printf("failed to connect: %v\n\n", err)
	}

	go func() {
		messenger.Start()
	}()

	go func() {
		handler.Handle()
	}()

	go func() {
		err := connection.Start()
		if errors.Is(err, io.EOF) {
			fmt.Println("\n\n\nServer is shutting down...")
			stop()

			return
		}
		if err != nil {
			log.Fatalf("connection failed: %v\n", err)
		}
	}()

	go func() {
		err := audio.Start()
		if err != nil {
			log.Fatalf("audio failed: %v\n", err)
		}
	}()

	<-ctx.Done()
}
