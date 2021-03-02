# first image used to build the sources
FROM golang:1.15.5-buster AS builder

ENV GO111MODULE=on \
  GOOS=linux \
  CGO_ENABLED=1 \
  GOARCH=amd64

WORKDIR /tor-proxy

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .


RUN go build -o torproxy ./cmd/*.go

WORKDIR /bin

RUN cp /tor-proxy/torproxy .

FROM debian:buster-slim

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates

COPY --from=builder /bin/torproxy /usr/local/bin/torproxy

RUN chmod a+x /usr/local/bin/torproxy

RUN useradd -ms /bin/bash torproxy

USER torproxy

# Prevents `VOLUME $HOME/.local/` being created as owned by `root`
RUN mkdir -p "$HOME/.local/"

ENTRYPOINT ["torproxy"]

