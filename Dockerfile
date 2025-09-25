FROM golang:1.22 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o docker-watchdog main.go

# Minimal runtime image
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates \
      && rm -rf /var/lib/apt/lists/*
WORKDIR /app

COPY --from=builder /app/docker-watchdog /usr/local/bin/docker-watchdog

# Needed if you use apprise inside the container
RUN apt-get update && apt-get install -y --no-install-recommends \
      python3 python3-pip \
      && pip3 install apprise \
      && rm -rf /var/lib/apt/lists/*

ENTRYPOINT ["docker-watchdog"]
