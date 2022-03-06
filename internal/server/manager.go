package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/lodthe/voice-chat/pkg/vprotocol"
	"github.com/pkg/errors"
)

type Manager struct {
	ctx    context.Context
	cancel context.CancelFunc

	clients       map[uuid.UUID]*Client
	clientsLocker sync.RWMutex
}

func NewManager(ctx context.Context) *Manager {
	ctx, cancel := context.WithCancel(ctx)

	return &Manager{
		ctx:     ctx,
		cancel:  cancel,
		clients: make(map[uuid.UUID]*Client),
	}
}

func (m *Manager) HandleConn(conn net.Conn) {
	defer conn.Close()

	c := NewClient(conn)

	err := m.addClient(c)
	if err != nil {
		log.Printf("failed to add a new client %s: %v\n", c, err)
		return
	}
	defer m.deleteClient(c)

	log.Printf("%s connected\n", c)

	for {
		select {
		case <-m.ctx.Done():
			log.Printf("disconnect %s\n", c)
			return

		default:
		}

		msg := new(vprotocol.Message)
		err = msg.Unmarshal(c.conn)
		if errors.Is(err, io.EOF) {
			log.Printf("%s disconnected\n", c)
			break
		}
		if err != nil {
			log.Printf("error while reading from %s: %v\n", c, err)
			break
		}

		err = m.handleMessage(c, msg)
		if err != nil {
			log.Printf("failed to handle message from %s: %v\n : %v\n", c, msg, err)
		}
	}
}

func (m *Manager) handleMessage(c *Client, msg *vprotocol.Message) error {
	var err error
	p := msg.Payload

	switch msg.Type {
	case vprotocol.TypeJoin:
		err = m.handleJoin(c, p.(*vprotocol.PayloadJoin))

	case vprotocol.TypeLeave:
		err = m.handleLeave(c, p.(*vprotocol.PayloadLeave))

	case vprotocol.TypeAudio:
		err = m.handleAudio(c, p.(*vprotocol.PayloadAudio))

	default:
		return fmt.Errorf("unknown message type %v", msg.Type)
	}

	if err != nil {
		return errors.Wrap(err, "handling failed")
	}

	return nil
}

func (m *Manager) handleJoin(c *Client, payload *vprotocol.PayloadJoin) error {
	payload.Room = strings.ToLower(payload.Room)

	oldDescr := c.Description()
	if oldDescr.Room != nil && strings.EqualFold(*oldDescr.Room, payload.Room) {
		return nil
	}

	err := m.handleLeave(c, &vprotocol.PayloadLeave{})
	if err != nil {
		return errors.Wrap(err, "failed to leave")
	}

	c.SetDescription(ClientDescription{
		Room: &payload.Room,
		Name: payload.Name,
	})

	log.Printf("%s joined #%s as %s\n", c, payload.Room, payload.Name)

	msg := vprotocol.Message{
		Type: vprotocol.TypeJoin,
		Payload: &vprotocol.PayloadJoin{
			Room: payload.Room,
			Name: payload.Name,
		},
	}

	m.forward(&msg, func(peer *Client) bool {
		return true
	})

	return nil
}

func (m *Manager) handleLeave(c *Client, _ *vprotocol.PayloadLeave) error {
	descr := c.Description()
	if descr.Room == nil {
		return nil
	}

	c.SetDescription(ClientDescription{
		Room: nil,
		Name: descr.Name,
	})

	log.Printf("%s left #%s as %s\n", c, *descr.Room, descr.Name)

	msg := vprotocol.Message{
		Type: vprotocol.TypeLeave,
		Payload: &vprotocol.PayloadLeave{
			Room: *descr.Room,
			Name: descr.Name,
		},
	}

	m.forward(&msg, func(peer *Client) bool {
		return true
	})

	return nil
}

func (m *Manager) handleAudio(c *Client, payload *vprotocol.PayloadAudio) error {
	descr := c.Description()
	if descr.Room == nil {
		return nil
	}

	msg := vprotocol.Message{
		Type:    vprotocol.TypeAudio,
		Payload: payload,
	}

	m.forward(&msg, func(peer *Client) bool {
		d := peer.Description()
		return d.Room != nil && *d.Room == *descr.Room && peer.ID() != c.ID()
	})

	return nil
}

func (m *Manager) forward(msg *vprotocol.Message, ok func(c *Client) bool) {
	m.clientsLocker.RLock()
	defer m.clientsLocker.RUnlock()

	for _, client := range m.clients {
		if ok(client) {
			err := msg.Marshal(client.conn)
			if err != nil {
				log.Printf("failed to send message to %s: %v\n", client, err)
			}
		}
	}
}

func (m *Manager) addClient(c *Client) error {
	m.clientsLocker.Lock()
	defer m.clientsLocker.Unlock()

	_, found := m.clients[c.ID()]
	if found {
		return fmt.Errorf("client with ID() %s already exists", c.ID())
	}

	m.clients[c.ID()] = c

	return nil
}

func (m *Manager) deleteClient(c *Client) {
	m.clientsLocker.Lock()
	defer m.clientsLocker.Unlock()

	delete(m.clients, c.ID())
}
