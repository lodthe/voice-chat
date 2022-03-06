package server

import (
	"fmt"
	"net"
	"sync"

	"github.com/google/uuid"
)

type ClientDescription struct {
	Room *string
	Name string
}

type Client struct {
	conn net.Conn

	id      uuid.UUID
	address string

	description       ClientDescription
	descriptionLocker sync.RWMutex
}

func NewClient(conn net.Conn) *Client {
	return &Client{
		conn:    conn,
		id:      uuid.New(),
		address: conn.RemoteAddr().String(),
	}
}

func (c *Client) String() string {
	return fmt.Sprintf("%s [%s]", c.address, c.id)
}

func (c *Client) ID() uuid.UUID {
	return c.id
}

func (c *Client) Description() ClientDescription {
	c.descriptionLocker.RLock()
	defer c.descriptionLocker.RUnlock()

	return c.description
}

func (c *Client) SetDescription(d ClientDescription) {
	c.descriptionLocker.Lock()
	defer c.descriptionLocker.Unlock()

	c.description = d
}
