package vprotocol

import "github.com/google/uuid"

type PayloadGetInfoResponse struct {
	ConnectedClients []ClientDescription `json:"clients"`
}

type ClientDescription struct {
	ID   uuid.UUID `json:"id"`
	Room string    `json:"room"`
	Name string    `json:"name"`
}
