package client

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/lodthe/voice-chat/pkg/vprotocol"
)

type Handler struct {
	ctx    context.Context
	cancel context.CancelFunc

	msgInput  <-chan *vprotocol.Message
	msgOutput chan<- *vprotocol.Message

	userInput  <-chan string
	userOutput chan<- string

	audioInput  <-chan []byte
	audioOutput chan<- []byte
}

func NewHandler(
	ctx context.Context,
	msgInput <-chan *vprotocol.Message,
	msgOutput chan<- *vprotocol.Message,
	userInput <-chan string,
	userOutput chan<- string,
	audioInput <-chan []byte,
	audioOutput chan<- []byte,
) *Handler {
	ctx, cancel := context.WithCancel(ctx)

	return &Handler{
		ctx:         ctx,
		cancel:      cancel,
		msgInput:    msgInput,
		msgOutput:   msgOutput,
		userInput:   userInput,
		userOutput:  userOutput,
		audioInput:  audioInput,
		audioOutput: audioOutput,
	}
}

func (h *Handler) Handle() {
	go h.handleUserInput()
	go h.handleIngoingMessages()
	go h.handleIngoingAudio()

	h.userOutput <- "Connected!\n"
	h.sendHelpCommand()
}

func (h *Handler) handleUserInput() {
	for {
		var input string
		select {
		case <-h.ctx.Done():
			return

		case input = <-h.userInput:
		}

		switch {
		case strings.HasPrefix(input, "join"):
			h.handleInputJoin(input)

		case strings.HasPrefix(input, "leave"):
			h.handleInputLeave(input)

		case strings.HasPrefix(input, "info"):
			h.handleInputInfo(input)

		default:
			h.sendError("Invalid command, please see help.")
		}
	}
}

func (h *Handler) handleInputJoin(input string) {
	splitted := strings.Split(input, " ")
	if len(splitted) != 3 {
		h.sendError("Invalid syntax, should be 'help [room] [name]'. Spaces in name and room are not allowed.")
	}

	h.msgOutput <- &vprotocol.Message{
		Type: vprotocol.TypeJoin,
		Payload: &vprotocol.PayloadJoin{
			Room: splitted[1],
			Name: splitted[2],
		},
	}
}

func (h *Handler) handleInputLeave(_ string) {
	h.msgOutput <- &vprotocol.Message{
		Type:    vprotocol.TypeLeave,
		Payload: &vprotocol.PayloadLeave{},
	}
}

func (h *Handler) handleInputInfo(_ string) {
	h.msgOutput <- &vprotocol.Message{
		Type:    vprotocol.TypeGetInfoRequest,
		Payload: &vprotocol.PayloadGetInfoRequest{},
	}
}

func (h *Handler) handleIngoingMessages() {
	for {
		var msg *vprotocol.Message
		select {
		case <-h.ctx.Done():
			return

		case msg = <-h.msgInput:
		}

		switch msg.Type {
		case vprotocol.TypeJoin:
			h.handleMessageJoin(msg.Payload.(*vprotocol.PayloadJoin))

		case vprotocol.TypeLeave:
			h.handleMessageLeave(msg.Payload.(*vprotocol.PayloadLeave))

		case vprotocol.TypeAudio:
			h.handleMessageAudio(msg.Payload.(*vprotocol.PayloadAudio))

		case vprotocol.TypeTextResponse:
			h.handleMessageTextResponse(msg.Payload.(*vprotocol.PayloadTextResponse))

		case vprotocol.TypeGetInfoResponse:
			h.handleMessageGetInfoResponse(msg.Payload.(*vprotocol.PayloadGetInfoResponse))
		}
	}
}

func (h *Handler) handleMessageJoin(payload *vprotocol.PayloadJoin) {
	h.sendWithPrompt(fmt.Sprintf("{+} %s joined #%s", payload.Name, payload.Room))
}

func (h *Handler) handleMessageLeave(payload *vprotocol.PayloadLeave) {
	h.sendWithPrompt(fmt.Sprintf("{-} %s left #%s", payload.Name, payload.Room))
}

func (h *Handler) handleMessageAudio(payload *vprotocol.PayloadAudio) {
	h.audioOutput <- payload.Data
}

func (h *Handler) handleMessageTextResponse(payload *vprotocol.PayloadTextResponse) {
	h.sendWithPrompt("[RESPONSE]: " + payload.Message)
}

func (h *Handler) handleMessageGetInfoResponse(payload *vprotocol.PayloadGetInfoResponse) {
	rooms := make(map[string][]vprotocol.ClientDescription)
	var roomNames []string
	for _, cli := range payload.ConnectedClients {
		_, exists := rooms[cli.Room]
		if !exists {
			roomNames = append(roomNames, cli.Room)
		}

		rooms[cli.Room] = append(rooms[cli.Room], cli)
	}

	sort.Strings(roomNames)

	info := "Information about connected clients:\n"

	for _, room := range roomNames {
		info += fmt.Sprintf("\n[#%s]\n", room)

		clients := rooms[room]
		sort.Slice(clients, func(i, j int) bool {
			return clients[i].ID.String() < clients[j].ID.String()
		})

		for _, client := range clients {
			info += fmt.Sprintf("  - %s [%s]\n", client.Name, client.ID.String()[:6])
		}
	}

	if len(roomNames) == 0 {
		info += "There are no connected clients :("
	}

	h.sendWithPrompt(info)
}

func (h *Handler) handleIngoingAudio() {
	for {
		var frame []byte
		select {
		case <-h.ctx.Done():
			return

		case frame = <-h.audioInput:
		}

		h.msgOutput <- &vprotocol.Message{
			Type: vprotocol.TypeAudio,
			Payload: &vprotocol.PayloadAudio{
				Data: frame,
			},
		}
	}
}

func (h *Handler) sendHelpCommand() {
	h.sendWithPrompt(`===========
Command list:

> help - print this message.

> join [room] [name] - join a voice room with the specified name (no spaces allowed).

> leave - disconnect from the voice room.

> info - list connected clients.
===========`)
}

func (h *Handler) sendError(text string) {
	h.sendWithPrompt("[ERROR] " + text)
}

func (h *Handler) sendWithPrompt(text string) {
	h.userOutput <- "\n\n" + text + "\n\nSend me something: "
}
