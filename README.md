# Voice-chat

A simple voice-chat written in Golang with portaudio and libopus under the hood.

## Installation

### Ubuntu

```bash
apt install -y pip libasound-dev portaudio19-dev libportaudio2 libportaudiocpp0 pkg-config libopus-dev libopusfile-dev
```

### Mac

```bash
brew install pkg-config opus opusfile portaudio
```

## Usage

### Server

Run the following command from the root of the repo:
```bash
go run cmd/server/main.go --address 0.0.0.0:9000
```

### Client

Run the following command from the root of the repo:
```bash
CGO_ENABLED=1 go run cmd/client/main.go
```

## Inspiration

The project was inspired by [kechako/vchat](https://github.com/kechako/vchat).