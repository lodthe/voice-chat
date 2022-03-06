// Inspired by https://github.com/kechako/vchat

package client

import (
	"context"
	"fmt"
	"log"

	"github.com/gordonklaus/portaudio"
	"github.com/hraban/opus"
	"github.com/pkg/errors"
)

type AudioConfig struct {
	Channels       int
	SampleRate     int
	FrameSize      int
	OpusDataLength int
}

var DefaultAudioConfig = AudioConfig{
	Channels:       1,
	SampleRate:     48000,
	FrameSize:      480,
	OpusDataLength: 1000,
}

type Audio struct {
	ctx    context.Context
	cancel context.CancelFunc

	config AudioConfig

	stream *portaudio.Stream
	inbuf  []int16
	outbuf []int16

	input  chan<- []byte
	output <-chan []byte
}

func NewAudio(ctx context.Context, config AudioConfig, input chan<- []byte, output <-chan []byte) (*Audio, error) {
	err := portaudio.Initialize()
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize portaudio")
	}

	ctx, cancel := context.WithCancel(ctx)

	a := &Audio{
		ctx:    ctx,
		cancel: cancel,
		config: config,
		input:  input,
		output: output,
	}

	a.inbuf = make([]int16, a.config.FrameSize)
	a.outbuf = make([]int16, a.config.FrameSize)

	a.stream, err = portaudio.OpenDefaultStream(a.config.Channels, a.config.Channels, float64(a.config.SampleRate), len(a.inbuf), a.inbuf, a.outbuf)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open default stream")
	}

	// Make sure that the raw PCM data you want to encode has a legal Opus frame size.
	// This means it must be exactly 2.5, 5, 10, 20, 40 or 60 ms long.
	// The number of bytes this corresponds to depends on the sample rate.
	// See the libopus documentation: https://www.opus-codec.org/docs/opus_api-1.1.3/group__opus__encoder.html
	frameSizeMs := float32(a.config.FrameSize) / float32(a.config.Channels) * 1000 / float32(a.config.SampleRate)
	switch frameSizeMs {
	case 2.5, 5, 10, 20, 40, 60:
	default:
		return nil, fmt.Errorf("illegal frame size: %d bytes (%f ms)", a.config.FrameSize, frameSizeMs)
	}

	return a, nil
}

func (a *Audio) Start() error {
	err := a.stream.Start()
	if err != nil {
		return errors.Wrap(err, "failed to start stream")
	}

	go a.inputSendLoop()
	go a.receiveOutputLoop()

	<-a.ctx.Done()

	return nil
}

func (a *Audio) Stop() error {
	a.cancel()

	err := a.stream.Close()
	if err != nil {
		return errors.Wrap(err, "failed to close stream")
	}

	err = portaudio.Terminate()
	if err != nil {
		return errors.Wrap(err, "failed to terminate portaudio")
	}

	return nil
}

func (a *Audio) inputSendLoop() {
	enc, err := opus.NewEncoder(a.config.SampleRate, a.config.Channels, opus.AppVoIP)
	if err != nil {
		log.Fatalf("failed to create opus encoder: %v\n", err)
	}

	for {
		select {
		case <-a.ctx.Done():
			return

		default:
			var err error
			err = a.stream.Read()
			if err != nil && err != portaudio.InputOverflowed {
				log.Fatalf("failed to read from audio stream: %v\n", err)
			}

			data := make([]byte, a.config.OpusDataLength)
			n, err := enc.Encode(a.inbuf, data)
			if err != nil {
				log.Fatalf("failed to encode opus: %v\n", err)
			}

			select {
			case a.input <- data[:n]:
			default:
			}
		}
	}
}

func (a *Audio) receiveOutputLoop() {
	dec, err := opus.NewDecoder(a.config.SampleRate, a.config.Channels)
	if err != nil {
		log.Fatalf("failed to create opus decoder: %v\n", err)
	}

	for {
		select {
		case <-a.ctx.Done():
			return

		case frame := <-a.output:
			_, err = dec.Decode(frame, a.outbuf)
			if err != nil {
				log.Fatalf("failed to decode opus: %v\n", err)
			}

			err = a.stream.Write()
			if err != nil && err != portaudio.OutputUnderflowed {
				log.Fatalf("failed to write to audio stream: %v\n", err)
			}
		}
	}
}
