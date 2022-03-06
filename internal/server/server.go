package server

import (
	"context"
	"log"
	"net"
)

type Server struct {
	ctx    context.Context
	cancel context.CancelFunc

	address string

	listener net.Listener
	handler  *Manager
}

func NewServer(ctx context.Context, address string, h *Manager) *Server {
	ctx, cancel := context.WithCancel(ctx)

	return &Server{
		ctx:     ctx,
		cancel:  cancel,
		address: address,
		handler: h,
	}
}

func (s *Server) Serve() error {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}

	defer listener.Close()

	log.Printf("started listening to %s\n", s.address)

	s.listener = listener

	for {
		select {
		case <-s.ctx.Done():
			return nil
		default:
		}

		conn, err := listener.Accept()
		if err != nil {
			log.Printf("accept failed: %v\n", err)
			continue
		}

		go s.handler.HandleConn(conn)
	}
}
