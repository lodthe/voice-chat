package vprotocol

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"

	"github.com/pkg/errors"
)

type MessageType uint8

const (
	TypeUnknown MessageType = iota
	TypeJoin
	TypeLeave
	TypeAudio
)

type Payload interface{}

type Message struct {
	Type    MessageType
	Payload Payload
}

func (m *Message) Unmarshal(r io.Reader) error {
	err := binary.Read(r, binary.BigEndian, &m.Type)
	if err != nil {
		return errors.Wrap(err, "failed to read message type")
	}

	switch m.Type {
	case TypeJoin:
		m.Payload = new(PayloadJoin)

	case TypeLeave:
		m.Payload = new(PayloadLeave)

	case TypeAudio:
		m.Payload = new(PayloadAudio)

	default:
		return fmt.Errorf("unknown message type %d", m.Type)
	}

	var size uint32
	err = binary.Read(r, binary.BigEndian, &size)
	if err != nil {
		return errors.Wrap(err, "failed to read payload size")
	}

	marshalled := make([]byte, size)
	_, err = io.ReadFull(r, marshalled)
	if err != nil {
		return errors.Wrap(err, "failed to read marshalled payload")
	}

	err = json.Unmarshal(marshalled, m.Payload)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal payload")
	}

	return nil
}

func (m *Message) Marshal(w io.Writer) error {
	err := binary.Write(w, binary.BigEndian, &m.Type)
	if err != nil {
		return errors.Wrap(err, "failed to write message type")
	}

	marshalled, err := json.Marshal(m.Payload)
	if err != nil {
		return errors.Wrap(err, "marshal fialed")
	}

	size := uint32(len(marshalled))
	err = binary.Write(w, binary.BigEndian, &size)
	if err != nil {
		return errors.Wrap(err, "failed to write payload size")
	}

	_, err = w.Write(marshalled)
	if err != nil {
		return errors.Wrap(err, "failed to write marshalled payload")
	}

	return nil
}
