# Dockerfile was generated from
# https://github.com/lodthe/dockerfiles/blob/main/go/Dockerfile

FROM golang:1.17.3-alpine3.14 AS builder

# Setup base software for building an app.
RUN apk update && \
    apk add bash ca-certificates git gcc g++ libc-dev binutils file

WORKDIR /opt

# Download dependencies.
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy application source.
COPY . .

# Build the application.
RUN CGO_ENABLED=1 go build -o bin/server cmd/server/main.go

# Prepare executor image.
FROM alpine:3.14 AS runner

RUN apk update && \
    apk add ca-certificates libc6-compat && \
    rm -rf /var/cache/apk/*

WORKDIR /opt

COPY --from=builder /opt/bin/server ./

# Add required static files.
#COPY assets assets

# Run the application.
CMD ["./server"]