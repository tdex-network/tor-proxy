FROM golang:1.18-buster AS builder

ENV GO111MODULE=on \
  CGO_ENABLED=0

WORKDIR /tor-proxy

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .


RUN go build -o torproxy ./cmd/*.go

WORKDIR /bin

RUN cp /tor-proxy/torproxy .

FROM debian:buster-slim

RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends ca-certificates

COPY --from=builder /bin/torproxy /usr/local/bin/torproxy

RUN chmod a+x /usr/local/bin/torproxy

RUN useradd -ms /bin/bash torproxy

USER torproxy

# Prevents `VOLUME $HOME/.local/` being created as owned by `root`
RUN mkdir -p "$HOME/.local/"


ENTRYPOINT torproxy start --insecure --port="${PORT}" --registry="${REGISTRY_URL}" --socks5-hostname="${SOCKS5_HOSTNAME}" --socks5-port="${SOCKS5_PORT}"

