package client

import (
	"context"
	"log"
	"net"

	"github.com/lodthe/voice-chat/pkg/vprotocol"
	"github.com/pkg/errors"
)

type Connection struct {
	ctx context.Context

	conn net.Conn

	ingoing  chan<- *vprotocol.Message
	outgoing <-chan *vprotocol.Message
}

func NewConnection(ctx context.Context, address string, ingoing chan<- *vprotocol.Message, outgoing <-chan *vprotocol.Message) (*Connection, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, errors.Wrap(err, "dial failed")
	}

	return &Connection{
		ctx:      ctx,
		conn:     conn,
		ingoing:  ingoing,
		outgoing: outgoing,
	}, nil
}

func (c *Connection) Start() error {
	go c.write()

	return c.read()
}

func (c *Connection) read() error {
	defer c.conn.Close()

	for {
		select {
		case <-c.ctx.Done():
			return nil
		default:
		}

		var msg vprotocol.Message
		err := msg.Unmarshal(c.conn)
		if err != nil {
			return errors.Wrap(err, "unmarshal failed")
		}

		c.ingoing <- &msg
	}
}

func (c *Connection) write() {
	for {
		var msg *vprotocol.Message
		select {
		case <-c.ctx.Done():
			return

		case msg = <-c.outgoing:
		}

		err := msg.Marshal(c.conn)
		if err != nil {
			log.Printf("failed to send a message: %v\n", err)
		}
	}
}
