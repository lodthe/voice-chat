package server

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/google/uuid"
)

type Client struct {
	ctx  context.Context
	conn net.Conn
	mu   sync.RWMutex

	id      uuid.UUID
	address string
	room    *string
	name    string
}

func (c *Client) String() string {
	return fmt.Sprintf("%s [%s]", c.address, c.id)
}

func (c *Client) ID() uuid.UUID {
	return c.id
}

func (c *Client) Room() *string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.room
}

func (c *Client) SetRoom(newRoom *string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.room = newRoom
}

func (c *Client) Name() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.name
}

func (c *Client) SetName(newName string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.name = newName
}
